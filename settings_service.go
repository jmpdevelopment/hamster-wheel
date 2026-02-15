package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"hamster-wheel/internal/adapter/reed"
	"hamster-wheel/internal/cv"
	"hamster-wheel/internal/db"
	"hamster-wheel/internal/keychain"
	"hamster-wheel/internal/localruntime"
)

const (
	settingReedAPIKey         = "reed_api_key"
	settingOpenAIAPIKey       = "openai_api_key"
	settingTheme              = "theme"
	settingKeyboardShortcuts  = "keyboard_shortcuts"
	settingLLMMode            = "llm_mode"
	settingLLMProvider        = "llm_provider"
	settingLLMModel           = "llm_model"
	settingLLMBaseURL         = "llm_base_url"
	settingLocalRuntimeEngine = "local_runtime_engine"
	settingLocalRuntimeModel  = "local_runtime_model"
	settingCVPath             = "cv_path"

	defaultLLMProvider  = "openai"
	defaultLLMModel     = "gpt-4o-mini"
	defaultLLMMode      = "cloud"
	defaultRuntimeModel = "llama3.1:8b"
)

// SettingsService handles application settings operations exposed to the frontend.
type SettingsService struct {
	db           *db.DB
	keychain     keychain.Store
	reedAdapter  *reed.Adapter // Direct reference for API key updates
	localRuntime localruntime.Manager
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
		db:           database,
		keychain:     kc,
		reedAdapter:  reedAdapter,
		localRuntime: localRuntimeManager,
	}
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
	if err := s.db.SetSetting(context.Background(), settingLocalRuntimeModel, model); err != nil {
		return fmt.Errorf("setting local runtime model: %w", err)
	}
	slog.Info("local runtime model updated", "model", model)
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
