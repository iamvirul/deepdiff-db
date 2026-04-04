---
sidebar_position: 1
---

# Docker

DeepDiff DB ships a minimal, scratch-based Docker image with zero native dependencies. The image is published to GitHub Container Registry on every release.

## Pull the image

```bash
# Latest stable release
docker pull ghcr.io/iamvirul/deepdiff-db:latest

# Specific version
docker pull ghcr.io/iamvirul/deepdiff-db:v1.2.0
```

## Run a diff

Mount your config file and an output directory, then pass any `deepdiffdb` subcommand:

```bash
docker run --rm \
  -v $(pwd)/deepdiffdb.config.yaml:/config/deepdiffdb.config.yaml:ro \
  -v $(pwd)/diff-output:/diff-output \
  ghcr.io/iamvirul/deepdiff-db:latest \
  diff --config /config/deepdiffdb.config.yaml
```

## Docker Compose

The repository ships a `docker-compose.example.yml` that spins up two MySQL instances (prod + dev) alongside the deepdiffdb container:

```bash
# Start databases and wait for health checks
docker compose -f docker-compose.example.yml up -d db-prod db-dev

# Run a diff
docker compose -f docker-compose.example.yml run --rm deepdiffdb diff

# Generate a migration pack
docker compose -f docker-compose.example.yml run --rm deepdiffdb gen-pack

# Tear down
docker compose -f docker-compose.example.yml down
```

Copy and adapt `docker-compose.example.yml` to point at your real databases.

## Build from source

```bash
docker build \
  --build-arg VERSION=v1.2.0 \
  -t deepdiff-db:v1.2.0 \
  .
```

The multi-stage build compiles a statically linked binary with `CGO_ENABLED=0`, then copies it into a `scratch` image alongside TLS certificates. The final image is typically **< 15 MB**.

## Environment variables

| Variable | Description |
|----------|-------------|
| `DEEPDIFFDB_CONFIG` | Path to config file inside the container |

## Image tags

| Tag | Description |
|-----|-------------|
| `latest` | Most recent stable release |
| `v1.2.0` | Specific version |
| `v1.2.0-amd64` | Architecture-specific manifest |
| `v1.2.0-arm64` | Architecture-specific manifest |

Multi-arch manifests cover `linux/amd64` and `linux/arm64`.
