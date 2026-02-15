// Package matcher runs asynchronous job matching independent of polling.
// It claims pending match rows, computes scores, and writes results.
package matcher

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"hamster-wheel/internal/cv"
	"hamster-wheel/internal/db"
	"hamster-wheel/internal/llm"
)

const (
	defaultPollInterval    = 3 * time.Second
	defaultBatchSize       = 4
	defaultProviderName    = "heuristic_v1"
	defaultMatchTimeout    = 20 * time.Second
	defaultStaleProcessing = 2 * time.Minute
	settingCVPath          = "cv_path"
	adzunaSourceName       = "adzuna_gb"
)

// Store is the DB contract required by the matcher worker.
type Store interface {
	ClaimNextPendingJobMatch(ctx context.Context) (*db.JobMatch, error)
	MarkJobMatchMatched(ctx context.Context, jobID string, score float64, summary string) error
	MarkJobMatchFailed(ctx context.Context, jobID string, reason string) error
	RequeueStaleProcessingJobMatches(ctx context.Context, staleBefore time.Time) (int, error)
	GetJob(ctx context.Context, id string) (*db.Job, error)
	GetFilter(ctx context.Context, id string) (*db.SearchFilter, error)
	GetSetting(ctx context.Context, key string) (string, error)
}

// ProviderResolver returns the provider that should be used for the current
// match execution. It allows runtime switching without worker restart.
type ProviderResolver func(ctx context.Context) (providerName string, provider llm.Provider, err error)

// WorkerConfig controls queue polling and per-item processing behavior.
type WorkerConfig struct {
	ProviderName      string
	ProviderResolver  ProviderResolver
	PollInterval      time.Duration
	BatchSize         int
	MatchTimeout      time.Duration
	ProviderTimeouts  map[string]time.Duration
	StaleAfter        time.Duration
	DescriptionsRunes int
}

// Worker processes pending match rows in the background.
type Worker struct {
	store     Store
	providers *llm.Registry
	logger    *slog.Logger

	providerName     string
	pollInterval     time.Duration
	batchSize        int
	matchTimeout     time.Duration
	providerTimeouts map[string]time.Duration
	staleAfter       time.Duration
	descriptionRunes int
	providerResolver ProviderResolver
	cvCacheMu        sync.Mutex
	cvCachePath      string
	cvCacheModTime   time.Time
	cvCacheSize      int64
	cvCacheProfile   string

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
	providerResolver := cfg.ProviderResolver
	if providerResolver == nil {
		providerResolver = func(context.Context) (string, llm.Provider, error) {
			provider, ok := providers.Get(providerName)
			if !ok {
				return providerName, nil, fmt.Errorf("provider %q is not registered", providerName)
			}
			return providerName, provider, nil
		}
	}
	providerTimeouts := make(map[string]time.Duration)
	for name, timeout := range cfg.ProviderTimeouts {
		name = strings.TrimSpace(name)
		if name == "" || timeout <= 0 {
			continue
		}
		providerTimeouts[name] = timeout
	}

	return &Worker{
		store:            store,
		providers:        providers,
		logger:           slog.Default().With("component", "matcher_worker"),
		providerName:     providerName,
		providerResolver: providerResolver,
		pollInterval:     pollInterval,
		batchSize:        batchSize,
		matchTimeout:     matchTimeout,
		providerTimeouts: providerTimeouts,
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
	w.log().Info("matcher worker starting",
		"default_provider", w.providerName,
		"poll_interval", pollInterval,
		"batch_size", w.batchSize,
		"match_timeout", w.matchTimeout,
		"stale_after", w.staleAfter,
	)

	go func() {
		defer close(done)
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		w.requeueStale(ctx)
		w.processBatch(ctx)

		for {
			select {
			case <-ctx.Done():
				w.log().Info("matcher worker stopped")
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

// SetLogger allows callers to inject a shared logger configuration.
func (w *Worker) SetLogger(logger *slog.Logger) {
	if logger == nil {
		return
	}
	w.stateMu.Lock()
	w.logger = logger
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
		w.log().Warn("requeued stale processing matches", "count", requeued)
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
			w.log().Error("matcher failed to claim pending row", "error", err)
			break
		}
		if claimed == nil {
			break
		}

		w.emitStatusChanged(claimed.JobID, db.JobMatchStatusProcessing)
		if err := w.processOne(ctx, claimed.JobID); err != nil {
			w.log().Error("matcher failed to process claimed row",
				"job_id", claimed.JobID,
				"default_provider", w.providerName,
				"error", err,
			)
		}
		processed++
	}
	if processed > 0 {
		w.log().Info("match batch processed",
			"count", processed,
			"default_provider", w.providerName,
		)
	}
	return processed
}

func (w *Worker) processOne(ctx context.Context, jobID string) error {
	job, err := w.store.GetJob(ctx, jobID)
	if err != nil {
		return w.failMatch(ctx, jobID, "", fmt.Sprintf("Loading job failed: %v", err))
	}
	if job == nil {
		// Job was removed after queueing; ignore if match row was cascaded away.
		if err := w.store.MarkJobMatchFailed(ctx, jobID, "Job no longer exists."); err != nil && !errors.Is(err, db.ErrJobMatchNotFound) {
			return fmt.Errorf("marking missing job as failed: %w", err)
		}
		return nil
	}

	providerName, provider, err := w.providerResolver(ctx)
	if err != nil {
		return w.failMatch(ctx, jobID, providerName, fmt.Sprintf("Provider resolution failed: %v", err))
	}
	if provider == nil {
		return w.failMatch(ctx, jobID, providerName, "Provider resolution returned no provider.")
	}
	w.log().Info("match calculation started",
		"job_id", jobID,
		"provider", providerName,
	)

	query := w.queryForJob(ctx, job)
	matchTimeout := w.matchTimeout
	if timeout, ok := w.providerTimeouts[strings.TrimSpace(providerName)]; ok && timeout > 0 {
		matchTimeout = timeout
	}

	matchCtx, cancel := context.WithTimeout(ctx, matchTimeout)
	defer cancel()

	result, err := provider.Match(matchCtx, llm.MatchRequest{
		Query:               query,
		CandidateProfile:    w.cvProfileForJob(ctx),
		JobTitle:            job.Title,
		JobCompany:          job.Company,
		JobLocation:         job.Location,
		JobURL:              sourceJobURL(job.Source, job.URL),
		JobDescription:      job.Description,
		JobDescriptionNote:  sourceDescriptionNote(job.Source, job.URL),
		MaxDescriptionRunes: w.descriptionRunes,
	})
	if err != nil {
		return w.failMatch(ctx, jobID, providerName, fmt.Sprintf("Provider scoring failed: %v", err))
	}

	result.Summary = attachProviderSummary(providerName, result.Summary)
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
	w.log().Info("match calculation completed",
		"job_id", jobID,
		"provider", providerName,
		"score", score,
		"estimated_prompt_tokens", result.EstimatedPromptTokens,
	)
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

func sourceJobURL(source, jobURL string) string {
	if strings.ToLower(strings.TrimSpace(source)) != adzunaSourceName {
		return ""
	}
	return strings.TrimSpace(jobURL)
}

func sourceDescriptionNote(source, jobURL string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case adzunaSourceName:
		url := strings.TrimSpace(jobURL)
		if url == "" {
			return "Adzuna provides a description snippet, not the full job ad."
		}
		return "Adzuna provides a description snippet, not the full job ad. The prompt also includes the job URL for additional context when URL access is available."
	default:
		return ""
	}
}

func (w *Worker) cvProfileForJob(ctx context.Context) string {
	cvPath, err := w.store.GetSetting(ctx, settingCVPath)
	if err != nil {
		w.log().Warn("failed to load cv path for matching context", "error", err)
		return ""
	}

	cvPath = strings.TrimSpace(cvPath)
	if cvPath == "" {
		w.clearCVCache()
		return ""
	}

	profile, err := w.loadCVProfile(cvPath)
	if err != nil {
		w.log().Warn(
			"cv profile unavailable; continuing without cv context",
			"error", err,
		)
		return ""
	}
	return profile
}

func (w *Worker) clearCVCache() {
	w.cvCacheMu.Lock()
	w.cvCachePath = ""
	w.cvCacheModTime = time.Time{}
	w.cvCacheSize = 0
	w.cvCacheProfile = ""
	w.cvCacheMu.Unlock()
}

func (w *Worker) loadCVProfile(cvPath string) (string, error) {
	info, err := os.Stat(cvPath)
	if err != nil {
		return "", fmt.Errorf("checking cv file metadata: %w", err)
	}
	if info.IsDir() {
		return "", errors.New("cv path points to a directory")
	}

	w.cvCacheMu.Lock()
	defer w.cvCacheMu.Unlock()
	if w.cvCachePath == cvPath &&
		w.cvCacheModTime.Equal(info.ModTime()) &&
		w.cvCacheSize == info.Size() &&
		w.cvCacheProfile != "" {
		return w.cvCacheProfile, nil
	}

	profile, err := cv.ExtractProfile(cvPath)
	if err != nil {
		return "", err
	}
	w.cvCachePath = cvPath
	w.cvCacheModTime = info.ModTime()
	w.cvCacheSize = info.Size()
	w.cvCacheProfile = profile
	return profile, nil
}

func (w *Worker) failMatch(ctx context.Context, jobID, providerName, reason string) error {
	if err := w.store.MarkJobMatchFailed(ctx, jobID, attachProviderSummary(providerName, reason)); err != nil {
		if errors.Is(err, db.ErrJobMatchNotFound) {
			return nil
		}
		return err
	}
	w.log().Warn("match calculation failed",
		"job_id", jobID,
		"provider", providerName,
		"reason", reason,
	)
	w.emitStatusChanged(jobID, db.JobMatchStatusFailed)
	return nil
}

func attachProviderSummary(providerName, summary string) string {
	trimmedSummary := strings.TrimSpace(summary)
	if trimmedSummary == "" {
		trimmedSummary = "Match computed."
	}

	trimmedProvider := strings.TrimSpace(providerName)
	if trimmedProvider == "" {
		return trimmedSummary
	}
	return fmt.Sprintf("Provider: %s\n%s", trimmedProvider, trimmedSummary)
}

func (w *Worker) emitStatusChanged(jobID, status string) {
	w.stateMu.RLock()
	hook := w.statusChangedHook
	w.stateMu.RUnlock()
	if hook != nil {
		hook(jobID, status)
	}
}

func (w *Worker) log() *slog.Logger {
	w.stateMu.RLock()
	logger := w.logger
	w.stateMu.RUnlock()
	if logger == nil {
		return slog.Default()
	}
	return logger
}
