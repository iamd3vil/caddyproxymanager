#!/bin/sh
set -e

echo "🚀 Starting Caddy Proxy Manager Container"
echo "========================================="

# Set default environment variables if not provided
export DATA_DIR=${DATA_DIR:-/data}
export STATIC_DIR=${STATIC_DIR:-/var/www/html}

# Display environment info
echo "📋 Environment Information:"
echo "   - Proxy Manager Web UI: http://localhost:8080"  
echo "   - Frontend: http://localhost (port 80)"
echo "   - Data directory: $DATA_DIR"
echo "   - Caddy config will be: $DATA_DIR/caddy-config.json"
echo ""

# Check if Caddy binary has DNS plugins
echo "🔍 Checking Caddy build:"
if /usr/bin/caddy list-modules | grep -q "dns.providers"; then
    echo "   ✅ DNS providers found:"
    /usr/bin/caddy list-modules | grep "dns.providers" | sed 's/^/      /'
else
    echo "   ❌ No DNS providers found"
fi
echo "   Expected: cloudflare, digitalocean, duckdns, hetzner, gandi, dnsimple"
echo ""

# Ensure necessary directories exist
mkdir -p /var/log/caddy /var/log/proxy-manager /var/log /var/run

# Start supervisor to manage both processes
echo "🏁 Starting services with supervisor..."
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf