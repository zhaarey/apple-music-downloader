#Requires -Version 5.1
param(
    [Parameter(Mandatory = $true)][string]$LogPath,
    [Parameter(ValueFromRemainingArguments = $true)][string[]]$CachePath
)

$ErrorActionPreference = "Continue"
"Aura sync repair started $(Get-Date -Format o)" | Out-File -FilePath $LogPath -Encoding utf8

foreach ($p in $CachePath) {
    if ([string]::IsNullOrWhiteSpace($p)) { continue }
    if (-not (Test-Path -LiteralPath $p)) {
        "Skip (missing): $p" | Out-File -FilePath $LogPath -Append -Encoding utf8
        continue
    }
    try {
        Remove-Item -LiteralPath $p -Recurse -Force -ErrorAction Stop
        "Cleared: $p" | Out-File -FilePath $LogPath -Append -Encoding utf8
    } catch {
        "Error: $p — $($_.Exception.Message)" | Out-File -FilePath $LogPath -Append -Encoding utf8
    }
}

"Aura sync repair finished $(Get-Date -Format o)" | Out-File -FilePath $LogPath -Append -Encoding utf8
