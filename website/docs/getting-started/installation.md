---
sidebar_position: 1
---

# Installation

DeepDiff DB ships as a single static binary with no runtime dependencies.

---

## Automatic Installation (Recommended)

These methods handle PATH setup for you automatically.

### macOS — Homebrew

```bash
brew install iamvirul/deepdiff-db/deepdiff-db
```

**Upgrade:**

```bash
brew upgrade deepdiff-db
```

---

### Linux — Package Manager

Download the `.deb` or `.rpm` package from the [GitHub Releases](https://github.com/iamvirul/deepdiff-db/releases) page and double-click to install, or use the terminal:

**Ubuntu / Debian (`.deb`)**

```bash
# amd64
curl -fsSL -O https://github.com/iamvirul/deepdiff-db/releases/latest/download/deepdiff-db_linux_amd64.deb
sudo dpkg -i deepdiff-db_linux_amd64.deb

# arm64
curl -fsSL -O https://github.com/iamvirul/deepdiff-db/releases/latest/download/deepdiff-db_linux_arm64.deb
sudo dpkg -i deepdiff-db_linux_arm64.deb
```

The binary is installed to `/usr/bin/deepdiffdb` — already on your `$PATH`.

**RHEL / Fedora / Rocky Linux (`.rpm`)**

```bash
# amd64
curl -fsSL -O https://github.com/iamvirul/deepdiff-db/releases/latest/download/deepdiff-db_linux_amd64.rpm
sudo rpm -i deepdiff-db_linux_amd64.rpm

# arm64
curl -fsSL -O https://github.com/iamvirul/deepdiff-db/releases/latest/download/deepdiff-db_linux_arm64.rpm
sudo rpm -i deepdiff-db_linux_arm64.rpm
```

**Alpine Linux (`.apk`)**

```bash
curl -fsSL -O https://github.com/iamvirul/deepdiff-db/releases/latest/download/deepdiff-db_linux_amd64.apk
sudo apk add --allow-untrusted deepdiff-db_linux_amd64.apk
```

---

### Windows — Scoop

[Scoop](https://scoop.sh) installs the binary and manages `%PATH%` automatically.

```powershell
# Install Scoop if you don't have it
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
Invoke-RestMethod -Uri https://get.scoop.sh | Invoke-Expression

# Add the DeepDiff DB bucket and install
scoop bucket add deepdiff-db https://github.com/iamvirul/deepdiff-db
scoop install deepdiff-db
```

**Upgrade:**

```powershell
scoop update deepdiff-db
```

---

### Docker

```bash
docker pull ghcr.io/iamvirul/deepdiff-db:latest

docker run --rm \
  -v $(pwd)/deepdiffdb.config.yaml:/config/deepdiffdb.config.yaml:ro \
  -v $(pwd)/diff-output:/diff-output \
  ghcr.io/iamvirul/deepdiff-db:latest diff
```

See the [Docker guide](/docs/deployment/docker) for full usage details.

---

## Manual Installation

Use this if you prefer to manage the binary location yourself, or if your platform is not listed above.

### Step 1 — Download the archive

Go to the [GitHub Releases](https://github.com/iamvirul/deepdiff-db/releases) page and download the archive for your platform:

| Platform | File |
|----------|------|
| Linux x64 | `deepdiffdb_linux_amd64.tar.gz` |
| Linux ARM64 | `deepdiffdb_linux_arm64.tar.gz` |
| macOS Apple Silicon | `deepdiffdb_darwin_arm64.tar.gz` |
| macOS Intel | `deepdiffdb_darwin_amd64.tar.gz` |
| Windows x64 | `deepdiffdb_windows_amd64.zip` |

### Step 2 — Extract and place the binary

**Linux / macOS**

```bash
# Example for Linux amd64 — adjust filename for your platform
tar -xzf deepdiffdb_linux_amd64.tar.gz
sudo mv deepdiffdb /usr/local/bin/deepdiffdb
```

**Windows**

1. Extract `deepdiffdb_windows_amd64.zip`
2. Move `deepdiffdb.exe` to a folder of your choice, e.g. `C:\Program Files\deepdiffdb\`
3. Add that folder to `%PATH%`:
   - Open **Start → Search "environment variables"**
   - Click **Edit the system environment variables → Environment Variables**
   - Under **System variables**, select `Path` → **Edit → New**
   - Paste the folder path (e.g. `C:\Program Files\deepdiffdb\`)
   - Click **OK** on all dialogs, then open a new terminal

### Step 3 — Verify

```bash
deepdiffdb --version
```

Expected output:

```
DeepDiff DB v1.0.0
```

---

## Build from Source

Requires Go 1.25.8 or later.

```bash
go install github.com/iamvirul/deepdiff-db/cmd/deepdiffdb@latest
```

This places the binary in `$GOPATH/bin` (usually `~/go/bin`). Ensure that directory is on your `PATH`.

Or clone and build locally:

```bash
git clone https://github.com/iamvirul/deepdiff-db.git
cd deepdiff-db
go build -o deepdiffdb ./cmd/deepdiffdb
sudo mv deepdiffdb /usr/local/bin/deepdiffdb
```

---

## Next Step

Follow the [Quick Start](/docs/getting-started/quick-start) guide to run your first diff.
