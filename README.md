# TenangDB

🛡️ **Backup yang Bikin Tenang** - Secure MySQL backup with intelligent automation.

*2-minute setup wizard. Production-ready systemd service. Zero configuration headaches.*

## 🎬 Live Demo

[![TenangDB Demo](https://asciinema.org/a/728588.svg)](https://asciinema.org/a/728588)

*Note: The "analytics" database backup error in the demo is expected - it shows how TenangDB handles insufficient privileges gracefully while continuing with other databases.*

## ⚡ Quick Start

**🚀 Production Setup (Recommended)**
```bash
# 1. Install binary
curl -sSL https://go.ainun.cloud/tenangdb-install.sh | bash

# 2. Interactive setup wizard (2 minutes!)
sudo tenangdb init --deploy-systemd

# 3. Done! ✅ 
sudo systemctl status tenangdb.timer
curl http://localhost:8080/metrics
```

**🐳 Docker (Alternative)**
```bash
# Quick test run
docker run -it --rm ghcr.io/abdullahainun/tenangdb:latest init

# Production with persistent config
mkdir tenangdb && cd tenangdb
docker run -it --user $(id -u):$(id -g) \
  -v $(pwd):/workspace \
  ghcr.io/abdullahainun/tenangdb:latest init
```

## ⚙️ Setup Options

**Interactive Wizard (Recommended)**
```bash
tenangdb init              # Guided setup wizard
tenangdb init --deploy-systemd  # + Auto systemd deployment
tenangdb init --config /custom/path.yaml  # Custom location
```

**Manual Config** ([Full example](config.yaml.example))
```yaml
database:
  host: localhost
  username: tenangdb_user
  password: secure_password
  
backup:
  databases: [app_db, logs_db]
  directory: /var/backups/tenangdb
  
upload:
  enabled: true
  destination: "s3:my-backups"
  
metrics:
  enabled: true
  port: 8080
```

## 🔧 Features

- **🧙‍♂️ Setup Wizard**: 2-minute interactive configuration with database testing
- **🚀 Auto Deployment**: One-command systemd service installation  
- **🛡️ Production Ready**: Security hardening, user isolation, auto-restart
- **📊 Smart Monitoring**: Prometheus metrics, health checks, centralized logging
- **☁️ Cloud Integration**: Upload to S3, GCS, Azure, or any rclone-supported storage
- **⚡ Fast Backups**: mydumper parallel processing with automatic fallback
- **🧠 Intelligent**: Frequency checking, duplicate prevention, graceful error handling

## 📋 Commands

```bash
# Setup & Deploy
tenangdb init                      # Interactive setup wizard
tenangdb init --deploy-systemd     # Setup + auto systemd deployment
tenangdb config                    # Show config paths

# Operations  
tenangdb backup                    # Interactive backup
tenangdb backup --yes              # Automated mode
tenangdb restore -b /path -d db    # Restore with safety checks
tenangdb cleanup                   # Clean old backups

# Systemd Management (after --deploy-systemd)
sudo systemctl status tenangdb.timer     # Check backup schedule
sudo systemctl start tenangdb.service    # Manual backup
sudo journalctl -u tenangdb.service -f   # View logs
curl http://localhost:8080/metrics       # Prometheus metrics
```

## 🔧 Advanced

**Custom Deployment**
```bash
# Custom systemd user
tenangdb init --deploy-systemd --systemd-user mybackup

# Multiple configs
tenangdb init --config /etc/tenangdb/prod.yaml
tenangdb init --config /etc/tenangdb/staging.yaml

# Docker with systemd
docker run -d --privileged \
  -v /sys/fs/cgroup:/sys/fs/cgroup:ro \
  tenangdb:latest /sbin/init
```

**MySQL User Setup**
```sql
CREATE USER 'tenangdb'@'%' IDENTIFIED BY 'secure_password';
GRANT SELECT, SHOW DATABASES, LOCK TABLES, EVENT, TRIGGER ON *.* TO 'tenangdb'@'%';
GRANT REPLICATION CLIENT ON *.* TO 'tenangdb'@'%';
FLUSH PRIVILEGES;
```

**Cloud Storage**
```bash
rclone config  # Setup remote (S3, GCS, Azure, etc.)
# Wizard will guide you through cloud setup
```

## 📊 Monitoring

**Built-in Metrics** (enabled with `--deploy-systemd`)
- ✅ Prometheus metrics on `:8080/metrics`
- ✅ Health check endpoint `:8080/health` 
- ✅ Centralized logging via `journalctl`
- ✅ Service status monitoring

**Key Metrics:**
- `tenangdb_backup_duration_seconds` - Backup execution time
- `tenangdb_backup_success_total` - Successful backups counter
- `tenangdb_upload_duration_seconds` - Cloud upload time
- `tenangdb_cleanup_files_removed_total` - Cleaned up files

**Grafana Dashboard:** [Import from examples/](grafana/dashboard.json)

## 🎯 Why TenangDB?

| Feature | Traditional Scripts | TenangDB |
|---------|-------------------|----------|
| **Setup Time** | 30+ minutes | 2 minutes |
| **Configuration** | Manual YAML editing | Interactive wizard |
| **Production Deploy** | Multiple manual steps | `--deploy-systemd` |
| **Error Handling** | Script breaks | Graceful fallbacks |
| **Monitoring** | DIY | Built-in Prometheus |
| **Security** | Basic | Hardened systemd |

## 📋 Compatibility

**Platforms:** Linux (systemd), macOS, Docker  
**MySQL:** 5.7+, 8.0+, MariaDB 10.3+  
**Cloud:** S3, GCS, Azure, 40+ providers via rclone

---

**📚 Docs:** [Config Reference](config.yaml.example) • **🐛 Issues:** [GitHub](https://github.com/abdullahainun/tenangdb/issues) • **📄 License:** [MIT](LICENSE)
