package torrent

import (
	"strings"
	"testing"
	"time"

	libtorrent "github.com/anacrolix/torrent"

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
