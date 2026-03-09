// Package handlers provides HTTP routing and request handlers for the AkaveLink API.
package handlers

import (
	"net/http"

	"github.com/akave-ai/go-akavelink/internal/logger"
	"github.com/gorilla/mux"
)

// createBucketHandler creates a new bucket.
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
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create bucket")
		logger.Error("create bucket failed", "bucket", bucketName, "error", err)
		return
	}

	s.writeSuccessResponse(w, http.StatusCreated, map[string]string{
		"message":    "Bucket created successfully",
		"bucketName": bucketName,
	})
}

// deleteBucketHandler deletes all files in a bucket and then the bucket itself.
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
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to list files for deletion")
		logger.Error("list files before delete bucket failed", "bucket", bucketName, "error", err)
		return
	}

	logger.Info("deleting bucket contents", "bucket", bucketName, "file_count", len(files))
	for _, file := range files {
		ipc, err := s.client.NewIPC()
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create IPC client for deletion")
			logger.Error("new IPC failed during bucket delete", "error", err)
			return
		}

		logger.Debug("deleting file from bucket", "bucket", bucketName, "file", file.Name)
		if err := ipc.FileDelete(ctx, bucketName, file.Name); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete file")
			logger.Error("file delete failed", "bucket", bucketName, "file", file.Name, "error", err)
			return
		}
	}

	logger.Info("deleting empty bucket", "bucket", bucketName)
	if err := s.client.DeleteBucket(ctx, bucketName); err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete empty bucket")
		logger.Error("delete bucket failed", "bucket", bucketName, "error", err)
		return
	}

	logger.Info("bucket and contents deleted", "bucket", bucketName)
	s.writeSuccessResponse(w, http.StatusOK, map[string]string{
		"message": "Bucket and all its contents deleted successfully",
	})
}

// viewBucketHandler lists all buckets.
func (s *Server) viewBucketHandler(w http.ResponseWriter, _ *http.Request) {
	buckets, err := s.client.ListBuckets()
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "failed to list buckets")
		logger.Error("list buckets failed", "error", err)
		return
	}
	s.writeSuccessResponse(w, http.StatusOK, buckets)
}
