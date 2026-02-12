// Package keychain provides OS-native secure credential storage.
// It wraps zalando/go-keyring for macOS Keychain and Windows Credential Manager.
package keychain

import (
	"fmt"

	gokeyring "github.com/zalando/go-keyring"
)

const serviceName = "HamsterWheel"

// Store defines the interface for secure credential storage.
type Store interface {
	// Get retrieves a secret by key name.
	// Returns empty string and no error if the key doesn't exist.
	Get(key string) (string, error)

	// Set stores a secret under the given key name.
	Set(key, value string) error

	// Delete removes a secret by key name. No error if the key doesn't exist.
	Delete(key string) error
}

// OSStore implements Store using the OS native keychain
// (macOS Keychain, Windows Credential Manager).
type OSStore struct{}

// NewOSStore creates a Store backed by the OS keychain.
func NewOSStore() *OSStore {
	return &OSStore{}
}

func (s *OSStore) Get(key string) (string, error) {
	val, err := gokeyring.Get(serviceName, key)
	if err == gokeyring.ErrNotFound {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get keychain secret %q: %w", key, err)
	}
	return val, nil
}

func (s *OSStore) Set(key, value string) error {
	if err := gokeyring.Set(serviceName, key, value); err != nil {
		return fmt.Errorf("failed to set keychain secret %q: %w", key, err)
	}
	return nil
}

func (s *OSStore) Delete(key string) error {
	err := gokeyring.Delete(serviceName, key)
	if err == gokeyring.ErrNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to delete keychain secret %q: %w", key, err)
	}
	return nil
}
