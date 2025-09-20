package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/akave-ai/go-akavelink/internal/handlers"
	akavesdk "github.com/akave-ai/go-akavelink/internal/sdk"
	"github.com/akave-ai/go-akavelink/internal/utils"
)

func MainFunc() {
	utils.LoadEnvConfig()

	key := os.Getenv("AKAVE_PRIVATE_KEY")
	node := os.Getenv("AKAVE_NODE_ADDRESS")
	maxconcurrency := os.Getenv("AKAVE_MAX_CONCURRENCY")
	if maxconcurrency == "" {
		maxconcurrency = "10"
	}
	blockpartsize := os.Getenv("AKAVE_BLOCK_PART_SIZE")
	if blockpartsize == "" {
		blockpartsize = "1048576" // 1 MiB
	}
	// Parse block part size to int64 as required by the SDK config
	blockPartSize, parseErr := strconv.ParseInt(blockpartsize, 10, 64)
	if parseErr != nil {
		log.Fatalf("invalid AKAVE_BLOCK_PART_SIZE %q: %v", blockpartsize, parseErr)
	}
	// Parse max concurrency to int as required by the SDK config
	maxConcurrency, parseErr2 := strconv.Atoi(maxconcurrency)
	if parseErr2 != nil {
		log.Fatalf("invalid AKAVE_MAX_CONCURRENCY %q: %v", maxconcurrency, parseErr2)
	}
	if key == "" || node == "" {
		log.Fatal("AKAVE_PRIVATE_KEY and AKAVE_NODE_ADDRESS must be set")
	}

	cfg := akavesdk.Config{
		NodeAddress:       node,
		MaxConcurrency:    maxConcurrency,
		BlockPartSize:     blockPartSize,
		UseConnectionPool: true,
		PrivateKeyHex:     key,
	}
	client, err := akavesdk.NewClient(cfg)
	if err != nil {
		log.Fatalf("client init error: %v", err)
	}
	defer client.Close()

	// Set up HTTP server with routes via internal/handlers
	r := handlers.NewRouter(client)
	log.Println("Server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func main() {
	MainFunc()
}
