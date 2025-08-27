// Package sdk provides a high-level client for interacting with the AkaveLink IPC API.
// It encapsulates bucket and file operations and manages the underlying SDK connection.
package sdk

import (
	"context"
	"fmt"
	"io"

	"github.com/akave-ai/akavesdk/sdk"
)

// Config defines parameters for initializing the IPC client.
type Config struct {
	// NodeAddress is the gRPC endpoint for the AkaveLink node, e.g., "localhost:9090".
	NodeAddress       string
	// MaxConcurrency controls parallelism for upload/download streams.
	MaxConcurrency    int
	// BlockPartSize specifies the size of each chunk in bytes.
	BlockPartSize     int64
	// UseConnectionPool toggles the SDK's connection pool.
	UseConnectionPool bool
	// PrivateKeyHex is the hex-encoded private key used for signing transactions.
	PrivateKeyHex     string
}

// Client wraps the AkaveLink IPC API and manages its SDK lifecycle.
type Client struct {
	*sdk.IPC
	core *sdk.SDK
}

// NewClient initializes the AkaveLink SDK and returns a configured IPC client.
// It returns an error if the private key is missing or the SDK cannot initialize.
func NewClient(cfg Config) (*Client, error) {
	if cfg.PrivateKeyHex == "" {
		return nil, fmt.Errorf("configuration error: missing PrivateKeyHex")
	}

	opts := []sdk.Option{
		sdk.WithPrivateKey(cfg.PrivateKeyHex),
	}

	core, err := sdk.New(
		cfg.NodeAddress,
		cfg.MaxConcurrency,
		cfg.BlockPartSize,
		cfg.UseConnectionPool,
		opts..., 
	)
	if err != nil {
		return nil, fmt.Errorf("SDK initialization failed: %w", err)
	}

	ipcClient, err := core.IPC()
	if err != nil {
		core.Close()
		return nil, fmt.Errorf("failed to obtain IPC interface: %w", err)
	}

	return &Client{IPC: ipcClient, core: core}, nil
}

// NewIPC returns a fresh IPC interface instance with an updated transaction nonce.
func (c *Client) NewIPC() (*sdk.IPC, error) {
	return c.core.IPC()
}

// Close terminates all underlying SDK connections.
func (c *Client) Close() error {
	return c.core.Close()
}

// CreateBucket provisions a new bucket on-chain.
func (c *Client) CreateBucket(ctx context.Context, name string) error {
	_, err := c.IPC.CreateBucket(ctx, name)
	if err != nil {
		return fmt.Errorf("bucket creation failed: %w", err)
	}
	return nil
}

// DeleteBucket removes the specified bucket and its contents.
func (c *Client) DeleteBucket(ctx context.Context, name string) error {
	return c.IPC.DeleteBucket(ctx, name)
}

// ListBuckets returns all bucket names accessible by this client.
func (c *Client) ListBuckets() ([]string, error) {
	list, err := c.IPC.ListBuckets(context.Background(), 0, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}
	names := make([]string, len(list))
	for i, b := range list {
		names[i] = b.Name
	}
	return names, nil
}

// CreateFileUpload opens an upload session for a new file.
func (c *Client) CreateFileUpload(ctx context.Context, bucket, fileName string) (*sdk.IPCFileUpload, error) {
	return c.IPC.CreateFileUpload(ctx, bucket, fileName)
}

// Upload streams data into an active upload session and returns file metadata.
func (c *Client) Upload(ctx context.Context, up *sdk.IPCFileUpload, r io.Reader) (sdk.IPCFileMetaV2, error) {
	return c.IPC.Upload(ctx, up, r)
}

// CreateFileDownload initiates a download session for an existing file.
func (c *Client) CreateFileDownload(ctx context.Context, bucket, fileName string) (sdk.IPCFileDownload, error) {
	return c.IPC.CreateFileDownload(ctx, bucket, fileName)
}

// Download streams file data to the provided writer.
func (c *Client) Download(ctx context.Context, dl sdk.IPCFileDownload, w io.Writer) error {
	return c.IPC.Download(ctx, dl, w)
}

// FileInfo retrieves metadata for a single file.
func (c *Client) FileInfo(ctx context.Context, bucket, fileName string) (sdk.IPCFileMeta, error) {
	return c.IPC.FileInfo(ctx, bucket, fileName)
}

// ListFiles returns metadata for all files in the specified bucket.
func (c *Client) ListFiles(ctx context.Context, bucket string) ([]sdk.IPCFileListItem, error) {
	return c.IPC.ListFiles(ctx, bucket, 0, 1000)
}

// FileDelete removes a file from a bucket.
func (c *Client) FileDelete(ctx context.Context, bucket, fileName string) error {
	return c.IPC.FileDelete(ctx, bucket, fileName)
}
