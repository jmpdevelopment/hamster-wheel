package main

import (
	"errors"
	"testing"
)

type stubKeychainStore struct {
	value string
	err   error
}

func (s stubKeychainStore) Get(string) (string, error) {
	return s.value, s.err
}

func (s stubKeychainStore) Set(string, string) error {
	return nil
}

func (s stubKeychainStore) Delete(string) error {
	return nil
}

func TestLoadReedAPIKeyPrefersKeychainValue(t *testing.T) {
	store := stubKeychainStore{value: " keychain-secret "}
	key := loadReedAPIKey(store, func(string) string {
		return "env-secret"
	})
	if key != "keychain-secret" {
		t.Fatalf("expected keychain value, got %q", key)
	}
}

func TestLoadReedAPIKeyFallsBackToEnvWhenKeychainEmpty(t *testing.T) {
	store := stubKeychainStore{}
	key := loadReedAPIKey(store, func(string) string {
		return " env-secret "
	})
	if key != "env-secret" {
		t.Fatalf("expected env fallback value, got %q", key)
	}
}

func TestLoadReedAPIKeyFallsBackToEnvWhenKeychainErrors(t *testing.T) {
	store := stubKeychainStore{err: errors.New("keychain unavailable")}
	key := loadReedAPIKey(store, func(string) string {
		return "env-secret"
	})
	if key != "env-secret" {
		t.Fatalf("expected env fallback on keychain error, got %q", key)
	}
}
