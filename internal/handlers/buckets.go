// Package handlers provides HTTP routing and request handlers for the AkaveLink API.
package handlers

import (
	"log"
	"net/http"

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
		log.Printf("create bucket error: %v", err)
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
		log.Printf("list files before delete bucket error: %v", err)
		return
	}

	log.Printf("Found %d files to delete.", len(files))
	for _, file := range files {
		ipc, err := s.client.NewIPC()
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create IPC client for deletion")
			log.Printf("new IPC error: %v", err)
			return
		}

		log.Printf("Deleting file: %s from bucket: %s", file.Name, bucketName)
		if err := ipc.FileDelete(ctx, bucketName, file.Name); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete file")
			log.Printf("file delete error for %s: %v", file.Name, err)
			return
		}
	}

	log.Printf("Deleting empty bucket: %s", bucketName)
	if err := s.client.DeleteBucket(ctx, bucketName); err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete empty bucket")
		log.Printf("delete bucket error: %v", err)
		return
	}

	log.Printf("Successfully deleted bucket and its contents: %s", bucketName)
	s.writeSuccessResponse(w, http.StatusOK, map[string]string{
		"message": "Bucket and all its contents deleted successfully",
	})
}

// viewBucketHandler lists all buckets.
func (s *Server) viewBucketHandler(w http.ResponseWriter, _ *http.Request) {
	buckets, err := s.client.ListBuckets()
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "failed to list buckets")
		log.Printf("list buckets error: %v", err)
		return
	}
	s.writeSuccessResponse(w, http.StatusOK, buckets)
}
