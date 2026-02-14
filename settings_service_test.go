package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"hamster-wheel/internal/adapter/reed"
	"hamster-wheel/internal/db"
	"hamster-wheel/internal/keychain"
)

type failingKeychainStore struct {
	getErr    error
	setErr    error
	deleteErr error
}

func (s failingKeychainStore) Get(string) (string, error) {
	if s.getErr != nil {
		return "", s.getErr
	}
	return "", nil
}

func (s failingKeychainStore) Set(string, string) error {
	return s.setErr
}

func (s failingKeychainStore) Delete(string) error {
	return s.deleteErr
}

func openSettingsTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.OpenAt(filepath.Join(t.TempDir(), "settings-test.db"))
	if err != nil {
		t.Fatalf("opening test DB: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func TestSettingsServiceAPIKeyLifecycle(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	has, err := svc.HasReedAPIKey()
	if err != nil {
		t.Fatalf("checking initial key presence: %v", err)
	}
	if has {
		t.Fatal("expected HasReedAPIKey=false initially")
	}
	if reedAdapter.HasAPIKey() {
		t.Fatal("expected adapter to start without API key")
	}

	if err := svc.SetReedAPIKey("  secret-key-123  "); err != nil {
		t.Fatalf("setting API key: %v", err)
	}

	has, err = svc.HasReedAPIKey()
	if err != nil {
		t.Fatalf("checking key presence after set: %v", err)
	}
	if !has {
		t.Fatal("expected HasReedAPIKey=true after set")
	}
	if !reedAdapter.HasAPIKey() {
		t.Fatal("expected adapter to have key after set")
	}

	stored, err := kc.Get(settingReedAPIKey)
	if err != nil {
		t.Fatalf("reading keychain key: %v", err)
	}
	if stored != "secret-key-123" {
		t.Fatalf("expected trimmed key to be stored, got %q", stored)
	}

	if err := svc.ClearReedAPIKey(); err != nil {
		t.Fatalf("clearing API key: %v", err)
	}

	has, err = svc.HasReedAPIKey()
	if err != nil {
		t.Fatalf("checking key presence after clear: %v", err)
	}
	if has {
		t.Fatal("expected HasReedAPIKey=false after clear")
	}
	if reedAdapter.HasAPIKey() {
		t.Fatal("expected adapter key to be cleared")
	}
}

func TestSetReedAPIKeyRejectsEmpty(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	if err := svc.SetReedAPIKey("   "); err == nil {
		t.Fatal("expected error for empty API key")
	}
}

func TestSettingsServiceAPIKeyErrorsPropagate(t *testing.T) {
	database := openSettingsTestDB(t)
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, failingKeychainStore{
		getErr:    errors.New("get failed"),
		setErr:    errors.New("set failed"),
		deleteErr: errors.New("delete failed"),
	}, reedAdapter)

	if _, err := svc.HasReedAPIKey(); err == nil {
		t.Fatal("expected error from HasReedAPIKey")
	}
	if err := svc.SetReedAPIKey("key-1"); err == nil {
		t.Fatal("expected error from SetReedAPIKey")
	}
	if err := svc.ClearReedAPIKey(); err == nil {
		t.Fatal("expected error from ClearReedAPIKey")
	}
	if reedAdapter.HasAPIKey() {
		t.Fatal("expected adapter key to remain unchanged on failures")
	}
}

func TestKeyboardShortcutsLifecycleAndValidation(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	current, err := svc.GetKeyboardShortcuts()
	if err != nil {
		t.Fatalf("reading default keyboard shortcuts setting: %v", err)
	}
	if current != "" {
		t.Fatalf("expected empty default keyboard shortcuts setting, got %q", current)
	}

	if err := svc.SetKeyboardShortcuts("true"); err != nil {
		t.Fatalf("setting keyboard shortcuts=true: %v", err)
	}
	current, err = svc.GetKeyboardShortcuts()
	if err != nil {
		t.Fatalf("reading keyboard shortcuts after true: %v", err)
	}
	if current != "true" {
		t.Fatalf("expected keyboard shortcuts value true, got %q", current)
	}

	if err := svc.SetKeyboardShortcuts("false"); err != nil {
		t.Fatalf("setting keyboard shortcuts=false: %v", err)
	}
	current, err = svc.GetKeyboardShortcuts()
	if err != nil {
		t.Fatalf("reading keyboard shortcuts after false: %v", err)
	}
	if current != "false" {
		t.Fatalf("expected keyboard shortcuts value false, got %q", current)
	}

	err = svc.SetKeyboardShortcuts("yes")
	if err == nil {
		t.Fatal("expected validation error for invalid keyboard shortcuts value")
	}
	if !strings.Contains(err.Error(), "invalid keyboard shortcuts value") {
		t.Fatalf("expected validation context in error, got %v", err)
	}
}

func TestKeyboardShortcutsDatabaseErrors(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	if err := database.Close(); err != nil {
		t.Fatalf("closing DB: %v", err)
	}

	_, err := svc.GetKeyboardShortcuts()
	if err == nil {
		t.Fatal("expected GetKeyboardShortcuts to fail on closed DB")
	}
	if !strings.Contains(err.Error(), "getting keyboard shortcuts setting") {
		t.Fatalf("expected get-setting context, got %v", err)
	}

	err = svc.SetKeyboardShortcuts("true")
	if err == nil {
		t.Fatal("expected SetKeyboardShortcuts to fail on closed DB")
	}
	if !strings.Contains(err.Error(), "setting keyboard shortcuts") {
		t.Fatalf("expected set-setting context, got %v", err)
	}
}
