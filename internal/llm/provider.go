package llm

import "context"

// MatchRequest contains the compact inputs used to score one job.
// Keep fields intentionally narrow so providers can minimize prompt/token usage.
type MatchRequest struct {
	Query               string
	JobTitle            string
	JobCompany          string
	JobLocation         string
	JobDescription      string
	MaxDescriptionRunes int
}

// MatchResult is the normalized provider response for one score operation.
type MatchResult struct {
	Score                 float64
	Summary               string
	EstimatedPromptTokens int
}

// Provider is the extension point for LLM-backed (or local) match scorers.
type Provider interface {
	Name() string
	DisplayName() string
	Match(ctx context.Context, req MatchRequest) (MatchResult, error)
	Validate(ctx context.Context) error
}
