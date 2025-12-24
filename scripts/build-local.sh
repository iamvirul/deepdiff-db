#!/bin/bash

# Build and install script for local development
# This script builds the latest version from source and installs it locally
# without requiring a release

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BINARY_NAME="deepdiffdb"
MAIN_PACKAGE="./cmd/deepdiffdb"

# Default installation locations
case "$(uname -s)" in
    Darwin)
        INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
        ;;
    Linux)
        INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
        ;;
    MINGW*|MSYS*|CYGWIN*)
        INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
        ;;
    *)
        INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
        ;;
esac

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
print_info() {
    echo -e "${GREEN}ℹ${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

# Parse arguments
INSTALL=false
BUILD_ONLY=false
VERSION_SUFFIX=""
FORCE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --install|-i)
            INSTALL=true
            shift
            ;;
        --build-only|-b)
            BUILD_ONLY=true
            shift
            ;;
        --install-dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        --version-suffix)
            VERSION_SUFFIX="$2"
            shift 2
            ;;
        --force|-f)
            FORCE=true
            shift
            ;;
        --help|-h)
            cat << EOF
Build and install script for deepdiffdb

Usage: $0 [options]

Options:
    --install, -i              Install the binary after building (default: false)
    --build-only, -b           Only build, don't install (default: false)
    --install-dir DIR          Installation directory (default: /usr/local/bin)
    --version-suffix SUFFIX    Add suffix to version (e.g., "-dev", "-local")
    --force, -f                Force overwrite existing binary
    --help, -h                 Show this help message

Examples:
    # Build only
    $0 --build-only

    # Build and install to default location
    $0 --install

    # Build and install to custom location
    $0 --install --install-dir ~/bin

    # Build with version suffix
    $0 --install --version-suffix "-dev"

Environment variables:
    INSTALL_DIR                Installation directory (overridden by --install-dir)
    GOOS                       Target OS (e.g., linux, darwin, windows)
    GOARCH                     Target architecture (e.g., amd64, arm64)

EOF
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Change to project root
cd "$PROJECT_ROOT"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go first."
    exit 1
fi

print_info "Go version: $(go version)"

# Determine binary name based on OS
GOOS="${GOOS:-$(go env GOOS)}"
GOARCH="${GOARCH:-$(go env GOARCH)}"

if [ "$GOOS" = "windows" ]; then
    BINARY_FILE="${BINARY_NAME}.exe"
else
    BINARY_FILE="${BINARY_NAME}"
fi

BUILD_DIR="$PROJECT_ROOT/bin"
BINARY_PATH="$BUILD_DIR/$BINARY_FILE"

# Create build directory
mkdir -p "$BUILD_DIR"

# Build flags - start with basic optimization flags
LDFLAGS="-s -w"

# Determine version information
if command -v git &> /dev/null && git rev-parse --git-dir > /dev/null 2>&1; then
    GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
    BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Get the latest git tag
    LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")

    # Parse version components (e.g., v0.2.0 -> 0.2.0)
    VERSION_NUM="${LATEST_TAG#v}"

    # Split version into major.minor.patch
    IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION_NUM"

    # Determine next version based on branch
    if [ "$GIT_BRANCH" = "main" ] || [ "$GIT_BRANCH" = "master" ]; then
        # On main branch, increment patch version
        NEXT_PATCH=$((PATCH + 1))
        NEXT_VERSION="v${MAJOR}.${MINOR}.${NEXT_PATCH}"
    else
        # On feature branch, increment minor version
        NEXT_MINOR=$((MINOR + 1))
        NEXT_VERSION="v${MAJOR}.${NEXT_MINOR}.0"
    fi

    # Check if we're on a tagged commit
    if git describe --exact-match --tags HEAD >/dev/null 2>&1; then
        # On a tagged commit - use the tag as version
        VERSION_INFO="$LATEST_TAG"
    else
        # Not on a tagged commit - use next version with pre-release info
        # Format: v0.3.0-dev.20231224T120000Z.abc1234
        TIMESTAMP=$(date -u +"%Y%m%dT%H%M%SZ")
        VERSION_INFO="${NEXT_VERSION}-dev.${TIMESTAMP}.${GIT_COMMIT}"

        # Add custom suffix if provided
        if [ -n "$VERSION_SUFFIX" ]; then
            VERSION_INFO="${VERSION_INFO}${VERSION_SUFFIX}"
        fi
    fi

    print_info "Version: $VERSION_INFO"
    print_info "Commit:  $GIT_COMMIT"
    print_info "Branch:  $GIT_BRANCH"

    # Set ldflags with version information
    LDFLAGS="${LDFLAGS} -X main.version=${VERSION_INFO} -X main.commit=${GIT_COMMIT} -X main.branch=${GIT_BRANCH} -X main.buildTime=${BUILD_TIME}"
else
    print_warn "Git not available - building without version information"
    VERSION_INFO="dev"
    LDFLAGS="${LDFLAGS} -X main.version=${VERSION_INFO}"
fi

# Build the binary
print_info "Building ${BINARY_NAME} for ${GOOS}/${GOARCH}..."
print_info "Build directory: $BUILD_DIR"
print_info "Output: $BINARY_PATH"

if GOOS="$GOOS" GOARCH="$GOARCH" go build -ldflags "$LDFLAGS" -o "$BINARY_PATH" "$MAIN_PACKAGE"; then
    print_success "Build successful!"
    
    # Show binary info
    if [ -f "$BINARY_PATH" ]; then
        BINARY_SIZE=$(du -h "$BINARY_PATH" | cut -f1)
        print_info "Binary size: $BINARY_SIZE"
        print_info "Binary location: $BINARY_PATH"
        
        # Test the binary
        if "$BINARY_PATH" --help &> /dev/null || "$BINARY_PATH" help &> /dev/null; then
            print_success "Binary is working correctly"
        else
            print_warn "Binary built but help command failed (this might be normal)"
        fi
    fi
else
    print_error "Build failed!"
    exit 1
fi

# Install if requested
if [ "$INSTALL" = true ] && [ "$BUILD_ONLY" = false ]; then
    # Check if binary already exists in install directory
    INSTALL_PATH="$INSTALL_DIR/$BINARY_FILE"
    
    if [ -f "$INSTALL_PATH" ] && [ "$FORCE" = false ]; then
        print_warn "Binary already exists at $INSTALL_PATH"
        read -p "Overwrite? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "Installation cancelled"
            exit 0
        fi
    fi
    
    # Check if install directory is writable
    if [ ! -w "$INSTALL_DIR" ]; then
        print_warn "Install directory $INSTALL_DIR is not writable"
        print_info "Attempting to install with sudo..."
        
        if sudo cp "$BINARY_PATH" "$INSTALL_PATH"; then
            sudo chmod +x "$INSTALL_PATH"
            print_success "Installed to $INSTALL_PATH (with sudo)"
        else
            print_error "Failed to install with sudo"
            exit 1
        fi
    else
        # Install without sudo
        cp "$BINARY_PATH" "$INSTALL_PATH"
        chmod +x "$INSTALL_PATH"
        print_success "Installed to $INSTALL_PATH"
    fi
    
    # Verify installation
    if command -v "$BINARY_NAME" &> /dev/null; then
        INSTALLED_VERSION=$("$BINARY_NAME" --version 2>/dev/null || echo "unknown")
        print_success "Installation verified"
        print_info "Run '${BINARY_NAME} --help' to get started"
    else
        print_warn "Binary installed but not found in PATH"
        print_info "Make sure $INSTALL_DIR is in your PATH"
        print_info "You can run it directly: $INSTALL_PATH"
    fi
elif [ "$BUILD_ONLY" = true ]; then
    print_info "Build only mode - binary not installed"
    print_info "To install, run: $0 --install"
    print_info "Or manually: cp $BINARY_PATH $INSTALL_DIR/"
fi

print_success "Done!"

