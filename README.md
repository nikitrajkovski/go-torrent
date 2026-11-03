# TorrentGo

TorrentGo is a lightweight, local-first BitTorrent desktop client for Windows, Linux, and macOS. It uses Go, Wails v3, React, TypeScript, Vite, Tailwind CSS, Lucide, and `github.com/anacrolix/torrent`.

## Status

**Phases 1–3 and the core transfer-management work from Phase 4 are implemented.** TorrentGo can add legal `.torrent` files and magnet links, download to a selected local folder, seed completed content, show real transfer progress and rates, and recover its local torrent list after a restart.

There is no telemetry, advertising, cloud download service, or hidden background process.

## Structure

```text
TorrentGo/
├── build/                 # Wails application metadata
├── cmd/torrentgo/         # Reserved for future CLI entry points
├── frontend/              # React + TypeScript + Vite + Tailwind application
├── internal/
│   ├── config/            # Local settings store
│   ├── models/            # Wails-serialisable view models
│   ├── services/          # Wails-facing service boundary
│   ├── storage/           # Persistent local torrent index
│   └── torrent/           # anacrolix/torrent integration
├── main.go                # Wails desktop entry point
└── go.mod
```

## Implemented features

- Add valid magnet links and `.torrent` files through native file dialogs.
- Choose a local destination folder through a native OS folder picker.
- Download and seed through `anacrolix/torrent`, using its available DHT, tracker, PEX, uTP, TCP, v1, and v2 support.
- Use verified file-backed piece storage, including detection of existing completed pieces after restart.
- Show actual completed bytes, size, progress, download/upload rates, ETA, connected peers, and connected seeders.
- Pause, resume, and remove individual torrents, or pause/resume the complete queue with the toolbar controls. Completed torrents have a separate Stop seeding control that keeps their files while preventing further uploads. These state changes are persisted atomically so they survive a restart.
- Distinguish metadata retrieval, waiting for peers, active download, paused, and seeding states without simulated progress. Display bytes uploaded for the current client session alongside actual transfer rates.
- Remove deliberately preserves downloaded payload files.
- Persist local settings and torrent source records in the operating system configuration directory. `.torrent` metainfo is kept locally so recovery does not depend on its original path.
- Push a complete snapshot to the UI once per second instead of polling each torrent independently.
- Provide sidebar filters, search, a selected-torrent general-details panel, and a real settings screen with a persisted light/dark theme preference.
- Accept `.torrent` files dropped from the native file manager onto the application window.
- Select individual files after metainfo is available; selection is persisted and changes the torrent library's file priorities.
- Apply persisted global upload/download rate limits and a concurrent-download queue without restarting the client.
- Display connected peer details and configured tracker URLs when the torrent library supplies them.

## Prerequisites

- Go 1.26 or newer
- Node.js 22 or newer with npm
- Wails CLI v3.0.0: `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0`

For Linux desktop builds, install the native libraries reported by `wails3 doctor` (typically `libgtk-4-dev`, `libwebkitgtk-6.0-dev`, and `libsoup-3.0-dev`). Windows 10/11 requires WebView2, which is normally already available.

## Run in development

```bash
wails3 dev
```

Wails automatically regenerates the TypeScript bindings in `frontend/bindings/` during development. They are committed so that a frontend-only typecheck works after a fresh clone as well.

## Checks

```bash
go test ./...
cd frontend && npm run build
```

On Linux, `go test ./...` requires the GTK/WebKit development packages because the Wails desktop package is part of the root build. The platform-independent checks are:

```bash
go test ./internal/config ./internal/storage ./internal/torrent
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o torrentgo.exe .
```

## Privacy and security notes

BitTorrent peers and trackers can observe your public IP address. BitTorrent protocol encryption/obfuscation is not anonymity; TorrentGo does not claim it is. TorrentGo never executes downloaded files and does not delete payload data without a future explicit confirmation flow. Automatic router port forwarding is disabled by default.

Malformed `.torrent` inputs are size-limited and parsed in Go before they reach the engine. The storage backend uses the torrent library’s safe native-path handling for downloaded files.

## Current limitations

- Context menus, notifications, tray behavior, installers, proxy configuration, and tracker-health reporting are planned later phases.
- File selection becomes available after metadata arrives. Tracker URLs are shown as configured; the engine does not expose a stable public per-tracker health API.
- Downloaded files cannot be removed from the UI yet. This prevents accidental deletion from a destination folder shared by multiple torrents.
- TorrentGo has no anonymity layer or proxy configuration yet.
