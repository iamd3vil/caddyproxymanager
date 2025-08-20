package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/sarat/caddyproxymanager/pkg/auth"
	"github.com/sarat/caddyproxymanager/pkg/models"
)

type AuthHandler struct {
	storage *auth.Storage
}

func NewAuthHandler(storage *auth.Storage) *AuthHandler {
	return &AuthHandler{storage: storage}
}

func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	response := models.StatusResponse{
		IsSetup:     h.storage.IsSetup(),
		AuthEnabled: os.Getenv("DISABLE_AUTH") != "true",
	}
	
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) Setup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Check if auth is disabled
	if os.Getenv("DISABLE_AUTH") == "true" {
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
	
	json.NewEncoder(w).Encode(models.AuthResponse{
		Success: true,
		Message: "Setup completed successfully",
		Token:   session.Token,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Check if auth is disabled
	if os.Getenv("DISABLE_AUTH") == "true" {
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
	
	json.NewEncoder(w).Encode(models.AuthResponse{
		Success: true,
		Message: "Login successful",
		Token:   session.Token,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Check if auth is disabled
	if os.Getenv("DISABLE_AUTH") == "true" {
		json.NewEncoder(w).Encode(models.AuthResponse{
			Success: true,
			Message: "Logged out",
		})
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
	
	json.NewEncoder(w).Encode(models.AuthResponse{
		Success: true,
		Message: "Logged out successfully",
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Check if auth is disabled
	if os.Getenv("DISABLE_AUTH") == "true" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"user": map[string]string{
				"username": "disabled",
			},
		})
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
	
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) badRequest(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(models.AuthResponse{
		Success: false,
		Message: message,
	})
}

func (h *AuthHandler) unauthorized(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(models.AuthResponse{
		Success: false,
		Message: message,
	})
}

func (h *AuthHandler) forbidden(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(models.AuthResponse{
		Success: false,
		Message: message,
	})
}

func (h *AuthHandler) internalError(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(models.AuthResponse{
		Success: false,
		Message: message,
	})
}