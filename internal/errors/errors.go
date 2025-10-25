// Package errors provides custom error types and error handling utilities
// for the go-akavelink service.
package errors

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// ErrorCode represents a standardized error code.
type ErrorCode string

const (
	// Client errors (4xx)
	ErrCodeInvalidRequest   ErrorCode = "INVALID_REQUEST"
	ErrCodeMissingParameter ErrorCode = "MISSING_PARAMETER"
	ErrCodeInvalidParameter ErrorCode = "INVALID_PARAMETER"
	ErrCodeUnauthorized     ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden        ErrorCode = "FORBIDDEN"
	ErrCodeNotFound         ErrorCode = "NOT_FOUND"
	ErrCodeMethodNotAllowed ErrorCode = "METHOD_NOT_ALLOWED"
	ErrCodeConflict         ErrorCode = "CONFLICT"
	ErrCodeTooLarge         ErrorCode = "TOO_LARGE"
	ErrCodeUnsupportedMedia ErrorCode = "UNSUPPORTED_MEDIA_TYPE"

	// Server errors (5xx)
	ErrCodeInternalError      ErrorCode = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeTimeout            ErrorCode = "TIMEOUT"
	ErrCodeDatabaseError      ErrorCode = "DATABASE_ERROR"
	ErrCodeNetworkError       ErrorCode = "NETWORK_ERROR"
	ErrCodeStorageError       ErrorCode = "STORAGE_ERROR"
	ErrCodeConfigurationError ErrorCode = "CONFIGURATION_ERROR"
)

// ErrorType represents the category of error.
type ErrorType string

const (
	ErrorTypeValidation ErrorType = "VALIDATION"
	ErrorTypeBusiness   ErrorType = "BUSINESS"
	ErrorTypeSystem     ErrorType = "SYSTEM"
	ErrorTypeNetwork    ErrorType = "NETWORK"
	ErrorTypeStorage    ErrorType = "STORAGE"
	ErrorTypeSecurity   ErrorType = "SECURITY"
	ErrorTypeTimeout    ErrorType = "TIMEOUT"
)

// ServiceError represents a structured service error.
type ServiceError struct {
	Code       ErrorCode              `json:"code"`
	Type       ErrorType              `json:"type"`
	Message    string                 `json:"message"`
	Details    string                 `json:"details,omitempty"`
	HTTPStatus int                    `json:"http_status"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Stack      string                 `json:"stack,omitempty"`
	Cause      error                  `json:"-"`
}

// Error implements the error interface.
func (e *ServiceError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Message, e.Details, e.Code)
	}
	return fmt.Sprintf("%s (%s)", e.Message, e.Code)
}

// Unwrap returns the underlying cause error.
func (e *ServiceError) Unwrap() error {
	return e.Cause
}

// WithContext adds context information to the error.
func (e *ServiceError) WithContext(key string, value interface{}) *ServiceError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithStack adds stack trace information to the error.
func (e *ServiceError) WithStack() *ServiceError {
	e.Stack = getStackTrace()
	return e
}

// NewServiceError creates a new service error.
func NewServiceError(code ErrorCode, message string, httpStatus int) *ServiceError {
	return &ServiceError{
		Code:       code,
		Type:       getErrorType(code),
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// NewServiceErrorWithCause creates a new service error with a cause.
func NewServiceErrorWithCause(code ErrorCode, message string, httpStatus int, cause error) *ServiceError {
	return &ServiceError{
		Code:       code,
		Type:       getErrorType(code),
		Message:    message,
		HTTPStatus: httpStatus,
		Cause:      cause,
	}
}

// NewValidationError creates a validation error.
func NewValidationError(message string, details string) *ServiceError {
	return NewServiceError(ErrCodeInvalidParameter, message, http.StatusBadRequest).WithContext("details", details)
}

// NewNotFoundError creates a not found error.
func NewNotFoundError(resource string) *ServiceError {
	return NewServiceError(ErrCodeNotFound, fmt.Sprintf("%s not found", resource), http.StatusNotFound)
}

// NewUnauthorizedError creates an unauthorized error.
func NewUnauthorizedError(message string) *ServiceError {
	return NewServiceError(ErrCodeUnauthorized, message, http.StatusUnauthorized)
}

// NewForbiddenError creates a forbidden error.
func NewForbiddenError(message string) *ServiceError {
	return NewServiceError(ErrCodeForbidden, message, http.StatusForbidden)
}

// NewInternalError creates an internal server error.
func NewInternalError(message string, cause error) *ServiceError {
	return NewServiceErrorWithCause(ErrCodeInternalError, message, http.StatusInternalServerError, cause).WithStack()
}

// NewStorageError creates a storage-related error.
func NewStorageError(operation string, cause error) *ServiceError {
	return NewServiceErrorWithCause(ErrCodeStorageError, fmt.Sprintf("Storage operation failed: %s", operation), http.StatusInternalServerError, cause).WithStack()
}

// NewNetworkError creates a network-related error.
func NewNetworkError(operation string, cause error) *ServiceError {
	return NewServiceErrorWithCause(ErrCodeNetworkError, fmt.Sprintf("Network operation failed: %s", operation), http.StatusInternalServerError, cause).WithStack()
}

// NewTimeoutError creates a timeout error.
func NewTimeoutError(operation string, timeout string) *ServiceError {
	return NewServiceError(ErrCodeTimeout, fmt.Sprintf("Operation timed out: %s", operation), http.StatusRequestTimeout).WithContext("timeout", timeout)
}

// NewConfigurationError creates a configuration error.
func NewConfigurationError(parameter string, cause error) *ServiceError {
	return NewServiceErrorWithCause(ErrCodeConfigurationError, fmt.Sprintf("Configuration error: %s", parameter), http.StatusInternalServerError, cause).WithStack()
}

// getErrorType determines the error type based on the error code.
func getErrorType(code ErrorCode) ErrorType {
	switch {
	case strings.HasPrefix(string(code), "INVALID_") || strings.HasPrefix(string(code), "MISSING_") || strings.HasPrefix(string(code), "TOO_"):
		return ErrorTypeValidation
	case strings.HasPrefix(string(code), "UNAUTHORIZED") || strings.HasPrefix(string(code), "FORBIDDEN"):
		return ErrorTypeSecurity
	case strings.HasPrefix(string(code), "NOT_FOUND") || strings.HasPrefix(string(code), "CONFLICT"):
		return ErrorTypeBusiness
	case strings.HasPrefix(string(code), "NETWORK_"):
		return ErrorTypeNetwork
	case strings.HasPrefix(string(code), "STORAGE_"):
		return ErrorTypeStorage
	case strings.HasPrefix(string(code), "TIMEOUT"):
		return ErrorTypeTimeout
	default:
		return ErrorTypeSystem
	}
}

// getStackTrace captures the current stack trace.
func getStackTrace() string {
	buf := make([]byte, 1024)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// IsServiceError checks if an error is a ServiceError.
func IsServiceError(err error) bool {
	_, ok := err.(*ServiceError)
	return ok
}

// GetServiceError extracts ServiceError from an error.
func GetServiceError(err error) (*ServiceError, bool) {
	serviceErr, ok := err.(*ServiceError)
	return serviceErr, ok
}

// WrapError wraps a generic error into a ServiceError.
func WrapError(err error, code ErrorCode, message string) *ServiceError {
	if serviceErr, ok := err.(*ServiceError); ok {
		return serviceErr
	}

	httpStatus := http.StatusInternalServerError
	if code == ErrCodeNotFound {
		httpStatus = http.StatusNotFound
	} else if code == ErrCodeUnauthorized {
		httpStatus = http.StatusUnauthorized
	} else if code == ErrCodeForbidden {
		httpStatus = http.StatusForbidden
	} else if code == ErrCodeInvalidParameter || code == ErrCodeMissingParameter {
		httpStatus = http.StatusBadRequest
	}

	return NewServiceErrorWithCause(code, message, httpStatus, err).WithStack()
}

// ErrorResponse represents the standard error response format.
type ErrorResponse struct {
	Success   bool                   `json:"success"`
	Error     string                 `json:"error"`
	Code      ErrorCode              `json:"code,omitempty"`
	Type      ErrorType              `json:"type,omitempty"`
	Details   string                 `json:"details,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

// ToErrorResponse converts a ServiceError to an ErrorResponse.
func (e *ServiceError) ToErrorResponse(requestID string) ErrorResponse {
	return ErrorResponse{
		Success:   false,
		Error:     e.Message,
		Code:      e.Code,
		Type:      e.Type,
		Details:   e.Details,
		RequestID: requestID,
		Context:   e.Context,
		Timestamp: fmt.Sprintf("%d", time.Now().Unix()),
	}
}

// Common error constructors for specific scenarios
func NewBucketNotFoundError(bucketName string) *ServiceError {
	return NewNotFoundError("Bucket").WithContext("bucket_name", bucketName)
}

func NewFileNotFoundError(bucketName, fileName string) *ServiceError {
	return NewNotFoundError("File").WithContext("bucket_name", bucketName).WithContext("file_name", fileName)
}

func NewBucketAlreadyExistsError(bucketName string) *ServiceError {
	return NewServiceError(ErrCodeConflict, "Bucket already exists", http.StatusConflict).WithContext("bucket_name", bucketName)
}

func NewFileTooLargeError(fileName string, size int64, maxSize int64) *ServiceError {
	return NewServiceError(ErrCodeTooLarge, "File too large", http.StatusRequestEntityTooLarge).
		WithContext("file_name", fileName).
		WithContext("file_size", size).
		WithContext("max_size", maxSize)
}

func NewInvalidFileTypeError(fileName, contentType string) *ServiceError {
	return NewValidationError("Invalid file type", fmt.Sprintf("File %s has unsupported content type: %s", fileName, contentType)).
		WithContext("file_name", fileName).
		WithContext("content_type", contentType)
}

func NewUploadFailedError(fileName string, cause error) *ServiceError {
	return NewStorageError("upload", cause).WithContext("file_name", fileName)
}

func NewDownloadFailedError(fileName string, cause error) *ServiceError {
	return NewStorageError("download", cause).WithContext("file_name", fileName)
}

func NewDeleteFailedError(resource string, cause error) *ServiceError {
	return NewStorageError("delete", cause).WithContext("resource", resource)
}
