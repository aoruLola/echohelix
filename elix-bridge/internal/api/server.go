package api

import (
	"context"
	"net/http"

	"echohelix/bridge/internal/process"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/rs/zerolog/log"
)

type Server struct {
	router         *mux.Router
	httpServer     *http.Server
	processManager *process.Manager
}

func NewServer(pm *process.Manager) *Server {
	s := &Server{
		router:         mux.NewRouter(),
		processManager: pm,
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// API v2 Routes
	v2 := s.router.PathPrefix("/api/v2").Subrouter()

	// Process Management
	v2.HandleFunc("/process/stop", s.HandleProcessStop).Methods("POST")

	// Health Check
	v2.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")
}

func (s *Server) Start(addr string) error {
	// CORS Handler
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // Allow all for local dev
		AllowedMethods:   []string{"GET", "POST", "OPTIONS", "DELETE", "PUT"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(s.router)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	log.Info().Str("addr", addr).Msg("Starting Bridge HTTP Server")
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
