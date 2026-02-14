# Architecture Decisions

Status legend:

- Accepted: locked and active.
- Proposed: not yet locked.

## D-001: Use Wails v3 for desktop app architecture

- Status: Accepted
- Decision: Build as Go backend + React frontend bundled as native desktop app.
- Rationale: Keeps business logic in Go, supports native packaging, and avoids maintaining separate local server processes.

## D-002: Use Reed UK as active job source

- Status: Accepted
- Decision: Standardize current source identifiers and adapter behavior on `reed_uk`.
- Rationale: Previous source assumptions became stale; locking on one reliable source keeps behavior consistent while adapter extension remains available.

## D-003: Split Wails API into five focused services

- Status: Accepted
- Decision: Expose `AppService`, `FilterService`, `JobService`, `PollingService`, and `SettingsService` as separate bindings.
- Rationale: Reduces coupling, keeps API surfaces clear, and makes tests and ownership boundaries explicit.

## D-004: Store API keys in OS keychain

- Status: Accepted
- Decision: Never return stored secrets to frontend; expose presence and lifecycle operations only.
- Rationale: Minimizes secret leakage risk and aligns with platform-native secure storage.

## D-005: Concurrent fetch with sequential DB writes

- Status: Accepted
- Decision: Poll filters concurrently but serialize persistence operations.
- Rationale: Increases network throughput while preserving predictable SQLite write behavior.

## D-006: Enforce DB-level deduplication

- Status: Accepted
- Decision: Keep `UNIQUE(source, source_id)` on `jobs`.
- Rationale: Deduplication must be guaranteed at persistence boundary, not only in app logic.

## D-007: Use pure-Go SQLite driver

- Status: Accepted
- Decision: Use `modernc.org/sqlite` instead of CGO-backed drivers.
- Rationale: Simplifies build/distribution and keeps local development environment lighter.

## D-008: Define interfaces at consumption points

- Status: Accepted
- Decision: Consumer packages define narrow interfaces (for example scheduler `JobStore`) rather than exporting wide producer interfaces.
- Rationale: Keeps dependencies minimal and improves test isolation.

## D-009: Propagate context through DB layer

- Status: Accepted
- Decision: All DB methods accept `context.Context`.
- Rationale: Enables cancellation/timeouts and consistent request-scoped tracing in future enhancements.
