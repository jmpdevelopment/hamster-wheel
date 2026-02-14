#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
THRESHOLD="${1:-70}"
SUMMARY_FILE="${ROOT_DIR}/frontend/coverage/coverage-summary.json"

cd "${ROOT_DIR}/frontend"

npm run test:coverage

if [[ ! -f "${SUMMARY_FILE}" ]]; then
  echo "Coverage summary not found at ${SUMMARY_FILE}"
  exit 1
fi

LINES_PCT="$(
  node -e "
    const fs = require('fs');
    const summary = JSON.parse(fs.readFileSync(process.argv[1], 'utf8'));
    const pct = Number(summary?.totals?.lines?.pct ?? 0);
    process.stdout.write(String(pct.toFixed(1)));
  " "${SUMMARY_FILE}"
)"

printf "Frontend line coverage: %s%%\n" "${LINES_PCT}"

awk -v total="${LINES_PCT}" -v threshold="${THRESHOLD}" '
BEGIN {
  if ((total + 0) < (threshold + 0)) {
    printf("Frontend line coverage %.1f%% is below threshold %.1f%%\n", total, threshold);
    exit 1;
  }
}
'

