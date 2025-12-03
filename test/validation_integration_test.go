package test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/akave-ai/go-akavelink/internal/handlers"
	sdksym "github.com/akave-ai/akavesdk/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockClient is a mock implementation of handlers.ClientAPI
type MockClient struct {
	mock.Mock
}

func (m *MockClient) CreateBucket(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockClient) DeleteBucket(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockClient) ListBuckets() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockClient) ListFiles(ctx context.Context, bucket string) ([]sdksym.IPCFileListItem, error) {
	args := m.Called(ctx, bucket)
	return args.Get(0).([]sdksym.IPCFileListItem), args.Error(1)
}

func (m *MockClient) CreateFileUpload(ctx context.Context, bucket, fileName string) (*sdksym.IPCFileUpload, error) {
	args := m.Called(ctx, bucket, fileName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sdksym.IPCFileUpload), args.Error(1)
}

func (m *MockClient) Upload(ctx context.Context, up *sdksym.IPCFileUpload, r io.Reader) (sdksym.IPCFileMetaV2, error) {
	args := m.Called(ctx, up, r)
	return args.Get(0).(sdksym.IPCFileMetaV2), args.Error(1)
}

func (m *MockClient) CreateFileDownload(ctx context.Context, bucket, fileName string) (sdksym.IPCFileDownload, error) {
	args := m.Called(ctx, bucket, fileName)
	return args.Get(0).(sdksym.IPCFileDownload), args.Error(1)
}

func (m *MockClient) Download(ctx context.Context, dl sdksym.IPCFileDownload, w io.Writer) error {
	args := m.Called(ctx, dl, w)
	return args.Error(0)
}

func (m *MockClient) FileInfo(ctx context.Context, bucket, fileName string) (sdksym.IPCFileMeta, error) {
	args := m.Called(ctx, bucket, fileName)
	return args.Get(0).(sdksym.IPCFileMeta), args.Error(1)
}

func (m *MockClient) FileDelete(ctx context.Context, bucket, fileName string) error {
	args := m.Called(ctx, bucket, fileName)
	return args.Error(0)
}

func (m *MockClient) NewIPC() (*sdksym.IPC, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sdksym.IPC), args.Error(1)
}

func TestCreateBucketValidation(t *testing.T) {
	tests := []struct {
		name           string
		bucketName     string
		expectedStatus int
	}{
		{
			name:           "valid bucket name",
			bucketName:     "valid-bucket-123",
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "invalid bucket name with special chars",
			bucketName:     "invalid!bucket",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid bucket name starting with hyphen",
			bucketName:     "-bucket",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "bucket name too long",
			bucketName:     "a1234567890123456789012345678901234567890123456789012345678901234567890",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockClient)
			if tt.expectedStatus == http.StatusCreated {
				mockClient.On("CreateBucket", mock.Anything, mock.Anything).Return(nil)
			}

			router := handlers.NewRouter(mockClient)

			req := httptest.NewRequest("POST", "/buckets/"+tt.bucketName, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestDeleteBucketValidation(t *testing.T) {
	tests := []struct {
		name           string
		bucketName     string
		expectedStatus int
	}{
		{
			name:           "valid bucket name",
			bucketName:     "valid-bucket",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid bucket name",
			bucketName:     "-bucket",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockClient)
			if tt.expectedStatus == http.StatusOK {
				mockClient.On("ListFiles", mock.Anything, mock.Anything).
					Return([]sdksym.IPCFileListItem{}, nil)
				mockClient.On("DeleteBucket", mock.Anything, mock.Anything).Return(nil)
			}

			router := handlers.NewRouter(mockClient)

			req := httptest.NewRequest("DELETE", "/buckets/"+tt.bucketName, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestFileInfoValidation(t *testing.T) {
	tests := []struct {
		name           string
		bucketName     string
		fileName       string
		expectedStatus int
	}{
		{
			name:           "valid bucket and file names",
			bucketName:     "my-bucket",
			fileName:       "document.pdf",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid bucket name",
			bucketName:     "-bucket",
			fileName:       "document.pdf",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid file name starting with dot",
			bucketName:     "my-bucket",
			fileName:       ".hidden",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid file name ending with dot",
			bucketName:     "my-bucket",
			fileName:       "file.",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockClient)
			if tt.expectedStatus == http.StatusOK {
				mockClient.On("FileInfo", mock.Anything, mock.Anything, mock.Anything).
					Return(sdksym.IPCFileMeta{}, nil)
			}

			router := handlers.NewRouter(mockClient)

			req := httptest.NewRequest("GET",
				"/buckets/"+tt.bucketName+"/files/"+tt.fileName, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestFileUploadValidation(t *testing.T) {
	tests := []struct {
		name           string
		bucketName     string
		fileName       string
		fileSize       int64
		contentType    string
		expectedStatus int
	}{
		{
			name:           "valid file upload",
			bucketName:     "my-bucket",
			fileName:       "document.pdf",
			fileSize:       1024,
			contentType:    "application/pdf",
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "invalid bucket name",
			bucketName:     "-bucket",
			fileName:       "document.pdf",
			fileSize:       1024,
			contentType:    "application/pdf",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "file too large",
			bucketName:     "my-bucket",
			fileName:       "large.bin",
			fileSize:       101 * 1024 * 1024, // 101 MB
			contentType:    "application/octet-stream",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid file name",
			bucketName:     "my-bucket",
			fileName:       ".hidden",
			fileSize:       1024,
			contentType:    "text/plain",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockClient)
			if tt.expectedStatus == http.StatusCreated {
				mockClient.On("CreateFileUpload", mock.Anything, mock.Anything, mock.Anything).
					Return(&sdksym.IPCFileUpload{}, nil)
				mockClient.On("Upload", mock.Anything, mock.Anything, mock.Anything).
					Return(sdksym.IPCFileMetaV2{}, nil)
			}

			router := handlers.NewRouter(mockClient)

			// Create multipart form
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			// Create file part
			part, err := writer.CreateFormFile("file", tt.fileName)
			assert.NoError(t, err)

			// Write file content
			fileContent := make([]byte, tt.fileSize)
			_, err = part.Write(fileContent)
			assert.NoError(t, err)

			err = writer.Close()
			assert.NoError(t, err)

			req := httptest.NewRequest("POST", "/buckets/"+tt.bucketName+"/files", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestFileDownloadValidation(t *testing.T) {
	tests := []struct {
		name           string
		bucketName     string
		fileName       string
		expectedStatus int
	}{
		{
			name:           "valid download",
			bucketName:     "my-bucket",
			fileName:       "document.pdf",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid bucket name",
			bucketName:     "-bucket",
			fileName:       "document.pdf",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid file name",
			bucketName:     "my-bucket",
			fileName:       ".hidden",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockClient)
			if tt.expectedStatus == http.StatusOK {
				mockClient.On("CreateFileDownload", mock.Anything, mock.Anything, mock.Anything).
					Return(sdksym.IPCFileDownload{}, nil)
				mockClient.On("Download", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
			}

			router := handlers.NewRouter(mockClient)

			req := httptest.NewRequest("GET",
				"/buckets/"+tt.bucketName+"/files/"+tt.fileName+"/download", nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestFileDeleteValidation(t *testing.T) {
	tests := []struct {
		name           string
		bucketName     string
		fileName       string
		expectedStatus int
	}{
		{
			name:           "valid delete",
			bucketName:     "my-bucket",
			fileName:       "document.pdf",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid bucket name",
			bucketName:     "-bucket",
			fileName:       "document.pdf",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid file name",
			bucketName:     "my-bucket",
			fileName:       "file.",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockClient)
			if tt.expectedStatus == http.StatusOK {
				mockClient.On("FileDelete", mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
			}

			router := handlers.NewRouter(mockClient)

			req := httptest.NewRequest("DELETE",
				"/buckets/"+tt.bucketName+"/files/"+tt.fileName, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}
