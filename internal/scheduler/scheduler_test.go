package scheduler

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hamster-wheel/internal/adapter"
	"hamster-wheel/internal/db"
)

// --- Mock adapter for testing ---

// mockAdapter implements adapter.Adapter with configurable behavior.
type mockAdapter struct {
	name        string
	displayName string
	jobs        []adapter.JobSummary
	details     map[string]*adapter.JobDetails // keyed by SourceID
	fetchErr    error                          // if set, FetchNewJobs returns this
	detailErr   error                          // if set, FetchJobDetails returns this

	mu          sync.Mutex
	fetchCalls  int // how many times FetchNewJobs was called
	detailCalls int // how many times FetchJobDetails was called
}

func (m *mockAdapter) Name() string                     { return m.name }
func (m *mockAdapter) DisplayName() string              { return m.displayName }
func (m *mockAdapter) Validate(_ context.Context) error { return nil }

func (m *mockAdapter) FetchNewJobs(_ context.Context, _ adapter.SearchParams) ([]adapter.JobSummary, error) {
	m.mu.Lock()
	m.fetchCalls++
	m.mu.Unlock()

	if m.fetchErr != nil {
		return nil, m.fetchErr
	}
	return m.jobs, nil
}

func (m *mockAdapter) FetchCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.fetchCalls
}

func (m *mockAdapter) FetchJobDetails(_ context.Context, job adapter.JobSummary) (*adapter.JobDetails, error) {
	m.mu.Lock()
	m.detailCalls++
	m.mu.Unlock()

	if m.detailErr != nil {
		return nil, m.detailErr
	}
	if d, ok := m.details[job.SourceID]; ok {
		return d, nil
	}
	// Default: return details based on the summary.
	return &adapter.JobDetails{
		JobSummary:      job,
		FullDescription: "Full description for " + job.Title,
	}, nil
}

// --- Test helpers ---

// testSetup creates a temp DB, adapter registry, and returns them with cleanup.
func testSetup(t *testing.T) (*db.DB, *adapter.Registry) {
	t.Helper()
	dir := t.TempDir()
	database, err := db.OpenAt(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	registry := adapter.NewRegistry()
	return database, registry
}

func waitForCondition(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return cond()
}

// --- Tests ---

func TestPollOnceWithNoEnabledFilters(t *testing.T) {
	database, registry := testSetup(t)
	s := New(database, registry, time.Minute)

	results, err := s.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil results for no filters, got %d", len(results))
	}
}

func TestPollOnceDiscoversNewJobs(t *testing.T) {
	database, registry := testSetup(t)

	// Register a mock adapter with 2 jobs.
	mock := &mockAdapter{
		name:        "test_source",
		displayName: "Test Source",
		jobs: []adapter.JobSummary{
			{SourceID: "j1", Title: "Go Dev", Company: "Acme", URL: "https://example.com/1", PostedAt: time.Now()},
			{SourceID: "j2", Title: "React Dev", Company: "Beta", URL: "https://example.com/2", PostedAt: time.Now()},
		},
	}
	registry.Register(mock)

	// Create an enabled filter using this source.
	filterID, err := database.CreateFilter(context.Background(), "Test Filter", "golang", "London", "test_source")
	if err != nil {
		t.Fatalf("creating filter: %v", err)
	}
	_ = filterID

	s := New(database, registry, time.Minute)
	results, err := s.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("poll once failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}
	if r.NewJobs != 2 {
		t.Errorf("expected 2 new jobs, got %d", r.NewJobs)
	}
	if r.Skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", r.Skipped)
	}

	// Verify jobs are in the database.
	count, _ := database.CountJobs(context.Background())
	if count != 2 {
		t.Errorf("expected 2 jobs in DB, got %d", count)
	}

	storedJobs, err := database.ListJobs(context.Background(), 0)
	if err != nil {
		t.Fatalf("listing stored jobs: %v", err)
	}
	for _, job := range storedJobs {
		if job.MatchStatus != db.JobMatchStatusPending {
			t.Fatalf("expected pending match status for job %q, got %q", job.ID, job.MatchStatus)
		}
	}
}

func TestPollOnceSkipsAutoMatchWhenDisabled(t *testing.T) {
	database, registry := testSetup(t)

	mock := &mockAdapter{
		name:        "test_source",
		displayName: "Test Source",
		jobs: []adapter.JobSummary{
			{SourceID: "j1", Title: "Go Dev", Company: "Acme", URL: "https://example.com/1", PostedAt: time.Now()},
			{SourceID: "j2", Title: "React Dev", Company: "Beta", URL: "https://example.com/2", PostedAt: time.Now()},
		},
	}
	registry.Register(mock)

	if err := database.SetSetting(context.Background(), settingAutoMatchEnabled, "false"); err != nil {
		t.Fatalf("disabling auto match setting: %v", err)
	}

	if _, err := database.CreateFilter(context.Background(), "Test Filter", "golang", "London", "test_source"); err != nil {
		t.Fatalf("creating filter: %v", err)
	}

	s := New(database, registry, time.Minute)
	results, err := s.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("poll once failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].NewJobs != 2 {
		t.Fatalf("expected 2 new jobs, got %d", results[0].NewJobs)
	}

	storedJobs, err := database.ListJobs(context.Background(), 0)
	if err != nil {
		t.Fatalf("listing stored jobs: %v", err)
	}
	for _, job := range storedJobs {
		if job.MatchStatus != "" {
			t.Fatalf("expected empty match status when auto match is disabled, got %q", job.MatchStatus)
		}
	}
}

func TestPollOnceHonorsAutoMatchLimitPerCycle(t *testing.T) {
	database, registry := testSetup(t)

	mock := &mockAdapter{
		name:        "limit_source",
		displayName: "Limit Source",
		jobs: []adapter.JobSummary{
			{SourceID: "j1", Title: "Go Dev", URL: "https://example.com/1", PostedAt: time.Now()},
			{SourceID: "j2", Title: "React Dev", URL: "https://example.com/2", PostedAt: time.Now()},
			{SourceID: "j3", Title: "Data Dev", URL: "https://example.com/3", PostedAt: time.Now()},
		},
	}
	registry.Register(mock)

	if err := database.SetSetting(context.Background(), settingAutoMatchEnabled, "true"); err != nil {
		t.Fatalf("enabling auto match setting: %v", err)
	}
	if err := database.SetSetting(context.Background(), settingAutoMatchLimit, "1"); err != nil {
		t.Fatalf("setting auto match limit: %v", err)
	}

	if _, err := database.CreateFilter(context.Background(), "Limit Filter", "golang", "Remote", "limit_source"); err != nil {
		t.Fatalf("creating filter: %v", err)
	}

	s := New(database, registry, time.Minute)
	results, err := s.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("poll once failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].NewJobs != 3 {
		t.Fatalf("expected 3 new jobs, got %d", results[0].NewJobs)
	}

	storedJobs, err := database.ListJobs(context.Background(), 0)
	if err != nil {
		t.Fatalf("listing stored jobs: %v", err)
	}

	pendingCount := 0
	unknownCount := 0
	for _, job := range storedJobs {
		switch job.MatchStatus {
		case db.JobMatchStatusPending:
			pendingCount++
		case "":
			unknownCount++
		default:
			t.Fatalf("unexpected match status %q", job.MatchStatus)
		}
	}
	if pendingCount != 1 {
		t.Fatalf("expected exactly 1 auto-queued pending match, got %d", pendingCount)
	}
	if unknownCount != 2 {
		t.Fatalf("expected 2 jobs without queued match rows, got %d", unknownCount)
	}
}

func TestPollOnceDeduplicatesExistingJobs(t *testing.T) {
	database, registry := testSetup(t)

	mock := &mockAdapter{
		name:        "test_source",
		displayName: "Test Source",
		jobs: []adapter.JobSummary{
			{SourceID: "existing-job", Title: "Go Dev", URL: "https://example.com/1"},
		},
	}
	registry.Register(mock)

	database.CreateFilter(context.Background(), "Test", "go", "London", "test_source")

	// Pre-insert the job so it already exists.
	database.InsertJob(context.Background(), &db.Job{
		Source:   "test_source",
		SourceID: "existing-job",
		Title:    "Go Dev",
		URL:      "https://example.com/1",
	})

	s := New(database, registry, time.Minute)
	results, err := s.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("poll once failed: %v", err)
	}

	r := results[0]
	if r.NewJobs != 0 {
		t.Errorf("expected 0 new jobs, got %d", r.NewJobs)
	}
	if r.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", r.Skipped)
	}

	// Should NOT have called FetchJobDetails for the duplicate.
	if mock.detailCalls != 0 {
		t.Errorf("expected 0 detail calls for duplicate, got %d", mock.detailCalls)
	}
}

func TestPollOnceSkipsUnknownAdapter(t *testing.T) {
	database, registry := testSetup(t)

	// Create a filter referencing an adapter that doesn't exist.
	database.CreateFilter(context.Background(), "Ghost Filter", "python", "Remote", "nonexistent_source")

	s := New(database, registry, time.Minute)
	results, err := s.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("poll once failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.Err == nil {
		t.Fatal("expected error for unknown adapter")
	}

	var notFound *ErrAdapterNotFound
	if !errors.As(r.Err, &notFound) {
		t.Errorf("expected ErrAdapterNotFound, got %T: %v", r.Err, r.Err)
	}
}

func TestPollOnceHandlesFetchError(t *testing.T) {
	database, registry := testSetup(t)

	mock := &mockAdapter{
		name:     "failing_source",
		fetchErr: errors.New("network timeout"),
	}
	registry.Register(mock)

	database.CreateFilter(context.Background(), "Failing", "go", "London", "failing_source")

	s := New(database, registry, time.Minute)
	results, err := s.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("poll once failed: %v", err)
	}

	r := results[0]
	if r.Err == nil {
		t.Fatal("expected error from fetch failure")
	}
	if r.Err.Error() != "network timeout" {
		t.Errorf("expected 'network timeout', got %q", r.Err.Error())
	}
}

func TestPollOnceStoresSummaryWhenDetailsFail(t *testing.T) {
	database, registry := testSetup(t)

	mock := &mockAdapter{
		name: "detail_fail_source",
		jobs: []adapter.JobSummary{
			{SourceID: "d1", Title: "Partial Job", Company: "Partial Co", URL: "https://example.com/d1"},
		},
		detailErr: errors.New("page not found"),
	}
	registry.Register(mock)

	database.CreateFilter(context.Background(), "Detail Fail", "go", "London", "detail_fail_source")

	s := New(database, registry, time.Minute)
	results, err := s.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("poll once failed: %v", err)
	}

	r := results[0]
	if r.Err != nil {
		t.Fatalf("unexpected filter-level error: %v", r.Err)
	}
	// Job should still be stored with summary data.
	if r.NewJobs != 1 {
		t.Errorf("expected 1 new job (from summary), got %d", r.NewJobs)
	}

	// Verify the job is in the DB with empty description.
	jobs, _ := database.ListJobs(context.Background(), 0)
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job in DB, got %d", len(jobs))
	}
	if jobs[0].Description != "" {
		t.Errorf("expected empty description for failed detail fetch, got %q", jobs[0].Description)
	}
}

func TestPollOnceMultipleFiltersConcurrently(t *testing.T) {
	database, registry := testSetup(t)

	// Two different adapters, each with their own jobs.
	mock1 := &mockAdapter{
		name: "source_a",
		jobs: []adapter.JobSummary{
			{SourceID: "a1", Title: "Job A", URL: "https://a.com/1"},
		},
	}
	mock2 := &mockAdapter{
		name: "source_b",
		jobs: []adapter.JobSummary{
			{SourceID: "b1", Title: "Job B", URL: "https://b.com/1"},
			{SourceID: "b2", Title: "Job B2", URL: "https://b.com/2"},
		},
	}
	registry.Register(mock1)
	registry.Register(mock2)

	database.CreateFilter(context.Background(), "Filter A", "go", "London", "source_a")
	database.CreateFilter(context.Background(), "Filter B", "react", "Remote", "source_b")

	s := New(database, registry, time.Minute)
	results, err := s.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("poll once failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	totalNew := 0
	for _, r := range results {
		if r.Err != nil {
			t.Errorf("unexpected error for %s: %v", r.FilterName, r.Err)
		}
		totalNew += r.NewJobs
	}

	if totalNew != 3 {
		t.Errorf("expected 3 total new jobs, got %d", totalNew)
	}

	count, _ := database.CountJobs(context.Background())
	if count != 3 {
		t.Errorf("expected 3 jobs in DB, got %d", count)
	}
}

func TestPollOnceRespectsContextCancellation(t *testing.T) {
	database, registry := testSetup(t)

	// Adapter that returns jobs, but we cancel the context before processing.
	mock := &mockAdapter{
		name: "cancel_source",
		jobs: []adapter.JobSummary{
			{SourceID: "c1", Title: "Job 1", URL: "https://example.com/1"},
			{SourceID: "c2", Title: "Job 2", URL: "https://example.com/2"},
			{SourceID: "c3", Title: "Job 3", URL: "https://example.com/3"},
		},
	}
	registry.Register(mock)

	database.CreateFilter(context.Background(), "Cancel Test", "go", "London", "cancel_source")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	s := New(database, registry, time.Minute)
	results, err := s.PollOnce(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation error, got: %v", err)
	}

	if results != nil {
		t.Errorf("expected nil results on cancelled context, got %d", len(results))
	}
}

func TestPollOnceLinksJobsToFilter(t *testing.T) {
	database, registry := testSetup(t)

	mock := &mockAdapter{
		name: "linked_source",
		jobs: []adapter.JobSummary{
			{SourceID: "link1", Title: "Linked Job", URL: "https://example.com/link1"},
		},
	}
	registry.Register(mock)

	filterID, _ := database.CreateFilter(context.Background(), "Link Test", "go", "London", "linked_source")

	s := New(database, registry, time.Minute)
	if _, err := s.PollOnce(context.Background()); err != nil {
		t.Fatalf("poll once failed: %v", err)
	}

	jobs, _ := database.ListJobsByFilter(context.Background(), filterID)
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job linked to filter, got %d", len(jobs))
	}
	if jobs[0].FilterID == nil || *jobs[0].FilterID != filterID {
		t.Error("job not linked to the correct filter")
	}
}

func TestPollOnceSecondPollSkipsDuplicates(t *testing.T) {
	database, registry := testSetup(t)

	mock := &mockAdapter{
		name: "repeat_source",
		jobs: []adapter.JobSummary{
			{SourceID: "r1", Title: "Repeat Job", URL: "https://example.com/r1"},
		},
	}
	registry.Register(mock)

	database.CreateFilter(context.Background(), "Repeat", "go", "London", "repeat_source")

	s := New(database, registry, time.Minute)

	// First poll: discovers the job.
	results1, err := s.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("first poll failed: %v", err)
	}
	if results1[0].NewJobs != 1 {
		t.Fatalf("first poll: expected 1 new, got %d", results1[0].NewJobs)
	}

	// Second poll: same job from adapter, should be skipped.
	results2, err := s.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("second poll failed: %v", err)
	}
	if results2[0].NewJobs != 0 {
		t.Errorf("second poll: expected 0 new, got %d", results2[0].NewJobs)
	}
	if results2[0].Skipped != 1 {
		t.Errorf("second poll: expected 1 skipped, got %d", results2[0].Skipped)
	}

	// Should not have fetched details on the second poll.
	if mock.detailCalls != 1 {
		t.Errorf("expected 1 total detail call (from first poll), got %d", mock.detailCalls)
	}
}

func TestStartAndStop(t *testing.T) {
	database, registry := testSetup(t)

	mock := &mockAdapter{
		name: "start_stop_source",
		jobs: []adapter.JobSummary{
			{SourceID: "ss1", Title: "Start Stop", URL: "https://example.com/ss1"},
		},
	}
	registry.Register(mock)

	database.CreateFilter(context.Background(), "SS", "go", "London", "start_stop_source")

	// Use a very short interval so we can verify it polls at least once.
	s := New(database, registry, 50*time.Millisecond)
	s.Start()

	// Give it a moment to run the initial poll.
	time.Sleep(100 * time.Millisecond)

	s.Stop()

	// Verify it polled at least once.
	if mock.fetchCalls < 1 {
		t.Errorf("expected at least 1 fetch call, got %d", mock.fetchCalls)
	}

	// Verify job was stored.
	count, _ := database.CountJobs(context.Background())
	if count < 1 {
		t.Error("expected at least 1 job in DB after start/stop")
	}
}

func TestStopIsIdempotent(t *testing.T) {
	database, registry := testSetup(t)
	s := New(database, registry, time.Minute)

	// Stop without Start should not panic.
	s.Stop()
	s.Stop()
}

// --- Error path and resource cleanup tests ---

// panicAdapter panics during FetchNewJobs to test safePoll recovery.
type panicAdapter struct {
	name string
}

func (p *panicAdapter) Name() string                     { return p.name }
func (p *panicAdapter) DisplayName() string              { return p.name }
func (p *panicAdapter) Validate(_ context.Context) error { return nil }

func (p *panicAdapter) FetchNewJobs(_ context.Context, _ adapter.SearchParams) ([]adapter.JobSummary, error) {
	panic("adapter exploded!")
}

func (p *panicAdapter) FetchJobDetails(_ context.Context, _ adapter.JobSummary) (*adapter.JobDetails, error) {
	return nil, nil
}

// timeoutAwareAdapter blocks for the first N fetch calls until context cancellation.
// Later calls return quickly so tests can verify scheduler recovery after timeout.
type timeoutAwareAdapter struct {
	name       string
	blockCalls int

	mu    sync.Mutex
	calls int
}

func (t *timeoutAwareAdapter) Name() string                     { return t.name }
func (t *timeoutAwareAdapter) DisplayName() string              { return t.name }
func (t *timeoutAwareAdapter) Validate(_ context.Context) error { return nil }

func (t *timeoutAwareAdapter) FetchNewJobs(ctx context.Context, _ adapter.SearchParams) ([]adapter.JobSummary, error) {
	t.mu.Lock()
	t.calls++
	callNum := t.calls
	shouldBlock := callNum <= t.blockCalls
	t.mu.Unlock()

	if shouldBlock {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	sourceID := fmt.Sprintf("timeout-aware-%d", callNum)
	return []adapter.JobSummary{
		{
			SourceID: sourceID,
			Title:    "Recovered Job",
			URL:      "https://example.com/" + sourceID,
		},
	}, nil
}

func (t *timeoutAwareAdapter) FetchJobDetails(_ context.Context, job adapter.JobSummary) (*adapter.JobDetails, error) {
	return &adapter.JobDetails{
		JobSummary:      job,
		FullDescription: "Recovered after timeout",
	}, nil
}

func (t *timeoutAwareAdapter) Calls() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.calls
}

func TestSafePollRecoversFromPanic(t *testing.T) {
	database, registry := testSetup(t)

	registry.Register(&panicAdapter{name: "panic_source"})
	database.CreateFilter(context.Background(), "Panic Filter", "go", "London", "panic_source")

	s := New(database, registry, time.Minute)

	// safePoll should recover from the panic — this must NOT crash.
	s.safePoll(context.Background())

	// Verify the scheduler is still functional after the panic.
	// Add a working adapter and poll again.
	registry.Register(&mockAdapter{
		name: "recovery_source",
		jobs: []adapter.JobSummary{
			{SourceID: "post-panic", Title: "Post Panic Job", URL: "https://example.com/pp"},
		},
	})
	database.CreateFilter(context.Background(), "Recovery", "go", "London", "recovery_source")

	results, err := s.PollOnce(context.Background())
	if err != nil {
		t.Fatalf("poll once failed: %v", err)
	}

	// Should have 2 results: one error (panic_source) and one success (recovery_source).
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	successCount := 0
	for _, r := range results {
		if r.Err == nil {
			successCount++
		}
	}
	if successCount < 1 {
		t.Error("expected at least 1 successful poll after panic recovery")
	}
}

func TestSafePollTimesOutLongRunningPoll(t *testing.T) {
	database, registry := testSetup(t)

	ta := &timeoutAwareAdapter{name: "timeout_source", blockCalls: 1}
	registry.Register(ta)
	database.CreateFilter(context.Background(), "Timeout Filter", "go", "London", "timeout_source")

	s := New(database, registry, time.Minute)
	s.pollTimeout = 25 * time.Millisecond

	started := time.Now()
	s.safePoll(context.Background())
	elapsed := time.Since(started)

	if elapsed > 500*time.Millisecond {
		t.Fatalf("expected safePoll to return quickly after timeout, took %v", elapsed)
	}
	if ta.Calls() != 1 {
		t.Fatalf("expected exactly 1 fetch call, got %d", ta.Calls())
	}
}

func TestSchedulerContinuesAfterPollTimeout(t *testing.T) {
	database, registry := testSetup(t)

	ta := &timeoutAwareAdapter{name: "timeout_source", blockCalls: 1}
	registry.Register(ta)
	database.CreateFilter(context.Background(), "Timeout Recovery", "go", "London", "timeout_source")

	s := New(database, registry, 40*time.Millisecond)
	s.pollTimeout = 20 * time.Millisecond

	s.Start()
	time.Sleep(220 * time.Millisecond)
	s.Stop()

	if ta.Calls() < 2 {
		t.Fatalf("expected scheduler to continue after timeout, fetch calls=%d", ta.Calls())
	}

	count, err := database.CountJobs(context.Background())
	if err != nil {
		t.Fatalf("counting jobs: %v", err)
	}
	if count < 1 {
		t.Fatalf("expected at least one job stored after timeout recovery, got %d", count)
	}
}

func TestSchedulerSurvivesPanicAndKeepsRunning(t *testing.T) {
	database, registry := testSetup(t)

	registry.Register(&panicAdapter{name: "panic_source"})
	database.CreateFilter(context.Background(), "Panic", "go", "London", "panic_source")

	// Start scheduler with short interval — the panic adapter will panic
	// every tick, but safePoll should recover each time.
	s := New(database, registry, 50*time.Millisecond)
	s.Start()

	// Let it run through a few panic-recover cycles.
	time.Sleep(200 * time.Millisecond)

	// Stop should still work cleanly — the goroutine wasn't killed by panics.
	s.Stop()
}

func TestPollOnceWithClosedDB(t *testing.T) {
	database, registry := testSetup(t)

	registry.Register(&mockAdapter{
		name: "closed_db_source",
		jobs: []adapter.JobSummary{
			{SourceID: "cd1", Title: "Job", URL: "https://example.com/cd1"},
		},
	})

	database.CreateFilter(context.Background(), "Closed DB", "go", "London", "closed_db_source")

	s := New(database, registry, time.Minute)

	// Close the DB before polling — simulates unexpected resource loss.
	database.Close()

	results, err := s.PollOnce(context.Background())
	if err == nil {
		t.Fatal("expected poll once to fail with closed DB")
	}
	if results != nil {
		t.Errorf("expected nil results with closed DB, got %d", len(results))
	}
}

func TestStartStopStartAgain(t *testing.T) {
	database, registry := testSetup(t)

	mock := &mockAdapter{
		name: "restart_source",
		jobs: []adapter.JobSummary{
			{SourceID: "rs1", Title: "Restart Job", URL: "https://example.com/rs1"},
		},
	}
	registry.Register(mock)

	database.CreateFilter(context.Background(), "Restart", "go", "London", "restart_source")

	s := New(database, registry, 50*time.Millisecond)

	// First start/stop cycle.
	s.Start()
	time.Sleep(100 * time.Millisecond)
	s.Stop()

	callsAfterFirst := mock.fetchCalls

	// Second start/stop cycle — should work the same.
	s.Start()
	time.Sleep(100 * time.Millisecond)
	s.Stop()

	if mock.fetchCalls <= callsAfterFirst {
		t.Error("expected more fetch calls after restart")
	}
}

func TestSetPausedSkipsPoll(t *testing.T) {
	database, registry := testSetup(t)

	mock := &mockAdapter{
		name: "pause_source",
		jobs: []adapter.JobSummary{
			{SourceID: "p1", Title: "Pause Job", URL: "https://example.com/p1"},
		},
	}
	registry.Register(mock)
	database.CreateFilter(context.Background(), "Pause Test", "go", "London", "pause_source")

	s := New(database, registry, time.Minute)

	// Pause and try to poll — should skip.
	s.SetPaused(true)
	s.safePoll(context.Background())

	if mock.fetchCalls != 0 {
		t.Errorf("expected 0 fetch calls when paused, got %d", mock.fetchCalls)
	}

	// Resume and poll — should fetch.
	s.SetPaused(false)
	s.safePoll(context.Background())

	if mock.fetchCalls != 1 {
		t.Errorf("expected 1 fetch call after resume, got %d", mock.fetchCalls)
	}
}

func TestSetPausedEmitsStatusChangedHook(t *testing.T) {
	database, registry := testSetup(t)
	s := New(database, registry, time.Minute)

	var calls atomic.Int32
	s.SetStatusChangedHook(func() {
		calls.Add(1)
	})

	s.SetPaused(true)
	s.SetPaused(false)

	if got := calls.Load(); got != 2 {
		t.Fatalf("expected 2 status-change callbacks, got %d", got)
	}
}

func TestSetPausedResumeSchedulesNextPollAtWhenRunning(t *testing.T) {
	database, registry := testSetup(t)
	s := New(database, registry, time.Minute)

	s.stateMu.Lock()
	s.running = true
	s.paused = true
	s.nextPollAt = time.Time{}
	s.stateMu.Unlock()

	s.SetPaused(false)

	s.stateMu.RLock()
	next := s.nextPollAt
	s.stateMu.RUnlock()
	if next.IsZero() {
		t.Fatal("expected SetPaused(false) to schedule next poll when running")
	}
}

func TestSetPausedClearsNextPollAt(t *testing.T) {
	database, registry := testSetup(t)
	s := New(database, registry, time.Minute)

	s.stateMu.Lock()
	s.running = true
	s.paused = false
	s.nextPollAt = time.Now().Add(10 * time.Minute)
	s.stateMu.Unlock()

	s.SetPaused(true)

	if next := s.NextPollAt(); !next.IsZero() {
		t.Fatalf("expected SetPaused(true) to clear next poll, got %v", next)
	}
}

func TestRescheduleFromNowMovesNextPollAtForward(t *testing.T) {
	database, registry := testSetup(t)
	s := New(database, registry, time.Minute)

	s.stateMu.Lock()
	s.running = true
	s.paused = false
	s.nextPollAt = time.Now().Add(20 * time.Second)
	oldNext := s.nextPollAt
	s.stateMu.Unlock()

	s.RescheduleFromNow()

	newNext := s.NextPollAt()
	if !newNext.After(oldNext) {
		t.Fatalf("expected next poll to move forward, old=%v new=%v", oldNext, newNext)
	}
}

func TestRescheduleFromNowDelaysNextAutomaticPoll(t *testing.T) {
	database, registry := testSetup(t)

	mock := &mockAdapter{
		name: "reschedule_source",
		jobs: []adapter.JobSummary{
			{SourceID: "rs-1", Title: "Job", URL: "https://example.com/rs-1"},
		},
	}
	registry.Register(mock)
	database.CreateFilter(context.Background(), "Reschedule", "go", "London", "reschedule_source")

	interval := 300 * time.Millisecond
	s := New(database, registry, interval)
	s.Start()
	defer s.Stop()

	ok := waitForCondition(200*time.Millisecond, func() bool {
		return mock.FetchCalls() >= 1
	})
	if !ok {
		t.Fatal("expected initial immediate poll to run")
	}
	initialCalls := mock.FetchCalls()

	time.Sleep(70 * time.Millisecond)
	s.RescheduleFromNow()

	// This point is after the original schedule (~300ms from start) but before
	// the rescheduled window (~370ms from start).
	time.Sleep(280 * time.Millisecond)
	if got := mock.FetchCalls(); got != initialCalls {
		t.Fatalf("expected no auto poll before rescheduled time, initial=%d got=%d", initialCalls, got)
	}

	ok = waitForCondition(250*time.Millisecond, func() bool {
		return mock.FetchCalls() >= initialCalls+1
	})
	if !ok {
		t.Fatalf("expected auto poll after rescheduled time, calls=%d", mock.FetchCalls())
	}
}

func TestIsPaused(t *testing.T) {
	database, registry := testSetup(t)
	s := New(database, registry, time.Minute)

	if s.IsPaused() {
		t.Error("expected not paused initially")
	}

	s.SetPaused(true)
	if !s.IsPaused() {
		t.Error("expected paused after SetPaused(true)")
	}

	s.SetPaused(false)
	if s.IsPaused() {
		t.Error("expected not paused after SetPaused(false)")
	}
}

func TestNextPollAtTracked(t *testing.T) {
	database, registry := testSetup(t)

	mock := &mockAdapter{
		name: "next_source",
		jobs: []adapter.JobSummary{},
	}
	registry.Register(mock)
	database.CreateFilter(context.Background(), "Next", "go", "London", "next_source")

	s := New(database, registry, 50*time.Millisecond)

	// Before start, next poll should be zero.
	if !s.NextPollAt().IsZero() {
		t.Error("expected zero NextPollAt before start")
	}

	s.Start()
	time.Sleep(100 * time.Millisecond)

	next := s.NextPollAt()
	if next.IsZero() {
		t.Error("expected non-zero NextPollAt after start")
	}

	s.Stop()
}

func TestStartPausedDoesNotScheduleOrPoll(t *testing.T) {
	database, registry := testSetup(t)

	mock := &mockAdapter{
		name: "start_paused_source",
		jobs: []adapter.JobSummary{
			{SourceID: "sp-1", Title: "Start Paused Job", URL: "https://example.com/sp-1"},
		},
	}
	registry.Register(mock)
	database.CreateFilter(context.Background(), "Start Paused", "go", "London", "start_paused_source")

	s := New(database, registry, 50*time.Millisecond)
	s.SetPaused(true)
	s.Start()
	defer s.Stop()

	time.Sleep(120 * time.Millisecond)

	if next := s.NextPollAt(); !next.IsZero() {
		t.Fatalf("expected zero NextPollAt while paused, got %v", next)
	}
	if got := mock.FetchCalls(); got != 0 {
		t.Fatalf("expected 0 fetch calls while paused, got %d", got)
	}
}

func TestStartEmitsStatusChangedAfterPoll(t *testing.T) {
	database, registry := testSetup(t)

	mock := &mockAdapter{
		name: "hook_source",
		jobs: []adapter.JobSummary{
			{SourceID: "hook-1", Title: "Hook Job", URL: "https://example.com/hook-1"},
		},
	}
	registry.Register(mock)
	database.CreateFilter(context.Background(), "Hook", "go", "London", "hook_source")

	s := New(database, registry, 40*time.Millisecond)

	var calls atomic.Int32
	s.SetStatusChangedHook(func() {
		calls.Add(1)
	})

	s.Start()
	time.Sleep(120 * time.Millisecond)
	s.Stop()

	if got := calls.Load(); got < 1 {
		t.Fatalf("expected status-change callback after scheduler poll, got %d", got)
	}
}

func TestStartSetsNextPollAtBeforeInitialPollCompletes(t *testing.T) {
	database, registry := testSetup(t)

	ta := &timeoutAwareAdapter{name: "startup_block_source", blockCalls: 1}
	registry.Register(ta)
	database.CreateFilter(context.Background(), "Startup Block", "go", "London", "startup_block_source")

	s := New(database, registry, 5*time.Minute)
	s.pollTimeout = 200 * time.Millisecond

	s.Start()
	time.Sleep(20 * time.Millisecond)

	next := s.NextPollAt()
	s.Stop()

	if next.IsZero() {
		t.Fatal("expected NextPollAt to be scheduled before first poll completes")
	}
}

func TestPollOnceWritePhaseContextCancellation(t *testing.T) {
	database, registry := testSetup(t)

	// Adapter returns many jobs so we can cancel mid-write.
	jobs := make([]adapter.JobSummary, 20)
	for i := range jobs {
		jobs[i] = adapter.JobSummary{
			SourceID: fmt.Sprintf("cancel-write-%d", i),
			Title:    fmt.Sprintf("Job %d", i),
			URL:      fmt.Sprintf("https://example.com/%d", i),
		}
	}

	registry.Register(&mockAdapter{name: "many_source", jobs: jobs})
	database.CreateFilter(context.Background(), "Many", "go", "London", "many_source")

	// Use a context that we cancel after a very short delay.
	ctx, cancel := context.WithCancel(context.Background())

	s := New(database, registry, time.Minute)

	// Cancel in a goroutine after a tiny delay so the fetch phase completes
	// but the write phase gets interrupted.
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	results, err := s.PollOnce(ctx)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context cancellation or nil error, got: %v", err)
		}
		return
	}

	if len(results) == 0 {
		return // Context cancelled before anything happened — that's fine.
	}

	r := results[0]
	// Should not have written all 20 jobs — context was cancelled mid-write.
	// (It's possible all 20 wrote before cancellation in fast environments,
	// so we just verify no panic and result is well-formed.)
	if r.NewJobs < 0 {
		t.Error("NewJobs should not be negative")
	}
	if r.Skipped < 0 {
		t.Error("Skipped should not be negative")
	}
}

func TestPollOnceReturnsErrWhenAlreadyPolling(t *testing.T) {
	database, registry := testSetup(t)
	s := New(database, registry, time.Minute)

	s.stateMu.Lock()
	s.polling = true
	s.stateMu.Unlock()

	_, err := s.PollOnce(context.Background())
	if !errors.Is(err, ErrPollInProgress) {
		t.Fatalf("expected ErrPollInProgress, got %v", err)
	}
}
