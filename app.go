package main

import (
	"context"
	"log/slog"

	"github.com/wailsapp/wails/v3/pkg/application"

	"hamster-wheel/internal/db"
	"hamster-wheel/internal/scheduler"
)

// AppService handles application lifecycle. It no longer exposes business logic methods
// to the frontend - those are now split into focused services (JobService, FilterService, etc.).
//
// In Wails v3, services are plain Go structs registered with the app.
// They optionally implement ServiceStartup/ServiceShutdown for lifecycle.
type AppService struct {
	db        *db.DB
	scheduler *scheduler.Scheduler
}

// NewAppService creates a new AppService. Dependencies are injected by main().
func NewAppService(database *db.DB, sched *scheduler.Scheduler) *AppService {
	return &AppService{
		db:        database,
		scheduler: sched,
	}
}

// ServiceStartup is called by Wails v3 when the application starts.
// It starts the background polling scheduler.
func (a *AppService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	a.scheduler.Start()
	slog.Info("app service started")
	return nil
}

// ServiceShutdown is called by Wails v3 when the application is shutting down.
// It stops the scheduler and closes the database.
func (a *AppService) ServiceShutdown() error {
	a.scheduler.Stop()
	if err := a.db.Close(); err != nil {
		slog.Error("failed to close database during shutdown", "error", err)
		return err
	}
	slog.Info("app service shut down")
	return nil
}
