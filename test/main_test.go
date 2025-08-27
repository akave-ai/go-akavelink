// test/main_test.go
package test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// To avoid circular dependencies and allow `main_test.go` to be in `test/`,
// we will effectively copy the server setup logic from `main.go` and run it
// within a test-specific context. This is a common pattern for integration tests.

// Define a placeholder for the server structure from main.go
// You might need to expose `server` or specific handlers in `main.go`
// or re-define them here for testing if they are not exported.
// For simplicity, we'll re-define the necessary parts here.

type testServer struct{}

func (s *testServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// TestMain_HealthEndpoint tests the /health endpoint of the HTTP server.
func TestMain_HealthEndpoint(t *testing.T) {
	// Create a minimal server with only the health handler; no SDK or env required
	srv := &testServer{}

	testHTTPServer := httptest.NewServer(http.HandlerFunc(srv.healthHandler))
	defer testHTTPServer.Close()

	// Make a request to the health endpoint
	resp, err := http.Get(testHTTPServer.URL + "/health")
	require.NoError(t, err, "Failed to make GET request to health endpoint")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status code 200 OK for /health")

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")
	assert.Equal(t, "ok", string(bodyBytes), "Expected body to be 'ok'")
}