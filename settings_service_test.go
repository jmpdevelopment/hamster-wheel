package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hamster-wheel/internal/adapter/adzuna"
	"hamster-wheel/internal/adapter/reed"
	"hamster-wheel/internal/db"
	"hamster-wheel/internal/keychain"
	"hamster-wheel/internal/localruntime"
)

type failingKeychainStore struct {
	getErr    error
	setErr    error
	deleteErr error
}

type failingLocalRuntimeManager struct {
	statusErr     error
	startErr      error
	stopErr       error
	listErr       error
	pullModelErr  error
	pullStateErr  error
	modelCatalog  localruntime.ModelCatalog
	pullModelResp localruntime.PullResult
	pullStateResp localruntime.PullProgress
}

func (m failingLocalRuntimeManager) Status(context.Context) (localruntime.Snapshot, error) {
	if m.statusErr != nil {
		return localruntime.Snapshot{}, m.statusErr
	}
	return localruntime.Snapshot{Status: localruntime.StatusReady}, nil
}

func (m failingLocalRuntimeManager) Start(context.Context) (localruntime.Snapshot, error) {
	if m.startErr != nil {
		return localruntime.Snapshot{}, m.startErr
	}
	return localruntime.Snapshot{Status: localruntime.StatusReady}, nil
}

func (m failingLocalRuntimeManager) Stop(context.Context) (localruntime.Snapshot, error) {
	if m.stopErr != nil {
		return localruntime.Snapshot{}, m.stopErr
	}
	return localruntime.Snapshot{Status: localruntime.StatusStopped}, nil
}

func (m failingLocalRuntimeManager) ListModels(context.Context) (localruntime.ModelCatalog, error) {
	if m.listErr != nil {
		return localruntime.ModelCatalog{}, m.listErr
	}
	if m.modelCatalog.Engine == "" {
		m.modelCatalog.Engine = localruntime.EngineOllama
	}
	return m.modelCatalog, nil
}

func (m failingLocalRuntimeManager) PullModel(_ context.Context, model string) (localruntime.PullResult, error) {
	if m.pullModelErr != nil {
		return localruntime.PullResult{}, m.pullModelErr
	}
	if m.pullModelResp.Model == "" {
		m.pullModelResp = localruntime.PullResult{
			Model:  model,
			Status: "success",
			Ready:  true,
		}
	}
	return m.pullModelResp, nil
}

func (m failingLocalRuntimeManager) GetPullProgress(context.Context) (localruntime.PullProgress, error) {
	if m.pullStateErr != nil {
		return localruntime.PullProgress{}, m.pullStateErr
	}
	if m.pullStateResp.Model == "" && !m.pullStateResp.Active {
		m.pullStateResp = localruntime.PullProgress{
			Model:          "llama3.1:8b",
			Active:         true,
			Status:         "downloading",
			TotalBytes:     1024,
			CompletedBytes: 256,
			Percent:        25,
		}
	}
	return m.pullStateResp, nil
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
	adzunaAdapter := adzuna.New("", "")
	svc := NewSettingsService(database, kc, reedAdapter)
	svc.setAdzunaAdapter(adzunaAdapter)

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

	hasAdzuna, err := svc.HasAdzunaCredentials()
	if err != nil {
		t.Fatalf("checking initial adzuna credentials presence: %v", err)
	}
	if hasAdzuna {
		t.Fatal("expected HasAdzunaCredentials=false initially")
	}
	if adzunaAdapter.HasCredentials() {
		t.Fatal("expected adzuna adapter to start without credentials")
	}

	if err := svc.SetAdzunaCredentials("  app-id-123  ", "  app-key-123  "); err != nil {
		t.Fatalf("setting adzuna credentials: %v", err)
	}

	hasAdzuna, err = svc.HasAdzunaCredentials()
	if err != nil {
		t.Fatalf("checking adzuna credentials after set: %v", err)
	}
	if !hasAdzuna {
		t.Fatal("expected HasAdzunaCredentials=true after set")
	}
	if !adzunaAdapter.HasCredentials() {
		t.Fatal("expected adzuna adapter to have credentials after set")
	}

	storedID, err := kc.Get(settingAdzunaAppID)
	if err != nil {
		t.Fatalf("reading adzuna app id key: %v", err)
	}
	storedKey, err := kc.Get(settingAdzunaAppKey)
	if err != nil {
		t.Fatalf("reading adzuna app key key: %v", err)
	}
	if storedID != "app-id-123" {
		t.Fatalf("expected trimmed adzuna app id to be stored, got %q", storedID)
	}
	if storedKey != "app-key-123" {
		t.Fatalf("expected trimmed adzuna app key to be stored, got %q", storedKey)
	}

	if err := svc.ClearAdzunaCredentials(); err != nil {
		t.Fatalf("clearing adzuna credentials: %v", err)
	}
	hasAdzuna, err = svc.HasAdzunaCredentials()
	if err != nil {
		t.Fatalf("checking adzuna credentials after clear: %v", err)
	}
	if hasAdzuna {
		t.Fatal("expected HasAdzunaCredentials=false after clear")
	}
	if adzunaAdapter.HasCredentials() {
		t.Fatal("expected adzuna adapter credentials to be cleared")
	}
}

type stubLocalRuntimeManager struct{}

func (stubLocalRuntimeManager) Status(context.Context) (localruntime.Snapshot, error) {
	return localruntime.Snapshot{Status: localruntime.StatusReady}, nil
}

func (stubLocalRuntimeManager) Start(context.Context) (localruntime.Snapshot, error) {
	return localruntime.Snapshot{Status: localruntime.StatusReady}, nil
}

func (stubLocalRuntimeManager) Stop(context.Context) (localruntime.Snapshot, error) {
	return localruntime.Snapshot{Status: localruntime.StatusStopped}, nil
}

func (stubLocalRuntimeManager) ListModels(context.Context) (localruntime.ModelCatalog, error) {
	return localruntime.ModelCatalog{
		Engine:      localruntime.EngineOllama,
		Recommended: []string{"llama3.1:8b"},
		Installed: []localruntime.ModelInfo{
			{Name: "llama3.1:8b", SizeBytes: 1024},
		},
	}, nil
}

func (stubLocalRuntimeManager) PullModel(_ context.Context, model string) (localruntime.PullResult, error) {
	return localruntime.PullResult{
		Model:  model,
		Status: "success",
		Ready:  true,
	}, nil
}

func (stubLocalRuntimeManager) GetPullProgress(context.Context) (localruntime.PullProgress, error) {
	return localruntime.PullProgress{
		Model:          "llama3.1:8b",
		Active:         false,
		Status:         "completed",
		TotalBytes:     1024,
		CompletedBytes: 1024,
		Percent:        100,
		Ready:          true,
	}, nil
}

func TestSettingsServiceLocalRuntimeDependencyDefaultsAndInjection(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")

	defaultSvc := NewSettingsService(database, kc, reedAdapter)
	if defaultSvc.localRuntime == nil {
		t.Fatal("expected default local runtime manager to be configured")
	}
	defaultSnapshot, err := defaultSvc.localRuntime.Status(context.Background())
	if err != nil {
		t.Fatalf("checking default local runtime status: %v", err)
	}
	if defaultSnapshot.Status != localruntime.StatusNotInstalled {
		t.Fatalf("expected default local runtime status %q, got %q", localruntime.StatusNotInstalled, defaultSnapshot.Status)
	}

	customManager := stubLocalRuntimeManager{}
	customSvc := NewSettingsService(database, kc, reedAdapter, customManager)
	if customSvc.localRuntime == nil {
		t.Fatal("expected injected local runtime manager to be set")
	}
	customSnapshot, err := customSvc.localRuntime.Status(context.Background())
	if err != nil {
		t.Fatalf("checking custom local runtime status: %v", err)
	}
	if customSnapshot.Status != localruntime.StatusReady {
		t.Fatalf("expected injected runtime status %q, got %q", localruntime.StatusReady, customSnapshot.Status)
	}
}

func TestSettingsServiceLocalRuntimeLifecycle(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	local := stubLocalRuntimeManager{}
	svc := NewSettingsService(database, kc, reedAdapter, local)

	status, err := svc.GetLocalRuntimeStatus()
	if err != nil {
		t.Fatalf("getting local runtime status: %v", err)
	}
	if status.Status != localruntime.StatusReady {
		t.Fatalf("expected local runtime status %q, got %q", localruntime.StatusReady, status.Status)
	}

	started, err := svc.StartLocalRuntime()
	if err != nil {
		t.Fatalf("starting local runtime: %v", err)
	}
	if started.Status != localruntime.StatusReady {
		t.Fatalf("expected start status %q, got %q", localruntime.StatusReady, started.Status)
	}

	stopped, err := svc.StopLocalRuntime()
	if err != nil {
		t.Fatalf("stopping local runtime: %v", err)
	}
	if stopped.Status != localruntime.StatusStopped {
		t.Fatalf("expected stop status %q, got %q", localruntime.StatusStopped, stopped.Status)
	}

	models, err := svc.GetLocalRuntimeModels()
	if err != nil {
		t.Fatalf("listing local runtime models: %v", err)
	}
	if models.Engine != localruntime.EngineOllama {
		t.Fatalf("expected model catalog engine %q, got %q", localruntime.EngineOllama, models.Engine)
	}
	if len(models.Installed) != 1 {
		t.Fatalf("expected 1 installed model, got %d", len(models.Installed))
	}

	pulled, err := svc.PullLocalRuntimeModel("llama3.1:8b")
	if err != nil {
		t.Fatalf("pulling local runtime model: %v", err)
	}
	if pulled.Model != "llama3.1:8b" {
		t.Fatalf("expected pulled model llama3.1:8b, got %q", pulled.Model)
	}
	if !pulled.Ready {
		t.Fatal("expected pulled model ready=true")
	}

	pullState, err := svc.GetLocalRuntimePullProgress()
	if err != nil {
		t.Fatalf("getting local runtime pull progress: %v", err)
	}
	if pullState.Percent != 100 {
		t.Fatalf("expected pull progress percent 100, got %v", pullState.Percent)
	}
	if !pullState.Ready {
		t.Fatal("expected pull progress ready=true")
	}
}

func TestSettingsServiceLocalRuntimeErrors(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter, failingLocalRuntimeManager{
		statusErr:    errors.New("status failed"),
		startErr:     errors.New("start failed"),
		stopErr:      errors.New("stop failed"),
		listErr:      errors.New("list models failed"),
		pullModelErr: errors.New("pull model failed"),
		pullStateErr: errors.New("pull progress failed"),
	})

	if _, err := svc.GetLocalRuntimeStatus(); err == nil || !strings.Contains(err.Error(), "getting local runtime status") {
		t.Fatalf("expected wrapped status error, got %v", err)
	}
	if _, err := svc.StartLocalRuntime(); err == nil || !strings.Contains(err.Error(), "starting local runtime") {
		t.Fatalf("expected wrapped start error, got %v", err)
	}
	if _, err := svc.StopLocalRuntime(); err == nil || !strings.Contains(err.Error(), "stopping local runtime") {
		t.Fatalf("expected wrapped stop error, got %v", err)
	}
	if _, err := svc.GetLocalRuntimeModels(); err == nil || !strings.Contains(err.Error(), "listing local runtime models") {
		t.Fatalf("expected wrapped list-models error, got %v", err)
	}
	if _, err := svc.PullLocalRuntimeModel("llama3.1:8b"); err == nil || !strings.Contains(err.Error(), "pulling local runtime model") {
		t.Fatalf("expected wrapped pull-model error, got %v", err)
	}
	if _, err := svc.GetLocalRuntimePullProgress(); err == nil || !strings.Contains(err.Error(), "getting local runtime pull progress") {
		t.Fatalf("expected wrapped pull-progress error, got %v", err)
	}
	if _, err := svc.PullLocalRuntimeModel("   "); err == nil || !strings.Contains(err.Error(), "local runtime model is required") {
		t.Fatalf("expected local runtime model validation error, got %v", err)
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
	if err := svc.SetAdzunaCredentials("   ", "app-key"); err == nil {
		t.Fatal("expected error for empty adzuna app id")
	}
	if err := svc.SetAdzunaCredentials("app-id", "   "); err == nil {
		t.Fatal("expected error for empty adzuna app key")
	}
}

func TestSettingsServiceAPIKeyErrorsPropagate(t *testing.T) {
	database := openSettingsTestDB(t)
	reedAdapter := reed.New("")
	adzunaAdapter := adzuna.New("", "")
	svc := NewSettingsService(database, failingKeychainStore{
		getErr:    errors.New("get failed"),
		setErr:    errors.New("set failed"),
		deleteErr: errors.New("delete failed"),
	}, reedAdapter)
	svc.setAdzunaAdapter(adzunaAdapter)

	if _, err := svc.HasReedAPIKey(); err == nil {
		t.Fatal("expected error from HasReedAPIKey")
	}
	if err := svc.SetReedAPIKey("key-1"); err == nil {
		t.Fatal("expected error from SetReedAPIKey")
	}
	if err := svc.ClearReedAPIKey(); err == nil {
		t.Fatal("expected error from ClearReedAPIKey")
	}
	if _, err := svc.HasAdzunaCredentials(); err == nil {
		t.Fatal("expected error from HasAdzunaCredentials")
	}
	if err := svc.SetAdzunaCredentials("app-id", "app-key"); err == nil {
		t.Fatal("expected error from SetAdzunaCredentials")
	}
	if err := svc.ClearAdzunaCredentials(); err == nil {
		t.Fatal("expected error from ClearAdzunaCredentials")
	}
	if reedAdapter.HasAPIKey() {
		t.Fatal("expected adapter key to remain unchanged on failures")
	}
	if adzunaAdapter.HasCredentials() {
		t.Fatal("expected adzuna adapter credentials to remain unchanged on failures")
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

func TestOpenAIAPIKeyLifecycle(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	has, err := svc.HasOpenAIAPIKey()
	if err != nil {
		t.Fatalf("checking initial openai key presence: %v", err)
	}
	if has {
		t.Fatal("expected HasOpenAIAPIKey=false initially")
	}

	if err := svc.SetOpenAIAPIKey("  sk-test-openai  "); err != nil {
		t.Fatalf("setting openai API key: %v", err)
	}

	has, err = svc.HasOpenAIAPIKey()
	if err != nil {
		t.Fatalf("checking openai key presence after set: %v", err)
	}
	if !has {
		t.Fatal("expected HasOpenAIAPIKey=true after set")
	}

	stored, err := kc.Get(settingOpenAIAPIKey)
	if err != nil {
		t.Fatalf("reading openai keychain key: %v", err)
	}
	if stored != "sk-test-openai" {
		t.Fatalf("expected trimmed key to be stored, got %q", stored)
	}

	if err := svc.ClearOpenAIAPIKey(); err != nil {
		t.Fatalf("clearing openai API key: %v", err)
	}

	has, err = svc.HasOpenAIAPIKey()
	if err != nil {
		t.Fatalf("checking openai key presence after clear: %v", err)
	}
	if has {
		t.Fatal("expected HasOpenAIAPIKey=false after clear")
	}
}

func TestOpenAIAPIKeyValidationAndErrors(t *testing.T) {
	database := openSettingsTestDB(t)
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, failingKeychainStore{
		getErr:    errors.New("get failed"),
		setErr:    errors.New("set failed"),
		deleteErr: errors.New("delete failed"),
	}, reedAdapter)

	if _, err := svc.HasOpenAIAPIKey(); err == nil {
		t.Fatal("expected error from HasOpenAIAPIKey")
	}
	if err := svc.SetOpenAIAPIKey("   "); err == nil {
		t.Fatal("expected validation error for empty openai API key")
	}
	if err := svc.SetOpenAIAPIKey("abc"); err == nil {
		t.Fatal("expected keychain error from SetOpenAIAPIKey")
	}
	if err := svc.ClearOpenAIAPIKey(); err == nil {
		t.Fatal("expected keychain error from ClearOpenAIAPIKey")
	}
}

func TestLLMModeDefaultsAndLifecycle(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	mode, err := svc.GetLLMMode()
	if err != nil {
		t.Fatalf("getting default llm mode: %v", err)
	}
	if mode != defaultLLMMode {
		t.Fatalf("expected default llm mode %q, got %q", defaultLLMMode, mode)
	}

	for _, valid := range []string{"cloud", "local", "advanced"} {
		if err := svc.SetLLMMode(valid); err != nil {
			t.Fatalf("setting llm mode %q: %v", valid, err)
		}
		mode, err = svc.GetLLMMode()
		if err != nil {
			t.Fatalf("getting llm mode after set %q: %v", valid, err)
		}
		if mode != valid {
			t.Fatalf("expected llm mode %q, got %q", valid, mode)
		}
	}

	if err := svc.SetLLMMode("experimental"); err == nil {
		t.Fatal("expected validation error for invalid llm mode")
	}
}

func TestLocalRuntimeEngineAndModelLifecycle(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	engine, err := svc.GetLocalRuntimeEngine()
	if err != nil {
		t.Fatalf("getting default local runtime engine: %v", err)
	}
	if engine != localruntime.EngineOllama {
		t.Fatalf("expected default engine %q, got %q", localruntime.EngineOllama, engine)
	}

	if err := svc.SetLocalRuntimeEngine(localruntime.EngineOllama); err != nil {
		t.Fatalf("setting local runtime engine: %v", err)
	}
	if err := svc.SetLocalRuntimeEngine("vllm"); err == nil {
		t.Fatal("expected validation error for unsupported local runtime engine")
	}

	model, err := svc.GetLocalRuntimeModel()
	if err != nil {
		t.Fatalf("getting default local runtime model: %v", err)
	}
	if model != defaultRuntimeModel {
		t.Fatalf("expected default runtime model %q, got %q", defaultRuntimeModel, model)
	}

	if err := svc.SetLocalRuntimeModel("llama3.1:8b"); err != nil {
		t.Fatalf("setting local runtime model: %v", err)
	}
	model, err = svc.GetLocalRuntimeModel()
	if err != nil {
		t.Fatalf("getting local runtime model after set: %v", err)
	}
	if model != "llama3.1:8b" {
		t.Fatalf("expected model llama3.1:8b, got %q", model)
	}

	if err := svc.SetLocalRuntimeModel("   "); err == nil {
		t.Fatal("expected validation error for empty local runtime model")
	}
	if err := svc.SetLocalRuntimeModel("qwen2.5:7b"); err == nil {
		t.Fatal("expected validation error for unsupported local runtime model")
	}
}

func TestLLMModeAndLocalRuntimeSettingsDatabaseErrors(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	if err := database.Close(); err != nil {
		t.Fatalf("closing DB: %v", err)
	}

	if _, err := svc.GetLLMMode(); err == nil {
		t.Fatal("expected GetLLMMode to fail on closed DB")
	}
	if err := svc.SetLLMMode("local"); err == nil {
		t.Fatal("expected SetLLMMode to fail on closed DB")
	}

	if _, err := svc.GetLocalRuntimeEngine(); err == nil {
		t.Fatal("expected GetLocalRuntimeEngine to fail on closed DB")
	}
	if err := svc.SetLocalRuntimeEngine(localruntime.EngineOllama); err == nil {
		t.Fatal("expected SetLocalRuntimeEngine to fail on closed DB")
	}

	if _, err := svc.GetLocalRuntimeModel(); err == nil {
		t.Fatal("expected GetLocalRuntimeModel to fail on closed DB")
	}
	if err := svc.SetLocalRuntimeModel("llama3.1:8b"); err == nil {
		t.Fatal("expected SetLocalRuntimeModel to fail on closed DB")
	}
}

func TestLLMProviderDefaultsAndLifecycle(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	provider, err := svc.GetLLMProvider()
	if err != nil {
		t.Fatalf("getting default llm provider: %v", err)
	}
	if provider != defaultLLMProvider {
		t.Fatalf("expected default provider %q, got %q", defaultLLMProvider, provider)
	}

	if err := svc.SetLLMProvider("openai"); err != nil {
		t.Fatalf("setting llm provider openai: %v", err)
	}
	provider, err = svc.GetLLMProvider()
	if err != nil {
		t.Fatalf("getting llm provider after set openai: %v", err)
	}
	if provider != "openai" {
		t.Fatalf("expected provider openai, got %q", provider)
	}

	if err := svc.SetLLMProvider("heuristic_v1"); err != nil {
		t.Fatalf("setting llm provider heuristic_v1: %v", err)
	}
	provider, err = svc.GetLLMProvider()
	if err != nil {
		t.Fatalf("getting llm provider after set heuristic_v1: %v", err)
	}
	if provider != "heuristic_v1" {
		t.Fatalf("expected provider heuristic_v1, got %q", provider)
	}

	if err := svc.SetLLMProvider("anthropic"); err == nil {
		t.Fatal("expected validation error for unsupported provider")
	}
}

func TestLLMProviderDatabaseErrors(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	if err := database.Close(); err != nil {
		t.Fatalf("closing DB: %v", err)
	}

	if _, err := svc.GetLLMProvider(); err == nil {
		t.Fatal("expected GetLLMProvider to fail on closed DB")
	}
	if err := svc.SetLLMProvider("openai"); err == nil {
		t.Fatal("expected SetLLMProvider to fail on closed DB")
	}
}

func TestLLMModelDefaultsAndLifecycle(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	model, err := svc.GetLLMModel()
	if err != nil {
		t.Fatalf("getting default llm model: %v", err)
	}
	if model != defaultLLMModel {
		t.Fatalf("expected default model %q, got %q", defaultLLMModel, model)
	}

	if err := svc.SetLLMModel("gpt-4o"); err != nil {
		t.Fatalf("setting llm model: %v", err)
	}
	model, err = svc.GetLLMModel()
	if err != nil {
		t.Fatalf("getting llm model after set: %v", err)
	}
	if model != "gpt-4o" {
		t.Fatalf("expected model gpt-4o, got %q", model)
	}

	if err := svc.SetLLMModel("   "); err == nil {
		t.Fatal("expected validation error for empty llm model")
	}
}

func TestLLMModelDatabaseErrors(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	if err := database.Close(); err != nil {
		t.Fatalf("closing DB: %v", err)
	}

	if _, err := svc.GetLLMModel(); err == nil {
		t.Fatal("expected GetLLMModel to fail on closed DB")
	}
	if err := svc.SetLLMModel("gpt-4o-mini"); err == nil {
		t.Fatal("expected SetLLMModel to fail on closed DB")
	}
}

func TestAutoPollingSettingsLifecycleAndValidation(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	enabled, err := svc.GetAutoPollingEnabled()
	if err != nil {
		t.Fatalf("getting default auto polling enabled: %v", err)
	}
	if enabled {
		t.Fatal("expected auto polling disabled by default")
	}

	intervalMinutes, err := svc.GetPollIntervalMinutes()
	if err != nil {
		t.Fatalf("getting default poll interval minutes: %v", err)
	}
	if intervalMinutes != defaultPollIntervalMin {
		t.Fatalf("expected default poll interval %d, got %d", defaultPollIntervalMin, intervalMinutes)
	}

	if err := svc.SetAutoPollingEnabled(true); err != nil {
		t.Fatalf("setting auto polling enabled true: %v", err)
	}
	enabled, err = svc.GetAutoPollingEnabled()
	if err != nil {
		t.Fatalf("getting auto polling enabled after set: %v", err)
	}
	if !enabled {
		t.Fatal("expected auto polling enabled=true after update")
	}

	if err := svc.SetPollIntervalMinutes(120); err != nil {
		t.Fatalf("setting poll interval minutes: %v", err)
	}
	intervalMinutes, err = svc.GetPollIntervalMinutes()
	if err != nil {
		t.Fatalf("getting poll interval minutes after set: %v", err)
	}
	if intervalMinutes != 120 {
		t.Fatalf("expected poll interval 120, got %d", intervalMinutes)
	}

	if err := svc.SetPollIntervalMinutes(minPollIntervalMinutes - 1); err == nil {
		t.Fatalf("expected validation error for interval below %d", minPollIntervalMinutes)
	}
	if err := svc.SetPollIntervalMinutes(maxPollIntervalMinutes + 1); err == nil {
		t.Fatalf("expected validation error for interval above %d", maxPollIntervalMinutes)
	}

	if err := database.SetSetting(context.Background(), settingAutoPollingEnabled, "unexpected"); err != nil {
		t.Fatalf("seeding invalid auto polling enabled value: %v", err)
	}
	enabled, err = svc.GetAutoPollingEnabled()
	if err != nil {
		t.Fatalf("getting auto polling enabled fallback: %v", err)
	}
	if enabled {
		t.Fatal("expected invalid auto polling enabled value to fallback to default false")
	}

	if err := database.SetSetting(context.Background(), settingPollIntervalMinutes, "not-a-number"); err != nil {
		t.Fatalf("seeding invalid poll interval value: %v", err)
	}
	intervalMinutes, err = svc.GetPollIntervalMinutes()
	if err != nil {
		t.Fatalf("getting poll interval fallback: %v", err)
	}
	if intervalMinutes != defaultPollIntervalMin {
		t.Fatalf("expected invalid poll interval to fallback to %d, got %d", defaultPollIntervalMin, intervalMinutes)
	}
}

func TestAutoPollingSettingsDatabaseErrors(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	if err := database.Close(); err != nil {
		t.Fatalf("closing DB: %v", err)
	}

	if _, err := svc.GetAutoPollingEnabled(); err == nil {
		t.Fatal("expected GetAutoPollingEnabled to fail on closed DB")
	}
	if err := svc.SetAutoPollingEnabled(false); err == nil {
		t.Fatal("expected SetAutoPollingEnabled to fail on closed DB")
	}
	if _, err := svc.GetPollIntervalMinutes(); err == nil {
		t.Fatal("expected GetPollIntervalMinutes to fail on closed DB")
	}
	if err := svc.SetPollIntervalMinutes(30); err == nil {
		t.Fatal("expected SetPollIntervalMinutes to fail on closed DB")
	}
}

func TestAutoMatchSettingsLifecycleAndValidation(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	enabled, err := svc.GetAutoMatchEnabled()
	if err != nil {
		t.Fatalf("getting default auto match enabled: %v", err)
	}
	if !enabled {
		t.Fatal("expected auto match enabled by default")
	}

	limit, err := svc.GetAutoMatchLimit()
	if err != nil {
		t.Fatalf("getting default auto match limit: %v", err)
	}
	if limit != 0 {
		t.Fatalf("expected default auto match limit 0 (unlimited), got %d", limit)
	}

	if err := svc.SetAutoMatchEnabled(false); err != nil {
		t.Fatalf("setting auto match enabled false: %v", err)
	}
	enabled, err = svc.GetAutoMatchEnabled()
	if err != nil {
		t.Fatalf("getting auto match enabled after set: %v", err)
	}
	if enabled {
		t.Fatal("expected auto match enabled=false after update")
	}

	if err := svc.SetAutoMatchLimit(15); err != nil {
		t.Fatalf("setting auto match limit: %v", err)
	}
	limit, err = svc.GetAutoMatchLimit()
	if err != nil {
		t.Fatalf("getting auto match limit after set: %v", err)
	}
	if limit != 15 {
		t.Fatalf("expected auto match limit 15, got %d", limit)
	}

	if err := svc.SetAutoMatchLimit(0); err != nil {
		t.Fatalf("setting auto match limit unlimited: %v", err)
	}
	limit, err = svc.GetAutoMatchLimit()
	if err != nil {
		t.Fatalf("getting auto match limit after unlimited set: %v", err)
	}
	if limit != 0 {
		t.Fatalf("expected auto match limit 0 after unlimited set, got %d", limit)
	}

	if err := svc.SetAutoMatchLimit(-1); err == nil {
		t.Fatal("expected validation error for negative auto match limit")
	}

	if err := database.SetSetting(context.Background(), settingAutoMatchEnabled, "unexpected"); err != nil {
		t.Fatalf("seeding invalid auto match enabled value: %v", err)
	}
	enabled, err = svc.GetAutoMatchEnabled()
	if err != nil {
		t.Fatalf("getting auto match enabled fallback: %v", err)
	}
	if !enabled {
		t.Fatal("expected invalid auto match enabled value to fallback to default true")
	}

	if err := database.SetSetting(context.Background(), settingAutoMatchLimit, "not-a-number"); err != nil {
		t.Fatalf("seeding invalid auto match limit value: %v", err)
	}
	limit, err = svc.GetAutoMatchLimit()
	if err != nil {
		t.Fatalf("getting auto match limit fallback: %v", err)
	}
	if limit != 0 {
		t.Fatalf("expected invalid auto match limit to fallback to 0, got %d", limit)
	}
}

func TestAutoMatchSettingsDatabaseErrors(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	if err := database.Close(); err != nil {
		t.Fatalf("closing DB: %v", err)
	}

	if _, err := svc.GetAutoMatchEnabled(); err == nil {
		t.Fatal("expected GetAutoMatchEnabled to fail on closed DB")
	}
	if err := svc.SetAutoMatchEnabled(true); err == nil {
		t.Fatal("expected SetAutoMatchEnabled to fail on closed DB")
	}
	if _, err := svc.GetAutoMatchLimit(); err == nil {
		t.Fatal("expected GetAutoMatchLimit to fail on closed DB")
	}
	if err := svc.SetAutoMatchLimit(10); err == nil {
		t.Fatal("expected SetAutoMatchLimit to fail on closed DB")
	}
}

func TestLLMBaseURLLifecycleAndValidation(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	baseURL, err := svc.GetLLMBaseURL()
	if err != nil {
		t.Fatalf("getting default llm base url: %v", err)
	}
	if baseURL != "" {
		t.Fatalf("expected empty default base url, got %q", baseURL)
	}

	if err := svc.SetLLMBaseURL("https://api.openai.com/v1"); err != nil {
		t.Fatalf("setting llm base url: %v", err)
	}
	baseURL, err = svc.GetLLMBaseURL()
	if err != nil {
		t.Fatalf("getting llm base url after set: %v", err)
	}
	if baseURL != "https://api.openai.com/v1" {
		t.Fatalf("expected base url https://api.openai.com/v1, got %q", baseURL)
	}

	if err := svc.SetLLMBaseURL(""); err != nil {
		t.Fatalf("clearing llm base url via empty value: %v", err)
	}
	baseURL, err = svc.GetLLMBaseURL()
	if err != nil {
		t.Fatalf("getting llm base url after clear: %v", err)
	}
	if baseURL != "" {
		t.Fatalf("expected empty base url after clear, got %q", baseURL)
	}

	if err := svc.SetLLMBaseURL("not-a-url"); err == nil {
		t.Fatal("expected validation error for invalid base url")
	}
}

func TestLLMBaseURLDatabaseErrors(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	if err := database.Close(); err != nil {
		t.Fatalf("closing DB: %v", err)
	}

	if _, err := svc.GetLLMBaseURL(); err == nil {
		t.Fatal("expected GetLLMBaseURL to fail on closed DB")
	}
	if err := svc.SetLLMBaseURL("https://example.com/v1"); err == nil {
		t.Fatal("expected SetLLMBaseURL to fail on closed DB")
	}
}

func TestCVPathLifecycleAndValidation(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	cvPath, err := svc.GetCVPath()
	if err != nil {
		t.Fatalf("getting default cv path: %v", err)
	}
	if cvPath != "" {
		t.Fatalf("expected empty default cv path, got %q", cvPath)
	}

	file := filepath.Join(t.TempDir(), "cv.txt")
	if err := os.WriteFile(file, []byte("Go backend engineer with 8 years of experience."), 0o600); err != nil {
		t.Fatalf("writing temp cv file: %v", err)
	}

	if err := svc.SetCVPath("  " + file + "  "); err != nil {
		t.Fatalf("setting cv path: %v", err)
	}
	cvPath, err = svc.GetCVPath()
	if err != nil {
		t.Fatalf("getting cv path after set: %v", err)
	}
	if cvPath != file {
		t.Fatalf("expected cv path %q, got %q", file, cvPath)
	}

	if err := svc.SetCVPath(""); err != nil {
		t.Fatalf("clearing cv path via empty value: %v", err)
	}
	cvPath, err = svc.GetCVPath()
	if err != nil {
		t.Fatalf("getting cv path after clear: %v", err)
	}
	if cvPath != "" {
		t.Fatalf("expected empty cv path after clear, got %q", cvPath)
	}

	if err := svc.SetCVPath(filepath.Join(t.TempDir(), "missing.txt")); err == nil {
		t.Fatal("expected validation error for missing cv file")
	}
	if err := svc.SetCVPath(t.TempDir()); err == nil {
		t.Fatal("expected validation error for cv directory path")
	}

	pdfPath := filepath.Join(t.TempDir(), "cv.pdf")
	if err := os.WriteFile(pdfPath, buildMinimalPDF("Go backend engineer"), 0o600); err != nil {
		t.Fatalf("writing pdf cv file: %v", err)
	}
	if err := svc.SetCVPath(pdfPath); err != nil {
		t.Fatalf("expected pdf cv path to be accepted, got %v", err)
	}

	unsupportedPath := filepath.Join(t.TempDir(), "cv.docx")
	if err := os.WriteFile(unsupportedPath, append([]byte("PK\x03\x04"), []byte("fake-docx")...), 0o600); err != nil {
		t.Fatalf("writing unsupported docx-like file: %v", err)
	}
	if err := svc.SetCVPath(unsupportedPath); err == nil {
		t.Fatal("expected unsupported docx-like cv to be rejected at submission")
	}
}

func TestCVPathDatabaseErrors(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	if err := database.Close(); err != nil {
		t.Fatalf("closing DB: %v", err)
	}

	if _, err := svc.GetCVPath(); err == nil {
		t.Fatal("expected GetCVPath to fail on closed DB")
	}
	if err := svc.SetCVPath(""); err == nil {
		t.Fatal("expected SetCVPath to fail on closed DB")
	}
}

func buildMinimalPDF(text string) []byte {
	escaped := escapePDFText(text)
	stream := fmt.Sprintf("BT\n/F1 12 Tf\n72 720 Td\n(%s) Tj\nET\n", escaped)

	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(stream), stream),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}

	xrefOffset := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n", len(objects)+1)
	out.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefOffset)

	return out.Bytes()
}

func escapePDFText(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "(", `\(`)
	value = strings.ReplaceAll(value, ")", `\)`)
	return value
}
