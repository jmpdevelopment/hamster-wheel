package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"hamster-wheel/internal/db"
	"hamster-wheel/internal/keychain"
	"hamster-wheel/internal/llm"
	"hamster-wheel/internal/llm/heuristic"
	"hamster-wheel/internal/llm/openai"
	"hamster-wheel/internal/localruntime"
	"hamster-wheel/internal/matcher"
)

const localProviderOllama = "local_ollama"

func newMatcherProviderResolver(
	database *db.DB,
	keychainStore keychain.Store,
	providers *llm.Registry,
	envOpenAIKey string,
	envOpenAIModel string,
	envOpenAIBaseURL string,
	envOllamaBaseURL string,
) matcher.ProviderResolver {
	envOpenAIKey = strings.TrimSpace(envOpenAIKey)
	envOpenAIModel = strings.TrimSpace(envOpenAIModel)
	envOpenAIBaseURL = strings.TrimSpace(envOpenAIBaseURL)
	envOllamaBaseURL = strings.TrimSpace(envOllamaBaseURL)
	if envOllamaBaseURL == "" {
		envOllamaBaseURL = localruntime.DefaultOllamaEndpoint
	}

	return func(ctx context.Context) (string, llm.Provider, error) {
		mode, err := database.GetSetting(ctx, settingLLMMode)
		if err != nil {
			return "", nil, fmt.Errorf("loading llm mode setting: %w", err)
		}
		mode = strings.TrimSpace(mode)
		if mode == "" {
			mode = defaultLLMMode
		}

		if mode == "local" {
			localModel, err := database.GetSetting(ctx, settingLocalRuntimeModel)
			if err != nil {
				return localProviderOllama, nil, fmt.Errorf("loading local runtime model setting: %w", err)
			}
			localModel = strings.TrimSpace(localModel)
			if localModel == "" {
				localModel = defaultRuntimeModel
			}

			return localProviderOllama, openai.New(openai.Config{
				APIKey:  "",
				Model:   localModel,
				BaseURL: envOllamaBaseURL,
			}), nil
		}

		providerName, err := database.GetSetting(ctx, settingLLMProvider)
		if err != nil {
			return "", nil, fmt.Errorf("loading llm provider setting: %w", err)
		}
		providerName = strings.TrimSpace(providerName)
		if providerName == "" {
			providerName = heuristic.ProviderName
		}

		switch providerName {
		case openai.ProviderName:
			model, err := database.GetSetting(ctx, settingLLMModel)
			if err != nil {
				return providerName, nil, fmt.Errorf("loading llm model setting: %w", err)
			}
			model = strings.TrimSpace(model)
			if model == "" {
				model = envOpenAIModel
			}

			baseURL, err := database.GetSetting(ctx, settingLLMBaseURL)
			if err != nil {
				return providerName, nil, fmt.Errorf("loading llm base url setting: %w", err)
			}
			baseURL = strings.TrimSpace(baseURL)
			if baseURL == "" {
				baseURL = envOpenAIBaseURL
			}

			openAIKey, err := keychainStore.Get(settingOpenAIAPIKey)
			if err != nil {
				slog.Error("failed to load OpenAI API key from keychain", "error", err)
			}
			openAIKey = strings.TrimSpace(openAIKey)
			if openAIKey == "" {
				openAIKey = envOpenAIKey
			}

			return providerName, openai.New(openai.Config{
				APIKey:  openAIKey,
				Model:   model,
				BaseURL: baseURL,
			}), nil
		default:
			provider, ok := providers.Get(providerName)
			if !ok {
				return providerName, nil, fmt.Errorf("provider %q is not registered", providerName)
			}
			return providerName, provider, nil
		}
	}
}
