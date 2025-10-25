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

	"github.com/akave-ai/go-akavelink/internal/errors"
	"github.com/akave-ai/go-akavelink/internal/handlers"
	"github.com/akave-ai/go-akavelink/internal/logging"
	akavesdk "github.com/akave-ai/go-akavelink/internal/sdk"
	"github.com/akave-ai/go-akavelink/internal/utils"
)

func MainFunc() {
	ctx := context.Background()

	// Initialize logging
	logging.Info(ctx, "Starting go-akavelink server", map[string]interface{}{
		"service": "go-akavelink",
		"version": "1.0.0",
	})

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
		serviceErr := errors.NewConfigurationError("AKAVE_BLOCK_PART_SIZE", parseErr)
		serviceErr = serviceErr.WithContext("value", blockpartsize)
		logging.Error(ctx, "Configuration error", serviceErr)
		log.Fatalf("invalid AKAVE_BLOCK_PART_SIZE %q: %v", blockpartsize, parseErr)
	}

	// Parse max concurrency to int as required by the SDK config
	maxConcurrency, parseErr2 := strconv.Atoi(maxconcurrency)
	if parseErr2 != nil {
		serviceErr := errors.NewConfigurationError("AKAVE_MAX_CONCURRENCY", parseErr2)
		serviceErr = serviceErr.WithContext("value", maxconcurrency)
		logging.Error(ctx, "Configuration error", serviceErr)
		log.Fatalf("invalid AKAVE_MAX_CONCURRENCY %q: %v", maxconcurrency, parseErr2)
	}

	if key == "" || node == "" {
		serviceErr := errors.NewConfigurationError("Required environment variables", nil)
		serviceErr = serviceErr.WithContext("missing_vars", []string{"AKAVE_PRIVATE_KEY", "AKAVE_NODE_ADDRESS"})
		logging.Error(ctx, "Configuration error", serviceErr)
		log.Fatal("AKAVE_PRIVATE_KEY and AKAVE_NODE_ADDRESS must be set")
	}

	cfg := akavesdk.Config{
		NodeAddress:       node,
		MaxConcurrency:    maxConcurrency,
		BlockPartSize:     blockPartSize,
		UseConnectionPool: true,
		PrivateKeyHex:     key,
	}

	logging.Info(ctx, "Initializing Akave SDK client", map[string]interface{}{
		"node_address":    node,
		"max_concurrency": maxConcurrency,
		"block_part_size": blockPartSize,
	})

	client, err := akavesdk.NewClient(cfg)
	if err != nil {
		serviceErr := errors.NewConfigurationError("SDK client initialization", err)
		logging.Error(ctx, "Client initialization error", serviceErr)
		log.Fatalf("client init error: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			logging.Error(ctx, "Client close error", err)
		}
	}()

	logging.Info(ctx, "Akave SDK client initialized successfully")

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

	logging.Info(ctx, "HTTP server configured", map[string]interface{}{
		"port":          port,
		"read_timeout":  "15s",
		"write_timeout": "15s",
		"idle_timeout":  "60s",
	})

	// Listen for shutdown signals
	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start server
	go func() {
		logging.Info(ctx, "Starting HTTP server", map[string]interface{}{
			"port": port,
		})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serviceErr := errors.NewInternalError("Server startup failed", err)
			logging.Error(ctx, "Server startup error", serviceErr)
			log.Fatalf("server error: %v", err)
		}
	}()

	// Block until a shutdown signal is received
	<-shutdownCtx.Done()
	logging.Info(ctx, "Shutdown signal received, shutting down gracefully...")

	gracefulShutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(gracefulShutdownCtx); err != nil {
		logging.Error(ctx, "Server forced to shutdown", err)
	}
	logging.Info(ctx, "Server exited cleanly")
}

func main() {
	MainFunc()
}
