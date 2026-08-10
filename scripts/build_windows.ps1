#Requires -Version 5.1
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$RootDir  = Split-Path -Parent $PSScriptRoot
$AppName  = 'Icon Creator'
$Version  = '1.3.10'
$BuildDir = Join-Path $RootDir 'build'
$DistDir  = Join-Path $RootDir 'dist'

$WailsBin = if ($env:WAILS_BIN) { $env:WAILS_BIN } else { 'wails' }

if (-not (Get-Command $WailsBin -ErrorAction SilentlyContinue)) {
    Write-Error "wails is required but was not found. Install it with: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
}

if (-not (Test-Path (Join-Path $BuildDir 'windows\icon.ico'))) {
    Write-Error "Missing build\windows\icon.ico"
}

if (Test-Path $DistDir) { Remove-Item -Recurse -Force $DistDir }
New-Item -ItemType Directory -Force $DistDir | Out-Null

Push-Location $RootDir
try {
    & $WailsBin build -clean -platform windows/amd64 -trimpath -ldflags='-s -w' -o $AppName
    if ($LASTEXITCODE -ne 0) { throw "wails build failed with exit code $LASTEXITCODE" }
} finally {
    Pop-Location
}

$ExeFrom = $null
foreach ($candidate in @("$AppName.exe", "$AppName", 'IconCreator.exe', 'IconCreator')) {
    $p = Join-Path $BuildDir "bin\$candidate"
    if (Test-Path $p) { $ExeFrom = $p; break }
}
if (-not $ExeFrom) {
    Write-Error "Wails did not produce the expected executable in build\bin\"
}

$ExeDest = Join-Path $DistDir "Icon-Creator-$Version-Windows-amd64.exe"
Copy-Item $ExeFrom $ExeDest

Write-Output $ExeDest
