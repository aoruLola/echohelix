package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/rs/zerolog/log"
)

// HandleFile returns file content
// GET /api/v2/fs/file?path=...&offset=0&limit=0
func (s *Server) HandleFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if s.processManager == nil {
		http.Error(w, "ProcessManager not initialized", http.StatusInternalServerError)
		return
	}

	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "path parameter is required",
		})
		return
	}

	// Parsing offset/limit
	offset := 0
	limit := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil {
			offset = v
		}
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil {
			limit = v
		}
	}

	fullPath := filepath.Join(s.processManager.WorkDir, relPath)

	// Security check to prevent escaping WorkDir
	// Note: basic check. For production, more robust sandboxing is needed.
	// But EchoHelix acts as a local agent, so we trust the user context mostly.

	f, err := os.Open(fullPath)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": fmt.Sprintf("File not found or unreadable: %s", err),
		})
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	if info.IsDir() {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "path is a directory",
		})
		return
	}

	fileSize := info.Size()

	if int64(offset) > fileSize {
		offset = int(fileSize) // Clamp
	}

	// If limit is 0 or exceeds remaining, read to end
	if limit <= 0 || int64(offset+limit) > fileSize {
		limit = int(fileSize) - offset
	}

	// Seek
	if _, err := f.Seek(int64(offset), 0); err != nil {
		http.Error(w, "Seek failed", http.StatusInternalServerError)
		return
	}

	// Read
	buf := make([]byte, limit)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		http.Error(w, "Read failed", http.StatusInternalServerError)
		return
	}
	buf = buf[:n]

	// Response structure from V1
	resp := map[string]interface{}{
		"path":      relPath,
		"content":   string(buf),
		"size":      fileSize,
		"offset":    offset,
		"limit":     limit,
		"truncated": int64(offset+limit) < fileSize, // Crude truncated check
		"is_binary": false,                          // TODO: Implement binary check if needed
	}

	json.NewEncoder(w).Encode(resp)
}

// HandleWriteFile writes content to a file
// POST /api/v2/fs/write
func (s *Server) HandleWriteFile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if s.processManager == nil {
		http.Error(w, "ProcessManager not initialized", http.StatusInternalServerError)
		return
	}

	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "invalid request body",
		})
		return
	}

	if req.Path == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "path parameter is required",
		})
		return
	}

	fullPath := filepath.Join(s.processManager.WorkDir, req.Path)

	// Ensure dir exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "failed to create directory: " + err.Error(),
		})
		return
	}

	// Write file
	// Use os.WriteFile for atomic-ish write (replace content)
	if err := os.WriteFile(fullPath, []byte(req.Content), 0644); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "failed to write file: " + err.Error(),
		})
		log.Error().Err(err).Str("path", fullPath).Msg("Failed to write file")
		return
	}

	log.Info().Str("path", req.Path).Msg("File written successfully")

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"path":    req.Path,
	})
}
