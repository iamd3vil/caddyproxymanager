package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/sarat/caddyproxymanager/pkg/audit"
	"github.com/sarat/caddyproxymanager/pkg/auth"
	"github.com/sarat/caddyproxymanager/pkg/models"
)

// Constants for repeated strings
const (
	AuthTrue = "true"
)

type AuthHandler struct {
	storage      *auth.Storage
	auditService *audit.Service
}

func NewAuthHandler(storage *auth.Storage, auditService *audit.Service) *AuthHandler {
	return &AuthHandler{
		storage:      storage,
		auditService: auditService,
	}
}

func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := models.StatusResponse{
		IsSetup:     h.storage.IsSetup(),
		AuthEnabled: os.Getenv("DISABLE_AUTH") != AuthTrue,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log error if needed, but response is already written
	}
}

func (h *AuthHandler) Setup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check if auth is disabled
	if os.Getenv("DISABLE_AUTH") == AuthTrue {
		h.badRequest(w, "Authentication is disabled")
		return
	}

	// Check if already setup
	if h.storage.IsSetup() {
		h.badRequest(w, "System already setup")
		return
	}

	var req models.SetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "Invalid request body")
		return
	}

	// Validate input
	if strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Password) == "" {
		h.badRequest(w, "Username and password are required")
		return
	}

	if len(req.Password) < 6 {
		h.badRequest(w, "Password must be at least 6 characters")
		return
	}

	// Create user
	user, err := h.storage.CreateUser(req.Username, req.Password)
	if err != nil {
		h.internalError(w, "Failed to create user: "+err.Error())
		return
	}

	// Create session
	session, err := h.storage.CreateSession(user.ID)
	if err != nil {
		h.internalError(w, "Failed to create session: "+err.Error())
		return
	}

	// Log setup action
	if h.auditService != nil {
		ipAddress := r.RemoteAddr
		if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
			ipAddress = ip
		}
		h.auditService.Log("SETUP_SUCCESS", "System setup completed", user.ID, req.Username, ipAddress)
	}

	if err := json.NewEncoder(w).Encode(models.AuthResponse{
		Success: true,
		Message: "Setup completed successfully",
		Token:   session.Token,
	}); err != nil {
		// Log error if needed, but response is already written
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check if auth is disabled
	if os.Getenv("DISABLE_AUTH") == AuthTrue {
		h.badRequest(w, "Authentication is disabled")
		return
	}

	// Check if setup is required
	if !h.storage.IsSetup() {
		h.forbidden(w, "Setup required")
		return
	}

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "Invalid request body")
		return
	}

	// Validate input
	if strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Password) == "" {
		h.badRequest(w, "Username and password are required")
		return
	}

	// Get user
	user, err := h.storage.GetUserByUsername(req.Username)
	if err != nil {
		h.unauthorized(w, "Invalid credentials")
		return
	}

	// Check password
	if !auth.CheckPassword(req.Password, user.Password) {
		h.unauthorized(w, "Invalid credentials")
		return
	}

	// Create session
	session, err := h.storage.CreateSession(user.ID)
	if err != nil {
		h.internalError(w, "Failed to create session: "+err.Error())
		return
	}

	// Log login action
	if h.auditService != nil {
		ipAddress := r.RemoteAddr
		if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
			ipAddress = ip
		}
		h.auditService.Log("LOGIN_SUCCESS", "User logged in", user.ID, req.Username, ipAddress)
	}

	if err := json.NewEncoder(w).Encode(models.AuthResponse{
		Success: true,
		Message: "Login successful",
		Token:   session.Token,
	}); err != nil {
		// Log error if needed, but response is already written
	}
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check if auth is disabled
	if os.Getenv("DISABLE_AUTH") == AuthTrue {
		if err := json.NewEncoder(w).Encode(models.AuthResponse{
			Success: true,
			Message: "Logged out",
		}); err != nil {
			// Log error if needed, but response is already written
		}
		return
	}

	// Get token from header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		h.badRequest(w, "Authorization header required")
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		h.badRequest(w, "Invalid authorization header format")
		return
	}

	token := parts[1]
	if token == "" {
		h.badRequest(w, "Token required")
		return
	}

	// Delete session
	if err := h.storage.DeleteSession(token); err != nil {
		// Don't return error if session doesn't exist
	}

	// Log logout action
	if h.auditService != nil {
		ipAddress := r.RemoteAddr
		if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
			ipAddress = ip
		}
		// Try to get user info from context
		user := auth.GetUserFromContext(r.Context())
		username := "unknown"
		userID := "unknown"
		if user != nil {
			username = user.Username
			userID = user.ID
		}
		h.auditService.Log("LOGOUT_SUCCESS", "User logged out", userID, username, ipAddress)
	}

	if err := json.NewEncoder(w).Encode(models.AuthResponse{
		Success: true,
		Message: "Logged out successfully",
	}); err != nil {
		// Log error if needed, but response is already written
	}
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check if auth is disabled
	if os.Getenv("DISABLE_AUTH") == AuthTrue {
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"user": map[string]string{
				"username": "disabled",
			},
		}); err != nil {
			// Log error if needed, but response is already written
		}
		return
	}

	// Get user from context (set by middleware)
	user := auth.GetUserFromContext(r.Context())
	if user == nil {
		h.unauthorized(w, "Not authenticated")
		return
	}

	// Return user info (without password)
	response := map[string]interface{}{
		"success": true,
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"created":  user.Created,
			"updated":  user.Updated,
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log error if needed, but response is already written
	}
}

func (h *AuthHandler) badRequest(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusBadRequest)
	if err := json.NewEncoder(w).Encode(models.AuthResponse{
		Success: false,
		Message: message,
	}); err != nil {
		// Log error if needed, but response is already written
	}
}

func (h *AuthHandler) unauthorized(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusUnauthorized)
	if err := json.NewEncoder(w).Encode(models.AuthResponse{
		Success: false,
		Message: message,
	}); err != nil {
		// Log error if needed, but response is already written
	}
}

func (h *AuthHandler) forbidden(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusForbidden)
	if err := json.NewEncoder(w).Encode(models.AuthResponse{
		Success: false,
		Message: message,
	}); err != nil {
		// Log error if needed, but response is already written
	}
}

func (h *AuthHandler) internalError(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusInternalServerError)
	if err := json.NewEncoder(w).Encode(models.AuthResponse{
		Success: false,
		Message: message,
	}); err != nil {
		// Log error if needed, but response is already written
	}
}
