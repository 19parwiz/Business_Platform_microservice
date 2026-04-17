# Run from repo root. Fixes empty/invalid GOPROXY on some Windows setups.
$ErrorActionPreference = "Stop"
if (-not $env:GOPROXY -or $env:GOPROXY.Trim() -eq "" -or $env:GOPROXY -match ",\s*,") {
    $env:GOPROXY = "https://proxy.golang.org,direct"
}
if (-not $env:GOSUMDB) {
    $env:GOSUMDB = "sum.golang.org"
}

$root = Split-Path -Parent $PSScriptRoot
$modules = @("user-service", "inventory-service", "order-service", "api-gateway")
foreach ($m in $modules) {
    Write-Host "`n=== go test $m ===" -ForegroundColor Cyan
    Push-Location (Join-Path $root $m)
    try {
        go test ./...
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    } finally {
        Pop-Location
    }
}
Write-Host "`nAll modules passed." -ForegroundColor Green
