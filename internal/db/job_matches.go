package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	JobMatchStatusPending    = "pending"
	JobMatchStatusProcessing = "processing"
	JobMatchStatusMatched    = "matched"
	JobMatchStatusFailed     = "failed"

	maxMatchSummaryLength = 500
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

// ResetJobMatchPending ensures a match row exists and sets it back to pending.
// Existing score/summary are cleared so UI does not show stale results.
func (db *DB) ResetJobMatchPending(ctx context.Context, jobID string) error {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return fmt.Errorf("job ID is required: %w", ErrInvalidInput)
	}

	var err error
	retryErr := withSQLiteBusyRetryCtx(ctx, func() error {
		tx, txErr := db.conn.BeginTx(ctx, nil)
		if txErr != nil {
			return fmt.Errorf("beginning reset pending transaction: %w", txErr)
		}
		defer func() {
			_ = tx.Rollback()
		}()

		// Upsert semantic: insert pending when missing.
		matchID := uuid.NewString()
		if _, txErr = tx.ExecContext(ctx,
			`INSERT INTO job_matches (id, job_id, status)
			 SELECT ?, ?, ?
			 WHERE NOT EXISTS (SELECT 1 FROM job_matches WHERE job_id = ?)`,
			matchID, jobID, JobMatchStatusPending, jobID,
		); txErr != nil {
			return fmt.Errorf("inserting missing job match row: %w", txErr)
		}

		// Always reset existing row back to pending and clear stale output.
		if _, txErr = tx.ExecContext(ctx,
			`UPDATE job_matches
			 SET status = ?, match_score = 0.0, match_summary = '', status_updated_at = datetime('now')
			 WHERE job_id = ?`,
			JobMatchStatusPending, jobID,
		); txErr != nil {
			return fmt.Errorf("resetting job match row to pending: %w", txErr)
		}

		if txErr = tx.Commit(); txErr != nil {
			return fmt.Errorf("committing reset pending transaction: %w", txErr)
		}
		return nil
	})
	if retryErr != nil {
		err = retryErr
	}
	if err != nil {
		return fmt.Errorf("resetting pending match for job %q: %w", jobID, err)
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

// ClaimNextPendingJobMatch atomically moves the oldest pending row to processing
// and returns it. Returns nil,nil when no pending rows exist.
func (db *DB) ClaimNextPendingJobMatch(ctx context.Context) (*JobMatch, error) {
	var claimed *JobMatch
	var err error

	retryErr := withSQLiteBusyRetryCtx(ctx, func() error {
		tx, txErr := db.conn.BeginTx(ctx, nil)
		if txErr != nil {
			return fmt.Errorf("beginning claim transaction: %w", txErr)
		}
		defer func() {
			_ = tx.Rollback()
		}()

		row := tx.QueryRowContext(ctx,
			`SELECT id, job_id, match_score, match_summary, status,
			        tailored_cv_path, tailored_cl_path, status_updated_at, created_at
			 FROM job_matches
			 WHERE status = ?
			 ORDER BY created_at ASC
			 LIMIT 1`,
			JobMatchStatusPending,
		)
		match, scanErr := scanJobMatch(row)
		if scanErr != nil {
			return fmt.Errorf("querying pending match row: %w", scanErr)
		}
		if match == nil {
			if commitErr := tx.Commit(); commitErr != nil {
				return fmt.Errorf("committing empty claim transaction: %w", commitErr)
			}
			claimed = nil
			return nil
		}

		result, updateErr := tx.ExecContext(ctx,
			`UPDATE job_matches
			 SET status = ?, status_updated_at = datetime('now')
			 WHERE id = ? AND status = ?`,
			JobMatchStatusProcessing,
			match.ID,
			JobMatchStatusPending,
		)
		if updateErr != nil {
			return fmt.Errorf("updating claimed match status: %w", updateErr)
		}
		rowsAffected, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			return fmt.Errorf("reading claim update rows affected: %w", rowsErr)
		}
		if rowsAffected == 0 {
			// Another worker claimed it first. Treat as no work for this loop.
			if commitErr := tx.Commit(); commitErr != nil {
				return fmt.Errorf("committing raced claim transaction: %w", commitErr)
			}
			claimed = nil
			return nil
		}

		match.Status = JobMatchStatusProcessing
		match.StatusUpdatedAt = time.Now().UTC()

		if commitErr := tx.Commit(); commitErr != nil {
			return fmt.Errorf("committing claim transaction: %w", commitErr)
		}

		claimed = match
		return nil
	})
	if retryErr != nil {
		err = retryErr
	}
	if err != nil {
		return nil, fmt.Errorf("claiming next pending job match: %w", err)
	}

	return claimed, nil
}

// MarkJobMatchMatched stores a completed match score and summary.
func (db *DB) MarkJobMatchMatched(ctx context.Context, jobID string, score float64, summary string) error {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return fmt.Errorf("job ID is required: %w", ErrInvalidInput)
	}
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return fmt.Errorf("score must be finite: %w", ErrInvalidInput)
	}
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	summary = normalizeMatchSummary(summary)

	var (
		result sql.Result
		err    error
	)
	retryErr := withSQLiteBusyRetryCtx(ctx, func() error {
		result, err = db.conn.ExecContext(ctx,
			`UPDATE job_matches
			 SET status = ?, match_score = ?, match_summary = ?, status_updated_at = datetime('now')
			 WHERE job_id = ?`,
			JobMatchStatusMatched,
			score,
			summary,
			jobID,
		)
		return err
	})
	if retryErr != nil {
		err = retryErr
	}
	if err != nil {
		return fmt.Errorf("marking matched status for job %q: %w", jobID, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking matched update rows: %w", err)
	}
	if rows == 0 {
		return ErrJobMatchNotFound
	}

	return nil
}

// MarkJobMatchFailed stores a failure summary for a match attempt.
func (db *DB) MarkJobMatchFailed(ctx context.Context, jobID string, reason string) error {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return fmt.Errorf("job ID is required: %w", ErrInvalidInput)
	}

	reason = normalizeMatchSummary(reason)
	if reason == "" {
		reason = "Matching failed."
	}

	var (
		result sql.Result
		err    error
	)
	retryErr := withSQLiteBusyRetryCtx(ctx, func() error {
		result, err = db.conn.ExecContext(ctx,
			`UPDATE job_matches
			 SET status = ?, match_summary = ?, status_updated_at = datetime('now')
			 WHERE job_id = ?`,
			JobMatchStatusFailed,
			reason,
			jobID,
		)
		return err
	})
	if retryErr != nil {
		err = retryErr
	}
	if err != nil {
		return fmt.Errorf("marking failed status for job %q: %w", jobID, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking failed update rows: %w", err)
	}
	if rows == 0 {
		return ErrJobMatchNotFound
	}

	return nil
}

// RequeueStaleProcessingJobMatches moves old processing rows back to pending
// so they can be retried after crashes or stuck provider calls.
func (db *DB) RequeueStaleProcessingJobMatches(ctx context.Context, staleBefore time.Time) (int, error) {
	if staleBefore.IsZero() {
		return 0, fmt.Errorf("staleBefore is required: %w", ErrInvalidInput)
	}
	staleBeforeUTC := staleBefore.UTC().Format("2006-01-02 15:04:05")

	var (
		result sql.Result
		err    error
	)
	retryErr := withSQLiteBusyRetryCtx(ctx, func() error {
		result, err = db.conn.ExecContext(ctx,
			`UPDATE job_matches
			 SET status = ?, status_updated_at = datetime('now')
			 WHERE status = ? AND status_updated_at < ?`,
			JobMatchStatusPending,
			JobMatchStatusProcessing,
			staleBeforeUTC,
		)
		return err
	})
	if retryErr != nil {
		err = retryErr
	}
	if err != nil {
		return 0, fmt.Errorf("requeueing stale processing matches: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("reading requeue rows affected: %w", err)
	}
	return int(rows), nil
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

func normalizeMatchSummary(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxMatchSummaryLength {
		return value
	}
	return string(runes[:maxMatchSummaryLength])
}
