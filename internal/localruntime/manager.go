package localruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	EngineOllama = "ollama"

	StatusNotInstalled = "not_installed"
	StatusStopped      = "stopped"
	StatusStarting     = "starting"
	StatusReady        = "ready"
	StatusError        = "error"
)

const (
	defaultBinary              = "ollama"
	DefaultOllamaEndpoint      = "http://127.0.0.1:11434"
	defaultHealthPath          = "/api/version"
	defaultStartupTimeout      = 8 * time.Second
	defaultStopTimeout         = 3 * time.Second
	defaultHealthCheckInterval = 250 * time.Millisecond
	defaultHTTPTimeout         = 1200 * time.Millisecond
	maxResponseBytes           = 1 << 20 // 1 MiB
)

var (
	ErrRuntimeNotInstalled = errors.New("local runtime is not installed")
	ErrRuntimeStartFailed  = errors.New("local runtime failed to start")
	ErrRuntimeUnsupported  = errors.New("local runtime feature is unsupported")
)

var recommendedModels = []string{
	"llama3.1:8b",
	"qwen2.5:7b",
	"mistral:7b",
}

// Snapshot is the runtime status returned to callers.
type Snapshot struct {
	Engine       string    `json:"engine"`
	Status       string    `json:"status"`
	Endpoint     string    `json:"endpoint"`
	Message      string    `json:"message,omitempty"`
	StartedByApp bool      `json:"startedByApp"`
	PID          int       `json:"pid,omitempty"`
	CheckedAt    time.Time `json:"checkedAt"`
}

// ModelInfo describes a model available in the local runtime.
type ModelInfo struct {
	Name       string    `json:"name"`
	Digest     string    `json:"digest,omitempty"`
	SizeBytes  int64     `json:"sizeBytes,omitempty"`
	ModifiedAt time.Time `json:"modifiedAt,omitempty"`
}

// ModelCatalog is the local runtime model inventory returned to callers.
type ModelCatalog struct {
	Engine      string      `json:"engine"`
	Recommended []string    `json:"recommended"`
	Installed   []ModelInfo `json:"installed"`
}

// PullResult captures the outcome of a local model pull operation.
type PullResult struct {
	Model   string `json:"model"`
	Status  string `json:"status"`
	Ready   bool   `json:"ready"`
	Message string `json:"message,omitempty"`
}

// Manager is the local runtime orchestration contract.
type Manager interface {
	Status(ctx context.Context) (Snapshot, error)
	Start(ctx context.Context) (Snapshot, error)
	Stop(ctx context.Context) (Snapshot, error)
	ListModels(ctx context.Context) (ModelCatalog, error)
	PullModel(ctx context.Context, model string) (PullResult, error)
}

// Config controls manager startup and probing behavior.
type Config struct {
	Binary              string
	Endpoint            string
	HealthPath          string
	StartupTimeout      time.Duration
	StopTimeout         time.Duration
	HealthCheckInterval time.Duration
	HTTPClient          *http.Client
	Logger              *slog.Logger
	Runner              Runner
}

// Runner abstracts process lookup/spawn for testability.
type Runner interface {
	LookPath(file string) (string, error)
	Start(binary string, args ...string) (Process, error)
}

// Process is a minimal OS process handle abstraction.
type Process interface {
	PID() int
	Signal(signal os.Signal) error
	Kill() error
	Wait() error
}

// RuntimeManager orchestrates a local runtime process and readiness checks.
type RuntimeManager struct {
	engine    string
	binary    string
	endpoint  string
	healthURL string

	startupTimeout      time.Duration
	stopTimeout         time.Duration
	healthCheckInterval time.Duration

	client *http.Client
	logger *slog.Logger
	runner Runner

	mu             sync.Mutex
	process        Process
	waitDone       chan struct{}
	startedByApp   bool
	stopping       bool
	lastProcessErr error
}

// NewOllamaManager constructs an Ollama runtime manager with safe defaults.
func NewOllamaManager(cfg Config) *RuntimeManager {
	binary := strings.TrimSpace(cfg.Binary)
	if binary == "" {
		binary = defaultBinary
	}
	endpoint := strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	if endpoint == "" {
		endpoint = DefaultOllamaEndpoint
	}
	healthPath := strings.TrimSpace(cfg.HealthPath)
	if healthPath == "" {
		healthPath = defaultHealthPath
	}
	if !strings.HasPrefix(healthPath, "/") {
		healthPath = "/" + healthPath
	}
	healthURL := endpoint + healthPath

	startupTimeout := cfg.StartupTimeout
	if startupTimeout <= 0 {
		startupTimeout = defaultStartupTimeout
	}
	stopTimeout := cfg.StopTimeout
	if stopTimeout <= 0 {
		stopTimeout = defaultStopTimeout
	}
	healthCheckInterval := cfg.HealthCheckInterval
	if healthCheckInterval <= 0 {
		healthCheckInterval = defaultHealthCheckInterval
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}
	runner := cfg.Runner
	if runner == nil {
		runner = osRunner{}
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default().With("component", "local_runtime", "engine", EngineOllama)
	}

	return &RuntimeManager{
		engine:              EngineOllama,
		binary:              binary,
		endpoint:            endpoint,
		healthURL:           healthURL,
		startupTimeout:      startupTimeout,
		stopTimeout:         stopTimeout,
		healthCheckInterval: healthCheckInterval,
		client:              client,
		logger:              logger,
		runner:              runner,
	}
}

// NewNoopManager returns a disabled manager used when runtime orchestration
// is not wired yet.
func NewNoopManager() Manager {
	return noopManager{}
}

func (m *RuntimeManager) Status(ctx context.Context) (Snapshot, error) {
	snapshot := Snapshot{
		Engine:    m.engine,
		Endpoint:  m.endpoint,
		CheckedAt: time.Now().UTC(),
	}

	installed, err := m.isInstalled()
	if err != nil {
		return snapshot, fmt.Errorf("checking local runtime installation: %w", err)
	}
	if !installed {
		snapshot.Status = StatusNotInstalled
		snapshot.Message = "Install Ollama to enable guided local model mode."
		return snapshot, nil
	}

	ready, probeMessage := m.probe(ctx)

	m.mu.Lock()
	process := m.process
	startedByApp := m.startedByApp
	lastProcessErr := m.lastProcessErr
	m.mu.Unlock()

	if ready {
		snapshot.Status = StatusReady
		if startedByApp && process != nil {
			snapshot.StartedByApp = true
			snapshot.PID = process.PID()
		}
		return snapshot, nil
	}

	if process != nil && startedByApp {
		snapshot.Status = StatusStarting
		snapshot.StartedByApp = true
		snapshot.PID = process.PID()
		if probeMessage != "" {
			snapshot.Message = probeMessage
		}
		return snapshot, nil
	}

	if lastProcessErr != nil {
		snapshot.Status = StatusError
		snapshot.Message = lastProcessErr.Error()
		return snapshot, nil
	}

	snapshot.Status = StatusStopped
	if probeMessage != "" {
		snapshot.Message = probeMessage
	}
	return snapshot, nil
}

func (m *RuntimeManager) Start(ctx context.Context) (Snapshot, error) {
	installed, err := m.isInstalled()
	if err != nil {
		return Snapshot{}, fmt.Errorf("checking local runtime installation: %w", err)
	}
	if !installed {
		snapshot, statusErr := m.Status(ctx)
		if statusErr != nil {
			return snapshot, statusErr
		}
		return snapshot, fmt.Errorf("%w: %s", ErrRuntimeNotInstalled, m.binary)
	}

	snapshot, err := m.Status(ctx)
	if err != nil {
		return snapshot, err
	}
	if snapshot.Status == StatusReady {
		return snapshot, nil
	}

	m.mu.Lock()
	if m.process == nil {
		process, startErr := m.runner.Start(m.binary, "serve")
		if startErr != nil {
			m.mu.Unlock()
			return snapshot, fmt.Errorf("starting local runtime process: %w", startErr)
		}
		m.process = process
		m.waitDone = make(chan struct{})
		m.startedByApp = true
		m.stopping = false
		m.lastProcessErr = nil

		waitDone := m.waitDone
		go m.watchProcess(process, waitDone)
	}
	m.mu.Unlock()

	return m.waitForReady(ctx)
}

func (m *RuntimeManager) Stop(ctx context.Context) (Snapshot, error) {
	m.mu.Lock()
	process := m.process
	waitDone := m.waitDone
	if process == nil || waitDone == nil {
		m.mu.Unlock()
		return m.Status(ctx)
	}
	m.stopping = true
	m.mu.Unlock()

	if err := process.Signal(os.Interrupt); err != nil {
		m.logger.Debug("failed to interrupt local runtime process; escalating to kill if needed", "error", err)
	}

	timer := time.NewTimer(m.stopTimeout)
	defer timer.Stop()

	select {
	case <-waitDone:
	case <-timer.C:
		if err := process.Kill(); err != nil {
			m.logger.Warn("failed to kill local runtime process", "error", err)
		}
		select {
		case <-waitDone:
		case <-time.After(500 * time.Millisecond):
		}
	case <-ctx.Done():
		_ = process.Kill()
		select {
		case <-waitDone:
		case <-time.After(500 * time.Millisecond):
		}
		return Snapshot{}, ctx.Err()
	}

	return m.Status(ctx)
}

func (m *RuntimeManager) waitForReady(ctx context.Context) (Snapshot, error) {
	deadline := time.Now().Add(m.startupTimeout)
	for {
		snapshot, err := m.Status(ctx)
		if err != nil {
			return snapshot, err
		}

		switch snapshot.Status {
		case StatusReady:
			return snapshot, nil
		case StatusError:
			return snapshot, fmt.Errorf("%w: %s", ErrRuntimeStartFailed, snapshot.Message)
		}

		if time.Now().After(deadline) {
			return snapshot, nil
		}

		timer := time.NewTimer(m.healthCheckInterval)
		select {
		case <-timer.C:
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return snapshot, ctx.Err()
		}
	}
}

func (m *RuntimeManager) watchProcess(process Process, waitDone chan struct{}) {
	err := process.Wait()

	m.mu.Lock()
	if m.process == process {
		wasStopping := m.stopping
		m.process = nil
		m.waitDone = nil
		m.startedByApp = false
		m.stopping = false

		if err != nil && !wasStopping {
			m.lastProcessErr = fmt.Errorf("local runtime process exited: %w", err)
		} else {
			m.lastProcessErr = nil
		}
	}
	m.mu.Unlock()

	close(waitDone)
}

func (m *RuntimeManager) isInstalled() (bool, error) {
	_, err := m.runner.LookPath(m.binary)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, exec.ErrNotFound) {
		return false, nil
	}
	return false, err
}

func (m *RuntimeManager) probe(ctx context.Context) (bool, string) {
	requestURL, err := url.Parse(m.healthURL)
	if err != nil {
		return false, fmt.Sprintf("invalid health endpoint %q", m.healthURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return false, "failed to construct health request"
	}
	response, err := m.client.Do(req)
	if err != nil {
		return false, "runtime endpoint is not reachable yet"
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 512))

	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		return true, ""
	}
	return false, fmt.Sprintf("runtime health check returned HTTP %d", response.StatusCode)
}

func (m *RuntimeManager) ListModels(ctx context.Context) (ModelCatalog, error) {
	catalog := ModelCatalog{
		Engine:      m.engine,
		Recommended: append([]string(nil), recommendedModels...),
		Installed:   []ModelInfo{},
	}

	installed, err := m.isInstalled()
	if err != nil {
		return catalog, fmt.Errorf("checking local runtime installation: %w", err)
	}
	if !installed {
		return catalog, fmt.Errorf("%w: %s", ErrRuntimeNotInstalled, m.binary)
	}

	requestURL, err := url.Parse(m.endpoint + "/api/tags")
	if err != nil {
		return catalog, fmt.Errorf("building model list request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return catalog, fmt.Errorf("creating model list request: %w", err)
	}

	response, err := m.client.Do(req)
	if err != nil {
		return catalog, fmt.Errorf("querying model catalog: %w", err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	if err != nil {
		return catalog, fmt.Errorf("reading model list response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return catalog, fmt.Errorf("model list failed with status %d", response.StatusCode)
	}

	var payload struct {
		Models []struct {
			Name       string `json:"name"`
			Digest     string `json:"digest"`
			Size       int64  `json:"size"`
			ModifiedAt string `json:"modified_at"`
		} `json:"models"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return catalog, fmt.Errorf("parsing model list response: %w", err)
	}

	for _, model := range payload.Models {
		name := strings.TrimSpace(model.Name)
		if name == "" {
			continue
		}
		info := ModelInfo{
			Name:      name,
			Digest:    strings.TrimSpace(model.Digest),
			SizeBytes: model.Size,
		}
		if modified := strings.TrimSpace(model.ModifiedAt); modified != "" {
			if parsed, parseErr := time.Parse(time.RFC3339Nano, modified); parseErr == nil {
				info.ModifiedAt = parsed
			}
		}
		catalog.Installed = append(catalog.Installed, info)
	}

	return catalog, nil
}

func (m *RuntimeManager) PullModel(ctx context.Context, model string) (PullResult, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return PullResult{}, errors.New("model name is required")
	}

	installed, err := m.isInstalled()
	if err != nil {
		return PullResult{}, fmt.Errorf("checking local runtime installation: %w", err)
	}
	if !installed {
		return PullResult{Model: model, Status: StatusNotInstalled}, fmt.Errorf("%w: %s", ErrRuntimeNotInstalled, m.binary)
	}

	body, err := json.Marshal(map[string]any{
		"name":   model,
		"stream": false,
	})
	if err != nil {
		return PullResult{}, fmt.Errorf("encoding pull request: %w", err)
	}

	requestURL, err := url.Parse(m.endpoint + "/api/pull")
	if err != nil {
		return PullResult{}, fmt.Errorf("building model pull request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL.String(), bytes.NewReader(body))
	if err != nil {
		return PullResult{}, fmt.Errorf("creating model pull request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	response, err := m.client.Do(req)
	if err != nil {
		return PullResult{}, fmt.Errorf("pulling model: %w", err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
	if err != nil {
		return PullResult{}, fmt.Errorf("reading model pull response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return PullResult{}, fmt.Errorf("model pull failed with status %d", response.StatusCode)
	}

	var payload struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return PullResult{}, fmt.Errorf("parsing model pull response: %w", err)
	}
	if strings.TrimSpace(payload.Error) != "" {
		return PullResult{}, fmt.Errorf("model pull failed: %s", strings.TrimSpace(payload.Error))
	}

	catalog, listErr := m.ListModels(ctx)
	ready := false
	if listErr == nil {
		for _, installedModel := range catalog.Installed {
			if installedModel.Name == model {
				ready = true
				break
			}
		}
	}

	status := strings.TrimSpace(payload.Status)
	if status == "" {
		status = "completed"
	}
	result := PullResult{
		Model:  model,
		Status: status,
		Ready:  ready,
	}
	if !ready && listErr != nil {
		result.Message = "Pull completed but readiness could not be confirmed yet."
	}
	return result, nil
}

type noopManager struct{}

func (noopManager) Status(context.Context) (Snapshot, error) {
	return Snapshot{
		Engine:    "none",
		Status:    StatusNotInstalled,
		Message:   "Local runtime orchestration is not configured.",
		CheckedAt: time.Now().UTC(),
	}, nil
}

func (noopManager) Start(context.Context) (Snapshot, error) {
	return Snapshot{
		Engine:    "none",
		Status:    StatusNotInstalled,
		Message:   "Local runtime orchestration is not configured.",
		CheckedAt: time.Now().UTC(),
	}, fmt.Errorf("%w: orchestration is disabled", ErrRuntimeNotInstalled)
}

func (noopManager) Stop(context.Context) (Snapshot, error) {
	return Snapshot{
		Engine:    "none",
		Status:    StatusStopped,
		Message:   "Local runtime orchestration is not configured.",
		CheckedAt: time.Now().UTC(),
	}, nil
}

func (noopManager) ListModels(context.Context) (ModelCatalog, error) {
	return ModelCatalog{
		Engine:      "none",
		Recommended: append([]string(nil), recommendedModels...),
		Installed:   []ModelInfo{},
	}, fmt.Errorf("%w: local runtime orchestration is disabled", ErrRuntimeUnsupported)
}

func (noopManager) PullModel(context.Context, string) (PullResult, error) {
	return PullResult{}, fmt.Errorf("%w: local runtime orchestration is disabled", ErrRuntimeUnsupported)
}

type osRunner struct{}

func (osRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (osRunner) Start(binary string, args ...string) (Process, error) {
	cmd := exec.Command(binary, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &execProcess{cmd: cmd}, nil
}

type execProcess struct {
	cmd *exec.Cmd
}

func (p *execProcess) PID() int {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return 0
	}
	return p.cmd.Process.Pid
}

func (p *execProcess) Signal(signal os.Signal) error {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return errors.New("process is not running")
	}
	return p.cmd.Process.Signal(signal)
}

func (p *execProcess) Kill() error {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return errors.New("process is not running")
	}
	return p.cmd.Process.Kill()
}

func (p *execProcess) Wait() error {
	if p == nil || p.cmd == nil {
		return nil
	}
	return p.cmd.Wait()
}

var _ Manager = (*RuntimeManager)(nil)
var _ Runner = osRunner{}
