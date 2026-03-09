package handlers

import (
	"net/http"
	"strconv"

	"github.com/akave-ai/go-akavelink/internal/logger"
	"github.com/gorilla/mux"
)

// fileInfoHandler returns metadata for a specific file.
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
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve file info")
		logger.Error("file info failed", "bucket", bucketName, "file", fileName, "error", err)
		return
	}

	s.writeSuccessResponse(w, http.StatusOK, info)
}

// listFilesHandler lists files in a bucket.
func (s *Server) listFilesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucketName"]
	if bucketName == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "bucketName is required")
		return
	}

	files, err := s.client.ListFiles(r.Context(), bucketName)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to list files")
		logger.Error("list files failed", "bucket", bucketName, "error", err)
		return
	}

	s.writeSuccessResponse(w, http.StatusOK, files)
}

// uploadHandler uploads a file to a bucket via multipart/form-data.
func (s *Server) uploadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucketName"]
	if bucketName == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "bucketName is required")
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32 MiB
		s.writeErrorResponse(w, http.StatusBadRequest, "Failed to parse multipart form")
		logger.Error("parse multipart failed", "bucket", bucketName, "error", err)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "Failed to retrieve file from form")
		logger.Error("form file missing or invalid", "bucket", bucketName, "error", err)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Warn("upload file close error", "error", err)
		}
	}()

	ctx := r.Context()
	upload, err := s.client.CreateFileUpload(ctx, bucketName, handler.Filename)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create file upload stream")
		logger.Error("create file upload failed", "bucket", bucketName, "file", handler.Filename, "error", err)
		return
	}

	meta, err := s.client.Upload(ctx, upload, file)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to upload file content")
		logger.Error("upload failed", "bucket", bucketName, "file", handler.Filename, "error", err)
		return
	}

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

// downloadHandler streams file content to the client. Supports Range requests (RFC 7233).
func (s *Server) downloadHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucketName"]
	fileName := vars["fileName"]
	if bucketName == "" || fileName == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "bucketName and fileName are required")
		return
	}

	rangeHeader := r.Header.Get("Range")
	var totalSize int64 = -1
	if rangeHeader != "" {
		info, err := s.client.FileInfo(r.Context(), bucketName, fileName)
		if err != nil {
			s.writeErrorResponse(w, http.StatusNotFound, "Failed to get file info for range")
			logger.Error("file info for range failed", "bucket", bucketName, "file", fileName, "error", err)
			return
		}
		totalSize = info.ActualSize
		start, end, ok := parseByteRange(rangeHeader, totalSize)
		if !ok {
			w.Header().Set("Content-Range", "bytes */"+strconv.FormatInt(totalSize, 10))
			s.writeErrorResponse(w, http.StatusRequestedRangeNotSatisfiable, "Invalid or unsatisfiable Range")
			return
		}
		// Single range: stream only [start, end]
		dl, err := s.client.CreateFileDownload(r.Context(), bucketName, fileName)
		if err != nil {
			s.writeErrorResponse(w, http.StatusNotFound, "Failed to create file download")
			logger.Error("create download failed", "bucket", bucketName, "file", fileName, "error", err)
			return
		}
		contentLength := end - start + 1
		w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Range", "bytes "+strconv.FormatInt(start, 10)+"-"+strconv.FormatInt(end, 10)+"/"+strconv.FormatInt(totalSize, 10))
		w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
		w.WriteHeader(http.StatusPartialContent)
		rangeW := newSkipLimitWriter(w, start, contentLength)
		if err := s.client.Download(r.Context(), dl, rangeW); err != nil {
			logger.Error("download stream failed", "bucket", bucketName, "file", fileName, "error", err)
			return
		}
		return
	}

	// Full file download
	dl, err := s.client.CreateFileDownload(r.Context(), bucketName, fileName)
	if err != nil {
		s.writeErrorResponse(w, http.StatusNotFound, "Failed to create file download")
		logger.Error("create download failed", "bucket", bucketName, "file", fileName, "error", err)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	w.Header().Set("Content-Type", "application/octet-stream")
	if err := s.client.Download(r.Context(), dl, w); err != nil {
		logger.Error("download stream failed", "bucket", bucketName, "file", fileName, "error", err)
		return
	}
}

// fileDeleteHandler deletes a single file in a bucket.
func (s *Server) fileDeleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucketName := vars["bucketName"]
	fileName := vars["fileName"]
	if bucketName == "" || fileName == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "bucketName and fileName are required")
		return
	}

	if err := s.client.FileDelete(r.Context(), bucketName, fileName); err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete file")
		logger.Error("file delete failed", "bucket", bucketName, "file", fileName, "error", err)
		return
	}

	s.writeSuccessResponse(w, http.StatusOK, map[string]string{"message": "File deleted successfully"})
}
