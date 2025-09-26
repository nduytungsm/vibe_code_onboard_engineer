#!/bin/bash
# Monitoring script for Repository Analyzer

APP_DIR="/opt/repo-analyzer"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_header() {
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE} Repository Analyzer Status${NC}"
    echo -e "${BLUE}================================${NC}"
}

print_section() {
    echo -e "\n${YELLOW}$1${NC}"
    echo "----------------------------------------"
}

cd "$APP_DIR"

print_header

# System resources
print_section "System Resources"
echo "CPU Usage: $(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | sed 's/%us,//')%"
echo "Memory Usage: $(free -h | awk 'NR==2{printf "%.1f%%", $3*100/$2 }')"
echo "Disk Usage: $(df -h / | awk 'NR==2{print $5}')"
echo "Load Average: $(uptime | awk -F'load average:' '{print $2}')"

# Docker containers status
print_section "Container Status"
docker-compose -f docker-compose.prod.yml ps

# Service health checks
print_section "Health Checks"
echo -n "Backend Health: "
if curl -s -f http://localhost:8080/health > /dev/null; then
    echo -e "${GREEN}✓ Healthy${NC}"
else
    echo -e "${RED}✗ Unhealthy${NC}"
fi

echo -n "Frontend Health: "
if curl -s -f http://localhost:80 > /dev/null; then
    echo -e "${GREEN}✓ Healthy${NC}"
else
    echo -e "${RED}✗ Unhealthy${NC}"
fi

echo -n "Nginx Health: "
if curl -s -f http://localhost/health > /dev/null; then
    echo -e "${GREEN}✓ Healthy${NC}"
else
    echo -e "${RED}✗ Unhealthy${NC}"
fi

# SSL certificate status
print_section "SSL Certificate"
if [ -f "/etc/letsencrypt/live/*/fullchain.pem" ]; then
    CERT_FILE=$(ls /etc/letsencrypt/live/*/fullchain.pem | head -1)
    EXPIRY=$(openssl x509 -in "$CERT_FILE" -noout -dates | grep notAfter | cut -d= -f2)
    DAYS_LEFT=$(( ($(date -d "$EXPIRY" +%s) - $(date +%s)) / 86400 ))
    if [ $DAYS_LEFT -gt 30 ]; then
        echo -e "${GREEN}✓ Valid (expires in $DAYS_LEFT days)${NC}"
    elif [ $DAYS_LEFT -gt 7 ]; then
        echo -e "${YELLOW}⚠ Expires in $DAYS_LEFT days${NC}"
    else
        echo -e "${RED}✗ Expires in $DAYS_LEFT days - URGENT RENEWAL NEEDED${NC}"
    fi
else
    echo -e "${RED}✗ No SSL certificate found${NC}"
fi

# Recent logs (errors only)
print_section "Recent Errors (Last 10 lines)"
docker-compose -f docker-compose.prod.yml logs --tail=100 | grep -i error | tail -10 || echo "No recent errors found"

# Storage usage
print_section "Storage Usage"
echo "Cache directory: $(du -sh cache 2>/dev/null || echo 'N/A')"
echo "Temp directory: $(docker exec repo-analyzer-backend du -sh /tmp/repos 2>/dev/null || echo 'N/A')"
echo "Log files: $(docker-compose -f docker-compose.prod.yml logs --no-color 2>/dev/null | wc -l) lines"

print_section "Quick Actions"
echo "View logs:        docker-compose -f $APP_DIR/docker-compose.prod.yml logs -f"
echo "Restart services: docker-compose -f $APP_DIR/docker-compose.prod.yml restart"
echo "Update app:       $APP_DIR/deployment/update.sh"
echo "Full status:      $APP_DIR/deployment/monitor.sh"
