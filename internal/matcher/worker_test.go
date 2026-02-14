package matcher

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"hamster-wheel/internal/db"
	"hamster-wheel/internal/llm"
	"hamster-wheel/internal/llm/heuristic"
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
