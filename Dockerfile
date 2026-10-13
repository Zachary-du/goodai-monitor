# Stage 1: Build static Go binary
FROM golang:1.22-alpine AS builder

WORKDIR /src

# Install build dependencies
RUN apk add --no-cache ca-certificates tzdata git

# Copy source code and dependency manifests
COPY . .

# Resolve and verify all dependencies then compile single static binary
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /goodai-monitor \
    ./cmd/monitor

# Stage 2: Minimal hardened runtime image
FROM alpine:3.20

# Install runtime dependencies (TLS certificates, timezone data, wget for healthcheck)
RUN apk --no-cache add ca-certificates tzdata wget \
    && addgroup -g 10001 -S monitor \
    && adduser -u 10001 -S monitor -G monitor \
    && mkdir -p /data \
    && chown -R monitor:monitor /data

# Copy precompiled binary from builder
COPY --from=builder /goodai-monitor /goodai-monitor

# Runtime configuration
ENV PORT=8090 \
    HOST=0.0.0.0 \
    STORAGE_PATH=/data/monitor.db \
    TZ=Pacific/Auckland \
    LOG_LEVEL=info

# Default data volume for SQLite persistence
VOLUME ["/data"]

# Expose internal listening port
EXPOSE 8090

# Non-root unprivileged user
USER 10001:10001

# Healthcheck probing internal endpoint
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8090/healthz || exit 1

ENTRYPOINT ["/goodai-monitor"]
