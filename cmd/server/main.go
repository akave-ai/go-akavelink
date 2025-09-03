package main

import (
	"log"
	"net/http"
	"os"

	"github.com/akave-ai/go-akavelink/internal/handlers"
	akavesdk "github.com/akave-ai/go-akavelink/internal/sdk"
	"github.com/akave-ai/go-akavelink/internal/utils"
)

func MainFunc() {
	utils.LoadEnvConfig()

	key := os.Getenv("AKAVE_PRIVATE_KEY")
	node := os.Getenv("AKAVE_NODE_ADDRESS")
	if key == "" || node == "" {
		log.Fatal("AKAVE_PRIVATE_KEY and AKAVE_NODE_ADDRESS must be set")
	}

	cfg := akavesdk.Config{
		NodeAddress:       node,
		MaxConcurrency:    10,
		BlockPartSize:     1 << 20,
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
