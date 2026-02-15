# Architecture

## Technology Stack

| Layer | Technology |
| --- | --- |
| Desktop framework | Wails v3 |
| Backend | Go 1.24 |
| Database | SQLite via `modernc.org/sqlite` |
| Frontend | React 18 + TypeScript + Vite + Tailwind |
| Secrets | `zalando/go-keyring` |
| Notifications | Wails native notifications |
| LLM integration | Provider interface with OpenAI-first rollout, guided local-runtime mode, and OpenAI-compatible extension path |

## Runtime Model

1. Scheduler polls enabled filters.
2. Adapter fetches jobs from source APIs.
3. Jobs are deduplicated by `UNIQUE(source, source_id)`.
4. New jobs are persisted and surfaced to UI immediately.
5. Newly discovered jobs are queued for asynchronous matching.
6. Match worker atomically claims queued jobs (`pending` -> `processing`) and processes independently of poll-cycle timing.
7. UI shows pending state while score is computing and updates when match completes.
8. High-score matches can trigger native notifications.

## System Principles

- Architecture: keep polling and matching decoupled so ingestion remains fast and deterministic.
- Testability: define narrow consumer-side interfaces and keep service/package boundaries explicit.
- Resilience: use bounded contexts/timeouts, atomic queue transitions, and retry-safe DB write paths.
- Evolvability: add providers/adapters without changing core scheduler or matcher orchestration contracts.
- Maintainability: centralize shared UI status semantics and structured logging to reduce drift.

## Service Boundaries

Wails-exposed services in root package:

- `AppService`: lifecycle startup/shutdown.
- `FilterService`: search-filter CRUD.
- `JobService`: job retrieval and deletion.
- `PollingService`: poll now, schedule, diagnostics export.
- `SettingsService`: persisted settings and API-key lifecycle operations.

Internal packages:

- `internal/db`: migrations and typed DB operations.
- `internal/adapter`: source adapters and registry.
- `internal/scheduler`: polling orchestration.
- `internal/matcher`: provider-agnostic matching + async orchestration.
- `internal/localruntime` (planned): local model runtime detection/start/health abstraction for guided local mode.
- `internal/keychain`: key storage abstraction.
- `internal/diagnostics`: poll diagnostics retention/export support.

## Data Model

### `search_filters` (Phase 1 complete)

- `id` (UUID, primary key)
- `name`, `keywords`, `location`
- `source` (example: `reed_uk`)
- `enabled`
- `created_at`, `updated_at`

### `jobs` (Phase 1 complete)

- `id` (UUID, primary key)
- `source`, `source_id` (unique pair)
- `title`, `company`, `location`, `description`, `url`
- `posted_at`, `discovered_at`
- `filter_id` (foreign key to `search_filters`)

### `settings` (Phase 1 complete, expanded Phase 2+)

Representative keys:

- `poll_interval_minutes` (default 30)
- `match_threshold` (default 0.7)
- `llm_mode` (`cloud`, `local`, `advanced`)
- `llm_provider` (default `openai`)
- `llm_model` (selected model for active mode)
- `llm_base_url` (advanced mode/manual endpoint override only)
- `local_runtime_engine`, `local_runtime_model` (guided local mode selections)
- `cv_path`, `cover_letter_draft`, `custom_instructions`
- `first_run_complete`

### `job_matches` (Phase 2 in progress; active queue + scoring state)

- `id`, `job_id`
- `match_score` (0.0-1.0)
- `match_summary`
- `status` (`pending`, `processing`, `matched`, `failed`)
- `tailored_cv_path`, `tailored_cl_path`
- `status_updated_at`, `created_at`

### `llm_prompts` (Phase 2 planned)

- `id` (`matching`, `cv_tailoring`, `cover_letter_tailoring`, `ai_detection`)
- `system_prompt`
- `is_custom`
- `updated_at`

### `approved_domains` (Phase 2 planned)

- `domain` (primary key)
- `approved_at`

## LLM Extension Point

Provider logic is isolated behind an interface so matching/tailoring flows stay provider-agnostic.
Initial production provider is OpenAI. Registry design should allow additional providers, including OpenAI-compatible local/self-hosted endpoints used by OSS models.

```go
type Provider interface {
	Name() string
	DisplayName() string
	Match(ctx context.Context, req MatchRequest) (MatchResult, error)
	Validate(ctx context.Context) error
}

type MatchRequest struct {
	Query               string
	JobTitle            string
	JobCompany          string
	JobLocation         string
	JobDescription      string
	MaxDescriptionRunes int
}
```

## LLM Runtime UX and Orchestration Model

- Default setup path is mode-based, not endpoint-based:
  - `Cloud`: hosted provider flow (OpenAI-first).
  - `Local`: app-managed local runtime flow.
  - `Advanced`: explicit manual endpoint overrides.
- `SettingsService` remains the frontend boundary for settings persistence and lifecycle actions needed by the LLM setup UI.
- Matching runtime selection remains dynamic via matcher provider resolver, reading persisted mode/provider configuration per match execution.
- Local runtime orchestration responsibilities (planned in `internal/localruntime`):
  - Detect existing compatible runtimes.
  - Start/stop/check readiness of managed runtime.
  - Report actionable status states for UI (`not_installed`, `downloading`, `starting`, `ready`, `error`).
  - Keep secrets and local process details out of frontend payloads except explicit user-facing status text.

## Security Model

- API keys are stored in OS keychain, not plaintext DB.
- SQL access uses parameterized queries.
- No telemetry/analytics calls.
- Frontend can only call explicitly bound Wails service methods.
