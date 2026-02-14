package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"hamster-wheel/internal/adapter"
	"hamster-wheel/internal/db"
)

type retryDetailsAdapter struct {
	source         string
	fetchErr       error
	fetchDetails   *adapter.JobDetails
	fetchCallCount int
	lastSummary    adapter.JobSummary
}

func (a *retryDetailsAdapter) Name() string {
	return a.source
}

func (a *retryDetailsAdapter) DisplayName() string {
	return "Retry Adapter"
}

func (a *retryDetailsAdapter) FetchNewJobs(context.Context, adapter.SearchParams) ([]adapter.JobSummary, error) {
	return nil, nil
}

func (a *retryDetailsAdapter) FetchJobDetails(_ context.Context, job adapter.JobSummary) (*adapter.JobDetails, error) {
	a.fetchCallCount++
	a.lastSummary = job
	if a.fetchErr != nil {
		return nil, a.fetchErr
	}
	if a.fetchDetails != nil {
		return a.fetchDetails, nil
	}
	return &adapter.JobDetails{JobSummary: job, FullDescription: "retried"}, nil
}

func (a *retryDetailsAdapter) Validate(context.Context) error {
	return nil
}

func openJobServiceTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.OpenAt(filepath.Join(t.TempDir(), "job-service-test.db"))
	if err != nil {
		t.Fatalf("opening test DB: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func insertRetryJob(t *testing.T, database *db.DB, source string) string {
	t.Helper()
	id, err := database.InsertJob(context.Background(), &db.Job{
		Source:      source,
		SourceID:    "source-123",
		Title:       "Go Engineer",
		Company:     "Acme",
		Location:    "Remote",
		Description: "",
		URL:         "https://example.com/jobs/123",
	})
	if err != nil {
		t.Fatalf("inserting test job: %v", err)
	}
	return id
}

func TestRetryFetchDescriptionSuccess(t *testing.T) {
	database := openJobServiceTestDB(t)
	registry := adapter.NewRegistry()
	retryAdapter := &retryDetailsAdapter{
		source: "reed_uk",
		fetchDetails: &adapter.JobDetails{
			FullDescription: "retried description",
		},
	}
	if err := registry.Register(retryAdapter); err != nil {
		t.Fatalf("registering adapter: %v", err)
	}

	jobID := insertRetryJob(t, database, "reed_uk")

	service := NewJobService(database, registry)
	if err := service.RetryFetchDescription(jobID); err != nil {
		t.Fatalf("RetryFetchDescription failed: %v", err)
	}

	if retryAdapter.fetchCallCount != 1 {
		t.Fatalf("expected one detail fetch call, got %d", retryAdapter.fetchCallCount)
	}
	if retryAdapter.lastSummary.SourceID != "source-123" {
		t.Fatalf("expected summary SourceID source-123, got %q", retryAdapter.lastSummary.SourceID)
	}
	if retryAdapter.lastSummary.Title != "Go Engineer" {
		t.Fatalf("expected summary Title Go Engineer, got %q", retryAdapter.lastSummary.Title)
	}

	job, err := database.GetJob(context.Background(), jobID)
	if err != nil {
		t.Fatalf("loading updated job: %v", err)
	}
	if job == nil {
		t.Fatal("expected updated job to exist")
	}
	if job.Description != "retried description" {
		t.Fatalf("expected updated description, got %q", job.Description)
	}
}

func TestRetryFetchDescriptionNotFound(t *testing.T) {
	database := openJobServiceTestDB(t)
	service := NewJobService(database, adapter.NewRegistry())

	err := service.RetryFetchDescription("missing-job-id")
	if !errors.Is(err, db.ErrJobNotFound) {
		t.Fatalf("expected ErrJobNotFound, got %v", err)
	}
}

func TestRetryFetchDescriptionAdapterMissing(t *testing.T) {
	database := openJobServiceTestDB(t)
	jobID := insertRetryJob(t, database, "reed_uk")
	service := NewJobService(database, adapter.NewRegistry())

	err := service.RetryFetchDescription(jobID)
	if err == nil {
		t.Fatal("expected missing adapter error")
	}
	if !strings.Contains(err.Error(), "no adapter registered") {
		t.Fatalf("expected missing adapter message, got %v", err)
	}
}

func TestRetryFetchDescriptionAdapterError(t *testing.T) {
	database := openJobServiceTestDB(t)
	registry := adapter.NewRegistry()
	retryAdapter := &retryDetailsAdapter{
		source:   "reed_uk",
		fetchErr: errors.New("upstream timeout"),
	}
	if err := registry.Register(retryAdapter); err != nil {
		t.Fatalf("registering adapter: %v", err)
	}

	jobID := insertRetryJob(t, database, "reed_uk")
	service := NewJobService(database, registry)

	err := service.RetryFetchDescription(jobID)
	if err == nil {
		t.Fatal("expected fetch error")
	}
	if !strings.Contains(err.Error(), "fetching description") {
		t.Fatalf("expected wrapped fetch context, got %v", err)
	}
	if !strings.Contains(err.Error(), "upstream timeout") {
		t.Fatalf("expected original fetch error message, got %v", err)
	}
}

func TestRetryFetchDescriptionLoadError(t *testing.T) {
	database := openJobServiceTestDB(t)
	service := NewJobService(database, adapter.NewRegistry())

	if err := database.Close(); err != nil {
		t.Fatalf("closing DB: %v", err)
	}

	err := service.RetryFetchDescription("any-id")
	if err == nil {
		t.Fatal("expected loading error")
	}
	if !strings.Contains(err.Error(), "loading job") {
		t.Fatalf("expected loading-job context in error, got %v", err)
	}
}

func TestSetJobFavoriteUpdatesJob(t *testing.T) {
	database := openJobServiceTestDB(t)
	service := NewJobService(database, adapter.NewRegistry())

	jobID := insertRetryJob(t, database, "reed_uk")

	if err := service.SetJobFavorite(jobID, true); err != nil {
		t.Fatalf("SetJobFavorite failed: %v", err)
	}

	job, err := database.GetJob(context.Background(), jobID)
	if err != nil {
		t.Fatalf("loading job: %v", err)
	}
	if job == nil {
		t.Fatal("expected job to exist")
	}
	if !job.IsFavorite {
		t.Fatal("expected job to be favorite")
	}
}

func TestSetJobsFavoriteUpdatesJobs(t *testing.T) {
	database := openJobServiceTestDB(t)
	service := NewJobService(database, adapter.NewRegistry())

	jobID1 := insertRetryJob(t, database, "reed_uk")
	jobID2, err := database.InsertJob(context.Background(), &db.Job{
		Source:      "reed_uk",
		SourceID:    "source-456",
		Title:       "React Engineer",
		Company:     "Acme",
		Location:    "Remote",
		Description: "",
		URL:         "https://example.com/jobs/456",
	})
	if err != nil {
		t.Fatalf("inserting second job: %v", err)
	}

	if err := service.SetJobsFavorite([]string{jobID1, jobID2}, true); err != nil {
		t.Fatalf("SetJobsFavorite failed: %v", err)
	}

	jobs, err := database.ListJobs(context.Background(), 0)
	if err != nil {
		t.Fatalf("listing jobs: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}
	for _, job := range jobs {
		if !job.IsFavorite {
			t.Fatalf("expected job %q to be favorite", job.ID)
		}
	}
}
