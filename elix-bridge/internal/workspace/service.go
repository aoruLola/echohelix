// Package workspace provides workspace management for EchoHelix Bridge.
//
// Copyright 2026 EchoHelix Contributors
// SPDX-License-Identifier: Apache-2.0
package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Workspace represents a saved project workspace
type Workspace struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	LastAccess time.Time `json:"last_access"`
}

// Service manages the list of saved workspaces
type Service struct {
	mu         sync.RWMutex
	workspaces []Workspace
	filePath   string
}

// NewService creates a new workspace service
func NewService(configDir string) *Service {
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".echohelix")
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Error().Err(err).Msg("Failed to create config directory")
	}

	filePath := filepath.Join(configDir, "workspaces.json")
	s := &Service{
		filePath: filePath,
	}

	if err := s.load(); err != nil {
		log.Warn().Err(err).Msg("No workspaces.json found or failed to load")
		s.workspaces = []Workspace{}
	}

	return s
}

// List returns the list of all workspaces
func (s *Service) List() []Workspace {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.workspaces
}

// Add adds a new workspace
func (s *Service) Add(name, path string) (Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if path already exists
	for _, w := range s.workspaces {
		if w.Path == path {
			return w, nil
		}
	}

	w := Workspace{
		ID:         fmt.Sprintf("ws_%d", time.Now().UnixNano()),
		Name:       name,
		Path:       path,
		LastAccess: time.Now(),
	}

	s.workspaces = append(s.workspaces, w)
	err := s.save()
	return w, err
}

// Remove removes a workspace by ID
func (s *Service) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, w := range s.workspaces {
		if w.ID == id {
			s.workspaces = append(s.workspaces[:i], s.workspaces[i+1:]...)
			return s.save()
		}
	}

	return fmt.Errorf("workspace not found: %s", id)
}

// UpdateAccess updates the last access time for a workspace
func (s *Service) UpdateAccess(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, w := range s.workspaces {
		if w.Path == path {
			s.workspaces[i].LastAccess = time.Now()
			s.save()
			return
		}
	}
}

func (s *Service) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.workspaces)
}

func (s *Service) save() error {
	data, err := json.MarshalIndent(s.workspaces, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
}
