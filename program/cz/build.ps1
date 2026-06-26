$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$repo = Split-Path -Parent $root
$outDir = Join-Path $repo "plugins/aiw-cz"

New-Item -ItemType Directory -Force -Path $outDir | Out-Null

Push-Location $PSScriptRoot
try {
    gbuild windows
    Move-Item -force bin\cz-windows-amd64.exe (Join-Path $outDir "cz.exe")

    gbuild linux
    Move-Item -force bin\cz-linux-amd64 (Join-Path $outDir "cz")
}
finally {
    Pop-Location
}
