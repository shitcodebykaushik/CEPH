# ── Stage 1: Builder ──────────────────────────────────────────
FROM golang:1.25-bookworm AS builder

WORKDIR /build

# Install C bindings development headers for RADOS and RBD
RUN apt-get update && apt-get install -y --no-install-recommends \
    librados-dev \
    librbd-dev \
    gcc \
    libc6-dev \
  && rm -rf /var/lib/apt/lists/*

# Cache Go dependencies matrix
COPY app/go.mod app/go.sum ./
RUN go mod download

# Compile statically linked CGO bindings
COPY app/ .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /orchestrator-api .

# ── Stage 2: Runtime ──────────────────────────────────────────
FROM debian:bookworm-slim AS runtime

WORKDIR /app

# Crucial: Install runtime system shared objects for Ceph + SQLite
RUN apt-get update && apt-get install -y --no-install-recommends \
    ceph-common \
    librados2 \
    librbd1 \
    rbd-nbd \
    libsqlite3-0 \
    ca-certificates \
    e2fsprogs \
  && rm -rf /var/lib/apt/lists/*

# Pull the binary from stage 
COPY --from=builder /orchestrator-api /app/orchestrator-api

USER root

EXPOSE 8080
ENTRYPOINT ["/app/orchestrator-api"]