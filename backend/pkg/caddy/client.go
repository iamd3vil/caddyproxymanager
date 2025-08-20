package caddy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/sarat/caddyproxymanager/pkg/models"
)

// Client handles communication with Caddy Admin API
type Client struct {
	BaseURL    string
	Client     *http.Client
	ConfigFile string
}

// New creates a new Caddy API client
func New(baseURL, configFile string) *Client {
	return &Client{
		BaseURL:    baseURL,
		ConfigFile: configFile,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
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

// AddProxy adds a new proxy configuration to Caddy
func (c *Client) AddProxy(proxy models.Proxy) error {
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

	if proxy.SSLMode == "none" {
		serverName = "http_only"
		listenPorts = []string{":80"}
		// Add specific port if domain includes port number
		if strings.Contains(proxy.Domain, ":") {
			parts := strings.Split(proxy.Domain, ":")
			if len(parts) == 2 {
				listenPorts = append(listenPorts, ":"+parts[1])
			}
		}
	} else {
		serverName = "https_enabled"
		listenPorts = []string{":80", ":443"}
		// Add specific port if domain includes port number
		if strings.Contains(proxy.Domain, ":") {
			parts := strings.Split(proxy.Domain, ":")
			if len(parts) == 2 {
				listenPorts = append(listenPorts, ":"+parts[1])
			}
		}
	}

	// Parse target URL to get proper dial address with port and scheme
	dialAddr, useHTTPS, targetHost, err := parseTargetURL(proxy.TargetURL)
	if err != nil {
		return fmt.Errorf("invalid target URL: %v", err)
	}

	// Create handler with upstreams
	handler := models.CaddyHandler{
		Handler: "reverse_proxy",
		Upstreams: []models.CaddyUpstream{
			{Dial: dialAddr},
		},
		// Set the Host header to the target hostname to avoid 421 Misdirected Request
		Headers: &models.CaddyHeaders{
			Request: &models.CaddyHeadersRequest{
				Set: map[string][]string{
					"Host": {targetHost},
				},
			},
		},
	}

	// Add HTTPS transport if target uses HTTPS
	if useHTTPS {
		handler.Transport = &models.CaddyTransport{
			Protocol: "http",
			TLS:      &struct{}{},
		}
	}

	// Add new route for this proxy
	newRoute := models.CaddyRoute{
		ID:     proxy.ID,
		Handle: []models.CaddyHandler{handler},
	}

	// Only add host matcher for domain names without ports
	if !strings.Contains(proxy.Domain, ":") {
		newRoute.Match = []models.CaddyMatch{
			{Host: []string{proxy.Domain}},
		}
	}

	// Add route to appropriate server
	if server, exists := config.Apps.HTTP.Servers[serverName]; exists {
		server.Routes = append(server.Routes, newRoute)

		// Add any new ports to the listen array
		for _, port := range listenPorts {
			if !slices.Contains(server.Listen, port) {
				server.Listen = append(server.Listen, port)
			}
		}

		// Add DNS challenge TLS policy if needed
		if proxy.SSLMode == "auto" && proxy.ChallengeType == "dns" {
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
			Routes: []models.CaddyRoute{newRoute},
		}

		// Disable automatic HTTPS for HTTP-only servers
		if proxy.SSLMode == "none" {
			newServer.AutomaticHTTPS = &models.CaddyAutomaticHTTPS{
				Disable: true,
			}
		}

		// Add DNS challenge TLS policy if needed
		if proxy.SSLMode == "auto" && proxy.ChallengeType == "dns" {
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

	// Update Caddy configuration
	return c.updateConfig(config)
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
		fmt.Printf("Warning: Failed to save config to file: %v\n", err)
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

	if err := os.WriteFile(c.ConfigFile, configJSON, 0644); err != nil {
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
			if len(route.Handle) > 0 && route.Handle[0].Handler == "reverse_proxy" {
				// Skip routes without IDs (not created by proxy manager)
				if route.ID == "" {
					continue
				}

				proxy := models.Proxy{
					ID:        route.ID,
					Status:    "active",
					CreatedAt: "2024-01-01T00:00:00Z", // Default timestamp for existing proxies
					UpdatedAt: "2024-01-01T00:00:00Z", // Default timestamp for existing proxies
				}

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
				if len(route.Handle[0].Upstreams) > 0 {
					dial := route.Handle[0].Upstreams[0].Dial
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
	switch proxy.DNSProvider {
	case "cloudflare":
		// Use provided credentials or fall back to environment variables
		if apiToken, ok := proxy.DNSCredentials["api_token"]; ok && apiToken != "" {
			dnsProvider.APIToken = apiToken
		} else if envToken := os.Getenv("CLOUDFLARE_API_TOKEN"); envToken != "" {
			dnsProvider.APIToken = envToken
		}
		if email, ok := proxy.DNSCredentials["email"]; ok && email != "" {
			dnsProvider.Email = email
		} else if envEmail := os.Getenv("CLOUDFLARE_EMAIL"); envEmail != "" {
			dnsProvider.Email = envEmail
		}
	case "digitalocean":
		if authToken, ok := proxy.DNSCredentials["auth_token"]; ok && authToken != "" {
			dnsProvider.AuthToken = authToken
		} else if envToken := os.Getenv("DO_AUTH_TOKEN"); envToken != "" {
			dnsProvider.AuthToken = envToken
		}
	case "duckdns":
		if token, ok := proxy.DNSCredentials["token"]; ok && token != "" {
			dnsProvider.Token = token
		} else if envToken := os.Getenv("DUCKDNS_TOKEN"); envToken != "" {
			dnsProvider.Token = envToken
		}
	}

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
	switch proxy.DNSProvider {
	case "cloudflare":
		if apiToken, ok := proxy.DNSCredentials["api_token"]; ok && apiToken != "" {
			dnsProvider.APIToken = apiToken
		} else if envToken := os.Getenv("CLOUDFLARE_API_TOKEN"); envToken != "" {
			dnsProvider.APIToken = envToken
		}
		if email, ok := proxy.DNSCredentials["email"]; ok && email != "" {
			dnsProvider.Email = email
		} else if envEmail := os.Getenv("CLOUDFLARE_EMAIL"); envEmail != "" {
			dnsProvider.Email = envEmail
		}
	case "digitalocean":
		if authToken, ok := proxy.DNSCredentials["auth_token"]; ok && authToken != "" {
			dnsProvider.AuthToken = authToken
		} else if envToken := os.Getenv("DO_AUTH_TOKEN"); envToken != "" {
			dnsProvider.AuthToken = envToken
		}
	case "duckdns":
		if token, ok := proxy.DNSCredentials["token"]; ok && token != "" {
			dnsProvider.Token = token
		} else if envToken := os.Getenv("DUCKDNS_TOKEN"); envToken != "" {
			dnsProvider.Token = envToken
		}
	}

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
