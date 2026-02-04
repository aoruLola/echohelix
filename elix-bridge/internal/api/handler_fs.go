package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"echohelix/bridge/internal/fs"

	"github.com/rs/zerolog/log"
)

// HandleFSList returns a list of files in the workspace
// GET /api/v2/fs/ls?path=.&recursive=true
func (s *Server) HandleFSList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	relPath := query.Get("path")
	if relPath == "" {
		relPath = "."
	}
	recursive := query.Get("recursive") == "true"

	if s.processManager == nil {
		http.Error(w, "ProcessManager not initialized", http.StatusInternalServerError)
		return
	}

	// Use ProcessManager's WorkDir as the root
	walker := fs.NewWalker(s.processManager.WorkDir)

	// Validate path is not escaping root (basic check)
	cleanPath := filepath.Clean(relPath)
	if cleanPath == ".." || cleanPath[:3] == "../" {
		http.Error(w, "Invalid path: cannot escape root", http.StatusBadRequest)
		return
	}

	entries, err := walker.ListFiles(cleanPath, recursive)
	if err != nil {
		log.Error().Err(err).Str("path", relPath).Msg("Failed to list files")
		http.Error(w, "Failed to list files: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(entries); err != nil {
		log.Error().Err(err).Msg("Failed to encode response")
		http.Error(w, "Internal serialization error", http.StatusInternalServerError)
	}
}
