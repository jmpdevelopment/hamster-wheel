// Package db provides the SQLite database layer for Hamster Wheel.
// All application data (jobs, filters, matches, settings) is stored
// in a single SQLite file in the OS-appropriate config directory.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "modernc.org/sqlite" // SQLite driver — registers itself with database/sql
)

// DB wraps the standard sql.DB connection with Hamster Wheel-specific methods.
type DB struct {
	conn *sql.DB
	path string
}

const (
	sqliteBusyMaxAttempts = 6
	sqliteBusyBaseDelay   = 50 * time.Millisecond
)

// dataDir returns the OS-appropriate directory for storing application data.
//   - macOS:   ~/Library/Application Support/HamsterWheel/
//   - Windows: %APPDATA%\HamsterWheel\
func dataDir() (string, error) {
	var base string

	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("getting home directory: %w", err)
		}
		base = filepath.Join(home, "Library", "Application Support")
	case "windows":
		base = os.Getenv("APPDATA")
		if base == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
	default:
		// Fallback for Linux/other (future support)
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("getting home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}

	return filepath.Join(base, "HamsterWheel"), nil
}

// Open creates (or opens) the SQLite database at the default location
// and runs any pending migrations. The data directory is created if
// it doesn't exist.
func Open() (*DB, error) {
	dir, err := dataDir()
	if err != nil {
		return nil, fmt.Errorf("determining data directory: %w", err)
	}
	return OpenAt(filepath.Join(dir, "hamsterwheel.db"))
}

// OpenAt opens a database at a specific path. Used by Open() for the default
// location, and by tests (which use a temp directory instead of the real config dir).
func OpenAt(path string) (*DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating data directory %s: %w", dir, err)
	}

	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode for better concurrent read performance.
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("setting journal mode: %w", err)
	}

	// Enable foreign key enforcement (off by default in SQLite).
	if _, err := conn.Exec("PRAGMA foreign_keys=ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	db := &DB{conn: conn, path: path}

	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// Path returns the file path to the database.
func (db *DB) Path() string {
	return db.path
}

// Conn returns the underlying *sql.DB for direct queries.
// Prefer using the typed methods (in jobs.go, filters.go, etc.) when available.
func (db *DB) Conn() *sql.DB {
	return db.conn
}

func withSQLiteBusyRetry(operation func() error) error {
	var err error
	for attempt := 1; attempt <= sqliteBusyMaxAttempts; attempt++ {
		err = operation()
		if err == nil {
			return nil
		}
		if !isSQLiteBusyError(err) || attempt == sqliteBusyMaxAttempts {
			return err
		}
		time.Sleep(sqliteBusyBaseDelay * time.Duration(attempt))
	}
	return err
}

func isSQLiteBusyError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return strings.Contains(message, "SQLITE_BUSY") ||
		strings.Contains(message, "database is locked")
}
