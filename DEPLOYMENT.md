# 🚀 Repository Analyzer Deployment Guide

This guide provides multiple deployment options for your repository analyzer application.

## 📋 Prerequisites

- Docker and Docker Compose installed
- Git installed (for repository cloning functionality)
- **OpenAI API Key** (required for analysis functionality)
- At least 2GB RAM and 2 CPU cores recommended
- Port 80 and 8080 available (or configure different ports)

## 🔑 Setup OpenAI API Key

The application requires an OpenAI API key for repository analysis. Get your API key from [OpenAI's platform](https://platform.openai.com/api-keys).

### Option 1: Environment Variable (Recommended)
```bash
export OPENAI_API_KEY="your-api-key-here"
```

### Option 2: .env File
```bash
# Copy the example file
cp .env.example .env

# Edit .env file and add your API key
# OPENAI_API_KEY=your-api-key-here
```

## 🎯 Deployment Options

### Option 1: Docker Compose (Recommended for Production)

**Best for**: Production environments, scalability, service separation

```bash
# Quick start
./deploy.sh

# Manual deployment
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

**Architecture:**
- Backend: Go service on port 8080
- Frontend: Nginx serving React app on port 80
- Services communicate via Docker network
- Automatic health checks and restarts

**Access:**
- Application: http://localhost
- API: http://localhost:8080
- Health Check: http://localhost:8080/health

### Option 2: Single Container (Simple Deployment)

**Best for**: Simple deployments, single-server setups

```bash
# Build combined image
docker build -f Dockerfile.combined -t repo-analyzer .

# Run container
docker run -d -p 8080:8080 --name repo-analyzer repo-analyzer

# Check logs
docker logs repo-analyzer
```

**Architecture:**
- Single container with backend serving static frontend
- All traffic through port 8080
- Backend serves React build files

**Access:**
- Application: http://localhost:8080
- API: http://localhost:8080/api
- Health Check: http://localhost:8080/health

### Option 3: Cloud Platform Deployment

#### Railway

```bash
# Install Railway CLI
npm install -g @railway/cli

# Login and deploy
railway login
railway up
```

#### Render

1. Connect your GitHub repository to Render
2. Create a new Web Service
3. Use `Dockerfile.combined` as the Docker file
4. Set environment variables as needed

#### Heroku

```bash
# Install Heroku CLI and login
heroku create your-app-name
heroku container:push web --app your-app-name
heroku container:release web --app your-app-name
```

## ⚙️ Configuration

### Environment Variables

**Backend:**
- `GO_ENV`: Set to "production" for production deployment
- `PORT`: Port for the backend service (default: 8080)

**Frontend:**
- `VITE_API_URL`: API base URL (auto-configured for each deployment)
- `NODE_ENV`: Set to "production" for production builds

### Custom Configuration

1. **Custom Ports:**
   ```yaml
   # docker-compose.yml
   services:
     frontend:
       ports:
         - "3000:80"  # Change external port
     backend:
       ports:
         - "8000:8080"  # Change external port
   ```

2. **Resource Limits:**
   ```yaml
   # docker-compose.yml
   services:
     backend:
       deploy:
         resources:
           limits:
             cpus: '2'
             memory: 2G
   ```

3. **SSL/HTTPS:**
   - Add SSL certificates to nginx configuration
   - Use a reverse proxy like Traefik or Cloudflare

## 🔧 Maintenance Commands

```bash
# View service status
docker-compose ps

# View logs
docker-compose logs -f [service_name]

# Update services
docker-compose pull
docker-compose up -d

# Restart specific service
docker-compose restart backend

# Clean up unused images
docker system prune

# Backup analysis cache
docker cp repo-analyzer-backend:/tmp/repo-analysis ./backup/
```

## 📊 Monitoring and Health Checks

### Built-in Health Checks

Both deployment options include health checks:

- **Backend**: `GET /health`
- **Frontend**: HTTP response check
- **Docker**: Automatic container restart on failure

### Log Monitoring

```bash
# Real-time logs
docker-compose logs -f

# Backend logs only
docker-compose logs -f backend

# Frontend logs only
docker-compose logs -f frontend

# Export logs
docker-compose logs > app-logs.txt
```

## 🛡️ Security Considerations

### Production Security Checklist

- [ ] Use HTTPS in production
- [ ] Set up proper firewall rules
- [ ] Regularly update base images
- [ ] Use secrets management for sensitive data
- [ ] Enable access logging
- [ ] Set up monitoring and alerting

### Network Security

```yaml
# docker-compose.yml - Internal network only
services:
  backend:
    expose:
      - "8080"  # Don't expose externally
    # Remove "ports" section for internal only
```

## 🚀 Performance Optimization

### Resource Allocation

```yaml
# docker-compose.yml
services:
  backend:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 500M
```

### Caching

- Frontend includes intelligent analysis caching
- Consider Redis for distributed caching in multi-instance deployments
- Use CDN for static assets in production

## 🐛 Troubleshooting

### Common Issues

1. **Port Already in Use**
   ```bash
   # Check what's using the port
   lsof -i :8080
   
   # Change ports in docker-compose.yml
   ports:
     - "8081:8080"
   ```

2. **Git Clone Failures**
   ```bash
   # Check if git is available in container
   docker exec repo-analyzer-backend git --version
   
   # Check repository URL accessibility
   docker exec repo-analyzer-backend git clone <test-repo-url> /tmp/test
   ```

3. **Out of Memory**
   ```bash
   # Increase memory limits
   docker-compose down
   # Edit docker-compose.yml to increase memory
   docker-compose up -d
   ```

4. **Frontend Not Loading**
   ```bash
   # Check frontend build
   docker-compose logs frontend
   
   # Verify nginx configuration
   docker exec repo-analyzer-frontend cat /etc/nginx/nginx.conf
   ```

### Debug Mode

```bash
# Run with debug logging
docker-compose -f docker-compose.yml -f docker-compose.debug.yml up -d
```

## 📈 Scaling

### Horizontal Scaling

```yaml
# docker-compose.yml
services:
  backend:
    deploy:
      replicas: 3
  frontend:
    deploy:
      replicas: 2
```

### Load Balancing

Add a load balancer (nginx, HAProxy, or cloud LB) in front of multiple instances.

## 🎉 Success!

Your Repository Analyzer should now be deployed and accessible. The system provides:

- ✅ **Complete repository analysis** with GitHub URL input
- ✅ **Authentication support** for private repositories
- ✅ **Intelligent caching** for performance
- ✅ **Real-time progress** indicators
- ✅ **Comprehensive visualizations** across all tabs
- ✅ **Production-ready deployment** with health checks

For support or issues, check the application logs and refer to this troubleshooting guide.
