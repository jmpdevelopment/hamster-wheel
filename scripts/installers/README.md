# Installer Files

This directory contains separate installer packaging flows for Windows and macOS.

## Important (Wails v3)

`wails3 build` does not use the old `-platform` flag.
Use platform task files (`darwin/Taskfile.yml`, `windows/Taskfile.yml`) for target-specific builds.

If these task files do not exist yet, generate build assets once from the repo root:

```bash
wails3 generate build-assets \
  -name "hamster-wheel" \
  -binaryname "hamster-wheel" \
  -productcompany "Hamster Wheel" \
  -productname "Hamster Wheel" \
  -productidentifier "com.hamsterwheel.app" \
  -productdescription "Job search monitoring and application assistant" \
  -productcopyright "(c) 2026, Hamster Wheel" \
  -productcomments "Automated job search monitoring with LLM-powered matching" \
  -productversion "0.1.0"
```

## Windows

Files:

- `scripts/installers/windows/installer.nsi`
- `scripts/installers/windows/build-installer.ps1`

Requirements:

- NSIS installed (`makensis` on `PATH`)
- Built app executable (default path: `bin\hamster-wheel.exe`)

Build executable using Wails v3 tasks:

```powershell
cd windows
wails3 task build ARCH=amd64
cd ..
```

Example:

```powershell
pwsh -File scripts/installers/windows/build-installer.ps1 `
  -AppExePath "bin\hamster-wheel.exe" `
  -Version "0.1.0"
```

Optional WebView2 bundling:

```powershell
pwsh -File scripts/installers/windows/build-installer.ps1 `
  -AppExePath "bin\hamster-wheel.exe" `
  -Version "0.1.0" `
  -WebView2Bootstrapper "build\windows\installer\tmp\MicrosoftEdgeWebview2Setup.exe"
```

Output:

- `dist\installers\windows\hamster-wheel-<version>-windows-installer.exe`

Alternative (Wails v3 native packagers):

```powershell
cd windows
wails3 task create:nsis:installer ARCH=amd64
wails3 task create:msix:package ARCH=amd64
cd ..
```

Outputs:

- `bin\hamster-wheel-amd64-installer.exe` (NSIS `.exe`)
- `bin\hamster-wheel-amd64.msix` (MSIX package)

## macOS

Files:

- `scripts/installers/macos/build-installers.sh`

Requirements:

- macOS host
- Built app bundle (default path: `bin/hamster-wheel.app`)
- `pkgbuild` and `hdiutil` available

Build app bundle using Wails v3 tasks:

```bash
cd darwin
wails3 task package:universal
cd ..
```

Example:

```bash
./scripts/installers/macos/build-installers.sh "bin/hamster-wheel.app" "0.1.0"
```

Optional package signing:

```bash
PKG_SIGN_IDENTITY="Developer ID Installer: Example, Inc. (TEAMID)" \
  ./scripts/installers/macos/build-installers.sh "bin/hamster-wheel.app" "0.1.0"
```

Output:

- `dist/installers/macos/hamster-wheel-<version>-macos-installer.pkg`
- `dist/installers/macos/hamster-wheel-<version>-macos-installer.dmg`
