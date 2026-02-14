#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
THRESHOLD="${1:-75}"
PROFILE_FILE="${ROOT_DIR}/coverage.out"

cd "${ROOT_DIR}"

go test -coverprofile="${PROFILE_FILE}" ./...

TOTAL="$(go tool cover -func="${PROFILE_FILE}" | awk '/^total:/ { gsub("%", "", $NF); print $NF }')"
printf "Backend total coverage: %s%%\n" "${TOTAL}"

awk -v total="${TOTAL}" -v threshold="${THRESHOLD}" '
BEGIN {
  if ((total + 0) < (threshold + 0)) {
    printf("Backend coverage %.1f%% is below threshold %.1f%%\n", total, threshold);
    exit 1;
  }
}
'

