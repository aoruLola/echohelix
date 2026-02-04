// Package auth provides pairing authentication for EchoHelix Bridge.
//
// Copyright 2026 EchoHelix Contributors
// SPDX-License-Identifier: Apache-2.0
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// PairingCode represents an active pairing code
type PairingCode struct {
	Code      string    `json:"code"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	DeviceID  string    `json:"device_id,omitempty"`
	Used      bool      `json:"-"`
}

// Token represents an authentication token
type Token struct {
	Value       string    `json:"token"`
	DeviceID    string    `json:"device_id"`
	DeviceName  string    `json:"device_name"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	LastUsedAt  time.Time `json:"last_used_at"`
	Permissions []string  `json:"permissions"`
}

// Service provides authentication services
type Service struct {
	mu           sync.RWMutex
	pairingCodes map[string]*PairingCode
	tokens       map[string]*Token
	deviceTokens map[string]string // deviceID -> tokenValue

	// 配置
	codeLength       int
	codeExpiry       time.Duration
	tokenExpiry      time.Duration
	maxActiveDevices int
	storagePath      string

	// 回调
	onPairingComplete func(deviceID, deviceName string)
}

// ServiceConfig configures the auth service
type ServiceConfig struct {
	CodeLength       int           // 配对码长度，默认 6
	CodeExpiry       time.Duration // 配对码过期时间，默认 5 分钟
	TokenExpiry      time.Duration // Token 过期时间，默认 30 天
	MaxActiveDevices int           // 最大活跃设备数，默认 5
	StoragePath      string        // 持久化存储路径，空则不持久化
}

// DefaultConfig returns default configuration
func DefaultConfig() ServiceConfig {
	return ServiceConfig{
		CodeLength:       6,
		CodeExpiry:       5 * time.Minute,
		TokenExpiry:      30 * 24 * time.Hour,
		MaxActiveDevices: 5,
	}
}

// NewService creates a new authentication service
func NewService(config ServiceConfig) *Service {
	if config.CodeLength == 0 {
		config.CodeLength = 6
	}
	if config.CodeExpiry == 0 {
		config.CodeExpiry = 5 * time.Minute
	}
	if config.TokenExpiry == 0 {
		config.TokenExpiry = 30 * 24 * time.Hour
	}
	if config.MaxActiveDevices == 0 {
		config.MaxActiveDevices = 5
	}

	s := &Service{
		pairingCodes:     make(map[string]*PairingCode),
		tokens:           make(map[string]*Token),
		deviceTokens:     make(map[string]string),
		codeLength:       config.CodeLength,
		codeExpiry:       config.CodeExpiry,
		tokenExpiry:      config.TokenExpiry,
		maxActiveDevices: config.MaxActiveDevices,
		storagePath:      config.StoragePath,
	}

	// 尝试从磁盘加载已保存的 Token
	if config.StoragePath != "" {
		if err := s.LoadState(); err != nil {
			log.Warn().Err(err).Msg("Failed to load auth state, starting fresh")
		}
	}

	// 启动过期清理 goroutine
	go s.cleanupExpired()

	return s
}

// GeneratePairingCode generates a new pairing code
func (s *Service) GeneratePairingCode() (*PairingCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 清理已过期的配对码
	s.cleanupExpiredCodesLocked()

	// 生成随机数字码
	code, err := generateNumericCode(s.codeLength)
	if err != nil {
		return nil, fmt.Errorf("failed to generate code: %w", err)
	}

	// 确保唯一性
	for s.pairingCodes[code] != nil {
		code, err = generateNumericCode(s.codeLength)
		if err != nil {
			return nil, fmt.Errorf("failed to generate unique code: %w", err)
		}
	}

	now := time.Now()
	pc := &PairingCode{
		Code:      code,
		CreatedAt: now,
		ExpiresAt: now.Add(s.codeExpiry),
	}

	s.pairingCodes[code] = pc

	log.Info().
		Str("code", code).
		Time("expires", pc.ExpiresAt).
		Msg("Pairing code generated")

	return pc, nil
}

// ValidatePairingCode validates a pairing code and issues a token
func (s *Service) ValidatePairingCode(code, deviceID, deviceName string) (*Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pc, exists := s.pairingCodes[code]
	if !exists {
		return nil, ErrInvalidCode
	}

	if time.Now().After(pc.ExpiresAt) {
		delete(s.pairingCodes, code)
		return nil, ErrCodeExpired
	}

	if pc.Used {
		return nil, ErrCodeAlreadyUsed
	}

	// 检查设备数量限制
	activeCount := 0
	for _, token := range s.tokens {
		if time.Now().Before(token.ExpiresAt) {
			activeCount++
		}
	}
	if activeCount >= s.maxActiveDevices {
		// 查找最旧的 token 并删除
		s.removeOldestTokenLocked()
	}

	// 标记配对码已使用
	pc.Used = true
	pc.DeviceID = deviceID
	delete(s.pairingCodes, code)

	// 生成 Token
	token, err := s.createTokenLocked(deviceID, deviceName)
	if err != nil {
		return nil, err
	}

	log.Info().
		Str("deviceID", deviceID).
		Str("deviceName", deviceName).
		Msg("Device paired successfully")

	if s.onPairingComplete != nil {
		go s.onPairingComplete(deviceID, deviceName)
	}

	return token, nil
}

// ValidateToken validates an authentication token
func (s *Service) ValidateToken(tokenValue string) (*Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, exists := s.tokens[tokenValue]
	if !exists {
		return nil, ErrInvalidToken
	}

	if time.Now().After(token.ExpiresAt) {
		delete(s.tokens, tokenValue)
		delete(s.deviceTokens, token.DeviceID)
		return nil, ErrTokenExpired
	}

	// 更新最后使用时间
	token.LastUsedAt = time.Now()

	return token, nil
}

// RevokeToken revokes a token
func (s *Service) RevokeToken(tokenValue string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, exists := s.tokens[tokenValue]
	if !exists {
		return false
	}

	delete(s.tokens, tokenValue)
	delete(s.deviceTokens, token.DeviceID)

	log.Info().
		Str("deviceID", token.DeviceID).
		Msg("Token revoked")

	return true
}

// RevokeDevice revokes all tokens for a device
func (s *Service) RevokeDevice(deviceID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokenValue, exists := s.deviceTokens[deviceID]
	if !exists {
		return false
	}

	delete(s.tokens, tokenValue)
	delete(s.deviceTokens, deviceID)

	log.Info().
		Str("deviceID", deviceID).
		Msg("Device revoked")

	return true
}

// ListActiveDevices returns all active paired devices
func (s *Service) ListActiveDevices() []*Token {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	devices := make([]*Token, 0)

	for _, token := range s.tokens {
		if now.Before(token.ExpiresAt) {
			devices = append(devices, token)
		}
	}

	return devices
}

// RefreshToken extends token expiry
func (s *Service) RefreshToken(tokenValue string) (*Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, exists := s.tokens[tokenValue]
	if !exists {
		return nil, ErrInvalidToken
	}

	if time.Now().After(token.ExpiresAt) {
		delete(s.tokens, tokenValue)
		delete(s.deviceTokens, token.DeviceID)
		return nil, ErrTokenExpired
	}

	// 延长过期时间
	token.ExpiresAt = time.Now().Add(s.tokenExpiry)
	token.LastUsedAt = time.Now()

	return token, nil
}

// OnPairingComplete sets callback for successful pairing
func (s *Service) OnPairingComplete(callback func(deviceID, deviceName string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onPairingComplete = callback
}

// GetActivePairingCode returns current active pairing code if exists
func (s *Service) GetActivePairingCode() *PairingCode {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	for _, pc := range s.pairingCodes {
		if !pc.Used && now.Before(pc.ExpiresAt) {
			return pc
		}
	}
	return nil
}

// Internal methods

func (s *Service) createTokenLocked(deviceID, deviceName string) (*Token, error) {
	// 如果设备已有 token，先删除
	if oldToken, exists := s.deviceTokens[deviceID]; exists {
		delete(s.tokens, oldToken)
	}

	tokenValue, err := generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	now := time.Now()
	token := &Token{
		Value:       tokenValue,
		DeviceID:    deviceID,
		DeviceName:  deviceName,
		CreatedAt:   now,
		ExpiresAt:   now.Add(s.tokenExpiry),
		LastUsedAt:  now,
		Permissions: []string{"read", "write", "execute"},
	}

	s.tokens[tokenValue] = token
	s.deviceTokens[deviceID] = tokenValue

	return token, nil
}

func (s *Service) removeOldestTokenLocked() {
	var oldestToken *Token
	var oldestKey string

	for key, token := range s.tokens {
		if oldestToken == nil || token.LastUsedAt.Before(oldestToken.LastUsedAt) {
			oldestToken = token
			oldestKey = key
		}
	}

	if oldestToken != nil {
		delete(s.tokens, oldestKey)
		delete(s.deviceTokens, oldestToken.DeviceID)
		log.Info().
			Str("deviceID", oldestToken.DeviceID).
			Msg("Oldest device token removed")
	}
}

func (s *Service) cleanupExpiredCodesLocked() {
	now := time.Now()
	for code, pc := range s.pairingCodes {
		if now.After(pc.ExpiresAt) || pc.Used {
			delete(s.pairingCodes, code)
		}
	}
}

func (s *Service) cleanupExpired() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()

		now := time.Now()

		// 清理过期配对码
		for code, pc := range s.pairingCodes {
			if now.After(pc.ExpiresAt) || pc.Used {
				delete(s.pairingCodes, code)
			}
		}

		// 清理过期 Token
		for tokenValue, token := range s.tokens {
			if now.After(token.ExpiresAt) {
				delete(s.tokens, tokenValue)
				delete(s.deviceTokens, token.DeviceID)
			}
		}

		s.mu.Unlock()
	}
}

// Helper functions

func generateNumericCode(length int) (string, error) {
	const charset = "0123456789"
	result := make([]byte, length)
	randomBytes := make([]byte, length)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	for i := 0; i < length; i++ {
		result[i] = charset[randomBytes[i]%byte(len(charset))]
	}

	for i := 0; i < length; i++ {
		result[i] = charset[randomBytes[i]%byte(len(charset))]
	}

	return string(result), nil
}

func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// 添加时间戳增加唯一性
	timestamp := time.Now().UnixNano()
	hash := sha256.Sum256(append(bytes, byte(timestamp)))

	return base64.URLEncoding.EncodeToString(hash[:]), nil
}

// HashToken hashes a token for secure storage
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// persistedState represents the data saved to disk
type persistedState struct {
	Tokens       map[string]*Token `json:"tokens"`
	DeviceTokens map[string]string `json:"device_tokens"`
	SavedAt      time.Time         `json:"saved_at"`
}

// SaveState saves tokens to disk for persistence
func (s *Service) SaveState() error {
	if s.storagePath == "" {
		return nil // 未配置持久化
	}

	s.mu.RLock()
	state := persistedState{
		Tokens:       make(map[string]*Token),
		DeviceTokens: make(map[string]string),
		SavedAt:      time.Now(),
	}
	// 复制未过期的 Token
	now := time.Now()
	for k, v := range s.tokens {
		if now.Before(v.ExpiresAt) {
			state.Tokens[k] = v
		}
	}
	for k, v := range s.deviceTokens {
		if _, ok := state.Tokens[v]; ok {
			state.DeviceTokens[k] = v
		}
	}
	s.mu.RUnlock()

	// 确保目录存在
	dir := filepath.Dir(s.storagePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create storage dir: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(s.storagePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	log.Debug().Int("tokens", len(state.Tokens)).Msg("Auth state saved")
	return nil
}

// LoadState loads tokens from disk
func (s *Service) LoadState() error {
	if s.storagePath == "" {
		return nil
	}

	data, err := os.ReadFile(s.storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在，跳过
		}
		return fmt.Errorf("failed to read state file: %w", err)
	}

	var state persistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 只加载未过期的 Token
	now := time.Now()
	loaded := 0
	for k, v := range state.Tokens {
		if now.Before(v.ExpiresAt) {
			s.tokens[k] = v
			loaded++
		}
	}
	for k, v := range state.DeviceTokens {
		if _, ok := s.tokens[v]; ok {
			s.deviceTokens[k] = v
		}
	}

	log.Info().Int("loaded", loaded).Time("saved_at", state.SavedAt).Msg("Auth state loaded")
	return nil
}

// Errors
var (
	ErrInvalidCode     = &AuthError{Code: "INVALID_CODE", Message: "Invalid pairing code"}
	ErrCodeExpired     = &AuthError{Code: "CODE_EXPIRED", Message: "Pairing code has expired"}
	ErrCodeAlreadyUsed = &AuthError{Code: "CODE_USED", Message: "Pairing code already used"}
	ErrInvalidToken    = &AuthError{Code: "INVALID_TOKEN", Message: "Invalid token"}
	ErrTokenExpired    = &AuthError{Code: "TOKEN_EXPIRED", Message: "Token has expired"}
)

// AuthError represents an authentication error
type AuthError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AuthError) Error() string {
	return e.Message
}
