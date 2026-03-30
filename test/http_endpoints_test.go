// Package test contains HTTP handler tests. By default they use a mock client (no network).
//
// To run all tests against the real Akave endpoint, set AKAVE_PRIVATE_KEY (and optionally
// AKAVE_NODE_ADDRESS) then run the integration test:
//
//	AKAVE_PRIVATE_KEY=your_hex_key go test -v ./test -run TestHTTP_Integration_RealEndpoints
//
// Or load from .env (e.g. in repo root):
//
//	go test -v ./test -run TestHTTP_Integration_RealEndpoints
package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	sdksym "github.com/akave-ai/akavesdk/sdk"
	"github.com/akave-ai/go-akavelink/internal/handlers"
	akavesdk "github.com/akave-ai/go-akavelink/internal/sdk"
	"github.com/akave-ai/go-akavelink/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockIPC implements handlers.IPCDeleter
type mockIPC struct {
	deleted []string
	err     error
}

func (m *mockIPC) FileDelete(ctx context.Context, bucket, file string) error { return nil }

// mockClient implements handlers.ClientAPI
type mockClient struct {
	Buckets         []string
	Files           []sdksym.IPCFileListItem
	CreateErr       error
	DeleteErr       error
	ListErr         error
	ListFilesErr    error
	InfoErr         error
	UploadErr       error
	DownloadErr     error
	NewIPCErr       error
	DownloadContent string   // body written by Download(); default "content"
	FileSize        int64    // ActualSize returned by FileInfo when > 0 (for range tests)
}

func (m *mockClient) CreateBucket(ctx context.Context, bucket string) error { return m.CreateErr }
func (m *mockClient) DeleteBucket(ctx context.Context, bucket string) error { return m.DeleteErr }
func (m *mockClient) ListBuckets() ([]string, error)                        { return m.Buckets, m.ListErr }
func (m *mockClient) ListFiles(ctx context.Context, bucket string) ([]sdksym.IPCFileListItem, error) {
	return m.Files, m.ListFilesErr
}
func (m *mockClient) CreateFileUpload(ctx context.Context, bucket, file string) (*sdksym.IPCFileUpload, error) {
	return nil, nil
}
func (m *mockClient) Upload(ctx context.Context, upload *sdksym.IPCFileUpload, r io.Reader) (sdksym.IPCFileMetaV2, error) {
	var v sdksym.IPCFileMetaV2
	return v, m.UploadErr
}
func (m *mockClient) CreateFileDownload(ctx context.Context, bucket, file string) (sdksym.IPCFileDownload, error) {
	var d sdksym.IPCFileDownload
	return d, nil
}
func (m *mockClient) Download(ctx context.Context, download sdksym.IPCFileDownload, w io.Writer) error {
	body := m.DownloadContent
	if body == "" {
		body = "content"
	}
	_, _ = w.Write([]byte(body))
	return m.DownloadErr
}
func (m *mockClient) FileInfo(ctx context.Context, bucket, file string) (sdksym.IPCFileMeta, error) {
	var v sdksym.IPCFileMeta
	if m.FileSize > 0 {
		v.ActualSize = m.FileSize
	}
	return v, m.InfoErr
}
func (m *mockClient) FileDelete(ctx context.Context, bucket, file string) error { return nil }
func (m *mockClient) NewIPC() (*sdksym.IPC, error)                              { return &sdksym.IPC{}, m.NewIPCErr }

func TestHTTP_Health(t *testing.T) {
	mc := &mockClient{}
	r := handlers.NewRouter(mc, "")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Success bool        `json:"success"`
		Data    interface{} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	if s, ok := resp.Data.(string); ok {
		assert.Equal(t, "ok", strings.TrimSpace(s))
	} else {
		t.Fatalf("expected data to be string, got %T", resp.Data)
	}
}

func TestHTTP_ListBuckets(t *testing.T) {
	mc := &mockClient{Buckets: []string{"a", "b"}}
	r := handlers.NewRouter(mc, "")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/buckets", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Success bool
		Data    []string
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.True(t, resp.Success)
	assert.ElementsMatch(t, []string{"a", "b"}, resp.Data)
}

func TestHTTP_CreateBucket(t *testing.T) {
	mc := &mockClient{}
	r := handlers.NewRouter(mc, "")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/buckets/testbucket", nil))
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestHTTP_Upload(t *testing.T) {
	mc := &mockClient{}
	r := handlers.NewRouter(mc, "")

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, _ := w.CreateFormFile("file", "hello.txt")
	_, _ = io.Copy(fw, strings.NewReader("hello"))
	_ = w.Close()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/buckets/b1/files", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
	var resp map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.Equal(t, true, resp["success"])
}

// TestHTTP_Upload_LargeFile verifies that uploads larger than the 32 MiB multipart
// memory threshold work correctly (the rest spills to disk via Go's multipart handling).
func TestHTTP_Upload_LargeFile(t *testing.T) {
	sizes := []struct {
		name string
		size int64
	}{
		{"50MB", 50 << 20},
		{"100MB", 100 << 20},
		{"500MB", 500 << 20},
	}

	for _, tc := range sizes {
		t.Run(tc.name, func(t *testing.T) {
			mc := &mockClient{}
			r := handlers.NewRouter(mc, "")

			// Use a pipe to avoid buffering the entire body in memory.
			pr, pw := io.Pipe()
			w := multipart.NewWriter(pw)

			go func() {
				fw, err := w.CreateFormFile("file", "largefile.bin")
				if err != nil {
					pw.CloseWithError(err)
					return
				}
				// Write tc.size zero-bytes without allocating a big buffer.
				_, err = io.CopyN(fw, zeroReader{}, tc.size)
				if err != nil {
					pw.CloseWithError(err)
					return
				}
				w.Close()
				pw.Close()
			}()

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/buckets/b1/files", pr)
			req.Header.Set("Content-Type", w.FormDataContentType())
			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())
			var resp map[string]interface{}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
			assert.Equal(t, true, resp["success"])
		})
	}
}

// zeroReader is an io.Reader that produces an infinite stream of zero bytes.
type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

func TestHTTP_Download(t *testing.T) {
	mc := &mockClient{}
	r := handlers.NewRouter(mc, "")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/buckets/b1/files/f.txt/download", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "content", rec.Body.String())
}

func TestHTTP_Download_ByteRange(t *testing.T) {
	// Mock returns "content" (7 bytes); FileInfo returns ActualSize 7 for range parsing
	mc := &mockClient{FileSize: 7, DownloadContent: "content"}
	r := handlers.NewRouter(mc, "")

	t.Run("first_bytes", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/buckets/b1/files/f.txt/download", nil)
		req.Header.Set("Range", "bytes=0-2")
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusPartialContent, rec.Code)
		assert.Equal(t, "con", rec.Body.String())
		assert.Equal(t, "bytes 0-2/7", rec.Header().Get("Content-Range"))
		assert.Equal(t, "3", rec.Header().Get("Content-Length"))
	})

	t.Run("middle_bytes", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/buckets/b1/files/f.txt/download", nil)
		req.Header.Set("Range", "bytes=2-5")
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusPartialContent, rec.Code)
		assert.Equal(t, "nten", rec.Body.String())
		assert.Equal(t, "bytes 2-5/7", rec.Header().Get("Content-Range"))
		assert.Equal(t, "4", rec.Header().Get("Content-Length"))
	})

	t.Run("open_end", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/buckets/b1/files/f.txt/download", nil)
		req.Header.Set("Range", "bytes=4-")
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusPartialContent, rec.Code)
		assert.Equal(t, "ent", rec.Body.String())
		assert.Equal(t, "bytes 4-6/7", rec.Header().Get("Content-Range"))
		assert.Equal(t, "3", rec.Header().Get("Content-Length"))
	})

	t.Run("suffix_bytes", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/buckets/b1/files/f.txt/download", nil)
		req.Header.Set("Range", "bytes=-3")
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusPartialContent, rec.Code)
		assert.Equal(t, "ent", rec.Body.String())
		assert.Equal(t, "bytes 4-6/7", rec.Header().Get("Content-Range"))
		assert.Equal(t, "3", rec.Header().Get("Content-Length"))
	})

	t.Run("416_unsatisfiable", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/buckets/b1/files/f.txt/download", nil)
		req.Header.Set("Range", "bytes=10-20")
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusRequestedRangeNotSatisfiable, rec.Code)
		assert.Equal(t, "bytes */7", rec.Header().Get("Content-Range"))
	})
}

func TestHTTP_DeleteFile(t *testing.T) {
	mc := &mockClient{}
	r := handlers.NewRouter(mc, "")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/buckets/b1/files/f.txt", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHTTP_DeleteBucket(t *testing.T) {
	mc := &mockClient{Files: []sdksym.IPCFileListItem{}}
	r := handlers.NewRouter(mc, "")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/buckets/b1", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestHTTP_Integration_RealEndpoints runs all HTTP endpoint tests against the real Akave API.
// Skip unless AKAVE_PRIVATE_KEY is set. Uses the same router + httptest; only the backend client is real.
func TestHTTP_Integration_RealEndpoints(t *testing.T) {
	utils.LoadEnvConfig()
	key := os.Getenv("AKAVE_PRIVATE_KEY")
	if key == "" {
		t.Skip("AKAVE_PRIVATE_KEY not set; skipping real-endpoint integration test")
	}
	node := os.Getenv("AKAVE_NODE_ADDRESS")
	if node == "" {
		node = "connect.akave.ai:5500"
	}
	maxConc := 10
	if v := os.Getenv("AKAVE_MAX_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			maxConc = n
		}
	}
	blockPartSize := int64(1048576)
	if v := os.Getenv("AKAVE_BLOCK_PART_SIZE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			blockPartSize = n
		}
	}

	cfg := akavesdk.Config{
		NodeAddress:       node,
		MaxConcurrency:    maxConc,
		BlockPartSize:     blockPartSize,
		UseConnectionPool: true,
		PrivateKeyHex:     key,
	}
	client, err := akavesdk.NewClient(cfg)
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	r := handlers.NewRouter(client, "")
	baseURL := ""
	unique := time.Now().UnixNano()
	bucket := fmt.Sprintf("e2e-http-%d", unique)
	fileName := "hello.txt"
	fileContent := "hello from integration test"

	// 1) Health
	t.Run("Health", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, baseURL+"/health", nil)
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		var resp struct {
			Success bool   `json:"success"`
			Data    string `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.True(t, resp.Success)
	})

	// 2) Create bucket
	t.Run("CreateBucket", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, baseURL+"/buckets/"+bucket, nil)
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.Bytes())
	})

	// 3) List buckets (should include our bucket)
	t.Run("ListBuckets", func(t *testing.T) {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, baseURL+"/buckets", nil))
		assert.Equal(t, http.StatusOK, rec.Code)
		var resp struct {
			Success bool     `json:"success"`
			Data    []string `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.True(t, resp.Success)
		assert.Contains(t, resp.Data, bucket)
	})

	// 4) Upload file
	t.Run("Upload", func(t *testing.T) {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		fw, err := w.CreateFormFile("file", fileName)
		require.NoError(t, err)
		_, err = io.Copy(fw, strings.NewReader(fileContent))
		require.NoError(t, err)
		require.NoError(t, w.Close())

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, baseURL+"/buckets/"+bucket+"/files", &body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.Bytes())
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, true, resp["success"])
	})

	// 5) Download file
	t.Run("Download", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, baseURL+"/buckets/"+bucket+"/files/"+fileName+"/download", nil)
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, fileContent, rec.Body.String())
	})

	// 5b) Download with byte range
	t.Run("DownloadByteRange", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, baseURL+"/buckets/"+bucket+"/files/"+fileName+"/download", nil)
		req.Header.Set("Range", "bytes=0-5")
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusPartialContent, rec.Code)
		assert.Equal(t, fileContent[:6], rec.Body.String(), "first 6 bytes")
		assert.Equal(t, "bytes 0-5/"+strconv.Itoa(len(fileContent)), rec.Header().Get("Content-Range"))
	})

	// 6) Delete file
	t.Run("DeleteFile", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, baseURL+"/buckets/"+bucket+"/files/"+fileName, nil)
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	// 7) Delete bucket
	t.Run("DeleteBucket", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, baseURL+"/buckets/"+bucket, nil)
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

// TestHTTP_Integration_LargeFileUpload tests uploading large files (100MB+) against the real
// Akave endpoint. Requires AKAVE_PRIVATE_KEY. Set LARGE_FILE_SIZE_MB to control file size
// (default 100). Uploads zero-filled data and verifies the upload succeeds end-to-end.
func TestHTTP_Integration_LargeFileUpload(t *testing.T) {
	utils.LoadEnvConfig()
	key := os.Getenv("AKAVE_PRIVATE_KEY")
	if key == "" {
		t.Skip("AKAVE_PRIVATE_KEY not set; skipping large file integration test")
	}
	node := os.Getenv("AKAVE_NODE_ADDRESS")
	if node == "" {
		node = "connect.akave.ai:5500"
	}
	maxConc := 10
	if v := os.Getenv("AKAVE_MAX_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			maxConc = n
		}
	}
	blockPartSize := int64(1048576)
	if v := os.Getenv("AKAVE_BLOCK_PART_SIZE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			blockPartSize = n
		}
	}

	sizeMB := int64(100)
	if v := os.Getenv("LARGE_FILE_SIZE_MB"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			sizeMB = n
		}
	}
	fileSize := sizeMB << 20

	cfg := akavesdk.Config{
		NodeAddress:       node,
		MaxConcurrency:    maxConc,
		BlockPartSize:     blockPartSize,
		UseConnectionPool: true,
		PrivateKeyHex:     key,
	}
	client, err := akavesdk.NewClient(cfg)
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	r := handlers.NewRouter(client, "")
	unique := time.Now().UnixNano()
	bucket := fmt.Sprintf("e2e-large-%d", unique)
	fileName := fmt.Sprintf("large-%dmb.bin", sizeMB)

	// Create bucket
	t.Run("CreateBucket", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/buckets/"+bucket, nil)
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.Bytes())
	})

	// Upload large file using pipe (no memory buffering)
	t.Run("UploadLargeFile", func(t *testing.T) {
		pr, pw := io.Pipe()
		w := multipart.NewWriter(pw)

		go func() {
			fw, err := w.CreateFormFile("file", fileName)
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			_, err = io.CopyN(fw, zeroReader{}, fileSize)
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			w.Close()
			pw.Close()
		}()

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/buckets/"+bucket+"/files", pr)
		req.Header.Set("Content-Type", w.FormDataContentType())

		t.Logf("uploading %d MB file...", sizeMB)
		start := time.Now()
		r.ServeHTTP(rec, req)
		elapsed := time.Since(start)
		t.Logf("upload completed in %s", elapsed)

		assert.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, true, resp["success"])
	})

	// Cleanup: delete file and bucket
	t.Run("Cleanup", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/buckets/"+bucket+"/files/"+fileName, nil)
		r.ServeHTTP(rec, req)

		rec = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodDelete, "/buckets/"+bucket, nil)
		r.ServeHTTP(rec, req)
	})
}
