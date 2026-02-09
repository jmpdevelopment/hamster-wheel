package adapter

import (
	"context"
	"testing"
)

// mockAdapter is a minimal Adapter implementation for testing the registry.
// In Go, any type that has all the methods of an interface automatically
// "implements" that interface — no explicit declaration needed.
// (In Python terms: duck typing, but checked at compile time.)
type mockAdapter struct {
	name        string
	displayName string
}

func (m *mockAdapter) Name() string        { return m.name }
func (m *mockAdapter) DisplayName() string { return m.displayName }
func (m *mockAdapter) FetchNewJobs(_ context.Context, _ SearchParams) ([]JobSummary, error) {
	return nil, nil
}
func (m *mockAdapter) FetchJobDetails(_ context.Context, _ JobSummary) (*JobDetails, error) {
	return nil, nil
}
func (m *mockAdapter) Validate(_ context.Context) error { return nil }

func TestRegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	mock := &mockAdapter{name: "indeed_uk", displayName: "Indeed UK"}

	if err := reg.Register(mock); err != nil {
		t.Fatalf("registering adapter: %v", err)
	}

	got, ok := reg.Get("indeed_uk")
	if !ok {
		t.Fatal("expected to find adapter, got not found")
	}
	if got.Name() != "indeed_uk" {
		t.Errorf("expected name %q, got %q", "indeed_uk", got.Name())
	}
	if got.DisplayName() != "Indeed UK" {
		t.Errorf("expected display name %q, got %q", "Indeed UK", got.DisplayName())
	}
}

func TestGetReturnsNotFound(t *testing.T) {
	reg := NewRegistry()

	_, ok := reg.Get("nonexistent")
	if ok {
		t.Error("expected not found for unregistered adapter")
	}
}

func TestRegisterDuplicateReturnsError(t *testing.T) {
	reg := NewRegistry()
	mock := &mockAdapter{name: "indeed_uk", displayName: "Indeed UK"}

	reg.Register(mock)

	err := reg.Register(mock)
	if err == nil {
		t.Fatal("expected error when registering duplicate, got nil")
	}
}

func TestList(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockAdapter{name: "indeed_uk", displayName: "Indeed UK"})
	reg.Register(&mockAdapter{name: "glassdoor", displayName: "Glassdoor"})

	adapters := reg.List()
	if len(adapters) != 2 {
		t.Fatalf("expected 2 adapters, got %d", len(adapters))
	}
}

func TestListEmpty(t *testing.T) {
	reg := NewRegistry()

	adapters := reg.List()
	if len(adapters) != 0 {
		t.Fatalf("expected 0 adapters, got %d", len(adapters))
	}
}

func TestNames(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&mockAdapter{name: "indeed_uk", displayName: "Indeed UK"})
	reg.Register(&mockAdapter{name: "reed", displayName: "Reed"})

	names := reg.Names()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}

	// Check both names are present (order not guaranteed with maps).
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}
	if !nameSet["indeed_uk"] || !nameSet["reed"] {
		t.Errorf("expected names to contain indeed_uk and reed, got %v", names)
	}
}
