# Input Validation & Security

This document describes the comprehensive input validation and sanitization system implemented in go-akavelink.

## Overview

The API implements multiple layers of security to protect against common attacks:

1. **Input Validation** - Validates all user inputs against strict rules
2. **Input Sanitization** - Cleans and normalizes inputs for safe processing
3. **Security Headers** - Adds HTTP security headers to all responses
4. **Request Logging** - Logs all incoming requests for audit trails

## Validation Rules

### Bucket Names

Bucket names must adhere to the following rules:

- **Length**: Between 1 and 63 characters
- **Allowed Characters**: 
  - Lowercase letters (a-z)
  - Uppercase letters (A-Z)
  - Numbers (0-9)
  - Hyphens (-)
  - Underscores (_)
- **Format Requirements**:
  - Must start with an alphanumeric character
  - Must end with an alphanumeric character
  - Cannot contain consecutive dots (..)
  - Cannot contain path separators (/ or \)
  - Cannot contain special characters (!@#$%^&*()+=[]{}|;:'",<>?)

**Valid Examples:**
```
my-bucket
data_store_123
bucket1
MyBucket-2024
user_data_v2
```

**Invalid Examples:**
```
../etc/passwd          # Path traversal
my bucket              # Contains space
bucket!@#              # Special characters
-mybucket              # Starts with hyphen
mybucket-              # Ends with hyphen
a1234567890...123      # Too long (>63 chars)
```

### File Names

File names must adhere to the following rules:

- **Length**: Maximum 255 characters
- **Allowed Characters**:
  - Lowercase letters (a-z)
  - Uppercase letters (A-Z)
  - Numbers (0-9)
  - Dots (.)
  - Hyphens (-)
  - Underscores (_)
- **Format Requirements**:
  - Must start with an alphanumeric character
  - Must end with an alphanumeric character
  - Cannot contain path traversal patterns (..)
  - Cannot contain path separators (/ or \)
  - Cannot contain null bytes (\x00)
  - Cannot contain invalid filename characters (<>:"|?*)

**Valid Examples:**
```
document.pdf
my-file_v2.txt
data.backup.tar.gz
image123.jpg
report-2024-01-15.csv
```

**Invalid Examples:**
```
../../../etc/passwd    # Path traversal
/etc/passwd            # Absolute path
file<>name.txt         # Invalid characters
file\x00name.txt       # Null byte
.hidden                # Starts with dot
file.                  # Ends with dot
path/to/file.txt       # Contains path separator
```

### File Uploads

File uploads are validated for size, type, and content:

#### Size Limits

- **Maximum Size**: 100 MB (104,857,600 bytes)
- **Minimum Size**: Must not be empty (> 0 bytes)

#### Allowed MIME Types

The following MIME types are permitted:

| Category | MIME Type | Description |
|----------|-----------|-------------|
| Binary | `application/octet-stream` | Generic binary data |
| Text | `text/plain` | Plain text files |
| Text | `text/csv` | CSV files |
| JSON | `application/json` | JSON data |
| PDF | `application/pdf` | PDF documents |
| Images | `image/jpeg` | JPEG images |
| Images | `image/png` | PNG images |
| Images | `image/gif` | GIF images |
| Images | `image/webp` | WebP images |
| Video | `video/mp4` | MP4 videos |
| Audio | `audio/mpeg` | MP3 audio |

**Note**: MIME type validation extracts the base type (before semicolon) to handle charset and other parameters.

#### Upload Validation Process

1. **Content-Length Check**: Validates the Content-Length header before parsing
2. **Multipart Form Parsing**: Parses the multipart form data (max 32 MiB in memory)
3. **File Header Validation**: Validates the file header metadata
4. **File Name Validation**: Validates and sanitizes the filename
5. **Size Validation**: Ensures file size is within limits
6. **MIME Type Validation**: Verifies the Content-Type header

## Sanitization

In addition to validation, the API sanitizes inputs for extra safety:

### Bucket Name Sanitization

```go
// Removes whitespace
// Converts to lowercase
// Removes invalid characters
// Truncates to max length
sanitized := validation.SanitizeBucketName(input)
```

**Example:**
```
Input:  "  My-Bucket!@# "
Output: "my-bucket"
```

### File Name Sanitization

```go
// Extracts base filename (removes path)
// Removes null bytes
// Removes path traversal patterns
// Removes path separators
// Truncates to max length (preserving extension)
sanitized := validation.SanitizeFileName(input)
```

**Example:**
```
Input:  "/path/to/../file.txt"
Output: "file.txt"
```

## Security Features

### Path Traversal Prevention

The validation system detects and blocks various path traversal attempts:

- Parent directory references (`..`)
- Absolute paths (`/` or `\`)
- URL-encoded traversal (`%2e%2e`, `%2f`, `%5c`)
- Null bytes (`\x00`)
- Invalid filename characters

### Attack Patterns Detected

The following malicious patterns are detected and rejected:

```
../etc/passwd
..\..\windows\system32
%2e%2e%2f
file\x00.txt
<script>alert('xss')</script>
../../../../../../etc/shadow
```

### Security Headers

All API responses include the following security headers:

```http
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Content-Security-Policy: default-src 'self'
```

#### Header Descriptions

- **X-Content-Type-Options**: Prevents browsers from MIME-sniffing responses
- **X-Frame-Options**: Prevents the page from being embedded in frames (clickjacking protection)
- **X-XSS-Protection**: Enables browser XSS filtering
- **Content-Security-Policy**: Restricts resource loading to same origin

## Error Responses

### Validation Errors

When validation fails, the API returns HTTP 400 (Bad Request) with a structured error response:

```json
{
  "error": "Validation Error",
  "field": "bucketName",
  "message": "bucket name must contain only alphanumeric characters, hyphens, and underscores, and must start and end with an alphanumeric character"
}
```

### Error Response Fields

- **error**: Always "Validation Error" for validation failures
- **field**: The name of the field that failed validation (e.g., "bucketName", "fileName", "file")
- **message**: A human-readable description of the validation error

### Common Validation Errors

#### Bucket Name Errors

```json
{
  "error": "Validation Error",
  "field": "bucketName",
  "message": "bucket name is required"
}
```

```json
{
  "error": "Validation Error",
  "field": "bucketName",
  "message": "bucket name must be between 1 and 63 characters"
}
```

```json
{
  "error": "Validation Error",
  "field": "bucketName",
  "message": "bucket name contains invalid characters or patterns"
}
```

#### File Name Errors

```json
{
  "error": "Validation Error",
  "field": "fileName",
  "message": "file name is required"
}
```

```json
{
  "error": "Validation Error",
  "field": "fileName",
  "message": "file name contains invalid characters or path traversal patterns"
}
```

#### File Upload Errors

```json
{
  "error": "Validation Error",
  "field": "file",
  "message": "file size exceeds maximum allowed size of 104857600 bytes"
}
```

```json
{
  "error": "Validation Error",
  "field": "file",
  "message": "file is empty"
}
```

```json
{
  "error": "Validation Error",
  "field": "file",
  "message": "file type 'application/x-msdownload' is not allowed"
}
```

## Implementation Details

### Package Structure

```
internal/
├── validation/
│   └── validator.go      # Core validation logic
├── middleware/
│   └── validation.go     # HTTP middleware
└── handlers/
    ├── buckets.go        # Bucket handlers with validation
    └── files.go          # File handlers with validation
```

### Validation Package

The `internal/validation` package provides:

- `ValidateBucketName(name string) error`
- `ValidateFileName(name string) error`
- `ValidateFileUpload(header *multipart.FileHeader) error`
- `ValidateContentLength(contentLength int64) error`
- `SanitizeBucketName(name string) string`
- `SanitizeFileName(name string) string`

### Middleware Package

The `internal/middleware` package provides:

- `ValidateBucketName(next http.Handler) http.Handler`
- `ValidateFileName(next http.Handler) http.Handler`
- `ValidateBucketAndFileName(next http.Handler) http.Handler`
- `ValidateContentLength(next http.Handler) http.Handler`
- `SecurityHeaders(next http.Handler) http.Handler`
- `LogRequest(next http.Handler) http.Handler`

### Handler Integration

All handlers validate and sanitize inputs:

```go
// Validate bucket name
if err := validation.ValidateBucketName(bucketName); err != nil {
    s.writeErrorResponse(w, http.StatusBadRequest, err.Error())
    return
}

// Sanitize bucket name for extra safety
bucketName = validation.SanitizeBucketName(bucketName)
```

## Testing

Comprehensive test coverage includes:

### Unit Tests

- `test/validation_test.go` - Tests all validation functions
- `test/middleware_test.go` - Tests all middleware functions

### Integration Tests

- `test/validation_integration_test.go` - Tests validation in HTTP handlers

### Test Coverage

The test suite covers:

- Valid inputs (positive tests)
- Invalid inputs (negative tests)
- Edge cases (boundary conditions)
- Security attacks (path traversal, injection, etc.)
- Sanitization behavior
- Error message accuracy

### Running Tests

```bash
# Run all tests
go test ./test/...

# Run validation tests only
go test ./test/ -run Validation

# Run with coverage
go test ./test/... -cover

# Run with verbose output
go test ./test/... -v
```

## Best Practices

### For API Users

1. **Always validate inputs client-side** before sending requests
2. **Use descriptive names** that follow the validation rules
3. **Handle validation errors gracefully** in your application
4. **Check file sizes** before attempting uploads
5. **Use appropriate MIME types** for your files

### For Developers

1. **Always validate before sanitizing** to catch malicious inputs
2. **Log validation failures** for security monitoring
3. **Never trust user input** - validate everything
4. **Keep validation rules consistent** across all endpoints
5. **Update tests** when adding new validation rules
6. **Document validation changes** in the changelog

## Security Considerations

### Defense in Depth

The validation system implements multiple layers of security:

1. **Input Validation** - First line of defense
2. **Input Sanitization** - Second layer of protection
3. **Security Headers** - Browser-level protection
4. **Request Logging** - Audit trail for security incidents

### Threat Mitigation

The system protects against:

- **Path Traversal Attacks** - Prevented by pattern detection
- **Malicious File Uploads** - Blocked by size and MIME type validation
- **Injection Attacks** - Mitigated by input sanitization
- **XSS Attacks** - Prevented by security headers
- **Clickjacking** - Blocked by X-Frame-Options header
- **MIME Sniffing** - Prevented by X-Content-Type-Options header

### Logging and Monitoring

All validation failures are logged for security monitoring:

```
Warning: Bucket name was sanitized from '../etc' to 'etc'
```

Monitor these logs for:
- Repeated validation failures from the same IP
- Path traversal attempts
- Unusual file upload patterns
- Suspicious input patterns

## Future Enhancements

Potential improvements to the validation system:

1. **Rate Limiting** - Implement per-IP rate limiting
2. **Content Scanning** - Add virus/malware scanning for uploads
3. **Advanced MIME Detection** - Use magic bytes for MIME type verification
4. **Configurable Limits** - Make size limits configurable via environment variables
5. **Custom Validation Rules** - Allow users to define custom validation rules
6. **Audit Logging** - Enhanced logging with structured audit trails
7. **Metrics** - Track validation failure rates and patterns

## References

- [OWASP Input Validation Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Input_Validation_Cheat_Sheet.html)
- [OWASP Path Traversal](https://owasp.org/www-community/attacks/Path_Traversal)
- [OWASP Secure Headers Project](https://owasp.org/www-project-secure-headers/)
- [CWE-22: Path Traversal](https://cwe.mitre.org/data/definitions/22.html)
- [CWE-434: Unrestricted Upload of File with Dangerous Type](https://cwe.mitre.org/data/definitions/434.html)
