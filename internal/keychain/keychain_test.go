package keychain

import "testing"

func TestNewOSStoreImplementsStore(t *testing.T) {
	var store Store = NewOSStore()
	if store == nil {
		t.Fatal("expected OS store instance")
	}
	if _, ok := store.(*OSStore); !ok {
		t.Fatalf("expected Store to be *OSStore, got %T", store)
	}
}

func TestSetAndGet(t *testing.T) {
	store := NewMemoryStore()

	if err := store.Set("reed_api_key", "test-key-123"); err != nil {
		t.Fatalf("setting key: %v", err)
	}

	val, err := store.Get("reed_api_key")
	if err != nil {
		t.Fatalf("getting key: %v", err)
	}
	if val != "test-key-123" {
		t.Errorf("expected %q, got %q", "test-key-123", val)
	}
}

func TestGetReturnsEmptyForMissing(t *testing.T) {
	store := NewMemoryStore()

	val, err := store.Get("nonexistent")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if val != "" {
		t.Errorf("expected empty string, got %q", val)
	}
}

func TestSetOverwrites(t *testing.T) {
	store := NewMemoryStore()

	store.Set("key", "value1")
	store.Set("key", "value2")

	val, _ := store.Get("key")
	if val != "value2" {
		t.Errorf("expected %q, got %q", "value2", val)
	}
}

func TestDelete(t *testing.T) {
	store := NewMemoryStore()

	store.Set("key", "value")
	if err := store.Delete("key"); err != nil {
		t.Fatalf("deleting: %v", err)
	}

	val, _ := store.Get("key")
	if val != "" {
		t.Errorf("expected empty after delete, got %q", val)
	}
}

func TestDeleteNonexistent(t *testing.T) {
	store := NewMemoryStore()

	if err := store.Delete("never_existed"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestMultipleKeys(t *testing.T) {
	store := NewMemoryStore()

	store.Set("reed_api_key", "reed-123")
	store.Set("claude_api_key", "claude-456")

	reed, _ := store.Get("reed_api_key")
	claude, _ := store.Get("claude_api_key")

	if reed != "reed-123" {
		t.Errorf("reed key: expected %q, got %q", "reed-123", reed)
	}
	if claude != "claude-456" {
		t.Errorf("claude key: expected %q, got %q", "claude-456", claude)
	}
}
