package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sarat/caddyproxymanager/internal/handlers"
	"github.com/sarat/caddyproxymanager/pkg/auth"
	"github.com/sarat/caddyproxymanager/pkg/caddy"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Initialize Caddy client - assuming Caddy admin API runs on localhost:2019
	caddyAdminURL := os.Getenv("CADDY_ADMIN_URL")
	if caddyAdminURL == "" {
		caddyAdminURL = "http://localhost:2019"
	}

	// Config file path for persistence
	configFile := os.Getenv("CADDY_CONFIG_FILE")
	if configFile == "" {
		configFile = "./caddy-config.json"
	}

	caddyClient := caddy.New(caddyAdminURL, configFile)

	// Try to restore configuration from file on startup
	if err := caddyClient.RestoreConfigFromFile(); err != nil {
		fmt.Printf("Warning: Could not restore config from file: %v\n", err)
		fmt.Println("Starting with empty configuration...")
	} else {
		fmt.Printf("Configuration restored from: %s\n", configFile)
	}

	// Initialize auth storage
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	authStorage := auth.NewStorage(dataDir)
	if err := authStorage.Initialize(); err != nil {
		log.Fatalf("Failed to initialize auth storage: %v", err)
	}

	// Start session cleanup goroutine
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := authStorage.CleanExpiredSessions(); err != nil {
					log.Printf("Failed to clean expired sessions: %v", err)
				}
			}
		}
	}()

	handler := handlers.New(caddyClient)
	authHandler := handlers.NewAuthHandler(authStorage)
	authMiddleware := auth.NewMiddleware(authStorage)

	mux := http.NewServeMux()

	// Enable CORS and setup routes
	mux.HandleFunc("/", authMiddleware.CORS(authMiddleware.CheckSetup(routeHandler(handler, authHandler, authMiddleware))))

	fmt.Printf("Server starting on port %s\n", port)
	fmt.Printf("Caddy Admin API: %s\n", caddyAdminURL)
	fmt.Printf("Config file: %s\n", configFile)
	fmt.Printf("Data directory: %s\n", dataDir)
	if os.Getenv("DISABLE_AUTH") == "true" {
		fmt.Println("Authentication: DISABLED")
	} else {
		fmt.Println("Authentication: ENABLED")
	}
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

// routeHandler wraps all requests and routes to appropriate handlers with auth
func routeHandler(h *handlers.Handler, authHandler *handlers.AuthHandler, authMiddleware *auth.Middleware) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Route to appropriate handler
		switch {
		// Auth routes (public)
		case r.URL.Path == "/api/auth/status" && r.Method == "GET":
			authHandler.Status(w, r)
		case r.URL.Path == "/api/auth/setup" && r.Method == "POST":
			authHandler.Setup(w, r)
		case r.URL.Path == "/api/auth/login" && r.Method == "POST":
			authHandler.Login(w, r)
		case r.URL.Path == "/api/auth/logout" && r.Method == "POST":
			authHandler.Logout(w, r)
		case r.URL.Path == "/api/auth/me" && r.Method == "GET":
			authMiddleware.RequireAuth(authHandler.Me)(w, r)
		// Protected API routes
		case r.URL.Path == "/api/health" && r.Method == "GET":
			authMiddleware.RequireAuth(h.Health)(w, r)
		case r.URL.Path == "/api/proxies" && r.Method == "GET":
			authMiddleware.RequireAuth(h.GetProxies)(w, r)
		case r.URL.Path == "/api/proxies" && r.Method == "POST":
			authMiddleware.RequireAuth(h.CreateProxy)(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/proxies/") && r.Method == "PUT":
			authMiddleware.RequireAuth(h.UpdateProxy)(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/proxies/") && r.Method == "DELETE":
			authMiddleware.RequireAuth(h.DeleteProxy)(w, r)
		case r.URL.Path == "/api/status" && r.Method == "GET":
			authMiddleware.RequireAuth(h.Status)(w, r)
		case r.URL.Path == "/api/reload" && r.Method == "POST":
			authMiddleware.RequireAuth(h.Reload)(w, r)
		default:
			// Serve static files for frontend
			staticDir := os.Getenv("STATIC_DIR")
			if staticDir == "" {
				staticDir = "./static/" // Default for development
			}

			fs := http.FileServer(http.Dir(staticDir))

			// Handle SPA routing - serve index.html for non-API routes
			if r.URL.Path != "/" && !strings.HasPrefix(r.URL.Path, "/api/") {
				if _, err := os.Stat(staticDir + r.URL.Path); os.IsNotExist(err) {
					r.URL.Path = "/"
				}
			}

			fs.ServeHTTP(w, r)
		}
	}
}
