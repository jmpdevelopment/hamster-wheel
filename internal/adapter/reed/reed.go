// Package reed implements the job source adapter for Reed.co.uk
// using their public Jobseeker REST API.
package reed

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"hamster-wheel/internal/adapter"
)

const (
	defaultBaseURL = "https://www.reed.co.uk/api/1.0"
	userAgent      = "HamsterWheel/1.0 (Job Search Assistant)"
	minRequestGap  = 2 * time.Second // Rate limit: minimum gap between HTTP requests
)

// Adapter implements adapter.Adapter for Reed UK via their REST API.
type Adapter struct {
	mu         sync.RWMutex // Protects apiKey and lastReq from concurrent access
	apiKey     string       // Reed API key (used as HTTP Basic Auth username)
	baseURL    string       // Base URL (overridable for testing)
	httpClient *http.Client // HTTP client (overridable for testing)
	lastReq    time.Time    // Timestamp of last HTTP request (for rate limiting)
}

// New creates a Reed UK adapter with the given API key.
func New(apiKey string) *Adapter {
	return &Adapter{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// newWithOptions creates an adapter with custom settings.
// Used by tests to point at a mock server.
func newWithOptions(apiKey, baseURL string, client *http.Client) *Adapter {
	return &Adapter{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: client,
	}
}

func (a *Adapter) Name() string        { return "reed_uk" }
func (a *Adapter) DisplayName() string { return "Reed UK" }

// SetAPIKey updates the API key used for authentication.
// Called when the user sets the key via the UI.
// Thread-safe: can be called concurrently with other operations.
func (a *Adapter) SetAPIKey(key string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.apiKey = key
}

// FetchNewJobs retrieves job listings from Reed's search API
// matching the given search parameters.
func (a *Adapter) FetchNewJobs(ctx context.Context, params adapter.SearchParams) ([]adapter.JobSummary, error) {
	q := url.Values{}
	q.Set("keywords", params.Keywords)
	if params.Location != "" {
		q.Set("locationName", params.Location)
	}
	q.Set("resultsToTake", "100")

	searchURL := a.baseURL + "/search?" + q.Encode()

	body, err := a.doGet(ctx, searchURL)
	if err != nil {
		return nil, fmt.Errorf("searching jobs: %w", err)
	}
	defer body.Close()

	var resp searchResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("parsing search response: %w", err)
	}

	jobs := make([]adapter.JobSummary, 0, len(resp.Results))
	for _, r := range resp.Results {
		job := adapter.JobSummary{
			SourceID: strconv.Itoa(r.JobID),
			Title:    r.JobTitle,
			Company:  r.EmployerName,
			Location: r.LocationName,
			URL:      r.JobURL,
			Snippet:  r.JobDescription,
		}

		if t, err := time.Parse("02/01/2006", r.Date); err == nil {
			job.PostedAt = t
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

// FetchJobDetails fetches full job details from Reed's jobs API.
func (a *Adapter) FetchJobDetails(ctx context.Context, job adapter.JobSummary) (*adapter.JobDetails, error) {
	detailURL := a.baseURL + "/jobs/" + job.SourceID

	body, err := a.doGet(ctx, detailURL)
	if err != nil {
		return nil, fmt.Errorf("fetching job details: %w", err)
	}
	defer body.Close()

	var resp jobDetailsResponse
	if err := json.NewDecoder(body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("parsing job details: %w", err)
	}

	details := &adapter.JobDetails{
		JobSummary:      job,
		FullDescription: resp.JobDescription,
		Salary:          formatSalary(resp.MinimumSalary, resp.MaximumSalary, resp.Currency, resp.SalaryType),
		JobType:         resp.ContractType,
	}

	// Override URL with the external URL if available (direct employer link).
	if resp.ExternalURL != "" {
		details.URL = resp.ExternalURL
	}

	return details, nil
}

// Validate checks that the Reed API is reachable and the API key is valid.
func (a *Adapter) Validate(ctx context.Context) error {
	testURL := a.baseURL + "/search?keywords=test&resultsToTake=1"
	body, err := a.doGet(ctx, testURL)
	if err != nil {
		return fmt.Errorf("reed UK validation failed: %w", err)
	}
	body.Close()
	return nil
}

// doGet performs an HTTP GET with rate limiting, Basic Auth, and context support.
// Returns the response body — the caller is responsible for closing it.
// Thread-safe: can be called concurrently from multiple goroutines.
func (a *Adapter) doGet(ctx context.Context, targetURL string) (io.ReadCloser, error) {
	// Rate limiting: wait if we made a request too recently.
	a.mu.Lock()
	since := time.Since(a.lastReq)
	if since < minRequestGap {
		a.mu.Unlock() // Unlock during wait to avoid blocking other callers
		wait := minRequestGap - since
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		a.mu.Lock() // Re-lock to update lastReq
	}
	a.lastReq = time.Now()
	apiKey := a.apiKey // Read apiKey under lock
	a.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.SetBasicAuth(apiKey, "") // Use local copy

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		return nil, fmt.Errorf("authentication failed (401) — check REED_API_KEY")
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, targetURL)
	}

	return resp.Body, nil
}

// formatSalary builds a human-readable salary string.
func formatSalary(min, max float64, currency, salaryType string) string {
	if min == 0 && max == 0 {
		return ""
	}

	symbol := "£"
	if currency != "" && currency != "GBP" {
		symbol = currency + " "
	}

	period := ""
	if salaryType != "" {
		period = " " + salaryType
	}

	if max == 0 || min == max {
		return fmt.Sprintf("%s%s%s", symbol, formatNumber(min), period)
	}

	return fmt.Sprintf("%s%s - %s%s%s", symbol, formatNumber(min), symbol, formatNumber(max), period)
}

// formatNumber formats a float as a comma-separated integer string.
func formatNumber(n float64) string {
	s := strconv.FormatFloat(n, 'f', 0, 64)

	// Add commas for thousands.
	if len(s) <= 3 {
		return s
	}

	var result strings.Builder
	remainder := len(s) % 3
	if remainder > 0 {
		result.WriteString(s[:remainder])
	}
	for i := remainder; i < len(s); i += 3 {
		if result.Len() > 0 {
			result.WriteByte(',')
		}
		result.WriteString(s[i : i+3])
	}
	return result.String()
}

// --- JSON response types (private) ---

type searchResponse struct {
	Results []searchResult `json:"results"`
}

type searchResult struct {
	JobID          int     `json:"jobId"`
	EmployerName   string  `json:"employerName"`
	JobTitle       string  `json:"jobTitle"`
	LocationName   string  `json:"locationName"`
	MinimumSalary  float64 `json:"minimumSalary"`
	MaximumSalary  float64 `json:"maximumSalary"`
	Currency       string  `json:"currency"`
	JobDescription string  `json:"jobDescription"`
	JobURL         string  `json:"jobUrl"`
	Date           string  `json:"date"`
}

type jobDetailsResponse struct {
	JobDescription string  `json:"jobDescription"`
	MinimumSalary  float64 `json:"minimumSalary"`
	MaximumSalary  float64 `json:"maximumSalary"`
	Currency       string  `json:"currency"`
	SalaryType     string  `json:"salaryType"`
	ContractType   string  `json:"contractType"`
	ExternalURL    string  `json:"externalUrl"`
	JobURL         string  `json:"jobUrl"`
}
