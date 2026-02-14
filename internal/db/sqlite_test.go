package db

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// testDB is a helper that creates a temporary database for testing.
// The database is automatically cleaned up when the test finishes.
func testDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir() // Go auto-cleans this after the test
	db, err := OpenAt(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestOpenCreatesDatabase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	db, err := OpenAt(path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer db.Close()

	// Verify the file was created.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("database file was not created")
	}

	// Verify the path is stored.
	if db.Path() != path {
		t.Errorf("expected path %q, got %q", path, db.Path())
	}
}

func TestOpenCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "sub", "dir", "test.db")

	db, err := OpenAt(nested)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(nested); os.IsNotExist(err) {
		t.Fatal("database file was not created in nested directory")
	}
}

func TestMigrationsCreateTables(t *testing.T) {
	db := testDB(t)

	// Query sqlite_master to check all expected tables exist.
	tables := []string{"search_filters", "jobs", "job_matches", "llm_prompts", "approved_domains", "settings"}

	for _, table := range tables {
		var name string
		err := db.Conn().QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}
}

func TestMigrationSetsVersion(t *testing.T) {
	db := testDB(t)

	var version int
	err := db.Conn().QueryRow("PRAGMA user_version").Scan(&version)
	if err != nil {
		t.Fatalf("reading schema version: %v", err)
	}

	if version != len(migrations) {
		t.Errorf("expected schema version %d, got %d", len(migrations), version)
	}
}

func TestMigrationsAreIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	// Open once — runs migrations.
	db1, err := OpenAt(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	db1.Close()

	// Open again — should not fail (migrations already applied).
	db2, err := OpenAt(path)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer db2.Close()
}

func TestForeignKeysEnabled(t *testing.T) {
	db := testDB(t)

	var fk int
	err := db.Conn().QueryRow("PRAGMA foreign_keys").Scan(&fk)
	if err != nil {
		t.Fatalf("reading foreign_keys pragma: %v", err)
	}

	if fk != 1 {
		t.Error("foreign keys are not enabled")
	}
}

func TestOpenUsesDefaultDataDir(t *testing.T) {
	tempHome := t.TempDir()

	switch runtime.GOOS {
	case "windows":
		t.Setenv("APPDATA", filepath.Join(tempHome, "AppData", "Roaming"))
	default:
		t.Setenv("HOME", tempHome)
	}

	database, err := Open()
	if err != nil {
		t.Fatalf("opening database at default path: %v", err)
	}
	defer database.Close()

	expectedDir, err := dataDir()
	if err != nil {
		t.Fatalf("resolving expected data directory: %v", err)
	}
	expectedPath := filepath.Join(expectedDir, "hamsterwheel.db")

	if database.Path() != expectedPath {
		t.Fatalf("expected DB path %q, got %q", expectedPath, database.Path())
	}
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("expected database file at default path, stat failed: %v", err)
	}
}

func TestOpenAtReturnsErrorWhenParentPathIsAFile(t *testing.T) {
	parentFile := filepath.Join(t.TempDir(), "parent-file")
	if err := os.WriteFile(parentFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("creating parent file: %v", err)
	}

	_, err := OpenAt(filepath.Join(parentFile, "test.db"))
	if err == nil {
		t.Fatal("expected error when parent path is a file")
	}
	if !strings.Contains(err.Error(), "creating data directory") {
		t.Fatalf("expected data directory context in error, got %v", err)
	}
}
