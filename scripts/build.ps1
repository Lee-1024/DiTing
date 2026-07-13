param(
    [string]$Output = ".\backend\bin\audit-server.exe"
)

$ErrorActionPreference = "Stop"

$root = Resolve-Path (Join-Path $PSScriptRoot "..")
$backend = Join-Path $root "backend"
$frontend = Join-Path $root "frontend"
$outputPath = Join-Path $root $Output
$outputDir = Split-Path $outputPath

New-Item -ItemType Directory -Force -Path $outputDir | Out-Null

$env:GOCACHE = Join-Path $root ".cache\go-build"
$env:GOTELEMETRY = "off"
$env:GOENV = Join-Path $root ".cache\goenv"

Write-Host "Running backend tests..."
Push-Location $backend
go test ./...

Write-Host "Building backend binary..."
go build -o $outputPath ./cmd/audit-server
Pop-Location

Write-Host "Building frontend..."
Push-Location $frontend
npm run build
Pop-Location

Write-Host ""
Write-Host "Build complete."
Write-Host "Backend:  $outputPath"
Write-Host "Frontend: $(Join-Path $frontend 'dist')"
