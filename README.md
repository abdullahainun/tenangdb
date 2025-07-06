# TenangDB

A robust MySQL backup tool with mydumper/mysqldump support, cloud uploads, and comprehensive management features.

🔗 **Repository:** https://github.com/abdullahainun/tenangdb

## Features

- **Dual Backup Engine**: mydumper (parallel) + mysqldump (traditional)
- **Backup Management**: Automated backup, restore, and cleanup
- **Cloud Storage**: Upload to S3, Minio, or any rclone-supported storage
- **Mydumper Integration**: Defaults-file support (~/.my.cnf, ~/.my_restore.cnf)
- **Batch Processing**: Handle multiple databases efficiently
- **Structured Logging**: JSON logs with detailed statistics
- **Systemd Ready**: Service and timer integration

## Quick Start

### 1. Install Dependencies
```bash
# Test all required dependencies
make test-deps

# Install missing dependencies (Ubuntu/Debian)
sudo apt update && sudo apt install mydumper mysql-client
curl https://rclone.org/install.sh | sudo bash
```

### 2. Build & Install
```bash
# Clone and build
git clone https://github.com/abdullahainun/tenangdb.git
cd tenangdb
make build

# Install system-wide (optional)
sudo make install
```

### 3. Configure
```bash
# Copy example config
cp configs/config.yaml /etc/tenangdb/config.yaml

# Edit configuration
nano /etc/tenangdb/config.yaml
# Update database credentials, backup directories, etc.
```

### 4. Usage Examples
```bash
# Test configuration
./tenangdb backup --config configs/config.yaml --dry-run

# Run backup
./tenangdb backup --config configs/config.yaml

# Selective cleanup
./tenangdb cleanup --databases myapp,logs --force

# Restore database
./tenangdb restore --backup-path /backup/myapp-2025-07-05_10-30-15 --target-database myapp_restored
```

📖 **For detailed installation guide, see [INSTALL.md](INSTALL.md)**

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
  rclone_config_path: /etc/tenangdb/rclone.conf
  destination: "minio:bucket/backups/"
```

## Commands

```bash
# Backup all configured databases
./tenangdb backup --config config.yaml

# Backup with specific log level
./tenangdb backup --config config.yaml --log-level debug

# Restore from backup
./tenangdb restore --backup-path /backup/db-2025-07-04_01-06-02 --target-database restored_db

# Cleanup uploaded files
./tenangdb cleanup --config config.yaml

# Force cleanup anytime (bypass weekend-only)
./tenangdb cleanup --force --config config.yaml

# Cleanup specific databases only
./tenangdb cleanup --databases app_db,logs_db --force

# Preview cleanup (no deletion)
./tenangdb cleanup --dry-run --force

# Age-based cleanup with verification
./tenangdb cleanup --force --config config.yaml  # Uses age_based_cleanup from config
```

### Log Levels
Available log levels: `panic`, `fatal`, `error`, `warn`, `info` (default), `debug`, `trace`

```bash
# Silent mode (errors only)
./tenangdb backup --log-level error

# Verbose debugging
./tenangdb backup --log-level trace
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
sudo systemctl enable tenangdb.timer
sudo systemctl start tenangdb.timer
```

## Security

- Uses dedicated backup users with minimal privileges
- Supports defaults-file for credential management
- No passwords in config files or logs
- Systemd hardening enabled

## License

MIT License - See LICENSE file for details