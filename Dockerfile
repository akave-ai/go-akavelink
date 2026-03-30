# Build stage (multi-platform: set TARGETOS/TARGETARCH when building)
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build binary for target platform
ARG TARGETOS
ARG TARGETARCH
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-w -s" -o /akavelink ./cmd/server

# Runtime stage (omit --platform so Docker uses target platform by default; avoids redundant warning)
FROM alpine:3.20

RUN apk --no-cache add ca-certificates
WORKDIR /app

COPY --from=builder /akavelink .

EXPOSE 8080

# Required at runtime: AKAVE_PRIVATE_KEY, AKAVE_NODE_ADDRESS
# Optional: PORT, LOG_LEVEL, LOG_FORMAT, CORS_ORIGINS, AKAVE_MAX_CONCURRENCY, AKAVE_BLOCK_PART_SIZE, READ_TIMEOUT, WRITE_TIMEOUT
ENV PORT=8080

# Health check (server exposes GET /health)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O /dev/null http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/akavelink"]
