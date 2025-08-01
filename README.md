# TenangDB

> 🚧 **Under Development** 
>
> This project is actively being developed. While core features are stable, expect:
> - Potential breaking changes in configuration and CLI
> - Some experimental features may not work as expected  
> - Thorough testing recommended before production use

🛡️ **Backup yang Bikin Tenang** - Secure MySQL backup with intelligent automation.

*Zero-configuration backup system with smart confirmations and cloud integration.*

## 🎬 Live Demo

[![TenangDB Demo](https://asciinema.org/a/728588.svg)](https://asciinema.org/a/728588)

*Note: The "analytics" database backup error in the demo is expected - it shows how TenangDB handles insufficient privileges gracefully while continuing with other databases.*

## ⚡ Quick Start

```bash
# Docker (Recommended)
docker pull ghcr.io/abdullahainun/tenangdb:latest

# 1. Setup
mkdir tenangdb && cd tenangdb
curl -L https://go.ainun.cloud/tenangdb-config.yaml.example -o config.yaml
nano config.yaml  # Edit database credentials

# 2. Run backup
mkdir -p backups
docker run -it --user $(id -u):$(id -g) \
  -v $(pwd)/config.yaml:/config.yaml \
  -v $(pwd)/backups:/backups \
  ghcr.io/abdullahainun/tenangdb:latest backup

# Binary install
curl -sSL https://go.ainun.cloud/tenangdb-install.sh | bash
tenangdb backup

# With metrics monitoring
tenangdb-exporter &  # Start metrics server
tenangdb backup      # Run backup with metrics
```

## ⚙️ Config

```yaml
database:
  host: your_mysql_host
  username: your_username
  password: your_password
  mydumper:
    enabled: true

backup:
  databases:
    - your_database1
    - your_database2
  # Smart features
  check_last_backup_time: true    # Prevent redundant backups
  min_backup_interval: 1h         # Minimum time between backups

logging:
  file_path: /logs/tenangdb.log    # Custom log path

upload:
  enabled: false
  destination: "your-remote:backup-folder"

metrics:
  enabled: true                        # Enable Prometheus metrics
  port: 9090                          # Metrics server port
  storage_path: /var/lib/tenangdb/metrics.json
```

## 🔧 Features

- **Smart Confirmations**: Interactive prompts with backup summary
- **Frequency Checking**: Prevents redundant backups with configurable intervals
- **Auto-Discovery**: Binary paths, directories, and optimal settings
- **Cloud Integration**: Seamless upload to any rclone-supported storage
- **Compression**: tar.gz/zst/xz support with hybrid local/cloud approach
- **Safety First**: Database overwrite warnings and confirmation prompts
- **Metrics & Monitoring**: Prometheus metrics with dedicated exporter binary

## 📋 Commands

```bash
# Backup
tenangdb backup                    # Interactive with smart confirmations
tenangdb backup --yes              # Automated mode (skip confirmations)
tenangdb backup --force            # Skip frequency checks

# Restore
tenangdb restore -b /path -d target_db    # Interactive with safety checks
tenangdb restore -b /path -d target_db --yes  # Automated mode

# Maintenance
tenangdb cleanup --force           # Clean old backups
tenangdb config                    # Show config paths

# Metrics (separate binary)
tenangdb-exporter                  # Start Prometheus metrics server on :9090
tenangdb-exporter --port 8080      # Custom port
tenangdb-exporter --config /path/to/config.yaml  # Custom config
```

## 🔧 Advanced

```bash
# Docker with proper volume mounting (important for frequency checking)
docker run -it --user $(id -u):$(id -g) \
  -v $(pwd)/config.yaml:/config.yaml \
  -v $(pwd)/backups:/backups \
  -v $(pwd)/tmp:/tmp \
  ghcr.io/abdullahainun/tenangdb:latest backup

# Cloud storage setup
rclone config
upload:
  enabled: true
  destination: "your-remote:database-backups"

# Production deployment with MySQL demo
docker-compose up -d  # Includes MySQL 8.0 + TenangDB

# MySQL user setup (MySQL 8.0+)
CREATE USER 'tenangdb'@'%' IDENTIFIED BY 'secure_password';
GRANT SELECT, SHOW DATABASES, LOCK TABLES, EVENT, TRIGGER, EXECUTE ON *.* TO 'tenangdb'@'%';
GRANT REPLICATION CLIENT ON *.* TO 'tenangdb'@'%';
GRANT INSERT, UPDATE, DELETE, CREATE, DROP, ALTER, INDEX, REFERENCES, CREATE TEMPORARY TABLES, CREATE VIEW ON *.* TO 'tenangdb'@'%';
FLUSH PRIVILEGES;
```

**💡 Docker Note:** Mount `/tmp` volume to persist backup frequency tracking between container restarts.

## 📊 Metrics & Monitoring

TenangDB provides Prometheus metrics via a dedicated exporter binary:

```bash
# Start metrics exporter
tenangdb-exporter --port 9090

# Metrics endpoints
curl http://localhost:9090/metrics   # Prometheus metrics
curl http://localhost:9090/health    # Health check
```

**Available Metrics:**
- `tenangdb_backup_duration_seconds` - Backup execution time
- `tenangdb_backup_success_total` - Total successful backups  
- `tenangdb_backup_failure_total` - Total failed backups
- `tenangdb_upload_duration_seconds` - Upload execution time
- `tenangdb_restore_duration_seconds` - Restore execution time
- `tenangdb_cleanup_files_removed_total` - Files removed during cleanup

**Docker with Metrics:**
```bash
# Run both backup and exporter
docker run -d --name tenangdb-exporter \
  -p 9090:9090 \
  -v $(pwd)/metrics:/var/lib/tenangdb \
  ghcr.io/abdullahainun/tenangdb:latest tenangdb-exporter

docker run --user $(id -u):$(id -g) \
  -v $(pwd)/config.yaml:/config.yaml \
  -v $(pwd)/backups:/backups \
  -v $(pwd)/metrics:/var/lib/tenangdb \
  ghcr.io/abdullahainun/tenangdb:latest backup
```

## 📋 Compatibility

**mydumper:** v0.9.1+ (Ubuntu 18.04) to v0.19.3+ (macOS Homebrew)  
**MySQL:** 5.7+, 8.0+, MariaDB 10.3+  
**Platforms:** macOS (Intel/Apple Silicon), Linux (Ubuntu/CentOS/Debian/Fedora)

## 📚 Documentation

[Installation Guide](INSTALL.md) • [Docker Guide](DOCKER.md) • [MySQL Setup](MYSQL_USER_SETUP.md) • [Production Deployment](PRODUCTION_DEPLOYMENT.md) • [Config Reference](config.yaml.example)

---

**Support:** [Issues](https://github.com/abdullahainun/tenangdb/issues) • **License:** [MIT](LICENSE)
