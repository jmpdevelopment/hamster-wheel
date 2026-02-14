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
	EnsureJobMatchPending(ctx context.Context, jobID string) error
}

// Scheduler manages periodic polling of job sources.
type Scheduler struct {
	store    JobStore
	adapters *adapter.Registry
	interval time.Duration
	// pollTimeout bounds one background poll cycle so sleep/wake networking
	// stalls cannot block the scheduler loop indefinitely.
	pollTimeout time.Duration

	cancel     context.CancelFunc // cancels the background goroutine
	done       chan struct{}      // closed when the background goroutine exits
	reschedule chan struct{}      // nudges the loop to recalculate next timer

	// stateMu protects running, paused, nextPollAt, statusChangedHook, cancel, done, and reschedule.
	stateMu            sync.RWMutex
	paused             bool
	polling            bool
	nextPollAt         time.Time
	lastPollSummary    PollCycleSummary
	hasLastPollSummary bool
	running            bool // true if scheduler is currently running
	statusChangedHook  func()
}

const (
	// Keep background polls bounded but generous enough for normal batches.
	minPollTimeout = 2 * time.Minute
	maxPollTimeout = 15 * time.Minute
)

func derivePollTimeout(interval time.Duration) time.Duration {
	if interval <= 0 {
		return minPollTimeout
	}

	timeout := interval / 2
	if timeout < minPollTimeout {
		return minPollTimeout
	}
	if timeout > maxPollTimeout {
		return maxPollTimeout
	}
	return timeout
}

// New creates a scheduler. Call Start() to begin polling.
func New(store JobStore, adapters *adapter.Registry, interval time.Duration) *Scheduler {
	return &Scheduler{
		store:       store,
		adapters:    adapters,
		interval:    interval,
		pollTimeout: derivePollTimeout(interval),
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
	s.reschedule = make(chan struct{}, 1)
	reschedule := s.reschedule
	s.stateMu.Unlock()

	go func(reschedule <-chan struct{}) {
		defer close(s.done)
		defer func() {
			s.stateMu.Lock()
			s.running = false
			s.stateMu.Unlock()
		}()

		slog.Info("scheduler started", "interval", s.interval)

		// Run immediately on start, then on each tick.
		// Set nextPollAt before polling so UI can show schedule even if a poll
		// is currently in-flight or temporarily slow.
		next := time.Now().Add(s.interval)
		s.setNextPollAt(next)
		s.safePoll(ctx)

		timer := time.NewTimer(remainingUntil(next))
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("scheduler stopped")
				return
			case <-timer.C:
				next = time.Now().Add(s.interval)
				s.setNextPollAt(next)
				s.safePoll(ctx)
				resetTimer(timer, remainingUntil(next))
			case <-reschedule:
				s.stateMu.RLock()
				next = s.nextPollAt
				s.stateMu.RUnlock()
				if next.IsZero() {
					next = time.Now().Add(s.interval)
					s.setNextPollAt(next)
				}
				resetTimer(timer, remainingUntil(next))
			}
		}
	}(reschedule)
}

// safePoll wraps PollOnce with panic recovery so a buggy adapter
// can't crash the entire application. Skips polling when paused.
func (s *Scheduler) safePoll(ctx context.Context) {
	s.stateMu.RLock()
	paused := s.paused
	pollTimeout := s.pollTimeout
	s.stateMu.RUnlock()

	if paused {
		slog.Debug("poll skipped (paused)")
		return
	}

	pollCtx := ctx
	cancel := func() {}
	if pollTimeout > 0 {
		pollCtx, cancel = context.WithTimeout(ctx, pollTimeout)
	}
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic in poll cycle (recovered)", "panic", r)
			s.setLastPollSummary(buildPollCycleSummary(nil, fmt.Errorf("panic: %v", r), time.Now().UTC()))
		}
	}()
	results, err := s.PollOnce(pollCtx)
	if err != nil {
		if errors.Is(err, ErrPollInProgress) {
			slog.Debug("poll cycle skipped (already in progress)")
			return
		}
		if errors.Is(err, context.DeadlineExceeded) {
			slog.Warn("poll cycle timed out", "timeout", pollTimeout)
			s.setLastPollSummary(buildPollCycleSummary(results, err, time.Now().UTC()))
			return
		}
		slog.Error("poll cycle failed", "error", err)
		s.setLastPollSummary(buildPollCycleSummary(results, err, time.Now().UTC()))
		return
	}
	s.setLastPollSummary(buildPollCycleSummary(results, nil, time.Now().UTC()))
}

func (s *Scheduler) beginPollCycle() bool {
	s.stateMu.Lock()
	if s.polling {
		s.stateMu.Unlock()
		return false
	}
	s.polling = true
	s.stateMu.Unlock()
	s.notifyStatusChanged()
	return true
}

func (s *Scheduler) finishPollCycle() {
	s.stateMu.Lock()
	s.polling = false
	s.stateMu.Unlock()
	s.notifyStatusChanged()
}

// SetStatusChangedHook registers a callback that runs when polling status
// transitions (for example, next poll rescheduled or paused/resumed).
// It is optional and safe to set once during startup wiring.
func (s *Scheduler) SetStatusChangedHook(hook func()) {
	s.stateMu.Lock()
	s.statusChangedHook = hook
	s.stateMu.Unlock()
}

func (s *Scheduler) notifyStatusChanged() {
	s.stateMu.RLock()
	hook := s.statusChangedHook
	s.stateMu.RUnlock()
	if hook != nil {
		hook()
	}
}

func (s *Scheduler) setLastPollSummary(summary PollCycleSummary) {
	s.stateMu.Lock()
	s.lastPollSummary = summary
	s.hasLastPollSummary = true
	s.stateMu.Unlock()
}

// LastPollSummary returns the most recent scheduler-triggered poll summary.
func (s *Scheduler) LastPollSummary() (PollCycleSummary, bool) {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	if !s.hasLastPollSummary {
		return PollCycleSummary{}, false
	}
	summary := s.lastPollSummary
	if len(summary.Filters) > 0 {
		summary.Filters = append([]PollFilterSummary(nil), summary.Filters...)
	}
	return summary, true
}

func remainingUntil(t time.Time) time.Duration {
	remaining := time.Until(t)
	if remaining < 0 {
		return 0
	}
	return remaining
}

func resetTimer(timer *time.Timer, d time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(d)
}

func (s *Scheduler) signalReschedule() {
	s.stateMu.RLock()
	ch := s.reschedule
	s.stateMu.RUnlock()
	if ch == nil {
		return
	}

	select {
	case ch <- struct{}{}:
	default:
	}
}

// setNextPollAt updates the next poll timestamp and logs it.
func (s *Scheduler) setNextPollAt(t time.Time) {
	s.stateMu.Lock()
	s.nextPollAt = t
	s.stateMu.Unlock()
	slog.Info("next poll scheduled", "at", t.Format(time.RFC3339))
	s.notifyStatusChanged()
}

// SetPaused pauses or resumes auto-polling. When paused, ticker
// events are silently skipped. Manual PollNow calls still work.
func (s *Scheduler) SetPaused(paused bool) {
	s.stateMu.Lock()
	s.paused = paused
	running := s.running
	s.stateMu.Unlock()
	if paused {
		slog.Info("auto-polling paused")
	} else {
		slog.Info("auto-polling resumed")
	}
	// Ensure status callers and scheduler timer both move to a fresh schedule after resume.
	if !paused && running {
		s.RescheduleFromNow()
		return
	}
	s.notifyStatusChanged()
}

// RescheduleFromNow pushes the next automatic poll to now + interval when
// the scheduler is running and unpaused.
func (s *Scheduler) RescheduleFromNow() {
	s.stateMu.RLock()
	running := s.running
	paused := s.paused
	interval := s.interval
	s.stateMu.RUnlock()

	if !running || paused {
		return
	}

	s.setNextPollAt(time.Now().Add(interval))
	s.signalReschedule()
}

// IsPaused returns whether auto-polling is currently paused.
func (s *Scheduler) IsPaused() bool {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.paused
}

// IsPolling returns whether a poll cycle is currently running.
func (s *Scheduler) IsPolling() bool {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.polling
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
	s.reschedule = nil
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

// PollFilterSummary stores one filter's poll result without Go error types.
type PollFilterSummary struct {
	FilterID   string
	FilterName string
	Source     string
	NewJobs    int
	Skipped    int
	Error      string
}

// PollCycleSummary stores one completed scheduler-triggered poll cycle.
type PollCycleSummary struct {
	CompletedAt   time.Time
	TotalFilters  int
	FailedFilters int
	NewJobs       int
	Skipped       int
	CycleError    string
	Filters       []PollFilterSummary
}

func buildPollCycleSummary(results []PollResult, cycleErr error, completedAt time.Time) PollCycleSummary {
	summary := PollCycleSummary{
		CompletedAt:  completedAt,
		TotalFilters: len(results),
	}
	if cycleErr != nil {
		summary.CycleError = cycleErr.Error()
	}
	if len(results) == 0 {
		return summary
	}

	summary.Filters = make([]PollFilterSummary, 0, len(results))
	for _, result := range results {
		filterSummary := PollFilterSummary{
			FilterID:   result.FilterID,
			FilterName: result.FilterName,
			Source:     result.Source,
			NewJobs:    result.NewJobs,
			Skipped:    result.Skipped,
		}
		if result.Err != nil {
			filterSummary.Error = result.Err.Error()
			summary.FailedFilters++
		}
		summary.NewJobs += result.NewJobs
		summary.Skipped += result.Skipped
		summary.Filters = append(summary.Filters, filterSummary)
	}

	return summary
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
//
// Returns:
//   - results + nil error for successful poll cycles (even if some filters failed)
//   - nil + error only when the entire cycle couldn't run (e.g. DB unavailable)
func (s *Scheduler) PollOnce(ctx context.Context) ([]PollResult, error) {
	if !s.beginPollCycle() {
		return nil, ErrPollInProgress
	}
	defer s.finishPollCycle()

	filters, err := s.store.ListEnabledFilters(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing enabled filters: %w", err)
	}

	if len(filters) == 0 {
		slog.Debug("no enabled filters, skipping poll")
		return nil, nil
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

			jobID, err := s.store.InsertJob(ctx, fj.job)
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
			if err := s.store.EnsureJobMatchPending(ctx, jobID); err != nil {
				slog.Error("failed to enqueue job for matching",
					"job_id", jobID,
					"title", fj.title,
					"error", err)
			}
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

	return results, nil
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

var ErrPollInProgress = errors.New("poll already in progress")
