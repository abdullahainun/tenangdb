# Database Backup Tool

A robust MySQL backup tool with mydumper/mysqldump support, cloud uploads, and comprehensive management features.

## Features

- **Dual Backup Engine**: mydumper (parallel) + mysqldump (traditional)
- **Backup Management**: Automated backup, restore, and cleanup
- **Cloud Storage**: Upload to S3, Minio, or any rclone-supported storage
- **Mydumper Integration**: Defaults-file support (~/.my.cnf, ~/.my_restore.cnf)
- **Batch Processing**: Handle multiple databases efficiently
- **Structured Logging**: JSON logs with detailed statistics
- **Systemd Ready**: Service and timer integration

## Quick Start

```bash
# Install
go build -o db-backup-tool cmd/main.go

# Configure
cp configs/config.yaml my-config.yaml
# Edit database credentials and mydumper settings

# Backup
./db-backup-tool --config my-config.yaml

# Restore
./db-backup-tool restore --backup-path /path/to/backup --database target_db

# Cleanup
./db-backup-tool cleanup --force --dry-run
```

## Configuration

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
  destination: "minio:bucket/backups/"
```

## Commands

```bash
# Backup all configured databases
./db-backup-tool --config config.yaml

# Restore from backup
./db-backup-tool restore -b /backup/db-2025-07-04_01-06-02 -d target_db

# Cleanup uploaded files (weekend-only by default)
./db-backup-tool cleanup

# Force cleanup anytime
./db-backup-tool cleanup --force

# Preview cleanup (no deletion)
./db-backup-tool cleanup --dry-run --force
```

## Storage Integration

### Minio Setup
```bash
# Configure rclone for Minio
rclone config  # Add S3-compatible endpoint

# Test connection
rclone lsd minio:bucket-name
```

### Cloud Storage
Supports any rclone backend: AWS S3, Google Cloud, Azure, Dropbox, etc.

## Production Deployment

```bash
# Install as systemd service
sudo cp scripts/*.service /etc/systemd/system/
sudo systemctl enable db-backup.timer
sudo systemctl start db-backup.timer
```

## Security

- Uses dedicated backup users with minimal privileges
- Supports defaults-file for credential management
- No passwords in config files or logs
- Systemd hardening enabled

## License

MIT License - See LICENSE file for details