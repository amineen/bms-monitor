package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:     "Pylontech BMS Monitor",
		Width:     1320,
		Height:    900,
		MinWidth:  1024,
		MinHeight: 700,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 9, G: 12, B: 20, A: 1},
		OnStartup:        app.startup,
		// A non-nil Mac block is required for the green traffic-light (zoom)
		// button to be enabled — Wails leaves it disabled when Mac is nil.
		Mac: &mac.Options{
			DisableZoom: false,
			Preferences: &mac.Preferences{
				FullscreenEnabled: mac.Enabled,
			},
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
