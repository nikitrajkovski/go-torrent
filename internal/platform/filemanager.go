// Package platform contains small, OS-facing helpers that do not belong to
// the torrent engine. Keeping them here lets the engine remain usable without
// Wails and makes their side effects explicit at the service boundary.
package platform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// OpenDirectory opens an existing local directory in the user's file manager.
// It deliberately accepts directories only: Torrade never opens or executes
// a downloaded file on the user's behalf.
func OpenDirectory(directory string) error {
	if directory == "" {
		return errors.New("download directory is empty")
	}
	absolute, err := filepath.Abs(filepath.Clean(directory))
	if err != nil {
		return fmt.Errorf("resolve download directory: %w", err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return fmt.Errorf("access download directory: %w", err)
	}
	if !info.IsDir() {
		return errors.New("download location is not a directory")
	}
	if err := openDirectory(absolute); err != nil {
		return fmt.Errorf("open download directory: %w", err)
	}
	return nil
}
