package localruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	EngineOllama = "ollama"

	RecommendedLlamaModel       = "llama3.1:8b"
	EstimatedLlamaDownloadBytes = int64(4_900_000_000)

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
	defaultPullTimeout         = 45 * time.Minute
	defaultHealthCheckInterval = 250 * time.Millisecond
	defaultHTTPTimeout         = 1200 * time.Millisecond
	defaultRetryDelay          = 300 * time.Millisecond
	defaultRequestAttempts     = 3
	maxResponseBytes           = 1 << 20 // 1 MiB
)

var (
	ErrRuntimeNotInstalled = errors.New("local runtime is not installed")
	ErrRuntimeStartFailed  = errors.New("local runtime failed to start")
	ErrRuntimeUnsupported  = errors.New("local runtime feature is unsupported")
)

var recommendedModels = []string{
	RecommendedLlamaModel,
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

// PullProgress captures in-flight model pull telemetry.
type PullProgress struct {
	Model          string    `json:"model"`
	Active         bool      `json:"active"`
	Status         string    `json:"status,omitempty"`
	Message        string    `json:"message,omitempty"`
	Digest         string    `json:"digest,omitempty"`
	TotalBytes     int64     `json:"totalBytes,omitempty"`
	CompletedBytes int64     `json:"completedBytes,omitempty"`
	Percent        float64   `json:"percent,omitempty"`
	Ready          bool      `json:"ready"`
	StartedAt      time.Time `json:"startedAt,omitempty"`
	UpdatedAt      time.Time `json:"updatedAt,omitempty"`
}

// Manager is the local runtime orchestration contract.
type Manager interface {
	Status(ctx context.Context) (Snapshot, error)
	Start(ctx context.Context) (Snapshot, error)
	Stop(ctx context.Context) (Snapshot, error)
	ListModels(ctx context.Context) (ModelCatalog, error)
	PullModel(ctx context.Context, model string) (PullResult, error)
	GetPullProgress(ctx context.Context) (PullProgress, error)
}

// Config controls manager startup and probing behavior.
type Config struct {
	Binary              string
	Endpoint            string
	HealthPath          string
	ManagedStateFile    string
	StartupTimeout      time.Duration
	StopTimeout         time.Duration
	PullTimeout         time.Duration
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
	pullTimeout         time.Duration
	healthCheckInterval time.Duration

	client *http.Client
	logger *slog.Logger
	runner Runner

	managedStateFile string

	mu             sync.Mutex
	process        Process
	waitDone       chan struct{}
	startedByApp   bool
	stopping       bool
	lastProcessErr error
	pullProgress   PullProgress
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
	pullTimeout := cfg.PullTimeout
	if pullTimeout <= 0 {
		pullTimeout = defaultPullTimeout
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
	managedStateFile := strings.TrimSpace(cfg.ManagedStateFile)
	if managedStateFile != "" {
		managedStateFile = filepath.Clean(managedStateFile)
	}

	manager := &RuntimeManager{
		engine:              EngineOllama,
		binary:              binary,
		endpoint:            endpoint,
		healthURL:           healthURL,
		startupTimeout:      startupTimeout,
		stopTimeout:         stopTimeout,
		pullTimeout:         pullTimeout,
		healthCheckInterval: healthCheckInterval,
		client:              client,
		logger:              logger,
		runner:              runner,
		managedStateFile:    managedStateFile,
	}
	if managedStateFile != "" {
		manager.reapStaleManagedProcess()
	}
	return manager
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
		snapshot.Message = "Install Ollama, open it once, then return to local model setup."
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
		if err := m.writeManagedState(process.PID()); err != nil {
			m.logger.Warn("failed to persist managed runtime state", "error", err)
		}

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
		if err := m.clearManagedState(); err != nil {
			m.logger.Debug("failed to clear managed runtime state after context cancellation", "error", err)
		}
		return Snapshot{}, ctx.Err()
	}

	if err := m.clearManagedState(); err != nil {
		m.logger.Debug("failed to clear managed runtime state during stop", "error", err)
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

	shouldClearManagedState := false
	m.mu.Lock()
	if m.process == process {
		wasStopping := m.stopping
		m.process = nil
		m.waitDone = nil
		m.startedByApp = false
		m.stopping = false
		shouldClearManagedState = true

		if err != nil && !wasStopping {
			m.lastProcessErr = fmt.Errorf("local runtime process exited: %w", err)
		} else {
			m.lastProcessErr = nil
		}
	}
	m.mu.Unlock()
	if shouldClearManagedState {
		_ = m.clearManagedState()
	}

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

	response, err := m.doWithRetry(ctx, "list-models", func() (*http.Response, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
		if reqErr != nil {
			return nil, fmt.Errorf("creating model list request: %w", reqErr)
		}
		return m.client.Do(req)
	})
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

func (m *RuntimeManager) PullModel(ctx context.Context, model string) (result PullResult, err error) {
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
	if err := m.beginPullTracking(model); err != nil {
		return PullResult{}, err
	}
	defer func() {
		if err != nil {
			m.completePullTracking(model, "failed", err.Error(), false)
			return
		}

		status := strings.TrimSpace(result.Status)
		if status == "" {
			status = "completed"
		}
		message := strings.TrimSpace(result.Message)
		if message == "" {
			message = status
		}
		m.completePullTracking(model, status, message, result.Ready)
	}()

	body, err := json.Marshal(map[string]any{
		"name":   model,
		"stream": true,
	})
	if err != nil {
		return PullResult{}, fmt.Errorf("encoding pull request: %w", err)
	}

	requestURL, err := url.Parse(m.endpoint + "/api/pull")
	if err != nil {
		return PullResult{}, fmt.Errorf("building model pull request: %w", err)
	}

	pullCtx := ctx
	cancelPull := func() {}
	if m.pullTimeout > 0 {
		pullCtx, cancelPull = context.WithTimeout(ctx, m.pullTimeout)
	}
	defer cancelPull()

	pullClient := m.cloneClientWithTimeout(0)
	response, err := m.doWithRetry(pullCtx, "pull-model", func() (*http.Response, error) {
		req, reqErr := http.NewRequestWithContext(pullCtx, http.MethodPost, requestURL.String(), bytes.NewReader(body))
		if reqErr != nil {
			return nil, fmt.Errorf("creating model pull request: %w", reqErr)
		}
		req.Header.Set("Content-Type", "application/json")
		return pullClient.Do(req)
	})
	if err != nil {
		return PullResult{}, fmt.Errorf("pulling model: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		raw, readErr := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes))
		if readErr != nil {
			return PullResult{}, fmt.Errorf("reading model pull failure response: %w", readErr)
		}
		if detail := decodePullFailureMessage(raw); detail != "" {
			return PullResult{}, fmt.Errorf("model pull failed with status %d: %s", response.StatusCode, detail)
		}
		return PullResult{}, fmt.Errorf("model pull failed with status %d", response.StatusCode)
	}

	decoder := json.NewDecoder(response.Body)
	lastStatus := "completed"
	for {
		var event pullStreamEvent
		decodeErr := decoder.Decode(&event)
		if decodeErr != nil {
			if errors.Is(decodeErr, io.EOF) {
				break
			}
			return PullResult{}, fmt.Errorf("parsing model pull progress stream: %w", decodeErr)
		}

		if eventError := strings.TrimSpace(event.Error); eventError != "" {
			return PullResult{}, fmt.Errorf("model pull failed: %s", eventError)
		}
		if status := strings.TrimSpace(event.Status); status != "" {
			lastStatus = status
		}
		m.updatePullTrackingFromEvent(model, event)
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

	result = PullResult{
		Model:  model,
		Status: strings.TrimSpace(lastStatus),
		Ready:  ready,
	}
	if result.Status == "" {
		result.Status = "completed"
	}
	if !ready && listErr != nil {
		result.Message = "Pull completed but readiness could not be confirmed yet."
	}
	return result, nil
}

// GetPullProgress returns the current model-pull progress snapshot.
func (m *RuntimeManager) GetPullProgress(context.Context) (PullProgress, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pullProgress, nil
}

type pullStreamEvent struct {
	Status    string `json:"status"`
	Error     string `json:"error"`
	Digest    string `json:"digest"`
	Total     int64  `json:"total"`
	Completed int64  `json:"completed"`
}

func decodePullFailureMessage(raw []byte) string {
	var payload struct {
		Error   string `json:"error"`
		Message string `json:"message"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	for _, candidate := range []string{payload.Error, payload.Message, payload.Status} {
		if text := strings.TrimSpace(candidate); text != "" {
			return text
		}
	}
	return ""
}

func (m *RuntimeManager) beginPullTracking(model string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.pullProgress.Active {
		activeModel := strings.TrimSpace(m.pullProgress.Model)
		if activeModel == "" {
			activeModel = "another model"
		}
		return fmt.Errorf("model pull already in progress for %s", activeModel)
	}

	now := time.Now().UTC()
	m.pullProgress = PullProgress{
		Model:     model,
		Active:    true,
		Status:    "starting",
		Message:   "Starting download",
		Ready:     false,
		StartedAt: now,
		UpdatedAt: now,
	}
	return nil
}

func (m *RuntimeManager) updatePullTrackingFromEvent(model string, event pullStreamEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.pullProgress.Active || m.pullProgress.Model != model {
		return
	}

	status := strings.TrimSpace(event.Status)
	if status != "" {
		m.pullProgress.Status = status
	}
	if digest := strings.TrimSpace(event.Digest); digest != "" {
		m.pullProgress.Digest = digest
	}
	if event.Total > 0 {
		m.pullProgress.TotalBytes = event.Total
	}
	if event.Completed >= 0 {
		m.pullProgress.CompletedBytes = event.Completed
	}
	if m.pullProgress.TotalBytes > 0 {
		m.pullProgress.Percent = clampPullPercent(
			float64(m.pullProgress.CompletedBytes) / float64(m.pullProgress.TotalBytes) * 100,
		)
	}
	if m.pullProgress.Status != "" {
		m.pullProgress.Message = m.pullProgress.Status
	}
	m.pullProgress.UpdatedAt = time.Now().UTC()
}

func (m *RuntimeManager) completePullTracking(model, status, message string, ready bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.pullProgress.Model != model {
		return
	}
	m.pullProgress.Active = false
	m.pullProgress.Ready = ready
	if text := strings.TrimSpace(status); text != "" {
		m.pullProgress.Status = text
	}
	if text := strings.TrimSpace(message); text != "" {
		m.pullProgress.Message = text
	}
	if ready && m.pullProgress.Percent < 100 {
		m.pullProgress.Percent = 100
	}
	m.pullProgress.UpdatedAt = time.Now().UTC()
}

func clampPullPercent(percent float64) float64 {
	switch {
	case math.IsNaN(percent), math.IsInf(percent, 0), percent < 0:
		return 0
	case percent > 100:
		return 100
	default:
		return percent
	}
}

func (m *RuntimeManager) cloneClientWithTimeout(timeout time.Duration) *http.Client {
	if m.client == nil {
		return &http.Client{Timeout: timeout}
	}
	clone := *m.client
	clone.Timeout = timeout
	return &clone
}

type managedProcessState struct {
	PID      int       `json:"pid"`
	Binary   string    `json:"binary"`
	Recorded time.Time `json:"recorded"`
}

func (m *RuntimeManager) doWithRetry(
	ctx context.Context,
	operation string,
	requestFn func() (*http.Response, error),
) (*http.Response, error) {
	var lastErr error
	for attempt := 1; attempt <= defaultRequestAttempts; attempt++ {
		response, err := requestFn()
		if err == nil {
			return response, nil
		}
		lastErr = err
		if !isRetryableNetworkError(err) || attempt == defaultRequestAttempts {
			break
		}
		delay := time.Duration(math.Pow(2, float64(attempt-1))) * defaultRetryDelay
		m.logger.Warn(
			"local runtime request failed; retrying",
			"operation", operation,
			"attempt", attempt,
			"error", err,
			"retry_delay_ms", delay.Milliseconds(),
		)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	if lastErr == nil {
		lastErr = errors.New("request failed")
	}
	if isRetryableNetworkError(lastErr) {
		return nil, fmt.Errorf("runtime endpoint is unreachable; ensure Ollama is running and network is available: %w", lastErr)
	}
	return nil, lastErr
}

func isRetryableNetworkError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false
	}
	lower := strings.ToLower(err.Error())
	retryableFragments := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"network is unreachable",
		"no route to host",
		"no such host",
		"temporary failure",
		"i/o timeout",
		"tls handshake timeout",
	}
	for _, fragment := range retryableFragments {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}

func (m *RuntimeManager) reapStaleManagedProcess() {
	state, err := m.readManagedState()
	if err != nil {
		m.logger.Warn("failed to read local runtime managed-state file; removing", "path", m.managedStateFile, "error", err)
		_ = m.clearManagedState()
		return
	}
	if state.PID <= 0 {
		_ = m.clearManagedState()
		return
	}

	if !processLooksLikeBinary(state.PID, state.Binary) {
		m.logger.Info("stale local runtime state file does not match a live managed process; clearing", "pid", state.PID)
		_ = m.clearManagedState()
		return
	}

	m.logger.Warn("reaping stale app-managed local runtime process from previous session", "pid", state.PID)
	if err := signalProcessTree(state.PID, os.Interrupt); err != nil {
		m.logger.Debug("failed to send interrupt to stale local runtime process", "pid", state.PID, "error", err)
	}
	deadline := time.Now().Add(m.stopTimeout)
	for time.Now().Before(deadline) {
		if !isProcessAlive(state.PID) {
			_ = m.clearManagedState()
			return
		}
		time.Sleep(75 * time.Millisecond)
	}
	if err := killProcessTree(state.PID); err != nil {
		m.logger.Warn("failed to kill stale local runtime process", "pid", state.PID, "error", err)
	}
	_ = m.clearManagedState()
}

func (m *RuntimeManager) writeManagedState(pid int) error {
	if strings.TrimSpace(m.managedStateFile) == "" || pid <= 0 {
		return nil
	}

	state := managedProcessState{
		PID:      pid,
		Binary:   m.binary,
		Recorded: time.Now().UTC(),
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encoding managed state: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(m.managedStateFile), 0o755); err != nil {
		return fmt.Errorf("creating managed state directory: %w", err)
	}
	tmpPath := m.managedStateFile + ".tmp"
	if err := os.WriteFile(tmpPath, raw, 0o600); err != nil {
		return fmt.Errorf("writing managed state tmp file: %w", err)
	}
	if err := os.Rename(tmpPath, m.managedStateFile); err != nil {
		return fmt.Errorf("moving managed state file into place: %w", err)
	}
	return nil
}

func (m *RuntimeManager) clearManagedState() error {
	if strings.TrimSpace(m.managedStateFile) == "" {
		return nil
	}
	if err := os.Remove(m.managedStateFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (m *RuntimeManager) readManagedState() (managedProcessState, error) {
	if strings.TrimSpace(m.managedStateFile) == "" {
		return managedProcessState{}, nil
	}
	raw, err := os.ReadFile(m.managedStateFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return managedProcessState{}, nil
		}
		return managedProcessState{}, err
	}
	var state managedProcessState
	if err := json.Unmarshal(raw, &state); err != nil {
		return managedProcessState{}, err
	}
	return state, nil
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

func (noopManager) GetPullProgress(context.Context) (PullProgress, error) {
	return PullProgress{
		Active:  false,
		Status:  "disabled",
		Message: "Local runtime orchestration is not configured.",
	}, nil
}

type osRunner struct{}

func (osRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (osRunner) Start(binary string, args ...string) (Process, error) {
	cmd := exec.Command(binary, args...)
	configureProcessGroup(cmd)
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
	if err := signalProcessTree(p.cmd.Process.Pid, signal); err == nil {
		return nil
	}
	return p.cmd.Process.Signal(signal)
}

func (p *execProcess) Kill() error {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return errors.New("process is not running")
	}
	if err := killProcessTree(p.cmd.Process.Pid); err == nil {
		return nil
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
