package models

// TorrentStatus is the current user-facing state of a torrent.
type TorrentStatus string

const (
	TorrentStatusMetadata    TorrentStatus = "Fetching metadata"
	TorrentStatusDownloading TorrentStatus = "Downloading"
	TorrentStatusSeeding     TorrentStatus = "Seeding"
	TorrentStatusPaused      TorrentStatus = "Paused"
	TorrentStatusError       TorrentStatus = "Error"
)

// TorrentInfo is a lightweight, serialisable snapshot for the desktop UI.
// All byte counts and rates are in bytes or bytes per second.
type TorrentInfo struct {
	ID                string        `json:"id"`
	Name              string        `json:"name"`
	DownloadDirectory string        `json:"downloadDirectory"`
	Size              int64         `json:"size"`
	Downloaded        int64         `json:"downloaded"`
	Remaining         int64         `json:"remaining"`
	Progress          float64       `json:"progress"`
	Status            TorrentStatus `json:"status"`
	DownloadRate      int64         `json:"downloadRate"`
	UploadRate        int64         `json:"uploadRate"`
	ETASeconds        int64         `json:"etaSeconds"`
	Peers             int           `json:"peers"`
	Seeds             int           `json:"seeds"`
	Paused            bool          `json:"paused"`
	Error             string        `json:"error,omitempty"`
}

// TorrentSnapshot is emitted as a single event so the frontend can update its
// complete view without polling individual torrents.
type TorrentSnapshot struct {
	Torrents     []TorrentInfo `json:"torrents"`
	DownloadRate int64         `json:"downloadRate"`
	UploadRate   int64         `json:"uploadRate"`
	Active       int           `json:"active"`
	NetworkReady bool          `json:"networkReady"`
}

// Settings contains the currently supported local application preferences.
type Settings struct {
	DownloadDirectory string `json:"downloadDirectory"`
}
