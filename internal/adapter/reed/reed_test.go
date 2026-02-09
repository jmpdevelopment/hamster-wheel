package reed

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hamster-wheel/internal/adapter"
)

// Mock search response matching Reed's API format.
var mockSearchResults = searchResponse{
	Results: []searchResult{
		{
			JobID:          12345,
			EmployerName:   "Acme Corp",
			JobTitle:       "Senior Go Developer",
			LocationName:   "London",
			MinimumSalary:  70000,
			MaximumSalary:  90000,
			Currency:       "GBP",
			JobDescription: "We need a senior Go developer for our platform team.",
			JobURL:         "https://www.reed.co.uk/jobs/senior-go-developer/12345",
			Date:           "03/02/2026",
		},
		{
			JobID:          67890,
			EmployerName:   "TechStartup Ltd",
			JobTitle:       "Backend Engineer",
			LocationName:   "Manchester",
			MinimumSalary:  50000,
			MaximumSalary:  65000,
			Currency:       "GBP",
			JobDescription: "Join our backend team building microservices.",
			JobURL:         "https://www.reed.co.uk/jobs/backend-engineer/67890",
			Date:           "02/02/2026",
		},
	},
}

// Mock job details response.
var mockJobDetails = jobDetailsResponse{
	JobDescription: "Full job description: We need a senior Go developer for our platform team. Requirements: 5+ years Go, microservices, Kubernetes.",
	MinimumSalary:  70000,
	MaximumSalary:  90000,
	Currency:       "GBP",
	SalaryType:     "per annum",
	ContractType:   "Permanent",
	ExternalURL:    "https://acmecorp.com/careers/12345",
	JobURL:         "https://www.reed.co.uk/jobs/senior-go-developer/12345",
}

const testAPIKey = "test-api-key"

// setupMockServer creates a test HTTP server that serves Reed-like JSON responses.
// Go's httptest.NewServer runs a real local HTTP server so you test actual HTTP
// requests without hitting the real internet — like Python's unittest.mock for HTTP.
func setupMockServer(t *testing.T) (*httptest.Server, *Adapter) {
	t.Helper()

	mux := http.NewServeMux()

	// Serve search results.
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		user, _, ok := r.BasicAuth()
		if !ok || user != testAPIKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockSearchResults)
	})

	// Serve job details.
	mux.HandleFunc("/jobs/", func(w http.ResponseWriter, r *http.Request) {
		user, _, ok := r.BasicAuth()
		if !ok || user != testAPIKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockJobDetails)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	a := newWithOptions(testAPIKey, server.URL, server.Client())
	// Disable rate limiting for tests.
	a.lastReq = a.lastReq.Add(-minRequestGap)

	return server, a
}

func TestName(t *testing.T) {
	a := New("key")
	if a.Name() != "reed_uk" {
		t.Errorf("expected %q, got %q", "reed_uk", a.Name())
	}
	if a.DisplayName() != "Reed UK" {
		t.Errorf("expected %q, got %q", "Reed UK", a.DisplayName())
	}
}

func TestFetchNewJobs(t *testing.T) {
	_, a := setupMockServer(t)
	ctx := context.Background()

	jobs, err := a.FetchNewJobs(ctx, adapter.SearchParams{
		Keywords: "Go Developer",
		Location: "London",
	})
	if err != nil {
		t.Fatalf("FetchNewJobs: %v", err)
	}

	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}

	// Check first job.
	if jobs[0].Title != "Senior Go Developer" {
		t.Errorf("job[0] title: expected %q, got %q", "Senior Go Developer", jobs[0].Title)
	}
	if jobs[0].SourceID != "12345" {
		t.Errorf("job[0] SourceID: expected %q, got %q", "12345", jobs[0].SourceID)
	}
	if jobs[0].Company != "Acme Corp" {
		t.Errorf("job[0] company: expected %q, got %q", "Acme Corp", jobs[0].Company)
	}
	if jobs[0].Location != "London" {
		t.Errorf("job[0] location: expected %q, got %q", "London", jobs[0].Location)
	}
	if jobs[0].Snippet == "" {
		t.Error("job[0] Snippet should not be empty")
	}
	if jobs[0].PostedAt.IsZero() {
		t.Error("job[0] PostedAt should be parsed")
	}

	// Check second job.
	if jobs[1].Title != "Backend Engineer" {
		t.Errorf("job[1] title: expected %q, got %q", "Backend Engineer", jobs[1].Title)
	}
	if jobs[1].SourceID != "67890" {
		t.Errorf("job[1] SourceID: expected %q, got %q", "67890", jobs[1].SourceID)
	}
}

func TestFetchNewJobsEmptyResults(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse{Results: []searchResult{}})
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	a := newWithOptions(testAPIKey, server.URL, server.Client())

	jobs, err := a.FetchNewJobs(context.Background(), adapter.SearchParams{Keywords: "nonexistent"})
	if err != nil {
		t.Fatalf("FetchNewJobs: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}
}

func TestFetchNewJobsAuth(t *testing.T) {
	var receivedAuth string
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		user, _, _ := r.BasicAuth()
		receivedAuth = user
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse{Results: []searchResult{}})
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	a := newWithOptions("my-secret-key", server.URL, server.Client())

	_, err := a.FetchNewJobs(context.Background(), adapter.SearchParams{Keywords: "test"})
	if err != nil {
		t.Fatalf("FetchNewJobs: %v", err)
	}
	if receivedAuth != "my-secret-key" {
		t.Errorf("expected auth user %q, got %q", "my-secret-key", receivedAuth)
	}
}

func TestFetchNewJobsAuthError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	a := newWithOptions("bad-key", server.URL, server.Client())

	_, err := a.FetchNewJobs(context.Background(), adapter.SearchParams{Keywords: "test"})
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("expected auth error message, got: %v", err)
	}
}

func TestFetchNewJobsNoLocation(t *testing.T) {
	var receivedQuery string
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse{Results: []searchResult{}})
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	a := newWithOptions(testAPIKey, server.URL, server.Client())

	_, err := a.FetchNewJobs(context.Background(), adapter.SearchParams{Keywords: "Python"})
	if err != nil {
		t.Fatalf("FetchNewJobs: %v", err)
	}
	if strings.Contains(receivedQuery, "locationName") {
		t.Errorf("URL should not contain locationName when empty, got: %s", receivedQuery)
	}
}

func TestFetchJobDetails(t *testing.T) {
	_, a := setupMockServer(t)
	ctx := context.Background()

	job := adapter.JobSummary{
		SourceID: "12345",
		Title:    "Senior Go Developer",
		Company:  "Acme Corp",
		URL:      "https://www.reed.co.uk/jobs/senior-go-developer/12345",
	}

	details, err := a.FetchJobDetails(ctx, job)
	if err != nil {
		t.Fatalf("FetchJobDetails: %v", err)
	}

	if details.FullDescription == "" {
		t.Error("expected non-empty description")
	}
	if !strings.Contains(details.FullDescription, "5+ years Go") {
		t.Errorf("description should contain requirements, got: %q", details.FullDescription)
	}
	if details.Salary != "£70,000 - £90,000 per annum" {
		t.Errorf("expected salary %q, got: %q", "£70,000 - £90,000 per annum", details.Salary)
	}
	if details.JobType != "Permanent" {
		t.Errorf("expected job type %q, got: %q", "Permanent", details.JobType)
	}

	// External URL should override the original URL.
	if details.URL != "https://acmecorp.com/careers/12345" {
		t.Errorf("expected external URL, got: %q", details.URL)
	}

	// Original fields should be preserved.
	if details.Title != "Senior Go Developer" {
		t.Errorf("expected title preserved, got: %q", details.Title)
	}
}

func TestFetchJobDetailsNoExternalURL(t *testing.T) {
	noExtDetails := jobDetailsResponse{
		JobDescription: "A job description.",
		MinimumSalary:  40000,
		MaximumSalary:  50000,
		Currency:       "GBP",
		SalaryType:     "per annum",
		ContractType:   "Contract",
		ExternalURL:    "",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/jobs/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(noExtDetails)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	a := newWithOptions(testAPIKey, server.URL, server.Client())

	originalURL := "https://www.reed.co.uk/jobs/test/99999"
	job := adapter.JobSummary{SourceID: "99999", URL: originalURL}

	details, err := a.FetchJobDetails(context.Background(), job)
	if err != nil {
		t.Fatalf("FetchJobDetails: %v", err)
	}

	// Without external URL, original URL should be preserved.
	if details.URL != originalURL {
		t.Errorf("expected original URL preserved, got: %q", details.URL)
	}
}

func TestFetchJobDetailsServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/jobs/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	a := newWithOptions(testAPIKey, server.URL, server.Client())

	job := adapter.JobSummary{SourceID: "12345"}
	_, err := a.FetchJobDetails(context.Background(), job)
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestFetchJobDetailsAuthError(t *testing.T) {
	_, a := setupMockServer(t)

	// Use a different API key so auth fails.
	a.apiKey = "wrong-key"

	job := adapter.JobSummary{SourceID: "12345"}
	_, err := a.FetchJobDetails(context.Background(), job)
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("expected auth error message, got: %v", err)
	}
}

func TestValidate(t *testing.T) {
	_, a := setupMockServer(t)
	if err := a.Validate(context.Background()); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestValidateAuthFailure(t *testing.T) {
	_, a := setupMockServer(t)
	a.apiKey = "wrong-key"

	err := a.Validate(context.Background())
	if err == nil {
		t.Fatal("expected error for auth failure, got nil")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("expected auth error, got: %v", err)
	}
}

func TestSourceIDDeterministic(t *testing.T) {
	// Reed's jobId is used directly as SourceID (converted to string).
	// Two identical job IDs should produce the same SourceID.
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse{
			Results: []searchResult{
				{JobID: 42, JobTitle: "Test Job"},
			},
		})
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	a := newWithOptions(testAPIKey, server.URL, server.Client())
	params := adapter.SearchParams{Keywords: "test"}

	jobs1, _ := a.FetchNewJobs(context.Background(), params)
	a.lastReq = a.lastReq.Add(-minRequestGap) // reset rate limit
	jobs2, _ := a.FetchNewJobs(context.Background(), params)

	if jobs1[0].SourceID != jobs2[0].SourceID {
		t.Errorf("SourceID should be deterministic: %q != %q", jobs1[0].SourceID, jobs2[0].SourceID)
	}
	if jobs1[0].SourceID != "42" {
		t.Errorf("SourceID should be string of jobId, got %q", jobs1[0].SourceID)
	}
}

func TestFormatSalary(t *testing.T) {
	tests := []struct {
		name       string
		min, max   float64
		currency   string
		salaryType string
		want       string
	}{
		{"both zero", 0, 0, "GBP", "per annum", ""},
		{"range", 50000, 60000, "GBP", "per annum", "£50,000 - £60,000 per annum"},
		{"min only", 50000, 0, "GBP", "per annum", "£50,000 per annum"},
		{"same min max", 50000, 50000, "GBP", "per annum", "£50,000 per annum"},
		{"no currency", 30000, 40000, "", "", "£30,000 - £40,000"},
		{"non-GBP", 60000, 80000, "EUR", "per annum", "EUR 60,000 - EUR 80,000 per annum"},
		{"daily rate", 400, 500, "GBP", "per day", "£400 - £500 per day"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSalary(tt.min, tt.max, tt.currency, tt.salaryType)
			if got != tt.want {
				t.Errorf("formatSalary(%v, %v, %q, %q) = %q, want %q",
					tt.min, tt.max, tt.currency, tt.salaryType, got, tt.want)
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0, "0"},
		{100, "100"},
		{1000, "1,000"},
		{50000, "50,000"},
		{100000, "100,000"},
		{1000000, "1,000,000"},
	}

	for _, tt := range tests {
		got := formatNumber(tt.input)
		if got != tt.want {
			t.Errorf("formatNumber(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
