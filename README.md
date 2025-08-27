# go-akavelink

🚀 A Go-based HTTP server that wraps the Akave SDK, exposing Akave APIs over REST. The previous version of this repo was a CLI wrapper around the Akave SDK; refer to [akavelink](https://github.com/akave-ai/akavelink).

## Project Goals

  - Provide a production-ready HTTP layer around Akave SDK
  - Replace dependency on CLI-based wrappers
  - Make it easy to integrate Akave storage into other systems via simple REST APIs

## Dev Setup

Follow these steps to set up and run `go-akavelink` locally:

1.  **Clone the Repository:**

    ```bash
    git clone https://github.com/akave-ai/go-akavelink
    cd go-akavelink
    ```

2.  **Get Akave Tokens and Private Key:**

      * Go to the Akave Faucet: [https://faucet.akave.ai/](https://faucet.akave.ai/)
      * Add the Akave network to your wallet.
      * Claim your tokens.
      * Obtain your private key from your wallet.

3.  **Configure Environment Variables:**
    Create a `.env` file in the root of your `go-akavelink` directory with the following content, replacing `YOUR_PRIVATE_KEY_HERE` with the private key you obtained:

    ```
    AKAVE_PRIVATE_KEY="YOUR_PRIVATE_KEY_HERE"
    AKAVE_NODE_ADDRESS="connect.akave.ai:5500"
    ```

4.  **Run Setup Script (Recommended):**

    The `scripts/` directory contains helper scripts (`setup.sh` and `setup.bat`) to automate the environment variable export process.

    **For macOS/Linux:**
    Navigate to the `scripts` directory and give execute permissions, then run the script:

    ```bash
    chmod +x scripts/setup.sh
    ./scripts/setup.sh
    # Or if you prefer bash specifically
    # chmod +x scripts/setup.bat
    # ./scripts/setup.bat
    ```

    This script will export the variables from your `.env` file into your current terminal session. To verify, you can run:

    ```bash
    echo $AKAVE_PRIVATE_KEY
    echo $AKAVE_NODE_ADDRESS
    ```

    These variables will persist for your current terminal session. For permanent environment variables, consider adding them to your shell's configuration file (e.g., `~/.bashrc`, `~/.zshrc`, or `~/.profile`).

    **For Windows PowerShell:**
    You might need to adjust your PowerShell execution policy to run scripts. Open PowerShell as an administrator and run:

    ```powershell
    Set-ExecutionPolicy RemoteSigned -Scope CurrentUser
    ```

    Then, you can run the script (note: Windows Subsystem for Linux (WSL) is recommended for running `.sh` scripts on Windows for a more native experience):

    ```powershell
    # If using PowerShell directly, you might need to execute it like this:
    # powershell -File .\scripts\setup.ps1  (assuming a PowerShell equivalent script)

    # However, for .sh or .bat scripts, it's best to use Git Bash or WSL.
    # If you have Git Bash installed:
    # bash scripts/setup.sh
    # Or, manually export variables as described below if you prefer not to use WSL/Git Bash:
    Get-Content .env | ForEach-Object {
        $line = $_.Trim()
        if (-not ([string]::IsNullOrEmpty($line)) -and -not $line.StartsWith("#")) {
            $parts = $line.Split('=', 2)
            if ($parts.Length -eq 2) {
                $varName = $parts[0]
                $varValue = $parts[1].Trim('"') # Remove quotes if present
                [System.Environment]::SetEnvironmentVariable($varName, $varValue, "Process")
                Write-Host "Exported variable: $varName"
            }
        }
    }
    ```

    To verify they are loaded in your current PowerShell session, you can run:

    ```powershell
    Get-Item Env:AKAVE_PRIVATE_KEY
    Get-Item Env:AKAVE_NODE_ADDRESS
    ```

    These variables will persist for this PowerShell session. For permanent environment variables, refer to Windows documentation on setting system or user-specific environment variables.

5.  **Install Go Modules:**
    Before running the server, ensure all Go modules are tidy and downloaded:

    ```bash
    go mod tidy
    ```

6.  **Run the Server:**

    ```bash
    go run ./cmd/server
    ```

    You should see output similar to:

    ```
    2025/07/07 03:17:14 Starting go-akavelink server on :8080...
    ```

7.  **Verify Installation:**
    Visit `http://localhost:8080/health` in your web browser to verify that the server is running correctly.

-----

## Project Structure

```
go-akavelink/
├── cmd/                # Main entrypoint
│   └── server/
│       └── main.go     # Starts HTTP server
├── internal/           # Internal logic, not exported
│   └── sdk/            # Wrapper around Akave SDK
├── pkg/                # Public packages (if needed)
├── docs/               # Architecture, design, etc.
├── scripts/            # Helper scripts (e.g., setup.sh for env vars)
│   ├── setup.sh
│   └── setup.bat
├── go.mod              # Go module definition
├── README.md           # This file
├── CONTRIBUTING.md     # Guide for contributors
```

## REST API

The server in `cmd/server/main.go` listens on `:8080` and exposes:

- **GET `/health`**
  - Returns plain text `ok`.

- **GET `/buckets/`**
  - Lists buckets: `{ "success": true, "data": ["bucket1", ...] }`.

- **POST `/buckets/{bucketName}`**
  - Creates a bucket. Returns 201 on success.

- **DELETE `/buckets/{bucketName}`**
  - Deletes all files in the bucket, then the bucket.

- **GET `/buckets/{bucketName}/files`**
  - Lists files in a bucket. Items include `RootCID`, `Name`, `EncodedSize`, `ActualSize`, `CreatedAt`.

- **POST `/buckets/{bucketName}/files`**
  - Upload a file (multipart/form-data). Form field: `file`.
  - Response includes `message`, `rootCID`, `bucketName`, `fileName`, `size`, `encodedSize`, `createdAt`, `committedAt`.

- **GET `/buckets/{bucketName}/files/{fileName}`**
  - Returns file info (`RootCID`, `Name`, `BucketName`, `EncodedSize`, `ActualSize`, `IsPublic`, `CreatedAt`).
  - Note: a missing file currently returns HTTP 500 with an SDK error message.

- **GET `/buckets/{bucketName}/files/{fileName}/download`**
  - Streams the file as a download (`Content-Disposition: attachment`).

- **DELETE `/buckets/{bucketName}/files/{fileName}`**
  - Deletes a single file.

### cURL examples

Assuming the server is running and env vars are set.

- Create a bucket
  ```bash
  curl -s -X POST http://localhost:8080/buckets/test_bucket | jq
  ```

- Upload a file
  ```bash
  curl -s -X POST -F "file=@./test.txt" \
    http://localhost:8080/buckets/test_bucket/files | jq
  ```

- List files
  ```bash
  curl -s http://localhost:8080/buckets/test_bucket/files | jq
  ```

- File info
  ```bash
  curl -s http://localhost:8080/buckets/test_bucket/files/test.txt | jq
  ```

- Download
  ```bash
  # -O saves using remote name; -J respects Content-Disposition
  curl -OJ http://localhost:8080/buckets/test_bucket/files/test.txt/download
  ```

- Delete file
  ```bash
  curl -s -X DELETE http://localhost:8080/buckets/test_bucket/files/test.txt | jq
  ```

- Delete bucket
  ```bash
  curl -s -X DELETE http://localhost:8080/buckets/test_bucket | jq
  ```

### Troubleshooting

- **Filename mismatch**: Use the exact `Name` from the list/upload response (e.g., `test.txt` vs `file.txt`). Filenames are case-sensitive.
- **URL encoding**: Quote or URL-encode filenames with spaces/special chars.
- **curl: "File exists"** when using `-OJ`: curl won’t overwrite existing local files. Remove/rename the file or use `-o newname`.
- **"file not exists" from SDK**: Verify `bucketName` and `fileName`. The download route returns 404 when the SDK can’t create a download; the info route currently returns 500.

## Build, Run, Test

- Build
  ```bash
  go build -o bin/server ./cmd/server
  ```

- Run
  ```bash
  go run ./cmd/server
  ```

- Test
  ```bash
  go test ./...
  ```
  Note: Integration tests may require valid `AKAVE_PRIVATE_KEY` and `AKAVE_NODE_ADDRESS`.

## Contributing

  This repo is open to contributions\! See [`CONTRIBUTING.md`](https://www.google.com/search?q=./CONTRIBUTING.md).
  
  - Check the [issue tracker](https://github.com/akave-ai/go-akavelink/issues) for `good first issue` and `help wanted` labels.
  - Follow the PR checklist and formatting conventions.

-----