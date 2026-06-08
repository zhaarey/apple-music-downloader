#Requires -Version 5.1
<#
.SYNOPSIS
  Build Apple Music Downloader GUI, CLI, and Windows installer.

.DESCRIPTION
  1. Builds frontend (npm)
  2. Builds amd.exe CLI
  3. Builds AppleMusicDownloader.exe via Wails
  4. Optionally compiles Inno Setup installer

  Place third-party binaries in dist\tools\ before packaging:
    MP4Box.exe, ffmpeg.exe, mp4decrypt.exe
#>
param(
    [switch]$SkipFrontend,
    [switch]$SkipInstaller
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

Write-Host "Building CLI (amd.exe)..."
& $go build -ldflags="-s -w" -o (Join-Path $Dist "amd.exe") ./cmd/amd
if ($LASTEXITCODE -ne 0) { throw "CLI build failed" }

Write-Host "Building GUI (Wails)..."
$wails = Find-Wails
Push-Location $Root
& $wails build -projectDirectory gui -platform windows/amd64 -clean
if (Test-Path (Join-Path $Root "gui\build\bin\AppleMusicDownloader.exe")) {
    Copy-Item (Join-Path $Root "gui\build\bin\AppleMusicDownloader.exe") (Join-Path $Dist "AppleMusicDownloader.exe") -Force
} elseif (Test-Path (Join-Path $Root "build\bin\AppleMusicDownloader.exe")) {
    Copy-Item (Join-Path $Root "build\bin\AppleMusicDownloader.exe") (Join-Path $Dist "AppleMusicDownloader.exe") -Force
}
Pop-Location
if ($LASTEXITCODE -ne 0) { throw "Wails build failed" }

# Verify bundled tools
$requiredTools = @("MP4Box.exe")
$optionalTools = @("ffmpeg.exe", "mp4decrypt.exe")
foreach ($t in $requiredTools) {
    if (-not (Test-Path (Join-Path $Tools $t))) {
        Write-Warning "Missing dist\tools\$t — download from GPAC and copy before creating installer."
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
        Write-Host "Installer: $(Join-Path $Dist 'AppleMusicDownloader-Setup.exe')"
    } else {
        Write-Warning "Inno Setup not found — skipping installer. Install from https://jrsoftware.org/isinfo.php"
    }
}

Write-Host ""
Write-Host "Build complete:"
Write-Host "  GUI:  $Dist\AppleMusicDownloader.exe"
Write-Host "  CLI:  $Dist\amd.exe"
Write-Host "  Tools: $Tools\"
