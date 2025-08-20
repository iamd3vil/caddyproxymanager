# Caddy Proxy Manager

A modern web-based management interface for Caddy reverse proxy configurations, similar to nginx proxy manager but built specifically for Caddy.

## ✨ Features

- **🌐 Web UI**: Clean, modern interface built with Vue 3, Vite, and DaisyUI
- **🔒 Automatic HTTPS**: Let's Encrypt integration with HTTP-01 and DNS-01 challenges
- **🌍 DNS Challenge Support**: Works behind firewalls with DNS providers (Cloudflare, DigitalOcean, DuckDNS)
- **⚡ Real-time Management**: Direct integration with Caddy Admin API
- **🐳 Containerized**: Complete Docker setup with all dependencies included
- **🔧 Easy Configuration**: No complex config files - manage everything through the UI
- **📊 Status Monitoring**: Real-time proxy status and health monitoring

## 🚀 Quick Start

### Using Docker Compose (Recommended)

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd caddyproxymanager
   ```

2. **Start the services**
   ```bash
   docker-compose up -d
   ```

3. **Access the web interface**
   - Proxy Manager UI: http://localhost:8080
   - Caddy Admin API: http://localhost:2019 (optional, for debugging)

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
```

### Supported DNS Providers

| Provider | Credentials Required | Notes |
|----------|---------------------|-------|
| **Cloudflare** | API Token | Create token with Zone:DNS:Edit permissions |
| **DigitalOcean** | Auth Token | Personal Access Token with write scope |
| **DuckDNS** | Token | Your DuckDNS account token |

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
    --with github.com/caddy-dns/duckdns
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

### Ports

| Port | Service | Description |
|------|---------|-------------|
| `80` | HTTP | Proxy traffic and ACME challenges |
| `443` | HTTPS | Secure proxy traffic |
| `8080` | Proxy Manager | Web management interface |
| `2019` | Caddy Admin | Caddy API (optional) |

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
