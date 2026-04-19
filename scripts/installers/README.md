# Installer Files

Packaging flows for Windows and macOS distributables.

App metadata (name, identifier, version, copyright) is sourced from [`build/config.yml`](../../build/config.yml). After editing it, regenerate the embedded assets:

```bash
wails3 task common:update:build-assets
```

## Windows

Files:

- [`installer.nsi`](windows/installer.nsi) — NSIS script (portable, used by both wrappers)
- [`build-installer.ps1`](windows/build-installer.ps1) — Windows host
- [`build-installer.sh`](windows/build-installer.sh) — macOS / Linux host

Requirements:

- NSIS (`makensis` on `PATH`)
  - macOS: `brew install nsis`
  - Windows: install from https://nsis.sourceforge.io
- Built `hamster-wheel.exe` (default path: `bin/hamster-wheel.exe`)

### Build from macOS (cross-compile, no Docker)

```bash
wails3 task windows:build                       # → bin/hamster-wheel.exe
scripts/installers/windows/build-installer.sh   # → dist/installers/windows/...
```

Override defaults:

```bash
scripts/installers/windows/build-installer.sh \
  bin/hamster-wheel.exe 1.0.0 dist/installers/windows
```

### Build from Windows

```powershell
wails3 task windows:build
pwsh -File scripts/installers/windows/build-installer.ps1 -Version 1.0.0
```

Optional WebView2 bundling (either host):

```bash
WEBVIEW2_BOOTSTRAPPER=path/to/MicrosoftEdgeWebview2Setup.exe \
  scripts/installers/windows/build-installer.sh
```

```powershell
pwsh -File scripts/installers/windows/build-installer.ps1 `
  -WebView2Bootstrapper "path\to\MicrosoftEdgeWebview2Setup.exe"
```

Output:

- `dist/installers/windows/hamster-wheel-<version>-windows-installer.exe`

The installer is unsigned. Windows SmartScreen will warn on first run; users click "More info → Run anyway."

## macOS

Files:

- [`build-installers.sh`](macos/build-installers.sh)

Requirements:

- macOS host (uses `pkgbuild` and `hdiutil`)
- Built `.app` bundle (default path: `bin/hamster-wheel.app`)

Build:

```bash
wails3 task package:universal                            # → bin/hamster-wheel.app (arm64+amd64)
scripts/installers/macos/build-installers.sh             # → dist/installers/macos/...
```

Override defaults:

```bash
scripts/installers/macos/build-installers.sh \
  bin/hamster-wheel.app 1.0.0 dist/installers/macos
```

Optional Developer ID signing:

```bash
PKG_SIGN_IDENTITY="Developer ID Installer: Example, Inc. (TEAMID)" \
  scripts/installers/macos/build-installers.sh
```

Output:

- `dist/installers/macos/hamster-wheel-<version>-macos-installer.pkg`
- `dist/installers/macos/hamster-wheel-<version>-macos-installer.dmg`

The `.app` is ad-hoc signed. Gatekeeper will warn on first run; users right-click → Open, or run `xattr -dr com.apple.quarantine /Applications/Hamster\ Wheel.app`.
