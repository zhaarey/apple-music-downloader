#Requires -Version 5.1
<#
.SYNOPSIS
  Build Aura Audio Downloader GUI, CLI, and Windows installer.

.DESCRIPTION
  1. Builds frontend (npm)
  2. Builds aura.exe and amd.exe CLI (legacy alias)
  3. Builds AuraAudioDownloader.exe via Wails
  4. Optionally compiles Inno Setup installer

  Place third-party binaries in dist\tools\ before packaging:
    MP4Box.exe, ffmpeg.exe, ffprobe.exe, mp4decrypt.exe, yt-dlp.exe

  Use -BundleTools to require MP4Box, ffmpeg, ffprobe, and yt-dlp in dist\tools\ before packaging.
#>
param(
    [switch]$SkipFrontend,
    [switch]$SkipInstaller,
    [switch]$BundleTools
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$Dist = Join-Path $Root "dist"
$Tools = Join-Path $Dist "tools"

Set-Location $Root

function Find-Go {
    if (Get-Command go -ErrorAction SilentlyContinue) { return "go" }
    $paths = @(
        "${env:ProgramFiles}\Go\bin\go.exe",
        "${env:LOCALAPPDATA}\Programs\Go\bin\go.exe"
    )
    foreach ($p in $paths) {
        if (Test-Path $p) { return $p }
    }
    throw "Go not found. Install from https://go.dev/dl/"
}

function Find-Wails {
    if (Get-Command wails -ErrorAction SilentlyContinue) { return "wails" }
    $go = Find-Go
    $gopath = & $go env GOPATH 2>$null
    if ($gopath) {
        $w = Join-Path $gopath "bin\wails.exe"
        if (Test-Path $w) { return $w }
    }
    throw "Wails CLI not found. Run: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
}

$go = Find-Go
Write-Host "Using Go: $go"

New-Item -ItemType Directory -Force -Path $Dist, $Tools | Out-Null

if (-not $SkipFrontend) {
    Write-Host "Building frontend..."
    Push-Location (Join-Path $Root "gui\frontend")
    if (-not (Test-Path "node_modules")) { npm install }
    npm run build
    Pop-Location
}

Write-Host "Building CLI (aura.exe + amd.exe)..."
& $go build -ldflags="-s -w" -o (Join-Path $Dist "aura.exe") ./cmd/amd
if ($LASTEXITCODE -ne 0) { throw "CLI build failed" }
Copy-Item (Join-Path $Dist "aura.exe") (Join-Path $Dist "amd.exe") -Force

Write-Host "Building GUI (Wails)..."
$wails = Find-Wails
$wailsArgs = @("build", "-platform", "windows/amd64", "-clean")
if (-not $SkipFrontend) {
    $wailsArgs += "-s"
}
Push-Location (Join-Path $Root "gui")
& $wails @wailsArgs
$builtGui = @(
    (Join-Path $Root "gui\build\bin\AuraAudioDownloader.exe"),
    (Join-Path $Root "build\bin\AuraAudioDownloader.exe")
) | Where-Object { Test-Path $_ } | Select-Object -First 1
if ($builtGui) {
    Copy-Item $builtGui (Join-Path $Dist "AuraAudioDownloader.exe") -Force
}
Pop-Location
if ($LASTEXITCODE -ne 0) { throw "Wails build failed" }

$ScriptsSrc = Join-Path $Root "scripts"
$ScriptsDst = Join-Path $Dist "scripts"
if (Test-Path $ScriptsSrc) {
    New-Item -ItemType Directory -Force -Path $ScriptsDst | Out-Null
    Copy-Item (Join-Path $ScriptsSrc "sync-repair-windows.ps1") $ScriptsDst -Force -ErrorAction SilentlyContinue
}

$requiredTools = @("MP4Box.exe")
$optionalTools = @("ffmpeg.exe", "ffprobe.exe", "mp4decrypt.exe", "yt-dlp.exe")
$bundledToolNames = @("MP4Box.exe", "ffmpeg.exe", "ffprobe.exe", "yt-dlp.exe")

if ($BundleTools) {
    $missing = @()
    foreach ($t in $bundledToolNames) {
        if (-not (Test-Path (Join-Path $Tools $t))) {
            $missing += $t
        }
    }
    if ($missing.Count -gt 0) {
        throw "-BundleTools requires these files in dist\tools\: $($missing -join ', ')"
    }
}

foreach ($t in $requiredTools) {
    if (-not (Test-Path (Join-Path $Tools $t))) {
        Write-Warning "Missing dist\tools\$t - download from GPAC and copy before creating installer."
    }
}
foreach ($t in $optionalTools) {
    if (-not (Test-Path (Join-Path $Tools $t))) {
        Write-Warning "Optional missing: dist\tools\$t"
    }
}

if (-not $SkipInstaller) {
    $iscc = @(
        "${env:ProgramFiles(x86)}\Inno Setup 6\ISCC.exe",
        "${env:ProgramFiles}\Inno Setup 6\ISCC.exe"
    ) | Where-Object { Test-Path $_ } | Select-Object -First 1

    if ($iscc) {
        Write-Host "Building installer with Inno Setup..."
        & $iscc (Join-Path $Root "installer\setup.iss")
        Write-Host "Installer: $(Join-Path $Dist 'AuraAudioDownloader-Setup.exe')"
    } else {
        Write-Warning "Inno Setup not found - skipping installer. Install from https://jrsoftware.org/isinfo.php"
    }
}

Write-Host ""
Write-Host "Build complete:"
Write-Host "  GUI:  $Dist\AuraAudioDownloader.exe"
Write-Host "  CLI:  $Dist\aura.exe (amd.exe is a copy for legacy scripts)"
Write-Host "  Tools: $Tools\"
