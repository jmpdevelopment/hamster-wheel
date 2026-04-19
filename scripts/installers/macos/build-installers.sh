#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

APP_BUNDLE_PATH="${1:-${REPO_ROOT}/bin/hamster-wheel.app}"
VERSION="${2:-1.0.0}"
OUTPUT_DIR="${3:-${REPO_ROOT}/dist/installers/macos}"

APP_NAME="${APP_NAME:-Hamster Wheel}"
PKG_IDENTIFIER="${PKG_IDENTIFIER:-com.hamsterwheel.app}"
PKG_SIGN_IDENTITY="${PKG_SIGN_IDENTITY:-}"

if [[ ! -d "${APP_BUNDLE_PATH}" ]]; then
  echo "App bundle not found: ${APP_BUNDLE_PATH}" >&2
  echo "Build it first (for example: cd darwin && wails3 task package:universal)." >&2
  exit 1
fi

if ! command -v pkgbuild >/dev/null 2>&1; then
  echo "pkgbuild is required and only available on macOS." >&2
  exit 1
fi

if ! command -v hdiutil >/dev/null 2>&1; then
  echo "hdiutil is required and only available on macOS." >&2
  exit 1
fi

mkdir -p "${OUTPUT_DIR}"
OUTPUT_DIR="$(cd "${OUTPUT_DIR}" && pwd)"

APP_BUNDLE_NAME="$(basename "${APP_BUNDLE_PATH}")"
PKG_PATH="${OUTPUT_DIR}/hamster-wheel-${VERSION}-macos-installer.pkg"
DMG_PATH="${OUTPUT_DIR}/hamster-wheel-${VERSION}-macos-installer.dmg"

pkgbuild_args=(
  --component "${APP_BUNDLE_PATH}"
  --identifier "${PKG_IDENTIFIER}"
  --version "${VERSION}"
  --install-location "/Applications"
)

if [[ -n "${PKG_SIGN_IDENTITY}" ]]; then
  pkgbuild_args+=(--sign "${PKG_SIGN_IDENTITY}")
fi

pkgbuild "${pkgbuild_args[@]}" "${PKG_PATH}"

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/hamster-wheel-installer.XXXXXX")"
trap 'rm -rf "${TMP_DIR}"' EXIT

DMG_STAGING="${TMP_DIR}/dmg"
mkdir -p "${DMG_STAGING}"
cp -R "${APP_BUNDLE_PATH}" "${DMG_STAGING}/${APP_BUNDLE_NAME}"
ln -s "/Applications" "${DMG_STAGING}/Applications"

hdiutil create \
  -volname "${APP_NAME}" \
  -srcfolder "${DMG_STAGING}" \
  -ov \
  -format UDZO \
  "${DMG_PATH}"

echo "macOS installers created:"
echo "  PKG: ${PKG_PATH}"
echo "  DMG: ${DMG_PATH}"
