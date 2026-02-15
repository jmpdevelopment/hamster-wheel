# Installer Files

This directory contains separate installer packaging flows for Windows and macOS.

## Windows

Files:

- `scripts/installers/windows/installer.nsi`
- `scripts/installers/windows/build-installer.ps1`

Requirements:

- NSIS installed (`makensis` on `PATH`)
- Built app executable (default path: `bin\hamster-wheel.exe`)

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

## macOS

Files:

- `scripts/installers/macos/build-installers.sh`

Requirements:

- macOS host
- Built app bundle (default path: `bin/Hamster Wheel.app`)
- `pkgbuild` and `hdiutil` available

Example:

```bash
./scripts/installers/macos/build-installers.sh "bin/Hamster Wheel.app" "0.1.0"
```

Optional package signing:

```bash
PKG_SIGN_IDENTITY="Developer ID Installer: Example, Inc. (TEAMID)" \
  ./scripts/installers/macos/build-installers.sh "bin/Hamster Wheel.app" "0.1.0"
```

Output:

- `dist/installers/macos/hamster-wheel-<version>-macos-installer.pkg`
- `dist/installers/macos/hamster-wheel-<version>-macos-installer.dmg`
