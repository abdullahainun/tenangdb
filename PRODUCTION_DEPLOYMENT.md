# TenangDB Production Deployment Guide

## 🚀 Quick Production Deploy (Recommended)

**🎯 One-Command Deploy:**
```bash
# Install + Setup + Deploy in one command
curl -sSL https://go.ainun.cloud/tenangdb-install.sh | bash
sudo tenangdb init --deploy-systemd

# Verify deployment
sudo systemctl status tenangdb.timer tenangdb-exporter.service
curl http://localhost:8080/metrics
```

**✨ This automatically:**
- Creates `tenangdb` system user with proper permissions
- Sets up directories: `/etc/tenangdb`, `/var/log/tenangdb`, `/var/backups/tenangdb`
- Installs binaries to `/opt/tenangdb/`
- Generates systemd service files with security hardening
- Enables daily backups + weekend cleanup + metrics server
- Tests database connection before deployment

---

## 🎛️ Advanced Deployment Options

### Custom Systemd User
```bash
# Deploy with custom user
sudo tenangdb init --deploy-systemd --systemd-user mybackup-user
```

### Multiple Environments
```bash
# Production environment
sudo tenangdb init --deploy-systemd --config /etc/tenangdb/prod.yaml

# Staging environment  
sudo tenangdb init --deploy-systemd --config /etc/tenangdb/staging.yaml \
  --systemd-user tenangdb-staging
```

### Manual Systemd Deploy (Legacy)

**Systemd Service File** (`/etc/systemd/system/tenangdb.service`):

```ini
[Unit]
Description=TenangDB MySQL Backup Service
After=network.target mysql.service
Wants=mysql.service

[Service]
Type=oneshot
User=tenangdb
Group=tenangdb
ExecStart=/usr/local/bin/tenangdb backup
StandardOutput=journal
StandardError=journal
SyslogIdentifier=tenangdb

# Security settings
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/var/backups/tenangdb /var/log/tenangdb

[Install]
WantedBy=multi-user.target
```

**Timer for Scheduled Backups** (`/etc/systemd/system/tenangdb.timer`):

```ini
[Unit]
Description=Run TenangDB backup daily
Requires=tenangdb.service

[Timer]
OnCalendar=daily
Persistent=true
RandomizedDelaySec=30m

[Install]
WantedBy=timers.target
```

### 3. Metrics Exporter Service (Optional)

**Systemd Service File for Exporter** (`/etc/systemd/system/tenangdb-exporter.service`):

```ini
[Unit]
Description=TenangDB Exporter

[Service]
User=tenangdb
Group=tenangdb
ExecStart=/usr/local/bin/tenangdb_exporter

# Security settings
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/var/lib/tenangdb /var/log/tenangdb

[Install]
WantedBy=multi-user.target
```

**Enable and Start Exporter:**

```bash
sudo systemctl daemon-reload
sudo systemctl enable tenangdb-exporter.timer
sudo systemctl start tenangdb-exporter.timer
sudo systemctl status tenangdb-exporter.timer
```

**Enable and Start Backup Service:**

```bash
sudo systemctl daemon-reload
sudo systemctl enable tenangdb.timer
sudo systemctl start tenangdb.timer
sudo systemctl status tenangdb.timer
```

### 2. User Service (Development/Testing)

**Run as regular user:**

```bash
# Download and setup config
mkdir -p ~/.config/tenangdb
curl -L https://raw.githubusercontent.com/abdullahainun/tenangdb/main/config.yaml.example -o ~/.config/tenangdb/config.yaml

# Run
tenangdb backup
```

## 📁 Production Directory Structure

### System Service (Root/Dedicated User)

**Linux:**
```
/etc/tenangdb/
├── config.yaml              # Main config
├── my_backup.cnf            # MySQL config for backup
└── my_restore.cnf           # MySQL config for restore

/var/log/tenangdb/
└── tenangdb.log             # Application logs

/var/backups/tenangdb/         # Backup files
├── database1/
└── database2/

/usr/local/bin/
└── tenangdb                 # Binary
```

**macOS (Homebrew):**
```
/usr/local/etc/tenangdb/
├── config.yaml              # Main config
├── my_backup.cnf            # MySQL config for backup
└── my_restore.cnf           # MySQL config for restore

/usr/local/var/log/tenangdb/
└── tenangdb.log             # Application logs

/usr/local/var/tenangdb/
└── backups/                 # Backup files
    ├── database1/
    └── database2/

/usr/local/bin/
└── tenangdb                 # Binary
```

### User Service

**Linux:**
```
~/.config/tenangdb/
└── config.yaml              # Main config

~/.local/share/tenangdb/
├── logs/
│   └── tenangdb.log         # Application logs
└── backups/                 # Backup files
    ├── database1/
    └── database2/
```

**macOS:**
```
~/Library/Application Support/TenangDB/
├── config.yaml              # Main config
└── backups/                 # Backup files
    ├── database1/
    └── database2/

~/Library/Logs/TenangDB/
└── tenangdb.log             # Application logs
```

## 🔧 Configuration Examples

### System Service Config (`/etc/tenangdb/config.yaml`)

```yaml
database:
  host: localhost
  port: 3306
  username: backup_user
  password: "secure_password"
  timeout: 30
  mydumper:
    enabled: true
    binary_path: /usr/bin/mydumper
    defaults_file: /etc/tenangdb/my_backup.cnf
    threads: 4

backup:
  directory: /var/backups/tenangdb
  databases:
    - production_db
    - analytics_db
  batch_size: 2
  concurrency: 1
  retry_count: 3

upload:
  enabled: true
  rclone_path: /usr/bin/rclone
  rclone_config_path: /etc/tenangdb/rclone.conf
  destination: "s3:my-bucket/database-backups/"
  timeout: 1800

logging:
  level: info
  format: json
  file_path: /var/log/tenangdb/tenangdb.log

cleanup:
  enabled: true
  age_based_cleanup: true
  max_age_days: 7
  remote_retention_days: 30
```

## 🔒 Security Considerations

1. **Dedicated User**: Run as non-root `tenangdb` user
2. **File Permissions**: Config files should be 600 (read/write owner only)
3. **MySQL Credentials**: Use dedicated backup user with minimal privileges
4. **Systemd Security**: Use security directives in service file

## 📊 Monitoring

**Check Service Status:**
```bash
# Backup service
sudo systemctl status tenangdb.service
sudo journalctl -u tenangdb.service -f

# Exporter service (if enabled)
sudo systemctl status tenangdb-exporter.service
sudo journalctl -u tenangdb-exporter.service -f
```

**Log Monitoring:**
```bash
tail -f /var/log/tenangdb/tenangdb.log
```

## 🔄 Backup Rotation

TenangDB handles backup rotation automatically based on:
- Local cleanup after successful upload
- Age-based cleanup for old files
- Remote retention policies

## 🚨 Troubleshooting

**Common Issues:**
1. **Permission Denied**: Check file/directory ownership
2. **MySQL Connection**: Verify credentials and network access
3. **Disk Space**: Monitor backup directory space usage
4. **Upload Failures**: Check rclone configuration and network connectivity

**Debug Mode:**
```bash
tenangdb backup --log-level debug
```