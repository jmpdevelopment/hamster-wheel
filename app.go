package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"hamster-wheel/internal/db"
	"hamster-wheel/internal/matcher"
	"hamster-wheel/internal/scheduler"
)

const startupCleanupTimeout = 15 * time.Second

// AppService handles application lifecycle. It no longer exposes business logic methods
// to the frontend - those are now split into focused services (JobService, FilterService, etc.).
//
// In Wails v3, services are plain Go structs registered with the app.
// They optionally implement ServiceStartup/ServiceShutdown for lifecycle.
type AppService struct {
	db        *db.DB
	scheduler *scheduler.Scheduler
	matcher   *matcher.Worker
}

// NewAppService creates a new AppService. Dependencies are injected by main().
func NewAppService(database *db.DB, sched *scheduler.Scheduler, matchWorker *matcher.Worker) *AppService {
	return &AppService{
		db:        database,
		scheduler: sched,
		matcher:   matchWorker,
	}
}

// ServiceStartup is called by Wails v3 when the application starts.
// It starts the scheduler loop and matcher worker.
// Scheduler polling behavior depends on pause/interval settings.
func (a *AppService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	cleanupCtx, cancel := context.WithTimeout(ctx, startupCleanupTimeout)
	defer cancel()
	if _, err := a.scheduler.CleanupExpiredJobs(cleanupCtx); err != nil {
		slog.Warn("job retention cleanup failed", "error", err)
	}

	a.scheduler.Start()
	if a.matcher != nil {
		a.matcher.Start()
	}
	slog.Info("app service started")
	return nil
}

// ServiceShutdown is called by Wails v3 when the application is shutting down.
// It stops the scheduler and closes the database.
func (a *AppService) ServiceShutdown() error {
	if a.matcher != nil {
		a.matcher.Stop()
	}
	a.scheduler.Stop()
	if err := a.db.Close(); err != nil {
		slog.Error("failed to close database during shutdown", "error", err)
		return err
	}
	slog.Info("app service shut down")
	return nil
}
