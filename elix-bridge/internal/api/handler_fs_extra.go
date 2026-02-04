package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

// HandleRoots returns available root directories
// GET /api/v2/fs/roots
func (s *Server) HandleRoots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var roots []map[string]interface{}

	if runtime.GOOS == "windows" {
		// Traverse A-Z
		for _, drive := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
			root := string(drive) + ":\\"
			if _, err := os.Stat(root); err == nil {
				roots = append(roots, map[string]interface{}{
					"name":         string(drive) + ":",
					"path":         root,
					"is_directory": true,
				})
			}
		}
		// Add Project Root if available
		if s.processManager != nil {
			roots = append(roots, map[string]interface{}{
				"name":         "Project Root",
				"path":         s.processManager.WorkDir,
				"is_directory": true,
			})
		}
		// Add User Home
		if home, err := os.UserHomeDir(); err == nil {
			roots = append(roots, map[string]interface{}{
				"name":         "Home",
				"path":         home,
				"is_directory": true,
			})
		}
	} else {
		// Unix
		roots = append(roots, map[string]interface{}{
			"name":         "/",
			"path":         "/",
			"is_directory": true,
		})
		if home, err := os.UserHomeDir(); err == nil {
			roots = append(roots, map[string]interface{}{
				"name":         filepath.Base(home),
				"path":         home,
				"is_directory": true,
			})
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"roots": roots,
	})
}

// HandleStat returns file info
// GET /api/v2/fs/stat?path=...
func (s *Server) HandleStat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := r.URL.Query().Get("path")
	if path == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "path parameter is required",
		})
		return
	}

	// Resolve against WorkDir if relative?
	// V1 used absolute or relative based on service config.
	// Here we should probably support absolute paths for browsing drives,
	// but relative paths for project access.
	// For simplicity, let's treat it as: if absolute, use it; if relative, join with WorkDir.

	targetPath := path
	if !filepath.IsAbs(targetPath) && s.processManager != nil {
		targetPath = filepath.Join(s.processManager.WorkDir, path)
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":          info.Name(),
		"size":          info.Size(),
		"is_directory":  info.IsDir(),
		"modified_time": info.ModTime(),
		"mode":          info.Mode().String(),
	})
}

// HandleExists checks existence
// GET /api/v2/fs/exists?path=...
func (s *Server) HandleExists(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := r.URL.Query().Get("path")
	if path == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "path parameter is required",
		})
		return
	}

	targetPath := path
	if !filepath.IsAbs(targetPath) && s.processManager != nil {
		targetPath = filepath.Join(s.processManager.WorkDir, path)
	}

	_, err := os.Stat(targetPath)
	exists := err == nil || !os.IsNotExist(err)

	isDir := false
	if exists {
		if info, err := os.Stat(targetPath); err == nil {
			isDir = info.IsDir()
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"exists":       exists,
		"is_directory": isDir,
	})
}
