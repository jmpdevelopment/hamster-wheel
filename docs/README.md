# Hamster Wheel Documentation

This documentation set is model-agnostic and optimized for both humans and AI agents.

## Design Goals

- Keep stable guidance separate from fast-changing execution status.
- Make it clear which files are required for each task type.
- Minimize prompt context by loading only the relevant docs.

## Read Order

1. `docs/ai-workflow.md` - session bootstrap and anti-slop guardrails.
2. `docs/core/product.md` - end goal, principles, scope.
3. `docs/core/architecture.md` - system shape, data model, service boundaries.
4. `docs/core/engineering-standards.md` - contribution workflow and quality bar.
5. `docs/core/testing-standards.md` - test and coverage expectations.
6. `docs/execution/status.md` - current state and latest completed work.
7. `docs/execution/roadmap.md` - what comes next and in what order.
8. `docs/execution/known-issues.md` - known bugs, risks, and watchlist.
9. `docs/core/decisions.md` - locked architecture decisions and rationale.

## Minimal Context Packs

Use the smallest pack that fits the task.

### Implement next feature

- `docs/core/product.md`
- `docs/core/architecture.md`
- `docs/core/engineering-standards.md`
- `docs/core/testing-standards.md`
- `docs/execution/status.md`
- `docs/execution/roadmap.md`

### Fix a bug or reliability issue

- `docs/core/architecture.md`
- `docs/core/engineering-standards.md`
- `docs/core/testing-standards.md`
- `docs/execution/status.md`
- `docs/execution/known-issues.md`

### Plan a new phase or major refactor

- `docs/core/product.md`
- `docs/core/architecture.md`
- `docs/core/decisions.md`
- `docs/execution/roadmap.md`
- `docs/execution/known-issues.md`

## Stability and Ownership

- Stable docs: `docs/core/*`
- Volatile docs: `docs/execution/*`

Update rules:

- Update `docs/execution/status.md` after each approved implementation step.
- Update `docs/execution/known-issues.md` when a bug is found/resolved.
- Update `docs/execution/roadmap.md` when sequencing changes.
- Update `docs/core/*` only when product direction, architecture, or standards change.

## Legacy Files

- Legacy working docs in `.claude/` are superseded by this `docs/` tree.
