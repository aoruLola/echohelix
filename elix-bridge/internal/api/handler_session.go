// Package api provides HTTP handlers for session management
package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"echohelix/bridge/internal/session"
)

// HandleSessionList returns all sessions
func (s *Server) HandleSessionList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 可选的状态过滤
	statusParam := r.URL.Query().Get("status")
	var sessions []*session.Session
	if statusParam != "" {
		sessions = s.sessionMgr.List(session.SessionStatus(statusParam))
	} else {
		sessions = s.sessionMgr.List()
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// HandleSessionCreate creates a new session
func (s *Server) HandleSessionCreate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Name             string `json:"name"`
		WorkingDirectory string `json:"working_directory"`
		Provider         string `json:"provider"`
		Model            string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	// 默认值
	if req.Name == "" {
		req.Name = "New Session"
	}
	if req.Provider == "" {
		req.Provider = "gemini"
	}
	if req.Model == "" {
		req.Model = "gemini-2.5-flash"
	}

	sess := s.sessionMgr.Create(req.Name, req.WorkingDirectory, req.Provider, req.Model)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sess)
}

// HandleSessionGet returns a specific session
func (s *Server) HandleSessionGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Session ID is required",
		})
		return
	}

	sess, ok := s.sessionMgr.Get(sessionID)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Session not found",
		})
		return
	}

	json.NewEncoder(w).Encode(sess)
}

// HandleSessionUpdate updates a session
func (s *Server) HandleSessionUpdate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Session ID is required",
		})
		return
	}

	var updates map[string]string
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	sess, ok := s.sessionMgr.Update(sessionID, updates)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Session not found",
		})
		return
	}

	json.NewEncoder(w).Encode(sess)
}

// HandleSessionDelete deletes a session
func (s *Server) HandleSessionDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Session ID is required",
		})
		return
	}

	if !s.sessionMgr.Delete(sessionID) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Session not found",
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// HandleSessionMessages returns messages for a session
func (s *Server) HandleSessionMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Session ID is required",
		})
		return
	}

	// 解析分页参数
	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
			limit = v
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = v
		}
	}

	messages, err := s.sessionMgr.GetMessages(sessionID, limit, offset)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": messages,
		"count":    len(messages),
		"offset":   offset,
		"limit":    limit,
	})
}

// HandleSessionAddMessage adds a message to a session
func (s *Server) HandleSessionAddMessage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Session ID is required",
		})
		return
	}

	var req struct {
		Role       string `json:"role"`
		Content    string `json:"content"`
		TokenCount int    `json:"token_count"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	msg, err := s.sessionMgr.AddMessage(sessionID, req.Role, req.Content, req.TokenCount)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(msg)
}
