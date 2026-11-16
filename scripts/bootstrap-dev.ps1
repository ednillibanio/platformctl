$ErrorActionPreference = "Stop"

$GoVersion = "1.26.6"
$GoChecksum = "5b6c5b556525810463b5c897b50dc7a82d6a3dc0bfaf55d990a7e9f31d6b2318"
$ProjectRoot = Split-Path -Parent $PSScriptRoot
$DevRoot = if ($env:PLATFORMCTL_DEV_ROOT) { $env:PLATFORMCTL_DEV_ROOT } else { $ProjectRoot }
$ToolsDir = Join-Path $DevRoot ".tools"
$LocalGo = Join-Path $ToolsDir "go\bin\go.exe"

$SystemGoCompatible = $false
$SystemGo = Get-Command go -ErrorAction SilentlyContinue
if ($SystemGo) {
    $Version = (& $SystemGo.Source env GOVERSION).Trim()
    if ($Version -match '^go1\.(\d+)' -and [int]$Matches[1] -ge 25) {
        $SystemGoCompatible = $true
    }
}

if (-not (Test-Path $LocalGo) -and -not $SystemGoCompatible) {
    if (-not [Environment]::Is64BitOperatingSystem) {
        throw "Only Windows amd64 is supported by this bootstrap."
    }
    $Archive = "go$GoVersion.windows-amd64.zip"
    $Download = Join-Path ([System.IO.Path]::GetTempPath()) $Archive
    Write-Host "[INFO] Installing Go $GoVersion under $ToolsDir"
    Invoke-WebRequest -Uri "https://go.dev/dl/$Archive" -OutFile $Download
    $ActualChecksum = (Get-FileHash -Algorithm SHA256 $Download).Hash.ToLowerInvariant()
    if ($ActualChecksum -ne $GoChecksum) {
        throw "Go archive checksum mismatch."
    }
    New-Item -ItemType Directory -Force -Path $ToolsDir | Out-Null
    $GoDir = Join-Path $ToolsDir "go"
    if (Test-Path $GoDir) {
        Remove-Item -Recurse -Force $GoDir
    }
    Expand-Archive -Path $Download -DestinationPath $ToolsDir
    Remove-Item -Force $Download
}

& (Join-Path $PSScriptRoot "go.ps1") version
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
& (Join-Path $PSScriptRoot "go.ps1") mod download
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Write-Host "[PASS] Development environment is ready."
