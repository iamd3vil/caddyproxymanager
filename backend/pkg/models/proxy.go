package models

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Proxy represents a reverse proxy configuration
type Proxy struct {
	ID             string            `json:"id"`
	Domain         string            `json:"domain"`
	TargetURL      string            `json:"target_url"`
	SSLMode        string            `json:"ssl_mode"`        // "auto", "custom", "none"
	ChallengeType  string            `json:"challenge_type"`  // "http", "dns"
	DNSProvider    string            `json:"dns_provider"`    // "cloudflare", "digitalocean", "duckdns"
	DNSCredentials map[string]string `json:"dns_credentials"` // provider-specific credentials
	CustomHeaders  map[string]string `json:"custom_headers"`  // custom request headers
	Status         string            `json:"status"`          // "active", "inactive", "error"
	CreatedAt      string            `json:"created_at"`
	UpdatedAt      string            `json:"updated_at"`
}

// NewProxy creates a new Proxy with generated ID and timestamps
func NewProxy(domain, targetURL, sslMode string) *Proxy {
	now := time.Now().Format(time.RFC3339)
	return &Proxy{
		ID:             GenerateProxyID(domain),
		Domain:         domain,
		TargetURL:      targetURL,
		SSLMode:        sslMode,
		ChallengeType:  "http", // default to HTTP challenge
		DNSProvider:    "",
		DNSCredentials: make(map[string]string),
		CustomHeaders:  make(map[string]string),
		Status:         "active",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// UpdateTimestamp updates the UpdatedAt field to current time
func (p *Proxy) UpdateTimestamp() {
	p.UpdatedAt = time.Now().Format(time.RFC3339)
}

// GenerateProxyID generates a unique ID for a proxy based on domain and timestamp
func GenerateProxyID(domain string) string {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	return fmt.Sprintf("proxy_%s_%s", strings.ReplaceAll(domain, ".", "_"), timestamp)
}
