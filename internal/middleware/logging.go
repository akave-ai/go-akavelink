// Package middleware provides HTTP middleware for logging and request handling.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/akave-ai/go-akavelink/internal/logging"
)

// RequestIDKey is the context key for request ID.
const RequestIDKey = "request_id"

// LoggingMiddleware provides structured logging for HTTP requests.
type LoggingMiddleware struct {
	logger *logging.Logger
}

// NewLoggingMiddleware creates a new logging middleware.
func NewLoggingMiddleware(service string) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: logging.NewLogger(service, "middleware"),
	}
}

// SetOutput sets the output writer for the logger.
func (lm *LoggingMiddleware) SetOutput(w io.Writer) {
	lm.logger.SetOutput(w)
}

// LoggingHandler wraps an HTTP handler with structured logging.
func (lm *LoggingMiddleware) LoggingHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Generate request ID
		requestID := generateRequestID()
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

		// Add request ID to response headers
		w.Header().Set("X-Request-ID", requestID)

		// Create a response writer wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Log request start
		lm.logger.Info(ctx, "HTTP request started", map[string]interface{}{
			"method":      r.Method,
			"path":        r.URL.Path,
			"query":       r.URL.RawQuery,
			"user_agent":  r.UserAgent(),
			"remote_addr": getClientIP(r),
			"request_id":  requestID,
		})

		// Process request
		next.ServeHTTP(rw, r.WithContext(ctx))

		// Calculate duration
		duration := time.Since(start)

		// Log request completion
		lm.logger.LogRequest(ctx, r.Method, r.URL.Path, r.UserAgent(), rw.statusCode, duration, requestID)

		// Log performance metrics
		lm.logger.LogPerformance(ctx, "http_request", duration, map[string]interface{}{
			"method":      r.Method,
			"path":        r.URL.Path,
			"status_code": rw.statusCode,
			"request_id":  requestID,
		})
	})
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// generateRequestID generates a unique request ID.
func generateRequestID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// getClientIP extracts the client IP address from the request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}
	return ip
}

// SecurityMiddleware provides security event logging.
type SecurityMiddleware struct {
	logger *logging.Logger
}

// NewSecurityMiddleware creates a new security middleware.
func NewSecurityMiddleware(service string) *SecurityMiddleware {
	return &SecurityMiddleware{
		logger: logging.NewLogger(service, "security"),
	}
}

// SetOutput sets the output writer for the logger.
func (sm *SecurityMiddleware) SetOutput(w io.Writer) {
	sm.logger.SetOutput(w)
}

// SecurityHandler wraps an HTTP handler with security logging.
func (sm *SecurityMiddleware) SecurityHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Log suspicious patterns
		if isSuspiciousRequest(r) {
			sm.logger.LogSecurity(ctx, "suspicious_request", map[string]interface{}{
				"method":      r.Method,
				"path":        r.URL.Path,
				"user_agent":  r.UserAgent(),
				"remote_addr": getClientIP(r),
				"headers":     r.Header,
			})
		}

		// Log authentication attempts
		if r.URL.Path == "/auth" || r.URL.Path == "/login" {
			sm.logger.LogSecurity(ctx, "authentication_attempt", map[string]interface{}{
				"method":      r.Method,
				"path":        r.URL.Path,
				"remote_addr": getClientIP(r),
			})
		}

		next.ServeHTTP(w, r)
	})
}

// isSuspiciousRequest checks for suspicious request patterns.
func isSuspiciousRequest(r *http.Request) bool {
	// Check for common attack patterns
	path := strings.ToLower(r.URL.Path)
	suspiciousPatterns := []string{
		"../", "..\\", "script", "javascript:", "vbscript:",
		"<script", "eval(", "exec(", "system(", "cmd",
		"admin", "root", "password", "passwd",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}

	// Check for unusual user agents
	userAgent := strings.ToLower(r.UserAgent())
	if userAgent == "" || strings.Contains(userAgent, "bot") || strings.Contains(userAgent, "crawler") {
		return true
	}

	return false
}

// AuditMiddleware provides audit trail logging.
type AuditMiddleware struct {
	logger *logging.Logger
}

// NewAuditMiddleware creates a new audit middleware.
func NewAuditMiddleware(service string) *AuditMiddleware {
	return &AuditMiddleware{
		logger: logging.NewLogger(service, "audit"),
	}
}

// SetOutput sets the output writer for the logger.
func (am *AuditMiddleware) SetOutput(w io.Writer) {
	am.logger.SetOutput(w)
}

// AuditHandler wraps an HTTP handler with audit logging.
func (am *AuditMiddleware) AuditHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Log data modification operations
		if isDataModification(r) {
			am.logger.LogAudit(ctx, "data_modification", r.URL.Path, map[string]interface{}{
				"method":      r.Method,
				"path":        r.URL.Path,
				"remote_addr": getClientIP(r),
				"user_agent":  r.UserAgent(),
			})
		}

		next.ServeHTTP(w, r)
	})
}

// isDataModification checks if the request modifies data.
func isDataModification(r *http.Request) bool {
	return r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" || r.Method == "DELETE"
}

// GetRequestID extracts the request ID from context.
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}
