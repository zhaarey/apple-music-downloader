#Requires -Version 5.1
param(
    [Parameter(Mandatory = $true)][string]$LogPath
)

$ErrorActionPreference = "Continue"

function Write-Log([string]$Message) {
    $line = "$(Get-Date -Format o)  $Message"
    $line | Out-File -FilePath $LogPath -Append -Encoding utf8
    Write-Output $line
}

function Stop-MusicProcesses {
    $names = @('AppleMusic', 'iTunes', 'iTunesHelper', 'distnoted')
    foreach ($name in $names) {
        Get-Process -Name $name -ErrorAction SilentlyContinue | ForEach-Object {
            try {
                Stop-Process -Id $_.Id -Force -ErrorAction Stop
                Write-Log "Stopped process: $($_.ProcessName) (pid $($_.Id))"
            } catch {
                Write-Log "Could not stop $($_.ProcessName): $($_.Exception.Message)"
            }
        }
    }
    Start-Sleep -Seconds 2
}

function Remove-TreeSafe([string]$Path, [string]$Reason) {
    if ([string]::IsNullOrWhiteSpace($Path)) { return }
    if (-not (Test-Path -LiteralPath $Path)) {
        Write-Log "Skip (missing): $Path"
        return
    }
    $norm = ($Path -replace '\\', '/').ToLowerInvariant()
    if ($norm -match 'itunes media|apple music media|\\music\\') {
        Write-Log "Refusing (media library): $Path"
        return
    }
    try {
        Remove-Item -LiteralPath $Path -Recurse -Force -ErrorAction Stop
        Write-Log "Cleared ($Reason): $Path"
    } catch {
        Write-Log "Error ($Reason): $Path — $($_.Exception.Message)"
    }
}

function Restart-MobileDeviceService {
    $svc = Get-Service -Name 'Apple Mobile Device Service' -ErrorAction SilentlyContinue
    if (-not $svc) {
        Write-Log 'Apple Mobile Device Service not installed — skip restart'
        return $true
    }
    try {
        if ($svc.Status -eq 'Running') {
            Stop-Service -Name 'Apple Mobile Device Service' -Force -ErrorAction Stop
            Write-Log 'Stopped Apple Mobile Device Service'
        }
        Start-Service -Name 'Apple Mobile Device Service' -ErrorAction Stop
        Write-Log 'Started Apple Mobile Device Service'
        return $true
    } catch {
        Write-Log "Service restart failed: $($_.Exception.Message)"
        return $false
    }
}

"Aura Apple Music deep purge started $(Get-Date -Format o)" | Out-File -FilePath $LogPath -Encoding utf8
Write-Log 'Stopping Apple Music / iTunes processes…'
Stop-MusicProcesses

$localAppData = [Environment]::GetFolderPath('LocalApplicationData')
$appData = [Environment]::GetFolderPath('ApplicationData')
$userProfile = $env:USERPROFILE

$targets = @(
    (Join-Path $localAppData 'Apple Computer\iTunes\Artwork')
    (Join-Path $localAppData 'Apple Computer\iTunes\Artwork\Cache')
    (Join-Path $localAppData 'Apple Computer\Media\Artwork')
    (Join-Path $localAppData 'Apple Computer\Apple Music\Artwork')
    (Join-Path $localAppData 'Apple Computer\Apple Music\Artwork\Cache')
    (Join-Path $userProfile 'Music\iTunes\Album Artwork\Cache')
    (Join-Path $userProfile 'Music\iTunes\Album Artwork\Custom')
    (Join-Path $userProfile 'Music\iTunes\Album Artwork\Downloaded')
    (Join-Path $userProfile 'Music\iTunes\Album Artwork\Local')
    (Join-Path $userProfile 'Music\iTunes\Album Artwork\Store')
    (Join-Path $appData 'Apple Computer\iTunes\Album Artwork')
    (Join-Path $localAppData 'com.apple.iTunes')
    (Join-Path $localAppData 'com.apple.Music')
)

foreach ($t in $targets) {
    Remove-TreeSafe -Path $t -Reason 'known artwork cache'
}

# UWP Apple Music package caches (never the user's Music Media folder).
$packagesRoot = Join-Path $localAppData 'Packages'
if (Test-Path -LiteralPath $packagesRoot) {
    Get-ChildItem -LiteralPath $packagesRoot -Directory -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -like 'AppleInc.AppleMusic*' -or $_.Name -like 'AppleInc.iTunes*' } |
        ForEach-Object {
            foreach ($sub in @('LocalCache', 'TempState', 'AC\INetCache', 'AC\Temp')) {
                Remove-TreeSafe -Path (Join-Path $_.FullName $sub) -Reason 'UWP cache'
            }
        }
}

Write-Log 'Restarting Apple Mobile Device Service…'
$serviceOk = Restart-MobileDeviceService
if (-not $serviceOk) {
    Write-Log 'EXIT 2: service restart needs administrator'
    exit 2
}

Write-Log 'Deep purge finished — re-import albums in Apple Music; delete music on iPhone before re-syncing.'
exit 0
