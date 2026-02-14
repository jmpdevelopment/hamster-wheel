package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"hamster-wheel/internal/adapter"
	"hamster-wheel/internal/db"
	"hamster-wheel/internal/diagnostics"
	"hamster-wheel/internal/scheduler"
)

type pollingTestAdapter struct {
	name      string
	fetchErr  error
	jobSource []adapter.JobSummary
}

func (a *pollingTestAdapter) Name() string        { return a.name }
func (a *pollingTestAdapter) DisplayName() string { return a.name }
func (a *pollingTestAdapter) Validate(context.Context) error {
	return nil
}
func (a *pollingTestAdapter) FetchNewJobs(context.Context, adapter.SearchParams) ([]adapter.JobSummary, error) {
	if a.fetchErr != nil {
		return nil, a.fetchErr
	}
	return a.jobSource, nil
}
func (a *pollingTestAdapter) FetchJobDetails(_ context.Context, job adapter.JobSummary) (*adapter.JobDetails, error) {
	return &adapter.JobDetails{
		JobSummary:      job,
		FullDescription: "full description",
	}, nil
}

func testPollingServiceDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.OpenAt(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("opening db: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func TestPollNowReturnsStructuredResultsAndSavesDiagnostics(t *testing.T) {
	database := testPollingServiceDB(t)
	registry := adapter.NewRegistry()

	okAdapter := &pollingTestAdapter{
		name: "ok_source",
		jobSource: []adapter.JobSummary{
			{
				SourceID: "ok-1",
				Title:    "Go Developer",
				Company:  "Acme",
				Location: "London",
				URL:      "https://example.com/ok-1",
				PostedAt: time.Now(),
			},
		},
	}
	failAdapter := &pollingTestAdapter{
		name:     "fail_source",
		fetchErr: errors.New("upstream timeout"),
	}
	if err := registry.Register(okAdapter); err != nil {
		t.Fatalf("registering ok adapter: %v", err)
	}
	if err := registry.Register(failAdapter); err != nil {
		t.Fatalf("registering fail adapter: %v", err)
	}

	if _, err := database.CreateFilter(context.Background(), "OK", "go", "london", "ok_source"); err != nil {
		t.Fatalf("creating OK filter: %v", err)
	}
	if _, err := database.CreateFilter(context.Background(), "Failing", "go", "london", "fail_source"); err != nil {
		t.Fatalf("creating failing filter: %v", err)
	}

	sched := scheduler.New(database, registry, time.Hour)
	store := diagnostics.NewStore(t.TempDir(), 20, 24*time.Hour)
	service := NewPollingService(sched, store)

	result := service.PollNow()

	if result.RunID == "" {
		t.Fatal("expected non-empty RunID")
	}
	if result.CycleError != "" {
		t.Fatalf("expected no cycle error, got %q", result.CycleError)
	}
	if result.TotalFilters != 2 {
		t.Fatalf("expected 2 filters, got %d", result.TotalFilters)
	}
	if result.FailedFilters != 1 {
		t.Fatalf("expected 1 failed filter, got %d", result.FailedFilters)
	}
	if result.NewJobs != 1 {
		t.Fatalf("expected 1 new job, got %d", result.NewJobs)
	}
	if result.DiagnosticsPath == "" {
		t.Fatal("expected diagnostics path for non-empty poll run")
	}
	if result.DiagnosticsError != "" {
		t.Fatalf("expected no diagnostics error, got %q", result.DiagnosticsError)
	}

	data, err := os.ReadFile(result.DiagnosticsPath)
	if err != nil {
		t.Fatalf("reading diagnostics file: %v", err)
	}
	var persisted diagnostics.PollRun
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("unmarshalling diagnostics file: %v", err)
	}
	if persisted.RunID != result.RunID {
		t.Fatalf("expected persisted run ID %q, got %q", result.RunID, persisted.RunID)
	}
	if persisted.TotalFilters != 2 {
		t.Fatalf("expected persisted total filters=2, got %d", persisted.TotalFilters)
	}
	if persisted.FailedFilters != 1 {
		t.Fatalf("expected persisted failed filters=1, got %d", persisted.FailedFilters)
	}
}

func TestPollNowCapturesCompleteFailureAndPersistsDiagnostics(t *testing.T) {
	database := testPollingServiceDB(t)
	registry := adapter.NewRegistry()
	sched := scheduler.New(database, registry, time.Hour)
	store := diagnostics.NewStore(t.TempDir(), 20, 24*time.Hour)
	service := NewPollingService(sched, store)

	// Closing DB forces ListEnabledFilters to fail in PollOnce.
	database.Close()

	result := service.PollNow()

	if result.CycleError == "" {
		t.Fatal("expected cycle error for complete poll failure")
	}
	if !strings.Contains(result.CycleError, "listing enabled filters") {
		t.Fatalf("expected cycle error to mention listing filters, got %q", result.CycleError)
	}
	if result.TotalFilters != 0 {
		t.Fatalf("expected 0 filters for complete failure, got %d", result.TotalFilters)
	}
	if result.DiagnosticsPath == "" {
		t.Fatal("expected diagnostics path for complete failure")
	}

	data, err := os.ReadFile(result.DiagnosticsPath)
	if err != nil {
		t.Fatalf("reading diagnostics file: %v", err)
	}
	var persisted diagnostics.PollRun
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("unmarshalling diagnostics file: %v", err)
	}
	if persisted.CycleError == "" {
		t.Fatal("expected persisted cycle error for complete failure")
	}
}

func TestPollNowNoEnabledFiltersSkipsDiagnosticsPersistence(t *testing.T) {
	database := testPollingServiceDB(t)
	registry := adapter.NewRegistry()
	sched := scheduler.New(database, registry, time.Hour)
	store := diagnostics.NewStore(t.TempDir(), 20, 24*time.Hour)
	service := NewPollingService(sched, store)

	result := service.PollNow()

	if result.CycleError != "" {
		t.Fatalf("expected no cycle error, got %q", result.CycleError)
	}
	if result.TotalFilters != 0 {
		t.Fatalf("expected no filters, got %d", result.TotalFilters)
	}
	if result.DiagnosticsPath != "" {
		t.Fatalf("expected no diagnostics path for no-op poll, got %q", result.DiagnosticsPath)
	}
}
