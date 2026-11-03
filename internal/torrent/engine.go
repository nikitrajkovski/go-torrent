package torrent

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	libtorrent "github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	libstorage "github.com/anacrolix/torrent/storage"

	"github.com/nikitrajkovski/torrentgo/internal/models"
	"github.com/nikitrajkovski/torrentgo/internal/storage"
)

const (
	sourceTypeMagnet        = "magnet"
	sourceTypeTorrentFile   = "torrent-file"
	maxTorrentFileSize      = 10 << 20
	defaultRateLimiterBurst = 1 << 20
	minimumRateLimiterBurst = 16 << 10
)

type managedTorrent struct {
	torrent           *libtorrent.Torrent
	storage           libstorage.ClientImplCloser
	record            storage.TorrentRecord
	paused            bool
	stopped           bool
	awaitingSelection bool
	activeTransfer    bool
	lastDownload      int64
	lastUpload        int64
	lastUpdatedAt     time.Time
}

// Engine is the local BitTorrent boundary. It has no dependency on Wails or
// React, which keeps protocol and filesystem operations testable in isolation.
type Engine struct {
	mu                     sync.Mutex
	client                 *libtorrent.Client
	entries                map[string]*managedTorrent
	store                  *storage.Store
	logger                 *slog.Logger
	downloadLimiter        *rate.Limiter
	uploadLimiter          *rate.Limiter
	maxConcurrentDownloads int
	closed                 bool
}

func New(store *storage.Store, logger *slog.Logger, initialSettings ...models.Settings) (*Engine, error) {
	if store == nil {
		return nil, errors.New("torrent store is required")
	}
	if logger == nil {
		logger = slog.Default()
	}

	settings := models.Settings{}
	if len(initialSettings) > 0 {
		settings = initialSettings[0]
	}
	clientConfig := libtorrent.NewDefaultClientConfig()
	clientConfig.DataDir = filepath.Dir(store.Path())
	clientConfig.ListenPort = 0
	clientConfig.Seed = true
	clientConfig.NoDefaultPortForwarding = true
	clientConfig.Slogger = logger
	downloadLimiter := rate.NewLimiter(rate.Inf, defaultRateLimiterBurst)
	uploadLimiter := rate.NewLimiter(rate.Inf, defaultRateLimiterBurst)
	setRateLimit(downloadLimiter, settings.DownloadLimit)
	setRateLimit(uploadLimiter, settings.UploadLimit)
	clientConfig.DownloadRateLimiter = downloadLimiter
	clientConfig.UploadRateLimiter = uploadLimiter
	client, err := libtorrent.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("create torrent client: %w", err)
	}

	engine := &Engine{
		client:                 client,
		entries:                make(map[string]*managedTorrent),
		store:                  store,
		logger:                 logger,
		downloadLimiter:        downloadLimiter,
		uploadLimiter:          uploadLimiter,
		maxConcurrentDownloads: settings.MaxConcurrentDownloads,
	}
	if err := engine.restore(); err != nil {
		client.Close()
		return nil, err
	}
	return engine, nil
}

// AddMagnet validates and adds a magnet URI. The URI stays local and is not
// logged because it can reveal tracker or content information.
func (e *Engine) AddMagnet(magnet, downloadDirectory string) (models.TorrentInfo, error) {
	return e.addMagnet(magnet, downloadDirectory, false)
}

// PrepareMagnet retrieves only metainfo. Payload transfer waits for an
// explicit StartPrepared call after the user has reviewed its files.
func (e *Engine) PrepareMagnet(magnet, downloadDirectory string) (models.TorrentInfo, error) {
	return e.addMagnet(magnet, downloadDirectory, true)
}

func (e *Engine) addMagnet(magnet, downloadDirectory string, awaitingSelection bool) (models.TorrentInfo, error) {
	magnet = strings.TrimSpace(magnet)
	if magnet == "" {
		return models.TorrentInfo{}, errors.New("magnet link is required")
	}
	spec, err := libtorrent.TorrentSpecFromMagnetUri(magnet)
	if err != nil {
		return models.TorrentInfo{}, fmt.Errorf("invalid magnet link: %w", err)
	}

	record := storage.TorrentRecord{
		ID:                torrentID(spec),
		Source:            magnet,
		SourceType:        sourceTypeMagnet,
		DownloadDirectory: downloadDirectory,
		Trackers:          cloneTrackerTiers(spec.Trackers),
		AwaitingSelection: awaitingSelection,
		AddedAt:           time.Now().UTC(),
	}
	return e.addSpec(spec, record, false)
}

// AddTorrentFile reads metainfo with a bounded input size, then keeps an
// encoded local copy for restart recovery. It never follows or executes data
// referenced by the metainfo file.
func (e *Engine) AddTorrentFile(path, downloadDirectory string) (models.TorrentInfo, error) {
	return e.addTorrentFile(path, downloadDirectory, false)
}

func (e *Engine) PrepareTorrentFile(path, downloadDirectory string) (models.TorrentInfo, error) {
	return e.addTorrentFile(path, downloadDirectory, true)
}

func (e *Engine) addTorrentFile(path, downloadDirectory string, awaitingSelection bool) (models.TorrentInfo, error) {
	if strings.ToLower(filepath.Ext(path)) != ".torrent" {
		return models.TorrentInfo{}, errors.New("selected file must have a .torrent extension")
	}
	info, err := os.Stat(path)
	if err != nil {
		return models.TorrentInfo{}, fmt.Errorf("inspect torrent file: %w", err)
	}
	if info.Size() <= 0 || info.Size() > maxTorrentFileSize {
		return models.TorrentInfo{}, fmt.Errorf("torrent file must be between 1 byte and %d MiB", maxTorrentFileSize>>20)
	}
	file, err := os.Open(path)
	if err != nil {
		return models.TorrentInfo{}, fmt.Errorf("open torrent file: %w", err)
	}
	defer file.Close()
	rawMetainfo, err := io.ReadAll(io.LimitReader(file, maxTorrentFileSize+1))
	if err != nil {
		return models.TorrentInfo{}, fmt.Errorf("read torrent file: %w", err)
	}
	if len(rawMetainfo) > maxTorrentFileSize {
		return models.TorrentInfo{}, fmt.Errorf("torrent file exceeds %d MiB", maxTorrentFileSize>>20)
	}
	metaInfo, err := metainfo.Load(bytes.NewReader(rawMetainfo))
	if err != nil {
		return models.TorrentInfo{}, fmt.Errorf("parse torrent file: %w", err)
	}
	spec, err := libtorrent.TorrentSpecFromMetaInfoErr(metaInfo)
	if err != nil {
		return models.TorrentInfo{}, fmt.Errorf("parse torrent metadata: %w", err)
	}

	record := storage.TorrentRecord{
		ID:                torrentID(spec),
		Source:            base64.StdEncoding.EncodeToString(rawMetainfo),
		SourceType:        sourceTypeTorrentFile,
		DownloadDirectory: downloadDirectory,
		Trackers:          cloneTrackerTiers(spec.Trackers),
		AwaitingSelection: awaitingSelection,
		AddedAt:           time.Now().UTC(),
	}
	return e.addSpec(spec, record, false)
}

// UpdateSettings changes live global transfer controls. The rate-limiters are
// retained by the client, so updating their limits affects existing and future
// peer connections without restarting the torrent engine.
func (e *Engine) UpdateSettings(settings models.Settings) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return errors.New("torrent engine is closed")
	}
	setRateLimit(e.downloadLimiter, settings.DownloadLimit)
	setRateLimit(e.uploadLimiter, settings.UploadLimit)
	e.maxConcurrentDownloads = settings.MaxConcurrentDownloads
	e.reconcileDownloadsLocked()
	return nil
}

func (e *Engine) Pause(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	entry, err := e.entryLocked(id)
	if err != nil {
		return err
	}
	if err := e.store.UpdateState(id, true, false); err != nil {
		return fmt.Errorf("persist paused torrent: %w", err)
	}
	entry.torrent.DisallowDataDownload()
	entry.torrent.DisallowDataUpload()
	entry.paused = true
	entry.stopped = false
	entry.record.Paused = true
	entry.record.Stopped = false
	e.reconcileDownloadsLocked()
	return nil
}

func (e *Engine) Resume(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	entry, err := e.entryLocked(id)
	if err != nil {
		return err
	}
	if err := e.store.UpdateState(id, false, false); err != nil {
		return fmt.Errorf("persist resumed torrent: %w", err)
	}
	entry.paused = false
	entry.stopped = false
	entry.record.Paused = false
	entry.record.Stopped = false
	e.reconcileDownloadsLocked()
	return nil
}

// Stop prevents a completed torrent from seeding while retaining its files,
// metadata, and place in the queue. Resume starts seeding again.
func (e *Engine) Stop(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	entry, err := e.entryLocked(id)
	if err != nil {
		return err
	}
	if err := e.store.UpdateState(id, false, true); err != nil {
		return fmt.Errorf("persist stopped torrent: %w", err)
	}
	entry.torrent.DisallowDataDownload()
	entry.torrent.DisallowDataUpload()
	entry.paused = false
	entry.stopped = true
	entry.record.Paused = false
	entry.record.Stopped = true
	e.reconcileDownloadsLocked()
	return nil
}

// PauseAll stops payload transfer for every active torrent and persists the
// complete state transition before changing the live client state.
func (e *Engine) PauseAll() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return errors.New("torrent engine is closed")
	}
	if err := e.store.SetAllPaused(true); err != nil {
		return fmt.Errorf("persist paused torrents: %w", err)
	}
	for _, entry := range e.entries {
		entry.paused = true
		entry.stopped = false
		entry.record.Paused = true
		entry.record.Stopped = false
	}
	e.reconcileDownloadsLocked()
	return nil
}

// ResumeAll restores payload transfer for every persisted torrent. Metadata
// downloads continue through the same client and pieces are only requested
// after the library reports that metainfo is available.
func (e *Engine) ResumeAll() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return errors.New("torrent engine is closed")
	}
	if err := e.store.SetAllPaused(false); err != nil {
		return fmt.Errorf("persist resumed torrents: %w", err)
	}
	for _, entry := range e.entries {
		entry.paused = false
		entry.stopped = false
		entry.record.Paused = false
		entry.record.Stopped = false
	}
	e.reconcileDownloadsLocked()
	return nil
}

// Remove drops the torrent from the engine and its persistent index. Torrade
// intentionally does not delete payload files in Phase 1: a shared destination
// can contain files from more than one torrent, so blind removal is unsafe.
func (e *Engine) Remove(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	entry, err := e.entryLocked(id)
	if err != nil {
		return err
	}
	if err := e.store.Delete(id); err != nil {
		return fmt.Errorf("remove persisted torrent: %w", err)
	}
	entry.torrent.Drop()
	delete(e.entries, id)
	if err := entry.storage.Close(); err != nil {
		return fmt.Errorf("close torrent storage: %w", err)
	}
	return nil
}

// Details returns only values the torrent library currently exposes. It does
// not infer tracker health or retain disconnected peer addresses.
func (e *Engine) Details(id string) (models.TorrentDetails, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	entry, err := e.entryLocked(id)
	if err != nil {
		return models.TorrentDetails{}, err
	}
	details := models.TorrentDetails{Info: snapshotEntry(entry, time.Now())}
	for tier, urls := range entry.record.Trackers {
		for _, trackerURL := range urls {
			details.Trackers = append(details.Trackers, models.TorrentTracker{URL: trackerURL, Tier: tier + 1})
		}
	}
	if entry.torrent.Info() == nil {
		details.MetadataPending = true
		return details, nil
	}
	files := entry.torrent.Files()
	details.Files = make([]models.TorrentFile, 0, len(files))
	for index, file := range files {
		details.Files = append(details.Files, models.TorrentFile{
			Index:      index,
			Path:       file.DisplayPath(),
			Size:       file.Length(),
			Downloaded: file.BytesCompleted(),
			Selected:   isFileSelected(entry.record.SelectedFiles, index),
		})
	}
	for _, peer := range entry.torrent.PeerConns() {
		client := "Unknown"
		if value := peer.PeerClientName.Load(); value != nil {
			if name, ok := value.(string); ok && name != "" {
				client = name
			}
		}
		address := "Unknown"
		if peer.RemoteAddr != nil {
			address = peer.RemoteAddr.String()
		}
		stats := peer.Stats()
		details.Peers = append(details.Peers, models.TorrentPeer{
			Address:      address,
			Network:      peer.Network,
			Client:       client,
			DownloadRate: stats.DownloadRate,
			UploadRate:   stats.LastWriteUploadRate,
		})
	}
	sort.Slice(details.Peers, func(left, right int) bool { return details.Peers[left].Address < details.Peers[right].Address })
	return details, nil
}

// SetFileSelected changes piece priority for an individual metainfo file. A
// file can only be selected after its metadata is available.
func (e *Engine) SetFileSelected(id string, index int, selected bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	entry, err := e.entryLocked(id)
	if err != nil {
		return err
	}
	if entry.torrent.Info() == nil {
		return errors.New("file selection is available after torrent metadata is fetched")
	}
	files := entry.torrent.Files()
	if index < 0 || index >= len(files) {
		return fmt.Errorf("file index %d is not available", index)
	}
	selection := normalizedSelection(entry.record.SelectedFiles, len(files))
	selection[index] = selected
	if err := e.store.UpdateSelectedFiles(id, selection); err != nil {
		return fmt.Errorf("persist file selection: %w", err)
	}
	entry.record.SelectedFiles = selection
	if entry.awaitingSelection {
		// The review flow records the choice only. Raising a File priority here
		// would request payload pieces before the explicit StartPrepared action.
		return nil
	}
	if selected {
		files[index].Download()
	} else {
		files[index].SetPriority(libtorrent.PiecePriorityNone)
	}
	e.reconcileDownloadsLocked()
	return nil
}

func (e *Engine) Snapshot() models.TorrentSnapshot {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.reconcileDownloadsLocked()

	now := time.Now()
	snapshot := models.TorrentSnapshot{
		Torrents:     make([]models.TorrentInfo, 0, len(e.entries)),
		NetworkReady: !e.closed,
	}
	for _, entry := range e.entries {
		info := snapshotEntry(entry, now)
		if entry.torrent.Info() != nil && entry.record.CachedMetainfo == "" {
			e.persistMetainfoRecoveryDataLocked(entry, info)
		}
		if info.Status == models.TorrentStatusCompleted {
			e.persistCompletedRecoveryDataLocked(entry, info)
		}
		snapshot.Torrents = append(snapshot.Torrents, info)
		snapshot.DownloadRate += info.DownloadRate
		snapshot.UploadRate += info.UploadRate
		if entry.activeTransfer {
			snapshot.Active++
		}
	}
	sort.Slice(snapshot.Torrents, func(left, right int) bool {
		return strings.ToLower(snapshot.Torrents[left].Name) < strings.ToLower(snapshot.Torrents[right].Name)
	})
	return snapshot
}

func (e *Engine) persistCompletedRecoveryDataLocked(entry *managedTorrent, info models.TorrentInfo) {
	if entry.record.KnownName == info.Name && entry.record.KnownSize == info.Size && entry.record.KnownDownloaded == info.Downloaded {
		return
	}
	if err := e.store.UpdateRecoveryData(entry.record.ID, entry.record.CachedMetainfo, info.Name, info.Size, info.Downloaded); err != nil {
		e.logger.Warn("persist completed torrent recovery data", "id", entry.record.ID, "error", err)
		return
	}
	entry.record.KnownName = info.Name
	entry.record.KnownSize = info.Size
	entry.record.KnownDownloaded = info.Downloaded
}

func (e *Engine) persistMetainfoRecoveryDataLocked(entry *managedTorrent, info models.TorrentInfo) {
	metaInfoBytes, err := metainfoBytes(entry.torrent.Metainfo())
	if err != nil {
		e.logger.Warn("cache torrent metainfo", "id", entry.record.ID, "error", err)
		return
	}
	encodedMetainfo := base64.StdEncoding.EncodeToString(metaInfoBytes)
	if err := e.store.UpdateRecoveryData(entry.record.ID, encodedMetainfo, info.Name, info.Size, info.Downloaded); err != nil {
		e.logger.Warn("persist torrent recovery data", "id", entry.record.ID, "error", err)
		return
	}
	entry.record.CachedMetainfo = encodedMetainfo
	entry.record.KnownName = info.Name
	entry.record.KnownSize = info.Size
	entry.record.KnownDownloaded = info.Downloaded
}

func (e *Engine) Close() error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil
	}
	e.closed = true
	client := e.client
	storages := make([]libstorage.ClientImplCloser, 0, len(e.entries))
	for _, entry := range e.entries {
		storages = append(storages, entry.storage)
	}
	e.mu.Unlock()

	closeErrors := client.Close()
	for _, torrentStorage := range storages {
		if err := torrentStorage.Close(); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("close torrent storage: %w", err))
		}
	}
	if len(closeErrors) > 0 {
		return fmt.Errorf("close torrent engine: %w", errors.Join(closeErrors...))
	}
	return nil
}

func (e *Engine) restore() error {
	records, err := e.store.Load()
	if err != nil {
		return fmt.Errorf("load persisted torrents: %w", err)
	}
	for _, record := range records {
		spec, err := specFromRecord(record)
		if err != nil {
			e.logger.Warn("skip invalid persisted torrent", "id", record.ID, "error", err)
			continue
		}
		if _, err := e.addSpec(spec, record, true); err != nil {
			e.logger.Warn("skip unavailable persisted torrent", "id", record.ID, "error", err)
		}
	}
	return nil
}

func (e *Engine) addSpec(spec *libtorrent.TorrentSpec, record storage.TorrentRecord, restoring bool) (models.TorrentInfo, error) {
	if err := validateDownloadDirectory(record.DownloadDirectory); err != nil {
		return models.TorrentInfo{}, err
	}
	if record.ID == "" {
		return models.TorrentInfo{}, errors.New("torrent has no supported info hash")
	}
	// TorrentSpec storage is not owned by anacrolix.Client. Keep it on the
	// managed torrent so its completion database is deterministically closed
	// after Drop/Client.Close. This is required on Windows, where SQLite keeps
	// the database file locked until its connection is explicitly closed.
	torrentStorage := libstorage.NewFile(record.DownloadDirectory)
	spec.Storage = torrentStorage
	// Add safely with payload transfer disabled. Reconciliation below admits it
	// according to the configured concurrent-download limit.
	spec.DisallowDataDownload = true
	spec.DisallowDataUpload = true

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		_ = torrentStorage.Close()
		return models.TorrentInfo{}, errors.New("torrent engine is closed")
	}
	if _, exists := e.entries[record.ID]; exists {
		_ = torrentStorage.Close()
		return models.TorrentInfo{}, errors.New("this torrent has already been added")
	}
	torrent, added, err := e.client.AddTorrentSpec(spec)
	if err != nil {
		_ = torrentStorage.Close()
		return models.TorrentInfo{}, fmt.Errorf("add torrent to engine: %w", err)
	}
	if !added {
		_ = torrentStorage.Close()
		return models.TorrentInfo{}, errors.New("this torrent has already been added")
	}
	if len(record.Trackers) == 0 {
		record.Trackers = cloneTrackerTiers(spec.Trackers)
	}
	entry := &managedTorrent{torrent: torrent, storage: torrentStorage, record: record, paused: record.Paused, stopped: record.Stopped, awaitingSelection: record.AwaitingSelection}
	e.entries[record.ID] = entry
	if !restoring {
		if err := e.store.Upsert(record); err != nil {
			torrent.Drop()
			_ = torrentStorage.Close()
			delete(e.entries, record.ID)
			return models.TorrentInfo{}, fmt.Errorf("persist torrent: %w", err)
		}
	}
	e.reconcileDownloadsLocked()
	return snapshotEntry(entry, time.Now()), nil
}

func (e *Engine) entryLocked(id string) (*managedTorrent, error) {
	if e.closed {
		return nil, errors.New("torrent engine is closed")
	}
	entry, exists := e.entries[id]
	if !exists {
		return nil, fmt.Errorf("torrent %q was not found", id)
	}
	return entry, nil
}

// StartPrepared permits payload transfer for a torrent that was added through
// the metadata-first flow. At least one file must remain selected.
func (e *Engine) StartPrepared(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	entry, err := e.entryLocked(id)
	if err != nil {
		return err
	}
	if !entry.awaitingSelection {
		return errors.New("torrent is not awaiting file selection")
	}
	if entry.torrent.Info() == nil {
		return errors.New("torrent metadata is not available yet")
	}
	if !hasSelectedFiles(entry.record.SelectedFiles) {
		return errors.New("select at least one file before starting the download")
	}
	if err := e.store.UpdateAwaitingSelection(id, false); err != nil {
		return fmt.Errorf("persist file selection confirmation: %w", err)
	}
	entry.awaitingSelection = false
	entry.record.AwaitingSelection = false
	e.reconcileDownloadsLocked()
	return nil
}

func (e *Engine) reconcileDownloadsLocked() {
	entries := make([]*managedTorrent, 0, len(e.entries))
	for _, entry := range e.entries {
		if entry.awaitingSelection {
			// Metadata exchange and peer discovery need a live data path. No
			// payload pieces are requested because no file priorities have been
			// raised until StartPrepared runs.
			entry.activeTransfer = false
			entry.torrent.AllowDataDownload()
			entry.torrent.DisallowDataUpload()
			continue
		}
		if entry.paused || entry.stopped || !hasSelectedFiles(entry.record.SelectedFiles) {
			entry.activeTransfer = false
			entry.torrent.DisallowDataDownload()
			entry.torrent.DisallowDataUpload()
			continue
		}
		if entry.torrent.Info() != nil && entry.torrent.Complete().Bool() {
			entry.activeTransfer = false
			entry.torrent.DisallowDataDownload()
			entry.torrent.AllowDataUpload()
			continue
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(left, right int) bool {
		return entries[left].record.AddedAt.Before(entries[right].record.AddedAt)
	})
	active := 0
	for _, entry := range entries {
		shouldRun := e.maxConcurrentDownloads == 0 || active < e.maxConcurrentDownloads
		if shouldRun {
			if !entry.activeTransfer {
				entry.activeTransfer = true
				go downloadSelectedWhenReady(entry.torrent, entry.record.SelectedFiles)
			}
			entry.torrent.AllowDataDownload()
			entry.torrent.AllowDataUpload()
			active++
			continue
		}
		entry.activeTransfer = false
		entry.torrent.DisallowDataDownload()
		entry.torrent.DisallowDataUpload()
	}
}

func downloadSelectedWhenReady(torrent *libtorrent.Torrent, selected []bool) {
	<-torrent.GotInfo()
	if len(selected) == 0 {
		torrent.DownloadAll()
		return
	}
	for index, file := range torrent.Files() {
		if isFileSelected(selected, index) {
			file.Download()
		} else {
			file.SetPriority(libtorrent.PiecePriorityNone)
		}
	}
}

func snapshotEntry(entry *managedTorrent, now time.Time) models.TorrentInfo {
	torrent := entry.torrent
	stats := torrent.Stats()
	downloadedOnWire := stats.BytesReadUsefulData.Int64()
	uploadedOnWire := stats.BytesWrittenData.Int64()
	downloadRate, uploadRate := rates(entry, downloadedOnWire, uploadedOnWire, now)

	name := torrent.Name()
	if name == "" {
		name = "Fetching metadata"
	}
	info := models.TorrentInfo{
		ID:                entry.record.ID,
		Name:              name,
		DownloadDirectory: entry.record.DownloadDirectory,
		DownloadRate:      downloadRate,
		UploadRate:        uploadRate,
		Peers:             stats.ActivePeers,
		Seeds:             stats.ConnectedSeeders,
		Paused:            entry.paused,
	}
	if torrent.Info() != nil {
		info.Size = torrent.Length()
		info.Downloaded = torrent.BytesCompleted()
		info.Uploaded = uploadedOnWire
		info.Remaining = torrent.BytesMissing()
	} else if entry.record.KnownSize > 0 {
		info.Name = entry.record.KnownName
		info.Size = entry.record.KnownSize
		info.Downloaded = entry.record.KnownDownloaded
		info.Remaining = max(0, info.Size-info.Downloaded)
	}
	if info.Size > 0 {
		info.Progress = math.Min(100, (float64(info.Downloaded)/float64(info.Size))*100)
	}

	switch {
	case entry.stopped && info.Size > 0 && info.Downloaded >= info.Size:
		info.Status = models.TorrentStatusCompleted
	case entry.stopped:
		info.Status = models.TorrentStatusStopped
	case entry.paused:
		info.Status = models.TorrentStatusPaused
	case torrent.Info() == nil:
		info.Status = models.TorrentStatusMetadata
	case entry.awaitingSelection:
		info.Status = models.TorrentStatusSelection
	case torrent.Complete().Bool():
		info.Status = models.TorrentStatusSeeding
	case downloadRate > 0:
		info.Status = models.TorrentStatusDownloading
	default:
		info.Status = models.TorrentStatusQueued
	}
	if info.Status == models.TorrentStatusDownloading && downloadRate > 0 && info.Remaining > 0 {
		info.ETASeconds = int64(math.Ceil(float64(info.Remaining) / float64(downloadRate)))
	}
	return info
}

func metainfoBytes(metaInfo metainfo.MetaInfo) ([]byte, error) {
	var buffer bytes.Buffer
	if err := metaInfo.Write(&buffer); err != nil {
		return nil, err
	}
	if buffer.Len() > maxTorrentFileSize {
		return nil, fmt.Errorf("torrent metainfo exceeds %d MiB", maxTorrentFileSize>>20)
	}
	return buffer.Bytes(), nil
}

func rates(entry *managedTorrent, downloaded, uploaded int64, now time.Time) (int64, int64) {
	if entry.lastUpdatedAt.IsZero() {
		entry.lastDownload = downloaded
		entry.lastUpload = uploaded
		entry.lastUpdatedAt = now
		return 0, 0
	}
	elapsed := now.Sub(entry.lastUpdatedAt).Seconds()
	if elapsed <= 0 {
		return 0, 0
	}
	downloadRate := int64(math.Max(0, float64(downloaded-entry.lastDownload)/elapsed))
	uploadRate := int64(math.Max(0, float64(uploaded-entry.lastUpload)/elapsed))
	entry.lastDownload = downloaded
	entry.lastUpload = uploaded
	entry.lastUpdatedAt = now
	return downloadRate, uploadRate
}

func torrentID(spec *libtorrent.TorrentSpec) string {
	if !spec.InfoHash.IsZero() {
		return "v1:" + spec.InfoHash.String()
	}
	if hash, ok := spec.InfoHashV2.AsTuple(); ok {
		return "v2:" + hash.String()
	}
	return ""
}

func specFromRecord(record storage.TorrentRecord) (*libtorrent.TorrentSpec, error) {
	if record.CachedMetainfo != "" {
		rawMetainfo, err := base64.StdEncoding.DecodeString(record.CachedMetainfo)
		if err != nil {
			return nil, fmt.Errorf("decode cached metainfo: %w", err)
		}
		metaInfo, err := metainfo.Load(bytes.NewReader(rawMetainfo))
		if err != nil {
			return nil, fmt.Errorf("parse cached metainfo: %w", err)
		}
		return libtorrent.TorrentSpecFromMetaInfoErr(metaInfo)
	}
	switch record.SourceType {
	case sourceTypeMagnet:
		return libtorrent.TorrentSpecFromMagnetUri(record.Source)
	case sourceTypeTorrentFile:
		rawMetainfo, err := base64.StdEncoding.DecodeString(record.Source)
		if err != nil {
			return nil, fmt.Errorf("decode persisted metainfo: %w", err)
		}
		metaInfo, err := metainfo.Load(bytes.NewReader(rawMetainfo))
		if err != nil {
			return nil, fmt.Errorf("parse persisted metainfo: %w", err)
		}
		return libtorrent.TorrentSpecFromMetaInfoErr(metaInfo)
	default:
		return nil, fmt.Errorf("unsupported persisted source type %q", record.SourceType)
	}
}

func validateDownloadDirectory(directory string) error {
	if strings.TrimSpace(directory) == "" || !filepath.IsAbs(directory) {
		return errors.New("download directory must be an absolute path")
	}
	info, err := os.Stat(directory)
	if err != nil {
		return fmt.Errorf("access download directory: %w", err)
	}
	if !info.IsDir() {
		return errors.New("download directory is not a directory")
	}
	return nil
}

func setRateLimit(limiter *rate.Limiter, bytesPerSecond int64) {
	if bytesPerSecond == 0 {
		limiter.SetLimit(rate.Inf)
		limiter.SetBurst(defaultRateLimiterBurst)
		return
	}
	burst := bytesPerSecond
	if burst < minimumRateLimiterBurst {
		burst = minimumRateLimiterBurst
	}
	if burst > defaultRateLimiterBurst {
		burst = defaultRateLimiterBurst
	}
	limiter.SetLimit(rate.Limit(bytesPerSecond))
	limiter.SetBurst(int(burst))
}

func cloneTrackerTiers(trackers [][]string) [][]string {
	if len(trackers) == 0 {
		return nil
	}
	clone := make([][]string, len(trackers))
	for index := range trackers {
		clone[index] = append([]string(nil), trackers[index]...)
	}
	return clone
}

func normalizedSelection(selection []bool, size int) []bool {
	if size <= 0 {
		return nil
	}
	normalized := make([]bool, size)
	for index := range normalized {
		normalized[index] = isFileSelected(selection, index)
	}
	return normalized
}

func isFileSelected(selection []bool, index int) bool {
	return len(selection) == 0 || (index >= 0 && index < len(selection) && selection[index])
}

func hasSelectedFiles(selection []bool) bool {
	if len(selection) == 0 {
		return true
	}
	for _, selected := range selection {
		if selected {
			return true
		}
	}
	return false
}
