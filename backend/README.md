# Caddy Proxy Manager Backend

A Go-based backend service for managing Caddy reverse proxy configurations through the Caddy Admin API.

## Project Structure

```
backend/
├── cmd/
│   └── server/          # Main application entry point
├── pkg/
│   ├── models/          # Data models and structures
│   └── caddy/           # Caddy Admin API client
├── internal/
│   └── handlers/        # HTTP request handlers
├── Makefile            # Build and development commands
└── go.mod              # Go module definition
```

## Quick Start

### Prerequisites

- Go 1.25.0 or later
- Caddy server running with Admin API enabled (default: localhost:2019)

### Building and Running

```bash
# Build the application
make build

# Run in development mode
make run-dev

# Or run directly with go
PORT=8081 go run ./cmd/server
```

### Environment Variables

- `PORT`: Server port (default: 8080)
- `CADDY_ADMIN_URL`: Caddy Admin API URL (default: http://localhost:2019)

## API Endpoints

- `GET /api/health` - Health check
- `GET /api/proxies` - List all proxy configurations
- `POST /api/proxies` - Create a new proxy
- `PUT /api/proxies/{id}` - Update a proxy
- `DELETE /api/proxies/{id}` - Delete a proxy
- `GET /api/status` - Get Caddy status
- `POST /api/reload` - Reload Caddy configuration
