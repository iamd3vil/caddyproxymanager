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
	"github.com/sarat/caddyproxymanager/pkg/auth"
	"github.com/sarat/caddyproxymanager/pkg/caddy"
	"github.com/sarat/caddyproxymanager/pkg/health"
)

const (
	magicNumber60            = 60
	magicNumber20            = 20
	readHeaderTimeoutSeconds = 30
	shutdownTimeoutSeconds   = 30
	defaultPort              = "8080"
	defaultCaddyAdminURL     = "http://localhost:2019"
	defaultDataDir           = "./data"
	defaultStaticDir         = "./static/"
	sessionCleanupInterval   = 1 * time.Hour
)

type serverConfig struct {
	port          string
	caddyAdminURL string
	dataDir       string
	configFile    string
	staticDir     string
}

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
	mux.HandleFunc("GET /api/status", corsHandler(authMiddleware.RequireAuth(handler.Status)))
	mux.HandleFunc("POST /api/reload", corsHandler(authMiddleware.RequireAuth(handler.Reload)))
}

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

func createServer(port string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:                         ":" + port,
		Handler:                      handler,
		ReadHeaderTimeout:            readHeaderTimeoutSeconds * time.Second,
		ReadTimeout:                  magicNumber60 * time.Second,
		WriteTimeout:                 magicNumber60 * time.Second,
		IdleTimeout:                  magicNumber60 * time.Second,
		MaxHeaderBytes:               1 << magicNumber20,
		DisableGeneralOptionsHandler: false,
		TLSConfig:                    nil,
		TLSNextProto:                 nil,
		ConnState:                    nil,
		ErrorLog:                     nil,
		BaseContext:                  nil,
		ConnContext:                  nil,
		HTTP2:                        nil,
		Protocols:                    nil,
	}
}

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

func initializeAuthStorage(dataDir string) *auth.Storage {
	authStorage := auth.NewStorage(dataDir)
	if err := authStorage.Initialize(); err != nil {
		log.Fatalf("Failed to initialize auth storage: %v", err)
	}

	return authStorage
}

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

func main() {
	var waitGroup sync.WaitGroup

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg := getServerConfig()
	caddyClient := initializeCaddy(cfg)

	healthService := health.NewService()
	startHealthChecks(caddyClient, healthService)

	authStorage := initializeAuthStorage(cfg.dataDir)

	startSessionCleanup(ctx, authStorage, &waitGroup)

	handler := handlers.New(caddyClient, healthService)
	authHandler := handlers.NewAuthHandler(authStorage)
	authMiddleware := auth.NewMiddleware(authStorage)

	mux := http.NewServeMux()
	corsHandler := authMiddleware.CORS

	setupRoutes(mux, handler, authHandler, corsHandler, authMiddleware)
	setupStaticHandler(mux, cfg.staticDir, corsHandler)

	server := createServer(cfg.port, mux)
	startServer(server, cfg, &waitGroup)

	<-ctx.Done()
	gracefulShutdown(server, &waitGroup, cancel)
}
