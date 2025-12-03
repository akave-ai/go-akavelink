// Package middleware provides HTTP middleware for the AkaveLink API.
package middleware

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/akave-ai/go-akavelink/internal/validation"
	"github.com/gorilla/mux"
)

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Field   string `json:"field,omitempty"`
}

// writeValidationError writes a validation error response.
func writeValidationError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	response := ErrorResponse{
		Error: "Validation Error",
	}

	if validationErr, ok := err.(*validation.ValidationError); ok {
		response.Field = validationErr.Field
		response.Message = validationErr.Message
	} else {
		response.Message = err.Error()
	}

	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		log.Printf("Error encoding validation error response: %v", encodeErr)
	}
}

// ValidateBucketName is a middleware that validates bucket name from URL parameters.
func ValidateBucketName(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		bucketName := vars["bucketName"]

		if err := validation.ValidateBucketName(bucketName); err != nil {
			writeValidationError(w, err)
			return
		}

		// Sanitize the bucket name for extra safety
		sanitized := validation.SanitizeBucketName(bucketName)
		if sanitized != bucketName {
			log.Printf("Warning: Bucket name was sanitized from '%s' to '%s'", bucketName, sanitized)
		}

		next.ServeHTTP(w, r)
	})
}

// ValidateFileName is a middleware that validates file name from URL parameters.
func ValidateFileName(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		fileName := vars["fileName"]

		if err := validation.ValidateFileName(fileName); err != nil {
			writeValidationError(w, err)
			return
		}

		// Sanitize the file name for extra safety
		sanitized := validation.SanitizeFileName(fileName)
		if sanitized != fileName {
			log.Printf("Warning: File name was sanitized from '%s' to '%s'", fileName, sanitized)
		}

		next.ServeHTTP(w, r)
	})
}

// ValidateBucketAndFileName is a middleware that validates both bucket and file names.
func ValidateBucketAndFileName(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		bucketName := vars["bucketName"]
		fileName := vars["fileName"]

		// Validate bucket name
		if err := validation.ValidateBucketName(bucketName); err != nil {
			writeValidationError(w, err)
			return
		}

		// Validate file name
		if err := validation.ValidateFileName(fileName); err != nil {
			writeValidationError(w, err)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ValidateContentLength is a middleware that validates Content-Length header.
func ValidateContentLength(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentLength := r.ContentLength

		if err := validation.ValidateContentLength(contentLength); err != nil {
			writeValidationError(w, err)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RateLimitByIP is a placeholder for rate limiting middleware.
// This can be implemented with a proper rate limiting library like golang.org/x/time/rate
func RateLimitByIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement rate limiting based on IP address
		// For now, just pass through
		next.ServeHTTP(w, r)
	})
}

// SecurityHeaders adds security-related HTTP headers.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Enable XSS protection
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Enforce HTTPS (if applicable)
		// w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Content Security Policy
		w.Header().Set("Content-Security-Policy", "default-src 'self'")

		next.ServeHTTP(w, r)
	})
}

// LogRequest logs incoming HTTP requests.
func LogRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.Method, r.RequestURI, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
