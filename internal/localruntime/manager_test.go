package localruntime

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestStatusReturnsNotInstalledWhenBinaryMissing(t *testing.T) {
	manager := NewOllamaManager(Config{
		Runner: &fakeRunner{
			lookPathErr: exec.ErrNotFound,
		},
	})

	snapshot, err := manager.Status(context.Background())
	if err != nil {
		t.Fatalf("checking status: %v", err)
	}
	if snapshot.Status != StatusNotInstalled {
		t.Fatalf("expected status %q, got %q", StatusNotInstalled, snapshot.Status)
	}
	if snapshot.StartedByApp {
		t.Fatal("expected StartedByApp=false")
	}
}

func TestStatusReadyWhenEndpointHealthy(t *testing.T) {
	client, runtime := newRuntimeClient()
	runtime.setHealthStatus(http.StatusOK)

	manager := NewOllamaManager(Config{
		Endpoint: "http://localruntime.test",
		Runner: &fakeRunner{
			lookPathResult: "/usr/local/bin/ollama",
		},
		HTTPClient: client,
	})

	snapshot, err := manager.Status(context.Background())
	if err != nil {
		t.Fatalf("checking status: %v", err)
	}
	if snapshot.Status != StatusReady {
		t.Fatalf("expected status %q, got %q", StatusReady, snapshot.Status)
	}
	if snapshot.StartedByApp {
		t.Fatal("expected StartedByApp=false for externally available runtime")
	}
}

func TestStartReturnsNotInstalledWhenBinaryMissing(t *testing.T) {
	manager := NewOllamaManager(Config{
		Runner: &fakeRunner{
			lookPathErr: exec.ErrNotFound,
		},
	})

	snapshot, err := manager.Start(context.Background())
	if err == nil {
		t.Fatal("expected start error for missing runtime")
	}
	if !errors.Is(err, ErrRuntimeNotInstalled) {
		t.Fatalf("expected ErrRuntimeNotInstalled, got %v", err)
	}
	if snapshot.Status != StatusNotInstalled {
		t.Fatalf("expected status %q, got %q", StatusNotInstalled, snapshot.Status)
	}
}

func TestStartLaunchesManagedProcessAndBecomesReady(t *testing.T) {
	client, runtime := newRuntimeClient()
	runtime.setHealthStatus(http.StatusServiceUnavailable)

	process := newFakeProcess(4242)
	process.exitOnSignal = true

	runner := &fakeRunner{
		lookPathResult: "/usr/local/bin/ollama",
		process:        process,
	}

	manager := NewOllamaManager(Config{
		Endpoint:            "http://localruntime.test",
		Runner:              runner,
		HTTPClient:          client,
		StartupTimeout:      700 * time.Millisecond,
		HealthCheckInterval: 25 * time.Millisecond,
		StopTimeout:         80 * time.Millisecond,
	})

	go func() {
		time.Sleep(60 * time.Millisecond)
		runtime.setHealthStatus(http.StatusOK)
	}()

	snapshot, err := manager.Start(context.Background())
	if err != nil {
		t.Fatalf("starting manager: %v", err)
	}
	if snapshot.Status != StatusReady {
		t.Fatalf("expected status %q, got %q", StatusReady, snapshot.Status)
	}
	if !snapshot.StartedByApp {
		t.Fatal("expected StartedByApp=true")
	}
	if snapshot.PID != 4242 {
		t.Fatalf("expected PID 4242, got %d", snapshot.PID)
	}

	// Simulate local process-only availability so stop returns stopped.
	runtime.setHealthStatus(http.StatusServiceUnavailable)

	stopped, err := manager.Stop(context.Background())
	if err != nil {
		t.Fatalf("stopping manager: %v", err)
	}
	if stopped.Status != StatusStopped {
		t.Fatalf("expected status %q after stop, got %q", StatusStopped, stopped.Status)
	}
	if process.signalCount() == 0 {
		t.Fatal("expected process to receive at least one signal")
	}
}

func TestStartTimeoutReturnsStartingStatus(t *testing.T) {
	client, runtime := newRuntimeClient()
	runtime.setHealthStatus(http.StatusServiceUnavailable)

	process := newFakeProcess(99)
	process.exitOnSignal = true

	manager := NewOllamaManager(Config{
		Endpoint: "http://localruntime.test",
		Runner: &fakeRunner{
			lookPathResult: "/usr/local/bin/ollama",
			process:        process,
		},
		HTTPClient:          client,
		StartupTimeout:      90 * time.Millisecond,
		HealthCheckInterval: 15 * time.Millisecond,
		StopTimeout:         50 * time.Millisecond,
	})

	snapshot, err := manager.Start(context.Background())
	if err != nil {
		t.Fatalf("starting manager: %v", err)
	}
	if snapshot.Status != StatusStarting {
		t.Fatalf("expected status %q after timeout, got %q", StatusStarting, snapshot.Status)
	}
	if !snapshot.StartedByApp {
		t.Fatal("expected StartedByApp=true while runtime warms up")
	}

	if _, err := manager.Stop(context.Background()); err != nil {
		t.Fatalf("stopping manager after timeout: %v", err)
	}
}

func TestStatusReportsManagedProcessExitAsError(t *testing.T) {
	client, runtime := newRuntimeClient()
	runtime.setHealthStatus(http.StatusServiceUnavailable)

	process := newFakeProcess(3123)
	process.exitOnSignal = true

	manager := NewOllamaManager(Config{
		Endpoint: "http://localruntime.test",
		Runner: &fakeRunner{
			lookPathResult: "/usr/local/bin/ollama",
			process:        process,
		},
		HTTPClient:          client,
		StartupTimeout:      70 * time.Millisecond,
		HealthCheckInterval: 10 * time.Millisecond,
	})

	if _, err := manager.Start(context.Background()); err != nil {
		t.Fatalf("starting manager: %v", err)
	}

	process.complete(errors.New("boom"))

	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		snapshot, err := manager.Status(context.Background())
		if err != nil {
			t.Fatalf("checking status: %v", err)
		}
		if snapshot.Status == StatusError {
			if snapshot.Message == "" {
				t.Fatal("expected non-empty error message")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected status to transition to error after managed process exit")
}

func TestStopEscalatesToKillWhenInterruptDoesNotExit(t *testing.T) {
	client, runtime := newRuntimeClient()
	runtime.setHealthStatus(http.StatusServiceUnavailable)

	process := newFakeProcess(8080)
	process.exitOnSignal = false
	process.exitOnKill = true

	manager := NewOllamaManager(Config{
		Endpoint: "http://localruntime.test",
		Runner: &fakeRunner{
			lookPathResult: "/usr/local/bin/ollama",
			process:        process,
		},
		HTTPClient:          client,
		StartupTimeout:      70 * time.Millisecond,
		HealthCheckInterval: 10 * time.Millisecond,
		StopTimeout:         20 * time.Millisecond,
	})

	if _, err := manager.Start(context.Background()); err != nil {
		t.Fatalf("starting manager: %v", err)
	}
	if _, err := manager.Stop(context.Background()); err != nil {
		t.Fatalf("stopping manager: %v", err)
	}
	if !process.wasKilled() {
		t.Fatal("expected manager stop to escalate to kill")
	}
}

func TestListModelsReturnsCatalog(t *testing.T) {
	client, runtime := newRuntimeClient()
	runtime.addModel("llama3.1:8b", "sha256:abc", 123)

	manager := NewOllamaManager(Config{
		Endpoint: "http://localruntime.test",
		Runner: &fakeRunner{
			lookPathResult: "/usr/local/bin/ollama",
		},
		HTTPClient: client,
	})

	catalog, err := manager.ListModels(context.Background())
	if err != nil {
		t.Fatalf("listing models: %v", err)
	}
	if catalog.Engine != EngineOllama {
		t.Fatalf("expected engine %q, got %q", EngineOllama, catalog.Engine)
	}
	if len(catalog.Recommended) == 0 {
		t.Fatal("expected recommended models to be returned")
	}
	if len(catalog.Installed) != 1 {
		t.Fatalf("expected 1 installed model, got %d", len(catalog.Installed))
	}
	if catalog.Installed[0].Name != "llama3.1:8b" {
		t.Fatalf("expected installed model name llama3.1:8b, got %q", catalog.Installed[0].Name)
	}
}

func TestListModelsReturnsNotInstalledError(t *testing.T) {
	manager := NewOllamaManager(Config{
		Runner: &fakeRunner{lookPathErr: exec.ErrNotFound},
	})

	_, err := manager.ListModels(context.Background())
	if err == nil {
		t.Fatal("expected list models to fail when runtime is missing")
	}
	if !errors.Is(err, ErrRuntimeNotInstalled) {
		t.Fatalf("expected ErrRuntimeNotInstalled, got %v", err)
	}
}

func TestPullModelSuccess(t *testing.T) {
	client, runtime := newRuntimeClient()

	manager := NewOllamaManager(Config{
		Endpoint: "http://localruntime.test",
		Runner: &fakeRunner{
			lookPathResult: "/usr/local/bin/ollama",
		},
		HTTPClient: client,
	})

	result, err := manager.PullModel(context.Background(), "qwen2.5:7b")
	if err != nil {
		t.Fatalf("pulling model: %v", err)
	}
	if result.Model != "qwen2.5:7b" {
		t.Fatalf("expected pulled model qwen2.5:7b, got %q", result.Model)
	}
	if !result.Ready {
		t.Fatal("expected pulled model to be marked ready")
	}
	if result.Status == "" {
		t.Fatal("expected non-empty pull status")
	}

	catalog, err := manager.ListModels(context.Background())
	if err != nil {
		t.Fatalf("listing models after pull: %v", err)
	}
	found := false
	for _, model := range catalog.Installed {
		if model.Name == "qwen2.5:7b" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected pulled model to appear in installed model list")
	}
	if runtime.pullCount() == 0 {
		t.Fatal("expected pull endpoint to be invoked")
	}
}

func TestPullModelRejectsEmptyName(t *testing.T) {
	client, _ := newRuntimeClient()

	manager := NewOllamaManager(Config{
		Endpoint: "http://localruntime.test",
		Runner: &fakeRunner{
			lookPathResult: "/usr/local/bin/ollama",
		},
		HTTPClient: client,
	})

	_, err := manager.PullModel(context.Background(), "   ")
	if err == nil {
		t.Fatal("expected empty model name to be rejected")
	}
}

func TestPullModelPropagatesRuntimeError(t *testing.T) {
	client, runtime := newRuntimeClient()
	runtime.setPullError("manifest not found")

	manager := NewOllamaManager(Config{
		Endpoint: "http://localruntime.test",
		Runner: &fakeRunner{
			lookPathResult: "/usr/local/bin/ollama",
		},
		HTTPClient: client,
	})

	_, err := manager.PullModel(context.Background(), "unknown:model")
	if err == nil {
		t.Fatal("expected pull error")
	}
	if !strings.Contains(err.Error(), "manifest not found") {
		t.Fatalf("expected runtime error message to be included, got %v", err)
	}
}

func TestNoopManagerContract(t *testing.T) {
	manager := NewNoopManager()

	status, err := manager.Status(context.Background())
	if err != nil {
		t.Fatalf("noop status: %v", err)
	}
	if status.Status != StatusNotInstalled {
		t.Fatalf("expected noop status %q, got %q", StatusNotInstalled, status.Status)
	}

	startSnapshot, err := manager.Start(context.Background())
	if err == nil {
		t.Fatal("expected noop start to fail")
	}
	if !errors.Is(err, ErrRuntimeNotInstalled) {
		t.Fatalf("expected ErrRuntimeNotInstalled from noop start, got %v", err)
	}
	if startSnapshot.Status != StatusNotInstalled {
		t.Fatalf("expected noop start status %q, got %q", StatusNotInstalled, startSnapshot.Status)
	}

	stopSnapshot, err := manager.Stop(context.Background())
	if err != nil {
		t.Fatalf("noop stop: %v", err)
	}
	if stopSnapshot.Status != StatusStopped {
		t.Fatalf("expected noop stop status %q, got %q", StatusStopped, stopSnapshot.Status)
	}

	catalog, err := manager.ListModels(context.Background())
	if err == nil {
		t.Fatal("expected noop list models to fail")
	}
	if !errors.Is(err, ErrRuntimeUnsupported) {
		t.Fatalf("expected ErrRuntimeUnsupported, got %v", err)
	}
	if len(catalog.Recommended) == 0 {
		t.Fatal("expected recommended models to be provided by noop manager")
	}

	if _, err := manager.PullModel(context.Background(), "llama3.1:8b"); err == nil {
		t.Fatal("expected noop pull to fail")
	}
}

func newRuntimeClient() (*http.Client, *fakeRuntimeTransport) {
	transport := &fakeRuntimeTransport{}
	transport.healthStatus.Store(int64(http.StatusServiceUnavailable))
	transport.tagsStatus.Store(int64(http.StatusOK))
	transport.pullStatus.Store(int64(http.StatusOK))
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Second,
	}
	return client, transport
}

type fakeRuntimeTransport struct {
	healthStatus atomic.Int64
	tagsStatus   atomic.Int64
	pullStatus   atomic.Int64

	mu         sync.Mutex
	models     map[string]ModelInfo
	pullError  string
	pullCalled int
}

func (t *fakeRuntimeTransport) setHealthStatus(status int) {
	t.healthStatus.Store(int64(status))
}

func (t *fakeRuntimeTransport) setPullError(message string) {
	t.mu.Lock()
	t.pullError = message
	t.mu.Unlock()
}

func (t *fakeRuntimeTransport) addModel(name, digest string, size int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.models == nil {
		t.models = make(map[string]ModelInfo)
	}
	t.models[name] = ModelInfo{
		Name:      name,
		Digest:    digest,
		SizeBytes: size,
	}
}

func (t *fakeRuntimeTransport) pullCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.pullCalled
}

func (t *fakeRuntimeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.URL.Path {
	case "/api/version":
		return buildFakeResponse(req, int(t.healthStatus.Load()), `{"version":"0.5.7"}`), nil
	case "/api/tags":
		t.mu.Lock()
		models := make([]map[string]any, 0, len(t.models))
		for _, model := range t.models {
			models = append(models, map[string]any{
				"name":        model.Name,
				"digest":      model.Digest,
				"size":        model.SizeBytes,
				"modified_at": time.Now().UTC().Format(time.RFC3339Nano),
			})
		}
		t.mu.Unlock()
		payload, _ := json.Marshal(map[string]any{"models": models})
		return buildFakeResponse(req, int(t.tagsStatus.Load()), string(payload)), nil
	case "/api/pull":
		t.mu.Lock()
		t.pullCalled++
		pullError := t.pullError
		t.mu.Unlock()

		if int(t.pullStatus.Load()) != http.StatusOK {
			return buildFakeResponse(req, int(t.pullStatus.Load()), `{"error":"pull failed"}`), nil
		}
		if strings.TrimSpace(pullError) != "" {
			payload, _ := json.Marshal(map[string]any{"error": pullError})
			return buildFakeResponse(req, http.StatusOK, string(payload)), nil
		}

		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			return buildFakeResponse(req, http.StatusBadRequest, `{"error":"invalid request"}`), nil
		}
		model := strings.TrimSpace(body.Name)
		if model != "" {
			t.addModel(model, "sha256:pulled", 1024)
		}
		return buildFakeResponse(req, http.StatusOK, `{"status":"success"}`), nil
	default:
		return buildFakeResponse(req, http.StatusNotFound, `{"error":"not found"}`), nil
	}
}

func buildFakeResponse(req *http.Request, statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

type fakeRunner struct {
	lookPathResult string
	lookPathErr    error
	startErr       error
	process        *fakeProcess
}

func (f *fakeRunner) LookPath(file string) (string, error) {
	if f.lookPathErr != nil {
		return "", f.lookPathErr
	}
	if f.lookPathResult == "" {
		return "/usr/local/bin/" + file, nil
	}
	return f.lookPathResult, nil
}

func (f *fakeRunner) Start(_ string, _ ...string) (Process, error) {
	if f.startErr != nil {
		return nil, f.startErr
	}
	if f.process == nil {
		f.process = newFakeProcess(1001)
	}
	return f.process, nil
}

type fakeProcess struct {
	pid int

	exitOnSignal bool
	exitOnKill   bool

	waitCh    chan error
	closeOnce sync.Once

	mu         sync.Mutex
	signals    []os.Signal
	killCalled bool
}

func newFakeProcess(pid int) *fakeProcess {
	return &fakeProcess{
		pid:        pid,
		exitOnKill: true,
		waitCh:     make(chan error, 1),
	}
}

func (p *fakeProcess) PID() int {
	return p.pid
}

func (p *fakeProcess) Signal(signal os.Signal) error {
	p.mu.Lock()
	p.signals = append(p.signals, signal)
	p.mu.Unlock()
	if p.exitOnSignal {
		p.complete(nil)
	}
	return nil
}

func (p *fakeProcess) Kill() error {
	p.mu.Lock()
	p.killCalled = true
	p.mu.Unlock()
	if p.exitOnKill {
		p.complete(nil)
	}
	return nil
}

func (p *fakeProcess) Wait() error {
	err, ok := <-p.waitCh
	if !ok {
		return nil
	}
	return err
}

func (p *fakeProcess) complete(err error) {
	p.closeOnce.Do(func() {
		p.waitCh <- err
		close(p.waitCh)
	})
}

func (p *fakeProcess) signalCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.signals)
}

func (p *fakeProcess) wasKilled() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.killCalled
}

var _ Manager = (*RuntimeManager)(nil)
