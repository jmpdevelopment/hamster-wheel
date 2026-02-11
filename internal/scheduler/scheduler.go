// Package scheduler runs background polling cycles that fetch new jobs
// from enabled search filters via their adapters and store them in the database.
//
// The scheduler is the orchestrator of the "Job Fetching" feature:
//
//	[Enabled Filters] → [Adapter Registry] → [FetchNewJobs] → [Dedup Check] → [FetchJobDetails] → [Insert to DB]
//
// Fetching happens concurrently (one goroutine per filter), but all database
// writes are serialized through a single writer (the main PollOnce goroutine).
// This avoids SQLite write contention entirely.
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"hamster-wheel/internal/adapter"
	"hamster-wheel/internal/db"
)

// JobStore is the database interface the scheduler uses for job operations.
// Defined here (not in db package) per Go convention: interfaces belong
// to the consumer, not the provider.
type JobStore interface {
	ListEnabledFilters(ctx context.Context) ([]db.SearchFilter, error)
	JobExistsBySourceID(ctx context.Context, source, sourceID string) (bool, error)
	InsertJob(ctx context.Context, job *db.Job) (string, error)
}

// Scheduler manages periodic polling of job sources.
type Scheduler struct {
	store    JobStore
	adapters *adapter.Registry
	interval time.Duration

	cancel context.CancelFunc // cancels the background goroutine
	done   chan struct{}       // closed when the background goroutine exits

	// stateMu protects running, paused, nextPollAt, cancel, and done
	stateMu    sync.RWMutex
	paused     bool
	nextPollAt time.Time
	running    bool // true if scheduler is currently running
}

// New creates a scheduler. Call Start() to begin polling.
func New(store JobStore, adapters *adapter.Registry, interval time.Duration) *Scheduler {
	return &Scheduler{
		store:    store,
		adapters: adapters,
		interval: interval,
	}
}

// Start begins the background polling loop. It immediately runs one poll cycle,
// then repeats at the configured interval. Thread-safe: calling Start() while
// already running is a no-op. Can be restarted after Stop().
func (s *Scheduler) Start() {
	s.stateMu.Lock()
	if s.running {
		s.stateMu.Unlock()
		return // Already running, ignore
	}
	s.running = true

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.done = make(chan struct{})
	s.stateMu.Unlock()

	go func() {
		defer close(s.done)
		defer func() {
			s.stateMu.Lock()
			s.running = false
			s.stateMu.Unlock()
		}()

		slog.Info("scheduler started", "interval", s.interval)

		// Run immediately on start, then on each tick.
		s.safePoll(ctx)
		s.setNextPollAt(time.Now().Add(s.interval))

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("scheduler stopped")
				return
			case <-ticker.C:
				s.safePoll(ctx)
				s.setNextPollAt(time.Now().Add(s.interval))
			}
		}
	}()
}

// safePoll wraps PollOnce with panic recovery so a buggy adapter
// can't crash the entire application. Skips polling when paused.
func (s *Scheduler) safePoll(ctx context.Context) {
	s.stateMu.RLock()
	paused := s.paused
	s.stateMu.RUnlock()

	if paused {
		slog.Debug("poll skipped (paused)")
		return
	}

	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic in poll cycle (recovered)", "panic", r)
		}
	}()
	s.PollOnce(ctx)
}

// setNextPollAt updates the next poll timestamp and logs it.
func (s *Scheduler) setNextPollAt(t time.Time) {
	s.stateMu.Lock()
	s.nextPollAt = t
	s.stateMu.Unlock()
	slog.Info("next poll scheduled", "at", t.Format(time.RFC3339))
}

// SetPaused pauses or resumes auto-polling. When paused, ticker
// events are silently skipped. Manual PollNow calls still work.
func (s *Scheduler) SetPaused(paused bool) {
	s.stateMu.Lock()
	s.paused = paused
	s.stateMu.Unlock()
	if paused {
		slog.Info("auto-polling paused")
	} else {
		slog.Info("auto-polling resumed")
	}
}

// IsPaused returns whether auto-polling is currently paused.
func (s *Scheduler) IsPaused() bool {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.paused
}

// NextPollAt returns the scheduled time for the next poll cycle.
// Returns zero time if the scheduler hasn't started yet.
func (s *Scheduler) NextPollAt() time.Time {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.nextPollAt
}

// Stop cancels the polling loop and waits for it to finish.
// Thread-safe: calling Stop() when not running is a no-op.
func (s *Scheduler) Stop() {
	s.stateMu.Lock()
	cancel := s.cancel
	done := s.done
	s.cancel = nil
	s.done = nil
	s.stateMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

// PollResult reports the outcome of polling a single filter.
type PollResult struct {
	FilterID   string
	FilterName string
	Source     string
	NewJobs    int   // How many new jobs were stored
	Skipped    int   // How many duplicates were skipped
	Err        error // Non-nil if the filter failed entirely
}

// fetchedJob is a job ready to be written to the database.
// Produced by fetch goroutines, consumed by the single writer in PollOnce.
type fetchedJob struct {
	job      *db.Job
	title    string // for logging
	company  string // for logging
	source   string // for logging
	filterID string // which filter found this
}

// filterFetchResult is what each fetch goroutine returns.
// It contains the fetched jobs (not yet written) and metadata.
type filterFetchResult struct {
	FilterID   string
	FilterName string
	Source     string
	Jobs       []fetchedJob // New jobs to insert (not yet in DB)
	Skipped    int          // Duplicates skipped during fetch
	Err        error        // Non-nil if the filter failed entirely
}

// PollOnce runs a single poll cycle: fetches enabled filters from the DB,
// polls each one concurrently via its adapter, then writes all discovered
// jobs to the database sequentially (single writer — no SQLite contention).
//
// This is the core logic, separated from the ticker so it's directly testable.
func (s *Scheduler) PollOnce(ctx context.Context) []PollResult {
	filters, err := s.store.ListEnabledFilters(ctx)
	if err != nil {
		slog.Error("failed to list enabled filters", "error", err)
		return nil
	}

	if len(filters) == 0 {
		slog.Debug("no enabled filters, skipping poll")
		return nil
	}

	slog.Info("poll cycle starting", "filters", len(filters))

	// Phase 1: Fetch concurrently — goroutines do HTTP + dedup reads.
	var (
		mu           sync.Mutex
		wg           sync.WaitGroup
		fetchResults []filterFetchResult
	)

	for _, f := range filters {
		wg.Add(1)
		go func(filter db.SearchFilter) {
			defer wg.Done()

			// Recover from panics in individual fetch goroutines so one
			// buggy adapter can't crash the whole poll cycle (or the app).
			defer func() {
				if r := recover(); r != nil {
					slog.Error("panic in fetch goroutine (recovered)",
						"filter", filter.Name,
						"source", filter.Source,
						"panic", r)

					mu.Lock()
					fetchResults = append(fetchResults, filterFetchResult{
						FilterID:   filter.ID,
						FilterName: filter.Name,
						Source:     filter.Source,
						Err:        fmt.Errorf("panic: %v", r),
					})
					mu.Unlock()
				}
			}()

			fr := s.fetchFilter(ctx, filter)

			mu.Lock()
			fetchResults = append(fetchResults, fr)
			mu.Unlock()
		}(f)
	}

	wg.Wait()

	// Phase 2: Write sequentially — single writer, no contention.
	results := make([]PollResult, 0, len(fetchResults))

	for _, fr := range fetchResults {
		result := PollResult{
			FilterID:   fr.FilterID,
			FilterName: fr.FilterName,
			Source:     fr.Source,
			Skipped:    fr.Skipped,
			Err:        fr.Err,
		}

		if fr.Err != nil {
			slog.Error("filter poll failed",
				"filter", fr.FilterName,
				"source", fr.Source,
				"error", fr.Err)
			results = append(results, result)
			continue
		}

		// Insert each job — all writes happen here, on this single goroutine.
		for _, fj := range fr.Jobs {
			if ctx.Err() != nil {
				result.Err = ctx.Err()
				break
			}

			_, err := s.store.InsertJob(ctx, fj.job)
			if err != nil {
				if errors.Is(err, db.ErrDuplicateJob) {
					// Race between dedup read and insert (rare but possible
					// if two filters find the same job). Just skip.
					result.Skipped++
					continue
				}
				slog.Error("failed to insert job",
					"title", fj.title,
					"error", err)
				continue
			}

			result.NewJobs++
			slog.Info("new job stored",
				"title", fj.title,
				"company", fj.company,
				"source", fj.source)
		}

		slog.Info("filter poll result",
			"filter", fr.FilterName,
			"new_jobs", result.NewJobs,
			"skipped", result.Skipped)

		results = append(results, result)
	}

	// Log summary.
	totalNew := 0
	totalSkipped := 0
	for _, r := range results {
		totalNew += r.NewJobs
		totalSkipped += r.Skipped
	}

	slog.Info("poll cycle complete",
		"filters", len(filters),
		"new_jobs", totalNew,
		"skipped", totalSkipped)

	return results
}

// fetchFilter does the I/O-heavy work for a single filter: looks up the
// adapter, fetches job summaries via HTTP, checks the DB for duplicates
// (reads only), and fetches full details for new jobs via HTTP.
//
// It does NOT write to the database — that's PollOnce's job (single writer).
func (s *Scheduler) fetchFilter(ctx context.Context, filter db.SearchFilter) filterFetchResult {
	fr := filterFetchResult{
		FilterID:   filter.ID,
		FilterName: filter.Name,
		Source:     filter.Source,
	}

	// Look up the adapter for this filter's source.
	a, ok := s.adapters.Get(filter.Source)
	if !ok {
		fr.Err = &ErrAdapterNotFound{Source: filter.Source}
		return fr
	}

	// Fetch job summaries from the source (HTTP).
	params := adapter.SearchParams{
		Keywords: filter.Keywords,
		Location: filter.Location,
	}

	slog.Info("fetching jobs",
		"filter", filter.Name,
		"keywords", params.Keywords,
		"location", params.Location,
		"source", filter.Source)

	summaries, err := a.FetchNewJobs(ctx, params)
	if err != nil {
		fr.Err = err
		return fr
	}

	slog.Info("fetched job summaries",
		"filter", filter.Name,
		"count", len(summaries))

	// For each summary: dedup check (DB read) → fetch details (HTTP).
	for _, summary := range summaries {
		if ctx.Err() != nil {
			fr.Err = ctx.Err()
			return fr
		}

		// Dedup check — this is a read, safe to do concurrently.
		exists, err := s.store.JobExistsBySourceID(ctx, a.Name(), summary.SourceID)
		if err != nil {
			slog.Error("dedup check failed",
				"source_id", summary.SourceID,
				"error", err)
			continue
		}
		if exists {
			fr.Skipped++
			continue
		}

		// Fetch full details for this new job (HTTP).
		details, err := a.FetchJobDetails(ctx, summary)
		if err != nil {
			slog.Warn("failed to fetch job details, storing summary only",
				"title", summary.Title,
				"error", err)
			details = &adapter.JobDetails{
				JobSummary: summary,
			}
		}

		// Build the DB job struct (but don't insert yet).
		postedAt := details.PostedAt
		filterID := filter.ID
		job := &db.Job{
			Source:      a.Name(),
			SourceID:    details.SourceID,
			Title:       details.Title,
			Company:     details.Company,
			Location:    details.Location,
			Description: details.FullDescription,
			URL:         details.URL,
			PostedAt:    &postedAt,
			FilterID:    &filterID,
		}

		if details.PostedAt.IsZero() {
			job.PostedAt = nil
		}

		fr.Jobs = append(fr.Jobs, fetchedJob{
			job:      job,
			title:    details.Title,
			company:  details.Company,
			source:   a.Name(),
			filterID: filter.ID,
		})
	}

	return fr
}

// ErrAdapterNotFound is returned when a filter references an adapter
// that isn't registered.
type ErrAdapterNotFound struct {
	Source string
}

func (e *ErrAdapterNotFound) Error() string {
	return "adapter not found: " + e.Source
}
