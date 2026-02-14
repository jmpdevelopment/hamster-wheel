// Package matcher runs asynchronous job matching independent of polling.
// It claims pending match rows, computes scores, and writes results.
package matcher

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"hamster-wheel/internal/db"
	"hamster-wheel/internal/llm"
)

const (
	defaultPollInterval    = 3 * time.Second
	defaultBatchSize       = 4
	defaultProviderName    = "heuristic_v1"
	defaultMatchTimeout    = 20 * time.Second
	defaultStaleProcessing = 2 * time.Minute
)

// Store is the DB contract required by the matcher worker.
type Store interface {
	ClaimNextPendingJobMatch(ctx context.Context) (*db.JobMatch, error)
	MarkJobMatchMatched(ctx context.Context, jobID string, score float64, summary string) error
	MarkJobMatchFailed(ctx context.Context, jobID string, reason string) error
	RequeueStaleProcessingJobMatches(ctx context.Context, staleBefore time.Time) (int, error)
	GetJob(ctx context.Context, id string) (*db.Job, error)
	GetFilter(ctx context.Context, id string) (*db.SearchFilter, error)
}

// WorkerConfig controls queue polling and per-item processing behavior.
type WorkerConfig struct {
	ProviderName      string
	PollInterval      time.Duration
	BatchSize         int
	MatchTimeout      time.Duration
	StaleAfter        time.Duration
	DescriptionsRunes int
}

// Worker processes pending match rows in the background.
type Worker struct {
	store     Store
	providers *llm.Registry

	providerName     string
	pollInterval     time.Duration
	batchSize        int
	matchTimeout     time.Duration
	staleAfter       time.Duration
	descriptionRunes int

	statusChangedHook func(jobID, status string)

	stateMu sync.RWMutex
	running bool
	cancel  context.CancelFunc
	done    chan struct{}
}

func New(store Store, providers *llm.Registry, cfg WorkerConfig) *Worker {
	providerName := strings.TrimSpace(cfg.ProviderName)
	if providerName == "" {
		providerName = defaultProviderName
	}
	pollInterval := cfg.PollInterval
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}
	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	matchTimeout := cfg.MatchTimeout
	if matchTimeout <= 0 {
		matchTimeout = defaultMatchTimeout
	}
	staleAfter := cfg.StaleAfter
	if staleAfter <= 0 {
		staleAfter = defaultStaleProcessing
	}
	descriptionRunes := cfg.DescriptionsRunes
	if descriptionRunes <= 0 {
		descriptionRunes = 1400
	}

	return &Worker{
		store:            store,
		providers:        providers,
		providerName:     providerName,
		pollInterval:     pollInterval,
		batchSize:        batchSize,
		matchTimeout:     matchTimeout,
		staleAfter:       staleAfter,
		descriptionRunes: descriptionRunes,
	}
}

// Start launches the background loop. Calling Start while running is a no-op.
func (w *Worker) Start() {
	w.stateMu.Lock()
	if w.running {
		w.stateMu.Unlock()
		return
	}
	w.running = true
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.done = make(chan struct{})
	done := w.done
	pollInterval := w.pollInterval
	w.stateMu.Unlock()

	go func() {
		defer close(done)
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		w.requeueStale(context.Background())
		w.processBatch(context.Background())

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.requeueStale(ctx)
				w.processBatch(ctx)
			}
		}
	}()
}

// Stop shuts down the worker loop. Calling Stop while stopped is safe.
func (w *Worker) Stop() {
	w.stateMu.Lock()
	cancel := w.cancel
	done := w.done
	w.cancel = nil
	w.done = nil
	w.running = false
	w.stateMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

// SetStatusChangedHook registers an optional callback for status transitions.
func (w *Worker) SetStatusChangedHook(hook func(jobID, status string)) {
	w.stateMu.Lock()
	w.statusChangedHook = hook
	w.stateMu.Unlock()
}

// RunOnce processes up to one configured batch. Useful for tests.
func (w *Worker) RunOnce(ctx context.Context) (int, error) {
	if err := w.requeueStale(ctx); err != nil {
		return 0, err
	}
	return w.processBatch(ctx), nil
}

func (w *Worker) requeueStale(ctx context.Context) error {
	staleBefore := time.Now().UTC().Add(-w.staleAfter)
	requeued, err := w.store.RequeueStaleProcessingJobMatches(ctx, staleBefore)
	if err != nil {
		return fmt.Errorf("requeueing stale processing matches: %w", err)
	}
	if requeued > 0 {
		slog.Warn("requeued stale processing matches", "count", requeued)
	}
	return nil
}

func (w *Worker) processBatch(ctx context.Context) int {
	processed := 0
	for processed < w.batchSize {
		if ctx.Err() != nil {
			break
		}

		claimed, err := w.store.ClaimNextPendingJobMatch(ctx)
		if err != nil {
			slog.Error("matcher failed to claim pending row", "error", err)
			break
		}
		if claimed == nil {
			break
		}

		w.emitStatusChanged(claimed.JobID, db.JobMatchStatusProcessing)
		if err := w.processOne(ctx, claimed.JobID); err != nil {
			slog.Error("matcher failed to process claimed row",
				"job_id", claimed.JobID,
				"error", err,
			)
		}
		processed++
	}
	return processed
}

func (w *Worker) processOne(ctx context.Context, jobID string) error {
	job, err := w.store.GetJob(ctx, jobID)
	if err != nil {
		return w.failMatch(ctx, jobID, fmt.Sprintf("Loading job failed: %v", err))
	}
	if job == nil {
		// Job was removed after queueing; ignore if match row was cascaded away.
		if err := w.store.MarkJobMatchFailed(ctx, jobID, "Job no longer exists."); err != nil && !errors.Is(err, db.ErrJobMatchNotFound) {
			return fmt.Errorf("marking missing job as failed: %w", err)
		}
		return nil
	}

	provider, ok := w.providers.Get(w.providerName)
	if !ok {
		return w.failMatch(ctx, jobID, fmt.Sprintf("Provider %q is not registered.", w.providerName))
	}

	query := w.queryForJob(ctx, job)
	matchCtx, cancel := context.WithTimeout(ctx, w.matchTimeout)
	defer cancel()

	result, err := provider.Match(matchCtx, llm.MatchRequest{
		Query:               query,
		JobTitle:            job.Title,
		JobCompany:          job.Company,
		JobLocation:         job.Location,
		JobDescription:      job.Description,
		MaxDescriptionRunes: w.descriptionRunes,
	})
	if err != nil {
		return w.failMatch(ctx, jobID, fmt.Sprintf("Provider scoring failed: %v", err))
	}

	if strings.TrimSpace(result.Summary) == "" {
		result.Summary = "Match computed."
	}
	score := result.Score
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	if err := w.store.MarkJobMatchMatched(ctx, jobID, score, result.Summary); err != nil {
		if errors.Is(err, db.ErrJobMatchNotFound) {
			return nil
		}
		return fmt.Errorf("marking job as matched: %w", err)
	}
	w.emitStatusChanged(jobID, db.JobMatchStatusMatched)
	return nil
}

func (w *Worker) queryForJob(ctx context.Context, job *db.Job) string {
	if job != nil && job.FilterID != nil {
		filter, err := w.store.GetFilter(ctx, *job.FilterID)
		if err == nil && filter != nil {
			keywords := strings.TrimSpace(filter.Keywords)
			if keywords != "" {
				return keywords
			}
		}
	}

	// Fallback keeps scoring deterministic even when filter data is unavailable.
	return strings.TrimSpace(job.Title + " " + job.Location)
}

func (w *Worker) failMatch(ctx context.Context, jobID string, reason string) error {
	if err := w.store.MarkJobMatchFailed(ctx, jobID, reason); err != nil {
		if errors.Is(err, db.ErrJobMatchNotFound) {
			return nil
		}
		return err
	}
	w.emitStatusChanged(jobID, db.JobMatchStatusFailed)
	return nil
}

func (w *Worker) emitStatusChanged(jobID, status string) {
	w.stateMu.RLock()
	hook := w.statusChangedHook
	w.stateMu.RUnlock()
	if hook != nil {
		hook(jobID, status)
	}
}
