param(
  [string]$Output = "server.exe"
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repoRoot = Split-Path -Parent $PSScriptRoot
$resourceFile = Join-Path $repoRoot "cmd/server/rsrc_windows_amd64.syso"

Push-Location $repoRoot
try {
  Write-Host "[1/3] Generando recurso de icono..."
  go generate ./cmd/server

  if (-not (Test-Path -LiteralPath $resourceFile)) {
    throw "No se genero el archivo de recurso: $resourceFile"
  }

  Write-Host "[2/3] Compilando server.exe..."
  go build -o $Output ./cmd/server

  Write-Host "[3/3] Build completado: $Output"
} finally {
  Pop-Location
}
