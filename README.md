# TenangDB

🛡️ **Backup yang Bikin Tenang** - Secure MySQL backup with intelligent automation.

*Zero-configuration backup system with smart confirmations and cloud integration.*

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
```

## 🔧 Features

- **Smart Confirmations**: Interactive prompts with backup summary
- **Frequency Checking**: Prevents redundant backups with configurable intervals
- **Auto-Discovery**: Binary paths, directories, and optimal settings
- **Cloud Integration**: Seamless upload to any rclone-supported storage
- **Compression**: tar.gz/zst/xz support with hybrid local/cloud approach
- **Safety First**: Database overwrite warnings and confirmation prompts

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

# Production deployment
docker-compose up -d  # See docker-compose.yml

# MySQL user setup
CREATE USER 'tenangdb'@'%' IDENTIFIED BY 'secure_password';
GRANT SELECT, SHOW DATABASES, SHOW VIEW, LOCK TABLES, EVENT, TRIGGER, ROUTINE, RELOAD, REPLICATION CLIENT ON *.* TO 'tenangdb'@'%';
GRANT INSERT, UPDATE, DELETE, CREATE, DROP, ALTER, INDEX, REFERENCES, CREATE TEMPORARY TABLES, CREATE VIEW ON *.* TO 'tenangdb'@'%';
FLUSH PRIVILEGES;
```

**💡 Docker Note:** Mount `/tmp` volume to persist backup frequency tracking between container restarts.

## 📋 Compatibility

**mydumper:** v0.9.1+ (Ubuntu 18.04) to v0.19.3+ (macOS Homebrew)  
**MySQL:** 5.7+, 8.0+, MariaDB 10.3+  
**Platforms:** macOS (Intel/Apple Silicon), Linux (Ubuntu/CentOS/Debian/Fedora)

## 📚 Documentation

[Installation Guide](INSTALL.md) • [Docker Guide](DOCKER.md) • [MySQL Setup](MYSQL_USER_SETUP.md) • [Production Deployment](PRODUCTION_DEPLOYMENT.md) • [Config Reference](config.yaml.example)

---

**Support:** [Issues](https://github.com/abdullahainun/tenangdb/issues) • **License:** [MIT](LICENSE)