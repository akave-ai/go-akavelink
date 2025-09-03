# go-akavelink

🚀 A Go-based HTTP server that wraps the Akave SDK, exposing Akave APIs over REST. The previous version of this repository was a CLI wrapper around the Akave SDK; refer to [akavelink](https://github.com/akave-ai/akavelink).

## Project Goals

* Provide a production-ready HTTP layer around the Akave SDK.
* Replace dependency on CLI-based wrappers.
* Facilitate integration of Akave storage into other systems via simple REST APIs.

---

## Dev Setup

Follow these steps to set up and run `go-akavelink` locally:

1.  **Clone the Repository:**

    ```bash
    git clone [https://github.com/akave-ai/go-akavelink](https://github.com/akave-ai/go-akavelink)
    cd go-akavelink
    ```

2.  **Obtain Akave Tokens and Private Key:**

    * Access the Akave Faucet: [https://faucet.akave.ai/](https://faucet.akave.ai/)
    * Add the Akave network to a wallet.
    * Claim tokens.
    * Obtain the private key from the wallet.

3.  **Configure Environment Variables:**
    Create a `.env` file in the root of the `go-akavelink` directory with the following content, replacing `YOUR_PRIVATE_KEY_HERE` with the obtained private key:

    ```
    AKAVE_PRIVATE_KEY="YOUR_PRIVATE_KEY_HERE"
    AKAVE_NODE_ADDRESS="connect.akave.ai:5500"
    AKAVE_ENCRYPTION_KEY="YOUR_AKAVE_ENCRYPTION_KEY"
    ```

4.  **Install Go Modules:**
    Before running the server, ensure all Go modules are tidy and downloaded:

    ```bash
    go mod tidy
    ```

5.  **Run the Server:**

    ```bash
    go run ./cmd/server
    ```

    Output similar to the following should appear:

    ```
    2025/07/07 03:17:14 Starting go-akavelink server on :8080...
    ```

6.  **Verify Installation:**
    Visit `http://localhost:8080/health` in a web browser to verify that the server is running correctly.

---

## Project Structure

```
go-akavelink
├── CONTRIBUTING.md
├── LICENSE
├── README.md
├── go.mod
├── go.sum
├── cmd/
│   └── server/
│       └── main.go
├── docs/
├── internal/
│   └── handlers/
│   └── sdk/
│   └── utils/
├── scripts/
└── test/
```

---

## Contributing

This repository is open to contributions! See [`CONTRIBUTING.md`](./CONTRIBUTING.md).

* Check the [issue tracker](https://github.com/akave-ai/go-akavelink/issues) for `good first issue` and `help wanted` labels.
* Follow the pull request checklist and formatting conventions.