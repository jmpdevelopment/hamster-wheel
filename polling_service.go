package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"hamster-wheel/internal/diagnostics"
	"hamster-wheel/internal/scheduler"

	"github.com/google/uuid"
)

// PollingService handles job polling and scheduler control operations exposed to the frontend.
type PollingService struct {
	scheduler   *scheduler.Scheduler
	diagnostics *diagnostics.Store
}

// PollFilterResult reports the outcome of polling one filter.
type PollFilterResult struct {
	FilterID   string `json:"filterID"`
	FilterName string `json:"filterName"`
	Source     string `json:"source"`
	NewJobs    int    `json:"newJobs"`
	Skipped    int    `json:"skipped"`
	Error      string `json:"error,omitempty"`
}

// PollRunResult reports one manual poll run and where its diagnostics were stored.
type PollRunResult struct {
	RunID            string             `json:"runID"`
	StartedAt        string             `json:"startedAt"`   // RFC3339 timestamp
	CompletedAt      string             `json:"completedAt"` // RFC3339 timestamp
	DurationMs       int64              `json:"durationMs"`
	TotalFilters     int                `json:"totalFilters"`
	FailedFilters    int                `json:"failedFilters"`
	NewJobs          int                `json:"newJobs"`
	Skipped          int                `json:"skipped"`
	CycleError       string             `json:"cycleError,omitempty"`
	DiagnosticsPath  string             `json:"diagnosticsPath,omitempty"`
	DiagnosticsError string             `json:"diagnosticsError,omitempty"`
	Filters          []PollFilterResult `json:"filters,omitempty"`
}

// NewPollingService creates a new PollingService.
func NewPollingService(sched *scheduler.Scheduler, store *diagnostics.Store) *PollingService {
	return &PollingService{
		scheduler:   sched,
		diagnostics: store,
	}
}

// pollNowTimeout caps how long a manual poll can run before being cancelled.
// This prevents the Wails binding from blocking the UI indefinitely if the
// Reed API is slow or unresponsive.
const pollNowTimeout = 5 * time.Minute
const maxPollReportBytes = 512 * 1024 // Keep report exports bounded to avoid accidental large writes.

// PollNow triggers an immediate poll cycle and returns structured run metadata.
//
// Poll-cycle failures are reported in CycleError while preserving diagnostics.
// The call only returns a transport error if the runtime itself fails.
func (s *PollingService) PollNow() PollRunResult {
	runID := uuid.NewString()
	startedAt := time.Now().UTC()

	ctx, cancel := context.WithTimeout(context.Background(), pollNowTimeout)
	defer cancel()

	// Manual polls reset the next auto-poll window to now + interval.
	s.scheduler.RescheduleFromNow()
	schedResults, cycleErr := s.scheduler.PollOnce(ctx)
	completedAt := time.Now().UTC()

	result := PollRunResult{
		RunID:       runID,
		StartedAt:   startedAt.Format(time.RFC3339),
		CompletedAt: completedAt.Format(time.RFC3339),
		DurationMs:  completedAt.Sub(startedAt).Milliseconds(),
	}
	if result.DurationMs < 0 {
		result.DurationMs = 0
	}

	diagRun := diagnostics.PollRun{
		RunID:       runID,
		StartedAt:   startedAt,
		CompletedAt: completedAt,
		DurationMs:  result.DurationMs,
	}

	if cycleErr != nil {
		result.CycleError = cycleErr.Error()
		diagRun.CycleError = cycleErr.Error()
	}

	if len(schedResults) > 0 {
		result.Filters = make([]PollFilterResult, 0, len(schedResults))
		diagRun.Filters = make([]diagnostics.FilterResult, 0, len(schedResults))
	}

	for _, r := range schedResults {
		filterResult := PollFilterResult{
			FilterID:   r.FilterID,
			FilterName: r.FilterName,
			Source:     r.Source,
			NewJobs:    r.NewJobs,
			Skipped:    r.Skipped,
		}
		diagFilterResult := diagnostics.FilterResult{
			FilterID:   r.FilterID,
			FilterName: r.FilterName,
			Source:     r.Source,
			NewJobs:    r.NewJobs,
			Skipped:    r.Skipped,
		}
		if r.Err != nil {
			filterResult.Error = r.Err.Error()
			diagFilterResult.Error = r.Err.Error()
			result.FailedFilters++
		}

		result.NewJobs += r.NewJobs
		result.Skipped += r.Skipped
		result.Filters = append(result.Filters, filterResult)
		diagRun.Filters = append(diagRun.Filters, diagFilterResult)
	}

	result.TotalFilters = len(result.Filters)
	diagRun.TotalFilters = result.TotalFilters
	diagRun.FailedFilters = result.FailedFilters
	diagRun.NewJobs = result.NewJobs
	diagRun.Skipped = result.Skipped

	// No enabled filters and no cycle error: skip diagnostics persistence for this no-op.
	if result.TotalFilters == 0 && result.CycleError == "" {
		return result
	}

	if s.diagnostics == nil {
		result.DiagnosticsError = "diagnostics store unavailable"
		slog.Error("poll diagnostics store unavailable", "runID", runID)
		return result
	}

	path, err := s.diagnostics.SavePollRun(diagRun)
	if path != "" {
		result.DiagnosticsPath = path
	}
	if err != nil {
		result.DiagnosticsError = err.Error()
		slog.Error("failed to persist poll diagnostics", "runID", runID, "error", err)
		return result
	}

	return result
}

// SavePollReport writes a user-exported poll report to the selected file path.
func (s *PollingService) SavePollReport(path, content string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("save path is required")
	}
	if strings.TrimSpace(content) == "" {
		return errors.New("report content is empty")
	}
	if len(content) > maxPollReportBytes {
		return fmt.Errorf("report too large (%d bytes): max %d bytes", len(content), maxPollReportBytes)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating report directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("writing poll report: %w", err)
	}
	return nil
}

// PollingStatus represents the current state of auto-polling.
type PollingStatus struct {
	Paused     bool           `json:"paused"`
	IsPolling  bool           `json:"isPolling"`
	NextPollAt string         `json:"nextPollAt"` // RFC3339 timestamp, empty if not scheduled
	LastRun    *PollRunResult `json:"lastRun,omitempty"`
}

// GetPollingStatus returns whether auto-polling is paused and when the next poll is due.
func (s *PollingService) GetPollingStatus() PollingStatus {
	next := s.scheduler.NextPollAt()
	nextStr := ""
	if !next.IsZero() {
		nextStr = next.Format(time.RFC3339)
	}
	status := PollingStatus{
		Paused:     s.scheduler.IsPaused(),
		IsPolling:  s.scheduler.IsPolling(),
		NextPollAt: nextStr,
	}
	if summary, ok := s.scheduler.LastPollSummary(); ok {
		run := pollRunFromSchedulerSummary(summary)
		status.LastRun = &run
	}
	return status
}

// SetPollingPaused pauses or resumes auto-polling. Manual PollNow still works.
func (s *PollingService) SetPollingPaused(paused bool) {
	s.scheduler.SetPaused(paused)
}

func pollRunFromSchedulerSummary(summary scheduler.PollCycleSummary) PollRunResult {
	completedAt := summary.CompletedAt.UTC()
	completedAtRFC3339 := ""
	if !completedAt.IsZero() {
		completedAtRFC3339 = completedAt.Format(time.RFC3339)
	}

	run := PollRunResult{
		RunID:         fmt.Sprintf("auto-%d", completedAt.UnixNano()),
		StartedAt:     completedAtRFC3339,
		CompletedAt:   completedAtRFC3339,
		DurationMs:    0,
		TotalFilters:  summary.TotalFilters,
		FailedFilters: summary.FailedFilters,
		NewJobs:       summary.NewJobs,
		Skipped:       summary.Skipped,
	}
	if summary.CycleError != "" {
		run.CycleError = summary.CycleError
	}
	if len(summary.Filters) > 0 {
		run.Filters = make([]PollFilterResult, 0, len(summary.Filters))
		for _, filterSummary := range summary.Filters {
			filter := PollFilterResult{
				FilterID:   filterSummary.FilterID,
				FilterName: filterSummary.FilterName,
				Source:     filterSummary.Source,
				NewJobs:    filterSummary.NewJobs,
				Skipped:    filterSummary.Skipped,
			}
			if filterSummary.Error != "" {
				filter.Error = filterSummary.Error
			}
			run.Filters = append(run.Filters, filter)
		}
	}
	return run
}
