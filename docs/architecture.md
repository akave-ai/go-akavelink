# go-akavelink Architecture

This document outlines the basic structure and design decisions for the project.

---

## 🎯 Purpose

`go-akavelink` is a lightweight HTTP API server written in Go, wrapping the Akave SDK directly to provide REST endpoints for file storage, retrieval, and management.

---

## 🧱 Project Structure

```
go-akavelink/
├── cmd/server/       # Entrypoint to the server (main.go)
├── internal/handlers # HTTP router + handlers (mux)
├── internal/sdk/     # Akave SDK client wrapper logic
├── internal/utils/   # env loader and helpers
├── docs/             # Technical documentation and specs
├── test/             # HTTP and SDK tests
```

---

## 🔄 Request Flow

```
Client --> go-akavelink HTTP API --> Akave SDK --> Akave Backend
```

---

## ✅ Implemented Endpoints

See `internal/handlers/router.go` for the canonical list.

- Health
  - GET `/health`

- Buckets
  - GET `/buckets/` — list all buckets
  - POST `/buckets/{bucketName}` — create a bucket
  - DELETE `/buckets/{bucketName}` — delete a bucket and all its files

- Files
  - GET `/buckets/{bucketName}/files` — list files in a bucket
  - POST `/buckets/{bucketName}/files` — upload file (multipart/form-data, field: `file`)
  - GET `/buckets/{bucketName}/files/{fileName}` — file metadata
  - GET `/buckets/{bucketName}/files/{fileName}/download` — download file content
  - DELETE `/buckets/{bucketName}/files/{fileName}` — delete a file

---

## ⚙️ Configuration

The server reads environment variables (optionally from a `.env` file at the repo root):

- `AKAVE_PRIVATE_KEY` (required)
- `AKAVE_NODE_ADDRESS` (required)
- `AKAVE_MAX_CONCURRENCY` (optional, default 10)
- `AKAVE_BLOCK_PART_SIZE` (optional, default 1048576)
- `DOTENV_PATH` (optional path override for .env)

See `internal/utils/env.go` for details and `cmd/server/main.go` for parsing/validation.

---

## 📌 Notes

- The HTTP layer is a thin wrapper over the SDK via `internal/sdk`.
- Handlers depend on the `ClientAPI` interface, enabling straightforward unit tests with mocks (see `test/http_endpoints_test.go`).
- Follow Go idioms: small interfaces, dependency injection where needed, idiomatic error handling.
