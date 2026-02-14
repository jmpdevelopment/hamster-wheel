#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_THRESHOLD="${BACKEND_COVERAGE_THRESHOLD:-80}"
FRONTEND_THRESHOLD="${FRONTEND_COVERAGE_THRESHOLD:-70}"

"${ROOT_DIR}/scripts/check-backend-coverage.sh" "${BACKEND_THRESHOLD}"
"${ROOT_DIR}/scripts/check-frontend-coverage.sh" "${FRONTEND_THRESHOLD}"

echo "Coverage checks passed."
