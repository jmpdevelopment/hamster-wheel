package main

import (
	"context"

	"hamster-wheel/internal/db"
)

// JobService handles job-related operations exposed to the frontend.
type JobService struct {
	db *db.DB
}

// NewJobService creates a new JobService.
func NewJobService(database *db.DB) *JobService {
	return &JobService{
		db: database,
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