package db

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEnsureJobMatchPendingCreatesMatchRow(t *testing.T) {
	database := testDB(t)
	jobID, err := database.InsertJob(context.Background(), makeTestJob("match-pending-create"))
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}

	if err := database.EnsureJobMatchPending(context.Background(), jobID); err != nil {
		t.Fatalf("ensuring pending match: %v", err)
	}

	match, err := database.GetJobMatchByJobID(context.Background(), jobID)
	if err != nil {
		t.Fatalf("getting job match: %v", err)
	}
	if match == nil {
		t.Fatal("expected job match row, got nil")
	}
	if match.Status != JobMatchStatusPending {
		t.Fatalf("expected status %q, got %q", JobMatchStatusPending, match.Status)
	}
}

func TestEnsureJobMatchPendingIsIdempotent(t *testing.T) {
	database := testDB(t)
	jobID, err := database.InsertJob(context.Background(), makeTestJob("match-pending-idempotent"))
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}

	if err := database.EnsureJobMatchPending(context.Background(), jobID); err != nil {
		t.Fatalf("first ensure pending match: %v", err)
	}
	first, err := database.GetJobMatchByJobID(context.Background(), jobID)
	if err != nil {
		t.Fatalf("getting first match row: %v", err)
	}
	if first == nil {
		t.Fatal("expected first match row, got nil")
	}

	if err := database.EnsureJobMatchPending(context.Background(), jobID); err != nil {
		t.Fatalf("second ensure pending match: %v", err)
	}
	second, err := database.GetJobMatchByJobID(context.Background(), jobID)
	if err != nil {
		t.Fatalf("getting second match row: %v", err)
	}
	if second == nil {
		t.Fatal("expected second match row, got nil")
	}
	if second.ID != first.ID {
		t.Fatalf("expected same row ID after idempotent insert, first=%q second=%q", first.ID, second.ID)
	}
}

func TestEnsureJobMatchPendingRejectsEmptyJobID(t *testing.T) {
	database := testDB(t)

	err := database.EnsureJobMatchPending(context.Background(), "   ")
	if err == nil {
		t.Fatal("expected error for empty job ID")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetJobMatchByJobIDReturnsNilWhenMissing(t *testing.T) {
	database := testDB(t)

	match, err := database.GetJobMatchByJobID(context.Background(), "missing-job-id")
	if err != nil {
		t.Fatalf("getting missing job match: %v", err)
	}
	if match != nil {
		t.Fatalf("expected nil match for missing job ID, got %+v", match)
	}
}

func TestClaimNextPendingJobMatchTransitionsStatus(t *testing.T) {
	database := testDB(t)
	jobID, err := database.InsertJob(context.Background(), makeTestJob("claim-pending"))
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}
	if err := database.EnsureJobMatchPending(context.Background(), jobID); err != nil {
		t.Fatalf("ensuring pending match: %v", err)
	}

	claimed, err := database.ClaimNextPendingJobMatch(context.Background())
	if err != nil {
		t.Fatalf("claiming pending match: %v", err)
	}
	if claimed == nil {
		t.Fatal("expected claimed match, got nil")
	}
	if claimed.Status != JobMatchStatusProcessing {
		t.Fatalf("expected claimed status %q, got %q", JobMatchStatusProcessing, claimed.Status)
	}
	if claimed.JobID != jobID {
		t.Fatalf("expected claimed job %q, got %q", jobID, claimed.JobID)
	}

	stored, err := database.GetJobMatchByJobID(context.Background(), jobID)
	if err != nil {
		t.Fatalf("getting stored match: %v", err)
	}
	if stored == nil {
		t.Fatal("expected stored match row")
	}
	if stored.Status != JobMatchStatusProcessing {
		t.Fatalf("expected stored status %q, got %q", JobMatchStatusProcessing, stored.Status)
	}
}

func TestClaimNextPendingJobMatchReturnsNilWhenNoRows(t *testing.T) {
	database := testDB(t)

	claimed, err := database.ClaimNextPendingJobMatch(context.Background())
	if err != nil {
		t.Fatalf("claiming with no rows: %v", err)
	}
	if claimed != nil {
		t.Fatalf("expected nil claim when no rows exist, got %+v", claimed)
	}
}

func TestMarkJobMatchMatchedUpdatesScoreAndSummary(t *testing.T) {
	database := testDB(t)
	jobID, err := database.InsertJob(context.Background(), makeTestJob("mark-matched"))
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}
	if err := database.EnsureJobMatchPending(context.Background(), jobID); err != nil {
		t.Fatalf("ensuring pending match: %v", err)
	}

	if err := database.MarkJobMatchMatched(context.Background(), jobID, 0.82, "Strong overlap."); err != nil {
		t.Fatalf("marking matched: %v", err)
	}

	match, err := database.GetJobMatchByJobID(context.Background(), jobID)
	if err != nil {
		t.Fatalf("getting match: %v", err)
	}
	if match == nil {
		t.Fatal("expected match row")
	}
	if match.Status != JobMatchStatusMatched {
		t.Fatalf("expected status %q, got %q", JobMatchStatusMatched, match.Status)
	}
	if match.MatchScore != 0.82 {
		t.Fatalf("expected score 0.82, got %.2f", match.MatchScore)
	}
	if match.MatchSummary != "Strong overlap." {
		t.Fatalf("expected summary %q, got %q", "Strong overlap.", match.MatchSummary)
	}
}

func TestMarkJobMatchFailedUpdatesStatus(t *testing.T) {
	database := testDB(t)
	jobID, err := database.InsertJob(context.Background(), makeTestJob("mark-failed"))
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}
	if err := database.EnsureJobMatchPending(context.Background(), jobID); err != nil {
		t.Fatalf("ensuring pending match: %v", err)
	}

	if err := database.MarkJobMatchFailed(context.Background(), jobID, "Provider timeout"); err != nil {
		t.Fatalf("marking failed: %v", err)
	}

	match, err := database.GetJobMatchByJobID(context.Background(), jobID)
	if err != nil {
		t.Fatalf("getting match: %v", err)
	}
	if match == nil {
		t.Fatal("expected match row")
	}
	if match.Status != JobMatchStatusFailed {
		t.Fatalf("expected status %q, got %q", JobMatchStatusFailed, match.Status)
	}
	if match.MatchSummary != "Provider timeout" {
		t.Fatalf("expected summary %q, got %q", "Provider timeout", match.MatchSummary)
	}
}

func TestRequeueStaleProcessingJobMatchesMovesOldRows(t *testing.T) {
	database := testDB(t)
	jobID, err := database.InsertJob(context.Background(), makeTestJob("stale-processing"))
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}
	if err := database.EnsureJobMatchPending(context.Background(), jobID); err != nil {
		t.Fatalf("ensuring pending match: %v", err)
	}
	if _, err := database.ClaimNextPendingJobMatch(context.Background()); err != nil {
		t.Fatalf("claiming pending match: %v", err)
	}

	// Force status_updated_at into the past so it is considered stale.
	_, err = database.Conn().Exec(
		`UPDATE job_matches SET status_updated_at = datetime('now', '-10 minutes') WHERE job_id = ?`,
		jobID,
	)
	if err != nil {
		t.Fatalf("forcing stale status_updated_at: %v", err)
	}

	requeued, err := database.RequeueStaleProcessingJobMatches(
		context.Background(),
		time.Now().UTC().Add(-2*time.Minute),
	)
	if err != nil {
		t.Fatalf("requeueing stale rows: %v", err)
	}
	if requeued != 1 {
		t.Fatalf("expected 1 row requeued, got %d", requeued)
	}

	match, err := database.GetJobMatchByJobID(context.Background(), jobID)
	if err != nil {
		t.Fatalf("getting match: %v", err)
	}
	if match == nil {
		t.Fatal("expected match row")
	}
	if match.Status != JobMatchStatusPending {
		t.Fatalf("expected status %q after requeue, got %q", JobMatchStatusPending, match.Status)
	}
}
