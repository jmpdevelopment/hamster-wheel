# Known Issues and Risks

Last reviewed: 2026-02-14

## Open Bugs

- No confirmed open functional defects are currently tracked in this file.

## Recently Resolved

- 2026-02-14: SQLite foreign-key actions (`ON DELETE SET NULL` / `CASCADE`)
  could be inconsistently enforced across pooled DB connections; fixed by
  pinning the app DB to a single shared SQLite connection and preserving
  explicit FK pragma initialization on startup.
- 2026-02-14: SQLite busy-retry behavior was inconsistent across DB write paths;
  fixed by applying context-aware bounded busy retry uniformly to filter/job/
  settings writes and configuring SQLite `busy_timeout` during DB startup.
- 2026-02-14: shift-range selection on job checkboxes was inconsistent after
  list refreshes and could only work once; fixed by stabilizing the
  selection-anchor state used by checkbox and card interactions.
- 2026-02-14: bulk-delete operations could fail with `SQLITE_BUSY` under heavy
  concurrent delete requests from UI; fixed by serializing delete calls and
  adding bounded retry for transient SQLite lock contention.
- 2026-02-14: favorites were stored only in frontend state and were lost on app
  restart; fixed by persisting favorite state in the `jobs` table.
- 2026-02-14: settings/favorite writes could intermittently fail with
  `SQLITE_BUSY` while poll writes were active; fixed by adding bounded SQLite
  busy-retry handling on DB write paths.
- 2026-02-14: first app-start poll completion did not show toast feedback;
  fixed by exposing latest scheduler run in polling status/event payloads and
  consuming it in frontend poll state.

When a bug is found, record:

- Severity (`critical`, `high`, `medium`, `low`)
- Impacted area
- Reproduction notes
- Planned fix step
- Regression test reference after fix

## Active Watchlist

- Distribution signing/notarization is deferred; users may see trust warnings on unsigned builds.
- LLM matching and tailoring phases are not fully implemented yet; quality depends on upcoming Phase 2-3 completion.
- PDF parsing/generation quality may vary by document structure and will need targeted validation.
- Job retention is currently indefinite; data hygiene controls (retention policy, archival/cleanup UX, and safe defaults) are deferred to end-phase hardening.

## Risk Register

| Risk | Current Mitigation |
| --- | --- |
| Job source API changes or deprecation | Adapter pattern allows source replacement with minimal impact. |
| LLM API cost growth | Keep matching targeted and thresholds configurable; default model choice is cost-aware. |
| Unsigned desktop package warnings | Document user workarounds now; revisit code-signing in distribution phase. |
| PDF parse inaccuracies | Add preview/review steps and fallback text workflows before final tailoring. |
| Unbounded local data growth from indefinite retention | Track planned data hygiene phase for retention/cleanup controls before distribution hardening completes. |
