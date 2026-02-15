package localruntime

import (
	"context"
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
	client, setStatus := newHealthClient()
	setStatus(http.StatusOK)

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
	client, setStatus := newHealthClient()
	setStatus(http.StatusServiceUnavailable)

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
		setStatus(http.StatusOK)
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
	setStatus(http.StatusServiceUnavailable)

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
	client, setStatus := newHealthClient()
	setStatus(http.StatusServiceUnavailable)

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
	client, setStatus := newHealthClient()
	setStatus(http.StatusServiceUnavailable)

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
	client, setStatus := newHealthClient()
	setStatus(http.StatusServiceUnavailable)

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
}

func newHealthClient() (*http.Client, func(status int)) {
	transport := &fakeHealthTransport{}
	var statusCode atomic.Int64
	statusCode.Store(int64(http.StatusServiceUnavailable))
	transport.statusCode = &statusCode

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Second,
	}

	return client, func(status int) {
		statusCode.Store(int64(status))
	}
}

type fakeHealthTransport struct {
	statusCode *atomic.Int64
}

func (t *fakeHealthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	code := int(t.statusCode.Load())
	return &http.Response{
		StatusCode: code,
		Status:     http.StatusText(code),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		Request:    req,
	}, nil
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
