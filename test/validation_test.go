package test

import (
	"bytes"
	"mime/multipart"
	"net/textproto"
	"testing"

	"github.com/akave-ai/go-akavelink/internal/validation"
	"github.com/stretchr/testify/assert"
)

func TestValidateBucketName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid bucket name",
			input:     "my-bucket-123",
			wantError: false,
		},
		{
			name:      "valid bucket name with underscores",
			input:     "my_bucket_123",
			wantError: false,
		},
		{
			name:      "empty bucket name",
			input:     "",
			wantError: true,
			errorMsg:  "bucket name is required",
		},
		{
			name:      "bucket name too long",
			input:     "a123456789012345678901234567890123456789012345678901234567890123456789",
			wantError: true,
			errorMsg:  "bucket name must be between",
		},
		{
			name:      "bucket name with spaces",
			input:     "my bucket",
			wantError: true,
			errorMsg:  "bucket name must contain only alphanumeric",
		},
		{
			name:      "bucket name with special characters",
			input:     "my-bucket!@#",
			wantError: true,
			errorMsg:  "bucket name must contain only alphanumeric",
		},
		{
			name:      "bucket name with path traversal",
			input:     "../etc/passwd",
			wantError: true,
			errorMsg:  "bucket name contains invalid characters",
		},
		{
			name:      "bucket name with forward slash",
			input:     "my/bucket",
			wantError: true,
			errorMsg:  "bucket name must contain only alphanumeric",
		},
		{
			name:      "bucket name with backslash",
			input:     "my\\bucket",
			wantError: true,
			errorMsg:  "bucket name contains invalid characters",
		},
		{
			name:      "bucket name starting with hyphen",
			input:     "-mybucket",
			wantError: true,
			errorMsg:  "bucket name must contain only alphanumeric",
		},
		{
			name:      "bucket name ending with hyphen",
			input:     "mybucket-",
			wantError: true,
			errorMsg:  "bucket name must contain only alphanumeric",
		},
		{
			name:      "single character bucket name",
			input:     "a",
			wantError: false,
		},
		{
			name:      "max length bucket name",
			input:     "a12345678901234567890123456789012345678901234567890123456789012",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateBucketName(tt.input)
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFileName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid file name",
			input:     "document.pdf",
			wantError: false,
		},
		{
			name:      "valid file name with multiple dots",
			input:     "my.file.name.txt",
			wantError: false,
		},
		{
			name:      "empty file name",
			input:     "",
			wantError: true,
			errorMsg:  "file name is required",
		},
		{
			name:      "file name too long",
			input:     string(make([]byte, 256)),
			wantError: true,
			errorMsg:  "file name must not exceed",
		},
		{
			name:      "file name with path traversal",
			input:     "../../../etc/passwd",
			wantError: true,
			errorMsg:  "file name contains invalid characters",
		},
		{
			name:      "file name with absolute path",
			input:     "/etc/passwd",
			wantError: true,
			errorMsg:  "file name contains invalid characters",
		},
		{
			name:      "file name with null byte",
			input:     "file\x00name.txt",
			wantError: true,
			errorMsg:  "file name contains invalid characters",
		},
		{
			name:      "file name with special characters",
			input:     "file<>name.txt",
			wantError: true,
			errorMsg:  "file name contains invalid characters",
		},
		{
			name:      "file name with backslash",
			input:     "path\\file.txt",
			wantError: true,
			errorMsg:  "file name contains invalid characters",
		},
		{
			name:      "file name starting with dot",
			input:     ".hidden",
			wantError: true,
			errorMsg:  "file name must contain only alphanumeric",
		},
		{
			name:      "file name ending with dot",
			input:     "file.",
			wantError: true,
			errorMsg:  "file name must contain only alphanumeric",
		},
		{
			name:      "valid file name with hyphens and underscores",
			input:     "my-file_name-123.txt",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateFileName(tt.input)
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFileUpload(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		size        int64
		contentType string
		wantError   bool
		errorMsg    string
	}{
		{
			name:        "valid file upload",
			filename:    "document.pdf",
			size:        1024,
			contentType: "application/pdf",
			wantError:   false,
		},
		{
			name:        "valid text file",
			filename:    "data.txt",
			size:        512,
			contentType: "text/plain",
			wantError:   false,
		},
		{
			name:        "valid image file",
			filename:    "photo.jpg",
			size:        2048,
			contentType: "image/jpeg",
			wantError:   false,
		},
		{
			name:        "file too large",
			filename:    "large.bin",
			size:        101 * 1024 * 1024, // 101 MB
			contentType: "application/octet-stream",
			wantError:   true,
			errorMsg:    "file size exceeds maximum",
		},
		{
			name:        "empty file",
			filename:    "empty.txt",
			size:        0,
			contentType: "text/plain",
			wantError:   true,
			errorMsg:    "file is empty",
		},
		{
			name:        "invalid MIME type",
			filename:    "script.exe",
			size:        1024,
			contentType: "application/x-msdownload",
			wantError:   true,
			errorMsg:    "file type",
		},
		{
			name:        "invalid filename with path traversal",
			filename:    "../../../etc/passwd",
			size:        1024,
			contentType: "text/plain",
			wantError:   true,
			errorMsg:    "file name contains invalid characters",
		},
		{
			name:        "default content type",
			filename:    "file.bin",
			size:        1024,
			contentType: "",
			wantError:   false,
		},
		{
			name:        "content type with charset",
			filename:    "data.json",
			size:        1024,
			contentType: "application/json; charset=utf-8",
			wantError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a multipart file header
			header := &multipart.FileHeader{
				Filename: tt.filename,
				Size:     tt.size,
			}

			// Set content type in header
			if tt.contentType != "" {
				header.Header = make(textproto.MIMEHeader)
				header.Header.Set("Content-Type", tt.contentType)
			}

			err := validation.ValidateFileUpload(header)
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSanitizeBucketName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already clean",
			input:    "mybucket",
			expected: "mybucket",
		},
		{
			name:     "uppercase to lowercase",
			input:    "MyBucket",
			expected: "mybucket",
		},
		{
			name:     "with whitespace",
			input:    "  my-bucket  ",
			expected: "my-bucket",
		},
		{
			name:     "remove special characters",
			input:    "my!@#bucket",
			expected: "mybucket",
		},
		{
			name:     "preserve hyphens and underscores",
			input:    "my-bucket_123",
			expected: "my-bucket_123",
		},
		{
			name:     "truncate long name",
			input:    "a1234567890123456789012345678901234567890123456789012345678901234567890",
			expected: "a12345678901234567890123456789012345678901234567890123456789012",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validation.SanitizeBucketName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already clean",
			input:    "document.pdf",
			expected: "document.pdf",
		},
		{
			name:     "remove path components",
			input:    "/path/to/file.txt",
			expected: "file.txt",
		},
		{
			name:     "remove path traversal",
			input:    "../../../etc/passwd",
			expected: "passwd", // filepath.Base extracts the last element
		},
		{
			name:     "remove null bytes",
			input:    "file\x00name.txt",
			expected: "filename.txt",
		},
		{
			name:     "remove backslashes",
			input:    "path\\to\\file.txt",
			expected: "pathtofile.txt",
		},
		{
			name:     "trim whitespace",
			input:    "  file.txt  ",
			expected: "file.txt",
		},
		{
			name:     "truncate long filename preserving extension",
			input:    string(bytes.Repeat([]byte("a"), 260)) + ".txt",
			expected: string(bytes.Repeat([]byte("a"), 251)) + ".txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validation.SanitizeFileName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateContentLength(t *testing.T) {
	tests := []struct {
		name      string
		length    int64
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid content length",
			length:    1024,
			wantError: false,
		},
		{
			name:      "zero content length",
			length:    0,
			wantError: false,
		},
		{
			name:      "negative content length",
			length:    -1,
			wantError: true,
			errorMsg:  "invalid content length",
		},
		{
			name:      "content length exceeds maximum",
			length:    101 * 1024 * 1024,
			wantError: true,
			errorMsg:  "content length exceeds maximum",
		},
		{
			name:      "maximum allowed content length",
			length:    100 * 1024 * 1024,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidateContentLength(tt.length)
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := &validation.ValidationError{
		Field:   "testField",
		Message: "test message",
	}

	expected := "testField: test message"
	assert.Equal(t, expected, err.Error())
}
