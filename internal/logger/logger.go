// Package logger provides structured logging (log/slog) and HTTP request logging.
// Configure via Init() from main; LOG_LEVEL and LOG_FORMAT env vars are supported.
package logger

import (
	"log/slog"
	"net/http"
	"os"
	"time"
)

var defaultLogger *slog.Logger

// Init configures the global logger. level: debug|info|warn|error; format: json|text.
// Safe to call once at startup. If level or format is empty, defaults to "info" and "json".
func Init(level, format string) {
	if level == "" {
		level = "info"
	}
	if format == "" {
		format = "json"
	}

	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: lvl, AddSource: false}
	if format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	defaultLogger = slog.New(handler)
}

// L returns the default logger. If Init was not called, returns slog.Default().
func L() *slog.Logger {
	if defaultLogger != nil {
		return defaultLogger
	}
	return slog.Default()
}

// Debug logs at LevelDebug.
func Debug(msg string, args ...any) { L().Debug(msg, args...) }

// Info logs at LevelInfo.
func Info(msg string, args ...any) { L().Info(msg, args...) }

// Warn logs at LevelWarn.
func Warn(msg string, args ...any) { L().Warn(msg, args...) }

// Error logs at LevelError.
func Error(msg string, args ...any) { L().Error(msg, args...) }

// responseWriter wraps http.ResponseWriter to capture status and bytes written.
type responseWriter struct {
	http.ResponseWriter
	status int
	written int64
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.written += int64(n)
	return n, err
}

// Middleware returns an http.Handler that logs each request: method, path, status, duration, client IP.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		dur := time.Since(start)
		clientIP := r.Header.Get("X-Forwarded-For")
		if clientIP == "" {
			clientIP = r.RemoteAddr
		}
		L().Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.status,
			"duration_ms", dur.Milliseconds(),
			"client_ip", clientIP,
			"bytes_out", wrapped.written,
		)
	})
}
