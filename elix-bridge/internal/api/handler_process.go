package api

import (
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
