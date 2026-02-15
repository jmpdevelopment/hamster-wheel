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

func TestPollNowReschedulesNextAutoPollWindow(t *testing.T) {
	database := testPollingServiceDB(t)
	registry := adapter.NewRegistry()
	sched := scheduler.New(database, registry, 2*time.Second)
	store := diagnostics.NewStore(t.TempDir(), 20, 24*time.Hour)
	service := NewPollingService(sched, store)

	sched.Start()
	defer sched.Stop()

	var initialNext time.Time
	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		initialNext = sched.NextPollAt()
		if !initialNext.IsZero() {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if initialNext.IsZero() {
		t.Fatal("expected initial next poll schedule after scheduler start")
	}

	time.Sleep(20 * time.Millisecond)
	_ = service.PollNow()

	updatedNext := sched.NextPollAt()
	if !updatedNext.After(initialNext) {
		t.Fatalf("expected PollNow to move next poll forward, initial=%v updated=%v", initialNext, updatedNext)
	}
}

func TestPollNowReportsDiagnosticsStoreUnavailable(t *testing.T) {
	database := testPollingServiceDB(t)
	registry := adapter.NewRegistry()

	okAdapter := &pollingTestAdapter{name: "ok_source"}
	if err := registry.Register(okAdapter); err != nil {
		t.Fatalf("registering adapter: %v", err)
	}
	if _, err := database.CreateFilter(context.Background(), "OK", "go", "london", "ok_source"); err != nil {
		t.Fatalf("creating filter: %v", err)
	}

	sched := scheduler.New(database, registry, time.Hour)
	service := NewPollingService(sched, nil)

	result := service.PollNow()

	if result.TotalFilters != 1 {
		t.Fatalf("expected 1 filter in poll result, got %d", result.TotalFilters)
	}
	if result.DiagnosticsPath != "" {
		t.Fatalf("expected empty diagnostics path when store unavailable, got %q", result.DiagnosticsPath)
	}
	if result.DiagnosticsError != "diagnostics store unavailable" {
		t.Fatalf("expected diagnostics store unavailable error, got %q", result.DiagnosticsError)
	}
}

func TestPollingIntervalSettings(t *testing.T) {
	database := testPollingServiceDB(t)
	registry := adapter.NewRegistry()
	sched := scheduler.New(database, registry, 45*time.Minute)
	store := diagnostics.NewStore(t.TempDir(), 20, 24*time.Hour)
	service := NewPollingService(sched, store)

	if got := service.GetPollingIntervalMinutes(); got != 45 {
		t.Fatalf("expected polling interval 45 minutes, got %d", got)
	}

	if err := service.SetPollingIntervalMinutes(120); err != nil {
		t.Fatalf("setting polling interval minutes: %v", err)
	}
	if got := service.GetPollingIntervalMinutes(); got != 120 {
		t.Fatalf("expected polling interval 120 minutes after update, got %d", got)
	}

	if err := service.SetPollingIntervalMinutes(minPollingIntervalMinutes - 1); err == nil {
		t.Fatalf("expected validation error below %d minutes", minPollingIntervalMinutes)
	}
	if err := service.SetPollingIntervalMinutes(maxPollingIntervalMinutes + 1); err == nil {
		t.Fatalf("expected validation error above %d minutes", maxPollingIntervalMinutes)
	}
}

func TestSavePollReportWritesFile(t *testing.T) {
	database := testPollingServiceDB(t)
	registry := adapter.NewRegistry()
	sched := scheduler.New(database, registry, time.Hour)
	store := diagnostics.NewStore(t.TempDir(), 20, 24*time.Hour)
	service := NewPollingService(sched, store)

	path := filepath.Join(t.TempDir(), "reports", "poll-report.txt")
	content := "Hamster Wheel Poll Report\nRun ID: run-1\n"

	if err := service.SavePollReport(path, content); err != nil {
		t.Fatalf("saving poll report: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading saved report: %v", err)
	}
	if string(data) != content {
		t.Fatalf("expected report content %q, got %q", content, string(data))
	}
}

func TestSavePollReportValidatesInputs(t *testing.T) {
	database := testPollingServiceDB(t)
	registry := adapter.NewRegistry()
	sched := scheduler.New(database, registry, time.Hour)
	store := diagnostics.NewStore(t.TempDir(), 20, 24*time.Hour)
	service := NewPollingService(sched, store)

	if err := service.SavePollReport("   ", "ok"); err == nil {
		t.Fatal("expected error for empty save path")
	}
	if err := service.SavePollReport(filepath.Join(t.TempDir(), "report.txt"), "   "); err == nil {
		t.Fatal("expected error for empty report content")
	}

	oversized := strings.Repeat("x", maxPollReportBytes+1)
	if err := service.SavePollReport(filepath.Join(t.TempDir(), "report.txt"), oversized); err == nil {
		t.Fatal("expected error for oversized report")
	}
}

func TestSavePollReportReturnsDirectoryCreationError(t *testing.T) {
	database := testPollingServiceDB(t)
	registry := adapter.NewRegistry()
	sched := scheduler.New(database, registry, time.Hour)
	store := diagnostics.NewStore(t.TempDir(), 20, 24*time.Hour)
	service := NewPollingService(sched, store)

	baseFile := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(baseFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("creating blocking file: %v", err)
	}

	err := service.SavePollReport(filepath.Join(baseFile, "report.txt"), "report")
	if err == nil {
		t.Fatal("expected directory creation error")
	}
	if !strings.Contains(err.Error(), "creating report directory") {
		t.Fatalf("expected directory creation context, got %v", err)
	}
}

func TestSavePollReportReturnsWriteError(t *testing.T) {
	database := testPollingServiceDB(t)
	registry := adapter.NewRegistry()
	sched := scheduler.New(database, registry, time.Hour)
	store := diagnostics.NewStore(t.TempDir(), 20, 24*time.Hour)
	service := NewPollingService(sched, store)

	targetDir := filepath.Join(t.TempDir(), "reports")
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		t.Fatalf("creating target directory: %v", err)
	}

	err := service.SavePollReport(targetDir, "report")
	if err == nil {
		t.Fatal("expected write error when target path is a directory")
	}
	if !strings.Contains(err.Error(), "writing poll report") {
		t.Fatalf("expected write context, got %v", err)
	}
}
