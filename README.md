# Torrade

Torrade is a lightweight, local-first BitTorrent desktop client for Windows, Linux, and macOS. It uses Go, Wails v3, React, TypeScript, Vite, Tailwind CSS, Lucide, and `github.com/anacrolix/torrent`.

## Status

**Phases 1–4 and the first release-readiness work from Phase 5 are implemented.** Torrade can add legal `.torrent` files and magnet links, download to a selected local folder, seed completed content, show real transfer progress and rates, and recover its local torrent list after a restart.

There is no telemetry, advertising, cloud download service, or hidden background process.

## Structure

```text
Torrade/
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
- Pause, resume, and remove individual torrents, or pause/resume the complete queue with the toolbar controls. Completed torrents have a separate Stop seeding control that keeps their files while preventing further uploads; stopped completed rows are clearly marked Completed. These state changes are persisted atomically so they survive a restart.
- Distinguish metadata retrieval, waiting for peers, active download, paused, and seeding states without simulated progress. Display bytes uploaded for the current client session alongside actual transfer rates.
- Remove deliberately preserves downloaded payload files.
- Persist local settings and torrent source records in the operating system configuration directory. `.torrent` metainfo is kept locally so recovery does not depend on its original path.
- Cache acquired magnet metainfo and completed-state summaries locally, allowing restored completed torrents to show their name, size, and Completed status immediately on later launches.
- Push a complete snapshot to the UI once per second instead of polling each torrent independently.
- Provide sidebar filters, search, a selected-torrent general-details panel, and a real settings screen with a persisted light/dark theme preference.
- Accept `.torrent` files dropped from the native file manager onto the application window.
- Add torrents through a metadata-first flow: select individual files after metainfo is available, then explicitly start payload transfer. Selections are persisted and change the torrent library's file priorities.
- Apply persisted global upload/download rate limits and a concurrent-download queue without restarting the client.
- Display connected peer details and configured tracker URLs when the torrent library supplies them.
- Provide a right-click action menu and open a torrent's selected download directory in the native file manager. Double-clicking a completed row has the same safe, directory-only behavior.
- Build stripped production executables for Linux and Windows, generate a Linux desktop entry, and create `.deb` and `.rpm` Linux packages.
- Include Windows version metadata, DPI awareness, a per-user NSIS installer definition, and the WebView2 bootstrapper generation step.
- Run backend tests, strict TypeScript compilation, and production builds in the checked-in GitHub Actions workflow.

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

The repository task wraps the same checks and regeneration steps:

```bash
wails3 task check
```

For a slower concurrency check of the torrent engine and local persistence code:

```bash
wails3 task test:race
```

## Release builds

Build artifacts are written to `bin/`, which is intentionally ignored by Git.

### Linux

Create a stripped executable:

```bash
wails3 task linux:build PRODUCTION=true
```

Create the Debian and RPM packages (verified on the Linux development environment):

```bash
wails3 task linux:package
```

This writes `bin/Torrade`, `bin/Torrade.deb`, and `bin/Torrade.rpm`. The packages declare the GTK 4 and WebKitGTK 6 runtime dependencies used by Wails.

### Windows

Create the Windows GUI executable from Linux, Windows, or macOS:

```bash
wails3 task windows:build PRODUCTION=true
```

To produce the installer, install [NSIS](https://nsis.sourceforge.io/) so that `makensis` is on `PATH`, then run:

```bash
wails3 task windows:package
```

The installer is per-user, includes Wails' WebView2 bootstrapper generation step, and is written as `bin/Torrade-amd64-installer.exe`. It is not code-signed yet; configure an Authenticode certificate before a public Windows release to avoid SmartScreen reputation warnings.

### Opening torrent files and magnet links

The installers register Torrade as a handler for `.torrent` files and `magnet:` links. Operating systems require the user to choose a default handler themselves; an app cannot safely override that decision during installation.

On Windows, install Torrade, then open **Settings → Apps → Default apps** and choose Torrade for the `.torrent` file type and the `MAGNET` link type. On Ubuntu, after installing the package, run:

```bash
xdg-mime default io.github.nikitrajkovski.Torrade.desktop application/x-bittorrent
xdg-mime default io.github.nikitrajkovski.Torrade.desktop x-scheme-handler/magnet
```

If Torrade is already open, another magnet link or torrent file is imported into that same window.

## CI

`.github/workflows/quality.yml` checks out the project, installs locked frontend dependencies, runs the strict frontend build and Go tests, then builds the Wails application on Ubuntu and Windows. Enable GitHub Actions in the repository after pushing this branch.

Every push to `main` also runs `.github/workflows/installers.yml`. It packages the Debian and Windows NSIS installers and uploads them to that workflow run as downloadable artifacts. The build version is automatically set to `0.1.<GitHub run number>` for both installers; the source-tree version files remain at the development baseline (`0.1.0`).

## Privacy and security notes

BitTorrent peers and trackers can observe your public IP address. BitTorrent protocol encryption/obfuscation is not anonymity; Torrade does not claim it is. Torrade never executes downloaded files and does not delete payload data without a future explicit confirmation flow. Automatic router port forwarding is disabled by default.

Malformed `.torrent` inputs are size-limited and parsed in Go before they reach the engine. The storage backend uses the torrent library’s safe native-path handling for downloaded files.

## Current limitations

- Desktop notifications, tray behavior, proxy configuration, and tracker-health reporting are not implemented yet.
- File selection becomes available after metadata arrives. Tracker URLs are shown as configured; the engine does not expose a stable public per-tracker health API.
- Downloaded files cannot be removed from the UI yet. This prevents accidental deletion from a destination folder shared by multiple torrents.
- Torrade has no anonymity layer or proxy configuration yet.
- The NSIS installer definition is ready, but this environment does not have NSIS installed, so its final installer executable was not locally produced here. The Windows application executable was built and inspected successfully.
