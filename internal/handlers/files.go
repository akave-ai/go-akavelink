package handlers

import (
	"net/http"
	"time"

	"github.com/akave-ai/go-akavelink/internal/errors"
	"github.com/akave-ai/go-akavelink/internal/logging"
	"github.com/akave-ai/go-akavelink/internal/middleware"
	"github.com/gorilla/mux"
)

// fileInfoHandler returns metadata for a specific file.
func (s *Server) fileInfoHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetRequestID(ctx)
	start := time.Now()

	vars := mux.Vars(r)
	bucketName := vars["bucketName"]
	fileName := vars["fileName"]

	if bucketName == "" || fileName == "" {
		err := errors.NewValidationError("bucketName and fileName are required", "both bucketName and fileName parameters are missing")
		s.writeServiceErrorResponse(w, err)
		return
	}

	logging.Info(ctx, "Getting file info", map[string]interface{}{
		"bucket_name": bucketName,
		"file_name":   fileName,
		"request_id":  requestID,
		"operation":   "get_file_info",
	})

	info, err := s.client.FileInfo(ctx, bucketName, fileName)
	if err != nil {
		serviceErr := errors.NewStorageError("get_file_info", err)
		serviceErr = serviceErr.WithContext("bucket_name", bucketName)
		serviceErr = serviceErr.WithContext("file_name", fileName)
		s.writeServiceErrorResponse(w, serviceErr)
		return
	}

	// Log performance
	duration := time.Since(start)
	logging.LogPerformance(ctx, "get_file_info", duration, map[string]interface{}{
		"bucket_name": bucketName,
		"file_name":   fileName,
		"request_id":  requestID,
	})

	logging.Info(ctx, "File info retrieved successfully", map[string]interface{}{
		"bucket_name": bucketName,
		"file_name":   fileName,
		"request_id":  requestID,
		"duration_ms": duration.Milliseconds(),
	})

	s.writeSuccessResponse(w, http.StatusOK, info)
}

// listFilesHandler lists files in a bucket.
func (s *Server) listFilesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetRequestID(ctx)
	start := time.Now()

	vars := mux.Vars(r)
	bucketName := vars["bucketName"]
	if bucketName == "" {
		err := errors.NewValidationError("bucketName is required", "bucketName parameter is missing")
		s.writeServiceErrorResponse(w, err)
		return
	}

	logging.Info(ctx, "Listing files", map[string]interface{}{
		"bucket_name": bucketName,
		"request_id":  requestID,
		"operation":   "list_files",
	})

	files, err := s.client.ListFiles(ctx, bucketName)
	if err != nil {
		serviceErr := errors.NewStorageError("list_files", err)
		serviceErr = serviceErr.WithContext("bucket_name", bucketName)
		s.writeServiceErrorResponse(w, serviceErr)
		return
	}

	// Log performance
	duration := time.Since(start)
	logging.LogPerformance(ctx, "list_files", duration, map[string]interface{}{
		"bucket_name": bucketName,
		"file_count":  len(files),
		"request_id":  requestID,
	})

	logging.Info(ctx, "Files listed successfully", map[string]interface{}{
		"bucket_name": bucketName,
		"file_count":  len(files),
		"request_id":  requestID,
		"duration_ms": duration.Milliseconds(),
	})

	s.writeSuccessResponse(w, http.StatusOK, files)
}

// uploadHandler uploads a file to a bucket via multipart/form-data.
func (s *Server) uploadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetRequestID(ctx)
	start := time.Now()

	vars := mux.Vars(r)
	bucketName := vars["bucketName"]
	if bucketName == "" {
		err := errors.NewValidationError("bucketName is required", "bucketName parameter is missing")
		s.writeServiceErrorResponse(w, err)
		return
	}

	logging.Info(ctx, "Starting file upload", map[string]interface{}{
		"bucket_name": bucketName,
		"request_id":  requestID,
		"operation":   "upload_file",
	})

	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32 MiB
		serviceErr := errors.NewValidationError("Failed to parse multipart form", err.Error())
		s.writeServiceErrorResponse(w, serviceErr)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		serviceErr := errors.NewValidationError("Failed to retrieve file from form", err.Error())
		s.writeServiceErrorResponse(w, serviceErr)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			logging.Error(ctx, "Failed to close file", err, map[string]interface{}{
				"request_id": requestID,
			})
		}
	}()

	// Validate file size (32MB limit)
	const maxFileSize = 32 << 20 // 32 MiB
	if handler.Size > maxFileSize {
		err := errors.NewFileTooLargeError(handler.Filename, handler.Size, maxFileSize)
		s.writeServiceErrorResponse(w, err)
		return
	}

	// Log audit trail
	logging.LogAudit(ctx, "upload_file", bucketName, map[string]interface{}{
		"bucket_name": bucketName,
		"file_name":   handler.Filename,
		"file_size":   handler.Size,
		"request_id":  requestID,
	})

	upload, err := s.client.CreateFileUpload(ctx, bucketName, handler.Filename)
	if err != nil {
		serviceErr := errors.NewStorageError("create_upload", err)
		serviceErr = serviceErr.WithContext("bucket_name", bucketName)
		serviceErr = serviceErr.WithContext("file_name", handler.Filename)
		s.writeServiceErrorResponse(w, serviceErr)
		return
	}

	meta, err := s.client.Upload(ctx, upload, file)
	if err != nil {
		serviceErr := errors.NewUploadFailedError(handler.Filename, err)
		serviceErr = serviceErr.WithContext("bucket_name", bucketName)
		s.writeServiceErrorResponse(w, serviceErr)
		return
	}

	// Log performance
	duration := time.Since(start)
	logging.LogPerformance(ctx, "upload_file", duration, map[string]interface{}{
		"bucket_name": bucketName,
		"file_name":   handler.Filename,
		"file_size":   handler.Size,
		"request_id":  requestID,
	})

	logging.Info(ctx, "File uploaded successfully", map[string]interface{}{
		"bucket_name": bucketName,
		"file_name":   handler.Filename,
		"file_size":   handler.Size,
		"root_cid":    meta.RootCID,
		"request_id":  requestID,
		"duration_ms": duration.Milliseconds(),
	})

	resp := map[string]interface{}{
		"message":     "File uploaded successfully",
		"rootCID":     meta.RootCID,
		"bucketName":  meta.BucketName,
		"fileName":    meta.Name,
		"size":        meta.Size,
		"encodedSize": meta.EncodedSize,
		"createdAt":   meta.CreatedAt,
		"committedAt": meta.CommittedAt,
	}
	s.writeSuccessResponse(w, http.StatusCreated, resp)
}

// downloadHandler streams file content to the client.
func (s *Server) downloadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetRequestID(ctx)
	start := time.Now()

	vars := mux.Vars(r)
	bucketName := vars["bucketName"]
	fileName := vars["fileName"]
	if bucketName == "" || fileName == "" {
		err := errors.NewValidationError("bucketName and fileName are required", "both bucketName and fileName parameters are missing")
		s.writeServiceErrorResponse(w, err)
		return
	}

	logging.Info(ctx, "Starting file download", map[string]interface{}{
		"bucket_name": bucketName,
		"file_name":   fileName,
		"request_id":  requestID,
		"operation":   "download_file",
	})

	dl, err := s.client.CreateFileDownload(ctx, bucketName, fileName)
	if err != nil {
		serviceErr := errors.NewDownloadFailedError(fileName, err)
		serviceErr = serviceErr.WithContext("bucket_name", bucketName)
		s.writeServiceErrorResponse(w, serviceErr)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	w.Header().Set("Content-Type", "application/octet-stream")

	if err := s.client.Download(ctx, dl, w); err != nil {
		logging.Error(ctx, "Error during file stream", err, map[string]interface{}{
			"bucket_name": bucketName,
			"file_name":   fileName,
			"request_id":  requestID,
		})
		return
	}

	// Log performance
	duration := time.Since(start)
	logging.LogPerformance(ctx, "download_file", duration, map[string]interface{}{
		"bucket_name": bucketName,
		"file_name":   fileName,
		"request_id":  requestID,
	})

	logging.Info(ctx, "File downloaded successfully", map[string]interface{}{
		"bucket_name": bucketName,
		"file_name":   fileName,
		"request_id":  requestID,
		"duration_ms": duration.Milliseconds(),
	})
}

// fileDeleteHandler deletes a single file in a bucket.
func (s *Server) fileDeleteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	requestID := middleware.GetRequestID(ctx)
	start := time.Now()

	vars := mux.Vars(r)
	bucketName := vars["bucketName"]
	fileName := vars["fileName"]
	if bucketName == "" || fileName == "" {
		err := errors.NewValidationError("bucketName and fileName are required", "both bucketName and fileName parameters are missing")
		s.writeServiceErrorResponse(w, err)
		return
	}

	logging.Info(ctx, "Deleting file", map[string]interface{}{
		"bucket_name": bucketName,
		"file_name":   fileName,
		"request_id":  requestID,
		"operation":   "delete_file",
	})

	// Log audit trail
	logging.LogAudit(ctx, "delete_file", bucketName, map[string]interface{}{
		"bucket_name": bucketName,
		"file_name":   fileName,
		"request_id":  requestID,
	})

	if err := s.client.FileDelete(ctx, bucketName, fileName); err != nil {
		serviceErr := errors.NewDeleteFailedError(fileName, err)
		serviceErr = serviceErr.WithContext("bucket_name", bucketName)
		s.writeServiceErrorResponse(w, serviceErr)
		return
	}

	// Log performance
	duration := time.Since(start)
	logging.LogPerformance(ctx, "delete_file", duration, map[string]interface{}{
		"bucket_name": bucketName,
		"file_name":   fileName,
		"request_id":  requestID,
	})

	logging.Info(ctx, "File deleted successfully", map[string]interface{}{
		"bucket_name": bucketName,
		"file_name":   fileName,
		"request_id":  requestID,
		"duration_ms": duration.Milliseconds(),
	})

	s.writeSuccessResponse(w, http.StatusOK, map[string]string{"message": "File deleted successfully"})
}
