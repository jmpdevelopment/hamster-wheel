# Known Issues and Risks

Last reviewed: 2026-02-15

## Purpose

Track active defects and active delivery risks only.

- Keep this file short and current.
- Do not maintain a resolved-issues changelog here; rely on git history and
  commit/PR notes for closure history.

## Open Bugs

- No confirmed open functional defects are currently tracked.

When a bug is added, include:

- Severity (`critical`, `high`, `medium`, `low`)
- Impacted area
- Reproduction notes
- Planned fix step
- Regression test reference

When a bug is fixed, remove it from this file in the same change that adds the
regression test.

## Active Risks

| Risk | Why it Matters | Current Mitigation | Planned Reduction |
| --- | --- | --- | --- |
| Job source API changes or deprecation | Could break polling and discovery quality. | Adapter pattern keeps source integration isolated. | Add adapter contract tests and source health checks as new sources are added. |
| LLM API cost growth | Could increase per-match spend and slow adoption. | Matching is asynchronous; default scorer is local heuristic; threshold is configurable. | Complete token-efficiency controls and guided local-runtime mode in Phase 2. |
| Local-model onboarding complexity for non-technical users | Manual endpoint/model concepts can cause setup drop-off and misconfiguration. | OpenAI provider and runtime selection wiring are complete; backend local-runtime orchestration plus service/binding exposure exist (`internal/localruntime` + `SettingsService` runtime methods); advanced endpoint controls remain available for experts. | Implement guided `Local` UX and managed model lifecycle with advanced-only manual endpoint fallback. |
| PDF parse/tailoring accuracy variance | Could reduce output trust in Phase 3 tailoring workflows. | Tailoring phase is gated behind current matching completion. | Add parse quality checks and deterministic preview loops in Phase 3. |
| Unsigned desktop package warnings | Can reduce user trust during distribution testing. | Document current unsigned-build behavior. | Add signing/notarization tasks in Phase 5 distribution hardening. |
| Unbounded local data growth | Long-running installs may accumulate too much local data. | Data hygiene step is planned in distribution phase. | Implement retention/cleanup controls and safe defaults before release hardening exit. |
