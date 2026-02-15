package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"hamster-wheel/internal/adapter"
	"hamster-wheel/internal/db"
	"hamster-wheel/internal/llm"
	"hamster-wheel/internal/llm/heuristic"
	"hamster-wheel/internal/matcher"
	"hamster-wheel/internal/scheduler"
)

func openAppServiceTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.OpenAt(filepath.Join(t.TempDir(), "app-service-test.db"))
	if err != nil {
		t.Fatalf("opening test DB: %v", err)
	}
	return database
}

func TestAppServiceStartupAndShutdown(t *testing.T) {
	database := openAppServiceTestDB(t)
	sched := scheduler.New(database, adapter.NewRegistry(), time.Hour)
	registry := llm.NewRegistry()
	if err := registry.Register(heuristic.New()); err != nil {
		t.Fatalf("registering provider: %v", err)
	}
	matchWorker := matcher.New(database, registry, matcher.WorkerConfig{
		ProviderName: heuristic.ProviderName,
		PollInterval: 100 * time.Millisecond,
		BatchSize:    1,
	})
	service := NewAppService(database, sched, matchWorker)

	if err := service.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatalf("ServiceStartup failed: %v", err)
	}

	if err := service.ServiceShutdown(); err != nil {
		t.Fatalf("ServiceShutdown failed: %v", err)
	}

	if _, err := database.CountJobs(context.Background()); err == nil {
		t.Fatal("expected closed DB to reject queries after shutdown")
	}
}

func TestAppServiceStartupRunsRetentionCleanup(t *testing.T) {
	database := openAppServiceTestDB(t)
	oldPostedAt := time.Now().UTC().AddDate(0, 0, -20)
	recentPostedAt := time.Now().UTC().AddDate(0, 0, -2)

	if _, err := database.InsertJob(context.Background(), &db.Job{
		Source:   "reed_uk",
		SourceID: "old-job",
		Title:    "Old Job",
		URL:      "https://example.com/old",
		PostedAt: &oldPostedAt,
	}); err != nil {
		t.Fatalf("inserting old job: %v", err)
	}
	if _, err := database.InsertJob(context.Background(), &db.Job{
		Source:   "reed_uk",
		SourceID: "recent-job",
		Title:    "Recent Job",
		URL:      "https://example.com/recent",
		PostedAt: &recentPostedAt,
	}); err != nil {
		t.Fatalf("inserting recent job: %v", err)
	}
	if err := database.SetSetting(context.Background(), settingJobRetentionDays, "7"); err != nil {
		t.Fatalf("setting retention days: %v", err)
	}

	sched := scheduler.New(database, adapter.NewRegistry(), time.Hour)
	registry := llm.NewRegistry()
	if err := registry.Register(heuristic.New()); err != nil {
		t.Fatalf("registering provider: %v", err)
	}
	matchWorker := matcher.New(database, registry, matcher.WorkerConfig{
		ProviderName: heuristic.ProviderName,
		PollInterval: 100 * time.Millisecond,
		BatchSize:    1,
	})
	service := NewAppService(database, sched, matchWorker)

	if err := service.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatalf("ServiceStartup failed: %v", err)
	}

	jobs, err := database.ListJobs(context.Background(), 0)
	if err != nil {
		t.Fatalf("listing jobs after startup cleanup: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job after startup cleanup, got %d", len(jobs))
	}
	if jobs[0].SourceID != "recent-job" {
		t.Fatalf("expected recent job to remain after cleanup, got %q", jobs[0].SourceID)
	}

	if err := service.ServiceShutdown(); err != nil {
		t.Fatalf("ServiceShutdown failed: %v", err)
	}
}

func TestAppServiceShutdownWithoutStartup(t *testing.T) {
	database := openAppServiceTestDB(t)
	sched := scheduler.New(database, adapter.NewRegistry(), time.Hour)
	service := NewAppService(database, sched, nil)

	if err := service.ServiceShutdown(); err != nil {
		t.Fatalf("ServiceShutdown failed: %v", err)
	}
}

func TestAppServiceShutdownAfterDatabaseAlreadyClosed(t *testing.T) {
	database := openAppServiceTestDB(t)
	sched := scheduler.New(database, adapter.NewRegistry(), time.Hour)
	service := NewAppService(database, sched, nil)

	if err := database.Close(); err != nil {
		t.Fatalf("closing DB before shutdown: %v", err)
	}

	if err := service.ServiceShutdown(); err != nil {
		t.Fatalf("expected ServiceShutdown to treat already-closed DB as no-op, got %v", err)
	}
}
