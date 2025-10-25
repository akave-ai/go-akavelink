// Package handlers provides HTTP routing and request handlers for the AkaveLink API.
package handlers

import (
	"net/http"
	"time"

	"regexp"

	"github.com/akave-ai/go-akavelink/internal/errors"
	"github.com/akave-ai/go-akavelink/internal/logging"
	"github.com/akave-ai/go-akavelink/internal/middleware"
	"github.com/gorilla/mux"
)

// isValidBucketName validates bucket name format.
func isValidBucketName(name string) bool {
	// Bucket name should be 3-63 characters, alphanumeric and hyphens only
	if len(name) < 3 || len(name) > 63 {
		return false
	}
	
	// Should start and end with alphanumeric
	if !isAlphanumeric(name[0]) || !isAlphanumeric(name[len(name)-1]) {
		return false
	}
	
	// Should only contain alphanumeric and hyphens
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9-]+$`, name)
	return matched
}

// isAlphanumeric checks if a character is alphanumeric.
func isAlphanumeric(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// createBucketHandler creates a new bucket.
func (s *Server) createBucketHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetRequestID(ctx)
	start := time.Now()
	
	logging.Info(ctx, "Creating bucket", map[string]interface{}{
		"request_id": requestID,
		"operation": "create_bucket",
	})

	vars := mux.Vars(r)
	bucketName := vars["bucketName"]
	if bucketName == "" {
		err := errors.NewValidationError("bucketName is required", "bucketName parameter is missing")
		s.writeServiceErrorResponse(w, err)
		return
	}

	// Validate bucket name
	if !isValidBucketName(bucketName) {
		err := errors.NewValidationError("Invalid bucket name", "bucket name contains invalid characters")
		err = err.WithContext("bucket_name", bucketName)
		s.writeServiceErrorResponse(w, err)
		return
	}

	// Log audit trail
	logging.LogAudit(ctx, "create_bucket", bucketName, map[string]interface{}{
		"bucket_name": bucketName,
		"request_id": requestID,
	})

	if err := s.client.CreateBucket(ctx, bucketName); err != nil {
		serviceErr := errors.NewStorageError("create_bucket", err)
		serviceErr = serviceErr.WithContext("bucket_name", bucketName)
		s.writeServiceErrorResponse(w, serviceErr)
		return
	}

	// Log performance
	duration := time.Since(start)
	logging.LogPerformance(ctx, "create_bucket", duration, map[string]interface{}{
		"bucket_name": bucketName,
		"request_id": requestID,
	})

	logging.Info(ctx, "Bucket created successfully", map[string]interface{}{
		"bucket_name": bucketName,
		"request_id": requestID,
		"duration_ms": duration.Milliseconds(),
	})

	s.writeSuccessResponse(w, http.StatusCreated, map[string]string{
		"message":    "Bucket created successfully",
		"bucketName": bucketName,
	})
}

// deleteBucketHandler deletes all files in a bucket and then the bucket itself.
func (s *Server) deleteBucketHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetRequestID(ctx)
	start := time.Now()
	
	logging.Info(ctx, "Deleting bucket", map[string]interface{}{
		"request_id": requestID,
		"operation": "delete_bucket",
	})

	vars := mux.Vars(r)
	bucketName := vars["bucketName"]
	if bucketName == "" {
		err := errors.NewValidationError("bucketName is required", "bucketName parameter is missing")
		s.writeServiceErrorResponse(w, err)
		return
	}

	// Log audit trail
	logging.LogAudit(ctx, "delete_bucket", bucketName, map[string]interface{}{
		"bucket_name": bucketName,
		"request_id": requestID,
	})

	files, err := s.client.ListFiles(ctx, bucketName)
	if err != nil {
		serviceErr := errors.NewStorageError("list_files", err)
		serviceErr = serviceErr.WithContext("bucket_name", bucketName)
		s.writeServiceErrorResponse(w, serviceErr)
		return
	}

	logging.Info(ctx, "Found files to delete", map[string]interface{}{
		"bucket_name": bucketName,
		"file_count": len(files),
		"request_id": requestID,
	})

	for _, file := range files {
		ipc, err := s.client.NewIPC()
		if err != nil {
			serviceErr := errors.NewInternalError("Failed to create IPC client for deletion", err)
			serviceErr = serviceErr.WithContext("bucket_name", bucketName)
			serviceErr = serviceErr.WithContext("file_name", file.Name)
			s.writeServiceErrorResponse(w, serviceErr)
			return
		}

		logging.Info(ctx, "Deleting file", map[string]interface{}{
			"bucket_name": bucketName,
			"file_name": file.Name,
			"request_id": requestID,
		})

		if err := ipc.FileDelete(ctx, bucketName, file.Name); err != nil {
			serviceErr := errors.NewStorageError("delete_file", err)
			serviceErr = serviceErr.WithContext("bucket_name", bucketName)
			serviceErr = serviceErr.WithContext("file_name", file.Name)
			s.writeServiceErrorResponse(w, serviceErr)
			return
		}
	}

	logging.Info(ctx, "Deleting empty bucket", map[string]interface{}{
		"bucket_name": bucketName,
		"request_id": requestID,
	})

	if err := s.client.DeleteBucket(ctx, bucketName); err != nil {
		serviceErr := errors.NewStorageError("delete_bucket", err)
		serviceErr = serviceErr.WithContext("bucket_name", bucketName)
		s.writeServiceErrorResponse(w, serviceErr)
		return
	}

	// Log performance
	duration := time.Since(start)
	logging.LogPerformance(ctx, "delete_bucket", duration, map[string]interface{}{
		"bucket_name": bucketName,
		"file_count": len(files),
		"request_id": requestID,
	})

	logging.Info(ctx, "Bucket deleted successfully", map[string]interface{}{
		"bucket_name": bucketName,
		"file_count": len(files),
		"request_id": requestID,
		"duration_ms": duration.Milliseconds(),
	})

	s.writeSuccessResponse(w, http.StatusOK, map[string]string{
		"message": "Bucket and all its contents deleted successfully",
	})
}

// viewBucketHandler lists all buckets.
func (s *Server) viewBucketHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetRequestID(ctx)
	start := time.Now()
	
	logging.Info(ctx, "Listing buckets", map[string]interface{}{
		"request_id": requestID,
		"operation": "list_buckets",
	})

	buckets, err := s.client.ListBuckets()
	if err != nil {
		serviceErr := errors.NewStorageError("list_buckets", err)
		s.writeServiceErrorResponse(w, serviceErr)
		return
	}

	// Log performance
	duration := time.Since(start)
	logging.LogPerformance(ctx, "list_buckets", duration, map[string]interface{}{
		"bucket_count": len(buckets),
		"request_id": requestID,
	})

	logging.Info(ctx, "Buckets listed successfully", map[string]interface{}{
		"bucket_count": len(buckets),
		"request_id": requestID,
		"duration_ms": duration.Milliseconds(),
	})

	s.writeSuccessResponse(w, http.StatusOK, buckets)
}
