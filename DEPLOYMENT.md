# TenangDB Production Deployment Guide

TenangDB supports multiple deployment options for production environments. This guide covers Docker Compose and Kubernetes deployments.

## 🏗️ Architecture

TenangDB production setup consists of two main workloads:

1. **Backup Service** - Scheduled backup execution
2. **Metrics Exporter** - Continuous Prometheus metrics HTTP server

## 🐳 Docker Compose Deployment

### Development/Testing

For development and testing:

```bash
# Start development environment
docker-compose up -d

# Run one-time backup
docker-compose exec tenangdb /app/tenangdb backup

# View logs
docker-compose logs -f tenangdb
```

### Production

For production deployments with monitoring:

```bash
# Copy environment file
cp .env.example .env

# Edit .env with your credentials
nano .env

# Start full production stack
docker-compose -f docker-compose.production.yml --profile monitoring up -d

# Or start specific services
docker-compose -f docker-compose.production.yml up -d tenangdb-exporter
```

#### Production Profiles

```bash
# Backup service only
docker-compose -f docker-compose.production.yml --profile backup up -d

# Scheduler service (alternative to cron)
docker-compose -f docker-compose.production.yml --profile scheduler up -d

# Full monitoring stack
docker-compose -f docker-compose.production.yml --profile monitoring up -d

# MySQL for testing
docker-compose -f docker-compose.production.yml --profile mysql up -d
```

### Production Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  TenangDB       │    │  TenangDB       │    │  MySQL          │
│  Backup         │    │  Metrics        │    │  Database       │
│  Service        │    │  Exporter       │    │  Service        │
│                 │    │                 │    │                 │
│  Cron/Schedule  │    │  HTTP :9090     │    │  Port :3306     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │  Prometheus     │
                    │  :9091          │
                    └─────────────────┘
                                 │
                    ┌─────────────────┐
                    │  Grafana        │
                    │  :3000          │
                    └─────────────────┘
```

## ☸️ Kubernetes Deployment

### Quick Start

```bash
# Apply all manifests
kubectl apply -f k8s/

# Or step by step
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/secret.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/pvc.yaml
kubectl apply -f k8s/rbac.yaml
kubectl apply -f k8s/cronjob.yaml
```

### Configuration

1. **Update secrets**:
   ```bash
   kubectl create secret generic tenangdb-secrets \
     --from-literal=MYSQL_USER=backup-user \
     --from-literal=MYSQL_PASSWORD=your-password \
     --namespace=tenangdb
   ```

2. **Update ConfigMap** with your database endpoints and backup configuration.

3. **Verify deployment**:
   ```bash
   kubectl get cronjobs -n tenangdb
   kubectl get pvc -n tenangdb
   ```

## 🔧 Configuration

### Key Configuration Files

- `config.yaml` - Main TenangDB configuration
- `.env` - Environment variables for Docker Compose
- `k8s/secret.yaml` - Kubernetes secrets
- `k8s/configmap.yaml` - Kubernetes configuration

### Critical Volume Mounts

Both Docker Compose and Kubernetes require these volumes:

```yaml
volumes:
  - ./backups:/backups                    # Backup storage
  - tenangdb-tracking:/tmp/tenangdb       # Frequency tracking (CRITICAL)
  - ./logs:/var/log/tenangdb             # Logs
  - tenangdb-metrics:/var/lib/tenangdb   # Metrics storage
```

**⚠️ Important**: The `tenangdb-tracking` volume is critical for frequency checking. Without it, duplicate backups will occur on container restarts.

## 📊 Monitoring

### Metrics Endpoints

- **TenangDB Metrics**: `http://localhost:9090/metrics`
- **Prometheus**: `http://localhost:9091`
- **Grafana**: `http://localhost:3000` (admin/admin)

### Available Metrics

- `tenangdb_backup_success_total` - Successful backups
- `tenangdb_backup_failed_total` - Failed backups
- `tenangdb_backup_duration_seconds` - Backup duration
- `tenangdb_backup_size_bytes` - Backup size
- `tenangdb_upload_success_total` - Successful uploads
- `tenangdb_system_health` - System health status

## 🚀 Scaling

### Docker Compose Scaling

```bash
# Scale exporter service
docker-compose -f docker-compose.production.yml up -d --scale tenangdb-exporter=3

# Run multiple backup jobs
docker-compose -f docker-compose.production.yml run --rm tenangdb-backup backup --databases db1
docker-compose -f docker-compose.production.yml run --rm tenangdb-backup backup --databases db2
```

### Kubernetes Scaling

```bash
# Scale exporter deployment
kubectl scale deployment tenangdb-exporter --replicas=3 -n tenangdb

# Create multiple backup jobs
kubectl create job tenangdb-backup-db1 --from=cronjob/tenangdb-backup -n tenangdb
kubectl create job tenangdb-backup-db2 --from=cronjob/tenangdb-backup -n tenangdb
```

## 🔐 Security

### Database Permissions

Create dedicated backup user with minimal permissions:

```sql
CREATE USER 'backup-user'@'%' IDENTIFIED BY 'secure-password';
GRANT SELECT, LOCK TABLES, SHOW VIEW, EVENT, TRIGGER ON *.* TO 'backup-user'@'%';
GRANT REPLICATION CLIENT ON *.* TO 'backup-user'@'%';
FLUSH PRIVILEGES;
```

### Secret Management

**Docker Compose**:
- Use `.env` files for sensitive data
- Mount secrets from external files
- Use Docker secrets in Swarm mode

**Kubernetes**:
- Store credentials in Kubernetes Secrets
- Use external secret management (Vault, AWS Secrets Manager)
- Implement RBAC for secret access

## 🛠️ Operations

### Manual Backup

**Docker Compose**:
```bash
docker-compose -f docker-compose.production.yml run --rm tenangdb-backup backup --force
```

**Kubernetes**:
```bash
kubectl create job tenangdb-manual-backup --from=cronjob/tenangdb-backup -n tenangdb
```

### Restore Operations

**Docker Compose**:
```bash
docker-compose -f docker-compose.production.yml run --rm -it tenangdb-backup restore
```

**Kubernetes**:
```bash
kubectl run tenangdb-restore --rm -it --image=ghcr.io/abdullahainun/tenangdb:latest --restart=Never -- restore
```

### Log Management

**Docker Compose**:
```bash
# View logs
docker-compose -f docker-compose.production.yml logs -f tenangdb-exporter

# Log rotation (add to docker-compose.yml)
logging:
  driver: "json-file"
  options:
    max-size: "10m"
    max-file: "3"
```

**Kubernetes**:
```bash
# View logs
kubectl logs -n tenangdb -l app=tenangdb

# Log aggregation with ELK/Fluentd
kubectl logs -n tenangdb -l app=tenangdb --tail=100
```

## 📋 Troubleshooting

### Common Issues

1. **Backup frequency not working**
   - Ensure `tenangdb-tracking` volume is mounted
   - Check `/tmp/tenangdb` directory permissions

2. **Metrics not updating**
   - Verify exporter service is running
   - Check metrics file permissions in `/var/lib/tenangdb`

3. **Database connection failed**
   - Verify database host and credentials
   - Check network connectivity between containers

4. **Cloud upload failed**
   - Verify rclone configuration
   - Check cloud storage credentials

### Debug Commands

```bash
# Check volume mounts
docker-compose exec tenangdb ls -la /tmp/tenangdb

# Test database connection
docker-compose exec tenangdb /app/tenangdb backup --dry-run

# Check metrics endpoint
curl http://localhost:9090/metrics

# Verify configuration
docker-compose exec tenangdb /app/tenangdb config show
```

## 📚 Best Practices

1. **Always mount persistent volumes** for tracking and metrics
2. **Use dedicated backup user** with minimal database permissions
3. **Implement proper log rotation** to prevent disk space issues
4. **Monitor backup success rates** with alerting
5. **Test restore procedures** regularly
6. **Secure sensitive configuration** with proper secret management
7. **Use resource limits** to prevent resource exhaustion

## 🔄 Backup Retention

Configure retention policies:

1. **Local retention**: Limited by PVC/volume size
2. **Cloud retention**: Configure lifecycle policies on cloud storage
3. **Metrics retention**: Configure in Prometheus settings

## 🚨 Alerting

Set up alerts for:
- Backup failures
- High backup duration
- Low disk space
- Exporter service down
- Database connection issues

Example Prometheus alert rules available in `k8s/` directory.