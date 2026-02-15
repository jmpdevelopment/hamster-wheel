package main

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"hamster-wheel/internal/adapter"
	"hamster-wheel/internal/adapter/adzuna"
	"hamster-wheel/internal/adapter/reed"
	"hamster-wheel/internal/db"
	"hamster-wheel/internal/diagnostics"
	"hamster-wheel/internal/keychain"
	"hamster-wheel/internal/llm"
	"hamster-wheel/internal/llm/heuristic"
	"hamster-wheel/internal/llm/openai"
	"hamster-wheel/internal/localruntime"
	"hamster-wheel/internal/matcher"
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

	// Create keychain store for secure API key storage.
	keychainStore := keychain.NewOSStore()

	// Load Reed API key: keychain first, then env var fallback.
	reedAPIKey := loadReedAPIKey(keychainStore, os.Getenv)
	if reedAPIKey == "" {
		slog.Warn("Reed API key not set — configure it in the app or set REED_API_KEY")
	}
	adzunaAppID, adzunaAppKey := loadAdzunaCredentials(keychainStore, os.Getenv)
	if adzunaAppID == "" || adzunaAppKey == "" {
		slog.Warn("Adzuna credentials not set — set ADZUNA_APP_ID and ADZUNA_APP_KEY to enable Adzuna polling")
	}

	// Set up adapter registry and register available job sources.
	reedAdapter := reed.New(reedAPIKey)
	adzunaAdapter := adzuna.New(adzunaAppID, adzunaAppKey)
	adapters := adapter.NewRegistry()
	if err := adapters.Register(reedAdapter); err != nil {
		log.Fatalf("failed to register Reed adapter: %v", err)
	}
	if err := adapters.Register(adzunaAdapter); err != nil {
		log.Fatalf("failed to register Adzuna adapter: %v", err)
	}
	openAIEnvKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	openAIEnvModel := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	openAIEnvBaseURL := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	ollamaBaseURL := strings.TrimSpace(os.Getenv("OLLAMA_BASE_URL"))

	// Create the scheduler (default 30 minute interval).
	// It starts polling when the app service receives ServiceStartup.
	pollInterval := 30 * time.Minute
	sched := scheduler.New(database, adapters, pollInterval)
	providers := llm.NewRegistry()
	if err := providers.Register(heuristic.New()); err != nil {
		log.Fatalf("failed to register heuristic match provider: %v", err)
	}
	if err := providers.Register(openai.New(openai.Config{
		APIKey:  openAIEnvKey,
		Model:   openAIEnvModel,
		BaseURL: openAIEnvBaseURL,
	})); err != nil {
		log.Fatalf("failed to register openai match provider: %v", err)
	}
	localRuntimeManager := localruntime.NewOllamaManager(localruntime.Config{
		Binary:           strings.TrimSpace(os.Getenv("OLLAMA_BINARY")),
		Endpoint:         ollamaBaseURL,
		ManagedStateFile: filepath.Join(os.TempDir(), "hamster-wheel", "localruntime-ollama-managed.json"),
	})
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()
		if _, err := localRuntimeManager.Stop(shutdownCtx); err != nil &&
			!errors.Is(err, context.Canceled) &&
			!errors.Is(err, context.DeadlineExceeded) &&
			!errors.Is(err, localruntime.ErrRuntimeNotInstalled) {
			slog.Warn("failed to stop managed local runtime during app shutdown", "error", err)
		}
	}()
	matchWorker := matcher.New(database, providers, matcher.WorkerConfig{
		ProviderName: heuristic.ProviderName, // fallback when no setting is persisted yet
		ProviderResolver: newMatcherProviderResolver(
			database,
			keychainStore,
			providers,
			openAIEnvKey,
			openAIEnvModel,
			openAIEnvBaseURL,
			ollamaBaseURL,
			localRuntimeManager,
		),
		PollInterval: 3 * time.Second,
		BatchSize:    3,
		ProviderTimeouts: map[string]time.Duration{
			localProviderOllama: localProviderMatchTimeout,
		},
	})
	matchWorker.SetLogger(slog.With("component", "matcher_worker"))
	diagStore := diagnostics.NewStore(
		filepath.Join(os.TempDir(), "hamster-wheel", "poll-diagnostics"),
		50,
		24*time.Hour,
	)

	// Create all services with their dependencies injected.
	appService := NewAppService(database, sched, matchWorker)
	jobService := NewJobService(database, adapters)
	filterService := NewFilterService(database)
	pollingService := NewPollingService(sched, diagStore)
	settingsService := NewSettingsService(database, keychainStore, reedAdapter, localRuntimeManager)
	settingsService.setAdzunaAdapter(adzunaAdapter)

	// Create the Wails v3 application.
	// Strip the "frontend/dist" prefix from the embedded filesystem.
	assetsFS, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		log.Fatalf("failed to create assets sub-filesystem: %v", err)
	}

	app := application.New(application.Options{
		Name: "Hamster Wheel",
		Services: []application.Service{
			application.NewService(appService),      // lifecycle (startup/shutdown)
			application.NewService(jobService),      // job CRUD
			application.NewService(filterService),   // filter CRUD
			application.NewService(pollingService),  // scheduler control
			application.NewService(settingsService), // settings + API keys
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assetsFS),
		},
		Mac: application.MacOptions{
			// Keep the app running when the window is closed (for system tray).
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
	})

	// Broadcast scheduler state transitions so the UI can refresh immediately.
	sched.SetStatusChangedHook(func() {
		next := sched.NextPollAt()
		nextPollAt := ""
		if !next.IsZero() {
			nextPollAt = next.Format(time.RFC3339)
		}
		payload := map[string]any{
			"paused":     sched.IsPaused(),
			"isPolling":  sched.IsPolling(),
			"nextPollAt": nextPollAt,
		}
		if summary, ok := sched.LastPollSummary(); ok {
			payload["lastRun"] = pollRunFromSchedulerSummary(summary)
		}
		app.Event.Emit("polling:status-changed", payload)
	})
	matchWorker.SetStatusChangedHook(func(jobID, status string) {
		app.Event.Emit("matching:status-changed", map[string]any{
			"jobID":  jobID,
			"status": status,
		})
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

func loadReedAPIKey(store keychain.Store, getenv func(string) string) string {
	reedAPIKey, err := store.Get(settingReedAPIKey)
	if err != nil {
		slog.Error("failed to load Reed API key from keychain", "error", err)
	}

	reedAPIKey = strings.TrimSpace(reedAPIKey)
	if reedAPIKey == "" {
		reedAPIKey = strings.TrimSpace(getenv("REED_API_KEY"))
	}
	return reedAPIKey
}

func loadAdzunaCredentials(store keychain.Store, getenv func(string) string) (string, string) {
	adzunaAppID, err := store.Get(settingAdzunaAppID)
	if err != nil {
		slog.Error("failed to load Adzuna app ID from keychain", "error", err)
	}
	adzunaAppKey, err := store.Get(settingAdzunaAppKey)
	if err != nil {
		slog.Error("failed to load Adzuna app key from keychain", "error", err)
	}

	adzunaAppID = strings.TrimSpace(adzunaAppID)
	adzunaAppKey = strings.TrimSpace(adzunaAppKey)
	if adzunaAppID == "" {
		adzunaAppID = strings.TrimSpace(getenv("ADZUNA_APP_ID"))
	}
	if adzunaAppKey == "" {
		adzunaAppKey = strings.TrimSpace(getenv("ADZUNA_APP_KEY"))
	}
	return adzunaAppID, adzunaAppKey
}

// setupSystemTray creates the system tray icon with menu for background operation.
func setupSystemTray(app *application.App, window *application.WebviewWindow, sched *scheduler.Scheduler) {
	// Create system tray.
	systray := app.SystemTray.New()
	systray.SetTemplateIcon(trayIcon) // Template icon adapts to light/dark mode on macOS
	systray.SetLabel("HW")            // Short label for menu bar

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
