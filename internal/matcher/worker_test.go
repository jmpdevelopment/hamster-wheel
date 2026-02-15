package matcher

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"hamster-wheel/internal/db"
	"hamster-wheel/internal/llm"
	"hamster-wheel/internal/llm/heuristic"
	"hamster-wheel/internal/llm/openai"
)

func testDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.OpenAt(filepath.Join(t.TempDir(), "matcher-test.db"))
	if err != nil {
		t.Fatalf("opening test DB: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func testWorker(t *testing.T, database *db.DB) *Worker {
	t.Helper()
	registry := llm.NewRegistry()
	if err := registry.Register(heuristic.New()); err != nil {
		t.Fatalf("registering heuristic provider: %v", err)
	}
	return New(database, registry, WorkerConfig{
		ProviderName: heuristic.ProviderName,
		PollInterval: 100 * time.Millisecond,
		BatchSize:    5,
		StaleAfter:   10 * time.Second,
	})
}

func TestRunOnceProcessesPendingRows(t *testing.T) {
	database := testDB(t)
	worker := testWorker(t, database)

	filterID, err := database.CreateFilter(context.Background(), "Backend", "go backend api", "Remote", "reed_uk")
	if err != nil {
		t.Fatalf("creating filter: %v", err)
	}
	jobID, err := database.InsertJob(context.Background(), &db.Job{
		Source:      "reed_uk",
		SourceID:    "matcher-pending-1",
		Title:       "Go Backend Engineer",
		Company:     "Acme",
		Location:    "Remote",
		Description: "Build Go APIs and backend services.",
		URL:         "https://example.com/jobs/1",
		FilterID:    &filterID,
	})
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}
	if err := database.EnsureJobMatchPending(context.Background(), jobID); err != nil {
		t.Fatalf("ensuring pending row: %v", err)
	}

	processed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("running matcher once: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected 1 processed row, got %d", processed)
	}

	match, err := database.GetJobMatchByJobID(context.Background(), jobID)
	if err != nil {
		t.Fatalf("getting match row: %v", err)
	}
	if match == nil {
		t.Fatal("expected match row")
	}
	if match.Status != db.JobMatchStatusMatched {
		t.Fatalf("expected matched status, got %q", match.Status)
	}
}

func TestRunOnceNoPendingRows(t *testing.T) {
	database := testDB(t)
	worker := testWorker(t, database)

	processed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("running matcher with no rows: %v", err)
	}
	if processed != 0 {
		t.Fatalf("expected 0 processed rows, got %d", processed)
	}
}

func TestRunOnceMarksMissingProviderAsFailed(t *testing.T) {
	database := testDB(t)
	registry := llm.NewRegistry()
	worker := New(database, registry, WorkerConfig{
		ProviderName: "missing",
		PollInterval: 100 * time.Millisecond,
		BatchSize:    1,
	})

	jobID, err := database.InsertJob(context.Background(), &db.Job{
		Source:      "reed_uk",
		SourceID:    "matcher-pending-missing-provider",
		Title:       "Go Backend Engineer",
		Description: "Build APIs.",
		URL:         "https://example.com/jobs/2",
	})
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}
	if err := database.EnsureJobMatchPending(context.Background(), jobID); err != nil {
		t.Fatalf("ensuring pending row: %v", err)
	}

	processed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("running matcher once: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected processed row count 1, got %d", processed)
	}

	match, err := database.GetJobMatchByJobID(context.Background(), jobID)
	if err != nil {
		t.Fatalf("getting match row: %v", err)
	}
	if match == nil {
		t.Fatal("expected match row")
	}
	if match.Status != db.JobMatchStatusFailed {
		t.Fatalf("expected failed status, got %q", match.Status)
	}
}

func TestRunOnceProcessesPendingRowsWithOpenAIProvider(t *testing.T) {
	database := testDB(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("expected chat completions path, got %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"message": {
						"content": "{\"score\":0.76,\"summary\":\"Strong alignment with Go backend responsibilities.\"}"
					}
				}
			],
			"usage": {"prompt_tokens": 81}
		}`))
	}))
	defer server.Close()

	registry := llm.NewRegistry()
	if err := registry.Register(openai.New(openai.Config{
		APIKey:     "test-key",
		Model:      "gpt-4o-mini",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})); err != nil {
		t.Fatalf("registering openai provider: %v", err)
	}

	worker := New(database, registry, WorkerConfig{
		ProviderName: openai.ProviderName,
		PollInterval: 100 * time.Millisecond,
		BatchSize:    2,
		StaleAfter:   10 * time.Second,
	})

	filterID, err := database.CreateFilter(context.Background(), "Backend", "go backend api", "Remote", "reed_uk")
	if err != nil {
		t.Fatalf("creating filter: %v", err)
	}
	jobID, err := database.InsertJob(context.Background(), &db.Job{
		Source:      "reed_uk",
		SourceID:    "matcher-pending-openai-1",
		Title:       "Go Backend Engineer",
		Company:     "Acme",
		Location:    "Remote",
		Description: "Build Go APIs and backend services.",
		URL:         "https://example.com/jobs/openai",
		FilterID:    &filterID,
	})
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}
	if err := database.EnsureJobMatchPending(context.Background(), jobID); err != nil {
		t.Fatalf("ensuring pending row: %v", err)
	}

	processed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("running matcher once: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected 1 processed row, got %d", processed)
	}

	match, err := database.GetJobMatchByJobID(context.Background(), jobID)
	if err != nil {
		t.Fatalf("getting match row: %v", err)
	}
	if match == nil {
		t.Fatal("expected match row")
	}
	if match.Status != db.JobMatchStatusMatched {
		t.Fatalf("expected matched status, got %q", match.Status)
	}
	if match.MatchScore <= 0 {
		t.Fatalf("expected positive match score, got %.2f", match.MatchScore)
	}
	if !strings.HasPrefix(match.MatchSummary, "Provider: openai\n") {
		t.Fatalf("expected openai provider prefix in summary, got %q", match.MatchSummary)
	}
}

func TestRunOnceUsesProviderResolver(t *testing.T) {
	database := testDB(t)
	registry := llm.NewRegistry()
	if err := registry.Register(heuristic.New()); err != nil {
		t.Fatalf("registering heuristic provider: %v", err)
	}

	worker := New(database, registry, WorkerConfig{
		ProviderName: heuristic.ProviderName,
		ProviderResolver: func(context.Context) (string, llm.Provider, error) {
			return openai.ProviderName, &stubProvider{
				name: openai.ProviderName,
				result: llm.MatchResult{
					Score:   0.61,
					Summary: "Resolver selected provider result.",
				},
			}, nil
		},
		PollInterval: 100 * time.Millisecond,
		BatchSize:    1,
	})

	jobID, err := database.InsertJob(context.Background(), &db.Job{
		Source:      "reed_uk",
		SourceID:    "matcher-provider-resolver-1",
		Title:       "Go Backend Engineer",
		Description: "Build APIs.",
		URL:         "https://example.com/jobs/3",
	})
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}
	if err := database.EnsureJobMatchPending(context.Background(), jobID); err != nil {
		t.Fatalf("ensuring pending row: %v", err)
	}

	processed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("running matcher once: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected processed row count 1, got %d", processed)
	}

	match, err := database.GetJobMatchByJobID(context.Background(), jobID)
	if err != nil {
		t.Fatalf("getting match row: %v", err)
	}
	if match == nil {
		t.Fatal("expected match row")
	}
	if !strings.HasPrefix(match.MatchSummary, "Provider: openai\n") {
		t.Fatalf("expected provider resolver output in summary, got %q", match.MatchSummary)
	}
}

func TestRunOnceUsesProviderSpecificTimeoutOverride(t *testing.T) {
	database := testDB(t)
	probe := &deadlineProbeProvider{
		name: "local_ollama",
		result: llm.MatchResult{
			Score:   0.55,
			Summary: "Local scoring completed.",
		},
	}

	worker := New(database, llm.NewRegistry(), WorkerConfig{
		ProviderName: "local_ollama",
		ProviderResolver: func(context.Context) (string, llm.Provider, error) {
			return "local_ollama", probe, nil
		},
		BatchSize:    1,
		MatchTimeout: 25 * time.Millisecond,
		ProviderTimeouts: map[string]time.Duration{
			"local_ollama": 150 * time.Millisecond,
		},
	})

	jobID, err := database.InsertJob(context.Background(), &db.Job{
		Source:      "reed_uk",
		SourceID:    "matcher-timeout-override-1",
		Title:       "Go Backend Engineer",
		Description: "Build APIs.",
		URL:         "https://example.com/jobs/timeout-override",
	})
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}
	if err := database.EnsureJobMatchPending(context.Background(), jobID); err != nil {
		t.Fatalf("ensuring pending row: %v", err)
	}

	processed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("running matcher once: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected 1 processed row, got %d", processed)
	}

	deadlineBudget, ok := probe.deadlineSnapshot()
	if !ok {
		t.Fatal("expected provider match context to include a deadline")
	}
	if deadlineBudget < 80*time.Millisecond {
		t.Fatalf("expected provider-specific timeout override budget >= 80ms, got %v", deadlineBudget)
	}
}

func TestRunOnceIncludesCVProfileFromConfiguredPath(t *testing.T) {
	database := testDB(t)
	cvPath := filepath.Join(t.TempDir(), "candidate-cv.txt")
	if err := os.WriteFile(cvPath, []byte("Go backend APIs distributed systems"), 0o600); err != nil {
		t.Fatalf("writing cv file: %v", err)
	}
	if err := database.SetSetting(context.Background(), settingCVPath, cvPath); err != nil {
		t.Fatalf("setting cv path: %v", err)
	}

	stub := &stubProvider{
		name: heuristic.ProviderName,
		result: llm.MatchResult{
			Score:   0.52,
			Summary: "Scored with CV context.",
		},
	}
	worker := New(database, llm.NewRegistry(), WorkerConfig{
		ProviderResolver: func(context.Context) (string, llm.Provider, error) {
			return heuristic.ProviderName, stub, nil
		},
		BatchSize: 1,
	})

	jobID, err := database.InsertJob(context.Background(), &db.Job{
		Source:      "reed_uk",
		SourceID:    "matcher-cv-profile-1",
		Title:       "Go Backend Engineer",
		Company:     "Acme",
		Location:    "Remote",
		Description: "Build Go APIs and backend services.",
		URL:         "https://example.com/jobs/cv",
	})
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}
	if err := database.EnsureJobMatchPending(context.Background(), jobID); err != nil {
		t.Fatalf("ensuring pending row: %v", err)
	}

	processed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("running matcher once: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected 1 processed row, got %d", processed)
	}
	if !strings.Contains(stub.lastReq.CandidateProfile, "distributed systems") {
		t.Fatalf("expected CV profile to be forwarded into match request, got %q", stub.lastReq.CandidateProfile)
	}
}

func TestRunOnceFallsBackWhenCVProfileCannotBeParsed(t *testing.T) {
	database := testDB(t)
	if err := database.SetSetting(context.Background(), settingCVPath, filepath.Join(t.TempDir(), "missing-cv.txt")); err != nil {
		t.Fatalf("setting cv path: %v", err)
	}

	stub := &stubProvider{
		name: heuristic.ProviderName,
		result: llm.MatchResult{
			Score:   0.41,
			Summary: "Scored without CV context.",
		},
	}
	worker := New(database, llm.NewRegistry(), WorkerConfig{
		ProviderResolver: func(context.Context) (string, llm.Provider, error) {
			return heuristic.ProviderName, stub, nil
		},
		BatchSize: 1,
	})

	jobID, err := database.InsertJob(context.Background(), &db.Job{
		Source:      "reed_uk",
		SourceID:    "matcher-cv-profile-fallback-1",
		Title:       "Go Backend Engineer",
		Description: "Build APIs.",
		URL:         "https://example.com/jobs/cv-fallback",
	})
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}
	if err := database.EnsureJobMatchPending(context.Background(), jobID); err != nil {
		t.Fatalf("ensuring pending row: %v", err)
	}

	processed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("running matcher once: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected 1 processed row, got %d", processed)
	}
	if stub.lastReq.CandidateProfile != "" {
		t.Fatalf("expected empty CV profile on parse failure fallback, got %q", stub.lastReq.CandidateProfile)
	}
}

func TestWorkerStartStopIdempotent(t *testing.T) {
	database := testDB(t)
	worker := testWorker(t, database)

	worker.Start()
	worker.Start() // no-op when already running
	time.Sleep(50 * time.Millisecond)
	worker.Stop()
	worker.Stop() // safe when already stopped
}

func TestWorkerSetLoggerAndStatusHook(t *testing.T) {
	database := testDB(t)
	worker := testWorker(t, database)

	worker.SetLogger(nil) // no-op path
	customLogger := slog.Default().With("test", "matcher-worker")
	worker.SetLogger(customLogger)
	if worker.log() == nil {
		t.Fatal("expected worker logger to be set")
	}

	var (
		mu       sync.Mutex
		statuses []string
	)
	worker.SetStatusChangedHook(func(_ string, status string) {
		mu.Lock()
		defer mu.Unlock()
		statuses = append(statuses, status)
	})

	filterID, err := database.CreateFilter(context.Background(), "Backend", "go backend api", "Remote", "reed_uk")
	if err != nil {
		t.Fatalf("creating filter: %v", err)
	}
	jobID, err := database.InsertJob(context.Background(), &db.Job{
		Source:      "reed_uk",
		SourceID:    "matcher-status-hook-1",
		Title:       "Go Backend Engineer",
		Company:     "Acme",
		Location:    "Remote",
		Description: "Build Go APIs.",
		URL:         "https://example.com/jobs/hook",
		FilterID:    &filterID,
	})
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}
	if err := database.EnsureJobMatchPending(context.Background(), jobID); err != nil {
		t.Fatalf("ensuring pending row: %v", err)
	}

	processed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("running matcher once: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected 1 processed row, got %d", processed)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(statuses) < 2 {
		t.Fatalf("expected at least processing+matched status updates, got %v", statuses)
	}
}

func TestRunOnceRequeueError(t *testing.T) {
	requeueErr := errors.New("requeue failed")
	worker := New(requeueErrStore{err: requeueErr}, llm.NewRegistry(), WorkerConfig{
		ProviderName: heuristic.ProviderName,
	})

	_, err := worker.RunOnce(context.Background())
	if err == nil {
		t.Fatal("expected requeue error from RunOnce")
	}
	if !errors.Is(err, requeueErr) {
		t.Fatalf("expected wrapped requeue error, got %v", err)
	}
}

type requeueErrStore struct {
	err error
}

func (s requeueErrStore) ClaimNextPendingJobMatch(context.Context) (*db.JobMatch, error) {
	return nil, nil
}

func (s requeueErrStore) MarkJobMatchMatched(context.Context, string, float64, string) error {
	return nil
}

func (s requeueErrStore) MarkJobMatchFailed(context.Context, string, string) error {
	return nil
}

func (s requeueErrStore) RequeueStaleProcessingJobMatches(context.Context, time.Time) (int, error) {
	return 0, s.err
}

func (s requeueErrStore) GetJob(context.Context, string) (*db.Job, error) {
	return nil, nil
}

func (s requeueErrStore) GetFilter(context.Context, string) (*db.SearchFilter, error) {
	return nil, nil
}

func (s requeueErrStore) GetSetting(context.Context, string) (string, error) {
	return "", nil
}

type stubProvider struct {
	name    string
	result  llm.MatchResult
	err     error
	lastReq llm.MatchRequest
}

type deadlineProbeProvider struct {
	name   string
	result llm.MatchResult

	mu             sync.Mutex
	seenDeadline   bool
	deadlineBudget time.Duration
}

func (p *deadlineProbeProvider) Name() string {
	return p.name
}

func (p *deadlineProbeProvider) DisplayName() string {
	return p.name
}

func (p *deadlineProbeProvider) Validate(context.Context) error {
	return nil
}

func (p *deadlineProbeProvider) Match(ctx context.Context, _ llm.MatchRequest) (llm.MatchResult, error) {
	deadline, ok := ctx.Deadline()
	budget := time.Duration(0)
	if ok {
		budget = time.Until(deadline)
	}

	p.mu.Lock()
	p.seenDeadline = ok
	p.deadlineBudget = budget
	p.mu.Unlock()

	return p.result, nil
}

func (p *deadlineProbeProvider) deadlineSnapshot() (time.Duration, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.deadlineBudget, p.seenDeadline
}

func (p *stubProvider) Name() string {
	return p.name
}

func (p *stubProvider) DisplayName() string {
	return p.name
}

func (p *stubProvider) Validate(context.Context) error {
	return nil
}

func (p *stubProvider) Match(_ context.Context, req llm.MatchRequest) (llm.MatchResult, error) {
	p.lastReq = req
	if p.err != nil {
		return llm.MatchResult{}, p.err
	}
	return p.result, nil
}
