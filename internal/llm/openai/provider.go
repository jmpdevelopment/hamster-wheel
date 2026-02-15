package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"hamster-wheel/internal/llm"
)

const (
	ProviderName = "openai"

	defaultModel               = "gpt-4o-mini"
	defaultBaseURL             = "https://api.openai.com"
	defaultRequestTimeout      = 20 * time.Second
	defaultMaxDescriptionRunes = 1400
	defaultMaxProfileRunes     = 2000
	defaultMaxSummaryRunes     = 120
	maxResponseBytes           = 1 << 20 // 1 MiB
	localRetryDelay            = 250 * time.Millisecond
)

var (
	ErrNotConfigured     = errors.New("llm endpoint provider is not configured")
	ErrUnauthorized      = errors.New("llm endpoint request unauthorized")
	ErrTimeout           = errors.New("llm endpoint request timed out")
	ErrMalformedResponse = errors.New("llm endpoint returned malformed response")
	ErrRateLimited       = errors.New("llm endpoint rate limited")
	ErrUpstreamFailure   = errors.New("llm endpoint upstream failure")
)

const matchSystemPromptTemplate = "You score job relevance for one candidate query. " +
	"Return JSON only with keys score and summary. " +
	"score must be a number between 0 and 1 inclusive. " +
	"summary must be one plain-text sentence that helps a user understand why the score was assigned. " +
	"Include the strongest fit or mismatch signal in concise wording. " +
	"Aim for 90-120 characters and never exceed %d characters. " +
	"No markdown, lists, or newlines."

const validateSystemPrompt = "Return JSON only: {\"ok\": true}."

// Config controls OpenAI provider behavior.
type Config struct {
	APIKey     string
	Model      string
	BaseURL    string
	HTTPClient *http.Client
}

// Provider scores job matches via OpenAI chat completions API.
// The same provider can target OpenAI-compatible endpoints via BaseURL.
type Provider struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

func New(cfg Config) *Provider {
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = defaultModel
	}

	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: defaultRequestTimeout,
		}
	}

	return &Provider{
		apiKey:     strings.TrimSpace(cfg.APIKey),
		model:      model,
		baseURL:    baseURL,
		httpClient: client,
	}
}

func (p *Provider) Name() string {
	return ProviderName
}

func (p *Provider) DisplayName() string {
	return "OpenAI"
}

func (p *Provider) Validate(ctx context.Context) error {
	if err := p.checkConfigured(); err != nil {
		return err
	}

	_, _, err := p.chatCompletion(ctx, chatCompletionRequest{
		Model:       p.model,
		MaxTokens:   5,
		Temperature: 0,
		ResponseFormat: &chatCompletionResponseFormat{
			Type: "json_object",
		},
		Messages: []chatCompletionMessage{
			{
				Role:    "system",
				Content: validateSystemPrompt,
			},
			{
				Role:    "user",
				Content: "Connectivity check.",
			},
		},
	})
	if err != nil {
		return fmt.Errorf("validating llm endpoint provider: %w", err)
	}

	return nil
}

func (p *Provider) Match(ctx context.Context, req llm.MatchRequest) (llm.MatchResult, error) {
	if err := p.checkConfigured(); err != nil {
		return llm.MatchResult{}, err
	}

	maxDescriptionRunes := req.MaxDescriptionRunes
	if maxDescriptionRunes <= 0 {
		maxDescriptionRunes = defaultMaxDescriptionRunes
	}

	description := truncateRunes(strings.TrimSpace(req.JobDescription), maxDescriptionRunes)
	profile := truncateRunes(strings.TrimSpace(req.CandidateProfile), defaultMaxProfileRunes)
	matchSystemPrompt := buildMatchSystemPrompt(defaultMaxSummaryRunes)
	userPrompt := buildMatchUserPrompt(req, description, profile)

	content, promptTokens, err := p.chatCompletion(ctx, chatCompletionRequest{
		Model:       p.model,
		MaxTokens:   180,
		Temperature: 0,
		ResponseFormat: &chatCompletionResponseFormat{
			Type: "json_object",
		},
		Messages: []chatCompletionMessage{
			{
				Role:    "system",
				Content: matchSystemPrompt,
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
	})
	if err != nil {
		return llm.MatchResult{}, fmt.Errorf("requesting match from llm endpoint: %w", err)
	}

	parsed, err := parseMatchContent(content)
	if err != nil {
		return llm.MatchResult{}, fmt.Errorf("parsing llm endpoint match payload: %w", err)
	}

	if promptTokens <= 0 {
		promptTokens = estimatePromptTokens(matchSystemPrompt + "\n" + userPrompt)
	}

	return llm.MatchResult{
		Score:                 parsed.Score,
		Summary:               parsed.Summary,
		EstimatedPromptTokens: promptTokens,
	}, nil
}

func (p *Provider) chatCompletion(ctx context.Context, payload chatCompletionRequest) (string, int, error) {
	cancel := func() {}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancelFn context.CancelFunc
		ctx, cancelFn = context.WithTimeout(ctx, defaultRequestTimeout)
		cancel = cancelFn
	}
	defer cancel()

	body, err := json.Marshal(payload)
	if err != nil {
		return "", 0, fmt.Errorf("encoding request body: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.baseURL+"/v1/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", 0, fmt.Errorf("creating request: %w", err)
	}
	if key := strings.TrimSpace(p.apiKey); key != "" {
		request.Header.Set("Authorization", "Bearer "+key)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := p.httpClient.Do(request)
	if err != nil && !usesOpenAICloudHost(p.baseURL) && isRetryableTransportError(err) {
		timer := time.NewTimer(localRetryDelay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		case <-timer.C:
		}

		retryRequest, retryErr := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			p.baseURL+"/v1/chat/completions",
			bytes.NewReader(body),
		)
		if retryErr != nil {
			return "", 0, fmt.Errorf("creating retry request: %w", retryErr)
		}
		if key := strings.TrimSpace(p.apiKey); key != "" {
			retryRequest.Header.Set("Authorization", "Bearer "+key)
		}
		retryRequest.Header.Set("Content-Type", "application/json")
		response, err = p.httpClient.Do(retryRequest)
	}
	if err != nil {
		return "", 0, classifyTransportError(ctx, err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	if err != nil {
		return "", 0, fmt.Errorf("reading response body: %w", err)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", 0, classifyStatusError(response.StatusCode, raw)
	}

	var parsed chatCompletionResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", 0, fmt.Errorf("%w: response body is not valid JSON", ErrMalformedResponse)
	}

	if len(parsed.Choices) == 0 {
		return "", parsed.Usage.PromptTokens, fmt.Errorf("%w: response had no choices", ErrMalformedResponse)
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		return "", parsed.Usage.PromptTokens, fmt.Errorf("%w: response content is empty", ErrMalformedResponse)
	}

	return content, parsed.Usage.PromptTokens, nil
}

func (p *Provider) checkConfigured() error {
	if strings.TrimSpace(p.model) == "" {
		return fmt.Errorf("%w: missing model", ErrNotConfigured)
	}

	// Cloud OpenAI endpoints require API-key auth, while many local/self-hosted
	// OpenAI-compatible runtimes intentionally do not.
	if strings.TrimSpace(p.apiKey) == "" && usesOpenAICloudHost(p.baseURL) {
		return fmt.Errorf("%w: missing API key", ErrNotConfigured)
	}
	return nil
}

func usesOpenAICloudHost(baseURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return true
	}
	switch host {
	case "api.openai.com", "openai.com", "www.openai.com":
		return true
	default:
		return false
	}
}

func isRetryableTransportError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	retryableFragments := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"network is unreachable",
		"no route to host",
		"temporary failure",
		"i/o timeout",
		"tls handshake timeout",
	}
	for _, fragment := range retryableFragments {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}

func classifyTransportError(ctx context.Context, err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("%w: deadline exceeded", ErrTimeout)
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return fmt.Errorf("%w: network timeout", ErrTimeout)
	}

	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}

	return fmt.Errorf("request transport failed: %w", err)
}

func classifyStatusError(statusCode int, body []byte) error {
	message := extractErrorMessage(body)

	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return statusError(ErrUnauthorized, statusCode, message)
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return statusError(ErrTimeout, statusCode, message)
	case http.StatusTooManyRequests:
		return statusError(ErrRateLimited, statusCode, message)
	default:
		if statusCode >= http.StatusInternalServerError {
			return statusError(ErrUpstreamFailure, statusCode, message)
		}
		if message != "" {
			return fmt.Errorf("llm endpoint request failed with status %d: %s", statusCode, message)
		}
		return fmt.Errorf("llm endpoint request failed with status %d", statusCode)
	}
}

func statusError(classification error, statusCode int, message string) error {
	if message != "" {
		return fmt.Errorf("%w: status %d: %s", classification, statusCode, message)
	}
	return fmt.Errorf("%w: status %d", classification, statusCode)
}

func extractErrorMessage(body []byte) string {
	var envelope struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil {
		msg := strings.TrimSpace(envelope.Error.Message)
		if msg != "" {
			return truncateRunes(msg, 300)
		}
	}
	return ""
}

type parsedMatchContent struct {
	Score   float64
	Summary string
}

func parseMatchContent(content string) (parsedMatchContent, error) {
	jsonPayload, err := extractJSONObject(content)
	if err != nil {
		return parsedMatchContent{}, err
	}

	type rawPayload struct {
		Score   json.Number `json:"score"`
		Summary string      `json:"summary"`
	}

	decoder := json.NewDecoder(strings.NewReader(jsonPayload))
	decoder.UseNumber()
	var raw rawPayload
	if err := decoder.Decode(&raw); err != nil {
		return parsedMatchContent{}, fmt.Errorf("%w: invalid JSON payload", ErrMalformedResponse)
	}

	if strings.TrimSpace(raw.Score.String()) == "" {
		return parsedMatchContent{}, fmt.Errorf("%w: score field is missing", ErrMalformedResponse)
	}
	score, err := raw.Score.Float64()
	if err != nil || math.IsNaN(score) || math.IsInf(score, 0) {
		return parsedMatchContent{}, fmt.Errorf("%w: score must be a finite number", ErrMalformedResponse)
	}
	if score < 0 || score > 1 {
		return parsedMatchContent{}, fmt.Errorf("%w: score %.4f is outside [0,1]", ErrMalformedResponse, score)
	}

	summary := normalizeSummary(raw.Summary, defaultMaxSummaryRunes)
	if summary == "" {
		return parsedMatchContent{}, fmt.Errorf("%w: summary field is empty", ErrMalformedResponse)
	}

	return parsedMatchContent{
		Score:   score,
		Summary: summary,
	}, nil
}

func extractJSONObject(content string) (string, error) {
	trimmed := strings.TrimSpace(content)
	trimmed = stripCodeFence(trimmed)

	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start < 0 || end <= start {
		return "", fmt.Errorf("%w: no JSON object found", ErrMalformedResponse)
	}
	return trimmed[start : end+1], nil
}

func stripCodeFence(value string) string {
	if !strings.HasPrefix(value, "```") {
		return value
	}

	lines := strings.Split(value, "\n")
	if len(lines) < 3 {
		return value
	}
	if !strings.HasPrefix(strings.TrimSpace(lines[0]), "```") {
		return value
	}
	last := strings.TrimSpace(lines[len(lines)-1])
	if !strings.HasPrefix(last, "```") {
		return value
	}

	return strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n"))
}

func buildMatchSystemPrompt(maxSummaryRunes int) string {
	if maxSummaryRunes <= 0 {
		maxSummaryRunes = defaultMaxSummaryRunes
	}
	return fmt.Sprintf(matchSystemPromptTemplate, maxSummaryRunes)
}

func buildMatchUserPrompt(req llm.MatchRequest, truncatedDescription string, truncatedProfile string) string {
	return fmt.Sprintf(
		"Candidate query:\n%s\n\nCandidate CV profile:\n%s\n\nJob title: %s\nCompany: %s\nLocation: %s\nJob description:\n%s\n\nReturn only JSON with keys score and summary.",
		nonEmpty(req.Query),
		nonEmpty(truncatedProfile),
		nonEmpty(req.JobTitle),
		nonEmpty(req.JobCompany),
		nonEmpty(req.JobLocation),
		nonEmpty(truncatedDescription),
	)
}

func nonEmpty(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "(empty)"
	}
	return value
}

func truncateRunes(value string, max int) string {
	if max <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

func normalizeSummary(value string, maxRunes int) string {
	trimmed := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if trimmed == "" {
		return ""
	}
	if maxRunes <= 0 {
		return trimmed
	}

	runes := []rune(trimmed)
	if len(runes) <= maxRunes {
		return trimmed
	}
	if maxRunes <= 3 {
		return string(runes[:maxRunes])
	}

	cutoff := maxRunes - 3
	candidate := strings.TrimSpace(string(runes[:cutoff]))
	// Prefer whole-word truncation so UI does not show cut-off partial words.
	if lastSpace := strings.LastIndex(candidate, " "); lastSpace >= cutoff/2 {
		candidate = strings.TrimSpace(candidate[:lastSpace])
	}
	if candidate == "" {
		candidate = strings.TrimSpace(string(runes[:cutoff]))
	}
	return candidate + "..."
}

func estimatePromptTokens(content string) int {
	const charsPerToken = 4
	runes := len([]rune(content))
	if runes == 0 {
		return 0
	}
	return (runes + charsPerToken - 1) / charsPerToken
}

type chatCompletionRequest struct {
	Model          string                        `json:"model"`
	Messages       []chatCompletionMessage       `json:"messages"`
	Temperature    float64                       `json:"temperature"`
	MaxTokens      int                           `json:"max_tokens,omitempty"`
	ResponseFormat *chatCompletionResponseFormat `json:"response_format,omitempty"`
}

type chatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponseFormat struct {
	Type string `json:"type"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
	} `json:"usage"`
}
