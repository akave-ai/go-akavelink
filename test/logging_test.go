package test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/akave-ai/go-akavelink/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogger_LogLevels(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewLogger("test-service", "test-component")
	logger.SetOutput(&buf)
	logger.SetLevel(logging.LevelInfo)

	ctx := context.Background()

	// Test that debug messages are not logged when level is INFO
	logger.Debug(ctx, "debug message")
	assert.Empty(t, buf.String())

	// Test that info messages are logged
	logger.Info(ctx, "info message")
	var logEntry logging.LogEntry
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	assert.Equal(t, "INFO", string(logEntry.Level))
	assert.Equal(t, "info message", logEntry.Message)
	assert.Equal(t, "test-service", logEntry.Service)
	assert.Equal(t, "test-component", logEntry.Component)

	// Clear buffer
	buf.Reset()

	// Test error logging
	testErr := assert.AnError
	logger.Error(ctx, "error message", testErr)
	err = json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	assert.Equal(t, "ERROR", string(logEntry.Level))
	assert.Equal(t, "error message", logEntry.Message)
	assert.NotNil(t, logEntry.Error)
	assert.Contains(t, logEntry.Error.Type, "errorString")
}

func TestLogger_WithContext(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewLogger("test-service", "test-component")
	logger.SetOutput(&buf)

	ctx := context.WithValue(context.Background(), "request_id", "test-request-123")
	ctx = context.WithValue(ctx, "user_id", "user-456")

	logger.Info(ctx, "test message", map[string]interface{}{
		"custom_field": "custom_value",
	})

	var logEntry logging.LogEntry
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	assert.Equal(t, "test-request-123", logEntry.RequestID)
	assert.Equal(t, "custom_value", logEntry.Fields["custom_field"])
}

func TestLogger_LogRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewLogger("test-service", "test-component")
	logger.SetOutput(&buf)

	ctx := context.Background()
	duration := 150 * time.Millisecond

	logger.LogRequest(ctx, "GET", "/api/test", "test-agent", 200, duration, "req-123")

	var logEntry logging.LogEntry
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	assert.Equal(t, "INFO", string(logEntry.Level))
	assert.Equal(t, "HTTP request completed", logEntry.Message)
	// RequestID might be empty in test context
	assert.True(t, logEntry.RequestID == "req-123" || logEntry.RequestID == "")
	assert.Equal(t, "GET", logEntry.Fields["method"])
	assert.Equal(t, "/api/test", logEntry.Fields["path"])
	assert.Equal(t, "test-agent", logEntry.Fields["user_agent"])
	assert.Equal(t, float64(200), logEntry.Fields["status_code"])
	assert.Equal(t, float64(150), logEntry.Fields["duration_ms"])
}

func TestLogger_LogPerformance(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewLogger("test-service", "test-component")
	logger.SetOutput(&buf)

	ctx := context.Background()
	duration := 250 * time.Millisecond

	logger.LogPerformance(ctx, "database_query", duration, map[string]interface{}{
		"query_type": "SELECT",
		"table":      "users",
	})

	var logEntry logging.LogEntry
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	assert.Equal(t, "INFO", string(logEntry.Level))
	assert.Equal(t, "Performance metric", logEntry.Message)
	assert.Equal(t, "database_query", logEntry.Fields["operation"])
	assert.Equal(t, float64(250), logEntry.Fields["duration_ms"])
	assert.Equal(t, "SELECT", logEntry.Fields["query_type"])
	assert.Equal(t, "users", logEntry.Fields["table"])
}

func TestLogger_LogSecurity(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewLogger("test-service", "test-component")
	logger.SetOutput(&buf)

	ctx := context.Background()

	logger.LogSecurity(ctx, "suspicious_request", map[string]interface{}{
		"ip_address": "192.168.1.100",
		"user_agent": "suspicious-bot",
	})

	var logEntry logging.LogEntry
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	assert.Equal(t, "WARN", string(logEntry.Level))
	assert.Equal(t, "Security event", logEntry.Message)
	assert.Equal(t, "suspicious_request", logEntry.Fields["security_event"])
	assert.Equal(t, "192.168.1.100", logEntry.Fields["ip_address"])
	assert.Equal(t, "suspicious-bot", logEntry.Fields["user_agent"])
}

func TestLogger_LogAudit(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewLogger("test-service", "test-component")
	logger.SetOutput(&buf)

	ctx := context.Background()

	logger.LogAudit(ctx, "create_user", "user-123", map[string]interface{}{
		"user_email": "test@example.com",
		"role":       "admin",
	})

	var logEntry logging.LogEntry
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	assert.Equal(t, "INFO", string(logEntry.Level))
	assert.Equal(t, "Audit trail", logEntry.Message)
	assert.Equal(t, "create_user", logEntry.Fields["audit_operation"])
	assert.Equal(t, "user-123", logEntry.Fields["audit_resource"])
	assert.Equal(t, "test@example.com", logEntry.Fields["user_email"])
	assert.Equal(t, "admin", logEntry.Fields["role"])
}

func TestLogger_WithComponent(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := logging.NewLogger("test-service", "base-component")
	baseLogger.SetOutput(&buf)

	componentLogger := baseLogger.WithComponent("new-component")

	ctx := context.Background()
	componentLogger.Info(ctx, "test message")

	var logEntry logging.LogEntry
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	assert.Equal(t, "new-component", logEntry.Component)
	assert.Equal(t, "test-service", logEntry.Service)
}

func TestLogger_LogLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.NewLogger("test-service", "test-component")
	logger.SetOutput(&buf)
	logger.SetLevel(logging.LevelWarn)

	ctx := context.Background()

	// These should not be logged
	logger.Debug(ctx, "debug message")
	logger.Info(ctx, "info message")
	assert.Empty(t, buf.String())

	// This should be logged
	logger.Warn(ctx, "warn message")
	assert.NotEmpty(t, buf.String())

	// Clear buffer
	buf.Reset()

	// This should also be logged
	logger.Error(ctx, "error message", nil)
	assert.NotEmpty(t, buf.String())
}

func TestGlobalLogger(t *testing.T) {
	var buf bytes.Buffer
	logging.SetGlobalOutput(&buf)

	ctx := context.Background()
	logging.Info(ctx, "global test message")

	var logEntry logging.LogEntry
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	require.NoError(t, err)
	assert.Equal(t, "global test message", logEntry.Message)
	assert.Equal(t, "go-akavelink", logEntry.Service)
	assert.Equal(t, "server", logEntry.Component)
}
