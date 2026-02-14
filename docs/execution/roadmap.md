# Roadmap

## Current Phase Snapshot

- Phase 1 (Foundation): complete.
- Phase 1.5 (UX standards): complete.
- Phase 2 (LLM matching): in progress.
- Phase 3 (Document tailoring): planned.
- Phase 4 (Full UI and workflow polish): planned.
- Phase 5 (Distribution): planned.

## Ordered Backlog

Build sequence remains:

1. Data layer.
2. Business logic.
3. Wails bindings.
4. React UI.
5. Verification and documentation updates.

## Phase 2: LLM Matching

1. LLM provider interface and provider registry.
2. OpenAI provider implementation.
3. OpenAI-compatible provider path (configurable base URL/model) for self-hosted/local models (for example llama/deepseek deployments).
4. CV parser path for matching inputs.
5. Provider-agnostic matcher logic.
6. Async post-poll matching orchestration (queue + worker); polling remains ingestion-only.
7. Native notifications for high-score matches.
8. Match score and pending-state surfaces in list/detail UI.
9. Match-threshold controls in settings.
10. Job-list sort controls (posted date and match score) after core matching stability.

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
