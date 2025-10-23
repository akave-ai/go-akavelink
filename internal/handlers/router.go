// Package handlers provides HTTP routing and request handlers for the AkaveLink API.
package handlers

import (
	"context"
	"io"
	"net/http"

	"github.com/akave-ai/go-akavelink/internal/middleware"
	"github.com/gorilla/mux"

	sdksym "github.com/akave-ai/akavesdk/sdk"
)

// ClientAPI is the dependency interface required by HTTP handlers.
// It is satisfied by `internal/sdk.Client`.
type ClientAPI interface {
	CreateBucket(ctx context.Context, name string) error
	DeleteBucket(ctx context.Context, name string) error
	ListBuckets() ([]string, error)
	ListFiles(ctx context.Context, bucket string) ([]sdksym.IPCFileListItem, error)

	CreateFileUpload(ctx context.Context, bucket, fileName string) (*sdksym.IPCFileUpload, error)
	Upload(ctx context.Context, up *sdksym.IPCFileUpload, r io.Reader) (sdksym.IPCFileMetaV2, error)

	CreateFileDownload(ctx context.Context, bucket, fileName string) (sdksym.IPCFileDownload, error)
	Download(ctx context.Context, dl sdksym.IPCFileDownload, w io.Writer) error

	FileInfo(ctx context.Context, bucket, fileName string) (sdksym.IPCFileMeta, error)
	FileDelete(ctx context.Context, bucket, fileName string) error

	NewIPC() (*sdksym.IPC, error)
}

// Server encapsulates dependencies for HTTP handlers.
type Server struct {
	client ClientAPI
}

// NewRouter wires all routes and returns a http.Handler you can mount in main.
func NewRouter(client ClientAPI) http.Handler {
	r := mux.NewRouter().StrictSlash(true)
	s := &Server{client: client}

	// Apply global middleware
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.LogRequest)

	// Health check (no validation needed)
	r.HandleFunc("/health", s.healthHandler).Methods("GET")

	// Buckets
	r.HandleFunc("/buckets/{bucketName}", s.createBucketHandler).Methods("POST")
	r.HandleFunc("/buckets/{bucketName}", s.deleteBucketHandler).Methods("DELETE")
	r.HandleFunc("/buckets", s.viewBucketHandler).Methods("GET")

	// Files - List and Upload (bucket name only)
	r.HandleFunc("/buckets/{bucketName}/files", s.listFilesHandler).Methods("GET")
	r.HandleFunc("/buckets/{bucketName}/files", s.uploadHandler).Methods("POST")

	// Files - Operations requiring both bucket and file name
	r.HandleFunc("/buckets/{bucketName}/files/{fileName}", s.fileInfoHandler).Methods("GET")
	r.HandleFunc("/buckets/{bucketName}/files/{fileName}/download", s.downloadHandler).Methods("GET")
	r.HandleFunc("/buckets/{bucketName}/files/{fileName}", s.fileDeleteHandler).Methods("DELETE")

	return r
}
