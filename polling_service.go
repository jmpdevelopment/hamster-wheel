package main

import (
	"context"
	"time"

	"hamster-wheel/internal/scheduler"
)

// PollingService handles job polling and scheduler control operations exposed to the frontend.
type PollingService struct {
	scheduler *scheduler.Scheduler
}

// NewPollingService creates a new PollingService.
func NewPollingService(sched *scheduler.Scheduler) *PollingService {
	return &PollingService{
		scheduler: sched,
	}
}

// pollNowTimeout caps how long a manual poll can run before being cancelled.
// This prevents the Wails binding from blocking the UI indefinitely if the
// Reed API is slow or unresponsive.
const pollNowTimeout = 5 * time.Minute

// PollNow triggers an immediate poll cycle and returns the results.
func (s *PollingService) PollNow() ([]scheduler.PollResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pollNowTimeout)
	defer cancel()
	return s.scheduler.PollOnce(ctx)
}

// PollingStatus represents the current state of auto-polling.
type PollingStatus struct {
	Paused     bool   `json:"paused"`
	NextPollAt string `json:"nextPollAt"` // RFC3339 timestamp, empty if not scheduled
}

// GetPollingStatus returns whether auto-polling is paused and when the next poll is due.
func (s *PollingService) GetPollingStatus() PollingStatus {
	next := s.scheduler.NextPollAt()
	nextStr := ""
	if !next.IsZero() {
		nextStr = next.Format(time.RFC3339)
	}
	return PollingStatus{
		Paused:     s.scheduler.IsPaused(),
		NextPollAt: nextStr,
	}
}

// SetPollingPaused pauses or resumes auto-polling. Manual PollNow still works.
func (s *PollingService) SetPollingPaused(paused bool) {
	s.scheduler.SetPaused(paused)
}
