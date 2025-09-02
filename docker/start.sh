#!/bin/sh
set -e

echo "🚀 Starting Caddy Proxy Manager Container"
echo "========================================="

# Display environment info
echo "📋 Environment Information:"
echo "   - Caddy Admin API: http://localhost:2019"
echo "   - Proxy Manager API: http://localhost:8080"  
echo "   - Frontend: http://localhost (port 80)"
echo "   - Config file: /config/caddy-config.json"
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

# Start supervisor to manage both processes
echo "🏁 Starting services with supervisor..."
exec /usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf