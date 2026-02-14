# Engineering Standards

## Delivery Workflow

1. Build incrementally: one small approved step at a time.
2. Include tests with each change.
3. Respect dependency order: data layer -> business logic -> bindings -> UI -> verification.
4. Stop for review/approval between planned steps when working from an approved backlog.
5. Update execution docs after each completed step.

## Definition of Done

A step is complete only when all are true:

- Required tests pass.
- Coverage thresholds are met.
- `go vet ./...` is clean.
- Errors are explicitly handled and wrapped with context.
- Logging is structured and secret-safe.
- Documentation status is updated.
- Commit is atomic and scoped to one approved step.

## Code Quality Rules

### Error Handling

- Do not ignore `error` values.
- Wrap errors with operation context (`fmt.Errorf("...: %w", err)`).
- Use sentinel errors and `errors.Is` for branching when needed.
- Surface friendly messages to UI; keep internals in logs.

### Logging

- Use `log/slog` structured logs.
- Use appropriate levels (`DEBUG`, `INFO`, `WARN`, `ERROR`).
- Never log API keys, credentials, or personal data.

### Package and Service Boundaries

- New backend packages go under `internal/`.
- Wails-exposed logic lives in focused root-level services.
- Keep interfaces narrow and consumer-defined.

### Frontend Contracts

- Do not expose secrets in frontend contracts.
- Failures in async mutations must propagate clearly and be test-covered.
- Accessibility baseline is required for new UI (focus-visible, keyboard behavior, ARIA semantics where applicable).

## Git and Contribution Practices

- One approved step = one commit.
- Keep commits focused and reviewable.
- Never commit secrets, DB files, coverage artifacts, or generated binaries.
- Commit prefix categories: `DB`, `SERVICE`, `API`, `UI`, `TEST`, `FIX`, `REFACTOR`, `DOCS`, `CONFIG`.

## Review Handoff Format

For each completed step provide:

1. What changed.
2. Why it changed.
3. Tests added/updated.
4. Verification commands and results.
5. Documentation updates made.

## Documentation Update Rules

- `docs/execution/status.md`: update after every completed step.
- `docs/execution/known-issues.md`: update when active bugs/risks change (keep resolved items out of this file).
- `docs/execution/roadmap.md`: update when priority or sequencing changes.
- `docs/core/*`: update only for durable product/architecture/process changes.

## Essential Commands

```bash
go test ./...
go test -coverprofile=coverage.out ./...
go vet ./...
go fmt ./...
cd frontend && npm test -- --run
cd frontend && npm run test:coverage
wails3 dev
wails3 build
```
