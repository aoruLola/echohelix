// Package api provides HTTP handlers for configuration management
package api

import (
	"encoding/json"
	"net/http"
)

// HandleConfigGet returns all config settings
func (s *Server) HandleConfigGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	settings := s.configSvc.GetAll()
	json.NewEncoder(w).Encode(settings)
}

// HandleConfigSet sets a config value
func (s *Server) HandleConfigSet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	key := r.URL.Query().Get("key")
	if key == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "key parameter is required",
		})
		return
	}

	var req struct {
		Value string `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid request body",
		})
		return
	}

	if err := s.configSvc.Set(key, req.Value); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"key":     key,
		"value":   req.Value,
	})
}
