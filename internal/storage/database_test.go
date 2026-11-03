package storage

import (
	"path/filepath"
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
