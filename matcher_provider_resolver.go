package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"hamster-wheel/internal/db"
	"hamster-wheel/internal/keychain"
	"hamster-wheel/internal/llm"
	"hamster-wheel/internal/llm/openai"
	"hamster-wheel/internal/localruntime"
	"hamster-wheel/internal/matcher"
)

const localProviderOllama = "local_ollama"

const localProviderMatchTimeout = 90 * time.Second

func newMatcherProviderResolver(
	database *db.DB,
	keychainStore keychain.Store,
	envOpenAIKey string,
	envOpenAIModel string,
	envOpenAIBaseURL string,
	envOllamaBaseURL string,
	localRuntimeManagers ...localruntime.Manager,
) matcher.ProviderResolver {
	envOpenAIKey = strings.TrimSpace(envOpenAIKey)
	envOpenAIModel = strings.TrimSpace(envOpenAIModel)
	envOpenAIBaseURL = strings.TrimSpace(envOpenAIBaseURL)
	envOllamaBaseURL = strings.TrimSpace(envOllamaBaseURL)
	if envOllamaBaseURL == "" {
		envOllamaBaseURL = localruntime.DefaultOllamaEndpoint
	}
	var localRuntimeManager localruntime.Manager
	if len(localRuntimeManagers) > 0 {
		localRuntimeManager = localRuntimeManagers[0]
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
			if localModel == "" || localModel != defaultRuntimeModel {
				localModel = defaultRuntimeModel
			}
			if localRuntimeManager != nil {
				snapshot, statusErr := localRuntimeManager.Status(ctx)
				if statusErr != nil {
					return localProviderOllama, nil, fmt.Errorf("checking local runtime status: %w", statusErr)
				}
				switch snapshot.Status {
				case localruntime.StatusReady:
				case localruntime.StatusStopped, localruntime.StatusStarting, localruntime.StatusError:
					startedSnapshot, startErr := localRuntimeManager.Start(ctx)
					if startErr != nil {
						return localProviderOllama, nil, fmt.Errorf("starting local runtime for local mode: %w", startErr)
					}
					if startedSnapshot.Status != localruntime.StatusReady {
						return localProviderOllama, nil, fmt.Errorf(
							"local runtime is not ready (%s): open Settings > LLM Providers > Local to finish setup",
							startedSnapshot.Status,
						)
					}
				case localruntime.StatusNotInstalled:
					return localProviderOllama, nil, errors.New("local runtime is not installed: install Ollama in Settings > LLM Providers > Local")
				default:
					return localProviderOllama, nil, fmt.Errorf("local runtime is not ready (%s)", snapshot.Status)
				}
			}

			return localProviderOllama, openai.New(openai.Config{
				APIKey:  "",
				Model:   localModel,
				BaseURL: envOllamaBaseURL,
				HTTPClient: &http.Client{
					Timeout: localProviderMatchTimeout,
				},
			}), nil
		}

		model, err := database.GetSetting(ctx, settingLLMModel)
		if err != nil {
			return openai.ProviderName, nil, fmt.Errorf("loading llm model setting: %w", err)
		}
		model = strings.TrimSpace(model)
		if model == "" {
			model = envOpenAIModel
		}

		openAIKey, err := keychainStore.Get(settingOpenAIAPIKey)
		if err != nil {
			slog.Error("failed to load OpenAI API key from keychain", "error", err)
		}
		openAIKey = strings.TrimSpace(openAIKey)
		if openAIKey == "" {
			openAIKey = envOpenAIKey
		}

		return openai.ProviderName, openai.New(openai.Config{
			APIKey:  openAIKey,
			Model:   model,
			BaseURL: envOpenAIBaseURL,
		}), nil
	}
}
