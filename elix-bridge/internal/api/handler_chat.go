package api

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for dev
	},
}

// HandleChatProxy upgrades the connection to WebSocket and proxies messages to the active kernel
func (s *Server) HandleChatProxy(w http.ResponseWriter, r *http.Request) {
	// 1. Upgrade Client Connection
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade websocket")
		return
	}
	defer clientConn.Close()

	// 2. Determine Target Kernel Port
	// Default to Gemini (41242) if not specified or tracked
	// Ideally ProcessManager should tell us the active port/kernel.
	// For now, we'll try to determine based on query param or default.
	// P4/P5 assumption: bridge routes based on "kernel" field in JSON-RPC?
	// ACTUALLY: The Architecture diagram says "App <--> Bridge <--> Kernel".
	// The Bridge is a "Dumb Pipe".
	// We need to connect to the RUNNING kernel.
	// Since ProcessManager only runs ONE kernel at a time (per our previous logic),
	// we can try to connect to the active one.

	// Issue: We don't know WHICH port is active easily without querying PM.
	// But PM currently takes a port as arg.
	// Let's assume standard ports: Gemini 41242, Aider 41243.
	// We'll try 41242 first, then 41243? Or use a query param `?kernel=aider`.

	kernel := r.URL.Query().Get("kernel")
	targetPort := 41242
	if kernel == "aider" {
		targetPort = 41243
	}

	targetURL := fmt.Sprintf("ws://127.0.0.1:%d", targetPort)
	if kernel == "aider" {
		// Aider server.py (FastAPI) doesn't expose a WS endpoint typically?
		// Wait, server.py uses HTTP POST /chat.
		// BRIDGE MUST TRANSLATE WS <-> HTTP for Aider?
		// OR we update Aider server.py to support WS?
		// "Core Policy": Modify source directly if wrapping.
		// Aider is a CLI. Wrapping it in FastAPI with /chat is Request/Response.
		// App expects WS.
		// Bridge must handle the conversion or Aider Server must support WS.
		// Easier to make Aider Server support WS.
		// BUT for now, let's implement the Proxy for Gemini (which is WS?)
		// Gemini "a2a-server" IS a WebSocket server?
		// Checking "a2a-server": it uses `ws` package. Yes.

		// For Aider: The plan was "Stream output".
		// Converting Aider to WS in server.py is best for consistency.
		// Let's assume Aider Server will accept WS at /ws.
		targetURL = fmt.Sprintf("ws://127.0.0.1:%d/ws", targetPort)
	}

	log.Info().Str("target", targetURL).Msg("Proxying Chat Connection")

	// 3. Connect to Backend Kernel
	backendConn, _, err := websocket.DefaultDialer.Dial(targetURL, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to backend kernel")
		clientConn.WriteJSON(map[string]string{"error": "Backend not available"})
		return
	}
	defer backendConn.Close()

	// 4. Pipe Data
	var wg sync.WaitGroup
	wg.Add(2)

	// Client -> Backend
	go func() {
		defer wg.Done()
		for {
			mt, message, err := clientConn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Error().Err(err).Msg("Client read error")
				}
				return
			}
			err = backendConn.WriteMessage(mt, message)
			if err != nil {
				log.Error().Err(err).Msg("Backend write error")
				return
			}
		}
	}()

	// Backend -> Client
	go func() {
		defer wg.Done()
		for {
			mt, message, err := backendConn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Error().Err(err).Msg("Backend read error")
				}
				return
			}
			err = clientConn.WriteMessage(mt, message)
			if err != nil {
				log.Error().Err(err).Msg("Client write error")
				return
			}
		}
	}()

	wg.Wait()
	log.Info().Msg("Chat Proxy Closed")
}
