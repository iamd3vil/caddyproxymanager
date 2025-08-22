package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sarat/caddyproxymanager/pkg/caddy"
	"github.com/sarat/caddyproxymanager/pkg/models"
)

type Handler struct {
	CaddyClient *caddy.Client
}

func New(caddyClient *caddy.Client) *Handler {
	return &Handler{
		CaddyClient: caddyClient,
	}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status": "ok", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`)); err != nil {
		// Log error if needed, but response is already written
		return
	}
}

func (h *Handler) GetProxies(w http.ResponseWriter, r *http.Request) {
	// Get current Caddy configuration
	config, err := h.CaddyClient.GetConfig()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to get Caddy config: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// Parse proxies from config
	proxies := h.CaddyClient.ParseProxiesFromConfig(config)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"proxies": proxies,
		"count":   len(proxies),
	}); err != nil {
		// Log error if needed, but response is already written
		return
	}
}

func (h *Handler) CreateProxy(w http.ResponseWriter, r *http.Request) {
	var proxyReq struct {
		Domain         string            `json:"domain"`
		TargetURL      string            `json:"target_url"`
		SSLMode        string            `json:"ssl_mode"`
		ChallengeType  string            `json:"challenge_type"`
		DNSProvider    string            `json:"dns_provider"`
		DNSCredentials map[string]string `json:"dns_credentials"`
		CustomHeaders  map[string]string `json:"custom_headers"`
		BasicAuth      *models.BasicAuth `json:"basic_auth"`
	}

	if err := json.NewDecoder(r.Body).Decode(&proxyReq); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	// Validate required fields
	if proxyReq.Domain == "" || proxyReq.TargetURL == "" {
		http.Error(w, `{"error": "Domain and target_url are required"}`, http.StatusBadRequest)
		return
	}

	// Set defaults if not provided
	if proxyReq.SSLMode == "" {
		proxyReq.SSLMode = "auto"
	}
	if proxyReq.ChallengeType == "" {
		proxyReq.ChallengeType = "http"
	}

	// Validate DNS challenge configuration
	if proxyReq.SSLMode == "auto" && proxyReq.ChallengeType == "dns" {
		if proxyReq.DNSProvider == "" {
			http.Error(w, `{"error": "DNS provider is required for DNS challenge"}`, http.StatusBadRequest)
			return
		}

		// Validate DNS credentials based on provider
		if err := h.validateDNSCredentials(proxyReq.DNSProvider, proxyReq.DNSCredentials); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%v"}`, err), http.StatusBadRequest)
			return
		}
	}

	// Create new proxy
	proxy := models.NewProxy(proxyReq.Domain, proxyReq.TargetURL, proxyReq.SSLMode)
	proxy.ChallengeType = proxyReq.ChallengeType
	proxy.DNSProvider = proxyReq.DNSProvider
	proxy.DNSCredentials = proxyReq.DNSCredentials
	proxy.CustomHeaders = proxyReq.CustomHeaders
	proxy.BasicAuth = proxyReq.BasicAuth

	// Add proxy to Caddy configuration
	if err := h.CaddyClient.AddProxy(*proxy); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to add proxy to Caddy: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(proxy); err != nil {
		// Log error if needed, but response is already written
		return
	}
}

func (h *Handler) UpdateProxy(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		http.Error(w, `{"error": "Invalid proxy ID"}`, http.StatusBadRequest)
		return
	}

	var proxyReq struct {
		Domain         string            `json:"domain"`
		TargetURL      string            `json:"target_url"`
		SSLMode        string            `json:"ssl_mode"`
		ChallengeType  string            `json:"challenge_type"`
		DNSProvider    string            `json:"dns_provider"`
		DNSCredentials map[string]string `json:"dns_credentials"`
		CustomHeaders  map[string]string `json:"custom_headers"`
		BasicAuth      *models.BasicAuth `json:"basic_auth"`
	}

	if err := json.NewDecoder(r.Body).Decode(&proxyReq); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	// Validate required fields
	if proxyReq.Domain == "" || proxyReq.TargetURL == "" {
		http.Error(w, `{"error": "Domain and target_url are required"}`, http.StatusBadRequest)
		return
	}

	// Set defaults if not provided
	if proxyReq.SSLMode == "" {
		proxyReq.SSLMode = "auto"
	}
	if proxyReq.ChallengeType == "" {
		proxyReq.ChallengeType = "http"
	}

	// Validate DNS challenge configuration
	if proxyReq.SSLMode == "auto" && proxyReq.ChallengeType == "dns" {
		if proxyReq.DNSProvider == "" {
			http.Error(w, `{"error": "DNS provider is required for DNS challenge"}`, http.StatusBadRequest)
			return
		}

		// Validate DNS credentials based on provider
		if err := h.validateDNSCredentials(proxyReq.DNSProvider, proxyReq.DNSCredentials); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%v"}`, err), http.StatusBadRequest)
			return
		}
	}

	// Create updated proxy
	proxy := models.NewProxy(proxyReq.Domain, proxyReq.TargetURL, proxyReq.SSLMode)
	proxy.ID = id
	proxy.ChallengeType = proxyReq.ChallengeType
	proxy.DNSProvider = proxyReq.DNSProvider
	proxy.DNSCredentials = proxyReq.DNSCredentials
	proxy.CustomHeaders = proxyReq.CustomHeaders
	proxy.BasicAuth = proxyReq.BasicAuth
	proxy.UpdateTimestamp()

	// Update proxy in Caddy configuration
	if err := h.CaddyClient.UpdateProxy(*proxy); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to update proxy in Caddy: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(proxy); err != nil {
		// Log error if needed, but response is already written
		return
	}
}

func (h *Handler) DeleteProxy(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromPath(r.URL.Path)
	if id == "" {
		http.Error(w, `{"error": "Invalid proxy ID"}`, http.StatusBadRequest)
		return
	}

	// Remove proxy from Caddy configuration
	if err := h.CaddyClient.DeleteProxy(id); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to delete proxy from Caddy: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(fmt.Sprintf(`{"message": "Proxy %s deleted successfully"}`, id))); err != nil {
		// Log error if needed, but response is already written
		return
	}
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	// Check Caddy status
	status, err := h.CaddyClient.GetStatus()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if encErr := json.NewEncoder(w).Encode(map[string]any{
			"caddy_status":    "error",
			"caddy_reachable": false,
			"error":           err.Error(),
			"last_checked":    time.Now().Format(time.RFC3339),
		}); encErr != nil {
			// Log error if needed, but response is already written
			return
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"caddy_status":    "running",
		"caddy_reachable": true,
		"upstreams":       status,
		"last_checked":    time.Now().Format(time.RFC3339),
	}); err != nil {
		// Log error if needed, but response is already written
		return
	}
}

func (h *Handler) Reload(w http.ResponseWriter, r *http.Request) {
	if err := h.CaddyClient.Reload(); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to reload Caddy: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"message": "Caddy configuration reloaded successfully"}`)); err != nil {
		// Log error if needed, but response is already written
		return
	}
}

// extractIDFromPath extracts ID from path like /api/proxies/proxy_example_com_1234567890
// validateDNSCredentials validates DNS provider credentials with environment variable fallback
func (h *Handler) validateDNSCredentials(provider string, credentials map[string]string) error {
	switch provider {
	case "cloudflare":
		apiToken := credentials["api_token"]
		// Check if token is provided in request or available as environment variable
		if apiToken == "" && os.Getenv("CLOUDFLARE_API_TOKEN") == "" {
			return fmt.Errorf("Cloudflare API token is required (provide in request or set CLOUDFLARE_API_TOKEN environment variable)")
		}
		// Optional email validation
		if email := credentials["email"]; email != "" {
			if !strings.Contains(email, "@") {
				return fmt.Errorf("Invalid email format")
			}
		}
	case "digitalocean":
		authToken := credentials["auth_token"]
		if authToken == "" && os.Getenv("DO_AUTH_TOKEN") == "" {
			return fmt.Errorf("DigitalOcean auth token is required (provide in request or set DO_AUTH_TOKEN environment variable)")
		}
	case "duckdns":
		token := credentials["token"]
		if token == "" && os.Getenv("DUCKDNS_TOKEN") == "" {
			return fmt.Errorf("DuckDNS token is required (provide in request or set DUCKDNS_TOKEN environment variable)")
		}
	default:
		return fmt.Errorf("Unsupported DNS provider: %s", provider)
	}
	return nil
}

func extractIDFromPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) >= 4 {
		return parts[3]
	}
	return ""
}
