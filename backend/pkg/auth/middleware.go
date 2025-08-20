package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/sarat/caddyproxymanager/pkg/models"
)

type contextKey string

const (
	UserContextKey    contextKey = "user"
	SessionContextKey contextKey = "session"
)

type Middleware struct {
	storage *Storage
}

func NewMiddleware(storage *Storage) *Middleware {
	return &Middleware{storage: storage}
}

func (m *Middleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if auth is disabled
		if os.Getenv("DISABLE_AUTH") == "true" {
			next.ServeHTTP(w, r)
			return
		}

		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			m.unauthorized(w, "Authorization header required")
			return
		}

		// Extract Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			m.unauthorized(w, "Invalid authorization header format")
			return
		}

		token := parts[1]
		if token == "" {
			m.unauthorized(w, "Token required")
			return
		}

		// Validate session
		session, err := m.storage.GetSession(token)
		if err != nil {
			m.unauthorized(w, "Invalid or expired session")
			return
		}

		// Get user (optional, for additional context)
		user, _ := m.storage.GetUserByID(session.UserID)

		// Add to context
		ctx := context.WithValue(r.Context(), SessionContextKey, session)
		if user != nil {
			ctx = context.WithValue(ctx, UserContextKey, user)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (m *Middleware) OptionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if auth is disabled
		if os.Getenv("DISABLE_AUTH") == "true" {
			next.ServeHTTP(w, r)
			return
		}

		// Try to get token, but don't fail if it's missing
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				token := parts[1]
				if token != "" {
					// Validate session
					if session, err := m.storage.GetSession(token); err == nil {
						// Get user
						user, _ := m.storage.GetUserByID(session.UserID)

						// Add to context
						ctx := context.WithValue(r.Context(), SessionContextKey, session)
						if user != nil {
							ctx = context.WithValue(ctx, UserContextKey, user)
						}
						r = r.WithContext(ctx)
					}
				}
			}
		}

		next.ServeHTTP(w, r)
	}
}

func (m *Middleware) CheckSetup(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if auth is disabled
		if os.Getenv("DISABLE_AUTH") == "true" {
			next.ServeHTTP(w, r)
			return
		}

		// If not setup, only allow setup and status endpoints
		if !m.storage.IsSetup() {
			allowedPaths := []string{"/api/auth/status", "/api/auth/setup"}
			for _, path := range allowedPaths {
				if r.URL.Path == path {
					next.ServeHTTP(w, r)
					return
				}
			}
			// Also allow static files and all frontend routes (SPA routing)
			if strings.HasPrefix(r.URL.Path, "/static/") || 
			   r.URL.Path == "/" || 
			   r.URL.Path == "/index.html" || 
			   !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}

			m.forbidden(w, "Setup required")
			return
		}

		next.ServeHTTP(w, r)
	}
}

func (m *Middleware) CORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func (m *Middleware) unauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(models.AuthResponse{
		Success: false,
		Message: message,
	})
}

func (m *Middleware) forbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(models.AuthResponse{
		Success: false,
		Message: message,
	})
}

func GetUserFromContext(ctx context.Context) *models.User {
	if user, ok := ctx.Value(UserContextKey).(*models.User); ok {
		return user
	}
	return nil
}

func GetSessionFromContext(ctx context.Context) *models.Session {
	if session, ok := ctx.Value(SessionContextKey).(*models.Session); ok {
		return session
	}
	return nil
}