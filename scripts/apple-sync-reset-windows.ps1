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

function Stop-NamedProcesses([string[]]$Names, [string]$Reason) {
    $count = 0
    foreach ($name in $Names) {
        Get-Process -Name $name -ErrorAction SilentlyContinue | ForEach-Object {
            try {
                Stop-Process -Id $_.Id -Force -ErrorAction Stop
                Write-Log "Stopped $Reason process: $($_.ProcessName) (pid $($_.Id))"
                $count++
            } catch {
                Write-Log "Could not stop $($_.ProcessName): $($_.Exception.Message)"
            }
        }
    }
    return $count
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

"Aura Apple sync reset started $(Get-Date -Format o)" | Out-File -FilePath $LogPath -Encoding utf8
Write-Log 'Emulating cancel-restart: stopping Apple Music and sync agents (no cache deletion)…'

# Same broad termination as a shutdown interrupt — Apple Music + device sync stack.
$musicApps = @('AppleMusic', 'iTunes', 'iTunesHelper', 'distnoted')
$syncAgents = @('AMPDevicesAgent', 'AMPDeviceDiscoveryAgent', 'AppleMobileDeviceHelper')

$stopped = 0
$stopped += Stop-NamedProcesses -Names $musicApps -Reason 'music'
Start-Sleep -Seconds 1
$stopped += Stop-NamedProcesses -Names $syncAgents -Reason 'sync'

Get-Process -ErrorAction SilentlyContinue |
    Where-Object {
        $_.ProcessName -like 'AMPDevice*' -or
        $_.ProcessName -like 'AppleMobileDevice*'
    } |
    ForEach-Object {
        try {
            Stop-Process -Id $_.Id -Force -ErrorAction Stop
            Write-Log "Stopped related process: $($_.ProcessName) (pid $($_.Id))"
            $stopped++
        } catch {
            Write-Log "Could not stop $($_.ProcessName): $($_.Exception.Message)"
        }
    }

if ($stopped -eq 0) {
    Write-Log 'No Apple sync processes were running (already idle).'
} else {
    Write-Log ("Stopped {0} process(es)." -f $stopped)
}

Start-Sleep -Seconds 2

Write-Log 'Restarting Apple Mobile Device Service…'
$serviceOk = Restart-MobileDeviceService
if (-not $serviceOk) {
    Write-Log 'EXIT 2: service restart needs administrator'
    exit 2
}

Write-Log 'Sync reset finished — reopen Apple Music / Apple Devices and sync one album first.'
exit 0
