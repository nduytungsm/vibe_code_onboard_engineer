#!/bin/bash
# Update script for Repository Analyzer

set -e

APP_DIR="/opt/repo-analyzer"
BACKUP_DIR="/opt/repo-analyzer-backups"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

cd "$APP_DIR"

# Create backup
print_status "Creating backup..."
sudo mkdir -p "$BACKUP_DIR"
BACKUP_NAME="backup-$(date +%Y%m%d-%H%M%S)"
sudo cp -r "$APP_DIR" "$BACKUP_DIR/$BACKUP_NAME"
print_success "Backup created: $BACKUP_DIR/$BACKUP_NAME"

# Pull latest code
print_status "Pulling latest code..."
git pull origin main

# Rebuild and restart containers
print_status "Rebuilding containers..."
docker-compose -f docker-compose.prod.yml build --no-cache

print_status "Restarting services..."
docker-compose -f docker-compose.prod.yml down
docker-compose -f docker-compose.prod.yml up -d

# Wait for services to be ready
print_status "Waiting for services to start..."
sleep 30

# Check health
print_status "Checking service health..."
if docker-compose -f docker-compose.prod.yml ps | grep -q "unhealthy"; then
    print_error "Update failed - services are unhealthy!"
    print_warning "Rolling back..."
    docker-compose -f docker-compose.prod.yml down
    sudo rm -rf "$APP_DIR"
    sudo cp -r "$BACKUP_DIR/$BACKUP_NAME" "$APP_DIR"
    cd "$APP_DIR"
    docker-compose -f docker-compose.prod.yml up -d
    print_error "Rolled back to previous version"
    exit 1
fi

# Clean up old Docker images
print_status "Cleaning up old Docker images..."
docker image prune -f

print_success "Update completed successfully!"
print_status "Services are running and healthy"
