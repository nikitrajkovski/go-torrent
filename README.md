# TorrentGo

TorrentGo is a planned lightweight, local-first BitTorrent desktop client for Windows, Linux, and macOS. It will use Go, Wails v3, React, TypeScript, Vite, Tailwind CSS, Lucide, and `github.com/anacrolix/torrent`.

## Status

This is the **initial project scaffold**. It deliberately has no torrent engine, Wails service bindings, persistence, network activity, or torrent-management UI. The only visible screen confirms that the desktop shell and frontend pipeline are connected.

## Structure

```text
TorrentGo/
├── build/                 # Wails application metadata
├── cmd/torrentgo/         # Reserved for future CLI entry points
├── frontend/              # React + TypeScript + Vite + Tailwind application
├── internal/
│   ├── config/            # Reserved for local settings
│   ├── models/            # Reserved for application models
│   ├── services/          # Reserved for Wails-facing services
│   ├── storage/           # Reserved for local persistence
│   └── torrent/           # Reserved for the BitTorrent engine
├── main.go                # Wails desktop entry point
└── go.mod
```

## Prerequisites

- Go 1.26 or newer
- Node.js 22 or newer with npm
- Wails CLI v3.0.0: `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0`

For Linux desktop builds, install the native libraries reported by `wails3 doctor`. Windows 10/11 requires WebView2, which is normally already available.

## Run the scaffold

```bash
cd frontend
npm install
npm run build
cd ..
wails3 dev
```

## Checks

```bash
go test ./...
cd frontend && npm run build
```

## Privacy commitment

TorrentGo will not include advertisements, telemetry, analytics SDKs, cloud download infrastructure, or a hidden background service. Future BitTorrent functionality will be local to the user's computer.

## Planned next milestone

Phase 1 will introduce the `anacrolix/torrent` engine, a Wails service boundary, magnet and `.torrent` ingestion, local download-directory selection, real progress reporting, and graceful shutdown.
