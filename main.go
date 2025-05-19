package main

import (
	"embed"
	"log/slog"
	"os"
	"smart-spotlight-ai/backend"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed wails.json
var wailsJson string

//go:embed all:frontend/dist
var assets embed.FS

func getVersion(isDev bool) string {
	var versionString = "0.1.0 beta"
	if wailsJson != "" {
		version := gjson.Get(wailsJson, "info.productVersion")
		if !version.Exists() {
			slog.Warn("Version not found in wails.json, using default")
		} else if version.String() == "" {
			slog.Warn("Empty version in wails.json, using default")
		} else {
			versionString = version.String() + " beta"
			slog.Info("Version", "value", versionString)
		}
	}

	if isDev {
		versionString = versionString + " (dev)"
	}

	return versionString
}

func main() {

	// Print all environment variables starting with APP

	// Check for development mode
	isDev := strings.ToLower(os.Getenv("SMARTSPOTLIGHT_DEV")) == "true"
	if isDev {
		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug, // Set log level to Debug
		})

		// Set the default logger to use the custom handler
		slog.SetDefault(slog.New(handler))
		slog.Info("Running in development mode")
	}

	version := getVersion(isDev)
	app := backend.NewApp()
	app.SetVersion(version)

	// Create application with options
	err := wails.Run(&options.App{
		Title:             "smart-spotlight",
		Width:             650,
		Height:            65,
		AlwaysOnTop:       true,
		DisableResize:     true,
		HideWindowOnClose: true,
		Assets:            assets,
		BackgroundColour:  &options.RGBA{R: 27, G: 38, B: 54, A: 0}, // Set alpha to 0 for full transparency
		OnStartup:         app.Startup,
		OnDomReady:        app.DomReady,
		Frameless:         true,
		StartHidden:       true,
		// OS specific options
		Windows: &windows.Options{
			WebviewIsTransparent: true,
		},
		Mac: &mac.Options{
			WebviewIsTransparent: true,
		},
		Linux: &linux.Options{
			WindowIsTranslucent: true,
			WebviewGpuPolicy:    linux.WebviewGpuPolicyNever,
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
