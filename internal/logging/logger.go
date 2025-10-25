// Package logging provides structured JSON logging with request ID tracking,
// log levels, and performance metrics.
package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"
)

// LogLevel represents the severity level of a log entry.
type LogLevel string

const (
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
)

// LogEntry represents a structured log entry.
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	RequestID string                 `json:"request_id,omitempty"`
	Service   string                 `json:"service"`
	Component string                 `json:"component,omitempty"`
	Operation string                 `json:"operation,omitempty"`
	Duration  *time.Duration         `json:"duration,omitempty"`
	Error     *ErrorContext          `json:"error,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Source    *SourceLocation        `json:"source,omitempty"`
}

// ErrorContext provides structured error information.
type ErrorContext struct {
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Stack   string `json:"stack,omitempty"`
}

// SourceLocation provides file and line information.
type SourceLocation struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Func string `json:"func"`
}

// Logger provides structured logging capabilities.
type Logger struct {
	service   string
	component string
	output    io.Writer
	level     LogLevel
}

// NewLogger creates a new structured logger.
func NewLogger(service, component string) *Logger {
	return &Logger{
		service:   service,
		component: component,
		output:    os.Stdout,
		level:     LevelInfo,
	}
}

// SetLevel sets the minimum log level.
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// SetOutput sets the output writer.
func (l *Logger) SetOutput(w io.Writer) {
	l.output = w
}

// WithComponent returns a new logger with the specified component.
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		service:   l.service,
		component: component,
		output:    l.output,
		level:     l.level,
	}
}

// Debug logs a debug message.
func (l *Logger) Debug(ctx context.Context, message string, fields ...map[string]interface{}) {
	l.log(ctx, LevelDebug, message, nil, fields...)
}

// Info logs an info message.
func (l *Logger) Info(ctx context.Context, message string, fields ...map[string]interface{}) {
	l.log(ctx, LevelInfo, message, nil, fields...)
}

// Warn logs a warning message.
func (l *Logger) Warn(ctx context.Context, message string, fields ...map[string]interface{}) {
	l.log(ctx, LevelWarn, message, nil, fields...)
}

// Error logs an error message.
func (l *Logger) Error(ctx context.Context, message string, err error, fields ...map[string]interface{}) {
	var errorCtx *ErrorContext
	if err != nil {
		errorCtx = &ErrorContext{
			Type:    fmt.Sprintf("%T", err),
			Message: err.Error(),
		}
	}
	l.log(ctx, LevelError, message, errorCtx, fields...)
}

// LogRequest logs an HTTP request.
func (l *Logger) LogRequest(ctx context.Context, method, path, userAgent string, statusCode int, duration time.Duration, requestID string) {
	fields := map[string]interface{}{
		"method":      method,
		"path":        path,
		"user_agent":  userAgent,
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
	}

	l.log(ctx, LevelInfo, "HTTP request completed", nil, fields)
}

// LogPerformance logs performance metrics.
func (l *Logger) LogPerformance(ctx context.Context, operation string, duration time.Duration, fields ...map[string]interface{}) {
	perfFields := map[string]interface{}{
		"operation":   operation,
		"duration_ms": duration.Milliseconds(),
	}

	// Merge additional fields
	for _, f := range fields {
		for k, v := range f {
			perfFields[k] = v
		}
	}

	l.log(ctx, LevelInfo, "Performance metric", nil, perfFields)
}

// LogSecurity logs security events.
func (l *Logger) LogSecurity(ctx context.Context, event string, fields ...map[string]interface{}) {
	secFields := map[string]interface{}{
		"security_event": event,
	}

	// Merge additional fields
	for _, f := range fields {
		for k, v := range f {
			secFields[k] = v
		}
	}

	l.log(ctx, LevelWarn, "Security event", nil, secFields)
}

// LogAudit logs audit trail events.
func (l *Logger) LogAudit(ctx context.Context, operation, resource string, fields ...map[string]interface{}) {
	auditFields := map[string]interface{}{
		"audit_operation": operation,
		"audit_resource":  resource,
	}

	// Merge additional fields
	for _, f := range fields {
		for k, v := range f {
			auditFields[k] = v
		}
	}

	l.log(ctx, LevelInfo, "Audit trail", nil, auditFields)
}

// log is the internal logging method.
func (l *Logger) log(ctx context.Context, level LogLevel, message string, errorCtx *ErrorContext, fields ...map[string]interface{}) {
	// Check if we should log at this level
	if !l.shouldLog(level) {
		return
	}

	// Get request ID from context
	requestID := getRequestID(ctx)

	// Get source location
	source := l.getSourceLocation()

	// Merge all fields
	allFields := make(map[string]interface{})
	for _, f := range fields {
		for k, v := range f {
			allFields[k] = v
		}
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Message:   message,
		RequestID: requestID,
		Service:   l.service,
		Component: l.component,
		Error:     errorCtx,
		Source:    source,
		Fields:    allFields,
	}

	// Marshal to JSON and write
	jsonData, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple logging if JSON marshaling fails
		fmt.Fprintf(l.output, "LOGGING ERROR: %v\n", err)
		return
	}

	fmt.Fprintf(l.output, "%s\n", jsonData)
}

// shouldLog determines if a message should be logged based on level.
func (l *Logger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		LevelDebug: 0,
		LevelInfo:  1,
		LevelWarn:  2,
		LevelError: 3,
	}
	return levels[level] >= levels[l.level]
}

// getSourceLocation gets the caller's file and line information.
func (l *Logger) getSourceLocation() *SourceLocation {
	_, file, line, ok := runtime.Caller(3) // Skip log, logWithLevel, and the actual log method
	if !ok {
		return nil
	}

	// Get function name
	pc, _, _, ok := runtime.Caller(3)
	if !ok {
		return &SourceLocation{
			File: file,
			Line: line,
		}
	}

	fn := runtime.FuncForPC(pc)
	funcName := "unknown"
	if fn != nil {
		funcName = fn.Name()
	}

	return &SourceLocation{
		File: file,
		Line: line,
		Func: funcName,
	}
}

// getRequestID extracts request ID from context.
func getRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return ""
}

// Global logger instance
var defaultLogger = NewLogger("go-akavelink", "server")

// SetGlobalLevel sets the global log level.
func SetGlobalLevel(level LogLevel) {
	defaultLogger.SetLevel(level)
}

// SetGlobalOutput sets the global output writer.
func SetGlobalOutput(w io.Writer) {
	defaultLogger.SetOutput(w)
}

// Global logging functions
func Debug(ctx context.Context, message string, fields ...map[string]interface{}) {
	defaultLogger.Debug(ctx, message, fields...)
}

func Info(ctx context.Context, message string, fields ...map[string]interface{}) {
	defaultLogger.Info(ctx, message, fields...)
}

func Warn(ctx context.Context, message string, fields ...map[string]interface{}) {
	defaultLogger.Warn(ctx, message, fields...)
}

func Error(ctx context.Context, message string, err error, fields ...map[string]interface{}) {
	defaultLogger.Error(ctx, message, err, fields...)
}

func LogRequest(ctx context.Context, method, path, userAgent string, statusCode int, duration time.Duration, requestID string) {
	defaultLogger.LogRequest(ctx, method, path, userAgent, statusCode, duration, requestID)
}

func LogPerformance(ctx context.Context, operation string, duration time.Duration, fields ...map[string]interface{}) {
	defaultLogger.LogPerformance(ctx, operation, duration, fields...)
}

func LogSecurity(ctx context.Context, event string, fields ...map[string]interface{}) {
	defaultLogger.LogSecurity(ctx, event, fields...)
}

func LogAudit(ctx context.Context, operation, resource string, fields ...map[string]interface{}) {
	defaultLogger.LogAudit(ctx, operation, resource, fields...)
}
