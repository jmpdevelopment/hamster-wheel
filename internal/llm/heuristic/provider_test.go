package heuristic

import (
	"context"
	"testing"

	"hamster-wheel/internal/llm"
)

func TestMatchGivesHigherScoreForRelevantJob(t *testing.T) {
	provider := New()

	relevant, err := provider.Match(context.Background(), llm.MatchRequest{
		Query:          "go backend distributed systems",
		JobTitle:       "Senior Go Backend Engineer",
		JobCompany:     "Acme",
		JobLocation:    "Remote",
		JobDescription: "Build distributed services in Go and optimize APIs.",
	})
	if err != nil {
		t.Fatalf("matching relevant job: %v", err)
	}

	unrelated, err := provider.Match(context.Background(), llm.MatchRequest{
		Query:          "go backend distributed systems",
		JobTitle:       "Social Media Designer",
		JobCompany:     "Brandify",
		JobLocation:    "London",
		JobDescription: "Design content calendars and social graphics.",
	})
	if err != nil {
		t.Fatalf("matching unrelated job: %v", err)
	}

	if relevant.Score <= unrelated.Score {
		t.Fatalf("expected relevant score > unrelated score, got relevant=%.3f unrelated=%.3f", relevant.Score, unrelated.Score)
	}
}

func TestMatchHandlesEmptyQuery(t *testing.T) {
	provider := New()
	result, err := provider.Match(context.Background(), llm.MatchRequest{
		Query:          "",
		JobTitle:       "Go Engineer",
		JobDescription: "Build APIs",
	})
	if err != nil {
		t.Fatalf("matching with empty query: %v", err)
	}
	if result.Score != 0 {
		t.Fatalf("expected score 0 for empty query, got %.3f", result.Score)
	}
}
