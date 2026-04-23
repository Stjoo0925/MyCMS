param()

$ErrorActionPreference = 'Stop'

$frontendRoot = $PSScriptRoot | Split-Path -Parent
$distRoot = Join-Path $frontendRoot 'dist'

if (Test-Path $distRoot) {
  Remove-Item -LiteralPath $distRoot -Recurse -Force
}

New-Item -ItemType Directory -Path $distRoot | Out-Null
Copy-Item -LiteralPath (Join-Path $frontendRoot 'index.html') -Destination $distRoot -Force
Copy-Item -LiteralPath (Join-Path $frontendRoot 'src') -Destination (Join-Path $distRoot 'src') -Recurse -Force
Copy-Item -LiteralPath (Join-Path $frontendRoot 'wailsjs') -Destination (Join-Path $distRoot 'wailsjs') -Recurse -Force
