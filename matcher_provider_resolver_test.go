package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"hamster-wheel/internal/db"
	"hamster-wheel/internal/keychain"
	"hamster-wheel/internal/llm/openai"
	"hamster-wheel/internal/localruntime"
)

func testResolverDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.OpenAt(filepath.Join(t.TempDir(), "resolver-test.db"))
	if err != nil {
		t.Fatalf("opening resolver test db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

type stubResolverRuntimeManager struct {
	status      localruntime.Snapshot
	statusErr   error
	start       localruntime.Snapshot
	startErr    error
	startCalled bool
}

func (m *stubResolverRuntimeManager) Status(context.Context) (localruntime.Snapshot, error) {
	if m.statusErr != nil {
		return localruntime.Snapshot{}, m.statusErr
	}
	return m.status, nil
}

func (m *stubResolverRuntimeManager) Start(context.Context) (localruntime.Snapshot, error) {
	m.startCalled = true
	if m.startErr != nil {
		return localruntime.Snapshot{}, m.startErr
	}
	return m.start, nil
}

func (*stubResolverRuntimeManager) Stop(context.Context) (localruntime.Snapshot, error) {
	return localruntime.Snapshot{Status: localruntime.StatusStopped}, nil
}

func (*stubResolverRuntimeManager) ListModels(context.Context) (localruntime.ModelCatalog, error) {
	return localruntime.ModelCatalog{}, nil
}

func (*stubResolverRuntimeManager) PullModel(context.Context, string) (localruntime.PullResult, error) {
	return localruntime.PullResult{}, nil
}

func (*stubResolverRuntimeManager) GetPullProgress(context.Context) (localruntime.PullProgress, error) {
	return localruntime.PullProgress{}, nil
}

func TestMatcherProviderResolverDefaultsToOpenAIWhenUnset(t *testing.T) {
	database := testResolverDB(t)
	resolver := newMatcherProviderResolver(
		database,
		keychain.NewMemoryStore(),
		"",
		"",
		"",
		"",
	)

	name, provider, err := resolver(context.Background())
	if err != nil {
		t.Fatalf("resolving provider: %v", err)
	}
	if name != openai.ProviderName {
		t.Fatalf("expected provider name %q, got %q", openai.ProviderName, name)
	}
	if provider == nil || provider.Name() != openai.ProviderName {
		t.Fatalf("expected openai provider instance, got %+v", provider)
	}
}

func TestMatcherProviderResolverBuildsOpenAIProviderFromSettingsAndKeychain(t *testing.T) {
	database := testResolverDB(t)
	if err := database.SetSetting(context.Background(), settingLLMModel, "gpt-4o-mini"); err != nil {
		t.Fatalf("setting llm model: %v", err)
	}

	kc := keychain.NewMemoryStore()
	if err := kc.Set(settingOpenAIAPIKey, "sk-test-key"); err != nil {
		t.Fatalf("setting openai key in keychain: %v", err)
	}

	resolver := newMatcherProviderResolver(
		database,
		kc,
		"",
		"",
		"",
		"",
	)

	name, provider, err := resolver(context.Background())
	if err != nil {
		t.Fatalf("resolving provider: %v", err)
	}
	if name != openai.ProviderName {
		t.Fatalf("expected provider name %q, got %q", openai.ProviderName, name)
	}
	if provider == nil || provider.Name() != openai.ProviderName {
		t.Fatalf("expected openai provider instance, got %+v", provider)
	}
}

func TestMatcherProviderResolverBuildsLocalProviderFromModeAndRuntimeSettings(t *testing.T) {
	database := testResolverDB(t)
	if err := database.SetSetting(context.Background(), settingLLMMode, "local"); err != nil {
		t.Fatalf("setting llm mode: %v", err)
	}
	if err := database.SetSetting(context.Background(), settingLocalRuntimeModel, "llama3.1:8b"); err != nil {
		t.Fatalf("setting local runtime model: %v", err)
	}

	resolver := newMatcherProviderResolver(
		database,
		keychain.NewMemoryStore(),
		"",
		"",
		"",
		"http://localhost:11434",
	)

	name, provider, err := resolver(context.Background())
	if err != nil {
		t.Fatalf("resolving provider: %v", err)
	}
	if name != localProviderOllama {
		t.Fatalf("expected provider label %q, got %q", localProviderOllama, name)
	}
	if provider == nil || provider.Name() != openai.ProviderName {
		t.Fatalf("expected openai provider instance for local mode, got %+v", provider)
	}
}

func TestMatcherProviderResolverStartsRuntimeWhenLocalModeIsNotReady(t *testing.T) {
	database := testResolverDB(t)
	if err := database.SetSetting(context.Background(), settingLLMMode, "local"); err != nil {
		t.Fatalf("setting llm mode: %v", err)
	}
	runtimeManager := &stubResolverRuntimeManager{
		status: localruntime.Snapshot{Status: localruntime.StatusStopped},
		start:  localruntime.Snapshot{Status: localruntime.StatusReady},
	}

	resolver := newMatcherProviderResolver(
		database,
		keychain.NewMemoryStore(),
		"",
		"",
		"",
		"http://localhost:11434",
		runtimeManager,
	)

	if _, _, err := resolver(context.Background()); err != nil {
		t.Fatalf("resolving provider: %v", err)
	}
	if !runtimeManager.startCalled {
		t.Fatal("expected resolver to attempt starting local runtime")
	}
}

func TestMatcherProviderResolverReturnsErrorWhenLocalRuntimeMissing(t *testing.T) {
	database := testResolverDB(t)
	if err := database.SetSetting(context.Background(), settingLLMMode, "local"); err != nil {
		t.Fatalf("setting llm mode: %v", err)
	}
	runtimeManager := &stubResolverRuntimeManager{
		status: localruntime.Snapshot{Status: localruntime.StatusNotInstalled},
	}

	resolver := newMatcherProviderResolver(
		database,
		keychain.NewMemoryStore(),
		"",
		"",
		"",
		"http://localhost:11434",
		runtimeManager,
	)

	_, _, err := resolver(context.Background())
	if err == nil {
		t.Fatal("expected local runtime missing error")
	}
	if !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("expected missing runtime message, got %v", err)
	}
}
