package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	JobMatchStatusPending    = "pending"
	JobMatchStatusProcessing = "processing"
	JobMatchStatusMatched    = "matched"
	JobMatchStatusFailed     = "failed"
)

// JobMatch stores matching metadata for one job.
type JobMatch struct {
	ID              string
	JobID           string
	MatchScore      float64
	MatchSummary    string
	Status          string
	TailoredCVPath  *string
	TailoredCLPath  *string
	StatusUpdatedAt time.Time
	CreatedAt       time.Time
}

// EnsureJobMatchPending creates a pending match record for a job if none exists.
// It is idempotent: existing rows for the job are left unchanged.
func (db *DB) EnsureJobMatchPending(ctx context.Context, jobID string) error {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return fmt.Errorf("job ID is required: %w", ErrInvalidInput)
	}

	matchID := uuid.NewString()
	var err error
	retryErr := withSQLiteBusyRetryCtx(ctx, func() error {
		_, err = db.conn.ExecContext(ctx,
			`INSERT INTO job_matches (id, job_id, status)
			 SELECT ?, ?, ?
			 WHERE NOT EXISTS (
			   SELECT 1 FROM job_matches WHERE job_id = ?
			 )`,
			matchID, jobID, JobMatchStatusPending, jobID,
		)
		return err
	})
	if retryErr != nil {
		err = retryErr
	}
	if err != nil {
		return fmt.Errorf("ensuring pending match for job %q: %w", jobID, err)
	}
	return nil
}

// GetJobMatchByJobID returns the match row for one job.
// Returns nil and no error when the job has no match row yet.
func (db *DB) GetJobMatchByJobID(ctx context.Context, jobID string) (*JobMatch, error) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return nil, fmt.Errorf("job ID is required: %w", ErrInvalidInput)
	}

	row := db.conn.QueryRowContext(ctx,
		`SELECT id, job_id, match_score, match_summary, status,
		        tailored_cv_path, tailored_cl_path, status_updated_at, created_at
		 FROM job_matches
		 WHERE job_id = ?`,
		jobID,
	)
	return scanJobMatch(row)
}

func scanJobMatch(row *sql.Row) (*JobMatch, error) {
	var jm JobMatch
	var tailoredCVPath sql.NullString
	var tailoredCLPath sql.NullString
	var statusUpdatedAt sql.NullString
	var createdAt sql.NullString

	err := row.Scan(
		&jm.ID,
		&jm.JobID,
		&jm.MatchScore,
		&jm.MatchSummary,
		&jm.Status,
		&tailoredCVPath,
		&tailoredCLPath,
		&statusUpdatedAt,
		&createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scanning job match row: %w", err)
	}

	if tailoredCVPath.Valid {
		jm.TailoredCVPath = &tailoredCVPath.String
	}
	if tailoredCLPath.Valid {
		jm.TailoredCLPath = &tailoredCLPath.String
	}
	if statusUpdatedAt.Valid {
		var parseErr error
		jm.StatusUpdatedAt, parseErr = time.Parse("2006-01-02 15:04:05", statusUpdatedAt.String)
		if parseErr != nil {
			slog.Warn("failed to parse job match status_updated_at",
				"job_id", jm.JobID,
				"value", statusUpdatedAt.String,
				"error", parseErr,
			)
		}
	}
	if createdAt.Valid {
		var parseErr error
		jm.CreatedAt, parseErr = time.Parse("2006-01-02 15:04:05", createdAt.String)
		if parseErr != nil {
			slog.Warn("failed to parse job match created_at",
				"job_id", jm.JobID,
				"value", createdAt.String,
				"error", parseErr,
			)
		}
	}

	return &jm, nil
}
