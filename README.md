# Caddy Proxy Manager

A modern web-based management interface for Caddy reverse proxy configurations, similar to nginx proxy manager but built specifically for Caddy.

## 📖 Table of Contents

- [✨ Features](#-features)
- [🚀 Quick Start](#-quick-start)
  - [Using Docker (Recommended)](#using-docker-recommended)
  - [Using Docker Compose (Alternative)](#using-docker-compose-alternative)
- [📦 Installation](#-installation)
  - [Pre-built Docker Image (GitHub Container Registry)](#pre-built-docker-image-github-container-registry)
  - [Docker Compose with Pre-built Image](#docker-compose-with-pre-built-image)
  - [Available Tags](#available-tags)
  - [Important Notes](#important-notes)
  - [Manual Installation](#manual-installation)
- [📖 Usage](#-usage)
  - [Creating a Proxy](#creating-a-proxy)
  - [Advanced Proxy Features](#advanced-proxy-features)
  - [SSL Certificate Options](#ssl-certificate-options)
  - [Supported DNS Providers](#supported-dns-providers)
- [🛠 Development](#-development)
  - [Project Structure](#project-structure)
  - [Development Commands](#development-commands)
  - [Building Custom Caddy](#building-custom-caddy)
- [🔧 Configuration](#-configuration)
  - [Environment Variables](#environment-variables)
  - [Ports](#ports)
- [🐳 Docker Configuration](#-docker-configuration)
  - [Volumes](#volumes)
- [🔒 Security](#-security)
- [🤝 Contributing](#-contributing)
- [📄 License](#-license)
- [🙏 Acknowledgments](#-acknowledgments)

## ✨ Features

- **🌐 Web UI**: Clean, modern interface built with Vue 3, Vite, and DaisyUI
- **🔒 Automatic HTTPS**: Let's Encrypt integration with HTTP-01 and DNS-01 challenges
- **🌍 DNS Challenge Support**: Works behind firewalls with DNS providers (Cloudflare, DigitalOcean, DuckDNS, Hetzner, Gandi, DNSimple)
- **⚡ Real-time Management**: Direct integration with Caddy Admin API
- **🐳 Containerized**: Complete Docker setup with all dependencies included
- **🔧 Easy Configuration**: No complex config files - manage everything through the UI
- **📊 Status Monitoring**: Real-time proxy status and health monitoring
- **🏥 Health Checks**: Monitor upstream server health with configurable intervals and failure thresholds
- **📝 Custom Headers**: Add custom request/response headers for enhanced functionality
- **🛡️ IP Access Control**: Whitelist or blacklist IP addresses for advanced security
- **📋 Audit Logging**: Comprehensive logging of all configuration changes
- **🔧 Custom Caddy JSON Snippets**: Advanced feature for inserting raw Caddy JSON configuration

## 🚀 Quick Start

### Using Docker (Recommended)

**Pull and run the pre-built image:**
```bash
docker run -d \
  --name caddy-proxy-manager \
  -p 80:80 \
  -p 443:443 \
  -p 8080:8080 \
  -v $(pwd)/config:/config \
  -v $(pwd)/data:/data \
  -v $(pwd)/logs:/var/log \
  ghcr.io/iamd3vil/caddyproxymanager:latest
```

**Access the web interface:**
- Proxy Manager UI: http://localhost:8080

### Using Docker Compose (Alternative)

1. **Clone the repository**
   ```bash
   git clone https://github.com/iamd3vil/caddyproxymanager.git
   cd caddyproxymanager
   ```

2. **Start the services**
   ```bash
   docker-compose up -d
   ```

3. **Access the web interface**
   - Proxy Manager UI: http://localhost:8080

## 📦 Installation

### Pre-built Docker Image (GitHub Container Registry)

The easiest way to get started is using the pre-built Docker image:

```bash
# Pull the latest image
docker pull ghcr.io/iamd3vil/caddyproxymanager:latest

# Run with basic configuration
docker run -d \
  --name caddy-proxy-manager \
  -p 80:80 \
  -p 443:443 \
  -p 8080:8080 \
  -v caddy_config:/config \
  -v caddy_data:/data \
  -v caddy_logs:/var/log \
  ghcr.io/iamd3vil/caddyproxymanager:latest
```

### Docker Compose with Pre-built Image

Create a `docker-compose.yml` file:

```yaml
services:
  caddy-proxy-manager:
    image: ghcr.io/iamd3vil/caddyproxymanager:latest
    container_name: caddy-proxy-manager
    ports:
      - "80:80"
      - "443:443" 
      - "8080:8080"
    volumes:
      - ./config:/config
      - ./data:/data
      - ./logs:/var/log
    environment:
      # Optional: Set DNS provider credentials
      - CLOUDFLARE_API_TOKEN=your-token-here
      - DO_AUTH_TOKEN=your-token-here  
      - DUCKDNS_TOKEN=your-token-here
    restart: unless-stopped
```

Then run:
```bash
docker-compose up -d
```

### Available Tags

- `ghcr.io/iamd3vil/caddyproxymanager:latest` - Latest stable release
- `ghcr.io/iamd3vil/caddyproxymanager:v1.0.0` - Specific version

### Important Notes

**Persistence**: The `/config` and `/data` directories contain your proxy configurations and SSL certificates. Make sure to mount these as volumes to preserve your settings across container updates.

**Security**: For production use, consider:
- Running on a private network or behind a firewall
- Using environment variables or Docker secrets for DNS provider credentials
- Regularly updating to the latest image version

**Network**: Ensure ports 80 and 443 are accessible from the internet if you need ACME HTTP-01 challenges for Let's Encrypt certificates.

### Manual Installation

#### Prerequisites
- Go 1.25+
- Node.js 20+
- Caddy 2.7+ with DNS plugins

#### Backend Setup
```bash
cd backend
go mod download
go build -o server ./cmd/server
./server
```

#### Frontend Setup
```bash
cd frontend
npm install
npm run build
# Serve the built files with your preferred web server
```

## 📖 Usage

### Creating a Proxy

1. **Access the UI** at http://localhost:8080
2. **Click "Add Proxy"**
3. **Configure your proxy:**
   - **Domain**: Your domain/subdomain (e.g., `api.example.com`)
   - **Target URL**: Where to proxy requests (e.g., `http://localhost:3000`)
   - **SSL Mode**: Choose automatic HTTPS or HTTP-only

### Advanced Proxy Features

#### Health Checks
Monitor the health of your upstream servers:
- **Enable Health Checks**: Toggle monitoring for each proxy
- **Check Interval**: Configure how often to check (default: 30 seconds)
- **Timeout**: Set request timeout for health checks
- **Failure Threshold**: Number of consecutive failures before marking as unhealthy
- **Success Threshold**: Number of consecutive successes to mark as healthy again

#### Custom Headers
Add custom headers to requests and responses:
- **Request Headers**: Headers sent to upstream servers
- **Response Headers**: Headers returned to clients
- **Common Use Cases**: CORS headers, authentication tokens, custom API headers

#### IP Access Control
Restrict access based on client IP addresses:
- **Whitelist Mode**: Only allow specified IP addresses/ranges
- **Blacklist Mode**: Block specified IP addresses/ranges
- **CIDR Support**: Use CIDR notation for IP ranges (e.g., `192.168.1.0/24`)
- **Multiple IPs**: Add multiple IP addresses or ranges separated by commas

#### Audit Logging
All configuration changes are automatically logged:
- **User Actions**: Track who made what changes
- **Timestamps**: When changes were made
- **Change Details**: What was modified
- **System Events**: Automatic system actions and health check status changes

#### Custom Caddy JSON Snippets
Advanced users can insert raw Caddy JSON snippets into their proxy configurations for features not directly exposed in the UI:
- **Deep Merge**: Custom JSON is deep-merged with UI-generated configuration
- **Override Control**: Custom values override any conflicting keys from the UI
- **Validation**: Built-in JSON validation prevents malformed configurations
- **Safety**: Clear warnings inform users about the advanced nature of the feature

To use this feature:
1. Open the "Advanced" section when creating or editing a proxy
2. Enter valid Caddy JSON in the provided text area
3. The JSON will be merged with the generated configuration when applied

Example use cases:
- Custom middleware handlers
- Advanced routing rules
- Specialized TLS configurations
- Custom logging formats

**Warning**: This is an advanced feature. Incorrect JSON syntax can break your proxy or the entire Caddy server.

### SSL Certificate Options

#### Automatic HTTPS (Recommended)
- **HTTP-01 Challenge**: Standard Let's Encrypt validation (requires port 80 accessible)
- **DNS-01 Challenge**: DNS-based validation (works behind firewalls)

#### DNS Challenge Configuration

For DNS challenges, you can configure credentials in two ways:

**Option 1: Through the UI (Recommended)**
1. Select "DNS-01 Challenge" when creating a proxy
2. Choose your DNS provider
3. Enter your API credentials directly in the form

**Option 2: Environment Variables**
Set these in your docker-compose.yml or environment:
```bash
# Cloudflare
CLOUDFLARE_API_TOKEN=your-api-token

# DigitalOcean
DO_AUTH_TOKEN=your-do-token

# DuckDNS
DUCKDNS_TOKEN=your-duckdns-token


# Hetzner
HETZNER_API_TOKEN=your-hetzner-token

# Gandi
GANDI_BEARER_TOKEN=your-gandi-token

# DNSimple
DNSIMPLE_API_ACCESS_TOKEN=your-dnsimple-token
```

### Supported DNS Providers

| Provider | Credentials Required | Notes |
|----------|---------------------|-------|
| **Cloudflare** | API Token | Create token with Zone:DNS:Edit permissions |
| **DigitalOcean** | Auth Token | Personal Access Token with write scope |
| **DuckDNS** | Token | Your DuckDNS account token |
| **Hetzner** | API Token | Create token in Hetzner Cloud Console |
| **Gandi** | Bearer Token | Personal Access Token (API Key deprecated) |
| **DNSimple** | API Access Token | Generate token in account settings |

## 🛠 Development

### Project Structure
```
├── backend/              # Go backend server
│   ├── cmd/server/      # Main application entry point
│   ├── internal/        # Internal packages
│   │   └── handlers/    # HTTP handlers
│   └── pkg/             # Public packages
│       ├── caddy/       # Caddy API client
│       └── models/      # Data models
├── frontend/            # Vue.js frontend
│   ├── src/
│   │   ├── components/  # Vue components
│   │   ├── views/       # Page views
│   │   └── services/    # API services
├── docker/              # Docker configuration files
└── docker-compose.yml   # Container orchestration
```

### Development Commands

Using the included Justfile:
```bash
# Start backend development server
just backend-run

# Start frontend development server
just frontend-dev

# Install dependencies for both
just setup

# Build both frontend and backend
just build
```

Or manually:
```bash
# Backend
cd backend && go run ./cmd/server

# Frontend
cd frontend && npm run dev
```

### Building Custom Caddy

The project uses xcaddy to build Caddy with DNS plugins:
```bash
xcaddy build \
    --with github.com/caddy-dns/cloudflare \
    --with github.com/caddy-dns/digitalocean \
    --with github.com/caddy-dns/duckdns \
    --with github.com/caddy-dns/hetzner \
    --with github.com/caddy-dns/gandi \
    --with github.com/caddy-dns/dnsimple
```

## 🔧 Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `CADDY_CONFIG_FILE` | Path to Caddy config JSON | `/config/caddy-config.json` |
| `STATIC_DIR` | Frontend static files directory | `/var/www/html` |
| `CLOUDFLARE_API_TOKEN` | Cloudflare DNS API token | - |
| `DO_AUTH_TOKEN` | DigitalOcean auth token | - |
| `DUCKDNS_TOKEN` | DuckDNS token | - |
| `HETZNER_API_TOKEN` | Hetzner DNS API token | - |
| `GANDI_BEARER_TOKEN` | Gandi bearer token | - |
| `DNSIMPLE_API_ACCESS_TOKEN` | DNSimple API access token | - |

### Ports

| Port | Service | Description |
|------|---------|-------------|
| `80` | HTTP | Proxy traffic and ACME challenges |
| `443` | HTTPS | Secure proxy traffic |
| `8080` | Proxy Manager | Web management interface |

## 🐳 Docker Configuration

The Docker setup includes:
- **Multi-stage build** for optimized image size
- **Caddy with DNS plugins** pre-compiled
- **Supervisord** for process management
- **Persistent volumes** for config and certificates

### Volumes
- `./config` → `/config` - Configuration persistence
- `./data` → `/data` - Caddy data (certificates, etc.)
- `./logs` → `/var/log` - Application logs

## 🔒 Security

- **Credentials**: Never logged or exposed in responses
- **HTTPS**: Automatic certificate management
- **API**: RESTful API with input validation
- **Environment**: Secure credential storage options

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- [Caddy](https://caddyserver.com/) - The amazing web server that makes this all possible
- [Vue.js](https://vuejs.org/) - The progressive JavaScript framework
- [DaisyUI](https://daisyui.com/) - Beautiful UI components

---

**Note**: This project is in active development. Features and documentation are continuously being improved. Most of the code is written by Claude Code.
