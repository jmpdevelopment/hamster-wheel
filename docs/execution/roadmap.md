# Roadmap

## Planning Principles

- Preserve decoupled architecture: polling is ingestion-only, matching is asynchronous.
- Preserve provider-agnostic contracts so proprietary and OSS/local models are interchangeable.
- Optimize for token efficiency and low latency before adding feature breadth.
- Keep delivery incremental in this order: data layer -> business logic -> bindings -> UI -> verification/docs.
- Maintain accessibility and visual consistency (light/dark support, color-safe status communication, keyboard-first UX).

## Current Phase Snapshot

- Phase 1 (Foundation): complete.
- Phase 1.5 (UX standards): complete.
- Phase 2 (LLM matching): in progress.
- Phase 3 (Document tailoring): planned.
- Phase 4 (Full UI/workflow polish): planned.
- Phase 5 (Distribution): planned.

## Phase 2 Completed Foundation

- Provider contract and registry exist in `internal/llm`.
- Matching is decoupled from polling through async queue + worker.
- Match queue uses atomic claim semantics with stale-processing recovery.
- Runtime provider selection/hot-switch wiring is implemented through matcher provider resolver (settings are read per match run; no restart required).
- List/detail UI surfaces match status and score; users can manually requeue recalculation.
- Default scorer is local heuristic until external providers are integrated.
- OpenAI provider implementation exists in `internal/llm/openai` with deterministic response parsing and classified timeout/auth/malformed failure handling.
- Settings APIs/UI now persist LLM provider/model/base-URL and OpenAI key lifecycle in `SettingsService` + Settings panel tabs.
- Local runtime backend foundation exists in `internal/localruntime` (detect/status/start/stop for Ollama-first orchestration with deterministic runtime state model).
- CV path submission and matcher-context ingestion are implemented for PDF + plain-text CV files:
  - `cv_path` is persisted via `SettingsService` and configurable in Settings UI.
  - Unsupported CV formats are rejected at submission-time validation.
  - Matcher parses/caches CV text and includes compact profile context in `MatchRequest`.
  - Runtime CV parse/load issues fail open to query-only scoring.

## Phase 2 Immediate Implementation Queue (Guided Local Runtime UX)

Authoritative goal for next slices: non-technical users can choose `Cloud` or `Local` matching without entering raw endpoint values, while power users still have an `Advanced` path.

1. LLM mode UX and progressive disclosure (`Cloud` / `Local` / `Advanced`).
   - Why: simplify setup for non-technical users and remove endpoint/networking concepts from default path.
   - Build: mode selector IA, explanatory copy, and visibility gating so raw base-URL fields appear only in `Advanced`.
   - Done when: default settings flow does not require manual endpoint entry and visibility rules are test-covered.
2. Local runtime service/binding integration (Ollama-first backend foundation is done).
   - Why: backend orchestration exists but is not yet consumable from UI flows.
   - Build: expose `internal/localruntime` state/actions through backend service APIs and bindings with explicit validation/error contracts.
   - Done when: frontend can query runtime status and trigger start/stop through stable service methods without direct process access.
3. Managed model lifecycle for `Local` mode.
   - Why: users need guided model acquisition and readiness checks.
   - Build: model selection, pull/download progress, readiness validation, and actionable error handling.
   - Done when: users can pick a recommended local model and reach `ready` state in-app without manual endpoint setup.
4. Matcher wiring for mode-based runtime selection.
   - Why: saved mode/provider settings must control active scoring path at runtime.
   - Build: provider resolver consumes mode + local-runtime settings and routes to cloud/local providers with safe fallback behavior.
   - Done when: switching modes changes scoring path without restart and is covered by integration tests.
5. OpenAI-compatible `Advanced` manual endpoint path.
   - Why: preserve expert flexibility and support existing self-hosted gateways.
   - Build: keep validated base-URL/model overrides in advanced-only flow and preserve current classified error handling.
   - Done when: advanced path works for OpenAI-compatible endpoints but is opt-in and hidden from default onboarding.
6. Token-efficiency controls.
   - Why: constrain cost and response time under continuous polling.
   - Build: compact prompt shaping, description truncation bounds, prefilter thresholds, and bounded context windows.
   - Done when: per-match token budgets are configurable and logged; scoring remains stable under large job descriptions.
7. Match-threshold settings and notifications.
   - Why: deliver actionable alerts without noisy low-signal matches.
   - Build: user-configurable threshold + native notification path for high-score transitions.
   - Done when: only matches meeting threshold trigger notifications and this behavior is test-covered.
8. Deferred UI sorting enhancement (later step, not blocking core matching).
   - Build: sort controls for posted date and match score.
   - Done when: sorting is deterministic, persisted if appropriate, and does not regress list virtualization performance.

## Phase 3: Document Tailoring

1. CV tailoring prompt and workflow.
2. Cover-letter tailoring prompt and workflow.
3. AI-detection self-check loop.
4. PDF generation for tailored outputs.
5. UI preview, download, and regenerate actions.

## Phase 4: UX and Workflow Expansion

1. First-run setup wizard.
2. Expanded dashboard with match/status views.
3. Status pipeline and Kanban flow.
4. Complete settings page (providers, prompts, custom instructions).

## Phase 5: Distribution

1. macOS `.dmg` and Windows `.msi` pipelines.
2. GitHub Actions CI.
3. End-user documentation packaging.
4. Data hygiene pass (retention/archival controls, cleanup tooling, and safety defaults for long-running local datasets).
