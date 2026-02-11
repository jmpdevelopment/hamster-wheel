package main

import (
	"context"
	"log/slog"

	"hamster-wheel/internal/adapter/reed"
	"hamster-wheel/internal/db"
)

const settingReedAPIKey = "reed_api_key"

// SettingsService handles application settings operations exposed to the frontend.
type SettingsService struct {
	db          *db.DB
	reedAdapter *reed.Adapter // Direct reference for API key updates
}

// NewSettingsService creates a new SettingsService.
func NewSettingsService(database *db.DB, reedAdapter *reed.Adapter) *SettingsService {
	return &SettingsService{
		db:          database,
		reedAdapter: reedAdapter,
	}
}

// GetReedAPIKey returns the stored Reed API key (empty if not set).
func (s *SettingsService) GetReedAPIKey() (string, error) {
	return s.db.GetSetting(context.Background(), settingReedAPIKey)
}

// SetReedAPIKey saves the Reed API key and updates the adapter immediately.
func (s *SettingsService) SetReedAPIKey(key string) error {
	if err := s.db.SetSetting(context.Background(), settingReedAPIKey, key); err != nil {
		return err
	}
	s.reedAdapter.SetAPIKey(key)
	slog.Info("Reed API key updated")
	return nil
}