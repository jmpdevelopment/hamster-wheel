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

// Job represents a discovered job posting stored in the database.
// Jobs are found by adapters during polling and deduplicated via the
// UNIQUE(source, source_id) constraint.
type Job struct {
	ID           string     // UUID primary key
	Source       string     // Which adapter found this, e.g. "reed_uk"
	SourceID     string     // External ID / URL hash for deduplication
	Title        string     // Job title
	Company      string     // Company name
	Location     string     // Job location
	Description  string     // Full job description text
	URL          string     // Link to original posting
	PostedAt     *time.Time // When the job was posted (from source), nullable
	DiscoveredAt time.Time  // When Hamster Wheel found this job
	FilterID     *string    // Which search filter discovered this, nullable
	IsFavorite   bool       // Whether user marked this job as favorite
	MatchStatus  string     // Current matching status (pending/processing/matched/failed), empty if unavailable
	MatchScore   float64    // Latest match score (0.0-1.0)
	MatchSummary string     // Latest match summary
}

// InsertJob stores a new job. Returns ErrDuplicateJob if a job with the same
// (source, source_id) already exists — this is the primary deduplication mechanism.
func (db *DB) InsertJob(ctx context.Context, job *Job) (string, error) {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}

	var postedAt *string
	if job.PostedAt != nil {
		s := job.PostedAt.UTC().Format("2006-01-02 15:04:05")
		postedAt = &s
	}

	var err error
	retryErr := withSQLiteBusyRetryCtx(ctx, func() error {
		_, err = db.conn.ExecContext(ctx,
			`INSERT INTO jobs (id, source, source_id, title, company, location, description, url, posted_at, filter_id)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			job.ID, job.Source, job.SourceID, job.Title, job.Company,
			job.Location, job.Description, job.URL, postedAt, job.FilterID,
		)
		return err
	})
	if retryErr != nil {
		err = retryErr
	}
	if err != nil {
		// SQLite returns "UNIQUE constraint failed" for duplicate (source, source_id).
		if isUniqueViolation(err) {
			return "", ErrDuplicateJob
		}
		return "", fmt.Errorf("inserting job %q: %w", job.Title, err)
	}

	return job.ID, nil
}

// GetJob retrieves a single job by ID.
// Returns nil and no error if the job doesn't exist.
func (db *DB) GetJob(ctx context.Context, id string) (*Job, error) {
	row := db.conn.QueryRowContext(ctx,
		`SELECT j.id, j.source, j.source_id, j.title, j.company, j.location, j.description,
		        j.url, j.posted_at, j.discovered_at, j.filter_id, j.is_favorite,
		        COALESCE(m.status, ''), COALESCE(m.match_score, 0.0), COALESCE(m.match_summary, '')
		 FROM jobs j
		 LEFT JOIN job_matches m ON m.job_id = j.id
		 WHERE j.id = ?`, id,
	)

	return scanJob(row)
}

// JobExistsBySourceID checks whether a job with the given source + source_id
// already exists in the database. The scheduler calls this before fetching
// full details to avoid redundant HTTP requests.
func (db *DB) JobExistsBySourceID(ctx context.Context, source, sourceID string) (bool, error) {
	var count int
	err := db.conn.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM jobs WHERE source = ? AND source_id = ?",
		source, sourceID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking job existence (%s/%s): %w", source, sourceID, err)
	}
	return count > 0, nil
}

// Maximum allowed limit for ListJobs to prevent performance issues.
const maxJobsLimit = 10000

// ListJobs returns all jobs, ordered by discovered_at (newest first).
// Pass limit=0 for no limit. Returns ErrInvalidLimit if limit is negative or exceeds maxJobsLimit.
func (db *DB) ListJobs(ctx context.Context, limit int) ([]Job, error) {
	if limit < 0 {
		return nil, fmt.Errorf("limit must be non-negative: %w", ErrInvalidLimit)
	}
	if limit > maxJobsLimit {
		return nil, fmt.Errorf("limit %d exceeds maximum %d: %w", limit, maxJobsLimit, ErrInvalidLimit)
	}

	query := `SELECT j.id, j.source, j.source_id, j.title, j.company, j.location, j.description,
	                 j.url, j.posted_at, j.discovered_at, j.filter_id, j.is_favorite,
	                 COALESCE(m.status, ''), COALESCE(m.match_score, 0.0), COALESCE(m.match_summary, '')
	          FROM jobs j
	          LEFT JOIN job_matches m ON m.job_id = j.id
	          ORDER BY j.discovered_at DESC`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("listing jobs: %w", err)
	}
	defer rows.Close()

	return collectJobs(rows)
}

// ListJobsByFilter returns jobs found through a specific search filter.
func (db *DB) ListJobsByFilter(ctx context.Context, filterID string) ([]Job, error) {
	rows, err := db.conn.QueryContext(ctx,
		`SELECT j.id, j.source, j.source_id, j.title, j.company, j.location, j.description,
		        j.url, j.posted_at, j.discovered_at, j.filter_id, j.is_favorite,
		        COALESCE(m.status, ''), COALESCE(m.match_score, 0.0), COALESCE(m.match_summary, '')
		 FROM jobs j
		 LEFT JOIN job_matches m ON m.job_id = j.id
		 WHERE j.filter_id = ? ORDER BY discovered_at DESC`,
		filterID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing jobs by filter %q: %w", filterID, err)
	}
	defer rows.Close()

	return collectJobs(rows)
}

// DeleteJob removes a job by ID. Due to ON DELETE CASCADE, this also
// removes any associated job_matches rows.
// Returns ErrJobNotFound if the job doesn't exist.
func (db *DB) DeleteJob(ctx context.Context, id string) error {
	var (
		result sql.Result
		err    error
	)
	retryErr := withSQLiteBusyRetryCtx(ctx, func() error {
		result, err = db.conn.ExecContext(ctx, "DELETE FROM jobs WHERE id = ?", id)
		return err
	})
	if retryErr != nil {
		err = retryErr
	}
	if err != nil {
		return fmt.Errorf("deleting job %q: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking delete result: %w", err)
	}
	if rows == 0 {
		return ErrJobNotFound
	}

	return nil
}

// SetJobFavorite updates the favorite status for a single job.
// Returns ErrJobNotFound if the job does not exist.
func (db *DB) SetJobFavorite(ctx context.Context, id string, favorite bool) error {
	var result sql.Result
	var err error
	retryErr := withSQLiteBusyRetryCtx(ctx, func() error {
		result, err = db.conn.ExecContext(ctx,
			"UPDATE jobs SET is_favorite = ? WHERE id = ?",
			boolToSQLiteInt(favorite), id,
		)
		return err
	})
	if retryErr != nil {
		err = retryErr
	}
	if err != nil {
		return fmt.Errorf("updating favorite for job %q: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking favorite update result: %w", err)
	}
	if rows == 0 {
		return ErrJobNotFound
	}

	return nil
}

// SetJobsFavorite updates favorite status for multiple jobs in one transaction.
// Missing IDs are ignored so stale UI selections do not fail the entire action.
func (db *DB) SetJobsFavorite(ctx context.Context, ids []string, favorite bool) error {
	retryErr := withSQLiteBusyRetryCtx(ctx, func() error {
		tx, err := db.conn.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("beginning favorites update transaction: %w", err)
		}
		defer func() {
			_ = tx.Rollback()
		}()

		stmt, err := tx.PrepareContext(ctx, "UPDATE jobs SET is_favorite = ? WHERE id = ?")
		if err != nil {
			return fmt.Errorf("preparing favorites update: %w", err)
		}
		defer stmt.Close()

		for _, id := range ids {
			trimmedID := strings.TrimSpace(id)
			if trimmedID == "" {
				continue
			}
			if _, err := stmt.ExecContext(ctx, boolToSQLiteInt(favorite), trimmedID); err != nil {
				return fmt.Errorf("updating favorite for job %q: %w", trimmedID, err)
			}
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("committing favorites update transaction: %w", err)
		}
		return nil
	})
	if retryErr != nil {
		return retryErr
	}
	return nil
}

// UpdateJobDescription updates the description of a job by ID.
// Returns ErrJobNotFound if the job doesn't exist.
func (db *DB) UpdateJobDescription(ctx context.Context, id, description string) error {
	var (
		result sql.Result
		err    error
	)
	retryErr := withSQLiteBusyRetryCtx(ctx, func() error {
		result, err = db.conn.ExecContext(ctx,
			"UPDATE jobs SET description = ? WHERE id = ?",
			description, id,
		)
		return err
	})
	if retryErr != nil {
		err = retryErr
	}
	if err != nil {
		return fmt.Errorf("updating job description %q: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking update result: %w", err)
	}
	if rows == 0 {
		return ErrJobNotFound
	}

	return nil
}

// CountJobs returns the total number of jobs in the database.
func (db *DB) CountJobs(ctx context.Context) (int, error) {
	var count int
	err := db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM jobs").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting jobs: %w", err)
	}
	return count, nil
}

// scanJob scans a single row into a Job.
// Returns nil (not an error) if no row was found.
func scanJob(row *sql.Row) (*Job, error) {
	var j Job
	var postedAt, discoveredAt sql.NullString
	var filterID sql.NullString
	var isFavorite int

	err := row.Scan(
		&j.ID, &j.Source, &j.SourceID, &j.Title, &j.Company,
		&j.Location, &j.Description, &j.URL, &postedAt, &discoveredAt,
		&filterID, &isFavorite, &j.MatchStatus, &j.MatchScore, &j.MatchSummary,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scanning job: %w", err)
	}

	if postedAt.Valid {
		t, parseErr := time.Parse("2006-01-02 15:04:05", postedAt.String)
		if parseErr != nil {
			slog.Warn("failed to parse job posted_at", "job_id", j.ID, "value", postedAt.String, "error", parseErr)
		}
		j.PostedAt = &t
	}
	if discoveredAt.Valid {
		var parseErr error
		j.DiscoveredAt, parseErr = time.Parse("2006-01-02 15:04:05", discoveredAt.String)
		if parseErr != nil {
			slog.Warn("failed to parse job discovered_at", "job_id", j.ID, "value", discoveredAt.String, "error", parseErr)
		}
	}
	if filterID.Valid {
		j.FilterID = &filterID.String
	}
	j.IsFavorite = isFavorite == 1

	return &j, nil
}

// collectJobs scans all rows into a slice of Job.
func collectJobs(rows *sql.Rows) ([]Job, error) {
	jobs := make([]Job, 0)

	for rows.Next() {
		var j Job
		var postedAt, discoveredAt sql.NullString
		var filterID sql.NullString
		var isFavorite int

		err := rows.Scan(
			&j.ID, &j.Source, &j.SourceID, &j.Title, &j.Company,
			&j.Location, &j.Description, &j.URL, &postedAt, &discoveredAt,
			&filterID, &isFavorite, &j.MatchStatus, &j.MatchScore, &j.MatchSummary,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning job row: %w", err)
		}

		if postedAt.Valid {
			t, parseErr := time.Parse("2006-01-02 15:04:05", postedAt.String)
			if parseErr != nil {
				slog.Warn("failed to parse job posted_at", "job_id", j.ID, "value", postedAt.String, "error", parseErr)
			}
			j.PostedAt = &t
		}
		if discoveredAt.Valid {
			var parseErr error
			j.DiscoveredAt, parseErr = time.Parse("2006-01-02 15:04:05", discoveredAt.String)
			if parseErr != nil {
				slog.Warn("failed to parse job discovered_at", "job_id", j.ID, "value", discoveredAt.String, "error", parseErr)
			}
		}
		if filterID.Valid {
			j.FilterID = &filterID.String
		}
		j.IsFavorite = isFavorite == 1

		jobs = append(jobs, j)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating job rows: %w", err)
	}

	return jobs, nil
}

func boolToSQLiteInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// isUniqueViolation checks if an error is a SQLite UNIQUE constraint violation.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// modernc.org/sqlite wraps constraint errors with this message.
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
