//go:build integration
// +build integration

package test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	akavesdk "github.com/akave-ai/go-akavelink/internal/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_BucketFileLifecycle exercises the full file lifecycle against a real Akave node.
// Requires valid AKAVE_PRIVATE_KEY and AKAVE_NODE_ADDRESS in the environment.
func TestIntegration_BucketFileLifecycle(t *testing.T) {
	key := os.Getenv("AKAVE_PRIVATE_KEY")
	node := os.Getenv("AKAVE_NODE_ADDRESS")
	if key == "" || node == "" {
		t.Skip("integration test requires AKAVE_PRIVATE_KEY and AKAVE_NODE_ADDRESS; skipping")
	}

	cfg := akavesdk.Config{
		NodeAddress:       node,
		MaxConcurrency:    2,
		BlockPartSize:     1 << 20,
		UseConnectionPool: true,
		PrivateKeyHex:     key,
	}

	client, err := akavesdk.NewClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)
	t.Cleanup(func() { _ = client.Close() })

	ctx := context.Background()
	bucket := fmt.Sprintf("akave-test-%d", time.Now().UnixNano())

	// Ensure cleanup even if test fails mid-way
	t.Cleanup(func() {
		_ = client.DeleteBucket(ctx, bucket)
	})

	// Create bucket
	require.NoError(t, client.CreateBucket(ctx, bucket))

	// Buckets list should include our new bucket (best-effort, some backends may be eventually consistent)
	buckets, err := client.ListBuckets()
	require.NoError(t, err)
	assert.Contains(t, buckets, bucket)

	// Upload a small file
	content := "hello world"
	up, err := client.CreateFileUpload(ctx, bucket, "test.txt")
	require.NoError(t, err)
	meta, err := client.Upload(ctx, up, strings.NewReader(content))
	require.NoError(t, err)
	assert.Equal(t, "test.txt", meta.Name)
	assert.Equal(t, bucket, meta.BucketName)

	// List files should show test.txt
	files, err := client.ListFiles(ctx, bucket)
	require.NoError(t, err)
	found := false
	for _, f := range files {
		if f.Name == "test.txt" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected test.txt in file list")

	// File info
	info, err := client.FileInfo(ctx, bucket, "test.txt")
	require.NoError(t, err)
	assert.Equal(t, "test.txt", info.Name)

	// Download
	dl, err := client.CreateFileDownload(ctx, bucket, "test.txt")
	require.NoError(t, err)
	var out bytes.Buffer
	require.NoError(t, client.Download(ctx, dl, &out))
	assert.Equal(t, content, out.String())

	// Delete file
	require.NoError(t, client.FileDelete(ctx, bucket, "test.txt"))

	// Confirm deletion
	filesAfter, err := client.ListFiles(ctx, bucket)
	require.NoError(t, err)
	for _, f := range filesAfter {
		if f.Name == "test.txt" {
			t.Fatalf("file should be deleted but is still present: %v", f.Name)
		}
	}

	// Delete bucket
	require.NoError(t, client.DeleteBucket(ctx, bucket))
}
