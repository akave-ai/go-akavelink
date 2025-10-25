package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/akave-ai/go-akavelink/internal/errors"
	"github.com/akave-ai/go-akavelink/internal/logging"
	"github.com/akave-ai/go-akavelink/internal/middleware"
)

// Response defines the standard JSON response envelope.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// SuccessResponse defines the standard success response.
type SuccessResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp string     `json:"timestamp,omitempty"`
}

// Helpers
func (s *Server) writeSuccessResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	ctx := context.Background()
	requestID := middleware.GetRequestID(ctx)
	
	response := SuccessResponse{
		Success:   true,
		Data:      data,
		RequestID: requestID,
		Timestamp: fmt.Sprintf("%d", time.Now().Unix()),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(response)
}

func (s *Server) writeErrorResponse(w http.ResponseWriter, statusCode int, errorMsg string) {
	response := Response{
		Success: false,
		Error:   errorMsg,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(response)
}

// writeServiceErrorResponse writes a structured error response from a ServiceError.
func (s *Server) writeServiceErrorResponse(w http.ResponseWriter, serviceErr *errors.ServiceError) {
	ctx := context.Background()
	requestID := middleware.GetRequestID(ctx)
	
	// Log the error
	logging.Error(ctx, "Service error occurred", serviceErr, map[string]interface{}{
		"error_code": serviceErr.Code,
		"error_type": serviceErr.Type,
		"http_status": serviceErr.HTTPStatus,
		"request_id": requestID,
	})
	
	// Convert to error response
	errorResponse := serviceErr.ToErrorResponse(requestID)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(serviceErr.HTTPStatus)
	_ = json.NewEncoder(w).Encode(errorResponse)
}

// handleError handles errors and writes appropriate responses.
func (s *Server) handleError(w http.ResponseWriter, r *http.Request, err error, operation string) {
	ctx := r.Context()
	requestID := middleware.GetRequestID(ctx)
	
	// Check if it's already a ServiceError
	if serviceErr, ok := err.(*errors.ServiceError); ok {
		s.writeServiceErrorResponse(w, serviceErr)
		return
	}
	
	// Wrap generic error
	serviceErr := errors.WrapError(err, errors.ErrCodeInternalError, "Internal server error")
	serviceErr = serviceErr.WithContext("operation", operation)
	
	// Log the error
	logging.Error(ctx, "Handler error", serviceErr, map[string]interface{}{
		"operation": operation,
		"request_id": requestID,
		"method": r.Method,
		"path": r.URL.Path,
	})
	
	s.writeServiceErrorResponse(w, serviceErr)
}
