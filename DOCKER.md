# 🐳 Docker Deployment Guide

## Overview

This containerized version packages everything you need:
- **Custom Caddy** with Cloudflare & Route53 DNS plugins
- **Proxy Manager Backend** (Go application)
- **Frontend UI** (Vue.js application)
- **Supervisor** to manage all processes

## 🚀 Quick Start

### 1. Build and Run with Docker Compose

```bash
# Clone the repository
git clone <your-repo>
cd caddyproxymanager

# Start the container
docker-compose up -d
```

### 2. Access the Application

- **Frontend UI**: http://localhost
- **Proxy Manager API**: http://localhost:8080
- **Caddy Admin API**: http://localhost:2019 (optional)

## 🔧 Configuration

### DNS Provider Setup

Edit `docker-compose.yml` and uncomment the environment variables for your DNS provider:

#### Cloudflare
```yaml
environment:
  - CLOUDFLARE_EMAIL=your-email@example.com
  - CLOUDFLARE_API_TOKEN=your-api-token
```

#### AWS Route53
```yaml
environment:
  - AWS_ACCESS_KEY_ID=your-access-key
  - AWS_SECRET_ACCESS_KEY=your-secret-key
  - AWS_REGION=us-east-1
```

### Persistent Storage

The container uses volumes for persistence:
- `./config` - Proxy configurations (survives container restarts)
- `./data` - Caddy data (SSL certificates, etc.)
- `./logs` - Application logs

## 📦 Manual Docker Build

```bash
# Build the image
docker build -t caddy-proxy-manager .

# Run the container
docker run -d \
  --name caddy-proxy-manager \
  -p 80:80 \
  -p 443:443 \
  -p 8080:8080 \
  -v $(pwd)/config:/config \
  -v $(pwd)/data:/data \
  caddy-proxy-manager
```

## 🔍 Verification

### Check DNS Plugins
```bash
# Connect to the running container
docker exec -it caddy-proxy-manager /bin/sh

# List available Caddy modules
/usr/bin/caddy list-modules | grep dns.providers
```

You should see:
```
dns.providers.cloudflare
dns.providers.route53
```

### View Logs
```bash
# View all logs
docker-compose logs -f

# View specific service logs
docker exec caddy-proxy-manager tail -f /var/log/proxy-manager/stdout.log
docker exec caddy-proxy-manager tail -f /var/log/caddy/stdout.log
```

## 🏗️ Architecture

```
┌─────────────────────────────────────┐
│          Docker Container          │
├─────────────────────────────────────┤
│  ┌─────────────┐  ┌──────────────┐  │
│  │   Caddy     │  │ Proxy Mgr    │  │
│  │   :80/:443  │  │ API :8080    │  │
│  │             │  │              │  │
│  │ DNS Plugins │◄─┤ Admin API    │  │
│  │ - Cloudflare│  │ :2019        │  │
│  │ - Route53   │  │              │  │
│  └─────────────┘  └──────────────┘  │
│  ┌─────────────────────────────────┐ │
│  │      Frontend (Vue.js)          │ │
│  │      Static Files               │ │
│  └─────────────────────────────────┘ │
├─────────────────────────────────────┤
│           Supervisor                │
│      (Process Management)           │
└─────────────────────────────────────┘
```

## 🛠️ Troubleshooting

### Container Won't Start
```bash
# Check container logs
docker-compose logs caddy-proxy-manager

# Check if ports are available
lsof -i :80
lsof -i :443
```

### DNS Challenges Not Working
1. Verify DNS provider credentials in `docker-compose.yml`
2. Check DNS provider API access
3. Ensure domain DNS is managed by the provider

### Config Not Persisting
Ensure the `./config` directory is mounted and writable:
```bash
mkdir -p config data logs
chmod 755 config data logs
```

## 🔄 Updates

```bash
# Rebuild and restart
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

## 📊 Health Checks

The container includes health checks accessible at:
```bash
curl http://localhost/api/health
```

## 🔒 Security Notes

- The Caddy Admin API (port 2019) is exposed for management
- Consider restricting access in production environments
- DNS provider credentials are passed via environment variables
- SSL certificates are automatically managed by Caddy

## 📋 Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `CADDY_ADMIN_URL` | Caddy Admin API URL | `http://localhost:2019` |
| `CADDY_CONFIG_FILE` | Config file path | `/config/caddy-config.json` |
| `STATIC_DIR` | Frontend files path | `/var/www/html` |
| `PORT` | Backend API port | `8080` |
| `CLOUDFLARE_EMAIL` | Cloudflare account email | - |
| `CLOUDFLARE_API_TOKEN` | Cloudflare API token | - |
| `AWS_ACCESS_KEY_ID` | AWS access key | - |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key | - |
| `AWS_REGION` | AWS region | - |