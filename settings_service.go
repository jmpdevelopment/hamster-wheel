package main

import (
	"context"
	"fmt"
	"log/slog"

	"hamster-wheel/internal/adapter/reed"
	"hamster-wheel/internal/db"
	"hamster-wheel/internal/keychain"
)

const (
	settingReedAPIKey          = "reed_api_key"
	settingTheme               = "theme"
	settingKeyboardShortcuts   = "keyboard_shortcuts"
)

// SettingsService handles application settings operations exposed to the frontend.
type SettingsService struct {
	db          *db.DB
	keychain    keychain.Store
	reedAdapter *reed.Adapter // Direct reference for API key updates
}

// NewSettingsService creates a new SettingsService.
func NewSettingsService(database *db.DB, kc keychain.Store, reedAdapter *reed.Adapter) *SettingsService {
	return &SettingsService{
		db:          database,
		keychain:    kc,
		reedAdapter: reedAdapter,
	}
}

// GetReedAPIKey returns the stored Reed API key (empty if not set).
func (s *SettingsService) GetReedAPIKey() (string, error) {
	return s.keychain.Get(settingReedAPIKey)
}

// SetReedAPIKey saves the Reed API key and updates the adapter immediately.
func (s *SettingsService) SetReedAPIKey(key string) error {
	if err := s.keychain.Set(settingReedAPIKey, key); err != nil {
		return err
	}
	s.reedAdapter.SetAPIKey(key)
	slog.Info("Reed API key updated")
	return nil
}

// GetTheme returns the stored theme preference ("dark", "light", "system", or "" if unset).
func (s *SettingsService) GetTheme() (string, error) {
	theme, err := s.db.GetSetting(context.Background(), settingTheme)
	if err != nil {
		return "", fmt.Errorf("getting theme setting: %w", err)
	}
	return theme, nil
}

// SetTheme saves the theme preference. Must be "dark", "light", or "system".
func (s *SettingsService) SetTheme(theme string) error {
	switch theme {
	case "dark", "light", "system":
	default:
		return fmt.Errorf("invalid theme %q: must be dark, light, or system", theme)
	}
	if err := s.db.SetSetting(context.Background(), settingTheme, theme); err != nil {
		return fmt.Errorf("setting theme: %w", err)
	}
	slog.Info("theme preference updated", "theme", theme)
	return nil
}

// GetKeyboardShortcuts returns whether keyboard shortcuts are enabled ("true", "false", or "" if unset).
// Empty string means default (enabled).
func (s *SettingsService) GetKeyboardShortcuts() (string, error) {
	val, err := s.db.GetSetting(context.Background(), settingKeyboardShortcuts)
	if err != nil {
		return "", fmt.Errorf("getting keyboard shortcuts setting: %w", err)
	}
	return val, nil
}

// SetKeyboardShortcuts saves the keyboard shortcuts preference. Must be "true" or "false".
func (s *SettingsService) SetKeyboardShortcuts(enabled string) error {
	switch enabled {
	case "true", "false":
	default:
		return fmt.Errorf("invalid keyboard shortcuts value %q: must be true or false", enabled)
	}
	if err := s.db.SetSetting(context.Background(), settingKeyboardShortcuts, enabled); err != nil {
		return fmt.Errorf("setting keyboard shortcuts: %w", err)
	}
	slog.Info("keyboard shortcuts preference updated", "enabled", enabled)
	return nil
}

