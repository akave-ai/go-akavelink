// Package handlers provides CORS middleware configurable via allowed origins.
package handlers

import (
	"net/http"
	"strings"
)

// DefaultCORSOrigins is the default value when CORS_ORIGINS is not set (allow all).
const DefaultCORSOrigins = "*"

// allowedOrigins parses a comma-separated list; empty or "*" means allow all.
type corsMiddleware struct {
	origins map[string]bool // empty means allow all (*)
	next    http.Handler
}

// CORSMiddleware returns a handler that adds CORS headers and handles OPTIONS preflight.
// allowedOrigins: use DefaultCORSOrigins ("*") or a comma-separated list, e.g. "https://app.example.com,https://admin.example.com".
// If empty, "*" is used.
func CORSMiddleware(allowedOrigins string, next http.Handler) http.Handler {
	origins := parseCORSOrigins(allowedOrigins)
	return &corsMiddleware{origins: origins, next: next}
}

func parseCORSOrigins(s string) map[string]bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil // nil means allow all
	}
	if s == "*" {
		return nil
	}
	out := make(map[string]bool)
	for _, o := range strings.Split(s, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			out[o] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (c *corsMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	allowOrigin := c.allowOrigin(origin)

	if allowOrigin != "" {
		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept")
	w.Header().Set("Access-Control-Max-Age", "86400")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	c.next.ServeHTTP(w, r)
}

func (c *corsMiddleware) allowOrigin(requestOrigin string) string {
	if c.origins == nil {
		return "*"
	}
	if requestOrigin != "" && c.origins[requestOrigin] {
		return requestOrigin
	}
	return ""
}
