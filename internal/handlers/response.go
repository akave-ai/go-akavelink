package handlers

import (
	"encoding/json"
	"net/http"
)

// Response defines the standard JSON response envelope.
// Moved from router.go to keep router focused on wiring.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Helpers
func (s *Server) writeSuccessResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(Response{Success: true, Data: data})
}

func (s *Server) writeErrorResponse(w http.ResponseWriter, statusCode int, errorMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(Response{Success: false, Error: errorMsg})
}
