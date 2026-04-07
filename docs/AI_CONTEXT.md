# Hamster Wheel Documentation

This documentation set is model-agnostic and optimized for both humans and AI agents.

## Documentation Contract

- `docs/core/product.md`: what we are building and why it exists.
- `docs/core/decisions.md`: locked decisions and constraints that should not be re-litigated each session.
- `docs/core/architecture.md`: system boundaries and implementation shape.
- `docs/core/engineering-standards.md` + `docs/core/testing-standards.md`: quality bar and delivery rules.
- `docs/execution/status.md`: current implemented state.
- `docs/execution/roadmap.md`: next prioritized implementation slices.
- `docs/execution/known-issues.md`: active defects/risks only.

## Read Order

1. `docs/core/product.md` - mission, problem, solution, phase scope.
2. `docs/core/decisions.md` - accepted architectural/product decisions.
3. `docs/core/architecture.md` - runtime model, boundaries, data model.
4. `docs/core/engineering-standards.md` - definition of done and workflow.
5. `docs/core/testing-standards.md` - required test/coverage policy.
6. `docs/execution/status.md` - current baseline and guardrails to preserve.
7. `docs/execution/roadmap.md` - next execution-ready backlog.
8. `docs/execution/known-issues.md` - active bugs/risks.
9. `docs/ai-workflow.md` - session template and anti-slop controls.

## Minimal Context Packs

Use the smallest pack that fits the task.

### Implement next feature

- `docs/core/product.md`
- `docs/core/decisions.md`
- `docs/core/architecture.md`
- `docs/core/engineering-standards.md`
- `docs/core/testing-standards.md`
- `docs/execution/status.md`
- `docs/execution/roadmap.md`

### Fix a bug or reliability issue

- `docs/core/decisions.md`
- `docs/core/architecture.md`
- `docs/core/engineering-standards.md`
- `docs/core/testing-standards.md`
- `docs/execution/status.md`
- `docs/execution/known-issues.md`

### Plan a new phase or major refactor

- `docs/core/product.md`
- `docs/core/decisions.md`
- `docs/core/architecture.md`
- `docs/execution/roadmap.md`
- `docs/execution/known-issues.md`

## Stability and Ownership

- Stable docs: `docs/core/*`
- Volatile docs: `docs/execution/*`

Update rules:

- Update `docs/execution/status.md` after each approved implementation step.
- Update `docs/execution/known-issues.md` when active bugs/risks change.
- Update `docs/execution/roadmap.md` when sequencing changes.
- Update `docs/core/*` only when product direction, architecture, or standards change.
