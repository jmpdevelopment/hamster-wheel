package llm

import (
	"context"
	"testing"
)

type mockProvider struct {
	name        string
	displayName string
}

func (m *mockProvider) Name() string        { return m.name }
func (m *mockProvider) DisplayName() string { return m.displayName }
func (m *mockProvider) Match(context.Context, MatchRequest) (MatchResult, error) {
	return MatchResult{Score: 0.5, Summary: "ok"}, nil
}
func (m *mockProvider) Validate(context.Context) error { return nil }

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	provider := &mockProvider{name: "mock", displayName: "Mock Provider"}

	if err := reg.Register(provider); err != nil {
		t.Fatalf("registering provider: %v", err)
	}

	got, ok := reg.Get("mock")
	if !ok {
		t.Fatal("expected provider to be found")
	}
	if got.DisplayName() != "Mock Provider" {
		t.Fatalf("expected display name %q, got %q", "Mock Provider", got.DisplayName())
	}
}

func TestRegistryRejectsDuplicateProviderName(t *testing.T) {
	reg := NewRegistry()
	p := &mockProvider{name: "dup", displayName: "Duplicate"}

	if err := reg.Register(p); err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	if err := reg.Register(p); err == nil {
		t.Fatal("expected duplicate register to fail")
	}
}

func TestRegistryNamesSorted(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(&mockProvider{name: "zeta", displayName: "Zeta"})
	_ = reg.Register(&mockProvider{name: "alpha", displayName: "Alpha"})

	names := reg.Names()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0] != "alpha" || names[1] != "zeta" {
		t.Fatalf("expected sorted names [alpha zeta], got %v", names)
	}
}
