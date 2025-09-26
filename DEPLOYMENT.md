# 🚀 VPS Deployment Guide

Complete step-by-step guide to deploy Repository Analyzer on a VPS with both backend and frontend.

## 📋 Prerequisites

- **VPS**: Ubuntu 20.04+ with minimum 2GB RAM, 20GB disk
- **Domain**: A domain name pointing to your VPS IP
- **OpenAI API Key**: For repository analysis functionality

## 🔧 Step 1: Initial VPS Setup

### 1.1 Connect to your VPS
```bash
ssh root@your-server-ip
```

### 1.2 Run the setup script
```bash
# Copy the deployment files to your VPS
scp -r deployment/ root@your-server-ip:/tmp/

# Run the setup script
chmod +x /tmp/deployment/vps-setup.sh
/tmp/deployment/vps-setup.sh
```

### 1.3 Log out and back in
```bash
exit
ssh root@your-server-ip
```

## 📦 Step 2: Deploy Application Code

### 2.1 Copy your application to VPS
```bash
# On your local machine, copy the application
scp -r . root@your-server-ip:/opt/repo-analyzer/

# Or clone from git
ssh root@your-server-ip
cd /opt/repo-analyzer
git clone https://github.com/your-username/repo-analyzer.git .
```

### 2.2 Set up environment variables
```bash
cd /opt/repo-analyzer
cp deployment/.env.production .env

# Edit the environment file
nano .env
```

**Important**: Update these variables in `.env`:
```bash
OPENAI_API_KEY=your_actual_openai_api_key_here
DOMAIN=your-domain.com
```

## 🌐 Step 3: Configure DNS

Point your domain to your VPS:
- Create an **A record** for `your-domain.com` → `your-vps-ip`
- Wait for DNS propagation (5-30 minutes)

## 🚀 Step 4: Deploy

### 4.1 Run the deployment script
```bash
cd /opt/repo-analyzer
chmod +x deployment/deploy.sh
./deployment/deploy.sh your-domain.com admin@your-domain.com
```

This script will:
- ✅ Build and start Docker containers
- ✅ Configure Nginx with SSL
- ✅ Obtain Let's Encrypt SSL certificate
- ✅ Set up automatic SSL renewal
- ✅ Configure systemd service for auto-start

### 4.2 Verify deployment
```bash
# Check container status
docker-compose -f docker-compose.prod.yml ps

# Check logs
docker-compose -f docker-compose.prod.yml logs -f

# Test endpoints
curl http://your-domain.com/health
curl https://your-domain.com/health
```

## 🔍 Step 5: Testing

### 5.1 Access your application
- **Frontend**: https://your-domain.com
- **API**: https://your-domain.com/api/health
- **Backend Health**: https://your-domain.com/health

### 5.2 Test repository analysis
1. Open https://your-domain.com
2. Enter a GitHub repository URL
3. Click "Analyze Repository"
4. Verify the analysis completes successfully

## 📊 Step 6: Monitoring & Maintenance

### 6.1 Monitor your application
```bash
# Run monitoring script
/opt/repo-analyzer/deployment/monitor.sh

# View logs
docker-compose -f /opt/repo-analyzer/docker-compose.prod.yml logs -f

# Check system resources
htop
df -h
```

### 6.2 Update your application
```bash
# Run update script
/opt/repo-analyzer/deployment/update.sh
```

### 6.3 Backup and restore
```bash
# Backups are automatically created during updates
ls -la /opt/repo-analyzer-backups/

# Manual backup
sudo cp -r /opt/repo-analyzer /opt/repo-analyzer-backups/manual-$(date +%Y%m%d)
```

## 🔧 Configuration Details

### Backend Configuration (`config.yaml`)
```yaml
openai:
  api_key: "${OPENAI_API_KEY}"
  model: "gpt-4o-mini"
  max_tokens_per_request: 4000

rate_limiting:
  concurrent_workers: 6
  requests_per_minute: 500

cache:
  enabled: true
  ttl_hours: 24
```

### Frontend-Backend Communication
The deployment sets up:

1. **Frontend** (React) runs on port 80 inside container
2. **Backend** (Go API) runs on port 8080 inside container  
3. **Nginx** proxy handles:
   - `https://domain.com/` → Frontend
   - `https://domain.com/api/` → Backend
   - `https://domain.com/health` → Backend health

### SSL & Security
- **Let's Encrypt SSL** automatically obtained and renewed
- **HTTPS redirect** for all traffic
- **Security headers** (HSTS, XSS protection, etc.)
- **Rate limiting** on API endpoints
- **GZIP compression** for better performance

## 🚨 Troubleshooting

### Common Issues

#### 1. Containers not starting
```bash
# Check logs
docker-compose -f docker-compose.prod.yml logs

# Check system resources
free -h
df -h

# Restart services
docker-compose -f docker-compose.prod.yml restart
```

#### 2. SSL certificate issues
```bash
# Check certificate status
sudo certbot certificates

# Renew certificate manually
sudo certbot renew

# Restart nginx
docker-compose -f docker-compose.prod.yml restart nginx
```

#### 3. API connection issues
```bash
# Check if backend is responsive
curl http://localhost:8080/health

# Check nginx configuration
docker-compose -f docker-compose.prod.yml exec nginx nginx -t

# Check firewall
sudo ufw status
```

#### 4. Performance issues
```bash
# Check system resources
htop
docker stats

# Clear cache
rm -rf /opt/repo-analyzer/cache/*.json

# Restart with fresh containers
docker-compose -f docker-compose.prod.yml down
docker-compose -f docker-compose.prod.yml up -d
```

## 📈 Performance Optimization

### For Large Repositories
1. **Increase server resources** (4GB+ RAM recommended)
2. **Adjust worker count** in `config.yaml`:
   ```yaml
   rate_limiting:
     concurrent_workers: 8  # Increase for more powerful servers
   ```
3. **Enable cache persistence**:
   ```bash
   # Cache directory is persistent across restarts
   docker volume create repo-analyzer-cache
   ```

### For High Traffic
1. **Scale horizontally** with load balancer
2. **Use CDN** for frontend assets
3. **Implement rate limiting** per user
4. **Monitor with tools** like Grafana/Prometheus

## 🔄 Updates & Maintenance

### Regular Maintenance Tasks
1. **Weekly**: Run monitoring script and check logs
2. **Monthly**: Update system packages and restart services  
3. **Quarterly**: Review and clean old backups
4. **As needed**: Update application code

### Maintenance Commands
```bash
# System updates
sudo apt update && sudo apt upgrade -y

# Docker cleanup
docker system prune -f

# Log rotation
sudo logrotate -f /etc/logrotate.d/repo-analyzer

# SSL renewal (automatic, but can be manual)
sudo certbot renew
```

## 💡 Advanced Configuration

### Custom Domain Configuration
```nginx
# Add to nginx.prod.conf for custom subdomain
server {
    listen 443 ssl http2;
    server_name api.your-domain.com;
    
    location / {
        proxy_pass http://backend;
        # ... other proxy settings
    }
}
```

### Database Integration (Optional)
```yaml
# Add to docker-compose.prod.yml
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: repo_analyzer
      POSTGRES_USER: analyzer
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
```

## 📞 Support

If you encounter issues:
1. Check the troubleshooting section above
2. Review application logs: `docker-compose logs -f`
3. Ensure all environment variables are set correctly
4. Verify DNS and SSL certificate configuration

**Your Repository Analyzer should now be running successfully on your VPS!** 🎉
