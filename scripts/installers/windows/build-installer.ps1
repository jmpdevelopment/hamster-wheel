[CmdletBinding()]
param(
    [string]$AppExePath = "bin\hamster-wheel.exe",
    [string]$Version = "0.1.0",
    [string]$OutputDir = "dist\installers\windows",
    [string]$AppName = "Hamster Wheel",
    [string]$Publisher = "Hamster Wheel",
    [string]$ProductCode = "HamsterWheel",
    [string]$WebView2Bootstrapper = ""
)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = (Resolve-Path (Join-Path $scriptDir "..\..\..")).Path
$installerScript = Join-Path $scriptDir "installer.nsi"

if (-not (Test-Path $installerScript -PathType Leaf)) {
    throw "Installer script not found: $installerScript"
}

$exePath = if ([System.IO.Path]::IsPathRooted($AppExePath)) { $AppExePath } else { Join-Path $repoRoot $AppExePath }
if (-not (Test-Path $exePath -PathType Leaf)) {
    throw "Application executable not found: $exePath"
}
$exePath = (Resolve-Path $exePath).Path

$outputDirPath = if ([System.IO.Path]::IsPathRooted($OutputDir)) { $OutputDir } else { Join-Path $repoRoot $OutputDir }
New-Item -Path $outputDirPath -ItemType Directory -Force | Out-Null
$outputDirPath = (Resolve-Path $outputDirPath).Path

if ($Version -notmatch '^\d+\.\d+\.\d+$') {
    throw "Version must use three numeric parts (for example 0.1.0). Received: $Version"
}

$makensis = Get-Command "makensis" -ErrorAction SilentlyContinue
if (-not $makensis) {
    throw "makensis not found. Install NSIS and ensure makensis is on PATH."
}

$outputFile = Join-Path $outputDirPath ("hamster-wheel-$Version-windows-installer.exe")
$appExeName = [System.IO.Path]::GetFileName($exePath)

$makensisArgs = @(
    "/DAPP_NAME=$AppName"
    "/DPUBLISHER=$Publisher"
    "/DAPP_VERSION=$Version"
    "/DPRODUCT_CODE=$ProductCode"
    "/DAPP_EXE=$exePath"
    "/DAPP_EXE_NAME=$appExeName"
    "/DOUTPUT_FILE=$outputFile"
)

if ($WebView2Bootstrapper -ne "") {
    $webviewPath = if ([System.IO.Path]::IsPathRooted($WebView2Bootstrapper)) { $WebView2Bootstrapper } else { Join-Path $repoRoot $WebView2Bootstrapper }
    if (-not (Test-Path $webviewPath -PathType Leaf)) {
        throw "WebView2 bootstrapper not found: $webviewPath"
    }
    $webviewPath = (Resolve-Path $webviewPath).Path
    $makensisArgs += "/DWEBVIEW2_BOOTSTRAPPER=$webviewPath"
}

$makensisArgs += $installerScript

& $makensis.Path @makensisArgs
if ($LASTEXITCODE -ne 0) {
    throw "makensis failed with exit code $LASTEXITCODE"
}

Write-Host "Windows installer created at: $outputFile"
