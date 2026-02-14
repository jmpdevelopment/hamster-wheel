package main

import (
	"context"
	"fmt"
	"time"

	"hamster-wheel/internal/adapter"
	"hamster-wheel/internal/db"
)

// retryFetchTimeout is the maximum time allowed for a retry description fetch.
const retryFetchTimeout = 30 * time.Second

// JobService handles job-related operations exposed to the frontend.
type JobService struct {
	db       *db.DB
	adapters *adapter.Registry
}

// NewJobService creates a new JobService.
func NewJobService(database *db.DB, adapters *adapter.Registry) *JobService {
	return &JobService{
		db:       database,
		adapters: adapters,
	}
}

// GetJobs returns jobs ordered by discovery date (newest first).
// Pass limit=0 for all jobs.
func (s *JobService) GetJobs(limit int) ([]db.Job, error) {
	return s.db.ListJobs(context.Background(), limit)
}

// GetJob returns a single job by ID, or nil if not found.
func (s *JobService) GetJob(id string) (*db.Job, error) {
	return s.db.GetJob(context.Background(), id)
}

// GetJobsByFilter returns jobs discovered through a specific filter.
func (s *JobService) GetJobsByFilter(filterID string) ([]db.Job, error) {
	return s.db.ListJobsByFilter(context.Background(), filterID)
}

// GetJobCount returns the total number of jobs in the database.
func (s *JobService) GetJobCount() (int, error) {
	return s.db.CountJobs(context.Background())
}

// DeleteJob removes a job by ID.
func (s *JobService) DeleteJob(id string) error {
	return s.db.DeleteJob(context.Background(), id)
}

// SetJobFavorite updates favorite state for one job.
func (s *JobService) SetJobFavorite(id string, favorite bool) error {
	return s.db.SetJobFavorite(context.Background(), id, favorite)
}

// SetJobsFavorite updates favorite state for multiple jobs.
func (s *JobService) SetJobsFavorite(ids []string, favorite bool) error {
	return s.db.SetJobsFavorite(context.Background(), ids, favorite)
}

// RetryFetchDescription re-fetches the full description for a job from the
// original source. Used when the initial detail fetch failed during polling.
func (s *JobService) RetryFetchDescription(jobID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), retryFetchTimeout)
	defer cancel()

	// Load the job from DB.
	job, err := s.db.GetJob(ctx, jobID)
	if err != nil {
		return fmt.Errorf("loading job: %w", err)
	}
	if job == nil {
		return db.ErrJobNotFound
	}

	// Find the adapter that originally discovered this job.
	a, ok := s.adapters.Get(job.Source)
	if !ok {
		return fmt.Errorf("no adapter registered for source %q", job.Source)
	}

	// Reconstruct a summary so the adapter can fetch details.
	summary := adapter.JobSummary{
		SourceID: job.SourceID,
		Title:    job.Title,
		Company:  job.Company,
		Location: job.Location,
		URL:      job.URL,
	}

	// Fetch full details from the source.
	details, err := a.FetchJobDetails(ctx, summary)
	if err != nil {
		return fmt.Errorf("fetching description: %w", err)
	}

	// Persist the description.
	return s.db.UpdateJobDescription(ctx, jobID, details.FullDescription)
}
