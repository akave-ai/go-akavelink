package handlers

import (
	"net/http"
)

// healthHandler returns a simple JSON response indicating the server is healthy.
func (s *Server) healthHandler(w http.ResponseWriter, _ *http.Request) {
	s.writeSuccessResponse(w, http.StatusOK, "ok")
}
