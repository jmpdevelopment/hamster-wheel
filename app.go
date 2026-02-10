package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"hamster-wheel/internal/adapter/reed"
	"hamster-wheel/internal/db"
	"hamster-wheel/internal/scheduler"
)

const settingReedAPIKey = "reed_api_key"

// AppService is the main Wails service. All exported methods are
// automatically exposed to the React frontend via Wails bindings.
//
// In Wails v3, services are plain Go structs registered with the app.
// They optionally implement ServiceStartup/ServiceShutdown for lifecycle.
type AppService struct {
	db          *db.DB
	scheduler   *scheduler.Scheduler
	reedAdapter *reed.Adapter // Direct reference for API key updates
}

// NewAppService creates a new AppService. Dependencies are injected by main().
func NewAppService(database *db.DB, sched *scheduler.Scheduler, reedAdapter *reed.Adapter) *AppService {
	return &AppService{
		db:          database,
		scheduler:   sched,
		reedAdapter: reedAdapter,
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

// --- Job methods (exposed to frontend) ---

// GetJobs returns jobs ordered by discovery date (newest first).
// Pass limit=0 for all jobs.
func (a *AppService) GetJobs(limit int) ([]db.Job, error) {
	return a.db.ListJobs(context.Background(), limit)
}

// GetJob returns a single job by ID, or nil if not found.
func (a *AppService) GetJob(id string) (*db.Job, error) {
	return a.db.GetJob(context.Background(), id)
}

// GetJobsByFilter returns jobs discovered through a specific filter.
func (a *AppService) GetJobsByFilter(filterID string) ([]db.Job, error) {
	return a.db.ListJobsByFilter(context.Background(), filterID)
}

// GetJobCount returns the total number of jobs in the database.
func (a *AppService) GetJobCount() (int, error) {
	return a.db.CountJobs(context.Background())
}

// DeleteJob removes a job by ID.
func (a *AppService) DeleteJob(id string) error {
	return a.db.DeleteJob(context.Background(), id)
}

// --- Filter methods (exposed to frontend) ---

// GetFilters returns all search filters.
func (a *AppService) GetFilters() ([]db.SearchFilter, error) {
	return a.db.ListFilters(context.Background())
}

// GetFilter returns a single filter by ID, or nil if not found.
func (a *AppService) GetFilter(id string) (*db.SearchFilter, error) {
	return a.db.GetFilter(context.Background(), id)
}

// CreateFilter creates a new search filter and returns its ID.
func (a *AppService) CreateFilter(name, keywords, location, source string) (string, error) {
	return a.db.CreateFilter(context.Background(), name, keywords, location, source)
}

// UpdateFilter updates an existing search filter.
func (a *AppService) UpdateFilter(id, name, keywords, location, source string, enabled bool) error {
	return a.db.UpdateFilter(context.Background(), id, name, keywords, location, source, enabled)
}

// DeleteFilter removes a search filter by ID.
func (a *AppService) DeleteFilter(id string) error {
	return a.db.DeleteFilter(context.Background(), id)
}

// --- Scheduler methods (exposed to frontend) ---

// pollNowTimeout caps how long a manual poll can run before being cancelled.
// This prevents the Wails binding from blocking the UI indefinitely if the
// Reed API is slow or unresponsive.
const pollNowTimeout = 5 * time.Minute

// PollNow triggers an immediate poll cycle and returns the results.
func (a *AppService) PollNow() []scheduler.PollResult {
	ctx, cancel := context.WithTimeout(context.Background(), pollNowTimeout)
	defer cancel()
	return a.scheduler.PollOnce(ctx)
}

// --- Polling control methods (exposed to frontend) ---

// PollingStatus represents the current state of auto-polling.
type PollingStatus struct {
	Paused     bool   `json:"paused"`
	NextPollAt string `json:"nextPollAt"` // RFC3339 timestamp, empty if not scheduled
}

// GetPollingStatus returns whether auto-polling is paused and when the next poll is due.
func (a *AppService) GetPollingStatus() PollingStatus {
	next := a.scheduler.NextPollAt()
	nextStr := ""
	if !next.IsZero() {
		nextStr = next.Format(time.RFC3339)
	}
	return PollingStatus{
		Paused:     a.scheduler.IsPaused(),
		NextPollAt: nextStr,
	}
}

// SetPollingPaused pauses or resumes auto-polling. Manual PollNow still works.
func (a *AppService) SetPollingPaused(paused bool) {
	a.scheduler.SetPaused(paused)
}

// --- Settings methods (exposed to frontend) ---

// GetReedAPIKey returns the stored Reed API key (empty if not set).
func (a *AppService) GetReedAPIKey() (string, error) {
	return a.db.GetSetting(context.Background(), settingReedAPIKey)
}

// SetReedAPIKey saves the Reed API key and updates the adapter immediately.
func (a *AppService) SetReedAPIKey(key string) error {
	if err := a.db.SetSetting(context.Background(), settingReedAPIKey, key); err != nil {
		return err
	}
	a.reedAdapter.SetAPIKey(key)
	slog.Info("Reed API key updated")
	return nil
}
