package db

import "fmt"

// migrations is an ordered list of SQL statements that build the schema.
// Each migration runs once and is tracked by the schema_version PRAGMA.
// New migrations are appended to the end — never modify existing ones.
var migrations = []string{
	// Migration 1: Core tables
	`CREATE TABLE IF NOT EXISTS search_filters (
		id         TEXT PRIMARY KEY,
		name       TEXT NOT NULL,
		keywords   TEXT NOT NULL,
		location   TEXT NOT NULL DEFAULT '',
		source     TEXT NOT NULL DEFAULT 'indeed_uk',
		enabled    INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS jobs (
		id            TEXT PRIMARY KEY,
		source        TEXT NOT NULL,
		source_id     TEXT NOT NULL,
		title         TEXT NOT NULL,
		company       TEXT NOT NULL DEFAULT '',
		location      TEXT NOT NULL DEFAULT '',
		description   TEXT NOT NULL DEFAULT '',
		url           TEXT NOT NULL,
		posted_at     TEXT,
		discovered_at TEXT NOT NULL DEFAULT (datetime('now')),
		filter_id     TEXT REFERENCES search_filters(id) ON DELETE SET NULL,
		UNIQUE(source, source_id)
	);

	CREATE TABLE IF NOT EXISTS job_matches (
		id                TEXT PRIMARY KEY,
		job_id            TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
		match_score       REAL NOT NULL DEFAULT 0.0,
		match_summary     TEXT NOT NULL DEFAULT '',
		status            TEXT NOT NULL DEFAULT 'matched',
		tailored_cv_path  TEXT,
		tailored_cl_path  TEXT,
		status_updated_at TEXT NOT NULL DEFAULT (datetime('now')),
		created_at        TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS llm_prompts (
		id            TEXT PRIMARY KEY,
		system_prompt TEXT NOT NULL,
		is_custom     INTEGER NOT NULL DEFAULT 0,
		updated_at    TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS approved_domains (
		domain      TEXT PRIMARY KEY,
		approved_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS settings (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);`,
}

// migrate applies any pending migrations based on the schema_version PRAGMA.
func (db *DB) migrate() error {
	// Get current schema version.
	var version int
	err := db.conn.QueryRow("PRAGMA user_version").Scan(&version)
	if err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}

	// Apply each migration that hasn't run yet.
	for i := version; i < len(migrations); i++ {
		tx, err := db.conn.Begin()
		if err != nil {
			return fmt.Errorf("beginning migration %d: %w", i+1, err)
		}

		if _, err := tx.Exec(migrations[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("executing migration %d: %w", i+1, err)
		}

		// Update the schema version inside the same transaction.
		if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", i+1)); err != nil {
			tx.Rollback()
			return fmt.Errorf("updating schema version to %d: %w", i+1, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("committing migration %d: %w", i+1, err)
		}
	}

	return nil
}
