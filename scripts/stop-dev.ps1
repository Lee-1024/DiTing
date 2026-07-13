$ErrorActionPreference = "SilentlyContinue"

$root = Resolve-Path (Join-Path $PSScriptRoot "..")
$rootPath = $root.Path
$logs = Join-Path $root "logs"
$pidFile = Join-Path $logs "dev-pids.json"
$stopped = @{}

function Stop-ProcessTree {
    param([int]$ProcessId)

    $process = Get-Process -Id $ProcessId -ErrorAction SilentlyContinue
    if ($process) {
        taskkill /PID $ProcessId /T /F | Out-Null
        $script:stopped[$ProcessId] = $true
        Write-Host "Stopped PID $ProcessId ($($process.ProcessName))"
    }
}

function Stop-DiTingProcessRemainders {
    $rootCache = Join-Path $rootPath ".cache\go-build"
    Get-Process -Name audit-server -ErrorAction SilentlyContinue | Where-Object {
        $_.Path -and $_.Path.StartsWith($rootCache, [System.StringComparison]::OrdinalIgnoreCase)
    } | ForEach-Object {
        Stop-ProcessTree -ProcessId $_.Id
    }

    netstat -ano | Select-String ":8089" | ForEach-Object {
        $parts = -split $_.Line.Trim()
        if ($parts.Length -ge 5 -and $parts[3] -eq "LISTENING") {
            $pid = [int]$parts[4]
            $process = Get-Process -Id $pid -ErrorAction SilentlyContinue
            if ($process -and $process.ProcessName -eq "audit-server") {
                Stop-ProcessTree -ProcessId $pid
            }
        }
    }
}

function Write-PortWarning {
    param([int]$Port)

    netstat -ano | Select-String ":$Port" | ForEach-Object {
        $parts = -split $_.Line.Trim()
        if ($parts.Length -ge 5 -and $parts[1] -match ":$Port$" -and $parts[3] -eq "LISTENING") {
            Write-Host "Warning: port $Port is still occupied by PID $($parts[4])."
        }
    }
}

if (Test-Path $pidFile) {
    $entries = Get-Content $pidFile -Raw | ConvertFrom-Json
    foreach ($entry in @($entries)) {
        if ($entry.Id) {
            Stop-ProcessTree -ProcessId ([int]$entry.Id)
        }
    }
    Remove-Item $pidFile -Force -ErrorAction SilentlyContinue
} else {
    Write-Host "No DiTing PID file found: $pidFile"
}

Stop-DiTingProcessRemainders
Write-PortWarning -Port 8089
Write-PortWarning -Port 5173
Write-PortWarning -Port 5174
Write-Host "DiTing development processes stopped."
