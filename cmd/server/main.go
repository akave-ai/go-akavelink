// Package main starts the AkaveLink HTTP server exposing the Akave SDK over REST.
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/akave-ai/go-akavelink/internal/handlers"
	"github.com/akave-ai/go-akavelink/internal/logger"
	akavesdk "github.com/akave-ai/go-akavelink/internal/sdk"
	"github.com/akave-ai/go-akavelink/internal/utils"
)

func MainFunc() {
	utils.LoadEnvConfig()
	logger.Init(os.Getenv("LOG_LEVEL"), os.Getenv("LOG_FORMAT"))

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
	blockPartSize, parseErr := strconv.ParseInt(blockpartsize, 10, 64)
	if parseErr != nil {
		logger.Error("invalid AKAVE_BLOCK_PART_SIZE", "value", blockpartsize, "error", parseErr)
		os.Exit(1)
	}
	// SDK internal rate limiter has burst 131072; larger block parts cause "Wait(n) exceeds limiter's burst"
	const maxBlockPartSize int64 = 131072
	if blockPartSize > maxBlockPartSize {
		logger.Warn("capping AKAVE_BLOCK_PART_SIZE to SDK limiter burst", "requested", blockPartSize, "capped", maxBlockPartSize)
		blockPartSize = maxBlockPartSize
	}
	maxConcurrency, parseErr2 := strconv.Atoi(maxconcurrency)
	if parseErr2 != nil {
		logger.Error("invalid AKAVE_MAX_CONCURRENCY", "value", maxconcurrency, "error", parseErr2)
		os.Exit(1)
	}
	if key == "" || node == "" {
		logger.Error("AKAVE_PRIVATE_KEY and AKAVE_NODE_ADDRESS must be set")
		os.Exit(1)
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
		logger.Error("client init failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := client.Close(); err != nil {
			logger.Warn("client close error", "error", err)
		}
	}()

	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = handlers.DefaultCORSOrigins
	}
	r := handlers.NewRouter(client, corsOrigins)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	readTimeout := parseDurationEnv("READ_TIMEOUT", 0)
	writeTimeout := parseDurationEnv("WRITE_TIMEOUT", 0)
	if readTimeout > 0 {
		logger.Info("server read timeout configured", "timeout", readTimeout)
	}
	if writeTimeout > 0 {
		logger.Info("server write timeout configured", "timeout", writeTimeout)
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("server listening", "port", port, "addr", ":"+port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received, shutting down gracefully")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Warn("server forced to shutdown", "error", err)
	}
	logger.Info("server exited cleanly")
}

// parseDurationEnv reads a duration from an env var (e.g. "30s", "5m").
// Returns fallback when the variable is unset or empty.
func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		logger.Error("invalid duration for env var", "key", key, "value", v, "error", err)
		os.Exit(1)
	}
	return d
}

func main() {
	MainFunc()
}
