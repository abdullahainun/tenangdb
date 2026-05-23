# TenangDB Kubernetes Deployment

This directory contains Kubernetes manifests for deploying TenangDB with automatic database backup functionality in a production Kubernetes cluster.

## Architecture

TenangDB deployment consists of:

1. **`tenangdb`** - Main backup binary (runs as CronJob for scheduled backups)
2. **`tenangdb-exporter`** - Metrics exporter (runs as Deployment for monitoring)

## 📋 Prerequisites

- Kubernetes cluster (1.20+)
- `kubectl` configured to access your cluster
- MySQL/MariaDB/PostgreSQL service accessible from cluster
- Local storage class or persistent storage available

## 🚀 Quick Start

### 1. Deploy Using Kustomize (Recommended)

```bash
# Deploy all manifests
kubectl apply -k .

# Check deployment status
kubectl get all -n tenangdb
```

### 2. Configure Database Connection

Edit `configmap.yaml` to update your database settings:

```yaml
database:
  host: 192.168.1.100  # Your database server IP/hostname
  port: 3306            # 5432 for PostgreSQL
  username: tenangdb
  password: "secure_password"
```

Then apply the updated config:

```bash
kubectl apply -f configmap.yaml
```

### 3. Verify Deployment

```bash
# Check all resources
kubectl get all,pvc -n tenangdb

# Check CronJob schedule
kubectl get cronjobs -n tenangdb

# Check metrics exporter
kubectl logs -n tenangdb deployment/tenangdb-metrics

# Test manual backup
kubectl create job --from=cronjob/tenangdb-backup tenangdb-test -n tenangdb
kubectl logs -n tenangdb job/tenangdb-test
```

## 📁 File Structure

```
k8s/
├── namespace.yaml                      # TenangDB namespace
├── configmap.yaml                      # TenangDB configuration
├── pv.yaml                            # Persistent volume (20Gi local storage)
├── pvc.yaml                           # Persistent volume claim
├── rbac.yaml                          # ServiceAccount and permissions
├── cronjob.yaml                       # Scheduled backup job (daily 2 AM)
├── metrics-deployment.yaml            # Metrics exporter + service + ServiceMonitor
├── kustomization.yaml                 # Kustomize configuration
├── backup-explorer.yaml.disabled      # Pod for accessing backup files
├── fix-permissions-job.yaml.disabled  # Emergency permissions fix for Talos OS
├── secret.yaml.disabled               # Not used (env vars removed)
└── README.md                          # This file
```

## ⚙️ Configuration

### Single PVC Storage Structure

All data is stored in a single 20Gi PVC with this structure:

```
/data/
├── backups/        # Database backup files (organized by date)
├── metrics/        # Metrics and tracking data
└── logs/           # Application logs
```

### Database Configuration

Update `configmap.yaml` with your database settings:

```yaml
database:
  host: your-database-server
  port: 3306            # 5432 for PostgreSQL
  username: backup-user
  password: "your-password"
  timeout: 30
```

### Backup Configuration

Configure which databases to backup:

```yaml
backup:
  directory: /data/backups
  databases:
    - your_database_1
    - your_database_2
  batch_size: 5
  concurrency: 3
```

### Schedule Configuration

Default: Daily at 2 AM Jakarta time. Modify in `cronjob.yaml`:

```yaml
spec:
  schedule: "0 2 * * *"        # Daily at 2 AM
  timeZone: "Asia/Jakarta"
  # schedule: "0 */6 * * *"    # Every 6 hours
  # schedule: "0 2 * * 0"      # Weekly on Sunday
```

## 🔧 Operations

### Manual Backup

```bash
# Create manual backup job
kubectl create job --from=cronjob/tenangdb-backup tenangdb-manual-$(date +%Y%m%d-%H%M%S) -n tenangdb

# Check job status
kubectl get jobs -n tenangdb

# View logs
kubectl logs -n tenangdb job/tenangdb-manual-<timestamp>
```

### Access Backup Files

Use the backup explorer pod:

```bash
# Enable backup explorer
mv backup-explorer.yaml.disabled backup-explorer.yaml
kubectl apply -f backup-explorer.yaml

# Browse backup files
kubectl exec -it backup-explorer -n tenangdb -- ls -la /data/backups

# Copy files to local
kubectl cp tenangdb/backup-explorer:/data/backups/database/2025-07 ./local-backup

# Cleanup
kubectl delete -f backup-explorer.yaml
```

### View Logs

```bash
# Check recent backup jobs
kubectl get jobs -n tenangdb --sort-by=.metadata.creationTimestamp

# View latest backup logs
kubectl logs -n tenangdb -l job-name=$(kubectl get jobs -n tenangdb --sort-by=.metadata.creationTimestamp -o name | tail -1 | cut -d/ -f2)

# Check CronJob history
kubectl get cronjobs tenangdb-backup -n tenangdb
```

## 📊 Monitoring

### Metrics Endpoint

The metrics exporter provides Prometheus metrics:

```bash
# Port forward to access metrics
kubectl port-forward -n tenangdb service/svc-tenangdb-metrics 9090:9090

# Check metrics
curl http://localhost:9090/metrics
curl http://localhost:9090/health
```

### Backup Status

```bash
# Check CronJob status
kubectl describe cronjob tenangdb-backup -n tenangdb

# Check recent jobs
kubectl get jobs -n tenangdb

# Check failed jobs
kubectl get jobs -n tenangdb --field-selector=status.failed=1
```

## 🛡️ Security

### Database User Permissions

Create a dedicated backup user with minimal permissions:

MySQL:
```sql
-- Create backup user
CREATE USER 'tenangdb'@'%' IDENTIFIED BY 'secure_password';

-- Grant minimal required permissions for mydumper
GRANT SELECT, RELOAD, LOCK TABLES, REPLICATION CLIENT ON *.* TO 'tenangdb'@'%';
GRANT SHOW VIEW ON *.* TO 'tenangdb'@'%';

-- For specific databases only
-- GRANT SELECT, LOCK TABLES ON database1.* TO 'tenangdb'@'%';
-- GRANT SELECT, LOCK TABLES ON database2.* TO 'tenangdb'@'%';

FLUSH PRIVILEGES;
```

PostgreSQL:
```sql
CREATE ROLE tenangdb WITH LOGIN PASSWORD 'secure_password';
GRANT pg_read_all_data TO tenangdb;
```

### Pod Security

- Runs as non-root user (1001:1001)
- Uses read-only config mounts
- Minimal resource limits
- Service account with restricted permissions

## 🚨 Troubleshooting

### Permission Issues (Talos OS)

If you encounter permission denied errors:

```bash
# Use the emergency permissions fix
mv fix-permissions-job.yaml.disabled fix-permissions-job.yaml
kubectl apply -f fix-permissions-job.yaml

# Wait for completion
kubectl get job fix-tenangdb-permissions -n tenangdb-privileged

# Cleanup
kubectl delete -f fix-permissions-job.yaml
```

### Common Issues

1. **PVC Pending**
   ```bash
   # Check PV status
   kubectl get pv | grep tenangdb
   
   # Check storage class
   kubectl get storageclass
   
   # Check node affinity
   kubectl describe pv pv-tenangdb-data
   ```

2. **Database Connection Failed**
   ```bash
   # Test connectivity from cluster
   kubectl run db-test --image=postgres:15 --rm -it --restart=Never -n tenangdb -- \
     psql -h 192.168.43.117 -U tenangdb -d postgres

   # Or for MySQL:
   kubectl run mysql-test --image=mysql:8.0 --rm -it --restart=Never -n tenangdb -- \
     mysql -h 192.168.43.117 -u tenangdb -p
   ```

3. **Backup Job Fails**
   ```bash
   # Check job logs
   kubectl logs -n tenangdb job/tenangdb-backup-<timestamp>
   
   # Check config
   kubectl get configmap tenangdb-config -n tenangdb -o yaml
   ```

4. **Metrics Pod Not Ready**
   ```bash
   # Check pod status
   kubectl describe pod -n tenangdb -l component=metrics
   
   # Check health endpoint
   kubectl exec -n tenangdb deployment/tenangdb-metrics -- wget -O- http://localhost:9090/health
   ```

## 🔄 Storage Management

### Backup Retention

- **Kubernetes Jobs**: CronJob keeps last 3 successful jobs
- **Local Storage**: 20Gi PVC provides space for multiple backup cycles
- **File Organization**: Backups organized by database and date for easy cleanup

### Storage Expansion

To increase storage capacity:

1. Update PV size in `pv.yaml`
2. Update PVC size in `pvc.yaml`
3. Apply changes and restart pods

## 📚 Additional Resources

- [TenangDB GitHub Repository](https://github.com/abdullahainun/tenangdb)
- [Kubernetes CronJob Documentation](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/)
- [Kubernetes Persistent Volumes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)
- [mydumper Documentation](https://mydumper.readthedocs.io/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/current/)

## 🚀 Production Considerations

1. **Resource Limits**: Adjust CPU/memory limits based on database size
2. **Network Policies**: Implement network policies for security
3. **Monitoring**: Set up alerts for backup failures
4. **Backup Validation**: Regularly test restore procedures
5. **Disaster Recovery**: Consider off-site backup strategies
