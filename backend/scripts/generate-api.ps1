# Regenerates internal/adapter/http/api/api.gen.go from api/openapi.yaml.
$ErrorActionPreference = 'Stop'
Push-Location (Join-Path $PSScriptRoot '..')
try {
    go generate ./internal/adapter/http/api
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    Write-Host 'Generated internal/adapter/http/api/api.gen.go'
} finally {
    Pop-Location
}
