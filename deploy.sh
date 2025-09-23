#!/bin/bash

# Repository Analyzer Deployment Script
set -e

echo "🚀 Starting Repository Analyzer deployment..."

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed. Please install Docker first."
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo "❌ Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

# Build and start services
echo "🔨 Building Docker images..."
docker-compose build --no-cache

echo "🏃 Starting services..."
docker-compose up -d

echo "⏳ Waiting for services to be ready..."
sleep 30

# Check backend health
echo "🔍 Checking backend health..."
for i in {1..10}; do
    if curl -f http://localhost:8080/health >/dev/null 2>&1; then
        echo "✅ Backend is healthy"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "❌ Backend health check failed"
        docker-compose logs backend
        exit 1
    fi
    echo "⏳ Waiting for backend... ($i/10)"
    sleep 5
done

# Check frontend
echo "🔍 Checking frontend..."
for i in {1..5}; do
    if curl -f http://localhost >/dev/null 2>&1; then
        echo "✅ Frontend is accessible"
        break
    fi
    if [ $i -eq 5 ]; then
        echo "❌ Frontend check failed"
        docker-compose logs frontend
        exit 1
    fi
    echo "⏳ Waiting for frontend... ($i/5)"
    sleep 3
done

echo ""
echo "🎉 Deployment successful!"
echo ""
echo "📍 Access your application at:"
echo "   Frontend: http://localhost"
echo "   Backend API: http://localhost:8080"
echo "   Health Check: http://localhost:8080/health"
echo ""
echo "📋 Useful commands:"
echo "   View logs: docker-compose logs -f"
echo "   Stop services: docker-compose down"
echo "   Restart: docker-compose restart"
echo "   Update: docker-compose pull && docker-compose up -d"
echo ""
echo "🔧 For production deployment, consider:"
echo "   - Using a reverse proxy (nginx, traefik)"
echo "   - Setting up SSL certificates"
echo "   - Configuring proper domain names"
echo "   - Setting up monitoring and logging"
