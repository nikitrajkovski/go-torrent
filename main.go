package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// assets contains the production frontend bundle. Wails serves the Vite dev
// server during development and this embedded bundle in packaged builds.
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := application.New(application.Options{
		Name:        "TorrentGo",
		Description: "A lightweight BitTorrent desktop client",
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
