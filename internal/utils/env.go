// Package utils provides helpers for loading environment configuration and locating the module root.
//
//revive:disable:var-naming // allow package name 'utils' for historical compatibility
package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/akave-ai/go-akavelink/internal/logger"
	"github.com/joho/godotenv"
)

// FindModuleRoot finds the root directory of the current Go module by
// traversing up from the caller's file path until a go.mod file is found.
func FindModuleRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(0) // This will be utils/env.go
	if !ok {
		return "", fmt.Errorf("could not get caller info to find module root")
	}
	currentDir := filepath.Dir(filename)

	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return currentDir, nil // Found go.mod, this is the root
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir { // Reached file system root
			break
		}
		currentDir = parentDir
	}
	return "", fmt.Errorf("go.mod not found by traversing up from %s", filepath.Dir(filename))
}

// LoadEnvConfig loads environment variables from a .env file.
// It prioritizes a path specified by the DOTENV_PATH environment variable.
// If DOTENV_PATH is not set, it attempts to find the Go module root and load
// the .env file from there.
func LoadEnvConfig() {
	// 1. Check for an explicit DOTENV_PATH environment variable
	dotenvPath := os.Getenv("DOTENV_PATH")

	if dotenvPath == "" {
		// 2. If DOTENV_PATH is not set, try to find the module root and load .env from there
		moduleRoot, err := FindModuleRoot()
		if err != nil {
			logger.Warn("failed to find module root, .env not loaded", "error", err)
		} else if moduleRoot != "" {
			dotenvPath = filepath.Join(moduleRoot, ".env")
		} else {
			logger.Warn("could not determine module root, .env not loaded")
		}
	}

	// 3. Load the .env file if a path was determined
	if dotenvPath != "" {
		err := godotenv.Load(dotenvPath)
		if err != nil {
			logger.Warn("could not load .env, using system env", "path", dotenvPath, "error", err)
		} else {
			logger.Info("loaded .env", "path", dotenvPath)
		}
	} else {
		logger.Warn("no .env path determined, using system environment only")
	}
}
