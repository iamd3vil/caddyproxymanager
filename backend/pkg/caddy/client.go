package caddy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/sarat/caddyproxymanager/pkg/models"
	"golang.org/x/crypto/bcrypt"
)

// Constants for repeated strings
const (
	SSLModeAuto = "auto"
	SSLModeNone = "none"
)

// Client handles communication with Caddy Admin API
type Client struct {
	BaseURL      string
	Client       *http.Client
	ConfigFile   string
	MetadataFile string
	metadata     *models.MetadataStore
}

// New creates a new Caddy API client
func New(baseURL, configFile string) *Client {
	dir := filepath.Dir(configFile)
	base := strings.TrimSuffix(filepath.Base(configFile), ".json")
	metadataFile := filepath.Join(dir, base+"-metadata.json")
	client := &Client{
		BaseURL:      baseURL,
		ConfigFile:   configFile,
		MetadataFile: metadataFile,
		metadata:     models.NewMetadataStore(),
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Load existing metadata
	if err := client.loadMetadataFromFile(); err != nil {
		log.Printf("Warning: Failed to load metadata: %v", err)
	}

	return client
}

// validateIPOrCIDR validates if a string is a valid IP address or CIDR range
func validateIPOrCIDR(ipOrCIDR string) error {
	// Try parsing as IP address first
	if ip := net.ParseIP(ipOrCIDR); ip != nil {
		return nil
	}

	// Try parsing as CIDR range
	if _, _, err := net.ParseCIDR(ipOrCIDR); err == nil {
		return nil
	}

	return fmt.Errorf("invalid IP address or CIDR range: %s", ipOrCIDR)
}

// validateIPList validates a list of IP addresses or CIDR ranges
func validateIPList(ips []string) error {
	for _, ip := range ips {
		if ip = strings.TrimSpace(ip); ip != "" {
			if err := validateIPOrCIDR(ip); err != nil {
				return err
			}
		}
	}
	return nil
}

// getCredential is a helper to get a credential from proxy config or environment variable
func getCredential(proxy models.Proxy, key, envVar string) string {
	if val, ok := proxy.DNSCredentials[key]; ok && val != "" {
		return val
	}
	return os.Getenv(envVar)
}

// dnsConfigurators maps DNS provider names to their configuration functions
var dnsConfigurators = map[string]func(*models.CaddyDNSProvider, models.Proxy){
	"cloudflare": func(dp *models.CaddyDNSProvider, p models.Proxy) {
		dp.APIToken = getCredential(p, "api_token", "CLOUDFLARE_API_TOKEN")
		dp.Email = getCredential(p, "email", "CLOUDFLARE_EMAIL")
	},
	"digitalocean": func(dp *models.CaddyDNSProvider, p models.Proxy) {
		dp.AuthToken = getCredential(p, "auth_token", "DO_AUTH_TOKEN")
	},
	"duckdns": func(dp *models.CaddyDNSProvider, p models.Proxy) {
		dp.Token = getCredential(p, "token", "DUCKDNS_TOKEN")
	},
	"hetzner": func(dp *models.CaddyDNSProvider, p models.Proxy) {
		dp.APIToken = getCredential(p, "api_token", "HETZNER_API_TOKEN")
	},
	"gandi": func(dp *models.CaddyDNSProvider, p models.Proxy) {
		dp.BearerToken = getCredential(p, "bearer_token", "GANDI_BEARER_TOKEN")
	},
	"dnsimple": func(dp *models.CaddyDNSProvider, p models.Proxy) {
		dp.APIAccessToken = getCredential(p, "api_access_token", "DNSIMPLE_API_ACCESS_TOKEN")
	},
}

// configureDNSProviderCredentials configures DNS provider credentials with environment fallback
func configureDNSProviderCredentials(dnsProvider *models.CaddyDNSProvider, proxy models.Proxy) {
	if configurator, ok := dnsConfigurators[proxy.DNSProvider]; ok {
		configurator(dnsProvider, proxy)
	}
}

// GetConfig retrieves the current Caddy configuration
func (c *Client) GetConfig() (*models.CaddyConfig, error) {
	resp, err := c.Client.Get(c.BaseURL + "/config/")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("caddy API returned status %d", resp.StatusCode)
	}

	var config models.CaddyConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// AddRedirect adds a new redirect configuration to Caddy
func (c *Client) AddRedirect(redirect models.Redirect) error {
	// Validate redirect
	if err := redirect.Validate(); err != nil {
		return fmt.Errorf("invalid redirect: %v", err)
	}

	// Build the redirect route
	newRoute, err := c.buildRedirectRoute(redirect)
	if err != nil {
		return fmt.Errorf("failed to build redirect route: %v", err)
	}

	// Get current config
	config, err := c.GetConfig()
	if err != nil || config.Apps.HTTP.Servers == nil {
		// If no config exists or servers is null, create a new one
		config = &models.CaddyConfig{
			Apps: models.CaddyApps{
				HTTP: models.CaddyHTTP{
					Servers: map[string]models.CaddyServer{},
				},
			},
		}
	}

	// Redirects always use the https_enabled server to handle both HTTP and HTTPS
	serverName := "https_enabled"
	listenPorts := []string{":80", ":443"}

	// Add route to server
	if server, exists := config.Apps.HTTP.Servers[serverName]; exists {
		server.Routes = append(server.Routes, *newRoute)

		// Add any new ports to the listen array
		for _, port := range listenPorts {
			if !slices.Contains(server.Listen, port) {
				server.Listen = append(server.Listen, port)
			}
		}

		config.Apps.HTTP.Servers[serverName] = server
	} else {
		// Create new server
		newServer := models.CaddyServer{
			Listen: listenPorts,
			Routes: []models.CaddyRoute{*newRoute},
		}

		config.Apps.HTTP.Servers[serverName] = newServer
	}

	// Update Caddy configuration
	return c.updateConfig(config)
}

// buildRedirectRoute creates a Caddy route for a redirect
func (c *Client) buildRedirectRoute(redirect models.Redirect) (*models.CaddyRoute, error) {
	// Build the redirect handler using static_response
	destinationURL := redirect.DestinationURL

	// Add path preservation if enabled
	if redirect.PreservePath {
		destinationURL = redirect.DestinationURL + "{http.request.uri}"
	}

	// Use two handlers: first set headers, then respond
	headersHandler := models.CaddyHandler{
		Handler: "headers",
		Response: &models.CaddyHeadersResponse{
			Set: map[string][]string{
				"Location": {destinationURL},
			},
		},
	}

	responseHandler := models.CaddyHandler{
		Handler:    "static_response",
		StatusCode: redirect.RedirectCode,
	}

	// Build matchers for all source domains
	var matchers []models.CaddyMatch
	for _, domain := range redirect.SourceDomains {
		// Only add host matcher if domain doesn't contain port
		if !strings.Contains(domain, ":") {
			matchers = append(matchers, models.CaddyMatch{
				Host: []string{domain},
			})
		}
	}

	// If no host matchers were created (all domains have ports), create a generic matcher
	if len(matchers) == 0 {
		matchers = append(matchers, models.CaddyMatch{})
	}

	// Create the route with both handlers
	return &models.CaddyRoute{
		ID:     redirect.ID,
		Handle: []models.CaddyHandler{headersHandler, responseHandler},
		Match:  matchers,
	}, nil
}

// UpdateRedirect updates an existing redirect configuration in Caddy
func (c *Client) UpdateRedirect(redirect models.Redirect) error {
	// For now, delete and re-add (more sophisticated update logic can be added later)
	if err := c.DeleteRedirect(redirect.ID); err != nil {
		return err
	}
	return c.AddRedirect(redirect)
}

// DeleteRedirect removes a redirect configuration from Caddy
func (c *Client) DeleteRedirect(id string) error {
	// Get current config to find which server contains the route
	config, err := c.GetConfig()
	if err != nil || config.Apps.HTTP.Servers == nil {
		return fmt.Errorf("failed to get current config: %v", err)
	}

	// Find and remove the route from all servers
	for serverName, server := range config.Apps.HTTP.Servers {
		var filteredRoutes []models.CaddyRoute
		found := false

		for _, route := range server.Routes {
			if route.ID != id {
				filteredRoutes = append(filteredRoutes, route)
			} else {
				found = true
			}
		}

		if found {
			// Update the server's routes
			server.Routes = filteredRoutes
			config.Apps.HTTP.Servers[serverName] = server

			// If server has no routes left, remove the server entirely
			if len(filteredRoutes) == 0 {
				delete(config.Apps.HTTP.Servers, serverName)
			}

			// Update entire configuration
			return c.updateConfig(config)
		}
	}

	return fmt.Errorf("redirect with ID %s not found", id)
}

// ParseRedirectsFromConfig extracts redirect configurations from Caddy config
func (c *Client) ParseRedirectsFromConfig(config *models.CaddyConfig) []models.Redirect {
	var redirects []models.Redirect

	if config == nil || config.Apps.HTTP.Servers == nil {
		return redirects
	}

	for _, server := range config.Apps.HTTP.Servers {
		for _, route := range server.Routes {
			// Skip routes without IDs (not created by proxy manager)
			if route.ID == "" || !strings.HasPrefix(route.ID, "redirect_") {
				continue
			}

			// Find the static_response handler and headers handler
			var responseHandler *models.CaddyHandler
			var headersHandler *models.CaddyHandler

			for i := range route.Handle {
				if route.Handle[i].Handler == "static_response" && route.Handle[i].StatusCode >= 301 && route.Handle[i].StatusCode <= 302 {
					responseHandler = &route.Handle[i]
				}
				if route.Handle[i].Handler == "headers" {
					headersHandler = &route.Handle[i]
				}
			}

			// Skip if no response handler found
			if responseHandler == nil {
				continue
			}

			// Extract destination URL from Location header in headers handler
			destinationURL := ""
			if headersHandler != nil && headersHandler.Response != nil && headersHandler.Response.Set != nil {
				if locations, ok := headersHandler.Response.Set["Location"]; ok && len(locations) > 0 {
					destinationURL = locations[0]
				}
			}

			if destinationURL == "" {
				continue // Skip if no location header found
			}

			redirect := models.Redirect{
				ID:             route.ID,
				DestinationURL: destinationURL,
				RedirectCode:   responseHandler.StatusCode,
				Status:         "active",
				CreatedAt:      "2024-01-01T00:00:00Z", // Default timestamp
				UpdatedAt:      "2024-01-01T00:00:00Z", // Default timestamp
			}

			// Check if path is preserved (destination URL ends with {http.request.uri})
			if strings.HasSuffix(destinationURL, "{http.request.uri}") {
				redirect.PreservePath = true
				redirect.DestinationURL = strings.TrimSuffix(destinationURL, "{http.request.uri}")
			}

			// Extract source domains from matchers
			for _, match := range route.Match {
				if len(match.Host) > 0 {
					redirect.SourceDomains = append(redirect.SourceDomains, match.Host...)
				}
			}

			redirects = append(redirects, redirect)
		}
	}

	return redirects
}

// AddProxy adds a new proxy configuration to Caddy
func (c *Client) AddProxy(proxy models.Proxy) error {
	// Validate IP lists
	if err := validateIPList(proxy.AllowedIPs); err != nil {
		return fmt.Errorf("invalid allowed IPs: %v", err)
	}
	if err := validateIPList(proxy.BlockedIPs); err != nil {
		return fmt.Errorf("invalid blocked IPs: %v", err)
	}

	// Build the route from the proxy model
	newRoute, err := c.buildProxyRoute(proxy)
	if err != nil {
		return fmt.Errorf("failed to build proxy route: %v", err)
	}

	// Get current config
	config, err := c.GetConfig()
	if err != nil || config.Apps.HTTP.Servers == nil {
		// If no config exists or servers is null, create a new one
		config = &models.CaddyConfig{
			Apps: models.CaddyApps{
				HTTP: models.CaddyHTTP{
					Servers: map[string]models.CaddyServer{},
				},
			},
		}
	}

	// Determine server name and listen ports based on SSL mode
	var serverName string
	var listenPorts []string

	if proxy.SSLMode == SSLModeNone {
		serverName = "http_only"
		listenPorts = []string{":80"}
	} else {
		serverName = "https_enabled"
		listenPorts = []string{":80", ":443"}
	}
	// Add specific port if domain includes port number
	if _, port, err := net.SplitHostPort(proxy.Domain); err == nil {
		listenPorts = append(listenPorts, ":"+port)
	}

	// Add route to appropriate server
	if server, exists := config.Apps.HTTP.Servers[serverName]; exists {
		server.Routes = append(server.Routes, *newRoute)

		// Add any new ports to the listen array
		for _, port := range listenPorts {
			if !slices.Contains(server.Listen, port) {
				server.Listen = append(server.Listen, port)
			}
		}

		// Add DNS challenge TLS policy if needed
		if proxy.SSLMode == SSLModeAuto && proxy.ChallengeType == "dns" {
			tlsPolicy := c.createDNSChallengeTLSPolicy(proxy)
			if tlsPolicy != nil {
				server.TLSPolicies = append(server.TLSPolicies, *tlsPolicy)
			}
		}

		config.Apps.HTTP.Servers[serverName] = server
	} else {
		// Create new server
		newServer := models.CaddyServer{
			Listen: listenPorts,
			Routes: []models.CaddyRoute{*newRoute},
		}

		// Disable automatic HTTPS for HTTP-only servers
		if proxy.SSLMode == SSLModeNone {
			newServer.AutomaticHTTPS = &models.CaddyAutomaticHTTPS{
				Disable: true,
			}
		}

		// Add DNS challenge TLS policy if needed
		if proxy.SSLMode == SSLModeAuto && proxy.ChallengeType == "dns" {
			tlsPolicy := c.createDNSChallengeTLSPolicy(proxy)
			if tlsPolicy != nil {
				newServer.TLSPolicies = []models.CaddyTLSPolicy{*tlsPolicy}
			}
		}

		config.Apps.HTTP.Servers[serverName] = newServer
	}

	// Configure global TLS settings for DNS challenges
	if proxy.SSLMode == "auto" && proxy.ChallengeType == "dns" {
		if config.Apps.TLS == nil {
			config.Apps.TLS = &models.CaddyTLS{}
		}
		c.configureDNSChallenge(config, proxy)
	}

	// Save metadata
	c.metadata.Set(proxy)
	if err := c.saveMetadataToFile(); err != nil {
		log.Printf("Warning: Failed to save metadata: %v", err)
	}

	// Update Caddy configuration
	return c.updateConfig(config)
}

// buildProxyRoute creates a Caddy route from a proxy model
func (c *Client) buildProxyRoute(proxy models.Proxy) (*models.CaddyRoute, error) {
	var handlers []models.CaddyHandler

	// Add basic auth handler if enabled
	if proxy.BasicAuth != nil && proxy.BasicAuth.Enabled && proxy.BasicAuth.Username != "" && proxy.BasicAuth.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(proxy.BasicAuth.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %v", err)
		}
		basicAuthHandler := models.CaddyHandler{
			Handler: "authentication",
			Providers: map[string]models.CaddyAuthProvider{
				"http_basic": {
					Accounts: []models.CaddyAccount{
						{
							Username: proxy.BasicAuth.Username,
							Password: string(hashedPassword),
						},
					},
				},
			},
		}
		handlers = append(handlers, basicAuthHandler)
	}

	// Build and add the reverse proxy handler
	reverseProxyHandler, err := c.buildReverseProxyHandler(proxy)
	if err != nil {
		return nil, err
	}
	handlers = append(handlers, *reverseProxyHandler)

	// Build matchers for the route
	matchers := c.buildRouteMatchers(proxy)

	// Create the final route
	newRoute := models.CaddyRoute{
		ID:     proxy.ID,
		Handle: handlers,
		Match:  matchers,
	}

	return &newRoute, nil
}

// buildReverseProxyHandler creates a Caddy reverse_proxy handler from a proxy model
func (c *Client) buildReverseProxyHandler(proxy models.Proxy) (*models.CaddyHandler, error) {
	dialAddr, useHTTPS, targetHost, err := parseTargetURL(proxy.TargetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL: %v", err)
	}

	// Create the handler with upstream and Host header override
	handler := models.CaddyHandler{
		Handler: "reverse_proxy",
		Upstreams: []models.CaddyUpstream{
			{Dial: dialAddr},
		},
		Headers: &models.CaddyHeaders{
			Request: &models.CaddyHeadersRequest{
				Set: map[string][]string{
					"Host": {targetHost},
				},
			},
		},
	}

	// Add custom headers
	if len(proxy.CustomHeaders) > 0 {
		for key, value := range proxy.CustomHeaders {
			handler.Headers.Request.Set[key] = []string{value}
		}
	}

	// Configure HTTPS transport if the target is HTTPS
	if useHTTPS {
		handler.Transport = &models.CaddyTransport{
			Protocol: "http",
			TLS:      &struct{}{},
		}
	}

	return &handler, nil
}

// buildRouteMatchers creates Caddy matchers from a proxy model, including IP filtering
func (c *Client) buildRouteMatchers(proxy models.Proxy) []models.CaddyMatch {
	baseMatch := models.CaddyMatch{}
	// Host matcher only works for domains without ports
	if !strings.Contains(proxy.Domain, ":") {
		baseMatch.Host = []string{proxy.Domain}
	}

	var routeMatches []models.CaddyMatch

	// Handle AllowedIPs (whitelist)
	if len(proxy.AllowedIPs) > 0 {
		var allowedIPs []string
		for _, ip := range proxy.AllowedIPs {
			if ip = strings.TrimSpace(ip); ip != "" {
				allowedIPs = append(allowedIPs, ip)
			}
		}
		if len(allowedIPs) > 0 {
			allowMatch := baseMatch
			allowMatch.RemoteIP = &models.CaddyRemoteIPMatch{Ranges: allowedIPs}
			routeMatches = append(routeMatches, allowMatch)
		}
	} else if len(proxy.BlockedIPs) > 0 { // Handle BlockedIPs (blacklist) only if no whitelist
		var blockedIPs []string
		for _, ip := range proxy.BlockedIPs {
			if ip = strings.TrimSpace(ip); ip != "" {
				blockedIPs = append(blockedIPs, ip)
			}
		}
		if len(blockedIPs) > 0 {
			blockMatch := baseMatch
			blockMatch.Not = &models.CaddyMatch{
				RemoteIP: &models.CaddyRemoteIPMatch{Ranges: blockedIPs},
			}
			routeMatches = append(routeMatches, blockMatch)
		}
	}

	// If no IP filtering was applied but we have a host, use the base match
	if len(routeMatches) == 0 && len(baseMatch.Host) > 0 {
		routeMatches = append(routeMatches, baseMatch)
	}

	return routeMatches
}

// UpdateProxy updates an existing proxy configuration in Caddy
func (c *Client) UpdateProxy(proxy models.Proxy) error {
	// For now, delete and re-add (more sophisticated update logic can be added later)
	if err := c.DeleteProxy(proxy.ID); err != nil {
		return err
	}
	return c.AddProxy(proxy)
}

// DeleteProxy removes a proxy configuration from Caddy
func (c *Client) DeleteProxy(id string) error {
	// Remove metadata
	c.metadata.Delete(id)
	if err := c.saveMetadataToFile(); err != nil {
		log.Printf("Warning: Failed to save metadata: %v", err)
	}
	// Get current config to find which server contains the route
	config, err := c.GetConfig()
	if err != nil || config.Apps.HTTP.Servers == nil {
		return fmt.Errorf("failed to get current config: %v", err)
	}

	// Find and remove the route from all servers
	for serverName, server := range config.Apps.HTTP.Servers {
		var filteredRoutes []models.CaddyRoute
		found := false

		for _, route := range server.Routes {
			if route.ID != id {
				filteredRoutes = append(filteredRoutes, route)
			} else {
				found = true
			}
		}

		if found {
			// Update the server's routes
			server.Routes = filteredRoutes
			config.Apps.HTTP.Servers[serverName] = server

			// If server has no routes left, remove the server entirely
			if len(filteredRoutes) == 0 {
				delete(config.Apps.HTTP.Servers, serverName)
			}

			// Update entire configuration
			return c.updateConfig(config)
		}
	}

	return fmt.Errorf("route with ID %s not found", id)
}

// GetStatus retrieves Caddy reverse proxy status
func (c *Client) GetStatus() (any, error) {
	resp, err := c.Client.Get(c.BaseURL + "/reverse_proxy/upstreams")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("caddy API returned status %d", resp.StatusCode)
	}

	var status any
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}

	return status, nil
}

// Reload reloads the Caddy configuration
func (c *Client) Reload() error {
	resp, err := c.Client.Post(c.BaseURL+"/load", "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to reload: %s", string(body))
	}

	return nil
}

// updateConfig updates the entire Caddy configuration and saves it to file
func (c *Client) updateConfig(config *models.CaddyConfig) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/load", bytes.NewBuffer(configJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update config: %s", string(body))
	}

	// Save config to file after successful update
	if err := c.saveConfigToFile(config); err != nil {
		// Log error but don't fail the operation since Caddy was updated successfully
		log.Printf("Warning: Failed to save config to file: %v", err)
	}

	return nil
}

// saveConfigToFile saves the configuration to a JSON file
func (c *Client) saveConfigToFile(config *models.CaddyConfig) error {
	if c.ConfigFile == "" {
		return nil // No config file specified
	}

	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(c.ConfigFile, configJSON, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// LoadConfigFromFile loads the configuration from a JSON file
func (c *Client) LoadConfigFromFile() (*models.CaddyConfig, error) {
	if c.ConfigFile == "" {
		return nil, fmt.Errorf("no config file specified")
	}

	data, err := os.ReadFile(c.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config models.CaddyConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return &config, nil
}

// RestoreConfigFromFile loads config from file and applies it to Caddy
func (c *Client) RestoreConfigFromFile() error {
	if c.ConfigFile == "" {
		return nil // No config file specified, nothing to restore
	}

	// Check if config file exists
	if _, err := os.Stat(c.ConfigFile); os.IsNotExist(err) {
		return nil // Config file doesn't exist, nothing to restore
	}

	config, err := c.LoadConfigFromFile()
	if err != nil {
		return fmt.Errorf("failed to load config from file: %v", err)
	}

	// Apply the config to Caddy (without saving to file again to avoid recursion)
	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+"/load", bytes.NewBuffer(configJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to restore config: %s", string(body))
	}

	return nil
}

// ParseProxiesFromConfig extracts proxy configurations from Caddy config
func (c *Client) ParseProxiesFromConfig(config *models.CaddyConfig) []models.Proxy {
	var proxies []models.Proxy

	if config == nil || config.Apps.HTTP.Servers == nil {
		return proxies
	}

	for serverName, server := range config.Apps.HTTP.Servers {
		for _, route := range server.Routes {
			// Skip routes without IDs (not created by proxy manager)
			if route.ID == "" {
				continue
			}

			// Find the reverse_proxy handler (might not be the first one due to authentication)
			var reverseProxyHandler *models.CaddyHandler
			for i := range route.Handle {
				if route.Handle[i].Handler == "reverse_proxy" {
					reverseProxyHandler = &route.Handle[i]
					break
				}
			}

			// Skip if no reverse_proxy handler found
			if reverseProxyHandler == nil {
				continue
			}

			proxy := models.Proxy{
				ID:        route.ID,
				Status:    "active",
				CreatedAt: "2024-01-01T00:00:00Z", // Default timestamp for existing proxies
				UpdatedAt: "2024-01-01T00:00:00Z", // Default timestamp for existing proxies
			}

			// Apply stored metadata
			c.metadata.ApplyToProxy(&proxy)

			// Extract domain from match or proxy ID
			if len(route.Match) > 0 && len(route.Match[0].Host) > 0 {
				proxy.Domain = route.Match[0].Host[0]
			} else {
				// For port-based proxies, extract domain from ID
				// ID format: "proxy_localhost:9801_1755490936"
				if strings.HasPrefix(route.ID, "proxy_") {
					parts := strings.Split(route.ID, "_")
					if len(parts) >= 3 {
						// Reconstruct domain from parts (handling colons in domain)
						domainParts := parts[1 : len(parts)-1]
						proxy.Domain = strings.Join(domainParts, "_")
						// Replace underscores back to colons for port-based domains
						proxy.Domain = strings.ReplaceAll(proxy.Domain, "_", ":")
					}
				}
			}

			// Extract target URL from upstreams
			if len(reverseProxyHandler.Upstreams) > 0 {
				dial := reverseProxyHandler.Upstreams[0].Dial
				// Determine scheme based on port or default to http
				scheme := "http"
				if strings.HasSuffix(dial, ":443") {
					scheme = "https"
				}
				proxy.TargetURL = fmt.Sprintf("%s://%s", scheme, dial)
			}

			// Determine SSL mode based on server configuration
			hasHTTPS := slices.Contains(server.Listen, ":443")

			if serverName == "http_only" || !hasHTTPS {
				proxy.SSLMode = "none"
			} else {
				proxy.SSLMode = "auto"
			}

			proxies = append(proxies, proxy)
		}
	}

	return proxies
}

// parseTargetURL parses the target URL and returns the dial address with proper port, whether to use HTTPS, and target hostname
func parseTargetURL(targetURL string) (string, bool, string, error) {
	originalURL := targetURL

	// If the URL doesn't contain ://, add http:// prefix for parsing
	if !strings.Contains(targetURL, "://") {
		targetURL = "http://" + targetURL
	}

	// Parse the URL
	u, err := url.Parse(targetURL)
	if err != nil {
		return "", false, "", fmt.Errorf("failed to parse URL: %v", err)
	}

	// Determine if original URL specified HTTPS
	useHTTPS := strings.HasPrefix(originalURL, "https://")

	// Get the host and port
	host := u.Hostname()
	port := u.Port()

	// If no port is specified, use default based on scheme
	if port == "" {
		switch u.Scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		default:
			// If no scheme, assume http and use port 80
			port = "80"
		}
	}

	return fmt.Sprintf("%s:%s", host, port), useHTTPS, host, nil
}

// createDNSChallengeTLSPolicy creates a TLS policy for DNS challenges
func (c *Client) createDNSChallengeTLSPolicy(proxy models.Proxy) *models.CaddyTLSPolicy {
	if proxy.ChallengeType != "dns" || proxy.DNSProvider == "" {
		return nil
	}

	// Create DNS provider configuration
	dnsProvider := models.CaddyDNSProvider{
		Name: fmt.Sprintf("dns.providers.%s", proxy.DNSProvider),
	}

	// Set provider-specific credentials with environment variable fallback
	configureDNSProviderCredentials(&dnsProvider, proxy)

	// Create the TLS policy with DNS challenge
	return &models.CaddyTLSPolicy{
		Match: &models.CaddyTLSMatch{
			SNI: []string{proxy.Domain},
		},
		Issuers: []models.CaddyIssuer{
			{
				Module: "acme",
				Challenges: models.CaddyChallenges{
					DNS: &models.CaddyDNSChallenge{
						Provider: dnsProvider,
					},
				},
			},
		},
	}
}

// configureDNSChallenge configures global DNS challenge settings
func (c *Client) configureDNSChallenge(config *models.CaddyConfig, proxy models.Proxy) {
	if proxy.ChallengeType != "dns" || proxy.DNSProvider == "" {
		return
	}

	// Initialize certificate authorities if not present
	if config.Apps.TLS.CertificateAuthorities == nil {
		config.Apps.TLS.CertificateAuthorities = make(map[string]models.CaddyCA)
	}

	// Create DNS provider configuration
	dnsProvider := models.CaddyDNSProvider{
		Name: fmt.Sprintf("dns.providers.%s", proxy.DNSProvider),
	}

	// Set provider-specific credentials with environment variable fallback
	configureDNSProviderCredentials(&dnsProvider, proxy)

	// Configure the ACME CA with DNS challenge
	acmeCA := models.CaddyCA{
		Module: "acme",
		Challenges: models.CaddyChallenges{
			DNS: &models.CaddyDNSChallenge{
				Provider: dnsProvider,
			},
		},
	}

	// Set the default ACME CA to use DNS challenges
	config.Apps.TLS.CertificateAuthorities["acme"] = acmeCA
}

// saveMetadataToFile saves the metadata to a JSON file
func (c *Client) saveMetadataToFile() error {
	if c.MetadataFile == "" {
		return nil // No metadata file specified
	}

	metadataJSON, err := json.MarshalIndent(c.metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}

	if err := os.WriteFile(c.MetadataFile, metadataJSON, 0644); err != nil {
		return fmt.Errorf("failed to write metadata file: %v", err)
	}

	return nil
}

// loadMetadataFromFile loads the metadata from a JSON file
func (c *Client) loadMetadataFromFile() error {
	if c.MetadataFile == "" {
		return nil // No metadata file specified
	}

	// Check if metadata file exists
	if _, err := os.Stat(c.MetadataFile); os.IsNotExist(err) {
		return nil // Metadata file doesn't exist, use empty store
	}

	data, err := os.ReadFile(c.MetadataFile)
	if err != nil {
		return fmt.Errorf("failed to read metadata file: %v", err)
	}

	var metadata models.MetadataStore
	if err := json.Unmarshal(data, &metadata); err != nil {
		return fmt.Errorf("failed to unmarshal metadata: %v", err)
	}

	c.metadata = &metadata
	return nil
}
