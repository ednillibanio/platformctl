$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$DevRoot = if ($env:PLATFORMCTL_DEV_ROOT) { $env:PLATFORMCTL_DEV_ROOT } else { $ProjectRoot }
$LocalGo = Join-Path $DevRoot ".tools\go\bin\go.exe"

if (Test-Path $LocalGo) {
    $GoBinary = $LocalGo
} elseif (Get-Command go -ErrorAction SilentlyContinue) {
    $GoBinary = (Get-Command go).Source
} else {
    throw "Go is not available. Run .\scripts\bootstrap-dev.ps1 first."
}

if (-not $env:GOCACHE) {
    $env:GOCACHE = Join-Path $DevRoot ".cache\go-build"
}
if (-not $env:GOPATH) {
    $env:GOPATH = Join-Path $DevRoot ".cache\gopath"
}
New-Item -ItemType Directory -Force -Path $env:GOCACHE, $env:GOPATH | Out-Null

& $GoBinary @args
exit $LASTEXITCODE
