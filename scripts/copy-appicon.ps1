$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$sourceIcon = Join-Path $repoRoot "assets/appicon.png"
$buildIcon = Join-Path $repoRoot "build/appicon.png"
$windowsIcon = Join-Path $repoRoot "build/windows/icon.ico"

Copy-Item -LiteralPath $sourceIcon -Destination $buildIcon -Force
Remove-Item -LiteralPath $windowsIcon -Force -ErrorAction SilentlyContinue
