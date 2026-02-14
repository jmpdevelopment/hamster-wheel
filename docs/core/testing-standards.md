# Testing Standards

## Test Commands

```bash
# Backend
go test ./...

# Frontend
cd frontend && npm test -- --run
```

## Coverage Commands

```bash
# Backend coverage profile and total
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -n 1

# Frontend coverage summary
cd frontend && npm run test:coverage
```

Frontend coverage uses a local custom Vitest provider:
`frontend/tools/vitest-lite-coverage-provider.mjs`

## CI-Ready Coverage Gates

```bash
# Defaults: backend >= 75, frontend lines >= 70
./scripts/check-coverage.sh

# Override thresholds
BACKEND_COVERAGE_THRESHOLD=80 FRONTEND_COVERAGE_THRESHOLD=75 ./scripts/check-coverage.sh
```

## Baseline (2026-02-14)

- Backend total statement coverage: 77.8%
- Frontend line coverage: 99.5%

## Quality Expectations for New Tests

- Keep tests adjacent to source (`*_test.go`, `*.test.ts[x]`).
- Cover success paths, edge cases, and failure paths.
- Use deterministic test inputs and explicit assertions.
- Add regression tests for each bug fix.
- Prefer unit tests first; add integration tests where boundaries are crossed.

## Coverage Policy

- Hard gate minimums: backend 75%, frontend lines 70%.
- Preferred working target: backend 80%+ on touched modules.
- Critical logic should approach full branch/path coverage where practical.
