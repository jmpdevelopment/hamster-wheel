package main

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"hamster-wheel/internal/adapter"
	"hamster-wheel/internal/adapter/reed"
	"hamster-wheel/internal/db"
	"hamster-wheel/internal/diagnostics"
	"hamster-wheel/internal/keychain"
	"hamster-wheel/internal/llm"
	"hamster-wheel/internal/llm/heuristic"
	"hamster-wheel/internal/matcher"
	"hamster-wheel/internal/scheduler"
)

// --- Test Helpers ---

// testDB creates a temporary database for testing. Cleaned up automatically.
func testDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.OpenAt(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

// testServices creates all 5 services wired to a real temp DB.
// Returns them along with the shared dependencies for inspection.
func testServices(t *testing.T) (
	*AppService, *JobService, *FilterService, *PollingService, *SettingsService,
	*db.DB, *scheduler.Scheduler, *reed.Adapter,
) {
	t.Helper()

	database := testDB(t)
	reedAdapter := reed.New("test-key")
	adapters := adapter.NewRegistry()
	if err := adapters.Register(reedAdapter); err != nil {
		t.Fatalf("registering adapter: %v", err)
	}

	sched := scheduler.New(database, adapters, 1*time.Hour)
	providers := llm.NewRegistry()
	if err := providers.Register(heuristic.New()); err != nil {
		t.Fatalf("registering heuristic provider: %v", err)
	}
	matchWorker := matcher.New(database, providers, matcher.WorkerConfig{
		ProviderName: heuristic.ProviderName,
		PollInterval: time.Second,
		BatchSize:    2,
	})
	diagStore := diagnostics.NewStore(t.TempDir(), 50, 24*time.Hour)
	kc := keychain.NewMemoryStore()

	appSvc := NewAppService(database, sched, matchWorker)
	jobSvc := NewJobService(database, adapters)
	filterSvc := NewFilterService(database)
	pollSvc := NewPollingService(sched, diagStore)
	settingsSvc := NewSettingsService(database, kc, reedAdapter)

	return appSvc, jobSvc, filterSvc, pollSvc, settingsSvc, database, sched, reedAdapter
}

// --- 1. Service Integration Tests ---

func TestAllServicesConstruct(t *testing.T) {
	appSvc, jobSvc, filterSvc, pollSvc, settingsSvc, _, _, _ := testServices(t)

	// Every service should be non-nil.
	if appSvc == nil {
		t.Error("AppService is nil")
	}
	if jobSvc == nil {
		t.Error("JobService is nil")
	}
	if filterSvc == nil {
		t.Error("FilterService is nil")
	}
	if pollSvc == nil {
		t.Error("PollingService is nil")
	}
	if settingsSvc == nil {
		t.Error("SettingsService is nil")
	}
}

func TestServicesShareDatabase(t *testing.T) {
	_, jobSvc, filterSvc, _, _, database, _, _ := testServices(t)

	// Create a filter via FilterService.
	filterID, err := filterSvc.CreateFilter("Test Filter", "golang", "London", "reed_uk")
	if err != nil {
		t.Fatalf("creating filter: %v", err)
	}

	// Verify filter exists via FilterService.
	filter, err := filterSvc.GetFilter(filterID)
	if err != nil {
		t.Fatalf("getting filter: %v", err)
	}
	if filter == nil {
		t.Fatal("filter not found after creation")
	}
	if filter.Name != "Test Filter" {
		t.Errorf("expected filter name %q, got %q", "Test Filter", filter.Name)
	}

	// JobService should see 0 jobs initially.
	count, err := jobSvc.GetJobCount()
	if err != nil {
		t.Fatalf("counting jobs: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 jobs initially, got %d", count)
	}

	// Insert a job via the shared database (JobService is read/delete only).
	// This verifies both services share the same underlying DB instance.
	ctx := context.Background()
	job := &db.Job{
		Source:   "reed_uk",
		SourceID: "arch-test-1",
		Title:    "Go Developer",
		Company:  "TestCo",
		Location: "London",
		URL:      "https://example.com/1",
		FilterID: &filterID,
	}
	_, err = database.InsertJob(ctx, job)
	if err != nil {
		t.Fatalf("inserting job: %v", err)
	}

	// JobService should now see the job inserted via the shared DB.
	count, err = jobSvc.GetJobCount()
	if err != nil {
		t.Fatalf("counting jobs after insert: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 job after insert, got %d", count)
	}

	// Verify job is associated with the filter created by FilterService.
	jobs, err := jobSvc.GetJobsByFilter(filterID)
	if err != nil {
		t.Fatalf("getting jobs by filter: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job for filter, got %d", len(jobs))
	}
	if jobs[0].Title != "Go Developer" {
		t.Errorf("expected job title %q, got %q", "Go Developer", jobs[0].Title)
	}
}

func TestAppServiceLifecycle(t *testing.T) {
	appSvc, _, _, pollSvc, _, _, sched, _ := testServices(t)

	// Before startup, scheduler is not running.
	status := pollSvc.GetPollingStatus()
	if status.NextPollAt != "" {
		t.Errorf("expected empty NextPollAt before startup, got %q", status.NextPollAt)
	}

	// Simulate Wails startup (we don't have a real ServiceOptions, but the method ignores it).
	// We need to import the application package for ServiceOptions.
	// Since we can't easily construct a real ServiceOptions, we test Start/Stop directly.
	sched.Start()
	t.Cleanup(func() { sched.Stop() })

	// After starting, scheduler should have a next poll time.
	// Give it a moment to run the initial poll and set nextPollAt.
	time.Sleep(100 * time.Millisecond)

	status = pollSvc.GetPollingStatus()
	if status.NextPollAt == "" {
		t.Error("expected non-empty NextPollAt after scheduler start")
	}

	// Pause via PollingService, verify via PollingService.
	pollSvc.SetPollingPaused(true)
	status = pollSvc.GetPollingStatus()
	if !status.Paused {
		t.Error("expected paused=true after SetPollingPaused(true)")
	}

	// AppService shutdown should stop the scheduler and close the DB.
	err := appSvc.ServiceShutdown()
	if err != nil {
		t.Fatalf("service shutdown failed: %v", err)
	}
}

func TestSettingsServiceUpdatesReedAdapter(t *testing.T) {
	_, _, _, _, settingsSvc, _, _, reedAdapter := testServices(t)

	// No key should be set by default.
	has, err := settingsSvc.HasReedAPIKey()
	if err != nil {
		t.Fatalf("checking initial Reed API key: %v", err)
	}
	if has {
		t.Fatal("expected HasReedAPIKey=false initially")
	}

	// Set a new API key via SettingsService.
	if err := settingsSvc.SetReedAPIKey("new-test-key-123"); err != nil {
		t.Fatalf("setting Reed API key: %v", err)
	}

	// Verify it persists as non-secret state and updates adapter.
	has, err = settingsSvc.HasReedAPIKey()
	if err != nil {
		t.Fatalf("checking Reed API key presence: %v", err)
	}
	if !has {
		t.Fatal("expected HasReedAPIKey=true after setting key")
	}
	if !reedAdapter.HasAPIKey() {
		t.Fatal("expected adapter key to be set")
	}

	// Clear the key and verify adapter state is updated immediately.
	if err := settingsSvc.ClearReedAPIKey(); err != nil {
		t.Fatalf("clearing Reed API key: %v", err)
	}
	has, err = settingsSvc.HasReedAPIKey()
	if err != nil {
		t.Fatalf("checking Reed API key after clear: %v", err)
	}
	if has {
		t.Fatal("expected HasReedAPIKey=false after clear")
	}
	if reedAdapter.HasAPIKey() {
		t.Fatal("expected adapter key to be cleared")
	}
}

func TestSettingsServiceTheme(t *testing.T) {
	_, _, _, _, settingsSvc, _, _, _ := testServices(t)

	// Default theme is empty string (no preference set).
	theme, err := settingsSvc.GetTheme()
	if err != nil {
		t.Fatalf("getting default theme: %v", err)
	}
	if theme != "" {
		t.Errorf("expected empty default theme, got %q", theme)
	}

	// Set theme to each valid value.
	for _, valid := range []string{"dark", "light", "system"} {
		if err := settingsSvc.SetTheme(valid); err != nil {
			t.Fatalf("setting theme to %q: %v", valid, err)
		}
		got, err := settingsSvc.GetTheme()
		if err != nil {
			t.Fatalf("getting theme after setting %q: %v", valid, err)
		}
		if got != valid {
			t.Errorf("expected theme %q, got %q", valid, got)
		}
	}

	// Invalid theme should return error.
	if err := settingsSvc.SetTheme("purple"); err == nil {
		t.Error("expected error for invalid theme, got nil")
	}
}

func TestSettingsServiceKeyboardShortcuts(t *testing.T) {
	_, _, _, _, settingsSvc, _, _, _ := testServices(t)

	// Default is empty string (no preference set — frontend treats as enabled).
	val, err := settingsSvc.GetKeyboardShortcuts()
	if err != nil {
		t.Fatalf("getting default keyboard shortcuts: %v", err)
	}
	if val != "" {
		t.Errorf("expected empty default, got %q", val)
	}

	// Set to each valid value.
	for _, valid := range []string{"true", "false"} {
		if err := settingsSvc.SetKeyboardShortcuts(valid); err != nil {
			t.Fatalf("setting keyboard shortcuts to %q: %v", valid, err)
		}
		got, err := settingsSvc.GetKeyboardShortcuts()
		if err != nil {
			t.Fatalf("getting keyboard shortcuts after setting %q: %v", valid, err)
		}
		if got != valid {
			t.Errorf("expected %q, got %q", valid, got)
		}
	}

	// Invalid value should return error.
	if err := settingsSvc.SetKeyboardShortcuts("maybe"); err == nil {
		t.Error("expected error for invalid value, got nil")
	}
}

// --- 2. Service Boundary Validation ---

func TestServiceDependencies(t *testing.T) {
	// Verify each service struct has exactly the expected fields.
	// This catches accidental cross-service dependencies being added.

	tests := []struct {
		name           string
		serviceType    reflect.Type
		expectedFields map[string]string // field name → type string
	}{
		{
			name:        "JobService",
			serviceType: reflect.TypeOf(JobService{}),
			expectedFields: map[string]string{
				"db":       "*db.DB",
				"adapters": "*adapter.Registry",
			},
		},
		{
			name:        "FilterService",
			serviceType: reflect.TypeOf(FilterService{}),
			expectedFields: map[string]string{
				"db": "*db.DB",
			},
		},
		{
			name:        "PollingService",
			serviceType: reflect.TypeOf(PollingService{}),
			expectedFields: map[string]string{
				"scheduler":   "*scheduler.Scheduler",
				"diagnostics": "*diagnostics.Store",
			},
		},
		{
			name:        "SettingsService",
			serviceType: reflect.TypeOf(SettingsService{}),
			expectedFields: map[string]string{
				"db":           "*db.DB",
				"keychain":     "keychain.Store",
				"reedAdapter":  "*reed.Adapter",
				"localRuntime": "localruntime.Manager",
			},
		},
		{
			name:        "AppService",
			serviceType: reflect.TypeOf(AppService{}),
			expectedFields: map[string]string{
				"db":        "*db.DB",
				"scheduler": "*scheduler.Scheduler",
				"matcher":   "*matcher.Worker",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			numFields := tt.serviceType.NumField()

			// Check field count matches.
			if numFields != len(tt.expectedFields) {
				var actual []string
				for i := 0; i < numFields; i++ {
					f := tt.serviceType.Field(i)
					actual = append(actual, f.Name+" "+f.Type.String())
				}
				t.Errorf("%s has %d fields (expected %d): %v",
					tt.name, numFields, len(tt.expectedFields), actual)
				return
			}

			// Check each expected field exists with correct type.
			for fieldName, expectedType := range tt.expectedFields {
				field, found := tt.serviceType.FieldByName(fieldName)
				if !found {
					t.Errorf("%s: missing expected field %q", tt.name, fieldName)
					continue
				}
				actualType := field.Type.String()
				if actualType != expectedType {
					t.Errorf("%s.%s: expected type %q, got %q",
						tt.name, fieldName, expectedType, actualType)
				}
			}
		})
	}
}

func TestNoServiceHasCrossServiceDependencies(t *testing.T) {
	// Verify no service struct has a field that's another service type.
	// This ensures services don't depend on each other directly.

	serviceTypes := []reflect.Type{
		reflect.TypeOf(AppService{}),
		reflect.TypeOf(JobService{}),
		reflect.TypeOf(FilterService{}),
		reflect.TypeOf(PollingService{}),
		reflect.TypeOf(SettingsService{}),
	}

	serviceTypeNames := make(map[string]bool)
	for _, st := range serviceTypes {
		serviceTypeNames[st.Name()] = true
		serviceTypeNames["*"+st.Name()] = true
	}

	for _, st := range serviceTypes {
		for i := 0; i < st.NumField(); i++ {
			field := st.Field(i)
			fieldTypeName := field.Type.Name()
			if field.Type.Kind() == reflect.Ptr {
				fieldTypeName = field.Type.Elem().Name()
			}
			if serviceTypeNames[fieldTypeName] {
				t.Errorf("%s has a field %q of service type %q — services should not depend on each other",
					st.Name(), field.Name, fieldTypeName)
			}
		}
	}
}

// --- 3. Binding Contract Tests ---

func TestBindingFilesExist(t *testing.T) {
	bindingsDir := filepath.Join("frontend", "bindings", "hamster-wheel")

	expectedFiles := []string{
		"jobservice.js",
		"filterservice.js",
		"pollingservice.js",
		"settingsservice.js",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(bindingsDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("binding file missing: %s", path)
		}
	}
}

func TestBindingFilesDontReferenceAppService(t *testing.T) {
	// After the service split, no binding file should import from 'appservice'.
	// This catches stale imports that would break at runtime.
	bindingsDir := filepath.Join("frontend", "bindings", "hamster-wheel")

	// The old appservice.js should not exist.
	oldPath := filepath.Join(bindingsDir, "appservice.js")
	if _, err := os.Stat(oldPath); err == nil {
		t.Error("stale binding file exists: appservice.js — should have been removed after service split")
	}
}

func TestBindingContractsMatchGoMethods(t *testing.T) {
	// Parse each binding JS file and extract exported function names.
	// Compare against the public methods on the corresponding Go service struct.

	bindingsDir := filepath.Join("frontend", "bindings", "hamster-wheel")

	tests := []struct {
		bindingFile string
		serviceType reflect.Type
	}{
		{"jobservice.js", reflect.TypeOf(&JobService{})},
		{"filterservice.js", reflect.TypeOf(&FilterService{})},
		{"pollingservice.js", reflect.TypeOf(&PollingService{})},
		{"settingsservice.js", reflect.TypeOf(&SettingsService{})},
	}

	// Regex to match exported JS functions: "export function FuncName("
	exportRe := regexp.MustCompile(`export function (\w+)\(`)

	for _, tt := range tests {
		t.Run(tt.bindingFile, func(t *testing.T) {
			// Extract exported function names from the JS file.
			jsPath := filepath.Join(bindingsDir, tt.bindingFile)
			jsFuncs, err := extractExportedFunctions(jsPath, exportRe)
			if err != nil {
				t.Fatalf("reading binding file %s: %v", tt.bindingFile, err)
			}

			// Extract public method names from the Go service type.
			// Exclude lifecycle methods (ServiceStartup, ServiceShutdown) — Wails
			// calls them internally, they're not exposed as frontend bindings.
			goMethods := exportedMethods(tt.serviceType, map[string]bool{
				"ServiceStartup":  true,
				"ServiceShutdown": true,
			})

			// Compare: every Go method should have a JS binding.
			sort.Strings(jsFuncs)
			sort.Strings(goMethods)

			for _, m := range goMethods {
				if !contains(jsFuncs, m) {
					t.Errorf("Go method %s.%s has no JS binding in %s",
						tt.serviceType.Elem().Name(), m, tt.bindingFile)
				}
			}

			// And vice versa: every JS binding should have a Go method.
			for _, f := range jsFuncs {
				if !contains(goMethods, f) {
					t.Errorf("JS binding %q in %s has no matching Go method on %s",
						f, tt.bindingFile, tt.serviceType.Elem().Name())
				}
			}
		})
	}
}

func TestSettingsBindingUsesIDCalls(t *testing.T) {
	path := filepath.Join("frontend", "bindings", "hamster-wheel", "settingsservice.js")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading settings binding file: %v", err)
	}

	content := string(raw)
	if strings.Contains(content, "$Call.ByName(") {
		t.Fatalf("settings binding contains $Call.ByName calls: regenerate bindings to use stable IDs")
	}
	if !strings.Contains(content, "$Call.ByID(") {
		t.Fatalf("settings binding does not contain any $Call.ByID calls")
	}
}

// extractExportedFunctions reads a JS file and returns the names of exported functions.
func extractExportedFunctions(path string, re *regexp.Regexp) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var funcs []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		matches := re.FindStringSubmatch(scanner.Text())
		if len(matches) == 2 {
			funcs = append(funcs, matches[1])
		}
	}
	return funcs, scanner.Err()
}

// exportedMethods returns the names of public methods on a type, excluding specified names.
func exportedMethods(t reflect.Type, exclude map[string]bool) []string {
	var methods []string
	for i := 0; i < t.NumMethod(); i++ {
		name := t.Method(i).Name
		if exclude[name] {
			continue
		}
		// Only include exported methods (uppercase first letter).
		if len(name) > 0 && strings.ToUpper(name[:1]) == name[:1] {
			methods = append(methods, name)
		}
	}
	return methods
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// --- 4. Service Failure Isolation ---

func TestDBErrorDoesNotAffectPollingService(t *testing.T) {
	_, _, _, pollSvc, _, database, sched, _ := testServices(t)

	// Start the scheduler so PollingService has something to interact with.
	sched.Start()
	t.Cleanup(func() { sched.Stop() })
	time.Sleep(100 * time.Millisecond)

	// Close the database to simulate a DB failure.
	database.Close()

	// PollingService should still work for non-DB operations.
	// Pause/resume doesn't touch the DB — it only toggles a scheduler flag.
	pollSvc.SetPollingPaused(true)
	status := pollSvc.GetPollingStatus()
	if !status.Paused {
		t.Error("expected paused=true even after DB is closed")
	}

	pollSvc.SetPollingPaused(false)
	status = pollSvc.GetPollingStatus()
	if status.Paused {
		t.Error("expected paused=false after resuming")
	}

	// NextPollAt should still be available (it's a scheduler-internal timestamp).
	if status.NextPollAt == "" {
		t.Error("expected non-empty NextPollAt even after DB closure")
	}
}

func TestDBErrorInJobServiceDoesNotCrash(t *testing.T) {
	_, jobSvc, _, _, _, database, _, _ := testServices(t)

	// Close the database to simulate a failure.
	database.Close()

	// JobService operations should return errors, not panic.
	_, err := jobSvc.GetJobs(10)
	if err == nil {
		t.Error("expected error from GetJobs after DB close, got nil")
	}

	_, err = jobSvc.GetJob("nonexistent")
	if err == nil {
		t.Error("expected error from GetJob after DB close, got nil")
	}

	_, err = jobSvc.GetJobCount()
	if err == nil {
		t.Error("expected error from GetJobCount after DB close, got nil")
	}

	err = jobSvc.DeleteJob("nonexistent")
	if err == nil {
		t.Error("expected error from DeleteJob after DB close, got nil")
	}
}

func TestDBErrorInFilterServiceDoesNotCrash(t *testing.T) {
	_, _, filterSvc, _, _, database, _, _ := testServices(t)

	// Close the database to simulate a failure.
	database.Close()

	// FilterService operations should return errors, not panic.
	_, err := filterSvc.GetFilters()
	if err == nil {
		t.Error("expected error from GetFilters after DB close, got nil")
	}

	_, err = filterSvc.GetFilter("nonexistent")
	if err == nil {
		t.Error("expected error from GetFilter after DB close, got nil")
	}

	_, err = filterSvc.CreateFilter("test", "go", "london", "reed_uk")
	if err == nil {
		t.Error("expected error from CreateFilter after DB close, got nil")
	}

	err = filterSvc.UpdateFilter("nonexistent", "test", "go", "london", "reed_uk", true)
	if err == nil {
		t.Error("expected error from UpdateFilter after DB close, got nil")
	}

	err = filterSvc.DeleteFilter("nonexistent")
	if err == nil {
		t.Error("expected error from DeleteFilter after DB close, got nil")
	}
}

func TestSettingsServiceAPIKeysWorkWithClosedDB(t *testing.T) {
	_, _, _, _, settingsSvc, database, _, _ := testServices(t)

	// Close the database to simulate a failure.
	database.Close()

	// API key operations use the keychain, not the DB — they should still work.
	err := settingsSvc.SetReedAPIKey("key-after-db-close")
	if err != nil {
		t.Errorf("expected no error from SetReedAPIKey after DB close, got: %v", err)
	}

	has, err := settingsSvc.HasReedAPIKey()
	if err != nil {
		t.Errorf("expected no error from HasReedAPIKey after DB close, got: %v", err)
	}
	if !has {
		t.Error("expected HasReedAPIKey=true after setting key with closed DB")
	}
	if err := settingsSvc.ClearReedAPIKey(); err != nil {
		t.Errorf("expected no error from ClearReedAPIKey after DB close, got: %v", err)
	}
	has, err = settingsSvc.HasReedAPIKey()
	if err != nil {
		t.Errorf("expected no error from HasReedAPIKey after clear, got: %v", err)
	}
	if has {
		t.Error("expected HasReedAPIKey=false after clear")
	}

	// Theme operations use the DB — they should error with closed DB.
	if _, err := settingsSvc.GetTheme(); err == nil {
		t.Error("expected error from GetTheme after DB close, got nil")
	}
	if err := settingsSvc.SetTheme("dark"); err == nil {
		t.Error("expected error from SetTheme after DB close, got nil")
	}
}

func TestServiceFailureIsolation(t *testing.T) {
	_, jobSvc, filterSvc, pollSvc, settingsSvc, database, sched, _ := testServices(t)

	// Start the scheduler.
	sched.Start()
	t.Cleanup(func() { sched.Stop() })
	time.Sleep(100 * time.Millisecond)

	// Close the database — this breaks JobService, FilterService, and SettingsService.
	database.Close()

	// Verify PollingService still works independently.
	pollSvc.SetPollingPaused(true)
	if !pollSvc.GetPollingStatus().Paused {
		t.Error("PollingService should still function after DB closure")
	}

	// Verify other services return errors (not panics).
	if _, err := jobSvc.GetJobCount(); err == nil {
		t.Error("JobService should error after DB closure")
	}
	if _, err := filterSvc.GetFilters(); err == nil {
		t.Error("FilterService should error after DB closure")
	}
	// SettingsService API key operations use keychain, not DB — they still work.
	if _, err := settingsSvc.HasReedAPIKey(); err != nil {
		t.Errorf("SettingsService API key status should work after DB closure, got: %v", err)
	}
	// SettingsService theme operations use DB — they should error.
	if _, err := settingsSvc.GetTheme(); err == nil {
		t.Error("SettingsService GetTheme should error after DB closure")
	}
}
