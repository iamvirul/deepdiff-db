# Build and install script for local development (PowerShell)
# This script builds the latest version from source and installs it locally
# without requiring a release

param(
    [switch]$Install,
    [switch]$BuildOnly,
    [string]$InstallDir = "",
    [string]$VersionSuffix = "",
    [switch]$Force,
    [switch]$Help
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir
$BinaryName = "deepdiffdb"
$MainPackage = "./cmd/deepdiffdb"

# Default installation location
if ([string]::IsNullOrEmpty($InstallDir)) {
    $InstallDir = "$env:LOCALAPPDATA\Programs\deepdiffdb"
    if (-not (Test-Path $InstallDir)) {
        $InstallDir = "$env:ProgramFiles\deepdiffdb"
    }
}

function Write-Info {
    param([string]$Message)
    Write-Host "ℹ $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "⚠ $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "✗ $Message" -ForegroundColor Red
}

function Write-Success {
    param([string]$Message)
    Write-Host "✓ $Message" -ForegroundColor Green
}

if ($Help) {
    Write-Host @"
Build and install script for deepdiffdb

Usage: .\build-local.ps1 [options]

Options:
    -Install              Install the binary after building (default: false)
    -BuildOnly            Only build, don't install (default: false)
    -InstallDir DIR       Installation directory (default: %LOCALAPPDATA%\Programs\deepdiffdb)
    -VersionSuffix SUFFIX Add suffix to version (e.g., "-dev", "-local")
    -Force                Force overwrite existing binary
    -Help                 Show this help message

Examples:
    # Build only
    .\build-local.ps1 -BuildOnly

    # Build and install to default location
    .\build-local.ps1 -Install

    # Build and install to custom location
    .\build-local.ps1 -Install -InstallDir "C:\Tools\deepdiffdb"

    # Build with version suffix
    .\build-local.ps1 -Install -VersionSuffix "-dev"

"@
    exit 0
}

# Change to project root
Set-Location $ProjectRoot

# Check if Go is installed
try {
    $goVersion = go version
    Write-Info "Go version: $goVersion"
} catch {
    Write-Error "Go is not installed. Please install Go first."
    exit 1
}

# Determine binary name
$BinaryFile = "$BinaryName.exe"
$BuildDir = Join-Path $ProjectRoot "bin"
$BinaryPath = Join-Path $BuildDir $BinaryFile

# Create build directory
New-Item -ItemType Directory -Force -Path $BuildDir | Out-Null

# Build flags
$BuildFlags = @("-ldflags=-s -w")

# Add version information if git is available
if (Get-Command git -ErrorAction SilentlyContinue) {
    try {
        $gitCommit = (git rev-parse --short HEAD 2>$null)
        $gitBranch = (git rev-parse --abbrev-ref HEAD 2>$null)
        $buildTime = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ")
        
        $versionInfo = "dev-$gitCommit"
        if (-not [string]::IsNullOrEmpty($VersionSuffix)) {
            $versionInfo = "$versionInfo$VersionSuffix"
        }
        
        $BuildFlags += "-X", "main.version=$versionInfo"
        $BuildFlags += "-X", "main.commit=$gitCommit"
        $BuildFlags += "-X", "main.branch=$gitBranch"
        $BuildFlags += "-X", "main.buildTime=$buildTime"
    } catch {
        Write-Warn "Could not get git information"
    }
}

# Build the binary
Write-Info "Building $BinaryName..."
Write-Info "Build directory: $BuildDir"
Write-Info "Output: $BinaryPath"

$env:GOOS = if ($env:GOOS) { $env:GOOS } else { "windows" }
$env:GOARCH = if ($env:GOARCH) { $env:GOARCH } else { "amd64" }

try {
    & go build $BuildFlags -o $BinaryPath $MainPackage
    Write-Success "Build successful!"
    
    if (Test-Path $BinaryPath) {
        $binarySize = (Get-Item $BinaryPath).Length / 1KB
        Write-Info "Binary size: $([math]::Round($binarySize, 2)) KB"
        Write-Info "Binary location: $BinaryPath"
    }
} catch {
    Write-Error "Build failed: $_"
    exit 1
}

# Install if requested
if ($Install -and -not $BuildOnly) {
    $InstallPath = Join-Path $InstallDir $BinaryFile
    
    # Create install directory if it doesn't exist
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    }
    
    if (Test-Path $InstallPath) {
        if (-not $Force) {
            $response = Read-Host "Binary already exists at $InstallPath. Overwrite? (y/N)"
            if ($response -ne "y" -and $response -ne "Y") {
                Write-Info "Installation cancelled"
                exit 0
            }
        }
    }
    
    try {
        Copy-Item $BinaryPath $InstallPath -Force
        Write-Success "Installed to $InstallPath"
        Write-Info "Add $InstallDir to your PATH to use '$BinaryName' command"
        Write-Info "Or run it directly: $InstallPath"
    } catch {
        Write-Error "Failed to install: $_"
        exit 1
    }
} elseif ($BuildOnly) {
    Write-Info "Build only mode - binary not installed"
    Write-Info "To install, run: .\build-local.ps1 -Install"
}

Write-Success "Done!"

