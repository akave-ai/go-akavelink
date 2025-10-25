package test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sdksym "github.com/akave-ai/akavesdk/sdk"
	"github.com/akave-ai/go-akavelink/internal/handlers"
	"github.com/stretchr/testify/assert"
)

// mockIPC implements handlers.IPCDeleter
type mockIPC struct {
	deleted []string
	err     error
}

func (m *mockIPC) FileDelete(ctx context.Context, bucket, file string) error { return nil }

// mockClient implements handlers.ClientAPI
type mockClient struct {
	Buckets      []string
	Files        []sdksym.IPCFileListItem
	CreateErr    error
	DeleteErr    error
	ListErr      error
	ListFilesErr error
	InfoErr      error
	UploadErr    error
	DownloadErr  error
	NewIPCErr    error
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
	_, _ = w.Write([]byte("content"))
	return m.DownloadErr
}
func (m *mockClient) FileInfo(ctx context.Context, bucket, file string) (sdksym.IPCFileMeta, error) {
	var v sdksym.IPCFileMeta
	return v, m.InfoErr
}
func (m *mockClient) FileDelete(ctx context.Context, bucket, file string) error { return nil }
func (m *mockClient) NewIPC() (*sdksym.IPC, error)                              { return &sdksym.IPC{}, m.NewIPCErr }

func TestHTTP_Health(t *testing.T) {
	mc := &mockClient{}
	r := handlers.NewRouter(mc)
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
	// The health endpoint now returns a structured response with additional fields
	if dataMap, ok := resp.Data.(map[string]interface{}); ok {
		assert.Equal(t, "ok", dataMap["status"])
		assert.Contains(t, dataMap, "timestamp")
		assert.Contains(t, dataMap, "service")
		assert.Contains(t, dataMap, "version")
	} else {
		t.Fatalf("expected data to be map, got %T", resp.Data)
	}
}

func TestHTTP_ListBuckets(t *testing.T) {
	mc := &mockClient{Buckets: []string{"a", "b"}}
	r := handlers.NewRouter(mc)
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
	r := handlers.NewRouter(mc)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/buckets/testbucket", nil))
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestHTTP_Upload(t *testing.T) {
	mc := &mockClient{}
	r := handlers.NewRouter(mc)

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

func TestHTTP_Download(t *testing.T) {
	mc := &mockClient{}
	r := handlers.NewRouter(mc)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/buckets/b1/files/f.txt/download", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "content", rec.Body.String())
}

func TestHTTP_DeleteFile(t *testing.T) {
	mc := &mockClient{}
	r := handlers.NewRouter(mc)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/buckets/b1/files/f.txt", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHTTP_DeleteBucket(t *testing.T) {
	mc := &mockClient{Files: []sdksym.IPCFileListItem{}}
	r := handlers.NewRouter(mc)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/buckets/b1", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
}
