# go-akavelink

A Go-based HTTP server that wraps the Akave SDK, exposing Akave APIs over REST. The previous version of this repository was a CLI wrapper around the Akave SDK; refer to [akavelink](https://github.com/akave-ai/akavelink).

## Project Goals

- Provide a production-ready HTTP layer around the Akave SDK.
- Replace dependency on CLI-based wrappers.
- Facilitate integration of Akave storage into other systems via simple REST APIs.

---

## Dev Setup

Follow these steps to set up and run `go-akavelink` locally:

1. Clone the repository

   ```bash
   git clone https://github.com/akave-ai/go-akavelink
   cd go-akavelink
   ```

2. Obtain Akave tokens and a private key

   - Visit the Akave Faucet: https://faucet.akave.ai/
   - Add the Akave network to a wallet
   - Claim test tokens
   - Export the account's private key

3. Configure environment variables

   You can place them in a `.env` at the repo root (auto-loaded by the server) or export them in your shell.

   Required
   - `AKAVE_PRIVATE_KEY` (hex string) — used to sign transactions
   - `AKAVE_NODE_ADDRESS` (host:port) — Akave node gRPC endpoint (e.g., `connect.akave.ai:5500`)

   Optional (with defaults)
   - `AKAVE_MAX_CONCURRENCY` (int, default: 10)
   - `AKAVE_BLOCK_PART_SIZE` (bytes, int64, default: 1048576)

   Example `.env`:

   ```
   AKAVE_PRIVATE_KEY=YOUR_PRIVATE_KEY_HERE
   AKAVE_NODE_ADDRESS=connect.akave.ai:5500
   AKAVE_MAX_CONCURRENCY=10
   AKAVE_BLOCK_PART_SIZE=1048576
   ```

   Tip: You can override the `.env` path by setting `DOTENV_PATH=/absolute/path/to/.env`.

4. Install Go modules

   ```bash
   go mod tidy
   ```

5. Run the server

   ```bash
   # Optionally set PORT (default: 8080)
   PORT=8080 go run ./cmd/server
   ```

   You should see a startup message similar to:

   ```
   20xx/xx/xx Server listening on :8080
   ```

6. Verify

   Visit http://localhost:8080/health or use curl:

   ```bash
   curl -sS http://localhost:8080/health | jq
   # => {"success": true, "data": "ok"}
   ```

---

## API Endpoints

Implemented routes (see `internal/handlers/`):

- Health
  - GET `/health` → 200 with JSON `{ "success": true, "data": "ok" }`

- Buckets
  - GET `/buckets` → list all buckets
  - POST `/buckets/{bucketName}` → create bucket
  - DELETE `/buckets/{bucketName}` → delete bucket and all files within

- Files
  - GET `/buckets/{bucketName}/files` → list files in a bucket
  - POST `/buckets/{bucketName}/files` → upload file (multipart/form-data, field name: `file`)
  - GET `/buckets/{bucketName}/files/{fileName}` → file metadata
  - GET `/buckets/{bucketName}/files/{fileName}/download` → download file content
  - DELETE `/buckets/{bucketName}/files/{fileName}` → delete file

---

## Project Structure

```
go-akavelink/
├── CONTRIBUTING.md
├── LICENSE
├── README.md
├── go.mod
├── go.sum
├── cmd/
│   └── server/
│       └── main.go
├── docs/
│   └── architecture.md
├── internal/
│   ├── handlers/
│   │   ├── router.go       # Wires routes only
│   │   ├── response.go     # JSON envelope + helpers
│   │   ├── health.go       # /health
│   │   ├── buckets.go      # bucket endpoints
│   │   └── files.go        # file endpoints
│   ├── sdk/
│   │   └── sdk.go
│   └── utils/
│       └── env.go
└── test/
    ├── http_endpoints_test.go
    ├── integration_sdk_test.go
    ├── main_test.go
    └── sdk_test.go
```

---

## Contributing

This repository is open to contributions! See [`CONTRIBUTING.md`](./CONTRIBUTING.md).

- Check the [issue tracker](https://github.com/akave-ai/go-akavelink/issues) for `good first issue` and `help wanted` labels.
- Follow the pull request checklist and formatting conventions.
