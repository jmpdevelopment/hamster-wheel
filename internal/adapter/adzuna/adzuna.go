// Package adzuna implements the job source adapter for Adzuna.
package adzuna

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	defaultBaseURL = "https://api.adzuna.com/v1/api"
	defaultCountry = "gb"
	firstPage      = 1
	resultsPerPage = 50
	maxResults     = 100
	userAgent      = "HamsterWheel/1.0 (Job Search Assistant)"

	// Public Adzuna access defaults to 25 requests/minute.
	minRequestGap = 2500 * time.Millisecond
)

// Adapter implements adapter.Adapter for Adzuna.
type Adapter struct {
	mu sync.RWMutex

	appID   string
	appKey  string
	country string

	baseURL    string
	httpClient *http.Client
	lastReq    time.Time
	requestGap time.Duration

	// Adzuna API search responses already include description snippets,
	// salary, and contract metadata. Cache by source ID so FetchJobDetails
	// can return stable detail fields without extra API requests.
	jobCache map[string]cachedDetails
}

type cachedDetails struct {
	fullDescription string
	salary          string
	jobType         string
	url             string
}

// New creates an Adzuna adapter with the given credentials.
func New(appID, appKey string) *Adapter {
	return &Adapter{
		appID:      appID,
		appKey:     appKey,
		country:    defaultCountry,
		baseURL:    defaultBaseURL,
		requestGap: minRequestGap,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		jobCache: make(map[string]cachedDetails),
	}
}

// newWithOptions creates an adapter with custom settings.
// Used by tests to point at a mock server.
func newWithOptions(appID, appKey, country, baseURL string, client *http.Client) *Adapter {
	if strings.TrimSpace(country) == "" {
		country = defaultCountry
	}
	return &Adapter{
		appID:      appID,
		appKey:     appKey,
		country:    strings.ToLower(strings.TrimSpace(country)),
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: client,
		requestGap: minRequestGap,
		jobCache:   make(map[string]cachedDetails),
	}
}

func (a *Adapter) Name() string        { return "adzuna_gb" }
func (a *Adapter) DisplayName() string { return "Adzuna UK" }

// HasCredentials reports whether adapter credentials are present.
func (a *Adapter) HasCredentials() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return strings.TrimSpace(a.appID) != "" && strings.TrimSpace(a.appKey) != ""
}

// SetCredentials updates API credentials at runtime.
func (a *Adapter) SetCredentials(appID, appKey string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.appID = appID
	a.appKey = appKey
}

// FetchNewJobs retrieves job listings from Adzuna's search API.
func (a *Adapter) FetchNewJobs(ctx context.Context, params adapter.SearchParams) ([]adapter.JobSummary, error) {
	q := url.Values{}
	q.Set("app_id", strings.TrimSpace(a.appID))
	q.Set("app_key", strings.TrimSpace(a.appKey))
	q.Set("what", params.Keywords)
	if location := strings.TrimSpace(params.Location); location != "" {
		q.Set("where", location)
	}
	q.Set("results_per_page", strconv.Itoa(resultsPerPage))
	q.Set("sort_by", "date")
	q.Set("content-type", "application/json")

	maxPages := (maxResults + resultsPerPage - 1) / resultsPerPage
	jobs := make([]adapter.JobSummary, 0, maxResults)
	cache := make(map[string]cachedDetails, maxResults)
	seen := make(map[string]struct{}, maxResults)

	for page := firstPage; page < firstPage+maxPages && len(jobs) < maxResults; page++ {
		searchURL := fmt.Sprintf(
			"%s/jobs/%s/search/%d?%s",
			strings.TrimRight(a.baseURL, "/"),
			a.country,
			page,
			q.Encode(),
		)

		body, err := a.doGet(ctx, searchURL)
		if err != nil {
			return nil, fmt.Errorf("searching jobs: %w", err)
		}

		var resp searchResponse
		if err := json.NewDecoder(body).Decode(&resp); err != nil {
			body.Close()
			return nil, fmt.Errorf("parsing search response: %w", err)
		}
		body.Close()

		if len(resp.Results) == 0 {
			break
		}

		for _, r := range resp.Results {
			if len(jobs) >= maxResults {
				break
			}

			sourceID := strings.TrimSpace(r.ID.String())
			if sourceID == "" {
				continue
			}
			if _, duplicate := seen[sourceID]; duplicate {
				continue
			}
			seen[sourceID] = struct{}{}

			jobURL := strings.TrimSpace(r.RedirectURL)
			if jobURL == "" {
				jobURL = strings.TrimSpace(r.AdzunaURL)
			}

			job := adapter.JobSummary{
				SourceID: sourceID,
				Title:    strings.TrimSpace(r.Title),
				Company:  strings.TrimSpace(r.Company.DisplayName),
				Location: strings.TrimSpace(r.Location.DisplayName),
				URL:      jobURL,
				Snippet:  strings.TrimSpace(r.Description),
			}
			if t, err := time.Parse(time.RFC3339, strings.TrimSpace(r.Created)); err == nil {
				job.PostedAt = t
			}
			jobs = append(jobs, job)

			cache[sourceID] = cachedDetails{
				fullDescription: job.Snippet,
				salary:          formatSalary(r.SalaryMin, r.SalaryMax),
				jobType:         formatJobType(r.ContractType, r.ContractTime),
				url:             jobURL,
			}
		}

		if len(resp.Results) < resultsPerPage {
			break
		}
	}

	a.storeJobCache(cache)

	return jobs, nil
}

// FetchJobDetails returns details for a job.
// Adzuna's public search API already supplies the available description text.
func (a *Adapter) FetchJobDetails(_ context.Context, job adapter.JobSummary) (*adapter.JobDetails, error) {
	details := &adapter.JobDetails{
		JobSummary:      job,
		FullDescription: strings.TrimSpace(job.Snippet),
	}

	if cached, ok := a.cachedJobDetails(job.SourceID); ok {
		if cached.fullDescription != "" {
			details.FullDescription = cached.fullDescription
		}
		details.Salary = cached.salary
		details.JobType = cached.jobType
		if cached.url != "" {
			details.URL = cached.url
		}
	}

	return details, nil
}

// Validate checks that the Adzuna API is reachable and credentials are valid.
func (a *Adapter) Validate(ctx context.Context) error {
	q := url.Values{}
	q.Set("app_id", strings.TrimSpace(a.appID))
	q.Set("app_key", strings.TrimSpace(a.appKey))
	q.Set("content-type", "application/json")

	versionURL := fmt.Sprintf(
		"%s/jobs/%s/version?%s",
		strings.TrimRight(a.baseURL, "/"),
		a.country,
		q.Encode(),
	)

	body, err := a.doGet(ctx, versionURL)
	if err != nil {
		return fmt.Errorf("adzuna validation failed: %w", err)
	}
	body.Close()
	return nil
}

// doGet performs an HTTP GET with credential checks and request spacing.
func (a *Adapter) doGet(ctx context.Context, targetURL string) (io.ReadCloser, error) {
	if err := a.acquireRequestSlot(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()
		return nil, fmt.Errorf("authentication failed (%d) — check ADZUNA_APP_ID and ADZUNA_APP_KEY", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		resp.Body.Close()
		return nil, errors.New("rate limited by Adzuna API (429)")
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status %d for %s", resp.StatusCode, targetURL)
	}

	return resp.Body, nil
}

// acquireRequestSlot waits until the minimum inter-request gap is satisfied.
func (a *Adapter) acquireRequestSlot(ctx context.Context) error {
	for {
		a.mu.Lock()
		appID := strings.TrimSpace(a.appID)
		appKey := strings.TrimSpace(a.appKey)
		if appID == "" || appKey == "" {
			a.mu.Unlock()
			return errors.New("Adzuna credentials not configured (set ADZUNA_APP_ID and ADZUNA_APP_KEY)")
		}

		since := time.Since(a.lastReq)
		if since >= a.requestGap {
			a.lastReq = time.Now()
			a.mu.Unlock()
			return nil
		}
		wait := a.requestGap - since
		a.mu.Unlock()

		timer := time.NewTimer(wait)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return ctx.Err()
		}
	}
}

func (a *Adapter) storeJobCache(entries map[string]cachedDetails) {
	if len(entries) == 0 {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for sourceID, details := range entries {
		a.jobCache[sourceID] = details
	}
}

func (a *Adapter) cachedJobDetails(sourceID string) (cachedDetails, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	details, ok := a.jobCache[sourceID]
	return details, ok
}

func formatSalary(min, max float64) string {
	if min == 0 && max == 0 {
		return ""
	}

	if max == 0 || min == max {
		return "GBP " + formatNumber(min)
	}
	if min == 0 {
		return "Up to GBP " + formatNumber(max)
	}
	return "GBP " + formatNumber(min) + " - GBP " + formatNumber(max)
}

func formatNumber(n float64) string {
	s := strconv.FormatFloat(n, 'f', 0, 64)
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

func formatJobType(contractType, contractTime string) string {
	parts := make([]string, 0, 2)
	if v := normalizeContractType(contractType); v != "" {
		parts = append(parts, v)
	}
	if v := normalizeContractTime(contractTime); v != "" {
		parts = append(parts, v)
	}
	return strings.Join(parts, " ")
}

func normalizeContractType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return ""
	case "permanent":
		return "Permanent"
	case "contract":
		return "Contract"
	case "temporary":
		return "Temporary"
	case "internship":
		return "Internship"
	default:
		return titleCase(strings.ReplaceAll(value, "_", " "))
	}
}

func normalizeContractTime(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return ""
	case "full_time":
		return "Full-time"
	case "part_time":
		return "Part-time"
	default:
		return titleCase(strings.ReplaceAll(value, "_", " "))
	}
}

func titleCase(value string) string {
	if value == "" {
		return ""
	}
	words := strings.Fields(strings.ToLower(value))
	for i, word := range words {
		if len(word) == 0 {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

// adzunaID normalizes Adzuna IDs that can be returned as strings or numbers.
type adzunaID string

func (id *adzunaID) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		*id = ""
		return nil
	}

	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		*id = adzunaID(asString)
		return nil
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var asNumber json.Number
	if err := dec.Decode(&asNumber); err == nil {
		*id = adzunaID(asNumber.String())
		return nil
	}

	return fmt.Errorf("invalid adzuna id payload: %s", strings.TrimSpace(string(data)))
}

func (id adzunaID) String() string {
	return string(id)
}

type searchResponse struct {
	Results []searchResult `json:"results"`
}

type searchResult struct {
	ID           adzunaID `json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Created      string   `json:"created"`
	RedirectURL  string   `json:"redirect_url"`
	AdzunaURL    string   `json:"adzuna_url"`
	SalaryMin    float64  `json:"salary_min"`
	SalaryMax    float64  `json:"salary_max"`
	ContractType string   `json:"contract_type"`
	ContractTime string   `json:"contract_time"`

	Company struct {
		DisplayName string `json:"display_name"`
	} `json:"company"`
	Location struct {
		DisplayName string `json:"display_name"`
	} `json:"location"`
}
