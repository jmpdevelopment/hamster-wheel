package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SearchFilter represents a saved search configuration that the scheduler
// uses to poll a job source. Each filter maps to one adapter (e.g., "reed_uk")
// and contains the keywords + location the user wants to monitor.
type SearchFilter struct {
	ID        string    // UUID primary key
	Name      string    // User-given name, e.g. "Backend London"
	Keywords  string    // Search keywords
	Location  string    // Location string
	Source    string    // Adapter identifier, e.g. "reed_uk"
	Enabled   bool      // Whether the scheduler should poll this filter
	CreatedAt time.Time // When the filter was created
	UpdatedAt time.Time // When the filter was last modified
}

// CreateFilter inserts a new search filter and returns the generated ID.
// Returns ErrInvalidInput if name, keywords, or source are empty.
func (db *DB) CreateFilter(ctx context.Context, name, keywords, location, source string) (string, error) {
	// Validate required fields.
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("name cannot be empty: %w", ErrInvalidInput)
	}
	if strings.TrimSpace(keywords) == "" {
		return "", fmt.Errorf("keywords cannot be empty: %w", ErrInvalidInput)
	}
	if strings.TrimSpace(source) == "" {
		return "", fmt.Errorf("source cannot be empty: %w", ErrInvalidInput)
	}

	id := uuid.New().String()

	var err error
	retryErr := withSQLiteBusyRetryCtx(ctx, func() error {
		_, err = db.conn.ExecContext(ctx,
			`INSERT INTO search_filters (id, name, keywords, location, source)
			 VALUES (?, ?, ?, ?, ?)`,
			id, name, keywords, location, source,
		)
		return err
	})
	if retryErr != nil {
		err = retryErr
	}
	if err != nil {
		slog.Error("failed to create search filter", "filter_id", id, "name", name, "source", source, "error", err)
		return "", fmt.Errorf("creating filter %q: %w", name, err)
	}

	slog.Info("search filter created", "filter_id", id, "name", name, "source", source)

	return id, nil
}

// GetFilter retrieves a single search filter by ID.
// Returns nil and no error if the filter doesn't exist.
func (db *DB) GetFilter(ctx context.Context, id string) (*SearchFilter, error) {
	row := db.conn.QueryRowContext(ctx,
		`SELECT id, name, keywords, location, source, enabled, created_at, updated_at
		 FROM search_filters WHERE id = ?`, id,
	)

	f, err := scanFilter(row)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// ListFilters returns all search filters, ordered by creation date (newest first).
func (db *DB) ListFilters(ctx context.Context) ([]SearchFilter, error) {
	rows, err := db.conn.QueryContext(ctx,
		`SELECT id, name, keywords, location, source, enabled, created_at, updated_at
		 FROM search_filters ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing filters: %w", err)
	}
	defer rows.Close()

	return collectFilters(rows)
}

// ListEnabledFilters returns only filters where enabled = true.
// This is what the scheduler calls to decide which filters to poll.
func (db *DB) ListEnabledFilters(ctx context.Context) ([]SearchFilter, error) {
	rows, err := db.conn.QueryContext(ctx,
		`SELECT id, name, keywords, location, source, enabled, created_at, updated_at
		 FROM search_filters WHERE enabled = 1 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing enabled filters: %w", err)
	}
	defer rows.Close()

	return collectFilters(rows)
}

// UpdateFilter updates a filter's mutable fields and bumps updated_at.
// Returns ErrInvalidInput if name, keywords, or source are empty.
func (db *DB) UpdateFilter(ctx context.Context, id, name, keywords, location, source string, enabled bool) error {
	// Same validation as CreateFilter — enforce consistent invariants.
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("name cannot be empty: %w", ErrInvalidInput)
	}
	if strings.TrimSpace(keywords) == "" {
		return fmt.Errorf("keywords cannot be empty: %w", ErrInvalidInput)
	}
	if strings.TrimSpace(source) == "" {
		return fmt.Errorf("source cannot be empty: %w", ErrInvalidInput)
	}

	enabledInt := 0
	if enabled {
		enabledInt = 1
	}

	var (
		result sql.Result
		err    error
	)
	retryErr := withSQLiteBusyRetryCtx(ctx, func() error {
		result, err = db.conn.ExecContext(ctx,
			`UPDATE search_filters
			 SET name = ?, keywords = ?, location = ?, source = ?, enabled = ?,
			     updated_at = datetime('now')
			 WHERE id = ?`,
			name, keywords, location, source, enabledInt, id,
		)
		return err
	})
	if retryErr != nil {
		err = retryErr
	}
	if err != nil {
		return fmt.Errorf("updating filter %q: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking update result: %w", err)
	}
	if rows == 0 {
		return ErrFilterNotFound
	}

	return nil
}

// DeleteFilter removes a search filter by ID.
// Returns ErrFilterNotFound if the filter doesn't exist.
func (db *DB) DeleteFilter(ctx context.Context, id string) error {
	var (
		result sql.Result
		err    error
	)
	retryErr := withSQLiteBusyRetryCtx(ctx, func() error {
		result, err = db.conn.ExecContext(ctx, "DELETE FROM search_filters WHERE id = ?", id)
		return err
	})
	if retryErr != nil {
		err = retryErr
	}
	if err != nil {
		slog.Error("failed to delete search filter", "filter_id", id, "error", err)
		return fmt.Errorf("deleting filter %q: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking delete result: %w", err)
	}
	if rows == 0 {
		slog.Warn("search filter delete requested but filter not found", "filter_id", id)
		return ErrFilterNotFound
	}

	slog.Info("search filter deleted", "filter_id", id)

	return nil
}

// scanFilter scans a single row into a SearchFilter.
// Returns nil (not an error) if no row was found — this lets callers
// distinguish "not found" from actual database errors.
func scanFilter(row interface{ Scan(dest ...any) error }) (*SearchFilter, error) {
	var f SearchFilter
	var enabled int
	var createdAt, updatedAt string

	err := row.Scan(
		&f.ID, &f.Name, &f.Keywords, &f.Location, &f.Source,
		&enabled, &createdAt, &updatedAt,
	)
	if err != nil {
		// database/sql returns sql.ErrNoRows when QueryRow finds nothing.
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scanning filter: %w", err)
	}

	f.Enabled = enabled == 1

	// Parse timestamps - log warnings if parsing fails but continue with zero time
	var parseErr error
	f.CreatedAt, parseErr = time.Parse("2006-01-02 15:04:05", createdAt)
	if parseErr != nil {
		slog.Warn("failed to parse filter created_at", "filter_id", f.ID, "value", createdAt, "error", parseErr)
	}
	f.UpdatedAt, parseErr = time.Parse("2006-01-02 15:04:05", updatedAt)
	if parseErr != nil {
		slog.Warn("failed to parse filter updated_at", "filter_id", f.ID, "value", updatedAt, "error", parseErr)
	}

	return &f, nil
}

// collectFilters scans all rows from a query into a slice of SearchFilter.
func collectFilters(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]SearchFilter, error) {
	// Return empty slice (not nil) so JSON encoding gives [] not null.
	filters := make([]SearchFilter, 0)

	for rows.Next() {
		var f SearchFilter
		var enabled int
		var createdAt, updatedAt string

		err := rows.Scan(
			&f.ID, &f.Name, &f.Keywords, &f.Location, &f.Source,
			&enabled, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning filter row: %w", err)
		}

		f.Enabled = enabled == 1

		// Parse timestamps with error checking
		var parseErr error
		f.CreatedAt, parseErr = time.Parse("2006-01-02 15:04:05", createdAt)
		if parseErr != nil {
			slog.Warn("failed to parse filter created_at in list", "filter_id", f.ID, "error", parseErr)
		}
		f.UpdatedAt, parseErr = time.Parse("2006-01-02 15:04:05", updatedAt)
		if parseErr != nil {
			slog.Warn("failed to parse filter updated_at in list", "filter_id", f.ID, "error", parseErr)
		}

		filters = append(filters, f)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating filter rows: %w", err)
	}

	return filters, nil
}
