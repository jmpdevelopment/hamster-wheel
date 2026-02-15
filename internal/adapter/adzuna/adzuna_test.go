package adzuna

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"hamster-wheel/internal/adapter"
)

const (
	testAppID  = "test-app-id"
	testAppKey = "test-app-key"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newMockClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func jsonResponse(status int, payload string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: io.NopCloser(strings.NewReader(payload)),
	}
}

func setupMockAdapter(t *testing.T) *Adapter {
	t.Helper()

	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/jobs/gb/search/1":
			if req.URL.Query().Get("app_id") != testAppID || req.URL.Query().Get("app_key") != testAppKey {
				return jsonResponse(http.StatusForbidden, `{"error":"forbidden"}`), nil
			}
			return jsonResponse(http.StatusOK, `{
				"results": [
					{
						"id": "1001",
						"title": "Senior Go Developer",
						"description": "Build distributed backend systems.",
						"created": "2026-02-10T12:00:00Z",
						"redirect_url": "https://www.adzuna.co.uk/land/ad/1001",
						"salary_min": 70000,
						"salary_max": 90000,
						"contract_type": "permanent",
						"contract_time": "full_time",
						"company": {"display_name": "Acme Corp"},
						"location": {"display_name": "London"}
					},
					{
						"id": "1002",
						"title": "Backend Engineer",
						"description": "Design resilient APIs.",
						"created": "2026-02-09T09:30:00Z",
						"redirect_url": "",
						"adzuna_url": "https://www.adzuna.co.uk/jobs/details/1002",
						"salary_min": 0,
						"salary_max": 50000,
						"contract_type": "contract",
						"contract_time": "part_time",
						"company": {"display_name": "Beta Ltd"},
						"location": {"display_name": "Manchester"}
					}
				]
			}`), nil
		case "/jobs/gb/version":
			if req.URL.Query().Get("app_id") != testAppID || req.URL.Query().Get("app_key") != testAppKey {
				return jsonResponse(http.StatusForbidden, `{"error":"forbidden"}`), nil
			}
			return jsonResponse(http.StatusOK, `{"api_version":"1.7.0"}`), nil
		default:
			return jsonResponse(http.StatusNotFound, `{"error":"not found"}`), nil
		}
	})

	a := newWithOptions(testAppID, testAppKey, "gb", "https://api.example.test", client)
	a.requestGap = 0
	return a
}

func buildSearchPayload(ids []string) string {
	results := make([]searchResult, 0, len(ids))
	for _, id := range ids {
		result := searchResult{
			ID:          adzunaID(id),
			Title:       "Role " + id,
			Description: "Snippet " + id,
			Created:     "2026-02-10T12:00:00Z",
			RedirectURL: "https://example.test/jobs/" + id,
		}
		result.Company.DisplayName = "Company " + id
		result.Location.DisplayName = "London"
		results = append(results, result)
	}
	raw, _ := json.Marshal(searchResponse{Results: results})
	return string(raw)
}

func TestNameAndDisplayName(t *testing.T) {
	a := New("id", "key")
	if got := a.Name(); got != "adzuna_gb" {
		t.Fatalf("expected name %q, got %q", "adzuna_gb", got)
	}
	if got := a.DisplayName(); got != "Adzuna UK" {
		t.Fatalf("expected display name %q, got %q", "Adzuna UK", got)
	}
}

func TestSetCredentialsAndHasCredentials(t *testing.T) {
	a := New("", "")
	if a.HasCredentials() {
		t.Fatal("expected credentials to be missing")
	}

	a.SetCredentials(" next-id ", " next-key ")
	if !a.HasCredentials() {
		t.Fatal("expected credentials to be set")
	}

	a.SetCredentials(" ", " ")
	if a.HasCredentials() {
		t.Fatal("expected blank credentials to be treated as missing")
	}
}

func TestFetchNewJobs(t *testing.T) {
	a := setupMockAdapter(t)

	jobs, err := a.FetchNewJobs(context.Background(), adapter.SearchParams{
		Keywords: "golang",
		Location: "London",
	})
	if err != nil {
		t.Fatalf("FetchNewJobs returned error: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobs))
	}

	first := jobs[0]
	if first.SourceID != "1001" {
		t.Fatalf("expected first source ID %q, got %q", "1001", first.SourceID)
	}
	if first.URL != "https://www.adzuna.co.uk/land/ad/1001" {
		t.Fatalf("expected redirect URL to be preserved, got %q", first.URL)
	}
	if first.PostedAt.IsZero() {
		t.Fatal("expected posted_at to be parsed")
	}

	second := jobs[1]
	if second.URL != "https://www.adzuna.co.uk/jobs/details/1002" {
		t.Fatalf("expected fallback adzuna URL, got %q", second.URL)
	}
}

func TestFetchNewJobsOmitsWhereWhenLocationEmpty(t *testing.T) {
	var rawQuery string
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		rawQuery = req.URL.RawQuery
		return jsonResponse(http.StatusOK, `{"results":[]}`), nil
	})
	a := newWithOptions(testAppID, testAppKey, "gb", "https://api.example.test", client)
	a.requestGap = 0

	_, err := a.FetchNewJobs(context.Background(), adapter.SearchParams{
		Keywords: "backend",
	})
	if err != nil {
		t.Fatalf("FetchNewJobs returned error: %v", err)
	}
	if strings.Contains(rawQuery, "where=") {
		t.Fatalf("expected query without where parameter, got %q", rawQuery)
	}
}

func TestFetchNewJobsAuthenticationError(t *testing.T) {
	client := newMockClient(func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusForbidden, `{"error":"forbidden"}`), nil
	})
	a := newWithOptions("bad-id", "bad-key", "gb", "https://api.example.test", client)
	a.requestGap = 0

	_, err := a.FetchNewJobs(context.Background(), adapter.SearchParams{Keywords: "go"})
	if err == nil {
		t.Fatal("expected authentication error, got nil")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Fatalf("expected authentication error, got %v", err)
	}
}

func TestFetchNewJobsRateLimitError(t *testing.T) {
	client := newMockClient(func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusTooManyRequests, `{"error":"rate-limited"}`), nil
	})
	a := newWithOptions(testAppID, testAppKey, "gb", "https://api.example.test", client)
	a.requestGap = 0

	_, err := a.FetchNewJobs(context.Background(), adapter.SearchParams{Keywords: "go"})
	if err == nil {
		t.Fatal("expected 429 error, got nil")
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("expected rate limit error, got %v", err)
	}
}

func TestFetchNewJobsPaginatesToOneHundredResults(t *testing.T) {
	page1IDs := make([]string, 0, 50)
	page2IDs := make([]string, 0, 50)
	for i := 1; i <= 50; i++ {
		page1IDs = append(page1IDs, strconv.Itoa(i))
	}
	for i := 51; i <= 100; i++ {
		page2IDs = append(page2IDs, strconv.Itoa(i))
	}

	var pageRequests []string
	client := newMockClient(func(req *http.Request) (*http.Response, error) {
		pageRequests = append(pageRequests, req.URL.Path)
		if got := req.URL.Query().Get("results_per_page"); got != "50" {
			return jsonResponse(http.StatusBadRequest, `{"error":"bad page size"}`), nil
		}

		switch req.URL.Path {
		case "/jobs/gb/search/1":
			return jsonResponse(http.StatusOK, buildSearchPayload(page1IDs)), nil
		case "/jobs/gb/search/2":
			return jsonResponse(http.StatusOK, buildSearchPayload(page2IDs)), nil
		default:
			return jsonResponse(http.StatusInternalServerError, `{"error":"unexpected page"}`), nil
		}
	})
	a := newWithOptions(testAppID, testAppKey, "gb", "https://api.example.test", client)
	a.requestGap = 0

	jobs, err := a.FetchNewJobs(context.Background(), adapter.SearchParams{Keywords: "go"})
	if err != nil {
		t.Fatalf("FetchNewJobs returned error: %v", err)
	}
	if len(jobs) != 100 {
		t.Fatalf("expected 100 jobs, got %d", len(jobs))
	}
	if jobs[0].SourceID != "1" || jobs[99].SourceID != "100" {
		t.Fatalf("unexpected first/last IDs: %q ... %q", jobs[0].SourceID, jobs[99].SourceID)
	}
	if len(pageRequests) != 2 {
		t.Fatalf("expected 2 paged requests, got %d", len(pageRequests))
	}
}

func TestFetchJobDetailsUsesCachedSearchFields(t *testing.T) {
	a := setupMockAdapter(t)

	jobs, err := a.FetchNewJobs(context.Background(), adapter.SearchParams{Keywords: "go"})
	if err != nil {
		t.Fatalf("FetchNewJobs returned error: %v", err)
	}

	details, err := a.FetchJobDetails(context.Background(), jobs[0])
	if err != nil {
		t.Fatalf("FetchJobDetails returned error: %v", err)
	}
	if details.FullDescription == "" {
		t.Fatal("expected full description from search snippet")
	}
	if details.Salary != "GBP 70,000 - GBP 90,000" {
		t.Fatalf("expected salary range, got %q", details.Salary)
	}
	if details.JobType != "Permanent Full-time" {
		t.Fatalf("expected combined job type, got %q", details.JobType)
	}
}

func TestFetchJobDetailsFallsBackToSummarySnippet(t *testing.T) {
	a := New(testAppID, testAppKey)
	job := adapter.JobSummary{
		SourceID: "missing",
		Title:    "Role",
		Snippet:  "summary snippet",
		URL:      "https://example.com",
	}

	details, err := a.FetchJobDetails(context.Background(), job)
	if err != nil {
		t.Fatalf("FetchJobDetails returned error: %v", err)
	}
	if details.FullDescription != "summary snippet" {
		t.Fatalf("expected summary snippet fallback, got %q", details.FullDescription)
	}
}

func TestValidate(t *testing.T) {
	a := setupMockAdapter(t)
	if err := a.Validate(context.Background()); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
}

func TestValidateMissingCredentials(t *testing.T) {
	a := New("", "")
	err := a.Validate(context.Background())
	if err == nil {
		t.Fatal("expected missing credentials error")
	}
	if !strings.Contains(err.Error(), "credentials not configured") {
		t.Fatalf("expected missing credentials message, got %v", err)
	}
}

func TestDoGetEnforcesRequestGapAcrossConcurrentCalls(t *testing.T) {
	const callCount = 4
	const gap = 40 * time.Millisecond
	const tolerance = 8 * time.Millisecond

	var (
		mu       sync.Mutex
		arrivals []time.Time
	)

	client := newMockClient(func(_ *http.Request) (*http.Response, error) {
		mu.Lock()
		arrivals = append(arrivals, time.Now())
		mu.Unlock()
		return jsonResponse(http.StatusOK, `{"ok":true}`), nil
	})
	a := newWithOptions(testAppID, testAppKey, "gb", "https://api.example.test", client)
	a.requestGap = gap

	targetURL := "https://api.example.test/jobs/gb/version?app_id=" + testAppID + "&app_key=" + testAppKey
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(callCount)
	for i := 0; i < callCount; i++ {
		go func() {
			defer wg.Done()
			<-start
			body, err := a.doGet(context.Background(), targetURL)
			if err != nil {
				t.Errorf("doGet returned error: %v", err)
				return
			}
			body.Close()
		}()
	}
	close(start)
	wg.Wait()

	if len(arrivals) != callCount {
		t.Fatalf("expected %d arrivals, got %d", callCount, len(arrivals))
	}
	sort.Slice(arrivals, func(i, j int) bool { return arrivals[i].Before(arrivals[j]) })
	for i := 1; i < len(arrivals); i++ {
		delta := arrivals[i].Sub(arrivals[i-1])
		if delta+tolerance < gap {
			t.Fatalf("gap too small: got %v want at least %v", delta, gap)
		}
	}
}

func TestDoGetRespectsContextWhileWaitingForRateLimit(t *testing.T) {
	client := newMockClient(func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusOK, `{"ok":true}`), nil
	})
	a := newWithOptions(testAppID, testAppKey, "gb", "https://api.example.test", client)
	a.requestGap = 250 * time.Millisecond
	a.lastReq = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	targetURL := "https://api.example.test/jobs/gb/version?app_id=" + testAppID + "&app_key=" + testAppKey
	_, err := a.doGet(ctx, targetURL)
	if err == nil {
		t.Fatal("expected deadline exceeded while waiting for request gap")
	}
	if !strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}

func TestFormatSalary(t *testing.T) {
	tests := []struct {
		name string
		min  float64
		max  float64
		want string
	}{
		{name: "empty", min: 0, max: 0, want: ""},
		{name: "range", min: 50000, max: 70000, want: "GBP 50,000 - GBP 70,000"},
		{name: "min only", min: 45000, max: 0, want: "GBP 45,000"},
		{name: "max only", min: 0, max: 40000, want: "Up to GBP 40,000"},
		{name: "same", min: 42000, max: 42000, want: "GBP 42,000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatSalary(tt.min, tt.max); got != tt.want {
				t.Fatalf("formatSalary(%v,%v)=%q want %q", tt.min, tt.max, got, tt.want)
			}
		})
	}
}

func TestFormatJobType(t *testing.T) {
	tests := []struct {
		name         string
		contractType string
		contractTime string
		want         string
	}{
		{name: "both", contractType: "contract", contractTime: "part_time", want: "Contract Part-time"},
		{name: "type only", contractType: "permanent", contractTime: "", want: "Permanent"},
		{name: "time only", contractType: "", contractTime: "full_time", want: "Full-time"},
		{name: "unknown", contractType: "fixed_term", contractTime: "seasonal_time", want: "Fixed Term Seasonal Time"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatJobType(tt.contractType, tt.contractTime); got != tt.want {
				t.Fatalf("formatJobType(%q,%q)=%q want %q", tt.contractType, tt.contractTime, got, tt.want)
			}
		})
	}
}

func TestAdzunaIDUnmarshalSupportsNumbersAndStrings(t *testing.T) {
	var id adzunaID
	if err := json.Unmarshal([]byte(`"abc123"`), &id); err != nil {
		t.Fatalf("unmarshal string id: %v", err)
	}
	if got := id.String(); got != "abc123" {
		t.Fatalf("expected string id, got %q", got)
	}

	if err := json.Unmarshal([]byte(`456`), &id); err != nil {
		t.Fatalf("unmarshal numeric id: %v", err)
	}
	if got := id.String(); got != "456" {
		t.Fatalf("expected numeric id as string, got %q", got)
	}
}
