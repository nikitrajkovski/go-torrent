package config

import (
	"path/filepath"
	"testing"

	"github.com/nikitrajkovski/torrentgo/internal/models"
)

func TestStorePersistsSettings(t *testing.T) {
	t.Parallel()
	downloadDirectory := t.TempDir()
	store := NewStoreAt(filepath.Join(t.TempDir(), "settings.json"))
	want := models.Settings{DownloadDirectory: downloadDirectory, Theme: "light"}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got != want {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
}
