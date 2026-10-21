# Aria Scope : compile Aria-Scope.exe.
# Normalement lancé par la CI (.github/workflows/build.yml), mais marche aussi
# sur ton PC.
#
# Usage : powershell -ExecutionPolicy Bypass -File .\build.ps1 [-Version 0.3.0]
# Prérequis : Go 1.26 ou plus, Node.js LTS.

param(
    [string] $Version = '0.3.0'
)

$ErrorActionPreference = 'Stop'

$Root     = $PSScriptRoot
$Src      = Join-Path $Root 'src'
$Frontend = Join-Path $Src 'frontend'
$WinresVersion = 'v0.3.3'

function Step($msg) { Write-Host "`n==> $msg" -ForegroundColor Cyan }

function Invoke-Native {
    param([string]$Exe, [string[]]$Arguments)
    & $Exe @Arguments
    if ($LASTEXITCODE -ne 0) { throw "$Exe a échoué (code $LASTEXITCODE)" }
}

Step 'Vérification des outils'
if (-not (Get-Command 'go' -ErrorAction SilentlyContinue)) { throw 'go introuvable. Installe Go 1.26 ou plus : https://go.dev/dl/' }
if (-not (Get-Command 'npm' -ErrorAction SilentlyContinue)) { throw 'npm introuvable. Installe Node.js LTS : https://nodejs.org/' }
Write-Host ("  " + (go version))
Write-Host ("  node " + (node --version))

Step 'Interface (Svelte)'
Push-Location $Frontend
try {
    Invoke-Native 'npm' @('ci', '--no-audit', '--no-fund')
    Invoke-Native 'npm' @('run', 'build')
} finally {
    Pop-Location
}

Step 'Ressources Windows (icône, manifeste, version)'
Push-Location $Src
try {
    Invoke-Native 'go' @('run', "github.com/tc-hib/go-winres@$WinresVersion", 'simply',
        '--arch', 'amd64',
        '--manifest', 'gui',
        '--icon', 'build\icon.ico',
        '--product-name', 'Aria Scope',
        '--file-description', 'Aria Scope',
        '--product-version', $Version,
        '--file-version', $Version,
        '--copyright', '© 2026 ZedjeK · licence MIT',
        '--original-filename', 'Aria-Scope.exe',
        '--out', 'rsrc')

    Step 'Compilation'
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    Invoke-Native 'go' @('build', '-trimpath', '-ldflags', "-H windowsgui -s -w -X main.version=$Version", '-o', (Join-Path $Root 'Aria-Scope.exe'), '.')
} finally {
    Remove-Item Env:GOOS, Env:GOARCH -ErrorAction SilentlyContinue
    Pop-Location
}

Step 'Terminé'
Write-Host "  Exe prêt : $(Join-Path $Root 'Aria-Scope.exe')" -ForegroundColor Green
