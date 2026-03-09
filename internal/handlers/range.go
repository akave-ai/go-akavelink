package handlers

import (
	"io"
	"strconv"
	"strings"
)

// parseByteRange parses a single "bytes=start-end" Range header per RFC 7233.
// Returns (start, end, totalCap, ok). totalCap is the known file size (for Content-Range); use -1 if unknown.
// Supports: "bytes=0-499", "bytes=500-" (from 500 to end), "bytes=-500" (last 500 bytes).
// If invalid or multiple ranges, returns ok=false.
func parseByteRange(rangeHeader string, totalSize int64) (start, end int64, ok bool) {
	rangeHeader = strings.TrimSpace(rangeHeader)
	if rangeHeader == "" || totalSize < 0 {
		return 0, 0, false
	}
	if !strings.HasPrefix(strings.ToLower(rangeHeader), "bytes=") {
		return 0, 0, false
	}
	rangeSpec := strings.TrimSpace(rangeHeader[6:])
	if rangeSpec == "" {
		return 0, 0, false
	}
	// Single range only (no comma)
	if strings.Contains(rangeSpec, ",") {
		return 0, 0, false
	}
	parts := strings.Split(rangeSpec, "-")
	if len(parts) != 2 {
		return 0, 0, false
	}
	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])

	if startStr == "" {
		// Suffix-byte-range: "-500" = last 500 bytes
		suffix, err := strconv.ParseInt(endStr, 10, 64)
		if err != nil || suffix <= 0 {
			return 0, 0, false
		}
		start = totalSize - suffix
		if start < 0 {
			start = 0
		}
		end = totalSize - 1
		return start, end, true
	}

	start, err := strconv.ParseInt(startStr, 10, 64)
	if err != nil || start < 0 {
		return 0, 0, false
	}
	if start >= totalSize {
		return 0, 0, false // unsatisfiable
	}

	if endStr == "" {
		// "500-" = from 500 to end
		end = totalSize - 1
		return start, end, true
	}

	end, err = strconv.ParseInt(endStr, 10, 64)
	if err != nil || end < start {
		return 0, 0, false
	}
	if end >= totalSize {
		end = totalSize - 1
	}
	return start, end, true
}

// skipLimitWriter is an io.Writer that discards the first skip bytes then writes up to limit bytes to dest.
type skipLimitWriter struct {
	dest   io.Writer
	skip   int64
	limit  int64
	skipped int64
	written int64
}

func newSkipLimitWriter(dest io.Writer, skip, limit int64) *skipLimitWriter {
	return &skipLimitWriter{dest: dest, skip: skip, limit: limit}
}

func (w *skipLimitWriter) Write(p []byte) (n int, err error) {
	consumed := 0
	// Consume skip
	if w.skipped < w.skip {
		toSkip := int64(len(p))
		if toSkip > w.skip-w.skipped {
			toSkip = w.skip - w.skipped
		}
		w.skipped += toSkip
		consumed += int(toSkip)
		p = p[toSkip:]
		if len(p) == 0 {
			return consumed, nil
		}
	}
	// Write up to limit (remaining bytes we're allowed to write)
	remaining := w.limit - w.written
	if remaining <= 0 {
		return consumed + len(p), nil // discard rest of p
	}
	toWrite := int64(len(p))
	if toWrite > remaining {
		toWrite = remaining
		p = p[:toWrite]
	}
	nn, err := w.dest.Write(p)
	consumed += nn
	if nn > 0 {
		w.written += int64(nn)
	}
	return consumed, err
}
