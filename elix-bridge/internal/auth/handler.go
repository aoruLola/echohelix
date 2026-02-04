// Package auth provides authentication handlers
package auth

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Handler handles authentication requests
type Handler struct {
	service *Service
}

// NewHandler creates a new auth handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// HandlePair handles pairing requests (Mobile App -> Bridge)
func (h *Handler) HandlePair(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Code       string `json:"code"`
		DeviceID   string `json:"device_id"`
		DeviceName string `json:"device_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	if req.Code == "" || req.DeviceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Code and Device ID are required",
		})
		return
	}

	token, err := h.service.ValidatePairingCode(req.Code, req.DeviceID, req.DeviceName)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(token)
}

// HandleGenerateCode generates a new pairing code (Desktop UI/CLI -> Bridge)
// This should optimally be protected or only accessible from localhost
func (h *Handler) HandleGenerateCode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Helper to check if request is from localhost
	// In production, this should have stricter checks
	if !isLocalRequest(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	code, err := h.service.GeneratePairingCode()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(code)
}

// HandleStatus checks token status
func (h *Handler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	token := extractToken(r)
	if token == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	tokenInfo, err := h.service.ValidateToken(token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "valid",
		"token":  tokenInfo,
	})
}

// AuthenticateMiddleware authenticates requests
func (h *Handler) AuthenticateMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow OPTIONS for CORS
		if r.Method == http.MethodOptions {
			next(w, r)
			return
		}

		token := extractToken(r)
		if token == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Authentication required",
			})
			return
		}

		if _, err := h.service.ValidateToken(token); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Invalid or expired token",
			})
			return
		}

		next(w, r)
	}
}

// Helper functions

func extractToken(r *http.Request) string {
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	// Check query param
	return r.URL.Query().Get("token")
}

func isLocalRequest(r *http.Request) bool {
	// 检查 X-Forwarded-For 头（如果存在则拒绝，因为有代理）
	if r.Header.Get("X-Forwarded-For") != "" {
		return false
	}

	// 获取远程地址，去掉端口号
	remoteAddr := r.RemoteAddr
	host := remoteAddr

	// 处理带端口的地址格式
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		// 检查是否是 IPv6 格式 [::1]:port
		if strings.HasPrefix(remoteAddr, "[") {
			if bracketIdx := strings.Index(remoteAddr, "]"); bracketIdx != -1 {
				host = remoteAddr[1:bracketIdx]
			}
		} else {
			host = remoteAddr[:idx]
		}
	}

	// 检查本地地址
	localAddrs := []string{
		"127.0.0.1",
		"::1",
		"localhost",
	}

	for _, addr := range localAddrs {
		if host == addr {
			return true
		}
	}

	return false
}
