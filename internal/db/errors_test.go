package db

import (
	"testing"
)

// closedDB returns a DB that has been intentionally closed.
// All operations on it should return errors, not panic.
func closedDB(t *testing.T) *DB {
	t.Helper()
	db := testDB(t)
	db.Close()
	return db
}

// --- Filters: operations on closed DB ---

func TestCreateFilterOnClosedDBReturnsError(t *testing.T) {
	db := closedDB(t)
	_, err := db.CreateFilter("Test", "go", "London", "indeed_uk")
	if err == nil {
		t.Error("expected error on closed DB, got nil")
	}
}

func TestGetFilterOnClosedDBReturnsError(t *testing.T) {
	db := closedDB(t)
	_, err := db.GetFilter("some-id")
	if err == nil {
		t.Error("expected error on closed DB, got nil")
	}
}

func TestListFiltersOnClosedDBReturnsError(t *testing.T) {
	db := closedDB(t)
	_, err := db.ListFilters()
	if err == nil {
		t.Error("expected error on closed DB, got nil")
	}
}

func TestListEnabledFiltersOnClosedDBReturnsError(t *testing.T) {
	db := closedDB(t)
	_, err := db.ListEnabledFilters()
	if err == nil {
		t.Error("expected error on closed DB, got nil")
	}
}

func TestUpdateFilterOnClosedDBReturnsError(t *testing.T) {
	db := closedDB(t)
	err := db.UpdateFilter("some-id", "Name", "kw", "loc", "indeed_uk", true)
	if err == nil {
		t.Error("expected error on closed DB, got nil")
	}
}

func TestDeleteFilterOnClosedDBReturnsError(t *testing.T) {
	db := closedDB(t)
	err := db.DeleteFilter("some-id")
	if err == nil {
		t.Error("expected error on closed DB, got nil")
	}
}

// --- Jobs: operations on closed DB ---

func TestInsertJobOnClosedDBReturnsError(t *testing.T) {
	db := closedDB(t)
	job := makeTestJob("closed-db-job")
	_, err := db.InsertJob(job)
	if err == nil {
		t.Error("expected error on closed DB, got nil")
	}
}

func TestGetJobOnClosedDBReturnsError(t *testing.T) {
	db := closedDB(t)
	_, err := db.GetJob("some-id")
	if err == nil {
		t.Error("expected error on closed DB, got nil")
	}
}

func TestJobExistsBySourceIDOnClosedDBReturnsError(t *testing.T) {
	db := closedDB(t)
	_, err := db.JobExistsBySourceID("indeed_uk", "some-id")
	if err == nil {
		t.Error("expected error on closed DB, got nil")
	}
}

func TestListJobsOnClosedDBReturnsError(t *testing.T) {
	db := closedDB(t)
	_, err := db.ListJobs(0)
	if err == nil {
		t.Error("expected error on closed DB, got nil")
	}
}

func TestListJobsByFilterOnClosedDBReturnsError(t *testing.T) {
	db := closedDB(t)
	_, err := db.ListJobsByFilter("some-filter")
	if err == nil {
		t.Error("expected error on closed DB, got nil")
	}
}

func TestDeleteJobOnClosedDBReturnsError(t *testing.T) {
	db := closedDB(t)
	err := db.DeleteJob("some-id")
	if err == nil {
		t.Error("expected error on closed DB, got nil")
	}
}

func TestCountJobsOnClosedDBReturnsError(t *testing.T) {
	db := closedDB(t)
	_, err := db.CountJobs()
	if err == nil {
		t.Error("expected error on closed DB, got nil")
	}
}

// --- Settings: operations on closed DB ---

func TestGetSettingOnClosedDBReturnsError(t *testing.T) {
	db := closedDB(t)
	_, err := db.GetSetting("key")
	if err == nil {
		t.Error("expected error on closed DB, got nil")
	}
}

func TestSetSettingOnClosedDBReturnsError(t *testing.T) {
	db := closedDB(t)
	err := db.SetSetting("key", "value")
	if err == nil {
		t.Error("expected error on closed DB, got nil")
	}
}

func TestDeleteSettingOnClosedDBReturnsError(t *testing.T) {
	db := closedDB(t)
	err := db.DeleteSetting("key")
	if err == nil {
		t.Error("expected error on closed DB, got nil")
	}
}

// --- Double close ---

func TestDoubleCloseDoesNotPanic(t *testing.T) {
	db := testDB(t)
	// First close is fine.
	err := db.Close()
	if err != nil {
		t.Fatalf("first close: %v", err)
	}
	// Second close should not panic (it may return an error, that's fine).
	db.Close()
}

// --- Opening invalid paths ---

func TestOpenAtInvalidPathReturnsError(t *testing.T) {
	// /dev/null is not a valid SQLite database.
	_, err := OpenAt("/dev/null/impossible/path/db.sqlite")
	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}
