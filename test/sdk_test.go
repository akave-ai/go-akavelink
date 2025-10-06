package test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	akavesdk "github.com/akave-ai/go-akavelink/internal/sdk"
	"github.com/akave-ai/go-akavelink/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Declare variables to hold values from environment variables
var (
	testPrivateKey  string
	testNodeAddress string
)

func init() {
	utils.LoadEnvConfig()

	testPrivateKey = os.Getenv("AKAVE_PRIVATE_KEY")
	if testPrivateKey == "" {
		testPrivateKey = "e11da8d70c0ef001264b59dc2f"
		log.Println("AKAVE_PRIVATE_KEY not set in environment or .env, using mock private key for tests.")
	}

	testNodeAddress = os.Getenv("AKAVE_NODE_ADDRESS")
	if testNodeAddress == "" {
		testNodeAddress = "connect.akave.ai:5500" // Fallback to a common remote test address
		log.Println("AKAVE_NODE_ADDRESS not set in environment or .env, using fallback node address for tests.")
	}
}

// TestNewClient_Success tests successful client initialization
func TestNewClient_Success(t *testing.T) {
	if os.Getenv("AKAVE_PRIVATE_KEY") == "" {
		t.Skip("AKAVE_PRIVATE_KEY environment variable not set, skipping TestNewClient_Success as it requires a real key.")
	}

	cfg := akavesdk.Config{
		NodeAddress:       testNodeAddress,
		MaxConcurrency:    1,
		BlockPartSize:     1024,
		UseConnectionPool: false,
		PrivateKeyHex:     testPrivateKey,
	}

	client, err := akavesdk.NewClient(cfg)
	require.NoError(t, err, "NewClient should not return an error with valid config")
	require.NotNil(t, client, "NewClient should return a non-nil client")

	defer func() {
		err := client.Close()
		assert.NoError(t, err, "client.Close() should not return an error")
	}()
}

// TestNewIPC_Success verifies we can obtain a fresh IPC instance from the SDK core.
func TestNewIPC_Success(t *testing.T) {
	if os.Getenv("AKAVE_PRIVATE_KEY") == "" {
		t.Skip("AKAVE_PRIVATE_KEY environment variable not set; skipping NewIPC test")
	}

	cfg := akavesdk.Config{
		NodeAddress:       testNodeAddress,
		MaxConcurrency:    1,
		BlockPartSize:     1024,
		UseConnectionPool: false,
		PrivateKeyHex:     testPrivateKey,
	}

	client, err := akavesdk.NewClient(cfg)
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	ipc, err := client.NewIPC()
	require.NoError(t, err)
	require.NotNil(t, ipc)
}

// TestBucketLifecycle_Minimal covers create bucket, list buckets, list files (empty), and cleanup.
func TestBucketLifecycle_Minimal(t *testing.T) {
	if os.Getenv("AKAVE_PRIVATE_KEY") == "" {
		t.Skip("AKAVE_PRIVATE_KEY environment variable not set; skipping bucket lifecycle test")
	}

	cfg := akavesdk.Config{
		NodeAddress:       testNodeAddress,
		MaxConcurrency:    1,
		BlockPartSize:     1024,
		UseConnectionPool: false,
		PrivateKeyHex:     testPrivateKey,
	}

	client, err := akavesdk.NewClient(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()
	bucket := fmt.Sprintf("sdk-min-%d", time.Now().UnixNano())

	// Ensure cleanup
	t.Cleanup(func() { _ = client.DeleteBucket(ctx, bucket) })

	// Create bucket
	require.NoError(t, client.CreateBucket(ctx, bucket))

	// ListBuckets should include the new bucket (eventual consistency may require a short wait)
	buckets, err := client.ListBuckets()
	require.NoError(t, err)
	assert.Contains(t, buckets, bucket)

	// ListFiles should be empty initially
	files, err := client.ListFiles(ctx, bucket)
	require.NoError(t, err)
	assert.Equal(t, 0, len(files))

	// Delete bucket
	require.NoError(t, client.DeleteBucket(ctx, bucket))
}

// TestNewClient_MissingPrivateKey tests client initialization without a private key
func TestNewClient_MissingPrivateKey(t *testing.T) {
	originalPrivateKey := os.Getenv("AKAVE_PRIVATE_KEY")
	os.Setenv("AKAVE_PRIVATE_KEY", "")
	defer os.Setenv("AKAVE_PRIVATE_KEY", originalPrivateKey)

	cfg := akavesdk.Config{
		NodeAddress:       testNodeAddress,
		MaxConcurrency:    1,
		BlockPartSize:     1024,
		UseConnectionPool: false,
		PrivateKeyHex:     "", // This will now be an empty string, simulating missing
	}

	client, err := akavesdk.NewClient(cfg)
	require.Error(t, err, "NewClient should return an error if private key is missing")
	assert.Contains(t, err.Error(), "missing PrivateKeyHex", "Error message should indicate missing PrivateKeyHex")
	assert.Nil(t, client, "NewClient should return a nil client on error")
}

// TestNewClient_SDKInitializationFailure tests a scenario where the underlying SDK fails
func TestNewClient_SDKInitializationFailure(t *testing.T) {
	originalPrivateKey := os.Getenv("AKAVE_PRIVATE_KEY")
	os.Setenv("AKAVE_PRIVATE_KEY", "0xinvalidkey")
	defer os.Setenv("AKAVE_PRIVATE_KEY", originalPrivateKey)

	invalidPrivateKey := os.Getenv("AKAVE_PRIVATE_KEY")

	cfg := akavesdk.Config{
		NodeAddress:       testNodeAddress,
		MaxConcurrency:    1,
		BlockPartSize:     1024,
		UseConnectionPool: false,
		PrivateKeyHex:     invalidPrivateKey,
	}

	client, err := akavesdk.NewClient(cfg)
	require.Error(t, err, "NewClient should return an error with an invalid private key (simulating SDK init failure)")
	assert.Contains(t, err.Error(), "invalid hex character", "Error message should indicate an invalid private key")
	assert.Nil(t, client, "NewClient should return a nil client on SDK init error")
}
