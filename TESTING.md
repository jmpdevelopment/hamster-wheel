# Testing and Coverage

## Standard Test Commands

```bash
go test ./...
cd frontend && npm test -- --run
```

## Coverage Commands

```bash
# Backend (writes coverage.out in repo root)
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -n 1

# Frontend (writes frontend/coverage/coverage-summary.json)
cd frontend && npm run test:coverage
```

Frontend coverage runs through a local custom Vitest provider at
`frontend/tools/vitest-lite-coverage-provider.mjs` so coverage reporting works
without downloading external provider packages.

## CI-Ready Coverage Checks

```bash
# Defaults: backend >= 75, frontend lines >= 70
./scripts/check-coverage.sh

# Override thresholds
BACKEND_COVERAGE_THRESHOLD=80 FRONTEND_COVERAGE_THRESHOLD=75 ./scripts/check-coverage.sh
```

## Current Baseline

- Backend total statement coverage (`go test -coverprofile=coverage.out ./...`): **77.8%**
- Frontend line coverage (`cd frontend && npm run test:coverage`): **99.5%**
