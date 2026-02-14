package diagnostics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	filePrefix = "poll-run-"
	fileSuffix = ".json"
)

// FilterResult stores per-filter diagnostics for a poll run.
type FilterResult struct {
	FilterID   string `json:"filterID"`
	FilterName string `json:"filterName"`
	Source     string `json:"source"`
	NewJobs    int    `json:"newJobs"`
	Skipped    int    `json:"skipped"`
	Error      string `json:"error,omitempty"`
}

// PollRun stores diagnostics for a single poll cycle.
type PollRun struct {
	RunID         string         `json:"runID"`
	StartedAt     time.Time      `json:"startedAt"`
	CompletedAt   time.Time      `json:"completedAt"`
	DurationMs    int64          `json:"durationMs"`
	TotalFilters  int            `json:"totalFilters"`
	FailedFilters int            `json:"failedFilters"`
	NewJobs       int            `json:"newJobs"`
	Skipped       int            `json:"skipped"`
	CycleError    string         `json:"cycleError,omitempty"`
	Filters       []FilterResult `json:"filters,omitempty"`
}

type fileInfo struct {
	name    string
	path    string
	modTime time.Time
}

// Store persists poll diagnostics in a temp directory with bounded retention.
type Store struct {
	dir      string
	maxFiles int
	maxAge   time.Duration
	now      func() time.Time
	mu       sync.Mutex
}

// NewStore creates a diagnostics store.
func NewStore(dir string, maxFiles int, maxAge time.Duration) *Store {
	if maxFiles <= 0 {
		maxFiles = 50
	}
	if maxAge <= 0 {
		maxAge = 24 * time.Hour
	}
	return &Store{
		dir:      dir,
		maxFiles: maxFiles,
		maxAge:   maxAge,
		now:      time.Now,
	}
}

// Dir returns the diagnostics directory.
func (s *Store) Dir() string {
	return s.dir
}

// SavePollRun persists one poll run and returns the created file path.
func (s *Store) SavePollRun(run PollRun) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return "", fmt.Errorf("creating diagnostics dir: %w", err)
	}

	now := s.now().UTC()
	if run.RunID == "" {
		run.RunID = uuid.NewString()
	}
	if run.StartedAt.IsZero() {
		run.StartedAt = now
	}
	if run.CompletedAt.IsZero() {
		run.CompletedAt = now
	}
	if run.DurationMs == 0 && run.CompletedAt.After(run.StartedAt) {
		run.DurationMs = run.CompletedAt.Sub(run.StartedAt).Milliseconds()
	}

	name := fmt.Sprintf("%s%s%s", filePrefix, run.RunID, fileSuffix)
	path := filepath.Join(s.dir, name)
	tmpPath := path + ".tmp"

	payload, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshalling poll diagnostics: %w", err)
	}

	if err := os.WriteFile(tmpPath, payload, 0o600); err != nil {
		return "", fmt.Errorf("writing temp diagnostics file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("finalizing diagnostics file: %w", err)
	}

	if err := s.cleanupLocked(); err != nil {
		return path, fmt.Errorf("cleaning diagnostics: %w", err)
	}

	return path, nil
}

// Cleanup removes diagnostics files that exceed retention limits.
func (s *Store) Cleanup() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cleanupLocked()
}

func (s *Store) cleanupLocked() error {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading diagnostics dir: %w", err)
	}

	files := make([]fileInfo, 0, len(entries))
	cutoff := s.now().Add(-s.maxAge)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, filePrefix) || !strings.HasSuffix(name, fileSuffix) {
			continue
		}

		path := filepath.Join(s.dir, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			_ = os.Remove(path)
			continue
		}

		files = append(files, fileInfo{
			name:    name,
			path:    path,
			modTime: info.ModTime(),
		})
	}

	if len(files) <= s.maxFiles {
		return nil
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	toDelete := len(files) - s.maxFiles
	for i := 0; i < toDelete; i++ {
		_ = os.Remove(files[i].path)
	}

	return nil
}
