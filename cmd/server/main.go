// Package main starts the AkaveLink HTTP server exposing the Akave SDK over REST.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

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
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("client close error: %v", err)
		}
	}()

	// Set up HTTP server with routes via internal/handlers
	r := handlers.NewRouter(client)
	// Resolve PORT from environment with default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Listen for shutdown signals
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start server
	go func() {
		log.Printf("Server listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Block until a shutdown signal is received
	<-ctx.Done()
	log.Println("Shutdown signal received, shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited cleanly")
}

func main() {
	MainFunc()
}
