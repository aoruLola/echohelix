package main

import (
	"os"

	"echohelix/bridge/internal/api"
	"echohelix/bridge/internal/process"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Setup Logging
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Info().Msg("EchoHelix Bridge v3 Starting...")

	// 1. Initialize Process Manager
	cwd, _ := os.Getwd()
	pm := process.NewManager(cwd)

	// Note: We are NOT auto-starting the Gemini Core here yet.
	// We will add a /process/start endpoint later or let the user control it.
	// For now, we focus on the Stop capability as requested.

	// 2. Initialize API Server
	server := api.NewServer(pm)

	// 3. Start Server
	// Bridge listens on 8765 (standard EchoHelix Bridge port)
	if err := server.Start(":8765"); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
