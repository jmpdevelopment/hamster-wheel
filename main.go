package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"hamster-wheel/internal/adapter"
	"hamster-wheel/internal/adapter/reed"
	"hamster-wheel/internal/db"
	"hamster-wheel/internal/scheduler"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed assets/iconTemplate.png
var trayIcon []byte

func main() {
	// Set up structured logging.
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))

	// Open the database (creates it on first run).
	database, err := db.Open()
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	// Load Reed API key: DB first, then env var fallback.
	reedAPIKey, _ := database.GetSetting(context.Background(), "reed_api_key")
	if reedAPIKey == "" {
		reedAPIKey = os.Getenv("REED_API_KEY")
	}
	if reedAPIKey == "" {
		slog.Warn("Reed API key not set — configure it in the app or set REED_API_KEY")
	}

	// Set up adapter registry and register available job sources.
	reedAdapter := reed.New(reedAPIKey)
	adapters := adapter.NewRegistry()
	if err := adapters.Register(reedAdapter); err != nil {
		log.Fatalf("failed to register Reed adapter: %v", err)
	}

	// Create the scheduler (default 30 minute interval).
	// It starts polling when the app service receives ServiceStartup.
	pollInterval := 30 * time.Minute
	sched := scheduler.New(database, adapters, pollInterval)

	// Create the app service with all dependencies injected.
	appService := NewAppService(database, sched, reedAdapter)

	// Create the Wails v3 application.
	// Strip the "frontend/dist" prefix from the embedded filesystem.
	assetsFS, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		log.Fatalf("failed to create assets sub-filesystem: %v", err)
	}

	app := application.New(application.Options{
		Name: "Hamster Wheel",
		Services: []application.Service{
			application.NewService(appService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assetsFS),
		},
		Mac: application.MacOptions{
			// Keep the app running when the window is closed (for system tray).
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	// Create the main application window.
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Hamster Wheel",
		Width:            1024,
		Height:           768,
		BackgroundColour: application.NewRGBA(27, 38, 54, 255),
	})

	// Hide to system tray when window close button is clicked (instead of terminating).
	window.OnWindowEvent(events.Common.WindowClosing, func(event *application.WindowEvent) {
		window.Hide()
		// Prevent the window from actually closing
		event.Cancel()
	})

	// Set up system tray with menu.
	setupSystemTray(app, window, sched)

	// Run the application (blocks until quit).
	if err := app.Run(); err != nil {
		log.Fatalf("wails error: %v", err)
	}
}

// setupSystemTray creates the system tray icon with menu for background operation.
func setupSystemTray(app *application.App, window *application.WebviewWindow, sched *scheduler.Scheduler) {
	// Create system tray.
	systray := app.SystemTray.New()
	systray.SetTemplateIcon(trayIcon) // Template icon adapts to light/dark mode on macOS
	systray.SetLabel("HW")             // Short label for menu bar

	// Create tray menu.
	menu := app.NewMenu()

	// "Open Hamster Wheel" - shows and focuses the window
	menu.Add("Open Hamster Wheel").OnClick(func(ctx *application.Context) {
		window.Show()
		window.Focus()
	})

	menu.AddSeparator()

	// "Pause/Resume Monitoring" - toggles scheduler pause state
	pauseItem := menu.Add("Pause Monitoring")
	pauseItem.OnClick(func(ctx *application.Context) {
		paused := sched.IsPaused()
		sched.SetPaused(!paused)

		// Update menu item label
		if !paused {
			pauseItem.SetLabel("Resume Monitoring")
		} else {
			pauseItem.SetLabel("Pause Monitoring")
		}
		menu.Update()
	})

	menu.AddSeparator()

	// "Quit" - exits the application
	menu.Add("Quit").OnClick(func(ctx *application.Context) {
		app.Quit()
	})

	systray.SetMenu(menu)
}
