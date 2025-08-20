# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Caddy Proxy Manager - a web-based management interface for Caddy similar to nginx proxy manager but for Caddy. The project provides:

- **Web UI**: Clean, modern interface for managing Caddy proxy configurations
- **Automatic HTTPS**: Let's Encrypt integration with HTTP-01 and DNS-01 challenges  
- **DNS Challenge Support**: Works behind firewalls with DNS providers (Cloudflare, DigitalOcean, DuckDNS)
- **Real-time Management**: Direct integration with Caddy Admin API
- **Containerized**: Complete Docker setup with all dependencies included

Use TODO.md file to track the todos and tasks and keep on working on them and update them. When asked to commit, don't include yourself in the commit message.

## Technology Stack

- **Backend**: Go (as indicated by go.mod requiring Go 1.25.0)
- **Frontend**: Vue 3, Vite, and DaisyUI with TypeScript and Tailwind CSS
- **Proxy Server**: Caddy (managed via Caddy Admin API)
- **Containerization**: Docker with multi-stage builds and supervisord process management

## Project Architecture

The project is designed to provide a web UI for managing Caddy proxy configurations through the Caddy Admin API.

### Current Structure

- **Backend** (`backend/`): Go backend with proper project structure
  - `cmd/server/main.go`: Main entry point for the server
  - `internal/handlers/`: HTTP handlers
  - `pkg/caddy/`: Caddy client for interacting with Caddy Admin API
  - `pkg/models/`: Data models for Caddy and proxy configurations
- **Frontend** (`frontend/`): Vue 3 + TypeScript + Tailwind CSS frontend
  - Includes Vue Router, Vite build system, and ESLint
- **Build System**: Uses Justfile for task automation

### Development Commands

- `just backend-run`: Run the backend server
- `just frontend-dev`: Run the frontend development server  
- `just dev`: Run backend (alias for backend-run)
- `just setup`: Install dependencies for both frontend and backend
- `just build`: Build both frontend and backend

### DNS Providers Support

The project supports DNS-01 challenges for HTTPS certificates with these providers:
- **Cloudflare**: Requires API Token with Zone:DNS:Edit permissions
- **DigitalOcean**: Requires Personal Access Token with write scope  
- **DuckDNS**: Requires DuckDNS account token

### Ports Used

- `80`: HTTP proxy traffic and ACME challenges
- `443`: HTTPS proxy traffic
- `8080`: Proxy Manager web interface
- `2019`: Caddy Admin API (optional, for debugging)
