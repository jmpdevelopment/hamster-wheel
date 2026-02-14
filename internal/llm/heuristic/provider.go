package heuristic

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"hamster-wheel/internal/llm"
)

const (
	ProviderName               = "heuristic_v1"
	defaultMaxDescriptionRunes = 1400
)

var tokenPattern = regexp.MustCompile(`[a-z0-9]+`)

// Provider is a low-latency local scorer used until external LLM providers are configured.
// It also acts as a cheap prefilter baseline to reduce unnecessary token usage later.
type Provider struct{}

func New() *Provider {
	return &Provider{}
}

func (p *Provider) Name() string {
	return ProviderName
}

func (p *Provider) DisplayName() string {
	return "Heuristic (Local)"
}

func (p *Provider) Validate(context.Context) error {
	return nil
}

func (p *Provider) Match(_ context.Context, req llm.MatchRequest) (llm.MatchResult, error) {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return llm.MatchResult{
			Score:                 0,
			Summary:               "No query keywords available for scoring.",
			EstimatedPromptTokens: estimateTokens(req),
		}, nil
	}

	desc := req.JobDescription
	maxRunes := req.MaxDescriptionRunes
	if maxRunes <= 0 {
		maxRunes = defaultMaxDescriptionRunes
	}
	desc = truncateRunes(desc, maxRunes)

	querySet := tokenSet(query)
	if len(querySet) == 0 {
		return llm.MatchResult{
			Score:                 0,
			Summary:               "No usable query tokens after normalization.",
			EstimatedPromptTokens: estimateTokens(req),
		}, nil
	}

	titleOverlap := overlapRatio(querySet, tokenSet(req.JobTitle))
	companyOverlap := overlapRatio(querySet, tokenSet(req.JobCompany))
	locationOverlap := overlapRatio(querySet, tokenSet(req.JobLocation))
	descriptionOverlap := overlapRatio(querySet, tokenSet(desc))

	score := (0.50 * titleOverlap) +
		(0.20 * descriptionOverlap) +
		(0.20 * companyOverlap) +
		(0.10 * locationOverlap)
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	summary := fmt.Sprintf(
		"Heuristic overlap: title %.0f%%, description %.0f%%, company %.0f%%, location %.0f%%.",
		titleOverlap*100,
		descriptionOverlap*100,
		companyOverlap*100,
		locationOverlap*100,
	)

	return llm.MatchResult{
		Score:                 score,
		Summary:               summary,
		EstimatedPromptTokens: estimateTokens(req),
	}, nil
}

func tokenSet(value string) map[string]struct{} {
	matches := tokenPattern.FindAllString(strings.ToLower(value), -1)
	set := make(map[string]struct{}, len(matches))
	for _, token := range matches {
		if len(token) < 2 {
			continue
		}
		set[token] = struct{}{}
	}
	return set
}

func overlapRatio(base map[string]struct{}, candidate map[string]struct{}) float64 {
	if len(base) == 0 {
		return 0
	}
	if len(candidate) == 0 {
		return 0
	}
	var hits int
	for token := range base {
		if _, ok := candidate[token]; ok {
			hits++
		}
	}
	return float64(hits) / float64(len(base))
}

func estimateTokens(req llm.MatchRequest) int {
	const charsPerToken = 4
	content := req.Query + " " + req.JobTitle + " " + req.JobCompany + " " + req.JobLocation + " " + req.JobDescription
	if content == "" {
		return 0
	}
	return (len([]rune(content)) + charsPerToken - 1) / charsPerToken
}

func truncateRunes(value string, max int) string {
	if max <= 0 {
		return value
	}
	r := []rune(value)
	if len(r) <= max {
		return value
	}
	return string(r[:max])
}
