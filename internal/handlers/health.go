package handlers

import (
	"net/http"
	"time"

	"github.com/akave-ai/go-akavelink/internal/logging"
	"github.com/akave-ai/go-akavelink/internal/middleware"
)

// healthHandler returns a simple JSON response indicating the server is healthy.
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetRequestID(ctx)
	
	// Log health check
	logging.Info(ctx, "Health check requested", map[string]interface{}{
		"request_id": requestID,
		"endpoint": "health",
	})
	
	start := time.Now()
	
	// Perform health checks
	healthData := map[string]interface{}{
		"status": "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service": "go-akavelink",
		"version": "1.0.0", // This could be set from build info
	}
	
	// Log performance
	duration := time.Since(start)
	logging.LogPerformance(ctx, "health_check", duration, map[string]interface{}{
		"request_id": requestID,
	})
	
	s.writeSuccessResponse(w, http.StatusOK, healthData)
}
