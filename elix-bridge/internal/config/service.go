package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Service handles configuration reading and writing
type Service struct {
	mu       sync.RWMutex
	envPath  string
	settings map[string]string
}

// NewService creates a new config service
func NewService(envPath string) *Service {
	if envPath == "" {
		envPath = ".env"
	}
	s := &Service{
		envPath:  envPath,
		settings: make(map[string]string),
	}
	_ = s.Load()
	return s
}

// Load loads settings from the .env file
func (s *Service) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(s.envPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	s.settings = make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			s.settings[parts[0]] = parts[1]
		}
	}
	return scanner.Err()
}

// Get gets a setting value
func (s *Service) Get(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings[key]
}

// Set sets a setting value and saves to disk
func (s *Service) Set(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.settings[key] = value

	// Read current file to preserve comments and order
	lines, err := s.readLines()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+"=") {
			lines[i] = fmt.Sprintf("%s=%s", key, value)
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	return os.WriteFile(s.envPath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

func (s *Service) readLines() ([]string, error) {
	file, err := os.Open(s.envPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// GetAll returns all settings
func (s *Service) GetAll() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make(map[string]string)
	for k, v := range s.settings {
		res[k] = v
	}
	return res
}
