# Multi-stage Dockerfile for Caddy Proxy Manager
# Combines custom Caddy build with DNS plugins + Go backend + Vue frontend

# Stage 1: Build custom Caddy with DNS plugins
FROM caddy:2-builder AS caddy-builder

# Build Caddy with stable DNS plugins (some have compatibility issues)
RUN xcaddy build \
    --with github.com/caddy-dns/cloudflare \
    --with github.com/caddy-dns/digitalocean \
    --with github.com/caddy-dns/duckdns

# Stage 2: Build Go backend
FROM golang:1.25-alpine AS backend-builder

WORKDIR /app/backend

# Copy go mod file for better caching  
COPY backend/go.mod ./
RUN go mod download

# Copy backend source code
COPY backend/ .

# Build the backend binary
RUN CGO_ENABLED=0 GOOS=linux go build -o proxy-manager ./cmd/server

# Stage 3: Build Vue frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy package files first for better caching
COPY frontend/package*.json ./
RUN npm install

# Copy frontend source code
COPY frontend/ .

# Build the frontend
RUN npm run build

# Stage 4: Final runtime image
FROM alpine:3.19

# Install necessary packages
RUN apk --no-cache add \
    ca-certificates \
    supervisor \
    tzdata \
    curl

# Copy custom Caddy binary with DNS plugins
COPY --from=caddy-builder /usr/bin/caddy /usr/bin/caddy

# Copy Go backend binary
COPY --from=backend-builder /app/backend/proxy-manager /usr/local/bin/proxy-manager

# Copy built frontend files (Vite builds to ../backend/static in our config)
COPY --from=frontend-builder /app/backend/static /var/www/html

# Create necessary directories
RUN mkdir -p /etc/caddy /var/log/caddy /var/log/proxy-manager /config

# Copy configuration files
COPY docker/supervisord.conf /etc/supervisor/conf.d/supervisord.conf
COPY docker/caddy-bootstrap.json /config/caddy-bootstrap.json
COPY docker/start.sh /usr/local/bin/start.sh

# Make startup script executable
RUN chmod +x /usr/local/bin/start.sh

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
  CMD curl -f http://localhost/api/health || exit 1

# Expose ports
EXPOSE 80 443 8080 2019

# Set work directory
WORKDIR /config

# Use custom startup script
CMD ["/usr/local/bin/start.sh"]