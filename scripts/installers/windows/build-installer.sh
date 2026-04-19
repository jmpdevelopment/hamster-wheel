#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
INSTALLER_NSI="${SCRIPT_DIR}/installer.nsi"

APP_EXE_PATH="${1:-${REPO_ROOT}/bin/hamster-wheel.exe}"
VERSION="${2:-1.0.0}"
OUTPUT_DIR="${3:-${REPO_ROOT}/dist/installers/windows}"

APP_NAME="${APP_NAME:-Hamster Wheel}"
PUBLISHER="${PUBLISHER:-Hamster Wheel}"
PRODUCT_CODE="${PRODUCT_CODE:-HamsterWheel}"
WEBVIEW2_BOOTSTRAPPER="${WEBVIEW2_BOOTSTRAPPER:-}"

if [[ ! -f "${APP_EXE_PATH}" ]]; then
  echo "Windows binary not found: ${APP_EXE_PATH}" >&2
  echo "Build it first: wails3 task windows:build" >&2
  exit 1
fi

if ! command -v makensis >/dev/null 2>&1; then
  echo "makensis not found. Install NSIS (macOS: brew install nsis)." >&2
  exit 1
fi

if [[ ! "${VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Version must use three numeric parts (e.g. 1.0.0). Received: ${VERSION}" >&2
  exit 1
fi

mkdir -p "${OUTPUT_DIR}"
OUTPUT_DIR="$(cd "${OUTPUT_DIR}" && pwd)"
OUTPUT_FILE="${OUTPUT_DIR}/hamster-wheel-${VERSION}-windows-installer.exe"
APP_EXE_NAME="$(basename "${APP_EXE_PATH}")"

makensis_args=(
  "-DAPP_NAME=${APP_NAME}"
  "-DPUBLISHER=${PUBLISHER}"
  "-DAPP_VERSION=${VERSION}"
  "-DPRODUCT_CODE=${PRODUCT_CODE}"
  "-DAPP_EXE=${APP_EXE_PATH}"
  "-DAPP_EXE_NAME=${APP_EXE_NAME}"
  "-DOUTPUT_FILE=${OUTPUT_FILE}"
)

if [[ -n "${WEBVIEW2_BOOTSTRAPPER}" ]]; then
  if [[ ! -f "${WEBVIEW2_BOOTSTRAPPER}" ]]; then
    echo "WebView2 bootstrapper not found: ${WEBVIEW2_BOOTSTRAPPER}" >&2
    exit 1
  fi
  makensis_args+=("-DWEBVIEW2_BOOTSTRAPPER=${WEBVIEW2_BOOTSTRAPPER}")
fi

makensis "${makensis_args[@]}" "${INSTALLER_NSI}"
echo "Windows installer created at: ${OUTPUT_FILE}"
