package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/nikitrajkovski/torrentgo/internal/config"
	"github.com/nikitrajkovski/torrentgo/internal/models"
	"github.com/nikitrajkovski/torrentgo/internal/torrent"
)

const snapshotEvent = "torrentgo:torrents"

func init() {
	application.RegisterEvent[models.TorrentSnapshot](snapshotEvent)
}

// TorrentService is the narrow Wails-facing API. It intentionally exposes
// serialisable view models instead of torrent-library types.
type TorrentService struct {
	engine        *torrent.Engine
	settingsStore *config.Store

	mu       sync.RWMutex
	settings models.Settings
}

func NewTorrentService(engine *torrent.Engine, settingsStore *config.Store) *TorrentService {
	return &TorrentService{engine: engine, settingsStore: settingsStore}
}

func (s *TorrentService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	settings, err := s.settingsStore.Load()
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}
	s.mu.Lock()
	s.settings = settings
	s.mu.Unlock()

	go s.emitSnapshots(ctx)
	return nil
}

func (s *TorrentService) ServiceShutdown() error {
	return s.engine.Close()
}

func (s *TorrentService) GetTorrents() models.TorrentSnapshot {
	return s.engine.Snapshot()
}

func (s *TorrentService) GetSettings() models.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

func (s *TorrentService) UpdateSettings(settings models.Settings) (models.Settings, error) {
	if err := s.settingsStore.Save(settings); err != nil {
		return models.Settings{}, err
	}
	s.mu.Lock()
	s.settings = settings
	s.mu.Unlock()
	return settings, nil
}

// ChooseDownloadDirectory presents a native folder selector and persists the
// selected existing folder. Cancellation leaves the current setting unchanged.
func (s *TorrentService) ChooseDownloadDirectory() (models.Settings, error) {
	current := s.GetSettings()
	directory, err := application.Get().Dialog.OpenFile().
		SetTitle("Choose download folder").
		SetDirectory(current.DownloadDirectory).
		CanChooseDirectories(true).
		CanChooseFiles(false).
		PromptForSingleSelection()
	if err != nil {
		return models.Settings{}, fmt.Errorf("open download folder dialog: %w", err)
	}
	if directory == "" {
		return current, nil
	}
	return s.UpdateSettings(models.Settings{DownloadDirectory: directory})
}

func (s *TorrentService) AddMagnet(magnet string) (models.TorrentInfo, error) {
	settings := s.GetSettings()
	return s.engine.AddMagnet(magnet, settings.DownloadDirectory)
}

// AddTorrentFile presents a native .torrent picker. No torrent is added when
// the picker is cancelled.
func (s *TorrentService) AddTorrentFile() error {
	path, err := application.Get().Dialog.OpenFile().
		SetTitle("Add torrent file").
		AddFilter("Torrent files", "*.torrent").
		AddFilter("All files", "*.*").
		PromptForSingleSelection()
	if err != nil {
		return fmt.Errorf("open torrent file dialog: %w", err)
	}
	if path == "" {
		return nil
	}
	settings := s.GetSettings()
	_, err = s.engine.AddTorrentFile(path, settings.DownloadDirectory)
	return err
}

func (s *TorrentService) PauseTorrent(id string) error {
	return s.engine.Pause(id)
}

func (s *TorrentService) ResumeTorrent(id string) error {
	return s.engine.Resume(id)
}

func (s *TorrentService) StopTorrent(id string) error {
	return s.engine.Stop(id)
}

func (s *TorrentService) PauseAll() error {
	return s.engine.PauseAll()
}

func (s *TorrentService) ResumeAll() error {
	return s.engine.ResumeAll()
}

func (s *TorrentService) RemoveTorrent(id string, deleteFiles bool) error {
	if deleteFiles {
		return fmt.Errorf("removing downloaded files is not available in Phase 1")
	}
	return s.engine.Remove(id)
}

func (s *TorrentService) emitSnapshots(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			application.Get().Event.Emit(snapshotEvent, s.engine.Snapshot())
		}
	}
}
