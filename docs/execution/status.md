# Status

Last updated: 2026-02-15

## Delivered Baseline

- Phase 1 foundation is complete.
- Phase 1.5 UX standards are complete.
- Polling reliability hardening is complete (single-flight execution, deterministic next poll scheduling, bounded retries, diagnostics export).
- Phase 2 async matching foundation is complete:
  - Polling is ingestion-only.
  - Matching runs asynchronously in a separate worker.
  - Queue processing uses atomic claim (`pending` -> `processing`) with stale-processing requeue.
- Match status UX is implemented in list and detail views, including `Match Score: X%` and per-job `Recalculate score`.
- Matcher observability and UI consistency improvements are complete (structured logs + reusable status badge system).
- Phase 2 OpenAI provider implementation is complete:
  - `internal/llm/openai` implements `Match` and `Validate`.
  - Match parsing is deterministic (`score` + `summary` JSON contract).
  - Timeout/auth/malformed-response paths are classified and test-covered.
  - Prompt-token reporting uses API usage when present with deterministic fallback estimation.
- Phase 2 runtime provider selection wiring is complete:
  - Matcher resolves provider/model/base-URL settings dynamically per match execution via provider resolver.
  - Runtime changes do not require worker restart assumptions.
- Settings panel IA refresh is complete:
  - Tabs are now split into `Interface`, `Jobs Providers`, and `LLM Providers`.
  - `Interface` includes theme + keyboard-shortcut controls.
  - `Jobs Providers` hosts Reed API key management.
  - `LLM Providers` hosts OpenAI key management and persisted provider/model/base-URL configuration.
- Backend coverage floor hardening is complete:
  - Coverage gate defaults now enforce backend `>= 80%`.
  - Added branch-focused matcher/OpenAI/settings tests to keep sustained headroom above gate.
- Phase 2 CV matching-context path is complete (PDF + plain-text scope):
  - Settings now support CV file-path submission (`cv_path`) in the Settings panel.
  - CV path submission performs immediate parse validation and rejects unsupported formats.
  - Matcher loads, parses, and caches CV text context from configured file paths.
  - Match requests now include compact CV profile context for provider scoring.
  - Runtime CV parse/load failures (for example moved/deleted files after save) degrade gracefully to query-only scoring.
- Phase 2 local-runtime backend foundation is complete (Ollama-first):
  - Added `internal/localruntime` manager for runtime detect/status/start/stop orchestration.
  - Manager emits deterministic states (`not_installed`, `stopped`, `starting`, `ready`, `error`) and tracks app-managed process ownership.
  - Runtime manager is now wired into backend composition (`main.go`) and injected into `SettingsService` for upcoming service/binding/UI integration.
- Phase 2 local-runtime service/binding integration is complete:
  - `SettingsService` now exposes `GetLocalRuntimeStatus`, `StartLocalRuntime`, and `StopLocalRuntime`.
  - Wails bindings now include local-runtime snapshot models and service call wrappers for these lifecycle actions.
  - Error paths are wrapped with operation context and test-covered.
- Phase 2 mode-aware runtime resolver + guided LLM mode UX is complete:
  - Matcher provider resolution now reads `llm_mode` per run and routes `Local` mode to app-managed Ollama runtime/model settings.
  - `SettingsService`/bindings now expose `llm_mode`, local runtime model settings, and local model catalog/pull operations for UI orchestration.
  - Settings UI now uses progressive disclosure (`Cloud`, `Local`, `Advanced`) so raw base-URL endpoint input is only visible in `Advanced`.
  - Cloud mode save flow resets manual base-URL overrides, preserving endpoint-free defaults for non-technical setup.
- Phase 2 local setup hardening is in place (Llama-only path):
  - Local mode now guides users through runtime readiness (`status`/`start`/`stop`) and in-app model pull for `llama3.1:8b`.
  - Local mode setup controls are explicit (`Start runtime` and `Download Llama`) with no combined one-click setup action.
  - Local setup shows estimated model download footprint and blocks Local-mode enablement until runtime + model readiness are confirmed.
  - Local setup now includes minimum/recommended system requirements and expected machine-impact messaging for local Llama usage.
  - Local setup instructions now explicitly require opening Ollama once after install before continuing local setup.
  - Local model pull now uses a dedicated long-running request path bounded by pull-timeout settings (avoids short generic health/request timeouts during first download).
  - Local model pull now exposes progress telemetry (`active`, status text, bytes, percent) via `GetLocalRuntimePullProgress`.
  - Settings local panel now restores in-flight download state on reopen, shows progress bar/details, blocks duplicate Llama download triggers while a pull is active, and auto-hides pull status UI after successful completion.
  - Matcher now applies a provider-specific timeout override for `local_ollama` to reduce first-match cold-start timeouts immediately after local model setup.
  - Runtime manager now tracks app-managed process state and reaps stale managed runtime processes on next launch after abnormal app exit.
  - OpenAI-compatible provider error text is now endpoint-agnostic to avoid cloud-brand leakage when local runtime path is active.
  - Local setup UI now surfaces Llama license/use-policy links and `Built with Llama` attribution notice.
- Phase 2 matching cost/control guardrails are now in place:
  - Users can now control auto-match queueing in Settings (`enabled/disabled` + per-poll-cycle limit with `0 = unlimited`).
  - When auto-match is disabled or capped, new jobs are still ingested while manual per-job recalculation remains available.
- Polling startup/settings control is now in place:
  - Auto polling is now disabled by default on startup until explicitly enabled by the user.
  - Jobs Providers settings now include auto-poll enable/disable and polling interval controls.
  - Polling interval is validated to `30`-`1440` minutes and applied live to the running scheduler.
- Job-retention controls are now in place:
  - Jobs Providers settings now include `job_retention_days` with validation to `1`-`30` calendar days.
  - Scheduler now skips fetched jobs older than the configured retention window (based on `posted_at`).
  - App startup now runs a retention cleanup pass against SQLite to prune stale persisted jobs.
- Phase 2 Adzuna prompt-context enhancement is complete:
  - Matcher now forwards Adzuna job URLs into match requests (Adzuna-only) so LLM prompts can include the listing URL alongside snippet text.
  - Adzuna scope note now explicitly states snippet-only source scope and URL-in-prompt context.
  - UI Adzuna notice now states that matching prompts include snippet + job URL when available.

## Current Runtime Posture (Preserve)

- Keep polling and matching decoupled.
- Keep provider integration behind `internal/llm.Provider` and `internal/llm.Registry`.
- Keep the default scorer local (`heuristic_v1`) until external providers are fully configured.
- Keep mode/provider/model/base-URL settings applied via matcher provider resolver per run.
- Keep event-driven updates (`polling:status-changed`, `matching:status-changed`) with bounded/coalesced UI refresh.
- Keep SQLite runtime safeguards (single shared connection + busy retry on writes).
- Keep CV-context matching fail-open: missing/invalid CV path logs warning and falls back to query-only matching.
- Preserve easy-first LLM setup direction for upcoming work: default UX is guided `Cloud`/`Local`; manual endpoints are advanced-only.
- Keep guided mode selector and progressive-disclosure visibility rules stable while local runtime lifecycle UX is expanded next.
- Keep Local mode pinned to guided `llama3.1:8b` workflow until multi-model UX + policy handling is intentionally designed.

## Verification Baseline (Latest Pass: 2026-02-14)

- `go test ./...`: passing
- `go vet ./...`: passing
- `./scripts/check-coverage.sh`: passing
- Backend total coverage: 82.6%
- Frontend line coverage: 99.6%

## Next Work

Execution-ready backlog is in `docs/execution/roadmap.md` under
`Phase 2 Immediate Implementation Queue`.
