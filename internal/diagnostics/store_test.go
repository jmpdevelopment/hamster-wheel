package diagnostics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSavePollRunWritesJSONAndDefaults(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, 10, 24*time.Hour)

	fixed := time.Date(2026, 2, 14, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return fixed }

	path, err := store.SavePollRun(PollRun{})
	if err != nil {
		t.Fatalf("saving poll run: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading saved poll run: %v", err)
	}

	var got PollRun
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshalling saved poll run: %v", err)
	}

	if got.RunID == "" {
		t.Fatal("expected run ID to be auto-generated")
	}
	if !got.StartedAt.Equal(fixed) {
		t.Fatalf("expected StartedAt=%s, got %s", fixed, got.StartedAt)
	}
	if !got.CompletedAt.Equal(fixed) {
		t.Fatalf("expected CompletedAt=%s, got %s", fixed, got.CompletedAt)
	}
	if got.DurationMs != 0 {
		t.Fatalf("expected DurationMs=0, got %d", got.DurationMs)
	}
}

func TestCleanupRemovesExpiredFiles(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 2, 14, 18, 0, 0, 0, time.UTC)
	store := NewStore(dir, 10, 24*time.Hour)
	store.now = func() time.Time { return now }

	oldPath, err := store.SavePollRun(PollRun{RunID: "old"})
	if err != nil {
		t.Fatalf("saving old poll run: %v", err)
	}
	newPath, err := store.SavePollRun(PollRun{RunID: "new"})
	if err != nil {
		t.Fatalf("saving new poll run: %v", err)
	}

	threeHoursAgo := now.Add(-3 * time.Hour)
	if err := os.Chtimes(oldPath, threeHoursAgo, threeHoursAgo); err != nil {
		t.Fatalf("setting old file mod time: %v", err)
	}
	thirtyMinsAgo := now.Add(-30 * time.Minute)
	if err := os.Chtimes(newPath, thirtyMinsAgo, thirtyMinsAgo); err != nil {
		t.Fatalf("setting new file mod time: %v", err)
	}

	store.maxAge = time.Hour
	if err := store.Cleanup(); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expected old file to be removed, stat err=%v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("expected new file to remain, stat err=%v", err)
	}
}

func TestCleanupEnforcesMaxFiles(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 2, 14, 20, 0, 0, 0, time.UTC)
	store := NewStore(dir, 10, 24*time.Hour)
	store.now = func() time.Time { return now }

	path1, err := store.SavePollRun(PollRun{RunID: "one"})
	if err != nil {
		t.Fatalf("saving run one: %v", err)
	}
	path2, err := store.SavePollRun(PollRun{RunID: "two"})
	if err != nil {
		t.Fatalf("saving run two: %v", err)
	}
	path3, err := store.SavePollRun(PollRun{RunID: "three"})
	if err != nil {
		t.Fatalf("saving run three: %v", err)
	}

	if err := os.Chtimes(path1, now.Add(-3*time.Hour), now.Add(-3*time.Hour)); err != nil {
		t.Fatalf("setting run one mod time: %v", err)
	}
	if err := os.Chtimes(path2, now.Add(-2*time.Hour), now.Add(-2*time.Hour)); err != nil {
		t.Fatalf("setting run two mod time: %v", err)
	}
	if err := os.Chtimes(path3, now.Add(-1*time.Hour), now.Add(-1*time.Hour)); err != nil {
		t.Fatalf("setting run three mod time: %v", err)
	}

	store.maxFiles = 2
	if err := store.Cleanup(); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	if _, err := os.Stat(path1); !os.IsNotExist(err) {
		t.Fatalf("expected oldest run file to be removed, stat err=%v", err)
	}
	if _, err := os.Stat(path2); err != nil {
		t.Fatalf("expected second run file to remain, stat err=%v", err)
	}
	if _, err := os.Stat(path3); err != nil {
		t.Fatalf("expected newest run file to remain, stat err=%v", err)
	}
}

func TestCleanupMissingDirNoError(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "missing"), 10, 24*time.Hour)
	if err := store.Cleanup(); err != nil {
		t.Fatalf("expected no error cleaning missing dir, got %v", err)
	}
}

func TestNewStoreAppliesDefaultRetentionValues(t *testing.T) {
	store := NewStore(t.TempDir(), 0, 0)
	if store.maxFiles != 50 {
		t.Fatalf("expected default maxFiles=50, got %d", store.maxFiles)
	}
	if store.maxAge != 24*time.Hour {
		t.Fatalf("expected default maxAge=24h, got %s", store.maxAge)
	}
}

func TestDirReturnsConfiguredDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "diagnostics")
	store := NewStore(dir, 5, time.Hour)
	if got := store.Dir(); got != dir {
		t.Fatalf("expected dir %q, got %q", dir, got)
	}
}
