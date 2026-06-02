# ── Stage 1: Builder ──────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /build
RUN apk add --no-cache git ca-certificates gcc musl-dev sqlite-dev

COPY app/go.mod app/go.sum ./
RUN go mod download

COPY app/ .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /orchestrator-api .

# ── Stage 2: Runtime ──────────────────────────────────────────
FROM alpine:3.19

WORKDIR /app
RUN apk add --no-cache ca-certificates docker-cli sqlite-libs

COPY --from=builder /orchestrator-api /app/orchestrator-api

EXPOSE 8080
ENTRYPOINT ["/app/orchestrator-api"]