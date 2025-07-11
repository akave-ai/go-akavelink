package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/akave-ai/go-akavelink/internal/sdk"
)

// AkaveResponse matches the standard response format
type AkaveResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// server holds the application dependencies
type server struct {
	client *sdk.Client
}

func (s *server) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{"status": "ok"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *server) bucketsHandler(w http.ResponseWriter, r *http.Request) {
	buckets, err := s.client.ListBuckets()
	if err != nil {
		log.Printf("Failed to list buckets: %v", err)
		s.writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list buckets: %v", err))
		return
	}

	// Return response in Akave format: { success: true, data: [...] }
	response := AkaveResponse{
		Success: true,
		Data:    buckets,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to encode response")
		return
	}
}

func (s *server) uploadHandler(w http.ResponseWriter, r *http.Request) {
	bucketName := r.PathValue("bucketName")

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Failed to parse multipart form: "+err.Error())
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Failed to retrieve file from form: "+err.Error())
		return
	}
	defer file.Close()

	ctx := r.Context()
	fileName := handler.Filename

	log.Printf("Initiating upload for '%s' to bucket '%s'", fileName, bucketName)
	fileUpload, err := s.client.CreateFileUpload(ctx, bucketName, fileName)
	if err != nil {
		log.Printf("Error creating file upload stream: %v", err)
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create file upload stream")
		return
	}

	log.Printf("Uploading content for file: %s", fileName)
	finalMetadata, err := s.client.Upload(ctx, fileUpload, file)
	if err != nil {
		log.Printf("Error uploading file content: %v", err)
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to upload file content")
		return
	}

	response := AkaveResponse{
		Success: true,
		Data: map[string]interface{}{
			"message":     "File uploaded successfully",
			"rootCID":     finalMetadata.RootCID,
			"bucketName":  finalMetadata.BucketName,
			"fileName":    finalMetadata.Name,
			"size":        finalMetadata.Size,
			"encodedSize": finalMetadata.EncodedSize,
			"createdAt":   finalMetadata.CreatedAt,
			"committedAt": finalMetadata.CommittedAt,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
	log.Printf("Successfully uploaded file '%s' with Root CID: %s", finalMetadata.Name, finalMetadata.RootCID)
}

func (s *server) downloadHandler(w http.ResponseWriter, r *http.Request) {
	bucketName := r.PathValue("bucketName")
	fileName := r.PathValue("fileName")

	ctx := r.Context()
	fileDownload, err := s.client.CreateFileDownload(ctx, bucketName, fileName)
	if err != nil {
		log.Printf("Error: Failed to create file download: %v", err)
		s.writeErrorResponse(w, http.StatusInternalServerError, "failed to create file download")
		return
	}

	if err := s.client.Download(ctx, fileDownload, w); err != nil {
		log.Printf("Error: Failed to complete file download: %v", err)
		return
	}
	log.Printf("Successfully downloaded: %s/%s", bucketName, fileName)
}

func (s *server) writeErrorResponse(w http.ResponseWriter, statusCode int, errorMsg string) {
	response := AkaveResponse{
		Success: false,
		Error:   errorMsg,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Get Akave configuration from environment
	nodeAddress := os.Getenv("AKAVE_NODE_ADDRESS")
	if nodeAddress == "" {
		nodeAddress = os.Getenv("NODE_ADDRESS") // fallback for backward compatibility
	}

	privateKey := os.Getenv("AKAVE_PRIVATE_KEY")
	if privateKey == "" {
		privateKey = os.Getenv("PRIVATE_KEY") // fallback for backward compatibility
	}

	if privateKey == "" {
		log.Fatal("AKAVE_PRIVATE_KEY (or PRIVATE_KEY) environment variable is required")
	}

	if nodeAddress == "" {
		log.Fatal("AKAVE_NODE_ADDRESS (or NODE_ADDRESS) environment variable is required")
	}

	// Log initialization info
	log.Printf("Initializing client with nodeAddress: %s, privateKeyLength: %d",
		nodeAddress, len(privateKey))

	// Initialize Akave SDK client
	client, err := sdk.NewClientSimple(nodeAddress, privateKey)
	if err != nil {
		log.Fatalf("Failed to initialize Akave client: %v", err)
	}
	defer func() {
		log.Println("Closing Akave SDK connection...")
		if closeErr := client.Close(); closeErr != nil {
			log.Printf("Error closing Akave SDK connection: %v", closeErr)
		} else {
			log.Println("Akave SDK connection closed successfully.")
		}
	}()

	// Create server instance
	srv := &server{
		client: client,
	}

	// Set up routes using Go 1.22+ pattern matching
	mux := http.NewServeMux()
	mux.HandleFunc("/health", srv.healthHandler)
	mux.HandleFunc("GET /buckets", srv.bucketsHandler)
	mux.HandleFunc("POST /files/upload/{bucketName}", srv.uploadHandler)
	mux.HandleFunc("GET /files/download/{bucketName}/{fileName}", srv.downloadHandler)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), mux))
}
