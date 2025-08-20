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

	// Enable CORS for all routes
	corsHandler := authMiddleware.CORS

	// Public auth routes
	mux.HandleFunc("GET /api/auth/status", corsHandler(authHandler.Status))
	mux.HandleFunc("POST /api/auth/setup", corsHandler(authHandler.Setup))
	mux.HandleFunc("POST /api/auth/login", corsHandler(authHandler.Login))
	mux.HandleFunc("POST /api/auth/logout", corsHandler(authHandler.Logout))
	mux.HandleFunc("GET /api/auth/me", corsHandler(authMiddleware.RequireAuth(authHandler.Me)))

	// Protected API routes
	mux.HandleFunc("GET /api/health", corsHandler(authMiddleware.RequireAuth(handler.Health)))
	mux.HandleFunc("GET /api/proxies", corsHandler(authMiddleware.RequireAuth(handler.GetProxies)))
	mux.HandleFunc("POST /api/proxies", corsHandler(authMiddleware.RequireAuth(handler.CreateProxy)))
	mux.HandleFunc("PUT /api/proxies/{id}", corsHandler(authMiddleware.RequireAuth(handler.UpdateProxy)))
	mux.HandleFunc("DELETE /api/proxies/{id}", corsHandler(authMiddleware.RequireAuth(handler.DeleteProxy)))
	mux.HandleFunc("GET /api/status", corsHandler(authMiddleware.RequireAuth(handler.Status)))
	mux.HandleFunc("POST /api/reload", corsHandler(authMiddleware.RequireAuth(handler.Reload)))

	// Static file serving for SPA
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "./static/" // Default for development
	}

	// Create file server for static files
	fs := http.FileServer(http.Dir(staticDir))

	// Handle SPA routing - serve index.html for non-API routes
	mux.HandleFunc("/", corsHandler(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// Check if the file exists, otherwise serve index.html
		if r.URL.Path != "/" {
			if _, err := os.Stat(staticDir + r.URL.Path); os.IsNotExist(err) {
				r.URL.Path = "/"
			}
		}

		fs.ServeHTTP(w, r)
	}))

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
