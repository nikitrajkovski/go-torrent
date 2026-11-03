package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const applicationDirectory = "torrentgo"

// TorrentRecord is the minimal state required to recreate a local torrent.
// Source is either a magnet URI or base64-encoded metainfo bytes. It is never
// written to application logs.
type TorrentRecord struct {
	ID                string     `json:"id"`
	Source            string     `json:"source"`
	SourceType        string     `json:"sourceType"`
	DownloadDirectory string     `json:"downloadDirectory"`
	Paused            bool       `json:"paused"`
	Stopped           bool       `json:"stopped"`
	SelectedFiles     []bool     `json:"selectedFiles,omitempty"`
	Trackers          [][]string `json:"trackers,omitempty"`
	AwaitingSelection bool       `json:"awaitingSelection,omitempty"`
	CachedMetainfo    string     `json:"cachedMetainfo,omitempty"`
	KnownName         string     `json:"knownName,omitempty"`
	KnownSize         int64      `json:"knownSize,omitempty"`
	KnownDownloaded   int64      `json:"knownDownloaded,omitempty"`
	AddedAt           time.Time  `json:"addedAt"`
}

// Store keeps the small torrent index in a local JSON file. A database is not
// warranted for the handful of atomic record operations required in Phase 1.
type Store struct {
	mu   sync.Mutex
	path string
}

func NewStore() (*Store, error) {
	configDirectory, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("find user configuration directory: %w", err)
	}
	return NewStoreAt(filepath.Join(configDirectory, applicationDirectory, "torrents.json")), nil
}

func NewStoreAt(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Load() ([]TorrentRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *Store) Upsert(record TorrentRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.loadLocked()
	if err != nil {
		return err
	}
	for index := range records {
		if records[index].ID == record.ID {
			records[index] = record
			return s.saveLocked(records)
		}
	}
	return s.saveLocked(append(records, record))
}

func (s *Store) UpdatePaused(id string, paused bool) error {
	return s.UpdateState(id, paused, false)
}

// UpdateState persists whether a torrent is temporarily paused or explicitly
// stopped. Stopped torrents remain in the list and can be resumed later.
func (s *Store) UpdateState(id string, paused, stopped bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.loadLocked()
	if err != nil {
		return err
	}
	for index := range records {
		if records[index].ID == id {
			records[index].Paused = paused
			records[index].Stopped = stopped
			return s.saveLocked(records)
		}
	}
	return fmt.Errorf("torrent %q is not persisted", id)
}

// UpdateSelectedFiles persists the choices by stable metainfo file index.
func (s *Store) UpdateSelectedFiles(id string, selected []bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.loadLocked()
	if err != nil {
		return err
	}
	for index := range records {
		if records[index].ID == id {
			records[index].SelectedFiles = append([]bool(nil), selected...)
			return s.saveLocked(records)
		}
	}
	return fmt.Errorf("torrent %q is not persisted", id)
}

// UpdateAwaitingSelection persists whether payload transfer waits for the user
// to confirm the metainfo file selection.
func (s *Store) UpdateAwaitingSelection(id string, awaiting bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.loadLocked()
	if err != nil {
		return err
	}
	for index := range records {
		if records[index].ID == id {
			records[index].AwaitingSelection = awaiting
			return s.saveLocked(records)
		}
	}
	return fmt.Errorf("torrent %q is not persisted", id)
}

// UpdateRecoveryData stores locally acquired metainfo and the last known
// piece totals. This lets a magnet restore its name, size, and completed
// state without waiting for the swarm on the next application launch.
func (s *Store) UpdateRecoveryData(id, cachedMetainfo, name string, size, downloaded int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.loadLocked()
	if err != nil {
		return err
	}
	for index := range records {
		if records[index].ID == id {
			if cachedMetainfo != "" {
				records[index].CachedMetainfo = cachedMetainfo
			}
			records[index].KnownName = name
			records[index].KnownSize = size
			records[index].KnownDownloaded = downloaded
			return s.saveLocked(records)
		}
	}
	return fmt.Errorf("torrent %q is not persisted", id)
}

// SetAllPaused applies one state transition to every persisted torrent using a
// single atomic file replacement. It keeps global start/pause operations from
// leaving a partially-written state behind if the application exits mid-write.
func (s *Store) SetAllPaused(paused bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.loadLocked()
	if err != nil {
		return err
	}
	for index := range records {
		records[index].Paused = paused
		records[index].Stopped = false
	}
	return s.saveLocked(records)
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.loadLocked()
	if err != nil {
		return err
	}
	filtered := records[:0]
	found := false
	for _, record := range records {
		if record.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, record)
	}
	if !found {
		return fmt.Errorf("torrent %q is not persisted", id)
	}
	return s.saveLocked(filtered)
}

func (s *Store) loadLocked() ([]TorrentRecord, error) {
	contents, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return []TorrentRecord{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read torrent store: %w", err)
	}
	var records []TorrentRecord
	if err := json.Unmarshal(contents, &records); err != nil {
		return nil, fmt.Errorf("decode torrent store: %w", err)
	}
	sort.Slice(records, func(left, right int) bool {
		return records[left].AddedAt.Before(records[right].AddedAt)
	})
	return records, nil
}

func (s *Store) saveLocked(records []TorrentRecord) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create torrent store directory: %w", err)
	}
	contents, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("encode torrent store: %w", err)
	}
	temporaryPath := s.path + ".tmp"
	if err := os.WriteFile(temporaryPath, contents, 0o600); err != nil {
		return fmt.Errorf("write temporary torrent store: %w", err)
	}
	if err := os.Rename(temporaryPath, s.path); err != nil {
		return fmt.Errorf("replace torrent store: %w", err)
	}
	return nil
}
