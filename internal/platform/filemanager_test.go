package platform

import (
	"strings"
	"testing"
)

func TestOpenDirectoryRejectsInvalidLocations(t *testing.T) {
	t.Parallel()
	if err := OpenDirectory(""); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("OpenDirectory(empty) error = %v, want empty-directory error", err)
	}
	if err := OpenDirectory(t.TempDir() + "/missing"); err == nil || !strings.Contains(err.Error(), "access") {
		t.Fatalf("OpenDirectory(missing) error = %v, want access error", err)
	}
}
