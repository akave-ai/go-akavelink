package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/akave-ai/go-akavelink/internal/middleware"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestValidateBucketNameMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		bucketName     string
		expectedStatus int
	}{
		{
			name:           "valid bucket name",
			bucketName:     "my-bucket-123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid bucket name with special chars",
			bucketName:     "my-bucket!@#",
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
			// Create a test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with middleware
			r := mux.NewRouter()
			r.Handle("/buckets/{bucketName}", middleware.ValidateBucketName(handler))

			// Create test request
			req := httptest.NewRequest("GET", "/buckets/"+tt.bucketName, nil)
			rr := httptest.NewRecorder()

			// Serve the request
			r.ServeHTTP(rr, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestValidateFileNameMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		fileName       string
		expectedStatus int
	}{
		{
			name:           "valid file name",
			fileName:       "document.pdf",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid file name starting with dot",
			fileName:       ".hidden",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid file name ending with dot",
			fileName:       "file.",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with middleware
			r := mux.NewRouter()
			r.Handle("/files/{fileName}", middleware.ValidateFileName(handler))

			// Create test request
			req := httptest.NewRequest("GET", "/files/"+tt.fileName, nil)
			rr := httptest.NewRecorder()

			// Serve the request
			r.ServeHTTP(rr, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestValidateBucketAndFileNameMiddleware(t *testing.T) {
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
			name:           "invalid file name",
			bucketName:     "my-bucket",
			fileName:       ".hidden",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "both invalid",
			bucketName:     "-bucket",
			fileName:       "file.",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with middleware
			r := mux.NewRouter()
			r.Handle("/buckets/{bucketName}/files/{fileName}",
				middleware.ValidateBucketAndFileName(handler))

			// Create test request
			req := httptest.NewRequest("GET",
				"/buckets/"+tt.bucketName+"/files/"+tt.fileName, nil)
			rr := httptest.NewRecorder()

			// Serve the request
			r.ServeHTTP(rr, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestValidateContentLengthMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		contentLength  string
		expectedStatus int
	}{
		{
			name:           "valid content length",
			contentLength:  "1024",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "content length at max",
			contentLength:  "104857600", // 100 MB
			expectedStatus: http.StatusOK,
		},
		{
			name:           "zero content length",
			contentLength:  "0",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with middleware
			wrappedHandler := middleware.ValidateContentLength(handler)

			// Create test request
			req := httptest.NewRequest("POST", "/upload", nil)
			if tt.contentLength != "" {
				req.Header.Set("Content-Length", tt.contentLength)
			}
			rr := httptest.NewRecorder()

			// Serve the request
			wrappedHandler.ServeHTTP(rr, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	wrappedHandler := middleware.SecurityHeaders(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	// Serve the request
	wrappedHandler.ServeHTTP(rr, req)

	// Check security headers
	assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rr.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", rr.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "default-src 'self'", rr.Header().Get("Content-Security-Policy"))
}

func TestLogRequestMiddleware(t *testing.T) {
	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	wrappedHandler := middleware.LogRequest(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// Serve the request (should not panic or error)
	wrappedHandler.ServeHTTP(rr, req)

	// Check that handler was called
	assert.Equal(t, http.StatusOK, rr.Code)
}
