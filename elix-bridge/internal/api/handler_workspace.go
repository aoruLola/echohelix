// Package api provides HTTP handlers for workspace management
package api

import (
	"encoding/json"
	"net/http"
	"os"
)

// HandleWorkspaceList returns the list of workspaces
func (s *Server) HandleWorkspaceList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	workspaces := s.workspaceSvc.List()
	json.NewEncoder(w).Encode(workspaces)
}

// HandleWorkspaceAdd adds a new workspace
func (s *Server) HandleWorkspaceAdd(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Name string `json:"name"`
		Path string `json:"path"`
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
			"error": "path is required",
		})
		return
	}

	ws, err := s.workspaceSvc.Add(req.Name, req.Path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(ws)
}

// HandleWorkspaceRemove removes a workspace
func (s *Server) HandleWorkspaceRemove(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := r.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "id parameter is required",
		})
		return
	}

	if err := s.workspaceSvc.Remove(id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// HandleWorkspaceValidate checks if a path is a valid directory
func (s *Server) HandleWorkspaceValidate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Path string `json:"path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	info, err := os.Stat(req.Path)
	exists := err == nil
	isDir := false
	if exists {
		isDir = info.IsDir()
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"exists":       exists,
		"is_directory": isDir,
		"path":         req.Path,
	})
}
