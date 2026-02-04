package api

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

func (s *Server) HandleProcessStop(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Received request to STOP process")

	if s.processManager == nil {
		log.Error().Msg("ProcessManager is nil")
		http.Error(w, "ProcessManager not initialized", http.StatusInternalServerError)
		return
	}

	err := s.processManager.Stop()
	if err != nil {
		log.Error().Err(err).Msg("Failed to stop process")
		http.Error(w, "Failed to stop process: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"stopped", "message":"Process terminated successfully"}`))
}

type StartRequest struct {
	Kernel string `json:"kernel"`
	Port   int    `json:"port"`
}

func (s *Server) HandleProcessStart(w http.ResponseWriter, r *http.Request) {
	var req StartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Port == 0 {
		req.Port = 41242 // Default port
	}
	if req.Kernel == "" {
		req.Kernel = "gemini"
	}

	log.Info().Str("kernel", req.Kernel).Int("port", req.Port).Msg("Received request to START process")

	if s.processManager == nil {
		http.Error(w, "ProcessManager not initialized", http.StatusInternalServerError)
		return
	}

	// Stop existing first? Or Manager handles it?
	// For simplicity, we assume manager.Start launches a new process.
	// Ideally we should check if running.
	// We'll call Stop first just in case?
	s.processManager.Stop()

	err := s.processManager.Start(req.Kernel, req.Port)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start process")
		http.Error(w, "Failed to start process: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"started", "message":"Process started successfully"}`))
}
