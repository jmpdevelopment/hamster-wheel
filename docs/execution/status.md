# Status

Last updated: 2026-02-14

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

## Current Runtime Posture (Preserve)

- Keep polling and matching decoupled.
- Keep provider integration behind `internal/llm.Provider` and `internal/llm.Registry`.
- Keep the default scorer local (`heuristic_v1`) until external providers are fully configured.
- Keep OpenAI provider registered but non-default until runtime provider-selection settings are complete.
- Keep LLM settings persisted via `SettingsService` while runtime hot-switch wiring remains in the next queue step.
- Keep event-driven updates (`polling:status-changed`, `matching:status-changed`) with bounded/coalesced UI refresh.
- Keep SQLite runtime safeguards (single shared connection + busy retry on writes).
- Keep CV-context matching fail-open: missing/invalid CV path logs warning and falls back to query-only matching.

## Verification Baseline (Latest Pass: 2026-02-14)

- `go test ./...`: passing
- `go vet ./...`: passing
- `./scripts/check-coverage.sh`: passing
- Backend total coverage: 82.6%
- Frontend line coverage: 99.6%

## Next Work

Execution-ready backlog is in `docs/execution/roadmap.md` under
`Phase 2 Immediate Implementation Queue`.
