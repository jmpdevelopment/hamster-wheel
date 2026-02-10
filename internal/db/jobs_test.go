package db

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// makeTestJob returns a Job with sensible defaults for testing.
// Override fields as needed after calling.
func makeTestJob(sourceID string) *Job {
	posted := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	return &Job{
		Source:      "indeed_uk",
		SourceID:    sourceID,
		Title:       "Go Developer",
		Company:     "Acme Corp",
		Location:    "London",
		Description: "We are looking for a Go developer...",
		URL:         "https://indeed.co.uk/job/" + sourceID,
		PostedAt:    &posted,
	}
}

func TestInsertJobReturnsID(t *testing.T) {
	db := testDB(t)

	job := makeTestJob("abc123")
	id, err := db.InsertJob(context.Background(), job)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty ID")
	}
}

func TestInsertJobGeneratesUUID(t *testing.T) {
	db := testDB(t)

	job1 := makeTestJob("job-1")
	id1, err := db.InsertJob(context.Background(), job1)
	if err != nil {
		t.Fatalf("inserting job 1: %v", err)
	}

	job2 := makeTestJob("job-2")
	id2, err := db.InsertJob(context.Background(), job2)
	if err != nil {
		t.Fatalf("inserting job 2: %v", err)
	}

	if id1 == id2 {
		t.Error("expected different IDs for different jobs")
	}
}

func TestInsertJobPreservesExplicitID(t *testing.T) {
	db := testDB(t)

	job := makeTestJob("explicit-id-test")
	job.ID = "my-custom-id"

	id, err := db.InsertJob(context.Background(), job)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "my-custom-id" {
		t.Errorf("expected %q, got %q", "my-custom-id", id)
	}
}

func TestInsertJobDuplicateSourceIDReturnsError(t *testing.T) {
	db := testDB(t)

	job1 := makeTestJob("same-source-id")
	_, err := db.InsertJob(context.Background(), job1)
	if err != nil {
		t.Fatalf("inserting first job: %v", err)
	}

	job2 := makeTestJob("same-source-id")
	job2.Title = "Different Title"
	_, err = db.InsertJob(context.Background(), job2)
	if !errors.Is(err, ErrDuplicateJob) {
		t.Errorf("expected ErrDuplicateJob, got: %v", err)
	}
}

func TestInsertJobSameSourceIDDifferentSourceAllowed(t *testing.T) {
	db := testDB(t)

	job1 := makeTestJob("shared-id")
	job1.Source = "indeed_uk"
	_, err := db.InsertJob(context.Background(), job1)
	if err != nil {
		t.Fatalf("inserting indeed job: %v", err)
	}

	job2 := makeTestJob("shared-id")
	job2.Source = "glassdoor_uk"
	_, err = db.InsertJob(context.Background(), job2)
	if err != nil {
		t.Fatalf("expected no error for different source, got: %v", err)
	}
}

func TestGetJobReturnsInsertedJob(t *testing.T) {
	db := testDB(t)

	job := makeTestJob("get-test")
	job.Company = "Test Corp"
	job.Location = "Manchester"
	job.Description = "A great job description"

	id, err := db.InsertJob(context.Background(), job)
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}

	got, err := db.GetJob(context.Background(), id)
	if err != nil {
		t.Fatalf("getting job: %v", err)
	}
	if got == nil {
		t.Fatal("expected job, got nil")
	}

	if got.ID != id {
		t.Errorf("ID: got %q, want %q", got.ID, id)
	}
	if got.Source != "indeed_uk" {
		t.Errorf("Source: got %q, want %q", got.Source, "indeed_uk")
	}
	if got.SourceID != "get-test" {
		t.Errorf("SourceID: got %q, want %q", got.SourceID, "get-test")
	}
	if got.Title != "Go Developer" {
		t.Errorf("Title: got %q, want %q", got.Title, "Go Developer")
	}
	if got.Company != "Test Corp" {
		t.Errorf("Company: got %q, want %q", got.Company, "Test Corp")
	}
	if got.Location != "Manchester" {
		t.Errorf("Location: got %q, want %q", got.Location, "Manchester")
	}
	if got.Description != "A great job description" {
		t.Errorf("Description: got %q, want %q", got.Description, "A great job description")
	}
	if got.PostedAt == nil {
		t.Error("expected PostedAt to be set")
	}
	if got.DiscoveredAt.IsZero() {
		t.Error("expected DiscoveredAt to be set automatically")
	}
}

func TestGetJobReturnsNilForMissing(t *testing.T) {
	db := testDB(t)

	got, err := db.GetJob(context.Background(), "nonexistent-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestInsertJobWithNilPostedAt(t *testing.T) {
	db := testDB(t)

	job := makeTestJob("nil-posted")
	job.PostedAt = nil

	id, err := db.InsertJob(context.Background(), job)
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}

	got, err := db.GetJob(context.Background(), id)
	if err != nil {
		t.Fatalf("getting job: %v", err)
	}
	if got.PostedAt != nil {
		t.Errorf("expected nil PostedAt, got %v", got.PostedAt)
	}
}

func TestInsertJobWithFilterID(t *testing.T) {
	db := testDB(t)

	// Create a filter first.
	filterID, err := db.CreateFilter(context.Background(),"Test", "golang", "London", "indeed_uk")
	if err != nil {
		t.Fatalf("creating filter: %v", err)
	}

	job := makeTestJob("with-filter")
	job.FilterID = &filterID

	id, err := db.InsertJob(context.Background(), job)
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}

	got, err := db.GetJob(context.Background(), id)
	if err != nil {
		t.Fatalf("getting job: %v", err)
	}
	if got.FilterID == nil {
		t.Fatal("expected FilterID to be set")
	}
	if *got.FilterID != filterID {
		t.Errorf("FilterID: got %q, want %q", *got.FilterID, filterID)
	}
}

func TestJobExistsBySourceIDReturnsTrueForExisting(t *testing.T) {
	db := testDB(t)

	job := makeTestJob("exists-check")
	_, err := db.InsertJob(context.Background(), job)
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}

	exists, err := db.JobExistsBySourceID(context.Background(),"indeed_uk", "exists-check")
	if err != nil {
		t.Fatalf("checking existence: %v", err)
	}
	if !exists {
		t.Error("expected job to exist")
	}
}

func TestJobExistsBySourceIDReturnsFalseForMissing(t *testing.T) {
	db := testDB(t)

	exists, err := db.JobExistsBySourceID(context.Background(),"indeed_uk", "does-not-exist")
	if err != nil {
		t.Fatalf("checking existence: %v", err)
	}
	if exists {
		t.Error("expected job to not exist")
	}
}

func TestJobExistsBySourceIDDistinguishesSources(t *testing.T) {
	db := testDB(t)

	job := makeTestJob("source-specific")
	job.Source = "indeed_uk"
	_, err := db.InsertJob(context.Background(), job)
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}

	// Same source_id but different source should not exist.
	exists, err := db.JobExistsBySourceID(context.Background(),"glassdoor_uk", "source-specific")
	if err != nil {
		t.Fatalf("checking existence: %v", err)
	}
	if exists {
		t.Error("expected job to not exist for different source")
	}
}

func TestListJobsReturnsAll(t *testing.T) {
	db := testDB(t)

	for i := 0; i < 3; i++ {
		job := makeTestJob(fmt.Sprintf("list-job-%d", i))
		_, err := db.InsertJob(context.Background(), job)
		if err != nil {
			t.Fatalf("inserting job %d: %v", i, err)
		}
	}

	jobs, err := db.ListJobs(context.Background(),0)
	if err != nil {
		t.Fatalf("listing jobs: %v", err)
	}
	if len(jobs) != 3 {
		t.Errorf("expected 3 jobs, got %d", len(jobs))
	}
}

func TestListJobsRespectsLimit(t *testing.T) {
	db := testDB(t)

	for i := 0; i < 5; i++ {
		job := makeTestJob(fmt.Sprintf("limit-job-%d", i))
		_, err := db.InsertJob(context.Background(), job)
		if err != nil {
			t.Fatalf("inserting job %d: %v", i, err)
		}
	}

	jobs, err := db.ListJobs(context.Background(),2)
	if err != nil {
		t.Fatalf("listing jobs: %v", err)
	}
	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(jobs))
	}
}

func TestListJobsReturnsEmptySlice(t *testing.T) {
	db := testDB(t)

	jobs, err := db.ListJobs(context.Background(),0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jobs == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}
}

func TestListJobsRejectsNegativeLimit(t *testing.T) {
	db := testDB(t)

	jobs, err := db.ListJobs(context.Background(),-1)
	if err == nil {
		t.Fatal("expected error for negative limit, got nil")
	}
	if !errors.Is(err, ErrInvalidLimit) {
		t.Errorf("expected ErrInvalidLimit, got: %v", err)
	}
	if jobs != nil {
		t.Errorf("expected nil jobs on error, got: %v", jobs)
	}
}

func TestListJobsRejectsExcessiveLimit(t *testing.T) {
	db := testDB(t)

	jobs, err := db.ListJobs(context.Background(),maxJobsLimit + 1)
	if err == nil {
		t.Fatal("expected error for excessive limit, got nil")
	}
	if !errors.Is(err, ErrInvalidLimit) {
		t.Errorf("expected ErrInvalidLimit, got: %v", err)
	}
	if jobs != nil {
		t.Errorf("expected nil jobs on error, got: %v", jobs)
	}
}

func TestListJobsByFilterReturnsMatchingJobs(t *testing.T) {
	db := testDB(t)

	filterA, _ := db.CreateFilter(context.Background(),"Filter A", "go", "London", "indeed_uk")
	filterB, _ := db.CreateFilter(context.Background(),"Filter B", "python", "Remote", "indeed_uk")

	jobA := makeTestJob("filter-a-job")
	jobA.FilterID = &filterA
	_, err := db.InsertJob(context.Background(), jobA)
	if err != nil {
		t.Fatalf("inserting job A: %v", err)
	}

	jobB := makeTestJob("filter-b-job")
	jobB.FilterID = &filterB
	_, err = db.InsertJob(context.Background(), jobB)
	if err != nil {
		t.Fatalf("inserting job B: %v", err)
	}

	jobs, err := db.ListJobsByFilter(context.Background(),filterA)
	if err != nil {
		t.Fatalf("listing by filter: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].SourceID != "filter-a-job" {
		t.Errorf("expected filter-a-job, got %q", jobs[0].SourceID)
	}
}

func TestDeleteJobRemovesIt(t *testing.T) {
	db := testDB(t)

	job := makeTestJob("to-delete")
	id, err := db.InsertJob(context.Background(), job)
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}

	err = db.DeleteJob(context.Background(),id)
	if err != nil {
		t.Fatalf("deleting job: %v", err)
	}

	got, err := db.GetJob(context.Background(), id)
	if err != nil {
		t.Fatalf("getting deleted job: %v", err)
	}
	if got != nil {
		t.Error("expected nil after deletion")
	}
}

func TestDeleteJobNotFoundReturnsError(t *testing.T) {
	db := testDB(t)

	err := db.DeleteJob(context.Background(),"nonexistent-id")
	if !errors.Is(err, ErrJobNotFound) {
		t.Errorf("expected ErrJobNotFound, got: %v", err)
	}
}

func TestCountJobs(t *testing.T) {
	db := testDB(t)

	count, err := db.CountJobs(context.Background())
	if err != nil {
		t.Fatalf("counting empty: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	for i := 0; i < 3; i++ {
		job := makeTestJob(fmt.Sprintf("count-job-%d", i))
		_, err := db.InsertJob(context.Background(), job)
		if err != nil {
			t.Fatalf("inserting job %d: %v", i, err)
		}
	}

	count, err = db.CountJobs(context.Background())
	if err != nil {
		t.Fatalf("counting after insert: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

func TestInsertJobWithEmptyOptionalFields(t *testing.T) {
	db := testDB(t)

	job := &Job{
		Source:   "indeed_uk",
		SourceID: "minimal-job",
		Title:    "Minimal Job",
		URL:      "https://example.com/job",
		// Company, Location, Description all empty
		// PostedAt nil, FilterID nil
	}

	id, err := db.InsertJob(context.Background(), job)
	if err != nil {
		t.Fatalf("inserting minimal job: %v", err)
	}

	got, err := db.GetJob(context.Background(), id)
	if err != nil {
		t.Fatalf("getting job: %v", err)
	}
	if got.Company != "" {
		t.Errorf("expected empty company, got %q", got.Company)
	}
	if got.Location != "" {
		t.Errorf("expected empty location, got %q", got.Location)
	}
	if got.FilterID != nil {
		t.Errorf("expected nil FilterID, got %v", got.FilterID)
	}
}
