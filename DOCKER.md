# 🐳 TenangDB Docker Guide

## 📦 Docker Images

TenangDB provides multi-platform Docker images via GitHub Container Registry:

- **Registry**: `ghcr.io/abdullahainun/tenangdb`
- **Platforms**: `linux/amd64`, `linux/arm64`
- **Tags**: `latest`, `v1.x.x`, `main`

## 🚀 Quick Start

### Pull and Run
```bash
# Pull latest image
docker pull ghcr.io/abdullahainun/tenangdb:latest

# Basic backup
docker run --rm -v $(pwd)/config.yaml:/config.yaml ghcr.io/abdullahainun/tenangdb:latest backup

# Interactive mode
docker run --rm -it -v $(pwd)/config.yaml:/config.yaml ghcr.io/abdullahainun/tenangdb:latest --help
```

### Using Docker Compose
```bash
# Get compose file
curl -L https://raw.githubusercontent.com/abdullahainun/tenangdb/main/docker-compose.yml -o docker-compose.yml

# Edit configuration
nano docker-compose.yml

# Start services
docker-compose up -d

# View logs
docker-compose logs -f tenangdb
```

## ⚙️ Volume Mounts

### Required Volumes
```bash
# Configuration file
-v $(pwd)/config.yaml:/config.yaml:ro

# Backup directory (for persistent storage)
-v $(pwd)/backups:/backups

# Logs directory
-v $(pwd)/logs:/logs
```

### Example with all volumes
```bash
docker run --rm \
  -v $(pwd)/config.yaml:/config.yaml:ro \
  -v $(pwd)/backups:/backups \
  -v $(pwd)/logs:/logs \
  ghcr.io/abdullahainun/tenangdb:latest backup
```

## 🔧 Configuration

### Environment Variables
```bash
# Set timezone
-e TZ=Asia/Jakarta

# Override config path
-e CONFIG_PATH=/custom/config.yaml

# Set log level
-e LOG_LEVEL=debug
```

### Custom Configuration
```yaml
# config.yaml
database:
  host: host.docker.internal  # Use for localhost from container
  port: 3306
  username: backup_user
  password: secure_password

backup:
  directory: /backups  # Container path
  
logging:
  file_path: /logs/tenangdb.log  # Container path
```

## 🎯 Use Cases

### 1. One-off Backup
```bash
docker run --rm \
  -v $(pwd)/config.yaml:/config.yaml:ro \
  -v $(pwd)/backups:/backups \
  ghcr.io/abdullahainun/tenangdb:latest backup
```

### 2. Scheduled Backup (Cron)
```bash
# Add to crontab
0 2 * * * docker run --rm -v /path/to/config.yaml:/config.yaml:ro -v /path/to/backups:/backups ghcr.io/abdullahainun/tenangdb:latest backup
```

### 3. Kubernetes Deployment
```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: tenangdb-backup
spec:
  schedule: "0 2 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: tenangdb
            image: ghcr.io/abdullahainun/tenangdb:latest
            args: ["backup"]
            volumeMounts:
            - name: config
              mountPath: /config.yaml
              subPath: config.yaml
            - name: backups
              mountPath: /backups
          volumes:
          - name: config
            configMap:
              name: tenangdb-config
          - name: backups
            persistentVolumeClaim:
              claimName: tenangdb-backups
          restartPolicy: OnFailure
```

### 4. Docker Compose with MySQL
```yaml
version: '3.8'
services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: rootpass
      MYSQL_DATABASE: testdb
    volumes:
      - mysql_data:/var/lib/mysql
    
  tenangdb:
    image: ghcr.io/abdullahainun/tenangdb:latest
    depends_on:
      - mysql
    volumes:
      - ./config.yaml:/config.yaml:ro
      - ./backups:/backups
    environment:
      - TZ=Asia/Jakarta
    command: ["backup"]
    
volumes:
  mysql_data:
```

## 🛡️ Security

### Non-root User
The Docker image runs as non-root user (UID 1001) for security:
```dockerfile
USER 1001:1001
```

### Read-only Config
Mount config as read-only:
```bash
-v $(pwd)/config.yaml:/config.yaml:ro
```

### Secrets Management
```bash
# Using Docker secrets
docker run --rm \
  -v $(pwd)/config.yaml:/config.yaml:ro \
  --secret mysql_password \
  ghcr.io/abdullahainun/tenangdb:latest backup
```

## 🔍 Troubleshooting

### Common Issues

**1. Permission Denied**
```bash
# Fix volume permissions
sudo chown -R 1001:1001 ./backups ./logs
```

**2. Config Not Found**
```bash
# Check mount path
docker run --rm -v $(pwd)/config.yaml:/config.yaml:ro ghcr.io/abdullahainun/tenangdb:latest config
```

**3. Database Connection**
```bash
# Test from container
docker run --rm -it --net host ghcr.io/abdullahainun/tenangdb:latest
# Inside container: ping your_db_host
```

**4. Debug Mode**
```bash
# Enable debug logging
docker run --rm \
  -v $(pwd)/config.yaml:/config.yaml:ro \
  -e LOG_LEVEL=debug \
  ghcr.io/abdullahainun/tenangdb:latest backup
```

### Health Checks
```bash
# Check container health
docker run --rm ghcr.io/abdullahainun/tenangdb:latest version

# Exec into running container
docker exec -it tenangdb_container /bin/sh
```

## 📊 Monitoring

### Container Logs
```bash
# View logs
docker logs tenangdb_container

# Follow logs
docker logs -f tenangdb_container

# With docker-compose
docker-compose logs -f tenangdb
```

### Health Check
```yaml
# In docker-compose.yml
healthcheck:
  test: ["CMD", "/tenangdb", "version"]
  interval: 30s
  timeout: 10s
  retries: 3
```

## 🎨 Advanced Usage

### Custom Entrypoint
```bash
# Custom script
docker run --rm \
  -v $(pwd)/backup-script.sh:/backup-script.sh \
  --entrypoint /backup-script.sh \
  ghcr.io/abdullahainun/tenangdb:latest
```

### Build from Source
```bash
# Clone repository
git clone https://github.com/abdullahainun/tenangdb.git
cd tenangdb

# Build custom image
docker build -t my-tenangdb .

# Run custom image
docker run --rm my-tenangdb version
```

### Multi-stage Build
```dockerfile
FROM ghcr.io/abdullahainun/tenangdb:latest as base
COPY custom-config.yaml /config.yaml
CMD ["backup"]
```

## 📋 Best Practices

1. **Use specific tags** instead of `latest` in production
2. **Mount config as read-only** for security
3. **Use persistent volumes** for backups and logs
4. **Set proper timezone** with `TZ` environment variable
5. **Monitor container health** with health checks
6. **Run as non-root** (default in image)
7. **Use secrets management** for sensitive data
8. **Regular image updates** for security patches

## 🎯 Production Deployment

### Docker Swarm
```yaml
version: '3.8'
services:
  tenangdb:
    image: ghcr.io/abdullahainun/tenangdb:v1.1.3
    deploy:
      replicas: 1
      restart_policy:
        condition: on-failure
    volumes:
      - /etc/tenangdb/config.yaml:/config.yaml:ro
      - tenangdb_backups:/backups
    environment:
      - TZ=Asia/Jakarta
    command: ["backup"]

volumes:
  tenangdb_backups:
    driver: local
```

### Kubernetes
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tenangdb
spec:
  replicas: 1
  selector:
    matchLabels:
      app: tenangdb
  template:
    metadata:
      labels:
        app: tenangdb
    spec:
      containers:
      - name: tenangdb
        image: ghcr.io/abdullahainun/tenangdb:v1.1.3
        command: ["backup"]
        volumeMounts:
        - name: config
          mountPath: /config.yaml
          subPath: config.yaml
        - name: backups
          mountPath: /backups
      volumes:
      - name: config
        configMap:
          name: tenangdb-config
      - name: backups
        persistentVolumeClaim:
          claimName: tenangdb-backups
```

---

## 🔗 Links

- **GitHub**: https://github.com/abdullahainun/tenangdb
- **Container Registry**: https://ghcr.io/abdullahainun/tenangdb
- **Documentation**: [README.md](README.md)
- **Issues**: https://github.com/abdullahainun/tenangdb/issues