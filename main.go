package main

import (
	"embed"
	"log"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/nikitrajkovski/torrentgo/internal/config"
	"github.com/nikitrajkovski/torrentgo/internal/services"
	"github.com/nikitrajkovski/torrentgo/internal/storage"
	"github.com/nikitrajkovski/torrentgo/internal/torrent"
)

// assets contains the production frontend bundle. Wails serves the Vite dev
// server during development and this embedded bundle in packaged builds.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	settingsStore, err := config.NewStore()
	if err != nil {
		log.Fatal("create settings store: ", err)
	}
	torrentStore, err := storage.NewStore()
	if err != nil {
		log.Fatal("create torrent store: ", err)
	}
	engine, err := torrent.New(torrentStore, logger)
	if err != nil {
		log.Fatal("start torrent engine: ", err)
	}
	defer func() {
		if err := engine.Close(); err != nil {
			logger.Error("close torrent engine", "error", err)
		}
	}()
	service := services.NewTorrentService(engine, settingsStore)

	app := application.New(application.Options{
		Name:        "TorrentGo",
		Description: "A lightweight BitTorrent desktop client",
		Services: []application.Service{
			application.NewService(service),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "main",
		Title:     "TorrentGo",
		Width:     1200,
		Height:    760,
		MinWidth:  920,
		MinHeight: 600,
	})
	window.Center()
	window.Show()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
