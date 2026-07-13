param(
    [string]$Config = ".\backend\configs\config.yaml",
    [int]$WebPort = 5173,
    [switch]$SkipFrontend,
    [switch]$SkipCollector
)

$ErrorActionPreference = "Stop"

$root = Resolve-Path (Join-Path $PSScriptRoot "..")
$backend = Join-Path $root "backend"
$frontend = Join-Path $root "frontend"
$logs = Join-Path $root "logs"
$pidFile = Join-Path $logs "dev-pids.json"
$configPath = Resolve-Path (Join-Path $root $Config)

New-Item -ItemType Directory -Force -Path $logs | Out-Null
$startedProcesses = @()

$env:GOCACHE = Join-Path $root ".cache\go-build"
$env:GOTELEMETRY = "off"
$env:GOENV = Join-Path $root ".cache\goenv"

function Test-PortInUse {
    param([int]$Port)

    $connections = netstat -ano | Select-String ":$Port"
    foreach ($connection in $connections) {
        $parts = -split $connection.Line.Trim()
        if ($parts.Length -ge 5 -and $parts[1] -match ":$Port$" -and $parts[3] -eq "LISTENING") {
            return $true
        }
    }
    return $false
}

if (Test-PortInUse -Port 8089) {
    throw "API port 8089 is already in use. Run .\scripts\stop-dev.ps1, or stop the process occupying 8089 before starting DiTing."
}

if (-not $SkipFrontend -and (Test-PortInUse -Port $WebPort)) {
    throw "Web port $WebPort is already in use. Stop the process occupying $WebPort, or start with another -WebPort value."
}

function Start-DiTingProcess {
    param(
        [string]$Name,
        [string]$FilePath,
        [string]$Arguments,
        [string]$WorkingDirectory,
        [string]$LogName
    )

    $stdout = Join-Path $logs "$LogName.out.log"
    $stderr = Join-Path $logs "$LogName.err.log"

    Write-Host "Starting $Name..."
    $process = Start-Process `
        -FilePath $FilePath `
        -ArgumentList $Arguments `
        -WorkingDirectory $WorkingDirectory `
        -RedirectStandardOutput $stdout `
        -RedirectStandardError $stderr `
        -WindowStyle Hidden `
        -PassThru

    $script:startedProcesses += [PSCustomObject]@{
        Name = $Name
        Id = $process.Id
        FilePath = $FilePath
        StartedAt = (Get-Date).ToString("o")
    }
}

function Resolve-CommandPath {
    param([string]$Command)

    $resolved = Get-Command $Command -ErrorAction Stop
    return $resolved.Source
}

$goCommand = Resolve-CommandPath "go"
$npmCommand = Resolve-CommandPath "npm.cmd"

Start-DiTingProcess `
    -Name "DiTing API" `
    -FilePath $goCommand `
    -Arguments "run ./cmd/audit-server api --config `"$configPath`"" `
    -WorkingDirectory $backend `
    -LogName "api"

if (-not $SkipCollector) {
    Start-DiTingProcess `
        -Name "DiTing Collector" `
        -FilePath $goCommand `
        -Arguments "run ./cmd/audit-server collector --config `"$configPath`"" `
        -WorkingDirectory $backend `
        -LogName "collector"
}

if (-not $SkipFrontend) {
    $webArguments = "run dev -- --port $WebPort --strictPort"
    Start-DiTingProcess `
        -Name "DiTing Web" `
        -FilePath $npmCommand `
        -Arguments $webArguments `
        -WorkingDirectory $frontend `
        -LogName "web"
}

$startedProcesses | ConvertTo-Json -Depth 3 | Set-Content -Path $pidFile -Encoding UTF8

Write-Host ""
Write-Host "DiTing development processes started."
Write-Host "API health: http://127.0.0.1:8089/healthz"
if (-not $SkipFrontend) {
    Write-Host "Web:        http://127.0.0.1:$WebPort"
}
Write-Host "Logs:       $logs (*.out.log / *.err.log)"
Write-Host "PID file:   $pidFile"
Write-Host ""
Write-Host "Stop processes with:"
Write-Host "  .\scripts\stop-dev.ps1"
