<#
.SYNOPSIS
    Builds EVE Buddy for Windows desktop.

.DESCRIPTION
    Automates the build process for EVE Buddy on Windows.
    Optionally runs tests, then produces either a development build or a
    release-packaged executable with embedded icon and metadata.

.PARAMETER Release
    Build a release package using `fyne package` (produces EVE_Buddy.exe
    with icon and version metadata embedded). Omit for a fast dev build.

.PARAMETER SkipTests
    Skip running `go test` before building.

.EXAMPLE
    # Fast dev build
    .\tools\build_exe.ps1

.EXAMPLE
    # Release build, no tests
    .\tools\build_exe.ps1 -Release -SkipTests

.EXAMPLE
    # Release build with tests
    .\tools\build_exe.ps1 -Release
#>

[CmdletBinding()]
param(
    [switch]$Release,
    [switch]$SkipTests
)

$ErrorActionPreference = 'Stop'

# ── Helpers ──────────────────────────────────────────────────────────────────

function Write-Step([string]$message) {
    Write-Host "`n==> $message" -ForegroundColor Cyan
}

function Write-Success([string]$message) {
    Write-Host "    $message" -ForegroundColor Green
}

function Write-Fail([string]$message) {
    Write-Host "    ERROR: $message" -ForegroundColor Red
}

# ── Prereq checks ─────────────────────────────────────────────────────────────

Write-Step "Checking prerequisites"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Fail "Go not found. Install from https://go.dev/dl/"
    exit 1
}
$goVersion = go version
Write-Success "Found $goVersion"

# Verify fyne tool is available via go tool directive
$fyneCheck = go tool fyne version 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Fail "fyne tool not available. Run: go get fyne.io/tools/cmd/fyne"
    exit 1
}
Write-Success "Found fyne tool"

# ── Change to repo root ────────────────────────────────────────────────────────

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot   = Split-Path -Parent $scriptDir
Set-Location $repoRoot
Write-Success "Working directory: $repoRoot"

# ── Tests ─────────────────────────────────────────────────────────────────────

if (-not $SkipTests) {
    Write-Step "Running tests"
    go test -test.short ./...
    if ($LASTEXITCODE -ne 0) {
        Write-Fail "Tests failed. Fix failures or use -SkipTests to skip."
        exit 1
    }
    Write-Success "All tests passed"
}

# ── Build ─────────────────────────────────────────────────────────────────────

$tags = "migrated_fynedo"

if ($Release) {
    Write-Step "Building release package (fyne package)"
    go tool fyne package --os windows --release --tags $tags
    if ($LASTEXITCODE -ne 0) {
        Write-Fail "Release build failed"
        exit 1
    }
    $output = Get-ChildItem -Path $repoRoot -Filter "*.exe" | Sort-Object LastWriteTime -Descending | Select-Object -First 1
    Write-Success "Release package ready: $($output.Name)"
} else {
    Write-Step "Building development binary (go build)"
    go build -tags $tags -o evebuddy.exe .
    if ($LASTEXITCODE -ne 0) {
        Write-Fail "Build failed"
        exit 1
    }
    Write-Success "Development binary ready: evebuddy.exe"
}

Write-Host "`nBuild complete." -ForegroundColor Green
