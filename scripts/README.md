# Build Scripts

This directory contains scripts to build and install the latest version of `deepdiffdb` locally for development purposes, without waiting for a release.

## Quick Start

### macOS/Linux

```bash
# Build only (outputs to bin/deepdiffdb)
./scripts/build-local.sh --build-only

# Build and install to /usr/local/bin
./scripts/build-local.sh --install

# Build and install to custom location
./scripts/build-local.sh --install --install-dir ~/bin
```

### Windows (PowerShell)

```powershell
# Build only
.\scripts\build-local.ps1 -BuildOnly

# Build and install
.\scripts\build-local.ps1 -Install
```

## Features

- Builds optimized binaries with version information
- Automatically includes git commit hash in version
- Cross-platform support (macOS, Linux, Windows)
- Handles installation permissions automatically
- Validates binary after build
- Customizable installation directory

## Options

### macOS/Linux (`build-local.sh`)

| Option | Description |
|--------|-------------|
| `--install`, `-i` | Install the binary after building |
| `--build-only`, `-b` | Only build, don't install |
| `--install-dir DIR` | Custom installation directory (default: `/usr/local/bin`) |
| `--version-suffix SUFFIX` | Add suffix to version (e.g., `-dev`, `-local`) |
| `--force`, `-f` | Force overwrite existing binary |
| `--help`, `-h` | Show help message |

### Windows (`build-local.ps1`)

| Option | Description |
|--------|-------------|
| `-Install` | Install the binary after building |
| `-BuildOnly` | Only build, don't install |
| `-InstallDir DIR` | Custom installation directory |
| `-VersionSuffix SUFFIX` | Add suffix to version |
| `-Force` | Force overwrite existing binary |
| `-Help` | Show help message |

## Environment Variables

- `INSTALL_DIR`: Override default installation directory
- `GOOS`: Target operating system (e.g., `linux`, `darwin`, `windows`)
- `GOARCH`: Target architecture (e.g., `amd64`, `arm64`)

## Examples

### Build for different platforms

```bash
# Build for Linux (from macOS)
GOOS=linux GOARCH=amd64 ./scripts/build-local.sh --build-only

# Build for Windows (from macOS)
GOOS=windows GOARCH=amd64 ./scripts/build-local.sh --build-only
```

### Development workflow

```bash
# Make changes to the code...

# Build and install locally
./scripts/build-local.sh --install

# Test the new version
deepdiffdb --help
```

### Version tagging

```bash
# Build with custom version suffix
./scripts/build-local.sh --install --version-suffix "-dev"
```

## Output

The build script outputs the binary to `bin/deepdiffdb` (or `bin/deepdiffdb.exe` on Windows).

When installing, the binary is copied to the installation directory and made executable.

## Troubleshooting

### Permission denied

If you get a permission error when installing, the script will automatically try using `sudo` on macOS/Linux.

### Binary not found in PATH

Make sure the installation directory is in your PATH:

```bash
# Check if /usr/local/bin is in PATH
echo $PATH | grep /usr/local/bin

# Add to PATH if needed (add to ~/.bashrc or ~/.zshrc)
export PATH="/usr/local/bin:$PATH"
```

### Build fails

Make sure you have:
- Go installed (`go version`)
- Git repository initialized
- All dependencies downloaded (`go mod download`)

