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
- List/detail UI surfaces match status and score; users can manually requeue recalculation.
- Default scorer is local heuristic until external providers are integrated.

## Phase 2 Immediate Implementation Queue (OpenAI-First)

1. OpenAI provider implementation.
   - Why: unlock production-grade LLM scoring while keeping current async architecture unchanged.
   - Build: `internal/llm/openai` provider with `Match` and `Validate`, deterministic parsing, timeout/error classification, and token estimate reporting.
   - Done when: matcher can score through OpenAI using configured key/model and tests cover success, malformed response, timeout, and auth failure paths.
2. OpenAI-compatible provider path for self-hosted/local models (llama/deepseek-style deployments).
   - Why: keep vendor lock-in low and support user-controlled infra.
   - Build: configurable base URL + model settings with OpenAI-compatible request/response contract.
   - Done when: same provider workflow works against non-OpenAI compatible endpoints without code fork in matcher logic.
3. Runtime provider selection and configuration wiring.
   - Why: allow users to switch providers safely without app restart assumptions.
   - Build: settings keys + service APIs for provider name/model/base URL and API-key lifecycle, then runtime selection in matcher worker.
   - Done when: switching settings changes the active provider path and invalid configuration errors are surfaced clearly to UI/logs.
4. CV parser path for matching inputs.
   - Why: improve score quality by grounding prompts on actual candidate profile rather than filter keywords alone.
   - Build: parse and cache CV text (bounded size), then include compact profile context in `MatchRequest`.
   - Done when: matching uses parsed CV context with tests for parse failure fallback behavior.
5. Token-efficiency controls.
   - Why: constrain cost and response time under continuous polling.
   - Build: compact prompt shaping, description truncation bounds, prefilter thresholds, and bounded context windows.
   - Done when: per-match token budgets are configurable and logged; scoring remains stable under large job descriptions.
6. Match-threshold settings and notifications.
   - Why: deliver actionable alerts without noisy low-signal matches.
   - Build: user-configurable threshold + native notification path for high-score transitions.
   - Done when: only matches meeting threshold trigger notifications and this behavior is test-covered.
7. Deferred UI sorting enhancement (later step, not blocking core matching).
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
