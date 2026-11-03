package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/nikitrajkovski/torrentgo/internal/models"
)

const applicationDirectory = "torrentgo"

const (
	themeDark  = "dark"
	themeLight = "light"
)

// Store persists user preferences in the operating system configuration
// directory. It deliberately contains no torrent content or credentials.
type Store struct {
	mu   sync.Mutex
	path string
}

func NewStore() (*Store, error) {
	configDirectory, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("find user configuration directory: %w", err)
	}

	return NewStoreAt(filepath.Join(configDirectory, applicationDirectory, "settings.json")), nil
}

func NewStoreAt(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Load() (models.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	defaultSettings, err := defaultSettings()
	if err != nil {
		return models.Settings{}, err
	}

	contents, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		if err := s.saveLocked(defaultSettings); err != nil {
			return models.Settings{}, err
		}
		return defaultSettings, nil
	}
	if err != nil {
		return models.Settings{}, fmt.Errorf("read settings: %w", err)
	}

	settings := defaultSettings
	if err := json.Unmarshal(contents, &settings); err != nil {
		return models.Settings{}, fmt.Errorf("decode settings: %w", err)
	}
	if err := validate(settings); err != nil {
		return models.Settings{}, fmt.Errorf("validate settings: %w", err)
	}
	return settings, nil
}

func (s *Store) Save(settings models.Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validate(settings); err != nil {
		return err
	}
	return s.saveLocked(settings)
}

func (s *Store) saveLocked(settings models.Settings) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}

	contents, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	if err := os.WriteFile(s.path, contents, 0o600); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return nil
}

func defaultSettings() (models.Settings, error) {
	homeDirectory, err := os.UserHomeDir()
	if err != nil {
		return models.Settings{}, fmt.Errorf("find user home directory: %w", err)
	}
	downloadDirectory := filepath.Join(homeDirectory, "Downloads")
	if err := os.MkdirAll(downloadDirectory, 0o755); err != nil {
		return models.Settings{}, fmt.Errorf("create default download directory: %w", err)
	}
	return models.Settings{DownloadDirectory: downloadDirectory, Theme: themeDark}, nil
}

func validate(settings models.Settings) error {
	directory := strings.TrimSpace(settings.DownloadDirectory)
	if directory == "" {
		return errors.New("download directory is required")
	}
	if !filepath.IsAbs(directory) {
		return errors.New("download directory must be an absolute path")
	}
	info, err := os.Stat(directory)
	if err != nil {
		return fmt.Errorf("access download directory: %w", err)
	}
	if !info.IsDir() {
		return errors.New("download directory is not a directory")
	}
	if settings.Theme != themeDark && settings.Theme != themeLight {
		return errors.New("theme must be dark or light")
	}
	return nil
}
