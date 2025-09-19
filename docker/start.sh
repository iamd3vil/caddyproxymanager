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

# Ensure log directories exist (safety check)
echo "📁 Ensuring directories exist..."
mkdir -p /var/log/caddy /var/log/proxy-manager /var/log /var/run /data
chmod 755 /data
echo "   ✅ Log and data directories created"

# Debug: Check if files exist
echo "🔍 Debug: Checking file structure..."
echo "   Static dir contents: $(ls -la /var/www/html 2>/dev/null | wc -l) files"
echo "   Proxy manager exists: $(test -f /usr/local/bin/proxy-manager && echo 'YES' || echo 'NO')"
echo "   Proxy manager permissions: $(ls -la /usr/local/bin/proxy-manager 2>/dev/null || echo 'NOT FOUND')"
echo "   Data dir: $(ls -la /data 2>/dev/null || echo 'NOT ACCESSIBLE')"

# Test proxy-manager directly to see the error
echo "🧪 Testing proxy-manager directly..."
export CADDY_ADMIN_URL="${CADDY_ADMIN_URL:-http://localhost:2019}"
export STATIC_DIR="${STATIC_DIR:-/var/www/html}"
export PORT="${PORT:-8080}"
export DATA_DIR="${DATA_DIR:-/data}"
echo "   Using PORT: $PORT"
timeout 3 /usr/local/bin/proxy-manager 2>&1 | head -10 || echo "   Direct test failed or timed out"

# Start supervisor to manage both processes
echo "🏁 Starting services with supervisor..."
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf