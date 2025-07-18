# TenangDB Kubernetes Deployment

This directory contains Kubernetes manifests for deploying TenangDB in a production Kubernetes cluster.

## 📋 Prerequisites

- Kubernetes cluster (1.20+)
- `kubectl` configured to access your cluster
- MySQL/MariaDB service running in cluster or accessible from cluster
- Cloud storage configured (S3, GCS, etc.) - optional but recommended

## 🚀 Quick Start

### 1. Create Namespace and Apply Manifests

```bash
# Apply all manifests in order
kubectl apply -f namespace.yaml
kubectl apply -f secret.yaml
kubectl apply -f configmap.yaml
kubectl apply -f pvc.yaml
kubectl apply -f rbac.yaml
kubectl apply -f cronjob.yaml
```

### 2. Configure Secrets

Edit the secret with your actual database credentials:

```bash
# Create secret with your credentials
kubectl create secret generic tenangdb-secrets \
  --from-literal=MYSQL_USER=your-backup-user \
  --from-literal=MYSQL_PASSWORD=your-secure-password \
  --namespace=tenangdb
```

### 3. Configure Database Connection

Edit `configmap.yaml` to update:
- `database.host`: Your MySQL service endpoint
- `backup.databases`: List of databases to backup
- `upload.destination`: Your cloud storage destination

```bash
# Update configmap
kubectl apply -f configmap.yaml
```

### 4. Verify Deployment

```bash
# Check if CronJob is created
kubectl get cronjobs -n tenangdb

# Check PVCs
kubectl get pvc -n tenangdb

# Check upcoming job schedule
kubectl get cronjobs tenangdb-backup -n tenangdb -o yaml
```

## 📁 File Structure

```
k8s/
├── namespace.yaml      # TenangDB namespace
├── secret.yaml         # Database credentials and cloud config
├── configmap.yaml      # TenangDB configuration
├── pvc.yaml           # Persistent volumes for backups/tracking/logs
├── rbac.yaml          # ServiceAccount and permissions
├── cronjob.yaml       # Scheduled backup job + manual job template
└── README.md          # This file
```

## ⚙️ Configuration

### Database Configuration

Update `configmap.yaml` with your database settings:

```yaml
database:
  host: mysql-service.default.svc.cluster.local
  port: 3306
  # credentials come from secret
```

### Backup Configuration

Configure which databases to backup:

```yaml
backup:
  databases:
    - production_db
    - analytics_db
    - user_data
```

### Cloud Upload Configuration

Configure cloud storage destination:

```yaml
upload:
  enabled: true
  destination: "s3:your-backup-bucket/tenangdb"
  # or "gcs:your-bucket/tenangdb"
```

### Schedule Configuration

Default schedule is daily at 2 AM. Modify in `cronjob.yaml`:

```yaml
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  # schedule: "0 */6 * * *"  # Every 6 hours
  # schedule: "0 2 * * 0"    # Weekly on Sunday at 2 AM
```

## 🔧 Operations

### Manual Backup

Run a manual backup job:

```bash
# Create manual backup job
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: tenangdb-manual-$(date +%Y%m%d-%H%M%S)
  namespace: tenangdb
spec:
  template:
    spec:
      serviceAccountName: tenangdb
      restartPolicy: Never
      containers:
      - name: tenangdb
        image: ghcr.io/abdullahainun/tenangdb:latest
        command: ["/app/tenangdb"]
        args: ["backup", "--force"]
        envFrom:
        - secretRef:
            name: tenangdb-secrets
        volumeMounts:
        - name: config
          mountPath: /config.yaml
          subPath: config.yaml
        - name: backups
          mountPath: /backups
        - name: tracking
          mountPath: /tmp/tenangdb
      volumes:
      - name: config
        configMap:
          name: tenangdb-config
      - name: backups
        persistentVolumeClaim:
          claimName: tenangdb-backups
      - name: tracking
        persistentVolumeClaim:
          claimName: tenangdb-tracking
EOF
```

### View Logs

```bash
# Check recent backup jobs
kubectl get jobs -n tenangdb

# View logs from latest backup
kubectl logs -n tenangdb -l job-name=tenangdb-backup-<timestamp>

# View logs from manual backup
kubectl logs -n tenangdb -l job-name=tenangdb-manual-<timestamp>
```

### Restore Operations

```bash
# Run restore job
kubectl run tenangdb-restore \
  --image=ghcr.io/abdullahainun/tenangdb:latest \
  --namespace=tenangdb \
  --rm -it --restart=Never \
  --overrides='{
    "spec": {
      "serviceAccountName": "tenangdb",
      "containers": [{
        "name": "tenangdb-restore",
        "image": "ghcr.io/abdullahainun/tenangdb:latest",
        "command": ["/app/tenangdb", "restore", "--interactive"],
        "envFrom": [{"secretRef": {"name": "tenangdb-secrets"}}],
        "volumeMounts": [
          {"name": "config", "mountPath": "/config.yaml", "subPath": "config.yaml"},
          {"name": "backups", "mountPath": "/backups"}
        ]
      }],
      "volumes": [
        {"name": "config", "configMap": {"name": "tenangdb-config"}},
        {"name": "backups", "persistentVolumeClaim": {"claimName": "tenangdb-backups"}}
      ]
    }
  }'
```

## 📊 Monitoring

### Check Backup Status

```bash
# Check CronJob status
kubectl get cronjobs -n tenangdb

# Check recent jobs
kubectl get jobs -n tenangdb --sort-by=.metadata.creationTimestamp

# Check failed jobs
kubectl get jobs -n tenangdb --field-selector=status.failed=1
```

### View Backup History

```bash
# Check persistent logs
kubectl exec -n tenangdb -it deployment/log-viewer -- cat /var/log/tenangdb/tenangdb.log

# Or mount the log PVC to view logs
kubectl run log-viewer \
  --image=alpine:latest \
  --namespace=tenangdb \
  --rm -it --restart=Never \
  --overrides='{
    "spec": {
      "containers": [{
        "name": "log-viewer",
        "image": "alpine:latest",
        "command": ["sh"],
        "volumeMounts": [{"name": "logs", "mountPath": "/logs"}]
      }],
      "volumes": [{"name": "logs", "persistentVolumeClaim": {"claimName": "tenangdb-logs"}}]
    }
  }'
```

## 🛡️ Security

### Database User Permissions

Create a dedicated backup user with minimal permissions:

```sql
-- Create backup user
CREATE USER 'backup-user'@'%' IDENTIFIED BY 'secure-password';

-- Grant minimal required permissions
GRANT SELECT, LOCK TABLES, SHOW VIEW, EVENT, TRIGGER ON *.* TO 'backup-user'@'%';
GRANT REPLICATION CLIENT ON *.* TO 'backup-user'@'%';

-- Apply privileges
FLUSH PRIVILEGES;
```

### Secret Management

- Store credentials in Kubernetes secrets
- Use RBAC to limit access to secrets
- Consider using external secret management (Vault, AWS Secrets Manager)

### Network Security

- Use NetworkPolicies to restrict access
- Ensure database connections use TLS
- Store cloud credentials securely

## 🔄 Backup Retention

Backup retention is handled by:
1. **Kubernetes**: CronJob keeps last 3 successful jobs
2. **Cloud Storage**: Configure lifecycle policies on your cloud storage
3. **Local Storage**: PVC size limits local retention

## 🚨 Troubleshooting

### Common Issues

1. **Permission Denied**
   ```bash
   # Check service account permissions
   kubectl auth can-i --list --as=system:serviceaccount:tenangdb:tenangdb -n tenangdb
   ```

2. **Database Connection Failed**
   ```bash
   # Test database connectivity
   kubectl run mysql-test --image=mysql:8.0 --rm -it --restart=Never -- \
     mysql -h mysql-service.default.svc.cluster.local -u backup-user -p
   ```

3. **PVC Issues**
   ```bash
   # Check PVC status
   kubectl get pvc -n tenangdb
   
   # Check storage class
   kubectl get storageclass
   ```

4. **Cloud Upload Failed**
   ```bash
   # Check rclone config
   kubectl get secret tenangdb-secrets -n tenangdb -o yaml
   ```

### Debug Mode

Enable debug logging by updating the ConfigMap:

```yaml
logging:
  level: debug
```

## 📚 Additional Resources

- [TenangDB Documentation](https://github.com/abdullahainun/tenangdb)
- [Kubernetes CronJob Documentation](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/)
- [Kubernetes Persistent Volumes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)