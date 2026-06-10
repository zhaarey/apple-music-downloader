#Requires -Version 5.1
param(
    [Parameter(Mandatory = $true)][string]$LogPath,
    [switch]$RestartService
)

$ErrorActionPreference = "Continue"

function Write-Log([string]$Message) {
    $line = "$(Get-Date -Format o)  $Message"
    $line | Out-File -FilePath $LogPath -Append -Encoding utf8
    Write-Output $line
}

function Stop-NamedProcesses([string[]]$Names, [string]$Reason) {
    $killed = @()
    foreach ($name in $Names) {
        Get-Process -Name $name -ErrorAction SilentlyContinue | ForEach-Object {
            try {
                Stop-Process -Id $_.Id -Force -ErrorAction Stop
                $msg = "Stopped $Reason process: $($_.ProcessName) (pid $($_.Id))"
                Write-Log $msg
                $killed += $msg
            } catch {
                Write-Log "Could not stop $($_.ProcessName): $($_.Exception.Message)"
            }
        }
    }
    return $killed
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

"Aura Apple sync unlock started $(Get-Date -Format o)" | Out-File -FilePath $LogPath -Encoding utf8

# Primary stuck-sync agents (Apple Devices app background workers).
$syncAgents = @(
    'AMPDevicesAgent'
    'AMPDeviceDiscoveryAgent'
    'AppleMobileDeviceHelper'
)
$killed = Stop-NamedProcesses -Names $syncAgents -Reason 'sync'

# Catch store/UWP variants that sometimes register under longer names.
Get-Process -ErrorAction SilentlyContinue |
    Where-Object { $_.ProcessName -like 'AMPDevice*' -or $_.ProcessName -like 'AppleMobileDevice*' } |
    ForEach-Object {
        if ($syncAgents -contains $_.ProcessName) { return }
        try {
            Stop-Process -Id $_.Id -Force -ErrorAction Stop
            $msg = "Stopped related process: $($_.ProcessName) (pid $($_.Id))"
            Write-Log $msg
            $killed += $msg
        } catch {
            Write-Log "Could not stop $($_.ProcessName): $($_.Exception.Message)"
        }
    }

if ($killed.Count -eq 0) {
    Write-Log 'No Apple sync agent processes were running (already idle).'
} else {
    Write-Log ("Released {0} process(es)." -f $killed.Count)
}

Start-Sleep -Seconds 1

if ($RestartService) {
    Write-Log 'Restarting Apple Mobile Device Service…'
    $serviceOk = Restart-MobileDeviceService
    if (-not $serviceOk) {
        Write-Log 'EXIT 2: service restart needs administrator'
        exit 2
    }
}

Write-Log 'Sync unlock finished — open Apple Devices and sync again (one album first if artwork was stale).'
exit 0
