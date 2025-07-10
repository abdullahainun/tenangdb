# TenangDB Enhanced Deployment Guide

## 🚀 New Architecture

TenangDB sekarang menggunakan arsitektur terpisah untuk backup job dan metrics monitoring:

- **Backup Job**: `tenangdb` - menjalankan backup sekali lalu exit (compatible dengan timer)
- **Metrics Exporter**: `tenangdb exporter` - daemon yang jalan terus untuk expose metrics ke Prometheus

## 📊 Deployment Steps

### 1. Install Binary

```bash
# Download binary release
curl -L https://github.com/abdullahainun/tenangdb/releases/latest/download/tenangdb-linux-amd64 -o tenangdb

# Install to system
sudo mkdir -p /opt/tenangdb
sudo cp tenangdb /opt/tenangdb/tenangdb
sudo chmod +x /opt/tenangdb/tenangdb
```

### 2. Setup Metrics Storage Directory

```bash
sudo mkdir -p /var/lib/tenangdb
sudo chown backup:backup /var/lib/tenangdb
```

### 3. Update Configuration

Add metrics storage path to your config:

```yaml
# config.yaml
metrics:
  enabled: true
  port: "9090"
  storage_path: "/var/lib/tenangdb/metrics.json"
```

### 4. Create Systemd Services

#### Metrics Exporter Service (Daemon)

```bash
sudo tee /etc/systemd/system/tenangdb-exporter.service << 'EOF'
[Unit]
Description=TenangDB Metrics Exporter
After=network.target

[Service]
Type=simple
User=backup
Group=backup
ExecStart=/opt/tenangdb/tenangdb exporter --config /etc/tenangdb/config.yaml --port 9090
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
```

#### Backup Timer (Job)

```bash
# tenangdb.service (existing - no changes needed)
sudo systemctl reload-daemon
```

### 5. Start Services

```bash
# Start metrics exporter daemon
sudo systemctl enable tenangdb-exporter.service
sudo systemctl start tenangdb-exporter.service

# Verify status
sudo systemctl status tenangdb-exporter.service

# Enable backup timer (existing)
sudo systemctl enable tenangdb.timer
sudo systemctl start tenangdb.timer
```

### 6. Import Enhanced Grafana Dashboard

1. Open Grafana Web UI
2. Go to Dashboards → Import
3. Upload `grafana/db-backup-dashboard.json`
4. Configure Prometheus datasource (uid: `prometheus`)

## 🔧 Usage Commands

### Backup Operations

```bash
# Manual backup (exits after completion)
tenangdb backup --config /etc/tenangdb/config.yaml

# Cleanup operations  
tenangdb cleanup --config /etc/tenangdb/config.yaml --force

# Restore database
tenangdb restore --config /etc/tenangdb/config.yaml \
  --backup-path /backup/mydb/2025-01/mydb-2025-01-06_10-30-00 \
  --database mydb_restore
```

### Metrics Operations

```bash
# Start metrics exporter (daemon)
tenangdb exporter --config /etc/tenangdb/config.yaml --port 9090

# Test metrics endpoint
curl http://localhost:9090/metrics

# Health check
curl http://localhost:9090/health
```

## 📈 Dashboard Features

### Enhanced Dashboard Sections:

1. **📊 System Overview**
   - System health status
   - Backup process status  
   - Key statistics summary

2. **💾 Backup Operations**
   - Success/failure rates
   - Duration performance
   - Backup sizes
   - Last backup times

3. **☁️ Upload Operations**
   - Upload success/failure rates
   - Upload duration & throughput
   - Upload bandwidth usage

4. **🔄 Restore Operations**
   - Restore success/failure rates
   - Restore duration performance
   - Restore operation tracking

5. **🧹 Cleanup Operations**
   - Files removed & bytes freed
   - Cleanup success/failure rates
   - Cleanup operation metrics

6. **🚨 Alerts & SLA**
   - 24h SLA compliance (>99% target)
   - Time since last backup alerts
   - Failed operations alerts

### Available Metrics:

```
# Backup metrics
tenangdb_backup_duration_seconds{database="mydb"}
tenangdb_backup_success_total{database="mydb"}
tenangdb_backup_failed_total{database="mydb"}
tenangdb_backup_size_bytes{database="mydb"}
tenangdb_backup_last_timestamp{database="mydb"}

# Upload metrics  
tenangdb_upload_duration_seconds{database="mydb"}
tenangdb_upload_success_total{database="mydb"}
tenangdb_upload_failed_total{database="mydb"}
tenangdb_upload_bytes_total{database="mydb"}
tenangdb_upload_last_timestamp{database="mydb"}

# Restore metrics
tenangdb_restore_duration_seconds{database="mydb"}
tenangdb_restore_success_total{database="mydb"}
tenangdb_restore_failed_total{database="mydb"}
tenangdb_restore_last_timestamp{database="mydb"}

# Cleanup metrics
tenangdb_cleanup_duration_seconds
tenangdb_cleanup_success_total
tenangdb_cleanup_failed_total
tenangdb_cleanup_files_removed_total
tenangdb_cleanup_bytes_freed_total
tenangdb_cleanup_last_timestamp

# System metrics
tenangdb_total_databases
tenangdb_backup_process_active
tenangdb_system_health
tenangdb_last_process_timestamp
```

## 🔍 Monitoring & Alerting

### Recommended Alerts:

1. **Backup Failure Alert**
   ```
   increase(tenangdb_backup_failed_total[1h]) > 0
   ```

2. **Backup Missed Alert** 
   ```
   time() - tenangdb_backup_last_timestamp > 86400
   ```

3. **SLA Breach Alert**
   ```
   (sum(rate(tenangdb_backup_success_total[24h])) / (sum(rate(tenangdb_backup_success_total[24h])) + sum(rate(tenangdb_backup_failed_total[24h])))) * 100 < 99
   ```

## 🧪 Testing

```bash
# Test backup with metrics
tenangdb backup --config /etc/tenangdb/config.yaml

# Verify metrics file created
cat /var/lib/tenangdb/metrics.json

# Test metrics endpoint
curl http://localhost:9090/metrics | grep tenangdb

# Test restore with metrics
tenangdb restore --backup-path /path/to/backup --database test_restore

# Test cleanup with metrics
tenangdb cleanup --config /etc/tenangdb/config.yaml --force
```

## 🚨 Troubleshooting

### Metrics Not Appearing:

1. Check metrics file exists: `ls -la /var/lib/tenangdb/`
2. Check exporter logs: `journalctl -u tenangdb-exporter.service -f`
3. Check metrics endpoint: `curl localhost:9090/metrics`
4. Verify permissions: `sudo chown backup:backup /var/lib/tenangdb/metrics.json`

### Dashboard Not Loading:

1. Verify Prometheus datasource configured
2. Check metrics endpoint accessible from Prometheus
3. Verify dashboard import successful
4. Check for metric name mismatches

## 🎯 Migration from Old Setup

If migrating from old single-process setup:

1. Stop old prometheus metrics in main process
2. Update config to disable embedded metrics
3. Start new exporter daemon
4. Import new dashboard
5. Update alerting rules to use new metrics

This maintains backward compatibility while providing enhanced monitoring! 📊