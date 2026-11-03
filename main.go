package main

import (
	"embed"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/nikitrajkovski/torrentgo/internal/config"
	"github.com/nikitrajkovski/torrentgo/internal/models"
	"github.com/nikitrajkovski/torrentgo/internal/services"
	"github.com/nikitrajkovski/torrentgo/internal/storage"
	"github.com/nikitrajkovski/torrentgo/internal/torrent"
)

// assets contains the production frontend bundle. Wails serves the Vite dev
// server during development and this embedded bundle in packaged builds.
//
//go:embed all:frontend/dist
var assets embed.FS

// importExternalTarget handles operating-system file and URL associations.
// A result with Added == 0 and no Error means the argument was not a torrent
// target and should be ignored.
func importExternalTarget(service *services.TorrentService, target string) models.TorrentImportResult {
	if strings.HasPrefix(strings.ToLower(target), "magnet:?") {
		if _, err := service.AddMagnet(target); err != nil {
			return models.TorrentImportResult{Error: err.Error()}
		}
		return models.TorrentImportResult{Added: 1}
	}
	if strings.EqualFold(filepath.Ext(target), ".torrent") {
		if _, err := service.AddTorrentPath(target); err != nil {
			return models.TorrentImportResult{Error: err.Error()}
		}
		return models.TorrentImportResult{Added: 1}
	}
	return models.TorrentImportResult{}
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	settingsStore, err := config.NewStore()
	if err != nil {
		log.Fatal("create settings store: ", err)
	}
	settings, err := settingsStore.Load()
	if err != nil {
		log.Fatal("load settings: ", err)
	}
	torrentStore, err := storage.NewStore()
	if err != nil {
		log.Fatal("create torrent store: ", err)
	}
	engine, err := torrent.New(torrentStore, logger, settings)
	if err != nil {
		log.Fatal("start torrent engine: ", err)
	}
	defer func() {
		if err := engine.Close(); err != nil {
			logger.Error("close torrent engine", "error", err)
		}
	}()
	service := services.NewTorrentService(engine, settingsStore)

	var app *application.App
	var window *application.WebviewWindow
	importTarget := func(target string) {
		result := importExternalTarget(service, target)
		if result.Added == 0 && result.Error == "" {
			return
		}
		if result.Error != "" {
			logger.Error("import externally opened torrent", "error", result.Error)
		}
		app.Event.Emit(services.TorrentImportEvent, result)
		if window != nil {
			window.Show()
			window.Focus()
		}
	}

	app = application.New(application.Options{
		Name:        "Torrade",
		Description: "A lightweight BitTorrent desktop client",
		Linux: application.LinuxOptions{
			// Keep the GTK/Wayland application ID aligned with the desktop
			// entry so Linux shells can use the correct display name and icon.
			ApplicationID: "io.github.nikitrajkovski.Torrade",
		},
		Services: []application.Service{
			application.NewService(service),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		FileAssociations: []string{".torrent"},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "io.github.nikitrajkovski.Torrade",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				for _, target := range data.Args[1:] {
					importTarget(target)
				}
			},
		},
	})
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		// Wails recognises URL arguments containing "://", but standard magnet
		// links are "magnet:?…". Import those directly after services start.
		for _, target := range os.Args[1:] {
			if strings.HasPrefix(strings.ToLower(target), "magnet:?") {
				importTarget(target)
			}
		}
	})
	app.Event.OnApplicationEvent(events.Common.ApplicationLaunchedWithUrl, func(event *application.ApplicationEvent) {
		importTarget(event.Context().URL())
	})
	app.Event.OnApplicationEvent(events.Common.ApplicationOpenedWithFile, func(event *application.ApplicationEvent) {
		importTarget(event.Context().Filename())
	})

	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:           "main",
		Title:          "Torrade",
		Frameless:      true,
		Width:          1200,
		Height:         760,
		MinWidth:       920,
		MinHeight:      600,
		EnableFileDrop: true,
	})
	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		if event.Context().DropTargetDetails().ElementID != "torrent-drop-zone" {
			return
		}
		result := models.TorrentImportResult{}
		for _, path := range event.Context().DroppedFiles() {
			if _, err := service.AddTorrentPath(path); err != nil {
				result.Error = "One or more dropped files could not be added. Only valid .torrent files are accepted."
				continue
			}
			result.Added++
		}
		if result.Added == 0 && result.Error == "" {
			result.Error = "No .torrent files were dropped."
		}
		app.Event.Emit(services.TorrentImportEvent, result)
	})
	window.Center()
	window.Show()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
