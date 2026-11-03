package torrent

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	libtorrent "github.com/anacrolix/torrent"

	"github.com/nikitrajkovski/torrentgo/internal/models"
	"github.com/nikitrajkovski/torrentgo/internal/storage"
)

func TestTorrentIDUsesV1InfoHash(t *testing.T) {
	t.Parallel()
	spec, err := libtorrent.TorrentSpecFromMagnetUri("magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567")
	if err != nil {
		t.Fatalf("TorrentSpecFromMagnetUri() error = %v", err)
	}
	got := torrentID(spec)
	want := "v1:0123456789abcdef0123456789abcdef01234567"
	if got != want {
		t.Fatalf("torrentID() = %q, want %q", got, want)
	}
}

func TestSpecFromRecordRejectsUnknownSource(t *testing.T) {
	t.Parallel()
	_, err := specFromRecord(storage.TorrentRecord{SourceType: "unknown"})
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("specFromRecord() error = %v, want unsupported-source error", err)
	}
}

func TestRatesUsesElapsedTime(t *testing.T) {
	t.Parallel()
	entry := &managedTorrent{
		lastDownload:  100,
		lastUpload:    50,
		lastUpdatedAt: time.Unix(100, 0),
	}
	downloadRate, uploadRate := rates(entry, 300, 100, time.Unix(102, 0))
	if downloadRate != 100 || uploadRate != 25 {
		t.Fatalf("rates() = (%d, %d), want (100, 25)", downloadRate, uploadRate)
	}
}

func TestAddMagnetReturnsSnapshotBeforeMetadata(t *testing.T) {
	store := storage.NewStoreAt(filepath.Join(t.TempDir(), "torrents.json"))
	engine, err := New(store, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := engine.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	info, err := engine.AddMagnet(
		"magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567&dn=metadata-test",
		t.TempDir(),
	)
	if err != nil {
		t.Fatalf("AddMagnet() error = %v", err)
	}
	if info.Status != "Fetching metadata" {
		t.Fatalf("AddMagnet() status = %q, want metadata state", info.Status)
	}
	if info.Size != 0 || info.Downloaded != 0 || info.Remaining != 0 {
		t.Fatalf("AddMagnet() metadata snapshot = %#v, want no piece-based values", info)
	}
}

func TestPauseAllAndResumeAllPersistState(t *testing.T) {
	store := storage.NewStoreAt(filepath.Join(t.TempDir(), "torrents.json"))
	engine, err := New(store, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := engine.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	for _, hash := range []string{
		"0123456789abcdef0123456789abcdef01234567",
		"1123456789abcdef0123456789abcdef01234567",
	} {
		if _, err := engine.AddMagnet("magnet:?xt=urn:btih:"+hash, t.TempDir()); err != nil {
			t.Fatalf("AddMagnet() error = %v", err)
		}
	}
	if err := engine.PauseAll(); err != nil {
		t.Fatalf("PauseAll() error = %v", err)
	}
	assertAllPaused(t, engine.Snapshot(), true)
	if err := engine.ResumeAll(); err != nil {
		t.Fatalf("ResumeAll() error = %v", err)
	}
	assertAllPaused(t, engine.Snapshot(), false)
}

func TestRecoveryRestoresPausedTorrent(t *testing.T) {
	store := storage.NewStoreAt(filepath.Join(t.TempDir(), "torrents.json"))
	downloadDirectory := t.TempDir()
	engine, err := New(store, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	added, err := engine.AddMagnet(
		"magnet:?xt=urn:btih:2123456789abcdef0123456789abcdef01234567&dn=recovery-test",
		downloadDirectory,
	)
	if err != nil {
		t.Fatalf("AddMagnet() error = %v", err)
	}
	if err := engine.Pause(added.ID); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}
	if err := engine.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	recovered, err := New(store, nil)
	if err != nil {
		t.Fatalf("New() after restart error = %v", err)
	}
	t.Cleanup(func() {
		if err := recovered.Close(); err != nil {
			t.Errorf("recovered Close() error = %v", err)
		}
	})
	snapshot := recovered.Snapshot()
	if len(snapshot.Torrents) != 1 || snapshot.Torrents[0].ID != added.ID || !snapshot.Torrents[0].Paused {
		t.Fatalf("recovered Snapshot() = %#v, want paused persisted torrent", snapshot)
	}
}

func TestStopPersistsAndRecoversTorrent(t *testing.T) {
	store := storage.NewStoreAt(filepath.Join(t.TempDir(), "torrents.json"))
	downloadDirectory := t.TempDir()
	engine, err := New(store, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	added, err := engine.AddMagnet(
		"magnet:?xt=urn:btih:3123456789abcdef0123456789abcdef01234567&dn=stop-test",
		downloadDirectory,
	)
	if err != nil {
		t.Fatalf("AddMagnet() error = %v", err)
	}
	if err := engine.Stop(added.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	assertStopped(t, engine.Snapshot(), added.ID)
	if err := engine.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	recovered, err := New(store, nil)
	if err != nil {
		t.Fatalf("New() after restart error = %v", err)
	}
	t.Cleanup(func() {
		if err := recovered.Close(); err != nil {
			t.Errorf("recovered Close() error = %v", err)
		}
	})
	assertStopped(t, recovered.Snapshot(), added.ID)
}

func TestConcurrentDownloadLimitQueuesAdditionalTorrents(t *testing.T) {
	store := storage.NewStoreAt(filepath.Join(t.TempDir(), "torrents.json"))
	engine, err := New(store, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = engine.Close() })
	if err := engine.UpdateSettings(models.Settings{MaxConcurrentDownloads: 1}); err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}
	for _, hash := range []string{"4123456789abcdef0123456789abcdef01234567", "5123456789abcdef0123456789abcdef01234567"} {
		if _, err := engine.AddMagnet("magnet:?xt=urn:btih:"+hash, t.TempDir()); err != nil {
			t.Fatalf("AddMagnet() error = %v", err)
		}
	}
	engine.mu.Lock()
	active := 0
	for _, entry := range engine.entries {
		if entry.activeTransfer {
			active++
		}
	}
	engine.mu.Unlock()
	if active != 1 {
		t.Fatalf("active transfers = %d, want 1", active)
	}
	if err := engine.UpdateSettings(models.Settings{MaxConcurrentDownloads: 2}); err != nil {
		t.Fatalf("UpdateSettings(2) error = %v", err)
	}
	if got := activeTransferCount(engine); got != 2 {
		t.Fatalf("active transfers after increasing limit = %d, want 2", got)
	}
	if err := engine.UpdateSettings(models.Settings{MaxConcurrentDownloads: 1}); err != nil {
		t.Fatalf("UpdateSettings(1) error = %v", err)
	}
	if got := activeTransferCount(engine); got != 1 {
		t.Fatalf("active transfers after decreasing limit = %d, want 1", got)
	}
}

func TestSelectionHelpers(t *testing.T) {
	t.Parallel()
	if got := normalizedSelection(nil, 3); !reflect.DeepEqual(got, []bool{true, true, true}) {
		t.Fatalf("normalizedSelection(nil, 3) = %#v", got)
	}
	if !hasSelectedFiles([]bool{false, true}) || hasSelectedFiles([]bool{false, false}) {
		t.Fatal("hasSelectedFiles() returned an unexpected value")
	}
}

func assertAllPaused(t *testing.T, snapshot models.TorrentSnapshot, paused bool) {
	t.Helper()
	if len(snapshot.Torrents) != 2 {
		t.Fatalf("Snapshot() count = %d, want 2", len(snapshot.Torrents))
	}
	for _, torrent := range snapshot.Torrents {
		if torrent.Paused != paused {
			t.Fatalf("torrent %q paused = %t, want %t", torrent.ID, torrent.Paused, paused)
		}
	}
}

func assertStopped(t *testing.T, snapshot models.TorrentSnapshot, id string) {
	t.Helper()
	if len(snapshot.Torrents) != 1 {
		t.Fatalf("Snapshot() count = %d, want 1", len(snapshot.Torrents))
	}
	torrent := snapshot.Torrents[0]
	if torrent.ID != id || torrent.Status != models.TorrentStatusStopped || torrent.Paused {
		t.Fatalf("Snapshot() = %#v, want stopped torrent %q", snapshot, id)
	}
}

func activeTransferCount(engine *Engine) int {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	count := 0
	for _, entry := range engine.entries {
		if entry.activeTransfer {
			count++
		}
	}
	return count
}
