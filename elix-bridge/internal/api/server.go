package api

import (
	"context"
	"net/http"

	"echohelix/bridge/internal/process"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/rs/zerolog/log"

	"echohelix/bridge/internal/auth"
)

type Server struct {
	router         *mux.Router
	httpServer     *http.Server
	processManager *process.Manager
	authHandler    *auth.Handler
}

func NewServer(pm *process.Manager) *Server {
	// Initialize Auth Service
	// TODO: Make config configurable via flags/env
	authConfig := auth.DefaultConfig()
	// Optionally persist auth state to home dir
	// authConfig.StoragePath = "..."

	authService := auth.NewService(authConfig)
	authHandler := auth.NewHandler(authService)

	s := &Server{
		router:         mux.NewRouter(),
		processManager: pm,
		authHandler:    authHandler,
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// API v2 Routes
	v2 := s.router.PathPrefix("/api/v2").Subrouter()

	// Health Check
	v2.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// Auth API (Public)
	v2.HandleFunc("/auth/pair", s.authHandler.HandlePair).Methods("POST")
	v2.HandleFunc("/auth/code", s.authHandler.HandleGenerateCode).Methods("POST")
	v2.HandleFunc("/auth/status", s.authHandler.HandleStatus).Methods("GET")

	// Protected Routes Wrapper
	protect := s.authHandler.AuthenticateMiddleware

	// Process Management (Protected)
	v2.HandleFunc("/process/stop", protect(s.HandleProcessStop)).Methods("POST")
	v2.HandleFunc("/process/start", protect(s.HandleProcessStart)).Methods("POST")

	// Chat Proxy (Protected)
	// Note: Websocket auth usually via query param, handled directly in handler or via middleware
	// We'll trust the middleware to check query param token too
	v2.HandleFunc("/chat/proxy", protect(s.HandleChatProxy))

	// File System (Protected)
	v2.HandleFunc("/fs/ls", protect(s.HandleFSList)).Methods("GET")
	v2.HandleFunc("/fs/file", protect(s.HandleFile)).Methods("GET")
	v2.HandleFunc("/fs/write", protect(s.HandleWriteFile)).Methods("POST")
	v2.HandleFunc("/fs/roots", protect(s.HandleRoots)).Methods("GET")
	v2.HandleFunc("/fs/stat", protect(s.HandleStat)).Methods("GET")
	v2.HandleFunc("/fs/exists", protect(s.HandleExists)).Methods("GET")

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
