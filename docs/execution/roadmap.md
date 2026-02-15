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
- Local runtime service/binding integration exists:
  - `SettingsService` now exposes runtime status/start/stop methods.
  - Wails bindings include local-runtime snapshot models and lifecycle method wrappers.
- Matcher wiring for mode-based runtime selection is implemented:
  - Provider resolver reads `llm_mode` dynamically per match execution.
  - `Local` mode routes to local runtime model settings; `Cloud`/`Advanced` continue through provider/base-URL settings.
- Guided LLM mode UX + progressive disclosure is implemented in Settings:
  - Users choose `Cloud`, `Local`, or `Advanced` mode.
  - Raw base-URL endpoint input is only shown in `Advanced`.
- Guided Local setup flow is partially implemented:
  - Settings local panel now orchestrates runtime status/start/stop and single-model pull for `llama3.1:8b`.
  - Local setup now includes Llama attribution + policy links and estimated model footprint messaging.
- CV path submission and matcher-context ingestion are implemented for PDF + plain-text CV files:
  - `cv_path` is persisted via `SettingsService` and configurable in Settings UI.
  - Unsupported CV formats are rejected at submission-time validation.
  - Matcher parses/caches CV text and includes compact profile context in `MatchRequest`.
  - Runtime CV parse/load issues fail open to query-only scoring.

## Phase 2 Immediate Implementation Queue (Guided Local Runtime UX)

Authoritative goal for next slices: non-technical users can choose `Cloud` or `Local` matching without entering raw endpoint values, while power users still have an `Advanced` path.

1. Managed model lifecycle for `Local` mode.
   - Why: users need guided model acquisition and readiness checks.
   - Build: preserve completed progress/timeout lifecycle hardening (streamed pull telemetry, in-progress state restore, duplicate-pull guard, and long-pull timeout behavior) and continue improving actionable failure guidance under intermittent/offline network faults.
   - Done when: users can complete guided local setup with explicit progress visibility and reach `ready` in-app without manual endpoint setup.
2. OpenAI-compatible `Advanced` manual endpoint path.
   - Why: preserve expert flexibility and support existing self-hosted gateways.
   - Build: harden advanced endpoint validation and UX affordances for custom gateways while preserving current classified error handling.
   - Done when: advanced path works for OpenAI-compatible endpoints but is opt-in and hidden from default onboarding.
3. Token-efficiency controls.
   - Why: constrain cost and response time under continuous polling.
   - Build: compact prompt shaping, description truncation bounds, prefilter thresholds, and bounded context windows.
   - Done when: per-match token budgets are configurable and logged; scoring remains stable under large job descriptions.
4. Match-threshold settings and notifications.
   - Why: deliver actionable alerts without noisy low-signal matches.
   - Build: user-configurable threshold + native notification path for high-score transitions.
   - Done when: only matches meeting threshold trigger notifications and this behavior is test-covered.
5. Deferred UI sorting enhancement (later step, not blocking core matching).
   - Build: sort controls for posted date and match score.
   - Done when: sorting is deterministic, persisted if appropriate, and does not regress list virtualization performance.

## Phase 3: Document Tailoring

1. CV tailoring prompt and workflow.
2. Cover-letter tailoring prompt and workflow.
3. AI-detection self-check loop.
4. PDF generation for tailored outputs.
5. UI preview, download, and regenerate actions.

## Phase 4: UX and Workflow Expansion

1. First-run setup wizard (complete; shared with Settings configuration flow).
2. Expanded dashboard with match/status views.
3. Status pipeline and Kanban flow.
4. Complete settings page (providers, prompts, custom instructions).

## Phase 5: Distribution

1. macOS `.dmg` and Windows `.msi` pipelines.
2. GitHub Actions CI.
3. End-user documentation packaging.
4. Data hygiene pass (retention/archival controls, cleanup tooling, and safety defaults for long-running local datasets).
