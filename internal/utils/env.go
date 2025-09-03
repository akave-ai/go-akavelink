// internal/utils/env.go
package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv" // Still need godotenv here for the loading function
)

// FindModuleRoot finds the root directory of the current Go module by
// traversing up from the caller's file path until a go.mod file is found.
func FindModuleRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(0) // This will be utils/env.go
	if !ok {
		return "", fmt.Errorf("could not get caller info to find module root")
	}
	// Start searching from the directory of the file that *called* FindModuleRoot
	// (or the directory of this file if used directly).
	// A common pattern is to make this function robust to being called from anywhere.
	// We'll start from the current file's directory.
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
			log.Printf("Warning: Failed to find Go module root: %v. .env won't be loaded automatically.", err)
		} else if moduleRoot != "" {
			dotenvPath = filepath.Join(moduleRoot, ".env")
		} else {
			log.Printf("Warning: Could not determine Go module root. .env won't be loaded automatically.")
		}
	}

	// 3. Load the .env file if a path was determined
	if dotenvPath != "" {
		err := godotenv.Load(dotenvPath)
		if err != nil {
			log.Printf("Warning: Could not load .env file from %s: %v. Relying on system environment variables.", dotenvPath, err)
		} else {
			log.Printf("Successfully loaded .env file from %s.", dotenvPath)
		}
	} else {
		log.Println("Warning: No .env path determined. Relying solely on system environment variables.")
	}
}
