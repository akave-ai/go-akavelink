package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/akave-ai/go-akavelink/internal/logging"
	"github.com/akave-ai/go-akavelink/internal/middleware"
	"github.com/stretchr/testify/assert"
)

func TestLoggingMiddleware(t *testing.T) {
	var buf bytes.Buffer

	// Create middleware with custom logger
	middleware := middleware.NewLoggingMiddleware("test-service")
	middleware.SetOutput(&buf)

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Wrap with middleware
	wrappedHandler := middleware.LoggingHandler(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.RemoteAddr = "192.168.1.1:12345"

	// Create response recorder
	rr := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "test response", rr.Body.String())

	// Check that request ID was added to response headers
	assert.NotEmpty(t, rr.Header().Get("X-Request-ID"))

	// Check that logs were written
	assert.NotEmpty(t, buf.String())

	// Check that the log contains expected messages
	logOutput := buf.String()
	assert.Contains(t, logOutput, "HTTP request started")
	assert.Contains(t, logOutput, "HTTP request completed")
	assert.Contains(t, logOutput, "GET")
	assert.Contains(t, logOutput, "/test")
	assert.Contains(t, logOutput, "test-agent")
	assert.Contains(t, logOutput, "192.168.1.1")
	assert.Contains(t, logOutput, "status_code")
	assert.Contains(t, logOutput, "duration_ms")
}

func TestSecurityMiddleware(t *testing.T) {
	var buf bytes.Buffer

	middleware := middleware.NewSecurityMiddleware("test-service")
	middleware.SetOutput(&buf)

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	wrappedHandler := middleware.SecurityHandler(handler)

	// Test normal request
	req := httptest.NewRequest("GET", "/normal", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Test suspicious request
	suspiciousReq := httptest.NewRequest("GET", "/admin/../etc/passwd", nil)
	suspiciousReq.Header.Set("User-Agent", "suspicious-bot")
	rr = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, suspiciousReq)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Check that security logs were written
	assert.NotEmpty(t, buf.String())

	// Parse log entries
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	var securityLogs []logging.LogEntry

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var entry logging.LogEntry
		err := json.Unmarshal(line, &entry)
		if err != nil {
			continue
		}
		if entry.Fields["security_event"] != nil {
			securityLogs = append(securityLogs, entry)
		}
	}

	// Should have security logs for suspicious request
	assert.GreaterOrEqual(t, len(securityLogs), 1)

	securityLog := securityLogs[0]
	assert.Equal(t, "WARN", string(securityLog.Level))
	assert.Equal(t, "Security event", securityLog.Message)
	assert.Equal(t, "suspicious_request", securityLog.Fields["security_event"])
}

func TestAuditMiddleware(t *testing.T) {
	var buf bytes.Buffer

	middleware := middleware.NewAuditMiddleware("test-service")
	middleware.SetOutput(&buf)

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	wrappedHandler := middleware.AuditHandler(handler)

	// Test GET request (should not be audited)
	req := httptest.NewRequest("GET", "/read", nil)
	rr := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Test POST request (should be audited)
	postReq := httptest.NewRequest("POST", "/create", nil)
	rr = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, postReq)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Test DELETE request (should be audited)
	deleteReq := httptest.NewRequest("DELETE", "/delete", nil)
	rr = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(rr, deleteReq)

	assert.Equal(t, http.StatusOK, rr.Code)

	// Check that audit logs were written
	assert.NotEmpty(t, buf.String())

	// Parse log entries
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	var auditLogs []logging.LogEntry

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var entry logging.LogEntry
		err := json.Unmarshal(line, &entry)
		if err != nil {
			continue
		}
		if entry.Fields["audit_operation"] != nil {
			auditLogs = append(auditLogs, entry)
		}
	}

	// Should have audit logs for POST and DELETE requests
	assert.GreaterOrEqual(t, len(auditLogs), 2)

	// Check audit log content
	auditLog := auditLogs[0]
	assert.Equal(t, "INFO", string(auditLog.Level))
	assert.Equal(t, "Audit trail", auditLog.Message)
	assert.Equal(t, "data_modification", auditLog.Fields["audit_operation"])
	assert.Contains(t, auditLog.Fields["audit_resource"], "/")
}

func TestGetRequestID(t *testing.T) {
	// Test with request ID in context
	ctx := context.WithValue(context.Background(), middleware.RequestIDKey, "test-request-123")
	requestID := middleware.GetRequestID(ctx)
	assert.Equal(t, "test-request-123", requestID)

	// Test without request ID in context
	emptyCtx := context.Background()
	requestID = middleware.GetRequestID(emptyCtx)
	assert.Empty(t, requestID)
}

func TestRequestIDGeneration(t *testing.T) {
	// Test that request IDs are unique
	req1 := httptest.NewRequest("GET", "/test1", nil)
	req2 := httptest.NewRequest("GET", "/test2", nil)

	middleware := middleware.NewLoggingMiddleware("test-service")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware.LoggingHandler(handler)

	rr1 := httptest.NewRecorder()
	rr2 := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr1, req1)
	wrappedHandler.ServeHTTP(rr2, req2)

	requestID1 := rr1.Header().Get("X-Request-ID")
	requestID2 := rr2.Header().Get("X-Request-ID")

	assert.NotEmpty(t, requestID1)
	assert.NotEmpty(t, requestID2)
	assert.NotEqual(t, requestID1, requestID2)
	assert.Len(t, requestID1, 32) // 16 bytes = 32 hex chars
	assert.Len(t, requestID2, 32)
}
