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

## Current Runtime Posture (Preserve)

- Keep polling and matching decoupled.
- Keep provider integration behind `internal/llm.Provider` and `internal/llm.Registry`.
- Keep the default scorer local (`heuristic_v1`) until external providers are fully configured.
- Keep event-driven updates (`polling:status-changed`, `matching:status-changed`) with bounded/coalesced UI refresh.
- Keep SQLite runtime safeguards (single shared connection + busy retry on writes).

## Verification Baseline (Latest Pass: 2026-02-14)

- `go test ./...`: passing
- `go vet ./...`: passing
- `./scripts/check-coverage.sh`: passing
- Backend total coverage: 78.3%
- Frontend line coverage: 99.6%

## Next Work

Execution-ready backlog is in `docs/execution/roadmap.md` under
`Phase 2 Immediate Implementation Queue`.
