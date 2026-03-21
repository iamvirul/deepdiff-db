# ── Stage 1: Build ────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

# Install git (needed for go modules with VCS stamping)
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

# Cache dependencies separately from source
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a statically linked binary
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -o /out/deepdiffdb \
      ./cmd/deepdiffdb

# ── Stage 2: Runtime ──────────────────────────────────────────────────────────
FROM scratch

# Pull in TLS certs and timezone data from the builder stage
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the static binary
COPY --from=builder /out/deepdiffdb /usr/local/bin/deepdiffdb

# Run as non-root (numeric UID so it works in scratch)
USER 65534:65534

ENTRYPOINT ["/usr/local/bin/deepdiffdb"]
CMD ["--help"]
