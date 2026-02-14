package db

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateReturnsErrorWhenConnectionIsClosed(t *testing.T) {
	database := testDB(t)
	if err := database.Close(); err != nil {
		t.Fatalf("closing DB: %v", err)
	}

	err := database.migrate()
	if err == nil {
		t.Fatal("expected migrate to fail on closed connection")
	}
	if !strings.Contains(err.Error(), "reading schema version") {
		t.Fatalf("expected schema-version context in error, got %v", err)
	}
}

func TestMigrateReturnsErrorForInvalidSQL(t *testing.T) {
	conn, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "broken.db"))
	if err != nil {
		t.Fatalf("opening sqlite connection: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	originalMigrations := migrations
	migrations = []string{"THIS IS NOT SQL;"}
	t.Cleanup(func() { migrations = originalMigrations })

	database := &DB{conn: conn}
	err = database.migrate()
	if err == nil {
		t.Fatal("expected migrate to fail for invalid SQL migration")
	}
	if !strings.Contains(err.Error(), "executing migration 1") {
		t.Fatalf("expected migration execution context in error, got %v", err)
	}
}
