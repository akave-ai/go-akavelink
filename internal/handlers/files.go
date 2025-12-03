package handlers

import (
	"log"
	"net/http"

	"github.com/akave-ai/go-akavelink/internal/validation"
	"github.com/gorilla/mux"
)

// fileInfoHandler returns metadata for a specific file.
func (s *Server) fileInfoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucketName"]
	fileName := vars["fileName"]

	// Validate bucket name
	if err := validation.ValidateBucketName(bucketName); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate file name
	if err := validation.ValidateFileName(fileName); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Sanitize inputs
	bucketName = validation.SanitizeBucketName(bucketName)
	fileName = validation.SanitizeFileName(fileName)

	info, err := s.client.FileInfo(r.Context(), bucketName, fileName)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve file info")
		log.Printf("file info error: %v", err)
		return
	}

	s.writeSuccessResponse(w, http.StatusOK, info)
}

// listFilesHandler lists files in a bucket.
func (s *Server) listFilesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucketName"]

	// Validate bucket name
	if err := validation.ValidateBucketName(bucketName); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Sanitize bucket name
	bucketName = validation.SanitizeBucketName(bucketName)

	files, err := s.client.ListFiles(r.Context(), bucketName)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to list files")
		log.Printf("list files error: %v", err)
		return
	}

	s.writeSuccessResponse(w, http.StatusOK, files)
}

// uploadHandler uploads a file to a bucket via multipart/form-data.
func (s *Server) uploadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucketName"]

	// Validate bucket name
	if err := validation.ValidateBucketName(bucketName); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Sanitize bucket name
	bucketName = validation.SanitizeBucketName(bucketName)

	// Validate content length before parsing
	if err := validation.ValidateContentLength(r.ContentLength); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32 MiB
		s.writeErrorResponse(w, http.StatusBadRequest, "Failed to parse multipart form")
		log.Printf("parse multipart error: %v", err)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Failed to retrieve file from form")
		log.Printf("form file error: %v", err)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("file close error: %v", err)
		}
	}()

	// Validate file upload (size, MIME type, filename)
	if err := validation.ValidateFileUpload(handler); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Sanitize filename
	fileName := validation.SanitizeFileName(handler.Filename)

	ctx := r.Context()
	upload, err := s.client.CreateFileUpload(ctx, bucketName, fileName)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create file upload stream")
		log.Printf("create upload error: %v", err)
		return
	}

	meta, err := s.client.Upload(ctx, upload, file)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to upload file content")
		log.Printf("upload error: %v", err)
		return
	}

	resp := map[string]interface{}{
		"message":       "File uploaded successfully",
		"rootCID":       meta.RootCID,
		"bucketName":    meta.BucketName,
		"fileName":      meta.Name,
		"originalName":  handler.Filename,
		"sanitizedName": fileName,
		"size":          meta.Size,
		"encodedSize":   meta.EncodedSize,
		"createdAt":     meta.CreatedAt,
		"committedAt":   meta.CommittedAt,
	}
	s.writeSuccessResponse(w, http.StatusCreated, resp)
}

// downloadHandler streams file content to the client.
func (s *Server) downloadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucketName"]
	fileName := vars["fileName"]

	// Validate bucket name
	if err := validation.ValidateBucketName(bucketName); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate file name
	if err := validation.ValidateFileName(fileName); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Sanitize inputs
	bucketName = validation.SanitizeBucketName(bucketName)
	fileName = validation.SanitizeFileName(fileName)

	dl, err := s.client.CreateFileDownload(r.Context(), bucketName, fileName)
	if err != nil {
		s.writeErrorResponse(w, http.StatusNotFound, "Failed to create file download")
		log.Printf("create download error: %v", err)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	w.Header().Set("Content-Type", "application/octet-stream")
	if err := s.client.Download(r.Context(), dl, w); err != nil {
		log.Printf("Error during file stream: %v", err)
		return
	}
}

// fileDeleteHandler deletes a single file in a bucket.
func (s *Server) fileDeleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucketName"]
	fileName := vars["fileName"]

	// Validate bucket name
	if err := validation.ValidateBucketName(bucketName); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate file name
	if err := validation.ValidateFileName(fileName); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Sanitize inputs
	bucketName = validation.SanitizeBucketName(bucketName)
	fileName = validation.SanitizeFileName(fileName)

	if err := s.client.FileDelete(r.Context(), bucketName, fileName); err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete file")
		log.Printf("file delete error: %v", err)
		return
	}

	s.writeSuccessResponse(w, http.StatusOK, map[string]string{"message": "File deleted successfully"})
}
