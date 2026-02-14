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
	desc := req.JobDescription
	maxRunes := req.MaxDescriptionRunes
	if maxRunes <= 0 {
		maxRunes = defaultMaxDescriptionRunes
	}
	desc = truncateRunes(desc, maxRunes)

	querySet := tokenSet(query)
	profileSet := tokenSet(req.CandidateProfile)
	if len(querySet) == 0 && len(profileSet) == 0 {
		return llm.MatchResult{
			Score:                 0,
			Summary:               "No usable query or CV profile tokens after normalization.",
			EstimatedPromptTokens: estimateTokens(req),
		}, nil
	}

	titleOverlap := overlapRatio(querySet, tokenSet(req.JobTitle))
	companyOverlap := overlapRatio(querySet, tokenSet(req.JobCompany))
	locationOverlap := overlapRatio(querySet, tokenSet(req.JobLocation))
	descriptionOverlap := overlapRatio(querySet, tokenSet(desc))

	queryScore := (0.50 * titleOverlap) +
		(0.20 * descriptionOverlap) +
		(0.20 * companyOverlap) +
		(0.10 * locationOverlap)

	jobContext := tokenSet(req.JobTitle + " " + req.JobCompany + " " + req.JobLocation + " " + desc)
	profileScore := overlapRatio(profileSet, jobContext)

	var score float64
	switch {
	case len(querySet) == 0:
		score = profileScore
	case len(profileSet) == 0:
		score = queryScore
	default:
		// Keep query relevance primary while still incorporating CV signal.
		score = (0.75 * queryScore) + (0.25 * profileScore)
	}

	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	summary := "Heuristic overlap:"
	if len(querySet) > 0 {
		summary = fmt.Sprintf(
			"%s query title %.0f%%, description %.0f%%, company %.0f%%, location %.0f%%.",
			summary,
			titleOverlap*100,
			descriptionOverlap*100,
			companyOverlap*100,
			locationOverlap*100,
		)
	}
	if len(profileSet) > 0 {
		summary = fmt.Sprintf("%s CV-to-job %.0f%%.", summary, profileScore*100)
	}

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
	content := req.Query + " " + req.CandidateProfile + " " + req.JobTitle + " " + req.JobCompany + " " + req.JobLocation + " " + req.JobDescription
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
