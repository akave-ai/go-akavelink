package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	sdksym "github.com/akave-ai/akavesdk/sdk"
)

// Response defines the standard JSON response envelope.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ClientAPI is the dependency interface required by HTTP handlers.
// It is satisfied by `internal/sdk.Client`.
type ClientAPI interface {
	CreateBucket(ctx context.Context, name string) error
	DeleteBucket(ctx context.Context, name string) error
	ListBuckets() ([]string, error)
	ListFiles(ctx context.Context, bucket string) ([]sdksym.IPCFileListItem, error)

	CreateFileUpload(ctx context.Context, bucket, fileName string) (*sdksym.IPCFileUpload, error)
	Upload(ctx context.Context, up *sdksym.IPCFileUpload, r io.Reader) (sdksym.IPCFileMetaV2, error)

	CreateFileDownload(ctx context.Context, bucket, fileName string) (sdksym.IPCFileDownload, error)
	Download(ctx context.Context, dl sdksym.IPCFileDownload, w io.Writer) error

	FileInfo(ctx context.Context, bucket, fileName string) (sdksym.IPCFileMeta, error)
	FileDelete(ctx context.Context, bucket, fileName string) error

	NewIPC() (*sdksym.IPC, error)
}

// Server encapsulates dependencies for HTTP handlers.
type Server struct {
	client ClientAPI
}

// NewRouter wires all routes and returns a http.Handler you can mount in main.
func NewRouter(client ClientAPI) http.Handler {
	r := mux.NewRouter()
	s := &Server{client: client}

	r.HandleFunc("/health", s.healthHandler).Methods("GET")
	// Buckets
	r.HandleFunc("/buckets/{bucketName}", s.createBucketHandler).Methods("POST")
	r.HandleFunc("/buckets/{bucketName}", s.deleteBucketHandler).Methods("DELETE")
	r.HandleFunc("/buckets/", s.viewBucketHandler).Methods("GET")
	// Files
	r.HandleFunc("/buckets/{bucketName}/files", s.listFilesHandler).Methods("GET")
	r.HandleFunc("/buckets/{bucketName}/files", s.uploadHandler).Methods("POST")
	r.HandleFunc("/buckets/{bucketName}/files/{fileName}", s.fileInfoHandler).Methods("GET")
	r.HandleFunc("/buckets/{bucketName}/files/{fileName}/download", s.downloadHandler).Methods("GET")
	r.HandleFunc("/buckets/{bucketName}/files/{fileName}", s.fileDeleteHandler).Methods("DELETE")

	return r
}

// Handlers

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) createBucketHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	vars := mux.Vars(r)
	bucketName := vars["bucketName"]
	if bucketName == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "bucketName is required")
		return
	}

	if err := s.client.CreateBucket(r.Context(), bucketName); err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create bucket: "+err.Error())
		return
	}

	s.writeSuccessResponse(w, http.StatusCreated, map[string]string{
		"message":    "Bucket created successfully",
		"bucketName": bucketName,
	})
}

func (s *Server) deleteBucketHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	vars := mux.Vars(r)
	bucketName := vars["bucketName"]
	if bucketName == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "bucketName is required")
		return
	}

	ctx := r.Context()
	files, err := s.client.ListFiles(ctx, bucketName)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to list files for deletion: "+err.Error())
		return
	}

	log.Printf("Found %d files to delete.", len(files))
	for _, file := range files {
		ipc, err := s.client.NewIPC()
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create IPC client for deletion: "+err.Error())
			return
		}

		log.Printf("Deleting file: %s from bucket: %s", file.Name, bucketName)
		if err := ipc.FileDelete(ctx, bucketName, file.Name); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete file '"+file.Name+"': "+err.Error())
			return
		}
	}

	log.Printf("Deleting empty bucket: %s", bucketName)
	if err := s.client.DeleteBucket(ctx, bucketName); err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete empty bucket: "+err.Error())
		return
	}

	log.Printf("Successfully deleted bucket and its contents: %s", bucketName)

	s.writeSuccessResponse(w, http.StatusOK, map[string]string{
		"message": "Bucket and all its contents deleted successfully",
	})
}

func (s *Server) fileInfoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucketName"]
	fileName := vars["fileName"]
	if bucketName == "" || fileName == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "bucketName and fileName are required")
		return
	}

	info, err := s.client.FileInfo(r.Context(), bucketName, fileName)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve file info: "+err.Error())
		return
	}

	s.writeSuccessResponse(w, http.StatusOK, info)
}

func (s *Server) listFilesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucketName"]
	if bucketName == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "bucketName is required")
		return
	}

	files, err := s.client.ListFiles(r.Context(), bucketName)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to list files: "+err.Error())
		return
	}

	s.writeSuccessResponse(w, http.StatusOK, files)
}

func (s *Server) viewBucketHandler(w http.ResponseWriter, r *http.Request) {
	buckets, err := s.client.ListBuckets()
	if err != nil {
		http.Error(w, "failed to list buckets: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(Response{Success: true, Data: buckets})
}

func (s *Server) uploadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucketName"]
	if bucketName == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "bucketName is required")
		return
	}

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
	fileUpload, err := s.client.CreateFileUpload(ctx, bucketName, handler.Filename)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create file upload stream")
		return
	}

	finalMetadata, err := s.client.Upload(ctx, fileUpload, file)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to upload file content")
		return
	}

	resp := Response{Success: true, Data: map[string]interface{}{
		"message":     "File uploaded successfully",
		"rootCID":     finalMetadata.RootCID,
		"bucketName":  finalMetadata.BucketName,
		"fileName":    finalMetadata.Name,
		"size":        finalMetadata.Size,
		"encodedSize": finalMetadata.EncodedSize,
		"createdAt":   finalMetadata.CreatedAt,
		"committedAt": finalMetadata.CommittedAt,
	}}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) downloadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucketName"]
	fileName := vars["fileName"]
	if bucketName == "" || fileName == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "bucketName and fileName are required")
		return
	}

	dl, err := s.client.CreateFileDownload(r.Context(), bucketName, fileName)
	if err != nil {
		s.writeErrorResponse(w, http.StatusNotFound, "Failed to create file download: "+err.Error())
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	w.Header().Set("Content-Type", "application/octet-stream")
	if err := s.client.Download(r.Context(), dl, w); err != nil {
		log.Printf("Error during file stream: %v", err)
		return
	}
}

func (s *Server) fileDeleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucketName"]
	fileName := vars["fileName"]
	if bucketName == "" || fileName == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "bucketName and fileName are required")
		return
	}

	if err := s.client.FileDelete(r.Context(), bucketName, fileName); err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete file: "+err.Error())
		return
	}

	s.writeSuccessResponse(w, http.StatusOK, map[string]string{"message": "File deleted successfully"})
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
