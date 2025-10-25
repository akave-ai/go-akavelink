package test

import (
	"errors"
	"net/http"
	"testing"

	apperrors "github.com/akave-ai/go-akavelink/internal/errors"
	"github.com/stretchr/testify/assert"
)

func TestServiceError_Basic(t *testing.T) {
	err := apperrors.NewServiceError(apperrors.ErrCodeNotFound, "Resource not found", http.StatusNotFound)

	assert.Equal(t, apperrors.ErrCodeNotFound, err.Code)
	assert.Equal(t, "Resource not found", err.Message)
	assert.Equal(t, http.StatusNotFound, err.HTTPStatus)
	assert.Equal(t, apperrors.ErrorTypeBusiness, err.Type)
	assert.Equal(t, "Resource not found (NOT_FOUND)", err.Error())
}

func TestServiceError_WithCause(t *testing.T) {
	originalErr := errors.New("original error")
	serviceErr := apperrors.NewServiceErrorWithCause(apperrors.ErrCodeInternalError, "Internal error", http.StatusInternalServerError, originalErr)

	assert.Equal(t, originalErr, serviceErr.Cause)
	assert.Equal(t, apperrors.ErrorTypeSystem, serviceErr.Type)
	assert.True(t, apperrors.IsServiceError(serviceErr))
}

func TestServiceError_WithContext(t *testing.T) {
	err := apperrors.NewServiceError(apperrors.ErrCodeNotFound, "Resource not found", http.StatusNotFound)
	err = err.WithContext("resource_id", "123")
	err = err.WithContext("resource_type", "user")

	assert.Equal(t, "123", err.Context["resource_id"])
	assert.Equal(t, "user", err.Context["resource_type"])
}

func TestServiceError_WithStack(t *testing.T) {
	err := apperrors.NewServiceError(apperrors.ErrCodeInternalError, "Internal error", http.StatusInternalServerError)
	err = err.WithStack()

	assert.NotEmpty(t, err.Stack)
	assert.Contains(t, err.Stack, "TestServiceError_WithStack")
}

func TestValidationError(t *testing.T) {
	err := apperrors.NewValidationError("Invalid input", "field 'email' is required")

	assert.Equal(t, apperrors.ErrCodeInvalidParameter, err.Code)
	assert.Equal(t, http.StatusBadRequest, err.HTTPStatus)
	assert.Equal(t, apperrors.ErrorTypeValidation, err.Type)
	assert.Equal(t, "Invalid input", err.Message)
	assert.Equal(t, "field 'email' is required", err.Context["details"])
}

func TestNotFoundError(t *testing.T) {
	err := apperrors.NewNotFoundError("User")

	assert.Equal(t, apperrors.ErrCodeNotFound, err.Code)
	assert.Equal(t, http.StatusNotFound, err.HTTPStatus)
	assert.Equal(t, apperrors.ErrorTypeBusiness, err.Type)
	assert.Equal(t, "User not found", err.Message)
}

func TestUnauthorizedError(t *testing.T) {
	err := apperrors.NewUnauthorizedError("Invalid credentials")

	assert.Equal(t, apperrors.ErrCodeUnauthorized, err.Code)
	assert.Equal(t, http.StatusUnauthorized, err.HTTPStatus)
	assert.Equal(t, apperrors.ErrorTypeSecurity, err.Type)
	assert.Equal(t, "Invalid credentials", err.Message)
}

func TestForbiddenError(t *testing.T) {
	err := apperrors.NewForbiddenError("Access denied")

	assert.Equal(t, apperrors.ErrCodeForbidden, err.Code)
	assert.Equal(t, http.StatusForbidden, err.HTTPStatus)
	assert.Equal(t, apperrors.ErrorTypeSecurity, err.Type)
	assert.Equal(t, "Access denied", err.Message)
}

func TestInternalError(t *testing.T) {
	originalErr := errors.New("database connection failed")
	err := apperrors.NewInternalError("Database operation failed", originalErr)

	assert.Equal(t, apperrors.ErrCodeInternalError, err.Code)
	assert.Equal(t, http.StatusInternalServerError, err.HTTPStatus)
	assert.Equal(t, apperrors.ErrorTypeSystem, err.Type)
	assert.Equal(t, originalErr, err.Cause)
	assert.NotEmpty(t, err.Stack)
}

func TestStorageError(t *testing.T) {
	originalErr := errors.New("disk full")
	err := apperrors.NewStorageError("save_file", originalErr)

	assert.Equal(t, apperrors.ErrCodeStorageError, err.Code)
	assert.Equal(t, http.StatusInternalServerError, err.HTTPStatus)
	assert.Equal(t, apperrors.ErrorTypeStorage, err.Type)
	assert.Equal(t, "Storage operation failed: save_file", err.Message)
	assert.Equal(t, originalErr, err.Cause)
}

func TestNetworkError(t *testing.T) {
	originalErr := errors.New("connection timeout")
	err := apperrors.NewNetworkError("api_call", originalErr)

	assert.Equal(t, apperrors.ErrCodeNetworkError, err.Code)
	assert.Equal(t, http.StatusInternalServerError, err.HTTPStatus)
	assert.Equal(t, apperrors.ErrorTypeNetwork, err.Type)
	assert.Equal(t, "Network operation failed: api_call", err.Message)
	assert.Equal(t, originalErr, err.Cause)
}

func TestTimeoutError(t *testing.T) {
	err := apperrors.NewTimeoutError("database_query", "30s")

	assert.Equal(t, apperrors.ErrCodeTimeout, err.Code)
	assert.Equal(t, http.StatusRequestTimeout, err.HTTPStatus)
	assert.Equal(t, apperrors.ErrorTypeTimeout, err.Type)
	assert.Equal(t, "Operation timed out: database_query", err.Message)
	assert.Equal(t, "30s", err.Context["timeout"])
}

func TestConfigurationError(t *testing.T) {
	originalErr := errors.New("invalid config file")
	err := apperrors.NewConfigurationError("DATABASE_URL", originalErr)

	assert.Equal(t, apperrors.ErrCodeConfigurationError, err.Code)
	assert.Equal(t, http.StatusInternalServerError, err.HTTPStatus)
	assert.Equal(t, apperrors.ErrorTypeSystem, err.Type)
	assert.Equal(t, "Configuration error: DATABASE_URL", err.Message)
	assert.Equal(t, originalErr, err.Cause)
}

func TestBucketNotFoundError(t *testing.T) {
	err := apperrors.NewBucketNotFoundError("my-bucket")

	assert.Equal(t, apperrors.ErrCodeNotFound, err.Code)
	assert.Equal(t, http.StatusNotFound, err.HTTPStatus)
	assert.Equal(t, "Bucket not found", err.Message)
	assert.Equal(t, "my-bucket", err.Context["bucket_name"])
}

func TestFileNotFoundError(t *testing.T) {
	err := apperrors.NewFileNotFoundError("my-bucket", "file.txt")

	assert.Equal(t, apperrors.ErrCodeNotFound, err.Code)
	assert.Equal(t, http.StatusNotFound, err.HTTPStatus)
	assert.Equal(t, "File not found", err.Message)
	assert.Equal(t, "my-bucket", err.Context["bucket_name"])
	assert.Equal(t, "file.txt", err.Context["file_name"])
}

func TestBucketAlreadyExistsError(t *testing.T) {
	err := apperrors.NewBucketAlreadyExistsError("my-bucket")

	assert.Equal(t, apperrors.ErrCodeConflict, err.Code)
	assert.Equal(t, http.StatusConflict, err.HTTPStatus)
	assert.Equal(t, "Bucket already exists", err.Message)
	assert.Equal(t, "my-bucket", err.Context["bucket_name"])
}

func TestFileTooLargeError(t *testing.T) {
	err := apperrors.NewFileTooLargeError("large-file.txt", 1000000, 500000)

	assert.Equal(t, apperrors.ErrCodeTooLarge, err.Code)
	assert.Equal(t, http.StatusRequestEntityTooLarge, err.HTTPStatus)
	assert.Equal(t, "File too large", err.Message)
	assert.Equal(t, "large-file.txt", err.Context["file_name"])
	assert.Equal(t, int64(1000000), err.Context["file_size"])
	assert.Equal(t, int64(500000), err.Context["max_size"])
}

func TestInvalidFileTypeError(t *testing.T) {
	err := apperrors.NewInvalidFileTypeError("script.exe", "application/x-executable")

	assert.Equal(t, apperrors.ErrCodeInvalidParameter, err.Code)
	assert.Equal(t, http.StatusBadRequest, err.HTTPStatus)
	assert.Equal(t, "Invalid file type", err.Message)
	assert.Equal(t, "script.exe", err.Context["file_name"])
	assert.Equal(t, "application/x-executable", err.Context["content_type"])
}

func TestUploadFailedError(t *testing.T) {
	originalErr := errors.New("network error")
	err := apperrors.NewUploadFailedError("file.txt", originalErr)

	assert.Equal(t, apperrors.ErrCodeStorageError, err.Code)
	assert.Equal(t, http.StatusInternalServerError, err.HTTPStatus)
	assert.Equal(t, "Storage operation failed: upload", err.Message)
	assert.Equal(t, originalErr, err.Cause)
	assert.Equal(t, "file.txt", err.Context["file_name"])
}

func TestDownloadFailedError(t *testing.T) {
	originalErr := errors.New("file not found")
	err := apperrors.NewDownloadFailedError("file.txt", originalErr)

	assert.Equal(t, apperrors.ErrCodeStorageError, err.Code)
	assert.Equal(t, http.StatusInternalServerError, err.HTTPStatus)
	assert.Equal(t, "Storage operation failed: download", err.Message)
	assert.Equal(t, originalErr, err.Cause)
	assert.Equal(t, "file.txt", err.Context["file_name"])
}

func TestDeleteFailedError(t *testing.T) {
	originalErr := errors.New("permission denied")
	err := apperrors.NewDeleteFailedError("file.txt", originalErr)

	assert.Equal(t, apperrors.ErrCodeStorageError, err.Code)
	assert.Equal(t, http.StatusInternalServerError, err.HTTPStatus)
	assert.Equal(t, "Storage operation failed: delete", err.Message)
	assert.Equal(t, originalErr, err.Cause)
	assert.Equal(t, "file.txt", err.Context["resource"])
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("generic error")
	wrappedErr := apperrors.WrapError(originalErr, apperrors.ErrCodeNotFound, "Resource not found")

	assert.Equal(t, apperrors.ErrCodeNotFound, wrappedErr.Code)
	assert.Equal(t, "Resource not found", wrappedErr.Message)
	assert.Equal(t, http.StatusNotFound, wrappedErr.HTTPStatus)
	assert.Equal(t, originalErr, wrappedErr.Cause)
	assert.NotEmpty(t, wrappedErr.Stack)
}

func TestWrapError_ServiceError(t *testing.T) {
	serviceErr := apperrors.NewServiceError(apperrors.ErrCodeNotFound, "Already a service error", http.StatusNotFound)
	wrappedErr := apperrors.WrapError(serviceErr, apperrors.ErrCodeInternalError, "Should not wrap")

	// Should return the original service error unchanged
	assert.Equal(t, serviceErr, wrappedErr)
}

func TestErrorResponse(t *testing.T) {
	err := apperrors.NewServiceError(apperrors.ErrCodeNotFound, "Resource not found", http.StatusNotFound)
	err = err.WithContext("resource_id", "123")

	response := err.ToErrorResponse("req-123")

	assert.False(t, response.Success)
	assert.Equal(t, "Resource not found", response.Error)
	assert.Equal(t, apperrors.ErrCodeNotFound, response.Code)
	assert.Equal(t, apperrors.ErrorTypeBusiness, response.Type)
	assert.Equal(t, "req-123", response.RequestID)
	assert.Equal(t, "123", response.Context["resource_id"])
	assert.NotEmpty(t, response.Timestamp)
}

func TestIsServiceError(t *testing.T) {
	serviceErr := apperrors.NewServiceError(apperrors.ErrCodeNotFound, "Not found", http.StatusNotFound)
	genericErr := errors.New("generic error")

	assert.True(t, apperrors.IsServiceError(serviceErr))
	assert.False(t, apperrors.IsServiceError(genericErr))
}

func TestGetServiceError(t *testing.T) {
	serviceErr := apperrors.NewServiceError(apperrors.ErrCodeNotFound, "Not found", http.StatusNotFound)
	genericErr := errors.New("generic error")

	retrievedErr, ok := apperrors.GetServiceError(serviceErr)
	assert.True(t, ok)
	assert.Equal(t, serviceErr, retrievedErr)

	_, ok = apperrors.GetServiceError(genericErr)
	assert.False(t, ok)
}
