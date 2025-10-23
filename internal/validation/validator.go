// Package validation provides input validation and sanitization for the AkaveLink API.
package validation

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// Bucket name constraints
	MinBucketNameLength = 1
	MaxBucketNameLength = 63

	// File name constraints
	MaxFileNameLength = 255
	MaxFileSize       = 100 * 1024 * 1024 // 100 MB

	// Allowed MIME types for uploads
	AllowedMIMETypes = "application/octet-stream|text/plain|text/csv|application/json|application/pdf|image/jpeg|image/png|image/gif|image/webp|video/mp4|audio/mpeg"
)

var (
	// bucketNameRegex validates bucket names: alphanumeric, hyphens, underscores only
	bucketNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*[a-zA-Z0-9]$`)

	// fileNameRegex validates file names: alphanumeric, hyphens, underscores, dots only
	fileNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*[a-zA-Z0-9]$`)

	// pathTraversalPatterns detects path traversal attempts
	pathTraversalPatterns = []*regexp.Regexp{
		regexp.MustCompile(`\.\.`),           // Parent directory reference
		regexp.MustCompile(`^/`),             // Absolute path
		regexp.MustCompile(`\\`),             // Windows path separator
		regexp.MustCompile(`%2e%2e`),         // URL encoded ..
		regexp.MustCompile(`%2f`),            // URL encoded /
		regexp.MustCompile(`%5c`),            // URL encoded \
		regexp.MustCompile(`\x00`),           // Null byte
		regexp.MustCompile(`[<>:"|?*\x00-\x1f]`), // Invalid filename characters
	}

	// allowedMIMETypesMap for quick lookup
	allowedMIMETypesMap = buildMIMETypeMap()
)

// ValidationError represents a validation error with details.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// buildMIMETypeMap creates a map of allowed MIME types.
func buildMIMETypeMap() map[string]bool {
	types := strings.Split(AllowedMIMETypes, "|")
	m := make(map[string]bool, len(types))
	for _, t := range types {
		m[t] = true
	}
	return m
}

// ValidateBucketName validates bucket name according to rules:
// - Length between 1-63 characters
// - Alphanumeric characters, hyphens, and underscores only
// - Must start and end with alphanumeric character
// - No path traversal patterns
func ValidateBucketName(name string) error {
	if name == "" {
		return &ValidationError{Field: "bucketName", Message: "bucket name is required"}
	}

	if len(name) < MinBucketNameLength || len(name) > MaxBucketNameLength {
		return &ValidationError{
			Field:   "bucketName",
			Message: fmt.Sprintf("bucket name must be between %d and %d characters", MinBucketNameLength, MaxBucketNameLength),
		}
	}

	// Check for path traversal patterns first
	if containsPathTraversal(name) {
		return &ValidationError{
			Field:   "bucketName",
			Message: "bucket name contains invalid characters or patterns",
		}
	}

	// For single character names, just check if alphanumeric
	if len(name) == 1 {
		if !((name[0] >= 'a' && name[0] <= 'z') || (name[0] >= 'A' && name[0] <= 'Z') || (name[0] >= '0' && name[0] <= '9')) {
			return &ValidationError{
				Field:   "bucketName",
				Message: "bucket name must contain only alphanumeric characters, hyphens, and underscores, and must start and end with an alphanumeric character",
			}
		}
		return nil
	}

	if !bucketNameRegex.MatchString(name) {
		return &ValidationError{
			Field:   "bucketName",
			Message: "bucket name must contain only alphanumeric characters, hyphens, and underscores, and must start and end with an alphanumeric character",
		}
	}

	return nil
}

// ValidateFileName validates file name according to rules:
// - Not empty
// - Length <= 255 characters
// - No path traversal patterns
// - Valid characters only
func ValidateFileName(name string) error {
	if name == "" {
		return &ValidationError{Field: "fileName", Message: "file name is required"}
	}

	if len(name) > MaxFileNameLength {
		return &ValidationError{
			Field:   "fileName",
			Message: fmt.Sprintf("file name must not exceed %d characters", MaxFileNameLength),
		}
	}

	// Check for path traversal patterns
	if containsPathTraversal(name) {
		return &ValidationError{
			Field:   "fileName",
			Message: "file name contains invalid characters or path traversal patterns",
		}
	}

	// Validate file name format
	if !fileNameRegex.MatchString(name) {
		return &ValidationError{
			Field:   "fileName",
			Message: "file name must contain only alphanumeric characters, dots, hyphens, and underscores, and must start and end with an alphanumeric character",
		}
	}

	return nil
}

// ValidateFileUpload validates file upload including size and MIME type.
func ValidateFileUpload(header *multipart.FileHeader) error {
	if header == nil {
		return &ValidationError{Field: "file", Message: "file is required"}
	}

	// Validate file name
	if err := ValidateFileName(header.Filename); err != nil {
		return err
	}

	// Validate file size
	if header.Size > MaxFileSize {
		return &ValidationError{
			Field:   "file",
			Message: fmt.Sprintf("file size exceeds maximum allowed size of %d bytes", MaxFileSize),
		}
	}

	if header.Size == 0 {
		return &ValidationError{Field: "file", Message: "file is empty"}
	}

	// Validate MIME type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream" // Default
	}

	// Extract base MIME type (remove charset and other parameters)
	baseMIMEType := strings.Split(contentType, ";")[0]
	baseMIMEType = strings.TrimSpace(baseMIMEType)

	if !allowedMIMETypesMap[baseMIMEType] {
		return &ValidationError{
			Field:   "file",
			Message: fmt.Sprintf("file type '%s' is not allowed", baseMIMEType),
		}
	}

	return nil
}

// SanitizeBucketName sanitizes bucket name by removing invalid characters.
// This should be used after validation for extra safety.
func SanitizeBucketName(name string) string {
	// Remove any whitespace
	name = strings.TrimSpace(name)

	// Convert to lowercase for consistency
	name = strings.ToLower(name)

	// Remove any characters that aren't alphanumeric, hyphen, or underscore
	var sanitized strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			sanitized.WriteRune(r)
		}
	}

	result := sanitized.String()

	// Ensure it doesn't exceed max length
	if len(result) > MaxBucketNameLength {
		result = result[:MaxBucketNameLength]
	}

	return result
}

// SanitizeFileName sanitizes file name by removing path components and invalid characters.
func SanitizeFileName(name string) string {
	// Extract just the filename (remove any path components)
	name = filepath.Base(name)

	// Remove any null bytes
	name = strings.ReplaceAll(name, "\x00", "")

	// Remove leading/trailing whitespace
	name = strings.TrimSpace(name)

	// Remove any path traversal patterns
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, "/", "")
	name = strings.ReplaceAll(name, "\\", "")

	// Ensure it doesn't exceed max length
	if len(name) > MaxFileNameLength {
		// Try to preserve the extension
		ext := filepath.Ext(name)
		if len(ext) > 0 && len(ext) < 10 {
			baseName := name[:len(name)-len(ext)]
			maxBaseLength := MaxFileNameLength - len(ext)
			if len(baseName) > maxBaseLength {
				baseName = baseName[:maxBaseLength]
			}
			name = baseName + ext
		} else {
			name = name[:MaxFileNameLength]
		}
	}

	return name
}

// containsPathTraversal checks if a string contains path traversal patterns.
func containsPathTraversal(s string) bool {
	// Convert to lowercase for case-insensitive matching
	lower := strings.ToLower(s)

	for _, pattern := range pathTraversalPatterns {
		if pattern.MatchString(lower) {
			return true
		}
	}

	return false
}

// ValidateContentLength validates the Content-Length header for uploads.
func ValidateContentLength(contentLength int64) error {
	if contentLength < 0 {
		return &ValidationError{Field: "Content-Length", Message: "invalid content length"}
	}

	if contentLength > MaxFileSize {
		return &ValidationError{
			Field:   "Content-Length",
			Message: fmt.Sprintf("content length exceeds maximum allowed size of %d bytes", MaxFileSize),
		}
	}

	return nil
}
