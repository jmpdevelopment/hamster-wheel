package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"

	"hamster-wheel/internal/adapter/adzuna"
	"hamster-wheel/internal/adapter/reed"
	"hamster-wheel/internal/cv"
	"hamster-wheel/internal/db"
	"hamster-wheel/internal/keychain"
	"hamster-wheel/internal/localruntime"
)

const (
	settingReedAPIKey          = "reed_api_key"
	settingAdzunaAppID         = "adzuna_app_id"
	settingAdzunaAppKey        = "adzuna_app_key"
	settingOpenAIAPIKey        = "openai_api_key"
	settingTheme               = "theme"
	settingKeyboardShortcuts   = "keyboard_shortcuts"
	settingJobListPreferences  = "job_list_preferences"
	settingLLMMode             = "llm_mode"
	settingLLMProvider         = "llm_provider"
	settingLLMModel            = "llm_model"
	settingLLMBaseURL          = "llm_base_url"
	settingLocalRuntimeEngine  = "local_runtime_engine"
	settingLocalRuntimeModel   = "local_runtime_model"
	settingAutoPollingEnabled  = "auto_poll_enabled"
	settingPollIntervalMinutes = "poll_interval_minutes"
	settingJobRetentionDays    = "job_retention_days"
	settingAutoMatchEnabled    = "auto_match_enabled"
	settingAutoMatchLimit      = "auto_match_limit"
	settingCVPath              = "cv_path"

	defaultLLMProvider      = "openai"
	defaultLLMModel         = "gpt-4o-mini"
	defaultLLMMode          = "cloud"
	defaultJobListSortMode  = "posted-desc"
	defaultPostedDateFilter = "any"
	defaultMatchScoreFilter = "any"
	defaultRuntimeModel     = "llama3.1:8b"
	defaultAutoPolling      = false
	defaultPollIntervalMin  = 30
	minPollIntervalMinutes  = 30
	maxPollIntervalMinutes  = 24 * 60
	defaultJobRetentionDays = 30
	minJobRetentionDays     = 1
	maxJobRetentionDays     = 30
	defaultAutoMatchLimit   = 0
	defaultAutoMatchEnabled = true
)

// JobListPreferences stores persisted job-list controls.
type JobListPreferences struct {
	FilterByFilterID     string `json:"filterByFilterId,omitempty"`
	SortMode             string `json:"sortMode"`
	PostedDateFilterMode string `json:"postedDateFilterMode"`
	MatchScoreFilterMode string `json:"matchScoreFilterMode"`
	ShowFavoritesOnly    bool   `json:"showFavoritesOnly"`
}

func defaultJobListPreferences() JobListPreferences {
	return JobListPreferences{
		SortMode:             defaultJobListSortMode,
		PostedDateFilterMode: defaultPostedDateFilter,
		MatchScoreFilterMode: defaultMatchScoreFilter,
		ShowFavoritesOnly:    false,
	}
}

func normalizeJobListPreferences(preferences JobListPreferences) JobListPreferences {
	preferences.FilterByFilterID = strings.TrimSpace(preferences.FilterByFilterID)
	preferences.SortMode = strings.TrimSpace(preferences.SortMode)
	preferences.PostedDateFilterMode = strings.TrimSpace(preferences.PostedDateFilterMode)
	preferences.MatchScoreFilterMode = strings.TrimSpace(preferences.MatchScoreFilterMode)

	if preferences.SortMode == "" {
		preferences.SortMode = defaultJobListSortMode
	}
	if preferences.PostedDateFilterMode == "" {
		preferences.PostedDateFilterMode = defaultPostedDateFilter
	}
	if preferences.MatchScoreFilterMode == "" {
		preferences.MatchScoreFilterMode = defaultMatchScoreFilter
	}
	return preferences
}

func validateJobListPreferences(preferences JobListPreferences) error {
	switch preferences.SortMode {
	case "posted-desc", "posted-asc", "score-desc", "score-asc":
	default:
		return fmt.Errorf("invalid job list sort mode %q", preferences.SortMode)
	}
	switch preferences.PostedDateFilterMode {
	case "any", "last-24h", "last-7d", "last-30d":
	default:
		return fmt.Errorf("invalid job list posted date filter %q", preferences.PostedDateFilterMode)
	}
	switch preferences.MatchScoreFilterMode {
	case "any", "scored", "score-80", "score-60", "score-40":
	default:
		return fmt.Errorf("invalid job list match score filter %q", preferences.MatchScoreFilterMode)
	}
	return nil
}

// SettingsService handles application settings operations exposed to the frontend.
type SettingsService struct {
	db            *db.DB
	keychain      keychain.Store
	reedAdapter   *reed.Adapter // Direct reference for API key updates
	adzunaAdapter *adzuna.Adapter
	localRuntime  localruntime.Manager
}

// NewSettingsService creates a new SettingsService.
func NewSettingsService(
	database *db.DB,
	kc keychain.Store,
	reedAdapter *reed.Adapter,
	localRuntimeManagers ...localruntime.Manager,
) *SettingsService {
	localRuntimeManager := localruntime.NewNoopManager()
	if len(localRuntimeManagers) > 0 && localRuntimeManagers[0] != nil {
		localRuntimeManager = localRuntimeManagers[0]
	}

	return &SettingsService{
		db:            database,
		keychain:      kc,
		reedAdapter:   reedAdapter,
		adzunaAdapter: nil,
		localRuntime:  localRuntimeManager,
	}
}

// setAdzunaAdapter wires the runtime Adzuna adapter so credential updates apply immediately.
func (s *SettingsService) setAdzunaAdapter(a *adzuna.Adapter) {
	s.adzunaAdapter = a
}

// HasReedAPIKey reports whether a Reed API key is currently stored.
// It never returns the secret to the frontend.
func (s *SettingsService) HasReedAPIKey() (bool, error) {
	key, err := s.keychain.Get(settingReedAPIKey)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(key) != "", nil
}

// SetReedAPIKey saves the Reed API key and updates the adapter immediately.
func (s *SettingsService) SetReedAPIKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("reed API key is required")
	}

	if err := s.keychain.Set(settingReedAPIKey, key); err != nil {
		return err
	}
	s.reedAdapter.SetAPIKey(key)
	slog.Info("Reed API key updated")
	return nil
}

// ClearReedAPIKey removes the stored Reed API key and clears the active adapter key.
func (s *SettingsService) ClearReedAPIKey() error {
	if err := s.keychain.Delete(settingReedAPIKey); err != nil {
		return err
	}
	s.reedAdapter.SetAPIKey("")
	slog.Info("Reed API key cleared")
	return nil
}

// HasAdzunaCredentials reports whether both Adzuna credentials are currently stored.
func (s *SettingsService) HasAdzunaCredentials() (bool, error) {
	appID, err := s.keychain.Get(settingAdzunaAppID)
	if err != nil {
		return false, err
	}
	appKey, err := s.keychain.Get(settingAdzunaAppKey)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(appID) != "" && strings.TrimSpace(appKey) != "", nil
}

// SetAdzunaCredentials saves Adzuna app credentials and updates the adapter immediately.
func (s *SettingsService) SetAdzunaCredentials(appID, appKey string) error {
	appID = strings.TrimSpace(appID)
	appKey = strings.TrimSpace(appKey)
	if appID == "" {
		return errors.New("adzuna app ID is required")
	}
	if appKey == "" {
		return errors.New("adzuna app key is required")
	}

	if err := s.keychain.Set(settingAdzunaAppID, appID); err != nil {
		return err
	}
	if err := s.keychain.Set(settingAdzunaAppKey, appKey); err != nil {
		// Best-effort cleanup if second write fails.
		_ = s.keychain.Delete(settingAdzunaAppID)
		return err
	}

	if s.adzunaAdapter != nil {
		s.adzunaAdapter.SetCredentials(appID, appKey)
	}
	slog.Info("Adzuna credentials updated")
	return nil
}

// ClearAdzunaCredentials removes stored Adzuna credentials and clears active adapter credentials.
func (s *SettingsService) ClearAdzunaCredentials() error {
	errID := s.keychain.Delete(settingAdzunaAppID)
	errKey := s.keychain.Delete(settingAdzunaAppKey)
	if errID != nil || errKey != nil {
		return errors.Join(errID, errKey)
	}

	if s.adzunaAdapter != nil {
		s.adzunaAdapter.SetCredentials("", "")
	}
	slog.Info("Adzuna credentials cleared")
	return nil
}

// HasOpenAIAPIKey reports whether an OpenAI API key is currently stored.
// It never returns the secret to the frontend.
func (s *SettingsService) HasOpenAIAPIKey() (bool, error) {
	key, err := s.keychain.Get(settingOpenAIAPIKey)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(key) != "", nil
}

// SetOpenAIAPIKey saves the OpenAI API key.
func (s *SettingsService) SetOpenAIAPIKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("openai API key is required")
	}

	if err := s.keychain.Set(settingOpenAIAPIKey, key); err != nil {
		return err
	}
	slog.Info("OpenAI API key updated")
	return nil
}

// ClearOpenAIAPIKey removes the stored OpenAI API key.
func (s *SettingsService) ClearOpenAIAPIKey() error {
	if err := s.keychain.Delete(settingOpenAIAPIKey); err != nil {
		return err
	}
	slog.Info("OpenAI API key cleared")
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

// GetJobListPreferences returns persisted job-list control selections.
func (s *SettingsService) GetJobListPreferences() (JobListPreferences, error) {
	defaults := defaultJobListPreferences()
	value, err := s.db.GetSetting(context.Background(), settingJobListPreferences)
	if err != nil {
		return JobListPreferences{}, fmt.Errorf("getting job list preferences: %w", err)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return defaults, nil
	}

	var preferences JobListPreferences
	if err := json.Unmarshal([]byte(value), &preferences); err != nil {
		slog.Warn("invalid stored job list preferences JSON; using defaults", "error", err)
		return defaults, nil
	}
	preferences = normalizeJobListPreferences(preferences)
	if err := validateJobListPreferences(preferences); err != nil {
		slog.Warn("invalid stored job list preferences; using defaults", "error", err)
		return defaults, nil
	}
	return preferences, nil
}

// SetJobListPreferences saves job-list control selections.
func (s *SettingsService) SetJobListPreferences(preferences JobListPreferences) error {
	preferences = normalizeJobListPreferences(preferences)
	if err := validateJobListPreferences(preferences); err != nil {
		return err
	}
	raw, err := json.Marshal(preferences)
	if err != nil {
		return fmt.Errorf("marshalling job list preferences: %w", err)
	}
	if err := s.db.SetSetting(
		context.Background(),
		settingJobListPreferences,
		string(raw),
	); err != nil {
		return fmt.Errorf("setting job list preferences: %w", err)
	}
	slog.Info("job list preferences updated")
	return nil
}

// GetLLMMode returns the configured LLM operation mode.
// Empty string falls back to default mode.
func (s *SettingsService) GetLLMMode() (string, error) {
	mode, err := s.db.GetSetting(context.Background(), settingLLMMode)
	if err != nil {
		return "", fmt.Errorf("getting llm mode setting: %w", err)
	}
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return defaultLLMMode, nil
	}
	return mode, nil
}

// SetLLMMode saves the configured LLM operation mode.
func (s *SettingsService) SetLLMMode(mode string) error {
	mode = strings.TrimSpace(mode)
	switch mode {
	case "cloud", "local", "advanced":
	default:
		return fmt.Errorf("invalid llm mode %q: must be cloud, local, or advanced", mode)
	}
	if err := s.db.SetSetting(context.Background(), settingLLMMode, mode); err != nil {
		return fmt.Errorf("setting llm mode: %w", err)
	}
	slog.Info("llm mode updated", "mode", mode)
	return nil
}

// GetLLMProvider returns the stored LLM provider ("openai" or "heuristic_v1").
// Empty string falls back to default provider.
func (s *SettingsService) GetLLMProvider() (string, error) {
	provider, err := s.db.GetSetting(context.Background(), settingLLMProvider)
	if err != nil {
		return "", fmt.Errorf("getting llm provider setting: %w", err)
	}
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return defaultLLMProvider, nil
	}
	return provider, nil
}

// SetLLMProvider saves the LLM provider.
func (s *SettingsService) SetLLMProvider(provider string) error {
	provider = strings.TrimSpace(provider)
	switch provider {
	case "openai", "heuristic_v1":
	default:
		return fmt.Errorf("invalid llm provider %q: must be openai or heuristic_v1", provider)
	}

	if err := s.db.SetSetting(context.Background(), settingLLMProvider, provider); err != nil {
		return fmt.Errorf("setting llm provider: %w", err)
	}
	slog.Info("llm provider updated", "provider", provider)
	return nil
}

// GetLLMModel returns the configured LLM model.
// Empty string falls back to default model.
func (s *SettingsService) GetLLMModel() (string, error) {
	model, err := s.db.GetSetting(context.Background(), settingLLMModel)
	if err != nil {
		return "", fmt.Errorf("getting llm model setting: %w", err)
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return defaultLLMModel, nil
	}
	return model, nil
}

// SetLLMModel saves the configured LLM model.
func (s *SettingsService) SetLLMModel(model string) error {
	model = strings.TrimSpace(model)
	if model == "" {
		return errors.New("llm model is required")
	}
	if err := s.db.SetSetting(context.Background(), settingLLMModel, model); err != nil {
		return fmt.Errorf("setting llm model: %w", err)
	}
	slog.Info("llm model updated", "model", model)
	return nil
}

// GetLLMBaseURL returns the configured provider base URL for OpenAI-compatible endpoints.
func (s *SettingsService) GetLLMBaseURL() (string, error) {
	baseURL, err := s.db.GetSetting(context.Background(), settingLLMBaseURL)
	if err != nil {
		return "", fmt.Errorf("getting llm base url setting: %w", err)
	}
	return strings.TrimSpace(baseURL), nil
}

// SetLLMBaseURL saves the provider base URL.
// Empty value resets to provider default endpoint behavior.
func (s *SettingsService) SetLLMBaseURL(baseURL string) error {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL != "" {
		parsed, err := url.ParseRequestURI(baseURL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("invalid llm base url %q", baseURL)
		}
	}

	if err := s.db.SetSetting(context.Background(), settingLLMBaseURL, baseURL); err != nil {
		return fmt.Errorf("setting llm base url: %w", err)
	}
	slog.Info("llm base url updated", "has_value", baseURL != "")
	return nil
}

// GetLocalRuntimeEngine returns the configured local runtime engine.
// Empty string falls back to Ollama.
func (s *SettingsService) GetLocalRuntimeEngine() (string, error) {
	engine, err := s.db.GetSetting(context.Background(), settingLocalRuntimeEngine)
	if err != nil {
		return "", fmt.Errorf("getting local runtime engine setting: %w", err)
	}
	engine = strings.TrimSpace(engine)
	if engine == "" {
		return localruntime.EngineOllama, nil
	}
	return engine, nil
}

// SetLocalRuntimeEngine saves the configured local runtime engine.
func (s *SettingsService) SetLocalRuntimeEngine(engine string) error {
	engine = strings.TrimSpace(engine)
	if engine != localruntime.EngineOllama {
		return fmt.Errorf("invalid local runtime engine %q: must be %s", engine, localruntime.EngineOllama)
	}
	if err := s.db.SetSetting(context.Background(), settingLocalRuntimeEngine, engine); err != nil {
		return fmt.Errorf("setting local runtime engine: %w", err)
	}
	slog.Info("local runtime engine updated", "engine", engine)
	return nil
}

// GetLocalRuntimeModel returns the configured local runtime model.
// Empty string falls back to a default recommended model.
func (s *SettingsService) GetLocalRuntimeModel() (string, error) {
	model, err := s.db.GetSetting(context.Background(), settingLocalRuntimeModel)
	if err != nil {
		return "", fmt.Errorf("getting local runtime model setting: %w", err)
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return defaultRuntimeModel, nil
	}
	return model, nil
}

// SetLocalRuntimeModel saves the configured local runtime model.
func (s *SettingsService) SetLocalRuntimeModel(model string) error {
	model = strings.TrimSpace(model)
	if model == "" {
		return errors.New("local runtime model is required")
	}
	if model != defaultRuntimeModel {
		return fmt.Errorf("invalid local runtime model %q: must be %s", model, defaultRuntimeModel)
	}
	if err := s.db.SetSetting(context.Background(), settingLocalRuntimeModel, model); err != nil {
		return fmt.Errorf("setting local runtime model: %w", err)
	}
	slog.Info("local runtime model updated", "model", model)
	return nil
}

// GetAutoPollingEnabled returns whether background auto-polling is enabled.
func (s *SettingsService) GetAutoPollingEnabled() (bool, error) {
	value, err := s.db.GetSetting(context.Background(), settingAutoPollingEnabled)
	if err != nil {
		return false, fmt.Errorf("getting auto polling enabled setting: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return defaultAutoPolling, nil
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		slog.Warn(
			"invalid auto polling enabled setting value, using default",
			"value",
			value,
		)
		return defaultAutoPolling, nil
	}
}

// SetAutoPollingEnabled saves whether background auto-polling should run.
func (s *SettingsService) SetAutoPollingEnabled(enabled bool) error {
	value := "false"
	if enabled {
		value = "true"
	}
	if err := s.db.SetSetting(context.Background(), settingAutoPollingEnabled, value); err != nil {
		return fmt.Errorf("setting auto polling enabled: %w", err)
	}
	slog.Info("auto polling enabled updated", "enabled", enabled)
	return nil
}

// GetPollIntervalMinutes returns the configured polling interval in minutes.
func (s *SettingsService) GetPollIntervalMinutes() (int, error) {
	value, err := s.db.GetSetting(context.Background(), settingPollIntervalMinutes)
	if err != nil {
		return 0, fmt.Errorf("getting poll interval minutes setting: %w", err)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultPollIntervalMin, nil
	}
	minutes, parseErr := strconv.Atoi(value)
	if parseErr != nil || minutes < minPollIntervalMinutes || minutes > maxPollIntervalMinutes {
		slog.Warn(
			"invalid poll interval minutes setting value, using default",
			"value",
			value,
		)
		return defaultPollIntervalMin, nil
	}
	return minutes, nil
}

// SetPollIntervalMinutes saves the polling interval in minutes.
func (s *SettingsService) SetPollIntervalMinutes(minutes int) error {
	if minutes < minPollIntervalMinutes || minutes > maxPollIntervalMinutes {
		return fmt.Errorf(
			"invalid poll interval minutes %d: must be between %d and %d",
			minutes,
			minPollIntervalMinutes,
			maxPollIntervalMinutes,
		)
	}
	if err := s.db.SetSetting(context.Background(), settingPollIntervalMinutes, strconv.Itoa(minutes)); err != nil {
		return fmt.Errorf("setting poll interval minutes: %w", err)
	}
	slog.Info("poll interval minutes updated", "minutes", minutes)
	return nil
}

// GetJobRetentionDays returns how long jobs should be kept based on posted_at age.
func (s *SettingsService) GetJobRetentionDays() (int, error) {
	value, err := s.db.GetSetting(context.Background(), settingJobRetentionDays)
	if err != nil {
		return 0, fmt.Errorf("getting job retention days setting: %w", err)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultJobRetentionDays, nil
	}
	days, parseErr := strconv.Atoi(value)
	if parseErr != nil || days < minJobRetentionDays || days > maxJobRetentionDays {
		slog.Warn(
			"invalid job retention days setting value, using default",
			"value",
			value,
		)
		return defaultJobRetentionDays, nil
	}
	return days, nil
}

// SetJobRetentionDays saves the retention window for jobs based on posted_at age.
func (s *SettingsService) SetJobRetentionDays(days int) error {
	if days < minJobRetentionDays || days > maxJobRetentionDays {
		return fmt.Errorf(
			"invalid job retention days %d: must be between %d and %d",
			days,
			minJobRetentionDays,
			maxJobRetentionDays,
		)
	}
	if err := s.db.SetSetting(context.Background(), settingJobRetentionDays, strconv.Itoa(days)); err != nil {
		return fmt.Errorf("setting job retention days: %w", err)
	}
	slog.Info("job retention days updated", "days", days)
	return nil
}

// GetAutoMatchEnabled returns whether new jobs should be auto-queued for matching.
func (s *SettingsService) GetAutoMatchEnabled() (bool, error) {
	value, err := s.db.GetSetting(context.Background(), settingAutoMatchEnabled)
	if err != nil {
		return false, fmt.Errorf("getting auto match enabled setting: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return defaultAutoMatchEnabled, nil
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		slog.Warn(
			"invalid auto match enabled setting value, using default",
			"value",
			value,
		)
		return defaultAutoMatchEnabled, nil
	}
}

// SetAutoMatchEnabled saves whether new jobs should be auto-queued for matching.
func (s *SettingsService) SetAutoMatchEnabled(enabled bool) error {
	value := "false"
	if enabled {
		value = "true"
	}
	if err := s.db.SetSetting(context.Background(), settingAutoMatchEnabled, value); err != nil {
		return fmt.Errorf("setting auto match enabled: %w", err)
	}
	slog.Info("auto match enabled updated", "enabled", enabled)
	return nil
}

// GetAutoMatchLimit returns the maximum number of auto-queued matches per poll cycle.
// A value of 0 means unlimited.
func (s *SettingsService) GetAutoMatchLimit() (int, error) {
	value, err := s.db.GetSetting(context.Background(), settingAutoMatchLimit)
	if err != nil {
		return 0, fmt.Errorf("getting auto match limit setting: %w", err)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultAutoMatchLimit, nil
	}
	limit, parseErr := strconv.Atoi(value)
	if parseErr != nil || limit < 0 {
		slog.Warn(
			"invalid auto match limit setting value, using default",
			"value",
			value,
		)
		return defaultAutoMatchLimit, nil
	}
	return limit, nil
}

// SetAutoMatchLimit saves the maximum number of auto-queued matches per poll cycle.
// A value of 0 means unlimited.
func (s *SettingsService) SetAutoMatchLimit(limit int) error {
	if limit < 0 {
		return fmt.Errorf("invalid auto match limit %d: must be >= 0", limit)
	}
	if err := s.db.SetSetting(context.Background(), settingAutoMatchLimit, strconv.Itoa(limit)); err != nil {
		return fmt.Errorf("setting auto match limit: %w", err)
	}
	slog.Info("auto match limit updated", "limit", limit)
	return nil
}

// GetCVPath returns the configured CV file path used for matching context.
func (s *SettingsService) GetCVPath() (string, error) {
	cvPath, err := s.db.GetSetting(context.Background(), settingCVPath)
	if err != nil {
		return "", fmt.Errorf("getting cv path setting: %w", err)
	}
	return strings.TrimSpace(cvPath), nil
}

// SetCVPath saves the CV file path used for matching context.
// Empty value clears the stored path.
func (s *SettingsService) SetCVPath(cvPath string) error {
	cvPath = strings.TrimSpace(cvPath)
	if cvPath != "" {
		if _, err := cv.ExtractProfile(cvPath); err != nil {
			return fmt.Errorf("invalid cv path %q: %w", cvPath, err)
		}
	}

	if err := s.db.SetSetting(context.Background(), settingCVPath, cvPath); err != nil {
		return fmt.Errorf("setting cv path: %w", err)
	}
	slog.Info("cv path updated", "configured", cvPath != "")
	return nil
}

// GetLocalRuntimeStatus returns the current local runtime orchestration status.
func (s *SettingsService) GetLocalRuntimeStatus() (localruntime.Snapshot, error) {
	snapshot, err := s.localRuntime.Status(context.Background())
	if err != nil {
		return localruntime.Snapshot{}, fmt.Errorf("getting local runtime status: %w", err)
	}
	return snapshot, nil
}

// GetLocalRuntimeModels returns recommended and installed local runtime models.
func (s *SettingsService) GetLocalRuntimeModels() (localruntime.ModelCatalog, error) {
	catalog, err := s.localRuntime.ListModels(context.Background())
	if err != nil {
		return localruntime.ModelCatalog{}, fmt.Errorf("listing local runtime models: %w", err)
	}
	return catalog, nil
}

// GetLocalRuntimePullProgress returns in-flight local model pull progress.
func (s *SettingsService) GetLocalRuntimePullProgress() (localruntime.PullProgress, error) {
	progress, err := s.localRuntime.GetPullProgress(context.Background())
	if err != nil {
		return localruntime.PullProgress{}, fmt.Errorf("getting local runtime pull progress: %w", err)
	}
	return progress, nil
}

// PullLocalRuntimeModel pulls a model into local runtime storage.
func (s *SettingsService) PullLocalRuntimeModel(model string) (localruntime.PullResult, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return localruntime.PullResult{}, errors.New("local runtime model is required")
	}
	result, err := s.localRuntime.PullModel(context.Background(), model)
	if err != nil {
		return localruntime.PullResult{}, fmt.Errorf("pulling local runtime model %q: %w", model, err)
	}
	return result, nil
}

// StartLocalRuntime starts the configured local runtime and returns current status.
func (s *SettingsService) StartLocalRuntime() (localruntime.Snapshot, error) {
	snapshot, err := s.localRuntime.Start(context.Background())
	if err != nil {
		return localruntime.Snapshot{}, fmt.Errorf("starting local runtime: %w", err)
	}
	return snapshot, nil
}

// StopLocalRuntime stops the app-managed local runtime process and returns current status.
func (s *SettingsService) StopLocalRuntime() (localruntime.Snapshot, error) {
	snapshot, err := s.localRuntime.Stop(context.Background())
	if err != nil {
		return localruntime.Snapshot{}, fmt.Errorf("stopping local runtime: %w", err)
	}
	return snapshot, nil
}
