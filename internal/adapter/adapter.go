// Package adapter defines the interface for job source integrations.
// Each job board (Indeed, Glassdoor, etc.) implements the Adapter interface,
// allowing the core polling logic to work identically across all sources.
package adapter

import (
	"context"
	"time"
)

// JobSummary represents a job listing as returned by a search/feed.
// This is the lightweight version — just enough to identify and deduplicate.
type JobSummary struct {
	SourceID string    // Unique ID within this source (for deduplication)
	Title    string    // Job title
	Company  string    // Company name
	Location string    // Job location
	URL      string    // Link to the original posting
	Snippet  string    // Short description / preview text
	PostedAt time.Time // When the job was posted (from the source)
}

// JobDetails extends JobSummary with the full description fetched from
// the job posting page.
type JobDetails struct {
	JobSummary
	FullDescription string // Complete job description text
	Salary          string // Salary info, if available
	JobType         string // Full-time, Part-time, Contract, etc.
}

// SearchParams contains the user's search criteria passed to FetchNewJobs.
type SearchParams struct {
	Keywords string
	Location string
}

// Adapter is the interface that all job source integrations must implement.
// To add a new job source:
//  1. Create a new package under internal/adapter/ (e.g., internal/adapter/glassdoor/)
//  2. Implement this interface
//  3. Register it with the Registry at app startup
type Adapter interface {
	// Name returns the unique identifier for this adapter (e.g., "reed_uk").
	// Used as the key in the registry and stored in the database.
	Name() string

	// DisplayName returns the human-readable name (e.g., "Reed UK").
	// Shown in the UI for users to select a job source.
	DisplayName() string

	// FetchNewJobs retrieves job listings matching the given search parameters.
	// The adapter makes HTTP requests and parses responses.
	// Deduplication is handled by the caller — the adapter returns all results.
	FetchNewJobs(ctx context.Context, params SearchParams) ([]JobSummary, error)

	// FetchJobDetails retrieves the full description for a specific job.
	// Called after FetchNewJobs to get the complete posting text.
	FetchJobDetails(ctx context.Context, job JobSummary) (*JobDetails, error)

	// Validate checks if the adapter is functional (e.g., can reach the site).
	Validate(ctx context.Context) error
}
