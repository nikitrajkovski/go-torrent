package storage

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestStoreCRUD(t *testing.T) {
	t.Parallel()
	store := NewStoreAt(filepath.Join(t.TempDir(), "torrents.json"))
	record := TorrentRecord{
		ID:                "abc",
		Source:            "magnet:?xt=urn:btih:abc",
		SourceType:        "magnet",
		DownloadDirectory: t.TempDir(),
		AddedAt:           time.Now().UTC(),
	}
	if err := store.Upsert(record); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if err := store.UpdatePaused(record.ID, true); err != nil {
		t.Fatalf("UpdatePaused() error = %v", err)
	}
	records, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(records) != 1 || !records[0].Paused {
		t.Fatalf("Load() = %#v, want one paused record", records)
	}
	if err := store.Delete(record.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	records, err = store.Load()
	if err != nil {
		t.Fatalf("Load() after Delete() error = %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("Load() after Delete() = %#v, want no records", records)
	}
}

func TestStoreSetAllPaused(t *testing.T) {
	t.Parallel()
	store := NewStoreAt(filepath.Join(t.TempDir(), "torrents.json"))
	for _, id := range []string{"first", "second"} {
		if err := store.Upsert(TorrentRecord{
			ID:                id,
			Source:            "magnet:?xt=urn:btih:" + id,
			SourceType:        "magnet",
			DownloadDirectory: t.TempDir(),
			AddedAt:           time.Now().UTC(),
		}); err != nil {
			t.Fatalf("Upsert(%q) error = %v", id, err)
		}
	}
	if err := store.SetAllPaused(true); err != nil {
		t.Fatalf("SetAllPaused() error = %v", err)
	}
	records, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	for _, record := range records {
		if !record.Paused {
			t.Fatalf("record %q was not paused", record.ID)
		}
	}
}

func TestStoreUpdateStatePersistsStopped(t *testing.T) {
	t.Parallel()
	store := NewStoreAt(filepath.Join(t.TempDir(), "torrents.json"))
	record := TorrentRecord{
		ID:                "stopped",
		Source:            "magnet:?xt=urn:btih:stopped",
		SourceType:        "magnet",
		DownloadDirectory: t.TempDir(),
		AddedAt:           time.Now().UTC(),
	}
	if err := store.Upsert(record); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if err := store.UpdateState(record.ID, false, true); err != nil {
		t.Fatalf("UpdateState() error = %v", err)
	}
	records, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(records) != 1 || records[0].Paused || !records[0].Stopped {
		t.Fatalf("Load() = %#v, want one stopped record", records)
	}
}

func TestStoreUpdateSelectedFiles(t *testing.T) {
	t.Parallel()
	store := NewStoreAt(filepath.Join(t.TempDir(), "torrents.json"))
	record := TorrentRecord{ID: "files", Source: "magnet:?xt=urn:btih:files", SourceType: "magnet", DownloadDirectory: t.TempDir(), AddedAt: time.Now().UTC()}
	if err := store.Upsert(record); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if err := store.UpdateSelectedFiles(record.ID, []bool{true, false, true}); err != nil {
		t.Fatalf("UpdateSelectedFiles() error = %v", err)
	}
	records, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(records) != 1 || !reflect.DeepEqual(records[0].SelectedFiles, []bool{true, false, true}) {
		t.Fatalf("Load() = %#v, want persisted selected files", records)
	}
}

func TestStoreUpdateRecoveryData(t *testing.T) {
	t.Parallel()
	store := NewStoreAt(filepath.Join(t.TempDir(), "torrents.json"))
	record := TorrentRecord{ID: "recovery", Source: "magnet:?xt=urn:btih:recovery", SourceType: "magnet", DownloadDirectory: t.TempDir(), AddedAt: time.Now().UTC()}
	if err := store.Upsert(record); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if err := store.UpdateRecoveryData(record.ID, "bWV0YWluZm8=", "example.iso", 1024, 1024); err != nil {
		t.Fatalf("UpdateRecoveryData() error = %v", err)
	}
	records, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(records) != 1 || records[0].CachedMetainfo != "bWV0YWluZm8=" || records[0].KnownName != "example.iso" || records[0].KnownSize != 1024 || records[0].KnownDownloaded != 1024 {
		t.Fatalf("Load() = %#v, want persisted recovery data", records)
	}
}
