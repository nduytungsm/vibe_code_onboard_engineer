#!/bin/bash
# Production Deployment Script for Repository Analyzer

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
APP_DIR="/opt/repo-analyzer"
DOMAIN=""
EMAIL=""

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
if [[ $EUID -eq 0 ]]; then
   print_error "This script should not be run as root"
   exit 1
fi

# Check if required arguments are provided
if [ $# -lt 2 ]; then
    print_error "Usage: $0 <domain> <email>"
    print_error "Example: $0 repo-analyzer.yourdomain.com admin@yourdomain.com"
    exit 1
fi

DOMAIN=$1
EMAIL=$2

print_status "Starting deployment for Repository Analyzer"
print_status "Domain: $DOMAIN"
print_status "Email: $EMAIL"

# Check if Docker is installed and running
if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed. Please run vps-setup.sh first."
    exit 1
fi

if ! docker info &> /dev/null; then
    print_error "Docker is not running or user is not in docker group."
    print_warning "Try: sudo systemctl start docker && newgrp docker"
    exit 1
fi

# Create application directory if it doesn't exist
if [ ! -d "$APP_DIR" ]; then
    print_status "Creating application directory..."
    sudo mkdir -p "$APP_DIR"
    sudo chown $USER:$USER "$APP_DIR"
fi

# Navigate to application directory
cd "$APP_DIR"

# Check if source code exists
if [ ! -f "docker-compose.prod.yml" ]; then
    print_error "Application source code not found in $APP_DIR"
    print_warning "Please copy your application files to $APP_DIR first"
    exit 1
fi

# Check if .env file exists
if [ ! -f ".env" ]; then
    print_warning "Environment file not found. Creating from template..."
    cp deployment/.env.production .env
    print_warning "Please edit .env file with your configuration before continuing"
    exit 1
fi

# Update domain in nginx configuration
print_status "Updating nginx configuration with domain: $DOMAIN"
sed -i "s/YOUR_DOMAIN_HERE/$DOMAIN/g" deployment/nginx.prod.conf

# Stop any running containers
print_status "Stopping existing containers..."
docker-compose -f docker-compose.prod.yml down || true

# Build and start containers
print_status "Building and starting containers..."
docker-compose -f docker-compose.prod.yml build --no-cache
docker-compose -f docker-compose.prod.yml up -d

# Wait for services to be ready
print_status "Waiting for services to be ready..."
sleep 30

# Check if services are healthy
print_status "Checking service health..."
if docker-compose -f docker-compose.prod.yml ps | grep -q "unhealthy"; then
    print_error "Some services are unhealthy. Check logs:"
    docker-compose -f docker-compose.prod.yml logs
    exit 1
fi

# Set up SSL certificate
print_status "Setting up SSL certificate..."
if [ ! -f "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" ]; then
    print_status "Obtaining SSL certificate from Let's Encrypt..."
    sudo certbot certonly --nginx -d "$DOMAIN" --email "$EMAIL" --agree-tos --non-interactive
    
    if [ $? -eq 0 ]; then
        print_success "SSL certificate obtained successfully"
        
        # Restart nginx to use SSL
        print_status "Restarting nginx with SSL configuration..."
        docker-compose -f docker-compose.prod.yml restart nginx
    else
        print_warning "Failed to obtain SSL certificate. Continuing without SSL."
    fi
else
    print_success "SSL certificate already exists"
fi

# Set up auto-renewal for SSL certificate
print_status "Setting up SSL certificate auto-renewal..."
sudo crontab -l 2>/dev/null | grep -v "certbot renew" | sudo crontab -
(sudo crontab -l 2>/dev/null; echo "0 12 * * * /usr/bin/certbot renew --quiet && docker-compose -f $APP_DIR/docker-compose.prod.yml restart nginx") | sudo crontab -

# Set up log rotation
print_status "Setting up log rotation..."
sudo tee /etc/logrotate.d/repo-analyzer > /dev/null << EOF
/opt/repo-analyzer/logs/*.log {
    daily
    missingok
    rotate 30
    compress
    notifempty
    create 644 root root
    postrotate
        docker-compose -f /opt/repo-analyzer/docker-compose.prod.yml restart
    endscript
}
EOF

# Create systemd service for auto-start
print_status "Creating systemd service..."
sudo tee /etc/systemd/system/repo-analyzer.service > /dev/null << EOF
[Unit]
Description=Repository Analyzer
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=$APP_DIR
ExecStart=/usr/local/bin/docker-compose -f docker-compose.prod.yml up -d
ExecStop=/usr/local/bin/docker-compose -f docker-compose.prod.yml down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl enable repo-analyzer.service
sudo systemctl daemon-reload

print_success "Deployment completed successfully!"
print_status "Your Repository Analyzer is now running at:"
print_status "HTTP:  http://$DOMAIN"
print_status "HTTPS: https://$DOMAIN"
print_status ""
print_status "Useful commands:"
print_status "  View logs:    docker-compose -f $APP_DIR/docker-compose.prod.yml logs -f"
print_status "  Restart:      docker-compose -f $APP_DIR/docker-compose.prod.yml restart"
print_status "  Stop:         docker-compose -f $APP_DIR/docker-compose.prod.yml down"
print_status "  Update:       cd $APP_DIR && git pull && docker-compose -f docker-compose.prod.yml build --no-cache && docker-compose -f docker-compose.prod.yml up -d"
print_status ""
print_warning "Don't forget to:"
print_warning "  1. Configure your OpenAI API key in .env file"
print_warning "  2. Set up DNS A record for $DOMAIN pointing to your server IP"
print_warning "  3. Configure firewall rules if needed"
