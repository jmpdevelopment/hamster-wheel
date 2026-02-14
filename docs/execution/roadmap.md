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
2. Claude provider implementation.
3. CV parser path for matching inputs.
4. Provider-agnostic matcher logic.
5. Poll-cycle integration of matching.
6. Native notifications for high-score matches.
7. Match score surfaces in list/detail UI.
8. Match-threshold controls in settings.

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
