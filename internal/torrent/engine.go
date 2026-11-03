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

	libtorrent "github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	libstorage "github.com/anacrolix/torrent/storage"

	"github.com/nikitrajkovski/torrentgo/internal/models"
	"github.com/nikitrajkovski/torrentgo/internal/storage"
)

const (
	sourceTypeMagnet      = "magnet"
	sourceTypeTorrentFile = "torrent-file"
	maxTorrentFileSize    = 10 << 20
)

type managedTorrent struct {
	torrent       *libtorrent.Torrent
	record        storage.TorrentRecord
	paused        bool
	lastDownload  int64
	lastUpload    int64
	lastUpdatedAt time.Time
}

// Engine is the local BitTorrent boundary. It has no dependency on Wails or
// React, which keeps protocol and filesystem operations testable in isolation.
type Engine struct {
	mu      sync.Mutex
	client  *libtorrent.Client
	entries map[string]*managedTorrent
	store   *storage.Store
	logger  *slog.Logger
	closed  bool
}

func New(store *storage.Store, logger *slog.Logger) (*Engine, error) {
	if store == nil {
		return nil, errors.New("torrent store is required")
	}
	if logger == nil {
		logger = slog.Default()
	}

	clientConfig := libtorrent.NewDefaultClientConfig()
	clientConfig.DataDir = filepath.Dir(store.Path())
	clientConfig.Seed = true
	clientConfig.NoDefaultPortForwarding = true
	clientConfig.Slogger = logger
	client, err := libtorrent.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("create torrent client: %w", err)
	}

	engine := &Engine{
		client:  client,
		entries: make(map[string]*managedTorrent),
		store:   store,
		logger:  logger,
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
		AddedAt:           time.Now().UTC(),
	}
	return e.addSpec(spec, record, false)
}

// AddTorrentFile reads metainfo with a bounded input size, then keeps an
// encoded local copy for restart recovery. It never follows or executes data
// referenced by the metainfo file.
func (e *Engine) AddTorrentFile(path, downloadDirectory string) (models.TorrentInfo, error) {
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
		AddedAt:           time.Now().UTC(),
	}
	return e.addSpec(spec, record, false)
}

func (e *Engine) Pause(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	entry, err := e.entryLocked(id)
	if err != nil {
		return err
	}
	entry.torrent.DisallowDataDownload()
	entry.torrent.DisallowDataUpload()
	entry.paused = true
	if err := e.store.UpdatePaused(id, true); err != nil {
		return fmt.Errorf("persist paused torrent: %w", err)
	}
	return nil
}

func (e *Engine) Resume(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	entry, err := e.entryLocked(id)
	if err != nil {
		return err
	}
	entry.torrent.AllowDataDownload()
	entry.torrent.AllowDataUpload()
	entry.paused = false
	if err := e.store.UpdatePaused(id, false); err != nil {
		return fmt.Errorf("persist resumed torrent: %w", err)
	}
	go downloadAllWhenReady(entry.torrent)
	return nil
}

// Remove drops the torrent from the engine and its persistent index. TorrentGo
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
	return nil
}

func (e *Engine) Snapshot() models.TorrentSnapshot {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	snapshot := models.TorrentSnapshot{
		Torrents:     make([]models.TorrentInfo, 0, len(e.entries)),
		NetworkReady: !e.closed,
	}
	for _, entry := range e.entries {
		info := snapshotEntry(entry, now)
		snapshot.Torrents = append(snapshot.Torrents, info)
		snapshot.DownloadRate += info.DownloadRate
		snapshot.UploadRate += info.UploadRate
		if info.Status == models.TorrentStatusDownloading || info.Status == models.TorrentStatusMetadata {
			snapshot.Active++
		}
	}
	sort.Slice(snapshot.Torrents, func(left, right int) bool {
		return strings.ToLower(snapshot.Torrents[left].Name) < strings.ToLower(snapshot.Torrents[right].Name)
	})
	return snapshot
}

func (e *Engine) Close() error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil
	}
	e.closed = true
	client := e.client
	e.mu.Unlock()

	if closeErrors := client.Close(); len(closeErrors) > 0 {
		return fmt.Errorf("close torrent client: %w", errors.Join(closeErrors...))
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
	spec.Storage = libstorage.NewFile(record.DownloadDirectory)
	spec.DisallowDataDownload = record.Paused
	spec.DisallowDataUpload = record.Paused

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return models.TorrentInfo{}, errors.New("torrent engine is closed")
	}
	if _, exists := e.entries[record.ID]; exists {
		return models.TorrentInfo{}, errors.New("this torrent has already been added")
	}
	torrent, added, err := e.client.AddTorrentSpec(spec)
	if err != nil {
		return models.TorrentInfo{}, fmt.Errorf("add torrent to engine: %w", err)
	}
	if !added {
		return models.TorrentInfo{}, errors.New("this torrent has already been added")
	}
	entry := &managedTorrent{torrent: torrent, record: record, paused: record.Paused}
	e.entries[record.ID] = entry
	if !restoring {
		if err := e.store.Upsert(record); err != nil {
			torrent.Drop()
			delete(e.entries, record.ID)
			return models.TorrentInfo{}, fmt.Errorf("persist torrent: %w", err)
		}
	}
	if !entry.paused {
		go downloadAllWhenReady(torrent)
	}
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

func downloadAllWhenReady(torrent *libtorrent.Torrent) {
	<-torrent.GotInfo()
	torrent.DownloadAll()
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
		Downloaded:        torrent.BytesCompleted(),
		Remaining:         torrent.BytesMissing(),
		DownloadRate:      downloadRate,
		UploadRate:        uploadRate,
		Peers:             stats.ActivePeers,
		Seeds:             stats.ConnectedSeeders,
		Paused:            entry.paused,
	}
	if torrent.Info() != nil {
		info.Size = torrent.Length()
	}
	if info.Size > 0 {
		info.Progress = math.Min(100, (float64(info.Downloaded)/float64(info.Size))*100)
	}

	switch {
	case entry.paused:
		info.Status = models.TorrentStatusPaused
	case torrent.Info() == nil:
		info.Status = models.TorrentStatusMetadata
	case torrent.Complete().Bool():
		info.Status = models.TorrentStatusSeeding
	case downloadRate > 0:
		info.Status = models.TorrentStatusDownloading
	default:
		info.Status = models.TorrentStatusDownloading
	}
	if info.Status == models.TorrentStatusDownloading && downloadRate > 0 && info.Remaining > 0 {
		info.ETASeconds = int64(math.Ceil(float64(info.Remaining) / float64(downloadRate)))
	}
	return info
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
