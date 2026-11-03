package models

// TorrentStatus is the current user-facing state of a torrent.
type TorrentStatus string

const (
	TorrentStatusMetadata    TorrentStatus = "Fetching metadata"
	TorrentStatusSelection   TorrentStatus = "Awaiting file selection"
	TorrentStatusQueued      TorrentStatus = "Waiting for peers"
	TorrentStatusDownloading TorrentStatus = "Downloading"
	TorrentStatusSeeding     TorrentStatus = "Seeding"
	TorrentStatusCompleted   TorrentStatus = "Completed"
	TorrentStatusPaused      TorrentStatus = "Paused"
	TorrentStatusStopped     TorrentStatus = "Stopped"
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
	Uploaded          int64         `json:"uploaded"`
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

// TorrentDetails contains the information that can safely be inspected for one
// live torrent. File, peer, and tracker data is omitted until metainfo exists.
type TorrentDetails struct {
	Info            TorrentInfo      `json:"info"`
	MetadataPending bool             `json:"metadataPending"`
	Files           []TorrentFile    `json:"files"`
	Peers           []TorrentPeer    `json:"peers"`
	Trackers        []TorrentTracker `json:"trackers"`
}

type TorrentFile struct {
	Index      int    `json:"index"`
	Path       string `json:"path"`
	Size       int64  `json:"size"`
	Downloaded int64  `json:"downloaded"`
	Selected   bool   `json:"selected"`
}

// TorrentPeer describes a connected peer. Addresses are available only while
// the torrent engine has an active connection to that peer.
type TorrentPeer struct {
	Address      string  `json:"address"`
	Network      string  `json:"network"`
	Client       string  `json:"client"`
	DownloadRate float64 `json:"downloadRate"`
	UploadRate   float64 `json:"uploadRate"`
}

// TorrentTracker is a configured announce URL. The torrent library does not
// expose a stable public per-tracker health API, so no health is inferred.
type TorrentTracker struct {
	URL  string `json:"url"`
	Tier int    `json:"tier"`
}

// TorrentImportResult reports the outcome of a native file-drop import without
// exposing the user's local path to the frontend or logs.
type TorrentImportResult struct {
	Added int    `json:"added"`
	Error string `json:"error,omitempty"`
}

// Settings contains the currently supported local application preferences.
type Settings struct {
	DownloadDirectory      string `json:"downloadDirectory"`
	Theme                  string `json:"theme"`
	DownloadLimit          int64  `json:"downloadLimit"`
	UploadLimit            int64  `json:"uploadLimit"`
	MaxConcurrentDownloads int    `json:"maxConcurrentDownloads"`
}
