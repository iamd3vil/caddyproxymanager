package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

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

	mux := http.NewServeMux()

	fmt.Printf("Server starting on port %s\n", port)
	fmt.Printf("Caddy Admin API: %s\n", caddyAdminURL)
	fmt.Printf("Config file: %s\n", configFile)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
