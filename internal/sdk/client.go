package sdk

import (
	"context"
	"fmt"
	"io"

	"github.com/akave-ai/akavesdk/sdk"
)

// Config holds configuration for the Akave SDK client
type Config struct {
	NodeAddress       string
	MaxConcurrency    int
	BlockPartSize     int64
	UseConnectionPool bool
	PrivateKeyHex     string
}

// Client wraps the official Akave SDK
type Client struct {
	*sdk.IPC
	sdk *sdk.SDK
}

// NewClient creates a new Akave client using the official SDK
func NewClient(cfg Config) (*Client, error) {
	if cfg.PrivateKeyHex == "" {
		return nil, fmt.Errorf("private key is required for IPC client but was not provided")
	}

	sdkOpts := []sdk.Option{
		sdk.WithPrivateKey(cfg.PrivateKeyHex),
	}

	// Initialize the official Akave SDK
	newSDK, err := sdk.New(
		cfg.NodeAddress,
		cfg.MaxConcurrency,
		cfg.BlockPartSize,
		cfg.UseConnectionPool,
		sdkOpts...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Akave SDK: %w", err)
	}

	// Get the IPC client from the SDK
	ipc, err := newSDK.IPC()
	if err != nil {
		newSDK.Close()
		return nil, fmt.Errorf("failed to get IPC client from Akave SDK: %w", err)
	}

	return &Client{
		IPC: ipc,
		sdk: newSDK,
	}, nil
}

// NewClientSimple creates a client with simple parameters (for backward compatibility)
func NewClientSimple(nodeAddress, privateKey string) (*Client, error) {
	cfg := Config{
		NodeAddress:       nodeAddress,
		MaxConcurrency:    10,
		BlockPartSize:     1024 * 1024, // 1MB
		UseConnectionPool: true,
		PrivateKeyHex:     privateKey,
	}
	return NewClient(cfg)
}

// ListBuckets lists all buckets using the official Akave SDK
func (c *Client) ListBuckets() ([]string, error) {
	ctx := context.Background()

	// Use the official SDK's bucket listing method
	buckets, err := c.IPC.ListBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	// Convert bucket objects to string slice (bucket names)
	var bucketNames []string
	for _, bucket := range buckets {
		bucketNames = append(bucketNames, bucket.Name)
	}

	return bucketNames, nil
}

// CreateFileUpload creates a file upload stream
func (c *Client) CreateFileUpload(ctx context.Context, bucketName, fileName string) (*sdk.IPCFileUpload, error) {
	return c.IPC.CreateFileUpload(ctx, bucketName, fileName)
}

// Upload uploads file content using the official SDK
func (c *Client) Upload(ctx context.Context, fileUpload *sdk.IPCFileUpload, reader io.Reader) (sdk.IPCFileMetaV2, error) {
	return c.IPC.Upload(ctx, fileUpload, reader)
}

// CreateFileDownload creates a file download stream
func (c *Client) CreateFileDownload(ctx context.Context, bucketName, fileName string) (sdk.IPCFileDownload, error) {
	return c.IPC.CreateFileDownload(ctx, bucketName, fileName)
}

// Download downloads file content using the official SDK
func (c *Client) Download(ctx context.Context, fileDownload sdk.IPCFileDownload, writer io.Writer) error {
	return c.IPC.Download(ctx, fileDownload, writer)
}

// Close gracefully shuts down the SDK connection
func (c *Client) Close() error {
	if c.sdk != nil {
		return c.sdk.Close()
	}
	return nil
}
