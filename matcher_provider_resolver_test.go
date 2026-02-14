package main

import (
	"context"
	"path/filepath"
	"testing"

	"hamster-wheel/internal/db"
	"hamster-wheel/internal/keychain"
	"hamster-wheel/internal/llm"
	"hamster-wheel/internal/llm/heuristic"
	"hamster-wheel/internal/llm/openai"
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

func testResolverRegistry(t *testing.T) *llm.Registry {
	t.Helper()
	registry := llm.NewRegistry()
	if err := registry.Register(heuristic.New()); err != nil {
		t.Fatalf("registering heuristic provider: %v", err)
	}
	return registry
}

func TestMatcherProviderResolverDefaultsToHeuristicWhenUnset(t *testing.T) {
	database := testResolverDB(t)
	resolver := newMatcherProviderResolver(
		database,
		keychain.NewMemoryStore(),
		testResolverRegistry(t),
		"",
		"",
		"",
	)

	name, provider, err := resolver(context.Background())
	if err != nil {
		t.Fatalf("resolving provider: %v", err)
	}
	if name != heuristic.ProviderName {
		t.Fatalf("expected provider name %q, got %q", heuristic.ProviderName, name)
	}
	if provider == nil || provider.Name() != heuristic.ProviderName {
		t.Fatalf("expected heuristic provider instance, got %+v", provider)
	}
}

func TestMatcherProviderResolverBuildsOpenAIProviderFromSettingsAndKeychain(t *testing.T) {
	database := testResolverDB(t)
	if err := database.SetSetting(context.Background(), settingLLMProvider, openai.ProviderName); err != nil {
		t.Fatalf("setting llm provider: %v", err)
	}
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
		testResolverRegistry(t),
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
