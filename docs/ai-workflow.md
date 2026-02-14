# AI Session Workflow

This workflow is model-agnostic. Use it when opening a new AI session.

## 1) Choose a Context Pack

### Feature work

- `docs/README.md`
- `docs/core/product.md`
- `docs/core/decisions.md`
- `docs/core/architecture.md`
- `docs/core/engineering-standards.md`
- `docs/core/testing-standards.md`
- `docs/execution/status.md`
- `docs/execution/roadmap.md`

### Bug fix

- `docs/README.md`
- `docs/core/decisions.md`
- `docs/core/architecture.md`
- `docs/core/engineering-standards.md`
- `docs/core/testing-standards.md`
- `docs/execution/status.md`
- `docs/execution/known-issues.md`

## 2) Session Prompt Template

Use a prompt in this structure:

```text
Goal: <single feature or bug-fix objective>
Constraints:
- Follow docs/core/engineering-standards.md
- Follow docs/core/testing-standards.md
- Keep change scoped to this objective only
- Add or update tests
- Update docs/execution/status.md and docs/execution/roadmap.md if implementation state/order changed
- Update docs/execution/known-issues.md only when active bugs/risks changed
Deliverables:
- Code changes
- Tests
- Verification command outputs summary
- Brief risk notes
```

## 3) Quality Gate Checklist

Before accepting an AI-generated change, verify:

1. Scope is narrow and aligned to one objective.
2. Tests include failure-path coverage.
3. Error handling and logging standards are met.
4. No secrets or generated artifacts were added.
5. Documentation execution files were updated if state changed.

## 4) Anti-Slop Guardrails

- Ask for one step at a time, not multi-feature bundles.
- Require exact file paths changed and why.
- Require explicit test command list run for that step.
- Reject changes that introduce new abstractions without a concrete need.
