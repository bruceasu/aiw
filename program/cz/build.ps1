$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$repo = Split-Path -Parent $root
$outDir = Join-Path $repo "plugins/aiw-cz"

New-Item -ItemType Directory -Force -Path $outDir | Out-Null

$exe = "cz.exe"
Push-Location $PSScriptRoot
try {
    go build -o (Join-Path $outDir $exe) .
}
finally {
    Pop-Location
}
