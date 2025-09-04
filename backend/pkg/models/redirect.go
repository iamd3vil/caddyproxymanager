package models

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Redirect represents an HTTP redirect configuration
type Redirect struct {
	ID             string   `json:"id"`
	SourceDomains  []string `json:"source_domains"`
	DestinationURL string   `json:"destination_url"`
	RedirectCode   int      `json:"redirect_code"` // 301 or 302
	PreservePath   bool     `json:"preserve_path"`
	Status         string   `json:"status"` // "active", "inactive", "error"
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

// NewRedirect creates a new Redirect with generated ID and timestamps
func NewRedirect(sourceDomains []string, destinationURL string, redirectCode int, preservePath bool) *Redirect {
	now := time.Now().Format(time.RFC3339)

	// Use first domain for ID generation or fallback
	firstDomain := "redirect"
	if len(sourceDomains) > 0 {
		firstDomain = sourceDomains[0]
	}

	return &Redirect{
		ID:             GenerateRedirectID(firstDomain),
		SourceDomains:  sourceDomains,
		DestinationURL: destinationURL,
		RedirectCode:   redirectCode,
		PreservePath:   preservePath,
		Status:         "active",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// UpdateTimestamp updates the UpdatedAt field to current time
func (r *Redirect) UpdateTimestamp() {
	r.UpdatedAt = time.Now().Format(time.RFC3339)
}

// GenerateRedirectID generates a unique ID for a redirect based on domain and timestamp
func GenerateRedirectID(domain string) string {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	return fmt.Sprintf("redirect_%s_%s", strings.ReplaceAll(domain, ".", "_"), timestamp)
}

// Validate validates the redirect configuration
func (r *Redirect) Validate() error {
	if len(r.SourceDomains) == 0 {
		return fmt.Errorf("at least one source domain is required")
	}

	if r.DestinationURL == "" {
		return fmt.Errorf("destination URL is required")
	}

	if r.RedirectCode != 301 && r.RedirectCode != 302 {
		return fmt.Errorf("redirect code must be 301 or 302")
	}

	// Basic URL validation - check if it starts with http:// or https://
	if !strings.HasPrefix(r.DestinationURL, "http://") && !strings.HasPrefix(r.DestinationURL, "https://") {
		return fmt.Errorf("destination URL must start with http:// or https://")
	}

	return nil
}
