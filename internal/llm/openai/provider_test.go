package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hamster-wheel/internal/llm"
)

func TestMatchSuccess(t *testing.T) {
	var capturedRequest chatCompletionRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("expected /v1/chat/completions, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("expected auth header, got %q", got)
		}

		if err := json.NewDecoder(r.Body).Decode(&capturedRequest); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"message": {
						"content": "{\"score\":0.82,\"summary\":\"Strong overlap with backend Go API requirements.\"}"
					}
				}
			],
			"usage": {"prompt_tokens": 123}
		}`))
	}))
	defer server.Close()

	provider := New(Config{
		APIKey:     "test-key",
		Model:      "gpt-4o-mini",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	result, err := provider.Match(context.Background(), llm.MatchRequest{
		Query:               "go backend apis",
		JobTitle:            "Senior Go Backend Engineer",
		JobCompany:          "Acme",
		JobLocation:         "Remote",
		JobDescription:      "Build and maintain distributed backend API services in Go.",
		MaxDescriptionRunes: 1400,
	})
	if err != nil {
		t.Fatalf("matching job: %v", err)
	}

	if result.Score != 0.82 {
		t.Fatalf("expected score 0.82, got %.2f", result.Score)
	}
	if result.Summary == "" {
		t.Fatal("expected non-empty summary")
	}
	if result.EstimatedPromptTokens != 123 {
		t.Fatalf("expected prompt tokens 123, got %d", result.EstimatedPromptTokens)
	}
	if capturedRequest.Model != "gpt-4o-mini" {
		t.Fatalf("expected model gpt-4o-mini, got %q", capturedRequest.Model)
	}
	if capturedRequest.ResponseFormat == nil || capturedRequest.ResponseFormat.Type != "json_object" {
		t.Fatalf("expected response_format json_object, got %+v", capturedRequest.ResponseFormat)
	}
}

func TestMatchMalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"choices": [{"message": {"content": "not valid json"}}],
			"usage": {"prompt_tokens": 11}
		}`))
	}))
	defer server.Close()

	provider := New(Config{
		APIKey:     "test-key",
		Model:      "gpt-4o-mini",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	_, err := provider.Match(context.Background(), llm.MatchRequest{
		Query:          "go backend",
		JobTitle:       "Go Engineer",
		JobDescription: "Build APIs",
	})
	if err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if !errors.Is(err, ErrMalformedResponse) {
		t.Fatalf("expected ErrMalformedResponse, got %v", err)
	}
}

func TestMatchTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(150 * time.Millisecond)
		_, _ = w.Write([]byte(`{
			"choices": [{"message": {"content": "{\"score\":0.5,\"summary\":\"ok\"}"}}]
		}`))
	}))
	defer server.Close()

	provider := New(Config{
		APIKey:     "test-key",
		Model:      "gpt-4o-mini",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := provider.Match(ctx, llm.MatchRequest{
		Query:          "go",
		JobTitle:       "Go Engineer",
		JobDescription: "Build APIs",
	})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("expected ErrTimeout, got %v", err)
	}
}

func TestMatchUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{
			"error": {"message": "invalid API key"}
		}`))
	}))
	defer server.Close()

	provider := New(Config{
		APIKey:     "bad-key",
		Model:      "gpt-4o-mini",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	_, err := provider.Match(context.Background(), llm.MatchRequest{
		Query:          "go",
		JobTitle:       "Go Engineer",
		JobDescription: "Build APIs",
	})
	if err == nil {
		t.Fatal("expected auth error, got nil")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestValidateSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"choices": [{"message": {"content": "{\"ok\":true}"}}],
			"usage": {"prompt_tokens": 2}
		}`))
	}))
	defer server.Close()

	provider := New(Config{
		APIKey:     "test-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	if err := provider.Validate(context.Background()); err != nil {
		t.Fatalf("validating provider: %v", err)
	}
}

func TestValidateNotConfigured(t *testing.T) {
	provider := New(Config{
		APIKey: "",
	})

	err := provider.Validate(context.Background())
	if err == nil {
		t.Fatal("expected validation error for missing API key")
	}
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}

func TestValidateAllowsNoAPIKeyForNonOpenAIHost(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if got := strings.TrimSpace(r.Header.Get("Authorization")); got != "" {
				t.Fatalf("expected no Authorization header, got %q", got)
			}
			return jsonResponse(http.StatusOK, `{
				"choices": [{"message": {"content": "{\"ok\":true}"}}],
				"usage": {"prompt_tokens": 2}
			}`), nil
		}),
	}

	provider := New(Config{
		APIKey:     "",
		Model:      "llama3.1:8b",
		BaseURL:    "http://localhost:11434",
		HTTPClient: client,
	})

	if err := provider.Validate(context.Background()); err != nil {
		t.Fatalf("expected keyless validation for local endpoint to succeed, got %v", err)
	}
}

func TestMatchOmitsAuthorizationForKeylessLocalEndpoint(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if got := strings.TrimSpace(r.Header.Get("Authorization")); got != "" {
				t.Fatalf("expected no Authorization header, got %q", got)
			}
			if got := r.URL.Path; got != "/v1/chat/completions" {
				t.Fatalf("expected /v1/chat/completions, got %q", got)
			}
			return jsonResponse(http.StatusOK, `{
				"choices": [{"message": {"content": "{\"score\":0.75,\"summary\":\"Good alignment.\"}"}}],
				"usage": {"prompt_tokens": 9}
			}`), nil
		}),
	}

	provider := New(Config{
		APIKey:     "",
		Model:      "llama3.1:8b",
		BaseURL:    "http://localhost:11434",
		HTTPClient: client,
	})

	result, err := provider.Match(context.Background(), llm.MatchRequest{
		Query:          "backend go",
		JobTitle:       "Go Engineer",
		JobDescription: "Build APIs",
	})
	if err != nil {
		t.Fatalf("matching with keyless local endpoint: %v", err)
	}
	if result.Score != 0.75 {
		t.Fatalf("expected score 0.75, got %.2f", result.Score)
	}
}

func TestNewDefaultsAndMetadata(t *testing.T) {
	provider := New(Config{
		APIKey: "test-key",
	})

	if provider.model != defaultModel {
		t.Fatalf("expected default model %q, got %q", defaultModel, provider.model)
	}
	if provider.baseURL != defaultBaseURL {
		t.Fatalf("expected default base URL %q, got %q", defaultBaseURL, provider.baseURL)
	}
	if provider.httpClient == nil {
		t.Fatal("expected default http client")
	}
	if provider.Name() != ProviderName {
		t.Fatalf("expected provider name %q, got %q", ProviderName, provider.Name())
	}
	if provider.DisplayName() != "OpenAI" {
		t.Fatalf("expected display name OpenAI, got %q", provider.DisplayName())
	}
}

func TestCheckConfiguredMissingModel(t *testing.T) {
	provider := &Provider{
		apiKey: "test-key",
		model:  "",
	}
	err := provider.checkConfigured()
	if err == nil {
		t.Fatal("expected missing-model error")
	}
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}

func TestUsesOpenAICloudHost(t *testing.T) {
	cases := []struct {
		baseURL string
		want    bool
	}{
		{baseURL: "https://api.openai.com", want: true},
		{baseURL: "https://api.openai.com/v1", want: true},
		{baseURL: "https://openai.com", want: true},
		{baseURL: "https://localhost:11434", want: false},
		{baseURL: "http://127.0.0.1:1234/v1", want: false},
		{baseURL: "not-a-url", want: true},
	}

	for _, tc := range cases {
		got := usesOpenAICloudHost(tc.baseURL)
		if got != tc.want {
			t.Fatalf("usesOpenAICloudHost(%q) expected %v, got %v", tc.baseURL, tc.want, got)
		}
	}
}

func TestIsRetryableTransportError(t *testing.T) {
	if isRetryableTransportError(errors.New("dial tcp: connection refused")) != true {
		t.Fatal("expected connection refused to be retryable")
	}
	if isRetryableTransportError(errors.New("network is unreachable")) != true {
		t.Fatal("expected network unreachable to be retryable")
	}
	if isRetryableTransportError(context.Canceled) != false {
		t.Fatal("expected context cancellation to be non-retryable")
	}
	if isRetryableTransportError(errors.New("unauthorized")) != false {
		t.Fatal("expected generic non-network error to be non-retryable")
	}
}

func TestClassifyStatusErrorBranches(t *testing.T) {
	t.Run("rate limited", func(t *testing.T) {
		err := classifyStatusError(http.StatusTooManyRequests, []byte(`{"error":{"message":"slow down"}}`))
		if !errors.Is(err, ErrRateLimited) {
			t.Fatalf("expected ErrRateLimited, got %v", err)
		}
	})

	t.Run("upstream failure", func(t *testing.T) {
		err := classifyStatusError(http.StatusBadGateway, []byte(`{"error":{"message":"gateway"}}`))
		if !errors.Is(err, ErrUpstreamFailure) {
			t.Fatalf("expected ErrUpstreamFailure, got %v", err)
		}
	})

	t.Run("timeout status", func(t *testing.T) {
		err := classifyStatusError(http.StatusGatewayTimeout, []byte(`{"error":{"message":"timeout"}}`))
		if !errors.Is(err, ErrTimeout) {
			t.Fatalf("expected ErrTimeout, got %v", err)
		}
	})

	t.Run("generic client error with message", func(t *testing.T) {
		err := classifyStatusError(http.StatusBadRequest, []byte(`{"error":{"message":"bad request"}}`))
		if err == nil {
			t.Fatal("expected error")
		}
		if errors.Is(err, ErrUnauthorized) || errors.Is(err, ErrTimeout) || errors.Is(err, ErrRateLimited) || errors.Is(err, ErrUpstreamFailure) {
			t.Fatalf("expected generic status error, got classified %v", err)
		}
		if !strings.Contains(err.Error(), "bad request") {
			t.Fatalf("expected error message in output, got %v", err)
		}
	})
}

func TestClassifyTransportErrorBranches(t *testing.T) {
	ctxDeadline, cancelDeadline := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancelDeadline()
	time.Sleep(2 * time.Millisecond)

	err := classifyTransportError(ctxDeadline, context.DeadlineExceeded)
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("expected ErrTimeout for deadline exceeded, got %v", err)
	}

	err = classifyTransportError(context.Background(), timeoutNetErr{})
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("expected ErrTimeout for net timeout, got %v", err)
	}

	err = classifyTransportError(context.Background(), context.Canceled)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled passthrough, got %v", err)
	}

	err = classifyTransportError(context.Background(), errors.New("tcp reset"))
	if err == nil || !strings.Contains(err.Error(), "transport failed") {
		t.Fatalf("expected generic transport failure, got %v", err)
	}
}

func TestStatusErrorWithoutMessage(t *testing.T) {
	err := statusError(ErrUnauthorized, http.StatusUnauthorized, "")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
	if !strings.Contains(err.Error(), "status 401") {
		t.Fatalf("expected status code in message, got %v", err)
	}
}

func TestExtractErrorMessageFallback(t *testing.T) {
	if got := extractErrorMessage([]byte(`{"error":{"message":"invalid key"}}`)); got != "invalid key" {
		t.Fatalf("expected invalid key message, got %q", got)
	}
	if got := extractErrorMessage([]byte(`{"not_error":true}`)); got != "" {
		t.Fatalf("expected empty message for missing error envelope, got %q", got)
	}
	if got := extractErrorMessage([]byte(`not-json`)); got != "" {
		t.Fatalf("expected empty message for invalid json, got %q", got)
	}
}

func TestParseMatchContentBranches(t *testing.T) {
	t.Run("accepts fenced payload", func(t *testing.T) {
		parsed, err := parseMatchContent("```json\n{\"score\":0.5,\"summary\":\"ok\"}\n```")
		if err != nil {
			t.Fatalf("parsing fenced payload: %v", err)
		}
		if parsed.Score != 0.5 || parsed.Summary != "ok" {
			t.Fatalf("unexpected parsed result: %+v", parsed)
		}
	})

	t.Run("rejects missing score", func(t *testing.T) {
		_, err := parseMatchContent(`{"summary":"ok"}`)
		if err == nil || !errors.Is(err, ErrMalformedResponse) {
			t.Fatalf("expected malformed error for missing score, got %v", err)
		}
	})

	t.Run("rejects non numeric score", func(t *testing.T) {
		_, err := parseMatchContent(`{"score":"high","summary":"ok"}`)
		if err == nil || !errors.Is(err, ErrMalformedResponse) {
			t.Fatalf("expected malformed error for non-numeric score, got %v", err)
		}
	})

	t.Run("rejects out of range score", func(t *testing.T) {
		_, err := parseMatchContent(`{"score":1.2,"summary":"ok"}`)
		if err == nil || !errors.Is(err, ErrMalformedResponse) {
			t.Fatalf("expected malformed error for out-of-range score, got %v", err)
		}
	})

	t.Run("rejects empty summary", func(t *testing.T) {
		_, err := parseMatchContent(`{"score":0.2,"summary":"   "}`)
		if err == nil || !errors.Is(err, ErrMalformedResponse) {
			t.Fatalf("expected malformed error for empty summary, got %v", err)
		}
	})

	t.Run("truncates long summary", func(t *testing.T) {
		longSummary := strings.Repeat("a", defaultMaxSummaryRunes+20)
		parsed, err := parseMatchContent(fmt.Sprintf(`{"score":0.2,"summary":"%s"}`, longSummary))
		if err != nil {
			t.Fatalf("expected long summary to be truncated, got error: %v", err)
		}
		if len([]rune(parsed.Summary)) != defaultMaxSummaryRunes {
			t.Fatalf("expected truncated summary length %d, got %d", defaultMaxSummaryRunes, len([]rune(parsed.Summary)))
		}
	})
}

func TestExtractJSONObjectAndStripCodeFence(t *testing.T) {
	_, err := extractJSONObject("no braces here")
	if err == nil || !errors.Is(err, ErrMalformedResponse) {
		t.Fatalf("expected malformed error for missing object, got %v", err)
	}

	fenced := "```json\n{\"score\":0.1,\"summary\":\"ok\"}\n```"
	trimmed := stripCodeFence(fenced)
	if trimmed != "{\"score\":0.1,\"summary\":\"ok\"}" {
		t.Fatalf("expected stripped code fence, got %q", trimmed)
	}

	plain := stripCodeFence("{\"score\":0.1}")
	if plain != "{\"score\":0.1}" {
		t.Fatalf("expected unchanged plain payload, got %q", plain)
	}
}

func TestPromptAndTokenHelpers(t *testing.T) {
	systemPrompt := buildMatchSystemPrompt(defaultMaxSummaryRunes)
	if !strings.Contains(systemPrompt, fmt.Sprintf("at most %d characters", defaultMaxSummaryRunes)) {
		t.Fatalf("expected system prompt to include summary limit, got %q", systemPrompt)
	}

	prompt := buildMatchUserPrompt(llm.MatchRequest{
		Query:            "",
		CandidateProfile: "",
		JobTitle:         "",
		JobCompany:       "",
		JobLocation:      "",
		JobDescription:   "",
	}, "", "")
	if !strings.Contains(prompt, "(empty)") {
		t.Fatalf("expected placeholder for empty values, got %q", prompt)
	}
	if !strings.Contains(prompt, "Candidate CV profile:\n(empty)") {
		t.Fatalf("expected prompt to include empty CV profile section, got %q", prompt)
	}

	if estimatePromptTokens("") != 0 {
		t.Fatalf("expected 0 tokens for empty content")
	}
	if truncateRunes("abcdef", 3) != "abc" {
		t.Fatalf("expected truncated runes")
	}
	if truncateRunes("abc", 0) != "abc" {
		t.Fatalf("expected unchanged string for non-positive max")
	}
}

func TestChatCompletionResponseEdgeCases(t *testing.T) {
	t.Run("invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`not-json`))
		}))
		defer server.Close()

		provider := New(Config{
			APIKey:     "test-key",
			BaseURL:    server.URL,
			HTTPClient: server.Client(),
		})

		_, _, err := provider.chatCompletion(context.Background(), chatCompletionRequest{
			Model: "gpt-4o-mini",
			Messages: []chatCompletionMessage{
				{Role: "user", Content: "hi"},
			},
		})
		if err == nil || !errors.Is(err, ErrMalformedResponse) {
			t.Fatalf("expected malformed response error, got %v", err)
		}
	})

	t.Run("missing choices", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"choices":[],"usage":{"prompt_tokens":7}}`))
		}))
		defer server.Close()

		provider := New(Config{
			APIKey:     "test-key",
			BaseURL:    server.URL,
			HTTPClient: server.Client(),
		})

		_, _, err := provider.chatCompletion(context.Background(), chatCompletionRequest{
			Model: "gpt-4o-mini",
			Messages: []chatCompletionMessage{
				{Role: "user", Content: "hi"},
			},
		})
		if err == nil || !errors.Is(err, ErrMalformedResponse) {
			t.Fatalf("expected malformed response error, got %v", err)
		}
	})

	t.Run("empty content", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"   "}}],"usage":{"prompt_tokens":7}}`))
		}))
		defer server.Close()

		provider := New(Config{
			APIKey:     "test-key",
			BaseURL:    server.URL,
			HTTPClient: server.Client(),
		})

		_, _, err := provider.chatCompletion(context.Background(), chatCompletionRequest{
			Model: "gpt-4o-mini",
			Messages: []chatCompletionMessage{
				{Role: "user", Content: "hi"},
			},
		})
		if err == nil || !errors.Is(err, ErrMalformedResponse) {
			t.Fatalf("expected malformed response error, got %v", err)
		}
	})
}

type timeoutNetErr struct{}

func (timeoutNetErr) Error() string   { return "i/o timeout" }
func (timeoutNetErr) Timeout() bool   { return true }
func (timeoutNetErr) Temporary() bool { return true }

var _ net.Error = timeoutNetErr{}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
