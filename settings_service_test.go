package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hamster-wheel/internal/adapter/reed"
	"hamster-wheel/internal/db"
	"hamster-wheel/internal/keychain"
	"hamster-wheel/internal/localruntime"
)

type failingKeychainStore struct {
	getErr    error
	setErr    error
	deleteErr error
}

type failingLocalRuntimeManager struct {
	statusErr error
	startErr  error
	stopErr   error
}

func (m failingLocalRuntimeManager) Status(context.Context) (localruntime.Snapshot, error) {
	if m.statusErr != nil {
		return localruntime.Snapshot{}, m.statusErr
	}
	return localruntime.Snapshot{Status: localruntime.StatusReady}, nil
}

func (m failingLocalRuntimeManager) Start(context.Context) (localruntime.Snapshot, error) {
	if m.startErr != nil {
		return localruntime.Snapshot{}, m.startErr
	}
	return localruntime.Snapshot{Status: localruntime.StatusReady}, nil
}

func (m failingLocalRuntimeManager) Stop(context.Context) (localruntime.Snapshot, error) {
	if m.stopErr != nil {
		return localruntime.Snapshot{}, m.stopErr
	}
	return localruntime.Snapshot{Status: localruntime.StatusStopped}, nil
}

func (s failingKeychainStore) Get(string) (string, error) {
	if s.getErr != nil {
		return "", s.getErr
	}
	return "", nil
}

func (s failingKeychainStore) Set(string, string) error {
	return s.setErr
}

func (s failingKeychainStore) Delete(string) error {
	return s.deleteErr
}

func openSettingsTestDB(t *testing.T) *db.DB {
	t.Helper()
	database, err := db.OpenAt(filepath.Join(t.TempDir(), "settings-test.db"))
	if err != nil {
		t.Fatalf("opening test DB: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func TestSettingsServiceAPIKeyLifecycle(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	has, err := svc.HasReedAPIKey()
	if err != nil {
		t.Fatalf("checking initial key presence: %v", err)
	}
	if has {
		t.Fatal("expected HasReedAPIKey=false initially")
	}
	if reedAdapter.HasAPIKey() {
		t.Fatal("expected adapter to start without API key")
	}

	if err := svc.SetReedAPIKey("  secret-key-123  "); err != nil {
		t.Fatalf("setting API key: %v", err)
	}

	has, err = svc.HasReedAPIKey()
	if err != nil {
		t.Fatalf("checking key presence after set: %v", err)
	}
	if !has {
		t.Fatal("expected HasReedAPIKey=true after set")
	}
	if !reedAdapter.HasAPIKey() {
		t.Fatal("expected adapter to have key after set")
	}

	stored, err := kc.Get(settingReedAPIKey)
	if err != nil {
		t.Fatalf("reading keychain key: %v", err)
	}
	if stored != "secret-key-123" {
		t.Fatalf("expected trimmed key to be stored, got %q", stored)
	}

	if err := svc.ClearReedAPIKey(); err != nil {
		t.Fatalf("clearing API key: %v", err)
	}

	has, err = svc.HasReedAPIKey()
	if err != nil {
		t.Fatalf("checking key presence after clear: %v", err)
	}
	if has {
		t.Fatal("expected HasReedAPIKey=false after clear")
	}
	if reedAdapter.HasAPIKey() {
		t.Fatal("expected adapter key to be cleared")
	}
}

type stubLocalRuntimeManager struct{}

func (stubLocalRuntimeManager) Status(context.Context) (localruntime.Snapshot, error) {
	return localruntime.Snapshot{Status: localruntime.StatusReady}, nil
}

func (stubLocalRuntimeManager) Start(context.Context) (localruntime.Snapshot, error) {
	return localruntime.Snapshot{Status: localruntime.StatusReady}, nil
}

func (stubLocalRuntimeManager) Stop(context.Context) (localruntime.Snapshot, error) {
	return localruntime.Snapshot{Status: localruntime.StatusStopped}, nil
}

func TestSettingsServiceLocalRuntimeDependencyDefaultsAndInjection(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")

	defaultSvc := NewSettingsService(database, kc, reedAdapter)
	if defaultSvc.localRuntime == nil {
		t.Fatal("expected default local runtime manager to be configured")
	}
	defaultSnapshot, err := defaultSvc.localRuntime.Status(context.Background())
	if err != nil {
		t.Fatalf("checking default local runtime status: %v", err)
	}
	if defaultSnapshot.Status != localruntime.StatusNotInstalled {
		t.Fatalf("expected default local runtime status %q, got %q", localruntime.StatusNotInstalled, defaultSnapshot.Status)
	}

	customManager := stubLocalRuntimeManager{}
	customSvc := NewSettingsService(database, kc, reedAdapter, customManager)
	if customSvc.localRuntime == nil {
		t.Fatal("expected injected local runtime manager to be set")
	}
	customSnapshot, err := customSvc.localRuntime.Status(context.Background())
	if err != nil {
		t.Fatalf("checking custom local runtime status: %v", err)
	}
	if customSnapshot.Status != localruntime.StatusReady {
		t.Fatalf("expected injected runtime status %q, got %q", localruntime.StatusReady, customSnapshot.Status)
	}
}

func TestSettingsServiceLocalRuntimeLifecycle(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	local := stubLocalRuntimeManager{}
	svc := NewSettingsService(database, kc, reedAdapter, local)

	status, err := svc.GetLocalRuntimeStatus()
	if err != nil {
		t.Fatalf("getting local runtime status: %v", err)
	}
	if status.Status != localruntime.StatusReady {
		t.Fatalf("expected local runtime status %q, got %q", localruntime.StatusReady, status.Status)
	}

	started, err := svc.StartLocalRuntime()
	if err != nil {
		t.Fatalf("starting local runtime: %v", err)
	}
	if started.Status != localruntime.StatusReady {
		t.Fatalf("expected start status %q, got %q", localruntime.StatusReady, started.Status)
	}

	stopped, err := svc.StopLocalRuntime()
	if err != nil {
		t.Fatalf("stopping local runtime: %v", err)
	}
	if stopped.Status != localruntime.StatusStopped {
		t.Fatalf("expected stop status %q, got %q", localruntime.StatusStopped, stopped.Status)
	}
}

func TestSettingsServiceLocalRuntimeErrors(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter, failingLocalRuntimeManager{
		statusErr: errors.New("status failed"),
		startErr:  errors.New("start failed"),
		stopErr:   errors.New("stop failed"),
	})

	if _, err := svc.GetLocalRuntimeStatus(); err == nil || !strings.Contains(err.Error(), "getting local runtime status") {
		t.Fatalf("expected wrapped status error, got %v", err)
	}
	if _, err := svc.StartLocalRuntime(); err == nil || !strings.Contains(err.Error(), "starting local runtime") {
		t.Fatalf("expected wrapped start error, got %v", err)
	}
	if _, err := svc.StopLocalRuntime(); err == nil || !strings.Contains(err.Error(), "stopping local runtime") {
		t.Fatalf("expected wrapped stop error, got %v", err)
	}
}

func TestSetReedAPIKeyRejectsEmpty(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	if err := svc.SetReedAPIKey("   "); err == nil {
		t.Fatal("expected error for empty API key")
	}
}

func TestSettingsServiceAPIKeyErrorsPropagate(t *testing.T) {
	database := openSettingsTestDB(t)
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, failingKeychainStore{
		getErr:    errors.New("get failed"),
		setErr:    errors.New("set failed"),
		deleteErr: errors.New("delete failed"),
	}, reedAdapter)

	if _, err := svc.HasReedAPIKey(); err == nil {
		t.Fatal("expected error from HasReedAPIKey")
	}
	if err := svc.SetReedAPIKey("key-1"); err == nil {
		t.Fatal("expected error from SetReedAPIKey")
	}
	if err := svc.ClearReedAPIKey(); err == nil {
		t.Fatal("expected error from ClearReedAPIKey")
	}
	if reedAdapter.HasAPIKey() {
		t.Fatal("expected adapter key to remain unchanged on failures")
	}
}

func TestKeyboardShortcutsLifecycleAndValidation(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	current, err := svc.GetKeyboardShortcuts()
	if err != nil {
		t.Fatalf("reading default keyboard shortcuts setting: %v", err)
	}
	if current != "" {
		t.Fatalf("expected empty default keyboard shortcuts setting, got %q", current)
	}

	if err := svc.SetKeyboardShortcuts("true"); err != nil {
		t.Fatalf("setting keyboard shortcuts=true: %v", err)
	}
	current, err = svc.GetKeyboardShortcuts()
	if err != nil {
		t.Fatalf("reading keyboard shortcuts after true: %v", err)
	}
	if current != "true" {
		t.Fatalf("expected keyboard shortcuts value true, got %q", current)
	}

	if err := svc.SetKeyboardShortcuts("false"); err != nil {
		t.Fatalf("setting keyboard shortcuts=false: %v", err)
	}
	current, err = svc.GetKeyboardShortcuts()
	if err != nil {
		t.Fatalf("reading keyboard shortcuts after false: %v", err)
	}
	if current != "false" {
		t.Fatalf("expected keyboard shortcuts value false, got %q", current)
	}

	err = svc.SetKeyboardShortcuts("yes")
	if err == nil {
		t.Fatal("expected validation error for invalid keyboard shortcuts value")
	}
	if !strings.Contains(err.Error(), "invalid keyboard shortcuts value") {
		t.Fatalf("expected validation context in error, got %v", err)
	}
}

func TestKeyboardShortcutsDatabaseErrors(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	if err := database.Close(); err != nil {
		t.Fatalf("closing DB: %v", err)
	}

	_, err := svc.GetKeyboardShortcuts()
	if err == nil {
		t.Fatal("expected GetKeyboardShortcuts to fail on closed DB")
	}
	if !strings.Contains(err.Error(), "getting keyboard shortcuts setting") {
		t.Fatalf("expected get-setting context, got %v", err)
	}

	err = svc.SetKeyboardShortcuts("true")
	if err == nil {
		t.Fatal("expected SetKeyboardShortcuts to fail on closed DB")
	}
	if !strings.Contains(err.Error(), "setting keyboard shortcuts") {
		t.Fatalf("expected set-setting context, got %v", err)
	}
}

func TestOpenAIAPIKeyLifecycle(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	has, err := svc.HasOpenAIAPIKey()
	if err != nil {
		t.Fatalf("checking initial openai key presence: %v", err)
	}
	if has {
		t.Fatal("expected HasOpenAIAPIKey=false initially")
	}

	if err := svc.SetOpenAIAPIKey("  sk-test-openai  "); err != nil {
		t.Fatalf("setting openai API key: %v", err)
	}

	has, err = svc.HasOpenAIAPIKey()
	if err != nil {
		t.Fatalf("checking openai key presence after set: %v", err)
	}
	if !has {
		t.Fatal("expected HasOpenAIAPIKey=true after set")
	}

	stored, err := kc.Get(settingOpenAIAPIKey)
	if err != nil {
		t.Fatalf("reading openai keychain key: %v", err)
	}
	if stored != "sk-test-openai" {
		t.Fatalf("expected trimmed key to be stored, got %q", stored)
	}

	if err := svc.ClearOpenAIAPIKey(); err != nil {
		t.Fatalf("clearing openai API key: %v", err)
	}

	has, err = svc.HasOpenAIAPIKey()
	if err != nil {
		t.Fatalf("checking openai key presence after clear: %v", err)
	}
	if has {
		t.Fatal("expected HasOpenAIAPIKey=false after clear")
	}
}

func TestOpenAIAPIKeyValidationAndErrors(t *testing.T) {
	database := openSettingsTestDB(t)
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, failingKeychainStore{
		getErr:    errors.New("get failed"),
		setErr:    errors.New("set failed"),
		deleteErr: errors.New("delete failed"),
	}, reedAdapter)

	if _, err := svc.HasOpenAIAPIKey(); err == nil {
		t.Fatal("expected error from HasOpenAIAPIKey")
	}
	if err := svc.SetOpenAIAPIKey("   "); err == nil {
		t.Fatal("expected validation error for empty openai API key")
	}
	if err := svc.SetOpenAIAPIKey("abc"); err == nil {
		t.Fatal("expected keychain error from SetOpenAIAPIKey")
	}
	if err := svc.ClearOpenAIAPIKey(); err == nil {
		t.Fatal("expected keychain error from ClearOpenAIAPIKey")
	}
}

func TestLLMProviderDefaultsAndLifecycle(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	provider, err := svc.GetLLMProvider()
	if err != nil {
		t.Fatalf("getting default llm provider: %v", err)
	}
	if provider != defaultLLMProvider {
		t.Fatalf("expected default provider %q, got %q", defaultLLMProvider, provider)
	}

	if err := svc.SetLLMProvider("openai"); err != nil {
		t.Fatalf("setting llm provider openai: %v", err)
	}
	provider, err = svc.GetLLMProvider()
	if err != nil {
		t.Fatalf("getting llm provider after set openai: %v", err)
	}
	if provider != "openai" {
		t.Fatalf("expected provider openai, got %q", provider)
	}

	if err := svc.SetLLMProvider("heuristic_v1"); err != nil {
		t.Fatalf("setting llm provider heuristic_v1: %v", err)
	}
	provider, err = svc.GetLLMProvider()
	if err != nil {
		t.Fatalf("getting llm provider after set heuristic_v1: %v", err)
	}
	if provider != "heuristic_v1" {
		t.Fatalf("expected provider heuristic_v1, got %q", provider)
	}

	if err := svc.SetLLMProvider("anthropic"); err == nil {
		t.Fatal("expected validation error for unsupported provider")
	}
}

func TestLLMProviderDatabaseErrors(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	if err := database.Close(); err != nil {
		t.Fatalf("closing DB: %v", err)
	}

	if _, err := svc.GetLLMProvider(); err == nil {
		t.Fatal("expected GetLLMProvider to fail on closed DB")
	}
	if err := svc.SetLLMProvider("openai"); err == nil {
		t.Fatal("expected SetLLMProvider to fail on closed DB")
	}
}

func TestLLMModelDefaultsAndLifecycle(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	model, err := svc.GetLLMModel()
	if err != nil {
		t.Fatalf("getting default llm model: %v", err)
	}
	if model != defaultLLMModel {
		t.Fatalf("expected default model %q, got %q", defaultLLMModel, model)
	}

	if err := svc.SetLLMModel("gpt-4o"); err != nil {
		t.Fatalf("setting llm model: %v", err)
	}
	model, err = svc.GetLLMModel()
	if err != nil {
		t.Fatalf("getting llm model after set: %v", err)
	}
	if model != "gpt-4o" {
		t.Fatalf("expected model gpt-4o, got %q", model)
	}

	if err := svc.SetLLMModel("   "); err == nil {
		t.Fatal("expected validation error for empty llm model")
	}
}

func TestLLMModelDatabaseErrors(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	if err := database.Close(); err != nil {
		t.Fatalf("closing DB: %v", err)
	}

	if _, err := svc.GetLLMModel(); err == nil {
		t.Fatal("expected GetLLMModel to fail on closed DB")
	}
	if err := svc.SetLLMModel("gpt-4o-mini"); err == nil {
		t.Fatal("expected SetLLMModel to fail on closed DB")
	}
}

func TestLLMBaseURLLifecycleAndValidation(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	baseURL, err := svc.GetLLMBaseURL()
	if err != nil {
		t.Fatalf("getting default llm base url: %v", err)
	}
	if baseURL != "" {
		t.Fatalf("expected empty default base url, got %q", baseURL)
	}

	if err := svc.SetLLMBaseURL("https://api.openai.com/v1"); err != nil {
		t.Fatalf("setting llm base url: %v", err)
	}
	baseURL, err = svc.GetLLMBaseURL()
	if err != nil {
		t.Fatalf("getting llm base url after set: %v", err)
	}
	if baseURL != "https://api.openai.com/v1" {
		t.Fatalf("expected base url https://api.openai.com/v1, got %q", baseURL)
	}

	if err := svc.SetLLMBaseURL(""); err != nil {
		t.Fatalf("clearing llm base url via empty value: %v", err)
	}
	baseURL, err = svc.GetLLMBaseURL()
	if err != nil {
		t.Fatalf("getting llm base url after clear: %v", err)
	}
	if baseURL != "" {
		t.Fatalf("expected empty base url after clear, got %q", baseURL)
	}

	if err := svc.SetLLMBaseURL("not-a-url"); err == nil {
		t.Fatal("expected validation error for invalid base url")
	}
}

func TestLLMBaseURLDatabaseErrors(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	if err := database.Close(); err != nil {
		t.Fatalf("closing DB: %v", err)
	}

	if _, err := svc.GetLLMBaseURL(); err == nil {
		t.Fatal("expected GetLLMBaseURL to fail on closed DB")
	}
	if err := svc.SetLLMBaseURL("https://example.com/v1"); err == nil {
		t.Fatal("expected SetLLMBaseURL to fail on closed DB")
	}
}

func TestCVPathLifecycleAndValidation(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	cvPath, err := svc.GetCVPath()
	if err != nil {
		t.Fatalf("getting default cv path: %v", err)
	}
	if cvPath != "" {
		t.Fatalf("expected empty default cv path, got %q", cvPath)
	}

	file := filepath.Join(t.TempDir(), "cv.txt")
	if err := os.WriteFile(file, []byte("Go backend engineer with 8 years of experience."), 0o600); err != nil {
		t.Fatalf("writing temp cv file: %v", err)
	}

	if err := svc.SetCVPath("  " + file + "  "); err != nil {
		t.Fatalf("setting cv path: %v", err)
	}
	cvPath, err = svc.GetCVPath()
	if err != nil {
		t.Fatalf("getting cv path after set: %v", err)
	}
	if cvPath != file {
		t.Fatalf("expected cv path %q, got %q", file, cvPath)
	}

	if err := svc.SetCVPath(""); err != nil {
		t.Fatalf("clearing cv path via empty value: %v", err)
	}
	cvPath, err = svc.GetCVPath()
	if err != nil {
		t.Fatalf("getting cv path after clear: %v", err)
	}
	if cvPath != "" {
		t.Fatalf("expected empty cv path after clear, got %q", cvPath)
	}

	if err := svc.SetCVPath(filepath.Join(t.TempDir(), "missing.txt")); err == nil {
		t.Fatal("expected validation error for missing cv file")
	}
	if err := svc.SetCVPath(t.TempDir()); err == nil {
		t.Fatal("expected validation error for cv directory path")
	}

	pdfPath := filepath.Join(t.TempDir(), "cv.pdf")
	if err := os.WriteFile(pdfPath, buildMinimalPDF("Go backend engineer"), 0o600); err != nil {
		t.Fatalf("writing pdf cv file: %v", err)
	}
	if err := svc.SetCVPath(pdfPath); err != nil {
		t.Fatalf("expected pdf cv path to be accepted, got %v", err)
	}

	unsupportedPath := filepath.Join(t.TempDir(), "cv.docx")
	if err := os.WriteFile(unsupportedPath, append([]byte("PK\x03\x04"), []byte("fake-docx")...), 0o600); err != nil {
		t.Fatalf("writing unsupported docx-like file: %v", err)
	}
	if err := svc.SetCVPath(unsupportedPath); err == nil {
		t.Fatal("expected unsupported docx-like cv to be rejected at submission")
	}
}

func TestCVPathDatabaseErrors(t *testing.T) {
	database := openSettingsTestDB(t)
	kc := keychain.NewMemoryStore()
	reedAdapter := reed.New("")
	svc := NewSettingsService(database, kc, reedAdapter)

	if err := database.Close(); err != nil {
		t.Fatalf("closing DB: %v", err)
	}

	if _, err := svc.GetCVPath(); err == nil {
		t.Fatal("expected GetCVPath to fail on closed DB")
	}
	if err := svc.SetCVPath(""); err == nil {
		t.Fatal("expected SetCVPath to fail on closed DB")
	}
}

func buildMinimalPDF(text string) []byte {
	escaped := escapePDFText(text)
	stream := fmt.Sprintf("BT\n/F1 12 Tf\n72 720 Td\n(%s) Tj\nET\n", escaped)

	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(stream), stream),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}

	xrefOffset := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n", len(objects)+1)
	out.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefOffset)

	return out.Bytes()
}

func escapePDFText(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "(", `\(`)
	value = strings.ReplaceAll(value, ")", `\)`)
	return value
}
