# Status

Last updated: 2026-02-14

## Current State

- Phase 1 foundation is complete.
- Architecture refactor is complete.
- Phase 1.5 UX standards are complete.
- Pre-LLM job management UX enhancements are complete:
  - Filter delete now supports optional deletion of associated jobs.
  - Job board supports multi-select (including shift-range) + select-all bulk delete.
  - Job board supports bulk favorite/unfavorite and favorites-only view.
  - Favorites are persisted in SQLite and survive app restart.
  - `Poll Now` is available when filters are enabled and blocked only during active poll cycles (spinner/status shown).
- Phase 2 has started with keychain manager completed.
- Phase 2 LLM sequencing is now OpenAI-first; Claude is deferred behind provider registry completion.
- Phase 2 matching architecture is async and decoupled from polling; UI should show pending match state while scores compute.
- Phase 2 async matching groundwork is implemented:
  - Polling now enqueues new jobs into `job_matches` with `pending` status.
  - Job queries expose match status/score/summary in the existing job payload.
  - Job list UI now shows a compact horizontal match-status badge in the title row (top-right) to preserve fixed card height and prevent row overflow.
- Phase 2 async matcher worker orchestration is implemented:
  - Matcher runs independently from polling and processes `pending` matches in background batches.
  - Claiming is atomic (`pending` -> `processing`) and stale `processing` rows are requeued on timeout windows.
  - Match updates emit `matching:status-changed` events and frontend coalesces refreshes for responsive UI updates.
  - Current default scorer is local heuristic (`heuristic_v1`) to keep latency low and token usage at zero while external providers are integrated.
- Reliability hardening pass is complete and regression-covered.

## Key Reliability Outcomes

- Scheduler polls are timeout-bounded and panic-safe.
- Poll execution is single-flight: overlapping manual/scheduled cycles are blocked.
- Poll status is event-driven (`polling:status-changed`) with focus/visibility fallback sync.
- SQLite runtime reliability is hardened:
  - DB runs with a single shared SQLite connection so FK actions remain deterministic.
  - SQLite startup applies `busy_timeout`; all DB writes use bounded context-aware busy retry.
- Polling schedule is deterministic:
  - `nextPollAt` is published before cycle work.
  - resume and manual `PollNow` both reschedule auto-poll to `now + interval`.
- Startup/scheduled poll completion summaries are surfaced to UI state so users
  receive poll-result toasts even when no manual `PollNow` action occurred.
- Frontend polling orchestration is isolated in `usePollingController`.
- Poll diagnostics and export paths are available for failure triage.

## Verification Baseline from Latest Status Pass

- Backend tests: passing.
- Frontend tests: passing.
- `go vet ./...`: passing.
- Coverage gates: passing.

## Next Work Queue

1. Phase 2: implement OpenAI provider.
2. Phase 2: implement OpenAI-compatible provider path for self-hosted/local models.
3. Phase 2: wire provider selection + key/model/base-URL settings into matcher runtime.
4. Phase 2: implement CV parser path for matching inputs.
5. Phase 2: tune token-efficiency controls (compact prompt shaping, prefilter thresholds, and bounded context windows).
6. Phase 2: wire completed match thresholds and notifications in UI.
