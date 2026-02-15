package main

import (
	"errors"
	"testing"
)

type stubKeychainStore struct {
	values map[string]string
	err    error
}

func (s stubKeychainStore) Get(key string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if s.values == nil {
		return "", nil
	}
	return s.values[key], nil
}

func (s stubKeychainStore) Set(string, string) error {
	return nil
}

func (s stubKeychainStore) Delete(string) error {
	return nil
}

func TestLoadReedAPIKeyPrefersKeychainValue(t *testing.T) {
	store := stubKeychainStore{values: map[string]string{
		settingReedAPIKey: " keychain-secret ",
	}}
	key := loadReedAPIKey(store, func(string) string {
		return "env-secret"
	})
	if key != "keychain-secret" {
		t.Fatalf("expected keychain value, got %q", key)
	}
}

func TestLoadReedAPIKeyFallsBackToEnvWhenKeychainEmpty(t *testing.T) {
	store := stubKeychainStore{values: map[string]string{}}
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

func TestLoadAdzunaCredentialsPrefersKeychainValues(t *testing.T) {
	store := stubKeychainStore{values: map[string]string{
		settingAdzunaAppID:  " id-keychain ",
		settingAdzunaAppKey: " key-keychain ",
	}}
	appID, appKey := loadAdzunaCredentials(store, func(string) string {
		return ""
	})
	if appID != "id-keychain" {
		t.Fatalf("expected keychain app id, got %q", appID)
	}
	if appKey != "key-keychain" {
		t.Fatalf("expected keychain app key, got %q", appKey)
	}
}

func TestLoadAdzunaCredentialsFallsBackToEnv(t *testing.T) {
	store := stubKeychainStore{values: map[string]string{}}
	appID, appKey := loadAdzunaCredentials(store, func(name string) string {
		switch name {
		case "ADZUNA_APP_ID":
			return " env-id "
		case "ADZUNA_APP_KEY":
			return " env-key "
		default:
			return ""
		}
	})
	if appID != "env-id" {
		t.Fatalf("expected env app id, got %q", appID)
	}
	if appKey != "env-key" {
		t.Fatalf("expected env app key, got %q", appKey)
	}
}

func TestLoadAdzunaCredentialsUsesEnvWhenKeychainErrors(t *testing.T) {
	store := stubKeychainStore{err: errors.New("keychain unavailable")}
	appID, appKey := loadAdzunaCredentials(store, func(name string) string {
		switch name {
		case "ADZUNA_APP_ID":
			return "env-id"
		case "ADZUNA_APP_KEY":
			return "env-key"
		default:
			return ""
		}
	})
	if appID != "env-id" {
		t.Fatalf("expected env app id, got %q", appID)
	}
	if appKey != "env-key" {
		t.Fatalf("expected env app key, got %q", appKey)
	}
}
