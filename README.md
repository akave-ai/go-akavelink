# go-akavelink

A Go-based HTTP server that wraps the Akave SDK, exposing Akave storage over REST. The previous version of this repository was a CLI wrapper; refer to [akavelink](https://github.com/akave-ai/akavelink).

## Project Goals

- Provide a production-ready HTTP layer around the Akave SDK.
- Replace dependency on CLI-based wrappers.
- Facilitate integration of Akave storage into other systems via simple REST APIs.

## Features

- **REST API** — Buckets and file upload/download, health check.
- **Structured logging** — JSON or text logs with configurable level; request logging (method, path, status, duration).
- **CORS** — Configurable allowed origins (default: allow all).
- **Byte-range downloads** — `Range` header support (RFC 7233) for partial file download.
- **Docker** — Multi-stage Dockerfile; build and push to Docker Hub.

---

## Environment configuration

Configuration is read from environment variables. You can use a `.env` file at the repo root (loaded automatically) or export variables in your shell. Override the `.env` path with `DOTENV_PATH`.

### Required

| Variable | Description |
|---------|-------------|
| `AKAVE_PRIVATE_KEY` | Hex-encoded private key used to sign transactions. |
| `AKAVE_NODE_ADDRESS` | Akave node gRPC endpoint (e.g. `connect.akave.ai:5500`). |

### Optional (with defaults)

| Variable | Default | Description |
|----------|---------|--------------|
| `PORT` | `8080` | HTTP server listen port. |
| `AKAVE_MAX_CONCURRENCY` | `10` | Max concurrency for upload/download streams. |
| `AKAVE_BLOCK_PART_SIZE` | `1048576` | Chunk size in bytes for SDK streams. **Capped at 131072** (128 KiB) to match the SDK’s internal rate limiter; larger values are automatically reduced and a warning is logged. |
| `LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error`. |
| `LOG_FORMAT` | `json` | Log format: `json` (for production) or `text`. |
| `CORS_ORIGINS` | `*` | Allowed CORS origins. Use `*` for all, or a comma-separated list (e.g. `https://app.example.com,https://admin.example.com`). |
| `DOTENV_PATH` | (auto) | Absolute path to `.env` file. If unset, the server looks for `.env` at the Go module root. |

### Example `.env`

```env
AKAVE_PRIVATE_KEY=your_hex_private_key
AKAVE_NODE_ADDRESS=connect.akave.ai:5500
PORT=8080
AKAVE_MAX_CONCURRENCY=10
AKAVE_BLOCK_PART_SIZE=131072
LOG_LEVEL=info
LOG_FORMAT=json
CORS_ORIGINS=*
```

---

## Dev setup

1. **Clone the repository**

   ```bash
   git clone https://github.com/akave-ai/go-akavelink
   cd go-akavelink
   ```

2. **Obtain Akave credentials**

   - Visit the [Akave Faucet](https://faucet.akave.ai/).
   - Add the Akave network to a wallet, claim test tokens, and export the account’s private key.

3. **Configure environment**

   Create a `.env` at the repo root (or set `DOTENV_PATH`) with `AKAVE_PRIVATE_KEY` and `AKAVE_NODE_ADDRESS` as above.

4. **Install dependencies**

   ```bash
   go mod tidy
   ```

5. **Run the server**

   ```bash
   go run ./cmd/server
   ```

   You should see a log line like: `"msg":"server listening","port":"8080","addr":":8080"`

6. **Verify**

   ```bash
   curl -sS http://localhost:8080/health | jq
   # => {"success": true, "data": "ok"}
   ```

---

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check. Returns `200` with `{ "success": true, "data": "ok" }`. |
| GET | `/buckets` | List all buckets. |
| POST | `/buckets/{bucketName}` | Create a bucket. |
| DELETE | `/buckets/{bucketName}` | Delete a bucket and all files in it. |
| GET | `/buckets/{bucketName}/files` | List files in a bucket. |
| POST | `/buckets/{bucketName}/files` | Upload a file. Use `multipart/form-data` with field name `file`. |
| GET | `/buckets/{bucketName}/files/{fileName}` | Get file metadata. |
| GET | `/buckets/{bucketName}/files/{fileName}/download` | Download file. Supports **byte ranges** (see below). |
| DELETE | `/buckets/{bucketName}/files/{fileName}` | Delete a file. |

### Byte-range download (RFC 7233)

Send a `Range` header on the download endpoint to request a partial response:

- `Range: bytes=0-499` — first 500 bytes
- `Range: bytes=500-` — from byte 500 to end
- `Range: bytes=-100` — last 100 bytes

Responses: `206 Partial Content` with `Content-Range` and `Content-Length`. Invalid or unsatisfiable ranges return `416 Range Not Satisfiable` with `Content-Range: bytes */<total>`.

---

## Docker

The image is multi-platform (linux/amd64, linux/arm64). Default build matches your host; use `--platform` or buildx for other arches.

### Build

```bash
# Build for current platform
docker build -t go-akavelink:latest .

# Build for a specific platform (e.g. arm64)
docker build --platform linux/arm64 -t go-akavelink:latest .
```

### Run

```bash
docker run --rm -p 8080:8080 \
  -e AKAVE_PRIVATE_KEY=your_hex_key \
  -e AKAVE_NODE_ADDRESS=connect.akave.ai:5500 \
  go-akavelink:latest
```

Optional env vars: `PORT`, `LOG_LEVEL`, `LOG_FORMAT`, `CORS_ORIGINS`, `AKAVE_MAX_CONCURRENCY`, `AKAVE_BLOCK_PART_SIZE`.

The image includes a **HEALTHCHECK** that hits `GET /health` every 30s (start period 5s, 3 retries).

### Push to Docker Hub

```bash
docker login
docker build -t YOUR_DOCKERHUB_USER/go-akavelink:latest .
docker push YOUR_DOCKERHUB_USER/go-akavelink:latest
```

---

## Testing

### Unit tests (no real API)

Uses mocks; no credentials required:

```bash
go test ./test -v
```

Integration tests are **skipped** unless `AKAVE_PRIVATE_KEY` is set.

### Integration tests (real Akave API)

```bash
# With .env containing AKAVE_PRIVATE_KEY (and optionally AKAVE_NODE_ADDRESS):
go test ./test -v -run TestHTTP_Integration_RealEndpoints

# Or inline:
AKAVE_PRIVATE_KEY=your_hex_key go test ./test -v -run TestHTTP_Integration_RealEndpoints
```

### HTTP endpoint tests only

```bash
go test ./test -v -run 'TestHTTP_'
```

### Byte-range download tests

```bash
go test ./test -v -run TestHTTP_Download_ByteRange
```

---

## Project structure

```
go-akavelink/
├── cmd/
│   └── server/
│       └── main.go          # Server entrypoint, env config, logger init
├── internal/
│   ├── handlers/
│   │   ├── router.go        # Route wiring, CORS + logging middleware
│   │   ├── response.go      # JSON envelope and helpers
│   │   ├── health.go        # GET /health
│   │   ├── buckets.go       # Bucket CRUD
│   │   ├── files.go         # File upload, download (with byte-range), delete
│   │   ├── cors.go          # CORS middleware
│   │   └── range.go         # Byte-range parsing and range writer
│   ├── logger/
│   │   └── logger.go        # Structured logging (slog) and request middleware
│   ├── sdk/
│   │   └── sdk.go           # Akave SDK client wrapper
│   └── utils/
│       └── env.go           # .env loading
├── test/
│   ├── http_endpoints_test.go   # HTTP handler tests (mock + integration)
│   ├── sdk_test.go
│   ├── integration_sdk_test.go
│   └── ...
├── Dockerfile
├── .dockerignore
├── go.mod
└── go.sum
```

---

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md). Check the [issue tracker](https://github.com/akave-ai/go-akavelink/issues) for `good first issue` and `help wanted` labels.
