# ⚙️ Configuration Guide

Complete configuration options for TenangDB.

## 📋 Basic Configuration

```yaml
database:
  mydumper:
    enabled: true
    defaults_file: ~/.my.cnf
    threads: 4
    compress_method: gzip
    myloader:
      enabled: true
      defaults_file: ~/.my_restore.cnf

backup:
  directory: /backup
  databases: [db1, db2]
  
upload:
  enabled: true
  rclone_config_path: /etc/tenangdb/rclone.conf
  destination: "minio:bucket/backups/"

logging:
  level: info
  format: json
  file_path: /var/log/tenangdb/tenangdb.log
```

## 🔧 Database Configuration

### MySQL Connection
```yaml
database:
  host: 192.168.1.100
  port: 3306
  username: backup_user
  password: "secure_password"
  timeout: 30
```

### Mydumper Settings
```yaml
database:
  mydumper:
    enabled: true
    defaults_file: ~/.my.cnf      # MySQL credentials file
    threads: 4                    # Parallel threads
    compress_method: gzip         # Compression: gzip, zstd, lz4
    chunk_filesize: 64            # Split tables into chunks (MB)
    single_transaction: true      # Consistent backup
    routines: true               # Include stored procedures
    triggers: true               # Include triggers
    events: true                 # Include events
```

## 📤 Cloud Upload Configuration

### Rclone Integration
```yaml
upload:
  enabled: true
  rclone_config_path: /etc/tenangdb/rclone.conf
  destination: "remote:bucket/backups/"
  verify_upload: true
  delete_after_upload: false
```

### Supported Cloud Providers
- AWS S3
- Google Cloud Storage
- Azure Blob Storage
- Minio
- Dropbox
- OneDrive
- And 40+ other providers via rclone

## 🧹 Cleanup Configuration

### Age-based Cleanup
```yaml
cleanup:
  enabled: true
  age_based_cleanup: true
  max_age_days: 7
  verify_cloud_exists: true
  weekend_only: true
  cleanup_uploaded_files: true
```

### Database-specific Cleanup
```yaml
cleanup:
  database_specific:
    production_db:
      max_age_days: 30
      keep_count: 10
    dev_db:
      max_age_days: 3
      keep_count: 5
```

## 📊 Monitoring Configuration

### Metrics Export
```yaml
metrics:
  enabled: true
  listen_addr: ":8080"
  path: "/metrics"
  labels:
    environment: production
    datacenter: us-east-1
```

### Logging Options
```yaml
logging:
  level: info                    # panic, fatal, error, warn, info, debug, trace
  format: json                   # json, text
  file_path: /var/log/tenangdb/tenangdb.log
  max_size: 100                  # MB
  max_backups: 3
  max_age: 28                    # days
  compress: true
```

## 🔐 Security Configuration

### Credential Management
```yaml
# Use defaults-file instead of plain text passwords
database:
  mydumper:
    defaults_file: ~/.my.cnf
    
# Example ~/.my.cnf
[client]
host=192.168.1.100
port=3306
user=backup_user
password=secure_password

[mydumper]
single-transaction=true
routines=true
triggers=true
events=true
```

### Systemd Hardening
```yaml
# Automatic when using systemd services
security:
  user: tenangdb
  group: tenangdb
  readonly_paths: ["/etc", "/usr"]
  no_new_privileges: true
  protect_system: strict
```

## 📝 Full Configuration Example

See [config.yaml](config.yaml) for a complete configuration example with all available options.

## 🆘 Troubleshooting

**Config validation failed:**
```bash
# Test configuration
./tenangdb backup --config config.yaml --dry-run
```

**Permission issues:**
```bash
# Check file permissions
ls -la /etc/tenangdb/config.yaml
sudo chown tenangdb:tenangdb /etc/tenangdb/config.yaml
```

**Database connection issues:**
```bash
# Test MySQL connection
mysql -h192.168.1.100 -ubackup_user -p
```