// Package session provides session management for EchoHelix Bridge.
//
// Copyright 2026 EchoHelix Contributors
// SPDX-License-Identifier: Apache-2.0
package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Session represents an active coding session
type Session struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	WorkingDirectory string        `json:"working_directory"`
	Provider         string        `json:"provider"`
	Model            string        `json:"model"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
	Status           SessionStatus `json:"status"`
	MessageCount     int           `json:"message_count"`
	TokensUsed       int64         `json:"tokens_used"`
	LastMessage      string        `json:"last_message,omitempty"`
}

// SessionStatus represents the status of a session
type SessionStatus string

const (
	StatusActive SessionStatus = "active"
	StatusIdle   SessionStatus = "idle"
	StatusClosed SessionStatus = "closed"
)

// Message represents a chat message in a session
type Message struct {
	ID         string     `json:"id"`
	SessionID  string     `json:"session_id"`
	Role       string     `json:"role"` // "user", "assistant", "system"
	Content    string     `json:"content"`
	Timestamp  time.Time  `json:"timestamp"`
	TokenCount int        `json:"token_count,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall represents a tool invocation in a message
type ToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
	Result    string                 `json:"result,omitempty"`
	Status    string                 `json:"status"` // "pending", "completed", "failed"
}

// Manager manages coding sessions with persistence
type Manager struct {
	sessions   map[string]*Session
	messages   map[string][]*Message // sessionID -> messages
	mu         sync.RWMutex
	storageDir string
	autoSave   bool
}

// ManagerConfig configures the session manager
type ManagerConfig struct {
	StorageDir string
	AutoSave   bool
}

// NewManager creates a new session manager
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
		messages: make(map[string][]*Message),
		autoSave: false,
	}
}

// NewManagerWithConfig creates a configured session manager
func NewManagerWithConfig(config ManagerConfig) *Manager {
	m := &Manager{
		sessions:   make(map[string]*Session),
		messages:   make(map[string][]*Message),
		storageDir: config.StorageDir,
		autoSave:   config.AutoSave,
	}

	// 如果配置了存储目录，尝试加载现有会话
	if m.storageDir != "" {
		if err := m.LoadAll(); err != nil {
			log.Warn().Err(err).Msg("Failed to load existing sessions")
		}
	}

	return m
}

// Create creates a new session
func (m *Manager) Create(name, workDir, provider, model string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := generateID()
	now := time.Now()

	session := &Session{
		ID:               id,
		Name:             name,
		WorkingDirectory: workDir,
		Provider:         provider,
		Model:            model,
		CreatedAt:        now,
		UpdatedAt:        now,
		Status:           StatusActive,
		MessageCount:     0,
		TokensUsed:       0,
	}

	m.sessions[id] = session
	m.messages[id] = make([]*Message, 0)

	log.Info().
		Str("id", id).
		Str("name", name).
		Str("workDir", workDir).
		Msg("Session created")

	if m.autoSave {
		go m.saveSession(session)
	}

	return session
}

// Get retrieves a session by ID
func (m *Manager) Get(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	return session, ok
}

// List returns all sessions, optionally filtered by status
func (m *Manager) List(statuses ...SessionStatus) []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	statusSet := make(map[SessionStatus]bool)
	for _, s := range statuses {
		statusSet[s] = true
	}

	for _, s := range m.sessions {
		if len(statusSet) == 0 || statusSet[s.Status] {
			sessions = append(sessions, s)
		}
	}

	// 按更新时间倒序排列
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions
}

// Update updates a session
func (m *Manager) Update(id string, updates map[string]string) (*Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[id]
	if !ok {
		return nil, false
	}

	if name, ok := updates["name"]; ok {
		session.Name = name
	}
	if workDir, ok := updates["working_directory"]; ok {
		session.WorkingDirectory = workDir
	}
	if provider, ok := updates["provider"]; ok {
		session.Provider = provider
	}
	if model, ok := updates["model"]; ok {
		session.Model = model
	}
	if status, ok := updates["status"]; ok {
		session.Status = SessionStatus(status)
	}

	session.UpdatedAt = time.Now()

	if m.autoSave {
		go m.saveSession(session)
	}

	return session, true
}

// Delete removes a session
func (m *Manager) Delete(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.sessions[id]; !ok {
		return false
	}

	delete(m.sessions, id)
	delete(m.messages, id)

	if m.storageDir != "" {
		go m.deleteSessionFile(id)
	}

	log.Info().Str("id", id).Msg("Session deleted")
	return true
}

// AddMessage adds a message to a session
func (m *Manager) AddMessage(sessionID, role, content string, tokenCount int) (*Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}

	msg := &Message{
		ID:         generateID(),
		SessionID:  sessionID,
		Role:       role,
		Content:    content,
		Timestamp:  time.Now(),
		TokenCount: tokenCount,
	}

	m.messages[sessionID] = append(m.messages[sessionID], msg)

	// 更新会话统计
	session.MessageCount++
	session.TokensUsed += int64(tokenCount)
	session.LastMessage = truncateString(content, 100)
	session.UpdatedAt = time.Now()
	session.Status = StatusActive

	if m.autoSave {
		go m.saveSession(session)
	}

	return msg, nil
}

// GetMessages returns messages for a session
func (m *Manager) GetMessages(sessionID string, limit, offset int) ([]*Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	msgs, ok := m.messages[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}

	// 应用分页
	total := len(msgs)
	if offset >= total {
		return []*Message{}, nil
	}

	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}

	return msgs[offset:end], nil
}

// SetStatus updates session status
func (m *Manager) SetStatus(id string, status SessionStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[id]
	if !ok {
		return ErrSessionNotFound
	}

	session.Status = status
	session.UpdatedAt = time.Now()

	if m.autoSave {
		go m.saveSession(session)
	}

	return nil
}

// GetActive returns the most recently active session
func (m *Manager) GetActive() *Session {
	sessions := m.List(StatusActive)
	if len(sessions) > 0 {
		return sessions[0]
	}
	return nil
}

// Persistence functions

// SaveAll saves all sessions to disk
func (m *Manager) SaveAll() error {
	if m.storageDir == "" {
		return ErrStorageNotConfigured
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, session := range m.sessions {
		if err := m.saveSession(session); err != nil {
			return err
		}
	}

	return nil
}

// LoadAll loads all sessions from disk
func (m *Manager) LoadAll() error {
	if m.storageDir == "" {
		return ErrStorageNotConfigured
	}

	// 确保目录存在
	if err := os.MkdirAll(m.storageDir, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(m.storageDir)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(m.storageDir, entry.Name())
		session, messages, err := m.loadSessionFile(filePath)
		if err != nil {
			log.Warn().Err(err).Str("file", entry.Name()).Msg("Failed to load session")
			continue
		}

		m.sessions[session.ID] = session
		m.messages[session.ID] = messages
	}

	log.Info().Int("count", len(m.sessions)).Msg("Sessions loaded")
	return nil
}

// sessionPersisted represents the data saved to disk
type sessionPersisted struct {
	Session  *Session   `json:"session"`
	Messages []*Message `json:"messages"`
}

func (m *Manager) saveSession(session *Session) error {
	if m.storageDir == "" {
		return nil
	}

	if err := os.MkdirAll(m.storageDir, 0755); err != nil {
		return err
	}

	// 同时保存会话和消息
	persisted := sessionPersisted{
		Session:  session,
		Messages: m.messages[session.ID],
	}

	filePath := filepath.Join(m.storageDir, session.ID+".json")
	data, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

func (m *Manager) loadSessionFile(filePath string) (*Session, []*Message, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, err
	}

	// 先尝试新格式（包含消息）
	var persisted sessionPersisted
	if err := json.Unmarshal(data, &persisted); err == nil && persisted.Session != nil {
		messages := persisted.Messages
		if messages == nil {
			messages = make([]*Message, 0)
		}
		return persisted.Session, messages, nil
	}

	// 回退到旧格式（仅会话）
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, nil, err
	}

	return &session, make([]*Message, 0), nil
}

func (m *Manager) deleteSessionFile(id string) {
	if m.storageDir == "" {
		return
	}

	filePath := filepath.Join(m.storageDir, id+".json")
	os.Remove(filePath)
}

// Helper functions

func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Errors
var (
	ErrSessionNotFound      = &SessionError{Code: "SESSION_NOT_FOUND", Message: "Session not found"}
	ErrStorageNotConfigured = &SessionError{Code: "STORAGE_NOT_CONFIGURED", Message: "Storage directory not configured"}
)

// SessionError represents a session-related error
type SessionError struct {
	Code    string
	Message string
}

func (e *SessionError) Error() string {
	return e.Message
}
