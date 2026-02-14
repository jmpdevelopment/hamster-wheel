package db

import (
	"context"
	"errors"
	"testing"
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
