package db

import (
	"context"
	"errors"
	"testing"
)

func TestCreateFilterReturnsID(t *testing.T) {
	db := testDB(t)

	id, err := db.CreateFilter(context.Background(),"Backend London", "golang backend", "London", "indeed_uk")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty ID")
	}
}

func TestCreateFilterGeneratesUniqueIDs(t *testing.T) {
	db := testDB(t)

	id1, err := db.CreateFilter(context.Background(),"Filter 1", "react", "London", "indeed_uk")
	if err != nil {
		t.Fatalf("creating filter 1: %v", err)
	}

	id2, err := db.CreateFilter(context.Background(),"Filter 2", "golang", "Manchester", "indeed_uk")
	if err != nil {
		t.Fatalf("creating filter 2: %v", err)
	}

	if id1 == id2 {
		t.Error("expected different IDs for different filters")
	}
}

func TestGetFilterReturnsCreatedFilter(t *testing.T) {
	db := testDB(t)

	id, err := db.CreateFilter(context.Background(),"Remote React", "react typescript", "Remote", "indeed_uk")
	if err != nil {
		t.Fatalf("creating filter: %v", err)
	}

	f, err := db.GetFilter(context.Background(),id)
	if err != nil {
		t.Fatalf("getting filter: %v", err)
	}
	if f == nil {
		t.Fatal("expected filter, got nil")
	}

	if f.ID != id {
		t.Errorf("ID: got %q, want %q", f.ID, id)
	}
	if f.Name != "Remote React" {
		t.Errorf("Name: got %q, want %q", f.Name, "Remote React")
	}
	if f.Keywords != "react typescript" {
		t.Errorf("Keywords: got %q, want %q", f.Keywords, "react typescript")
	}
	if f.Location != "Remote" {
		t.Errorf("Location: got %q, want %q", f.Location, "Remote")
	}
	if f.Source != "indeed_uk" {
		t.Errorf("Source: got %q, want %q", f.Source, "indeed_uk")
	}
	if !f.Enabled {
		t.Error("expected filter to be enabled by default")
	}
	if f.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if f.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestGetFilterReturnsNilForMissing(t *testing.T) {
	db := testDB(t)

	f, err := db.GetFilter(context.Background(),"nonexistent-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f != nil {
		t.Errorf("expected nil, got %+v", f)
	}
}

func TestListFiltersReturnsAll(t *testing.T) {
	db := testDB(t)

	_, err := db.CreateFilter(context.Background(),"Filter A", "python", "London", "indeed_uk")
	if err != nil {
		t.Fatalf("creating filter A: %v", err)
	}
	_, err = db.CreateFilter(context.Background(),"Filter B", "golang", "Manchester", "indeed_uk")
	if err != nil {
		t.Fatalf("creating filter B: %v", err)
	}

	filters, err := db.ListFilters(context.Background())
	if err != nil {
		t.Fatalf("listing filters: %v", err)
	}

	if len(filters) != 2 {
		t.Fatalf("expected 2 filters, got %d", len(filters))
	}
}

func TestListFiltersReturnsEmptySlice(t *testing.T) {
	db := testDB(t)

	filters, err := db.ListFilters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if filters == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(filters) != 0 {
		t.Errorf("expected 0 filters, got %d", len(filters))
	}
}

func TestListEnabledFiltersExcludesDisabled(t *testing.T) {
	db := testDB(t)

	id1, err := db.CreateFilter(context.Background(),"Enabled", "react", "London", "indeed_uk")
	if err != nil {
		t.Fatalf("creating filter 1: %v", err)
	}

	id2, err := db.CreateFilter(context.Background(),"Disabled", "python", "Remote", "indeed_uk")
	if err != nil {
		t.Fatalf("creating filter 2: %v", err)
	}

	// Disable the second filter.
	err = db.UpdateFilter(context.Background(),id2, "Disabled", "python", "Remote", "indeed_uk", false)
	if err != nil {
		t.Fatalf("disabling filter: %v", err)
	}

	enabled, err := db.ListEnabledFilters(context.Background())
	if err != nil {
		t.Fatalf("listing enabled: %v", err)
	}

	if len(enabled) != 1 {
		t.Fatalf("expected 1 enabled filter, got %d", len(enabled))
	}
	if enabled[0].ID != id1 {
		t.Errorf("expected filter %q, got %q", id1, enabled[0].ID)
	}
}

func TestUpdateFilterChangesFields(t *testing.T) {
	db := testDB(t)

	id, err := db.CreateFilter(context.Background(),"Old Name", "old keywords", "Old City", "indeed_uk")
	if err != nil {
		t.Fatalf("creating filter: %v", err)
	}

	err = db.UpdateFilter(context.Background(),id, "New Name", "new keywords", "New City", "indeed_uk", false)
	if err != nil {
		t.Fatalf("updating filter: %v", err)
	}

	f, err := db.GetFilter(context.Background(),id)
	if err != nil {
		t.Fatalf("getting filter: %v", err)
	}

	if f.Name != "New Name" {
		t.Errorf("Name: got %q, want %q", f.Name, "New Name")
	}
	if f.Keywords != "new keywords" {
		t.Errorf("Keywords: got %q, want %q", f.Keywords, "new keywords")
	}
	if f.Location != "New City" {
		t.Errorf("Location: got %q, want %q", f.Location, "New City")
	}
	if f.Enabled {
		t.Error("expected filter to be disabled after update")
	}
}

func TestUpdateFilterBumpsUpdatedAt(t *testing.T) {
	db := testDB(t)

	id, err := db.CreateFilter(context.Background(),"Test", "test", "London", "indeed_uk")
	if err != nil {
		t.Fatalf("creating filter: %v", err)
	}

	before, _ := db.GetFilter(context.Background(),id)

	err = db.UpdateFilter(context.Background(),id, "Test Updated", "test", "London", "indeed_uk", true)
	if err != nil {
		t.Fatalf("updating filter: %v", err)
	}

	after, _ := db.GetFilter(context.Background(),id)

	// UpdatedAt should be >= before (SQLite datetime resolution is 1 second,
	// so in fast tests they might be equal).
	if after.UpdatedAt.Before(before.UpdatedAt) {
		t.Errorf("UpdatedAt went backwards: %v -> %v", before.UpdatedAt, after.UpdatedAt)
	}
}

func TestUpdateFilterNotFoundReturnsError(t *testing.T) {
	db := testDB(t)

	err := db.UpdateFilter(context.Background(),"nonexistent-id", "Name", "kw", "loc", "indeed_uk", true)
	if !errors.Is(err, ErrFilterNotFound) {
		t.Errorf("expected ErrFilterNotFound, got: %v", err)
	}
}

func TestUpdateFilterRejectsEmptyName(t *testing.T) {
	db := testDB(t)

	id, err := db.CreateFilter(context.Background(),"Valid Name", "golang", "London", "reed_uk")
	if err != nil {
		t.Fatalf("creating filter: %v", err)
	}

	err = db.UpdateFilter(context.Background(),id, "", "golang", "London", "reed_uk", true)
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestUpdateFilterRejectsEmptyKeywords(t *testing.T) {
	db := testDB(t)

	id, err := db.CreateFilter(context.Background(),"Valid Name", "golang", "London", "reed_uk")
	if err != nil {
		t.Fatalf("creating filter: %v", err)
	}

	err = db.UpdateFilter(context.Background(),id, "Valid Name", "", "London", "reed_uk", true)
	if err == nil {
		t.Fatal("expected error for empty keywords, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestUpdateFilterRejectsEmptySource(t *testing.T) {
	db := testDB(t)

	id, err := db.CreateFilter(context.Background(),"Valid Name", "golang", "London", "reed_uk")
	if err != nil {
		t.Fatalf("creating filter: %v", err)
	}

	err = db.UpdateFilter(context.Background(),id, "Valid Name", "golang", "London", "", true)
	if err == nil {
		t.Fatal("expected error for empty source, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestUpdateFilterRejectsWhitespaceOnlyFields(t *testing.T) {
	db := testDB(t)

	id, err := db.CreateFilter(context.Background(),"Valid Name", "golang", "London", "reed_uk")
	if err != nil {
		t.Fatalf("creating filter: %v", err)
	}

	tests := []struct {
		name     string
		fname    string
		keywords string
		source   string
	}{
		{"whitespace name", "   ", "golang", "reed_uk"},
		{"whitespace keywords", "Valid Name", "   ", "reed_uk"},
		{"whitespace source", "Valid Name", "golang", "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.UpdateFilter(context.Background(),id, tt.fname, tt.keywords, "London", tt.source, true)
			if err == nil {
				t.Fatal("expected error for whitespace-only field, got nil")
			}
			if !errors.Is(err, ErrInvalidInput) {
				t.Errorf("expected ErrInvalidInput, got: %v", err)
			}
		})
	}
}

func TestDeleteFilterRemovesIt(t *testing.T) {
	db := testDB(t)

	id, err := db.CreateFilter(context.Background(),"To Delete", "delete me", "London", "indeed_uk")
	if err != nil {
		t.Fatalf("creating filter: %v", err)
	}

	err = db.DeleteFilter(context.Background(),id)
	if err != nil {
		t.Fatalf("deleting filter: %v", err)
	}

	f, err := db.GetFilter(context.Background(),id)
	if err != nil {
		t.Fatalf("getting deleted filter: %v", err)
	}
	if f != nil {
		t.Error("expected nil after deletion")
	}
}

func TestDeleteFilterNotFoundReturnsError(t *testing.T) {
	db := testDB(t)

	err := db.DeleteFilter(context.Background(),"nonexistent-id")
	if !errors.Is(err, ErrFilterNotFound) {
		t.Errorf("expected ErrFilterNotFound, got: %v", err)
	}
}

func TestDeleteFilterSetsJobFilterIDToNull(t *testing.T) {
	db := testDB(t)

	// Create a filter, then manually insert a job referencing it.
	filterID, err := db.CreateFilter(context.Background(),"Test Filter", "golang", "London", "indeed_uk")
	if err != nil {
		t.Fatalf("creating filter: %v", err)
	}

	// Insert a job that references this filter (using raw SQL since jobs CRUD isn't built yet).
	_, err = db.conn.Exec(
		`INSERT INTO jobs (id, source, source_id, title, url, filter_id)
		 VALUES ('job-1', 'indeed_uk', 'src-1', 'Go Developer', 'https://example.com/job', ?)`,
		filterID,
	)
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}

	// Delete the filter — foreign key ON DELETE SET NULL should clear filter_id.
	err = db.DeleteFilter(context.Background(),filterID)
	if err != nil {
		t.Fatalf("deleting filter: %v", err)
	}

	// Verify the job still exists but its filter_id is now NULL.
	var jobFilterID *string
	err = db.conn.QueryRow("SELECT filter_id FROM jobs WHERE id = 'job-1'").Scan(&jobFilterID)
	if err != nil {
		t.Fatalf("querying job: %v", err)
	}
	if jobFilterID != nil {
		t.Errorf("expected NULL filter_id, got %q", *jobFilterID)
	}
}

func TestCreateFilterAllowsDuplicateNames(t *testing.T) {
	db := testDB(t)

	_, err := db.CreateFilter(context.Background(),"Same Name", "react", "London", "indeed_uk")
	if err != nil {
		t.Fatalf("creating first filter: %v", err)
	}

	_, err = db.CreateFilter(context.Background(),"Same Name", "golang", "Manchester", "indeed_uk")
	if err != nil {
		t.Fatalf("creating second filter with same name: %v", err)
	}

	filters, err := db.ListFilters(context.Background())
	if err != nil {
		t.Fatalf("listing filters: %v", err)
	}
	if len(filters) != 2 {
		t.Errorf("expected 2 filters, got %d", len(filters))
	}
}

func TestCreateFilterWithEmptyLocation(t *testing.T) {
	db := testDB(t)

	id, err := db.CreateFilter(context.Background(),"Remote Only", "python", "", "indeed_uk")
	if err != nil {
		t.Fatalf("creating filter: %v", err)
	}

	f, err := db.GetFilter(context.Background(),id)
	if err != nil {
		t.Fatalf("getting filter: %v", err)
	}
	if f.Location != "" {
		t.Errorf("expected empty location, got %q", f.Location)
	}
}

func TestCreateFilterRejectsEmptyName(t *testing.T) {
	db := testDB(t)

	_, err := db.CreateFilter(context.Background(),"", "golang", "London", "indeed_uk")
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestCreateFilterRejectsEmptyKeywords(t *testing.T) {
	db := testDB(t)

	_, err := db.CreateFilter(context.Background(),"My Filter", "", "London", "indeed_uk")
	if err == nil {
		t.Fatal("expected error for empty keywords, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestCreateFilterRejectsEmptySource(t *testing.T) {
	db := testDB(t)

	_, err := db.CreateFilter(context.Background(),"My Filter", "golang", "London", "")
	if err == nil {
		t.Fatal("expected error for empty source, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got: %v", err)
	}
}

func TestCreateFilterRejectsWhitespaceOnlyFields(t *testing.T) {
	db := testDB(t)

	tests := []struct {
		name     string
		fname    string
		keywords string
		location string
		source   string
	}{
		{"whitespace name", "   ", "golang", "London", "indeed_uk"},
		{"whitespace keywords", "My Filter", "   ", "London", "indeed_uk"},
		{"whitespace source", "My Filter", "golang", "London", "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.CreateFilter(context.Background(),tt.fname, tt.keywords, tt.location, tt.source)
			if err == nil {
				t.Fatal("expected error for whitespace-only field, got nil")
			}
			if !errors.Is(err, ErrInvalidInput) {
				t.Errorf("expected ErrInvalidInput, got: %v", err)
			}
		})
	}
}
