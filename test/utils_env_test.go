package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/akave-ai/go-akavelink/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUtils_LoadEnvConfig_WithDotenvPath(t *testing.T) {
	dir := t.TempDir()
	dotenv := filepath.Join(dir, ".env")
	require.NoError(t, os.WriteFile(dotenv, []byte("FOO=bar\nBAZ=qux\n"), 0o600))

	// Backup and set DOTENV_PATH
	prevDotenv := os.Getenv("DOTENV_PATH")
	t.Cleanup(func() { os.Setenv("DOTENV_PATH", prevDotenv) })
	require.NoError(t, os.Setenv("DOTENV_PATH", dotenv))

	// Clear any previous values
	_ = os.Unsetenv("FOO")
	_ = os.Unsetenv("BAZ")

	utils.LoadEnvConfig()

	assert.Equal(t, "bar", os.Getenv("FOO"))
	assert.Equal(t, "qux", os.Getenv("BAZ"))
}

// Test that calling LoadEnvConfig without DOTENV_PATH doesn't panic and is safe.
func TestUtils_LoadEnvConfig_NoDotenvPath_Safe(t *testing.T) {
	prevDotenv := os.Getenv("DOTENV_PATH")
	t.Cleanup(func() { os.Setenv("DOTENV_PATH", prevDotenv) })
	_ = os.Unsetenv("DOTENV_PATH")

	utils.LoadEnvConfig()

	_, exists := os.LookupEnv("SOME_NON_EXISTENT_VAR")
	assert.False(t, exists)
}
