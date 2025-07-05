# Grafana Dashboard untuk Database Backup Tool

Dashboard ini menyediakan monitoring lengkap untuk database backup tool dengan Prometheus metrics.

## Fitur Dashboard

### 1. **Backup Success/Failure Rate**
- Grafik time series yang menampilkan tingkat keberhasilan dan kegagalan backup
- Menampilkan rate per database dengan interval 5 menit

### 2. **Backup Process Status**  
- Gauge yang menunjukkan status proses backup (Running/Stopped)
- Indikator real-time apakah backup sedang berjalan

### 3. **Backup Duration**
- Histogram percentile (50th dan 95th) untuk durasi backup
- Membantu identify performance bottlenecks

### 4. **Backup Size**
- Time series ukuran backup file per database
- Tracking pertumbuhan ukuran data

### 5. **Upload Duration**
- Performance metrics untuk upload ke cloud storage
- Percentile analysis untuk upload time

### 6. **Last Backup Time**
- Tabel yang menampilkan timestamp backup terakhir per database
- Format "time ago" untuk mudah dibaca

### 7. **Statistics Panel**
- Total databases configured
- Total successful backups
- Total failed backups  
- Total successful uploads

## Cara Import Dashboard

### 1. Via Grafana UI
1. Login ke Grafana
2. Pilih "+" > "Import"
3. Upload file `db-backup-dashboard.json`
4. Configure data source (Prometheus)
5. Save dashboard

### 2. Via API
```bash
curl -X POST \
  http://admin:admin@localhost:3000/api/dashboards/db \
  -H 'Content-Type: application/json' \
  -d @db-backup-dashboard.json
```

## Konfigurasi Data Source

Pastikan Prometheus data source sudah dikonfigurasi dengan:
- **Name**: prometheus
- **URL**: http://localhost:9090 (atau sesuai setup Anda)
- **Access**: Server (default)

## Alerting Rules (Opsional)

Untuk setup alerting, tambahkan rules berikut di Prometheus:

```yaml
groups:
  - name: tenangdb_alerts
    rules:
      - alert: BackupFailed
        expr: increase(tenangdb_backup_failed_total[1h]) > 0
        for: 0m
        labels:
          severity: critical
        annotations:
          summary: "Database backup failed"
          description: "Database {{ $labels.database }} backup failed"

      - alert: BackupTooSlow
        expr: histogram_quantile(0.95, rate(tenangdb_backup_duration_seconds_bucket[5m])) > 1800
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Backup taking too long"
          description: "Database {{ $labels.database }} backup taking more than 30 minutes"

      - alert: NoRecentBackup
        expr: time() - tenangdb_backup_last_timestamp > 86400
        for: 0m
        labels:
          severity: warning
        annotations:
          summary: "No recent backup"
          description: "Database {{ $labels.database }} hasn't been backed up in 24 hours"
```

## Refresh Interval

Dashboard di-set dengan refresh otomatis setiap 5 detik untuk monitoring real-time.

## Tags

Dashboard menggunakan tags:
- `database`
- `backup` 
- `monitoring`

Untuk memudahkan pencarian dan kategori.