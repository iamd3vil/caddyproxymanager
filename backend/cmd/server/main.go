// Package main implements the main entry point for the Caddy Proxy Manager backend server.
// It provides a REST API for managing Caddy reverse proxy configurations with authentication.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sarat/caddyproxymanager/internal/handlers"
	"github.com/sarat/caddyproxymanager/pkg/audit"
	"github.com/sarat/caddyproxymanager/pkg/auth"
	"github.com/sarat/caddyproxymanager/pkg/caddy"
	"github.com/sarat/caddyproxymanager/pkg/health"
)

const (
	timeout60s               = 60 // Default timeout for HTTP operations in seconds
	readHeaderTimeoutSeconds = 30 // Maximum time to read request headers
	shutdownTimeoutSeconds   = 30 // Maximum time to wait for graceful shutdown
	defaultPort              = "8080"
	defaultCaddyAdminURL     = "http://localhost:2019"
	defaultDataDir           = "./data"
	defaultStaticDir         = "./static/"
	sessionCleanupInterval   = 1 * time.Hour // Interval for cleaning expired sessions
)

// serverConfig holds all configuration parameters for the proxy manager server
type serverConfig struct {
	port          string // Port for the HTTP server to listen on
	caddyAdminURL string // URL for the Caddy Admin API
	dataDir       string // Directory for storing persistent data
	configFile    string // Path to the Caddy configuration file
	staticDir     string // Directory for static assets
}

// getServerConfig retrieves server configuration from environment variables with fallback defaults
func getServerConfig() *serverConfig {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	caddyAdminURL := os.Getenv("CADDY_ADMIN_URL")
	if caddyAdminURL == "" {
		caddyAdminURL = defaultCaddyAdminURL
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = defaultDataDir
	}

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = defaultStaticDir
	}

	return &serverConfig{
		port:          port,
		caddyAdminURL: caddyAdminURL,
		dataDir:       dataDir,
		configFile:    filepath.Join(dataDir, "caddy-config.json"),
		staticDir:     staticDir,
	}
}

// initializeCaddy creates and configures a Caddy client, attempting to restore previous configuration
func initializeCaddy(cfg *serverConfig) *caddy.Client {
	caddyClient := caddy.New(cfg.caddyAdminURL, cfg.configFile)

	if err := caddyClient.RestoreConfigFromFile(); err != nil {
		log.Printf("Warning: Could not restore config from file: %v\n", err)
		log.Println("Starting with empty configuration...")
	} else {
		log.Printf("Configuration restored from: %s\n", cfg.configFile)
	}

	return caddyClient
}

// startHealthChecks initializes health monitoring for all configured proxies that have it enabled
func startHealthChecks(caddyClient *caddy.Client, healthService *health.Service) {
	config, err := caddyClient.GetConfig()
	if err != nil {
		return
	}

	proxies := caddyClient.ParseProxiesFromConfig(config)
	for _, proxy := range proxies {
		if proxy.HealthCheckEnabled {
			if err := healthService.StartHealthCheck(proxy); err != nil {
				log.Printf("Warning: Failed to start health check for proxy %s: %v\n", proxy.ID, err)
			}
		}
	}

	log.Printf("Started health checks for %d proxies\n", len(proxies))
}

// startSessionCleanup runs a background goroutine that periodically removes expired authentication sessions
func startSessionCleanup(ctx context.Context, authStorage *auth.Storage, waitGroup *sync.WaitGroup) {
	waitGroup.Add(1)

	tickerFunc := func() {
		defer waitGroup.Done()

		ticker := time.NewTicker(sessionCleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := authStorage.CleanExpiredSessions(); err != nil {
					log.Printf("Failed to clean expired sessions: %v", err)
				}
			case <-ctx.Done():
				log.Println("Session cleanup goroutine shutting down...")

				return
			}
		}
	}

	go tickerFunc()
}

// setupRoutes registers all HTTP routes for the API, separating public auth routes from protected routes
func setupRoutes(
	mux *http.ServeMux,
	handler *handlers.Handler,
	authHandler *handlers.AuthHandler,
	corsHandler func(http.HandlerFunc) http.HandlerFunc,
	authMiddleware *auth.Middleware,
) {
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
	mux.HandleFunc("GET /api/proxies/{id}/status", corsHandler(authMiddleware.RequireAuth(handler.GetProxyStatus)))
	mux.HandleFunc("GET /api/redirects", corsHandler(authMiddleware.RequireAuth(handler.GetRedirects)))
	mux.HandleFunc("POST /api/redirects", corsHandler(authMiddleware.RequireAuth(handler.CreateRedirect)))
	mux.HandleFunc("PUT /api/redirects/{id}", corsHandler(authMiddleware.RequireAuth(handler.UpdateRedirect)))
	mux.HandleFunc("DELETE /api/redirects/{id}", corsHandler(authMiddleware.RequireAuth(handler.DeleteRedirect)))
	mux.HandleFunc("GET /api/status", corsHandler(authMiddleware.RequireAuth(handler.Status)))
	mux.HandleFunc("POST /api/reload", corsHandler(authMiddleware.RequireAuth(handler.Reload)))
	mux.HandleFunc("GET /api/audit-log", corsHandler(authMiddleware.RequireAuth(handler.GetAuditLog)))
}

// setupStaticHandler configures serving of static files with SPA fallback support
func setupStaticHandler(mux *http.ServeMux, staticDir string, corsHandler func(http.HandlerFunc) http.HandlerFunc) {
	fileServer := http.FileServer(http.Dir(staticDir))

	mux.HandleFunc("/", corsHandler(func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasPrefix(request.URL.Path, "/api/") {
			http.NotFound(writer, request)

			return
		}

		if request.URL.Path != "/" {
			if _, err := os.Stat(staticDir + request.URL.Path); os.IsNotExist(err) {
				request.URL.Path = "/"
			}
		}

		fileServer.ServeHTTP(writer, request)
	}))
}

// createServer configures and returns an HTTP server with appropriate timeouts and limits
func createServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:                         ":" + port,
		Handler:                      handler,
		ReadHeaderTimeout:            readHeaderTimeoutSeconds * time.Second,
		ReadTimeout:                  timeout60s * time.Second,
		WriteTimeout:                 timeout60s * time.Second,
		IdleTimeout:                  timeout60s * time.Second,
		MaxHeaderBytes:               1 << 20,
		DisableGeneralOptionsHandler: false,
	}
}

// startServer launches the HTTP server in a goroutine with configuration logging
func startServer(server *http.Server, cfg *serverConfig, waitGroup *sync.WaitGroup) {
	waitGroup.Add(1)

	serverFunc := func() {
		defer waitGroup.Done()
		log.Printf("Server starting on port %s\n", cfg.port)
		log.Printf("Caddy Admin API: %s\n", cfg.caddyAdminURL)
		log.Printf("Config file: %s\n", cfg.configFile)
		log.Printf("Data directory: %s\n", cfg.dataDir)

		if os.Getenv("DISABLE_AUTH") == "true" {
			log.Println("Authentication: DISABLED")
		} else {
			log.Println("Authentication: ENABLED")
		}

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server failed to start: %v", err)
		}
	}

	go serverFunc()
}

// initializeAuthStorage creates and initializes the authentication storage system
func initializeAuthStorage(dataDir string) *auth.Storage {
	authStorage := auth.NewStorage(dataDir)
	if err := authStorage.Initialize(); err != nil {
		log.Fatalf("Failed to initialize auth storage: %v", err)
	}

	return authStorage
}

// gracefulShutdown handles server shutdown by stopping HTTP server and waiting for all goroutines to complete
func gracefulShutdown(server *http.Server, waitGroup *sync.WaitGroup, cancel context.CancelFunc) {
	log.Println("\nShutdown signal received, initiating graceful shutdown...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeoutSeconds*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	} else {
		log.Println("HTTP server gracefully stopped")
	}

	log.Println("Waiting for goroutines to finish...")
	waitGroup.Wait()
	log.Println("All goroutines finished, graceful shutdown completed")
}

// main is the entry point that initializes and orchestrates all server components
func main() {
	var waitGroup sync.WaitGroup

	// Set up signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Load configuration and initialize core services
	cfg := getServerConfig()
	caddyClient := initializeCaddy(cfg)

	// Initialize health monitoring system
	healthService := health.NewService()
	startHealthChecks(caddyClient, healthService)

	// Set up authentication system
	authStorage := initializeAuthStorage(cfg.dataDir)
	startSessionCleanup(ctx, authStorage, &waitGroup)

	// Initialize audit logging
	auditService := audit.NewService(cfg.dataDir)

	// Create HTTP handlers and middleware
	handler := handlers.New(caddyClient, healthService, auditService)
	authHandler := handlers.NewAuthHandler(authStorage, auditService)
	authMiddleware := auth.NewMiddleware(authStorage)

	// Configure HTTP routing
	mux := http.NewServeMux()
	corsHandler := authMiddleware.CORS

	setupRoutes(mux, handler, authHandler, corsHandler, authMiddleware)
	setupStaticHandler(mux, cfg.staticDir, corsHandler)

	// Start the HTTP server
	server := createServer(cfg.port, mux)
	startServer(server, cfg, &waitGroup)

	// Wait for shutdown signal
	<-ctx.Done()
	gracefulShutdown(server, &waitGroup, cancel)
}
