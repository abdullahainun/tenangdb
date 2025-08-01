<div align="center">

# 🛡️ TenangDB

### *Backup yang Bikin Tenang*
**Secure MySQL backup with intelligent automation**

[![GitHub release](https://img.shields.io/github/release/abdullahainun/tenangdb.svg)](https://github.com/abdullahainun/tenangdb/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/abdullahainun/tenangdb)](https://goreportcard.com/report/github.com/abdullahainun/tenangdb)
[![Docker Pulls](https://img.shields.io/docker/pulls/abdullahainun/tenangdb)](https://hub.docker.com/r/abdullahainun/tenangdb)

*2-minute setup wizard. Production-ready systemd service. Zero configuration headaches.*

</div>

---

## 🎬 Live Demo

[![TenangDB Demo](https://asciinema.org/a/728588.svg)](https://asciinema.org/a/728588)

> **Note:** The "analytics" database backup error in the demo is expected - it shows how TenangDB handles insufficient privileges gracefully while continuing with other databases.

---

## ✨ Overview

TenangDB transforms the complex world of MySQL backups into a simple, secure, and automated experience. Built for both developers and production environments, it eliminates the traditional pain points of database backup management.

### 🚀 Why TenangDB?

<div align="center">

| **Traditional Scripts** | **TenangDB** |
|------------------------|--------------|
| 30+ minutes setup | **2 minutes** |
| Manual YAML editing | **Interactive wizard** |
| Multiple manual steps | **`--deploy-systemd`** |
| Script breaks on errors | **Graceful fallbacks + detailed reporting** |
| DIY monitoring | **Built-in Prometheus with conflict detection** |
| Basic security | **Hardened systemd + privilege-aware paths** |

</div>

---

## 🔧 Key Features

<div align="center">

### 🧙‍♂️ Setup Wizard
*2-minute interactive configuration with database testing*

### 🚀 Auto Deployment  
*One-command systemd service installation with privilege detection*

### 🛡️ Production Ready
*Security hardening, user isolation, proper file permissions*

### 📊 Smart Monitoring
*Prometheus metrics with graceful port conflict handling*

### ☁️ Cloud Integration
*Upload to S3, GCS, Azure, or any rclone-supported storage*

### ⚡ Fast Backups
*mydumper parallel processing with automatic fallback*

</div>

---

## 🚀 Quick Start

### **🏗️ Production Setup (Recommended)**
```bash
# One-command install + setup (includes dependencies!)
curl -sSL https://go.ainun.cloud/tenangdb-install.sh | sudo bash

# Done! ✅ Check your setup:
sudo systemctl status tenangdb.timer
curl http://localhost:8080/metrics
```

### **👤 Personal Setup (Development)**
```bash
# Install for current user only
curl -sSL https://go.ainun.cloud/tenangdb-install.sh | bash

# Run your first backup
tenangdb backup
```

### **🐳 Docker (Verified Working)**
```bash
# Pull the latest TenangDB image
docker pull ghcr.io/abdullahainun/tenangdb:latest

# Method 1: Quick backup with Docker networking
mkdir tenangdb-docker && cd tenangdb-docker

# Create config pointing to your MySQL host
echo "database:
  host: mysql-host
  username: your-user
  password: your-pass
backup:
  databases: [your-db]
  directory: /backups" > config.yaml

# Run backup
docker run --rm \
  -v $(pwd)/config.yaml:/config.yaml:ro \
  -v $(pwd)/backups:/backups \
  ghcr.io/abdullahainun/tenangdb:latest backup --config /config.yaml --yes

# Method 2: Full docker-compose with MySQL + TenangDB
curl -L https://raw.githubusercontent.com/abdullahainun/tenangdb/main/docker-compose.yml -o docker-compose.yml
docker-compose up -d  # Includes MySQL, TenangDB, and metrics exporter
```

---

## ⚙️ Configuration

### **🧙‍♂️ Interactive Wizard (Recommended)**
```bash
tenangdb init              # Guided setup wizard
tenangdb init --deploy-systemd  # + Auto systemd deployment
tenangdb init --config /custom/path.yaml  # Custom location
```

### **📝 Manual Config** ([Full example](config.yaml.example))
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

---

## 📋 Commands Reference

<div align="center">

### **Setup & Deploy**
</div>

```bash
tenangdb init                      # Interactive setup wizard (privilege-aware)
tenangdb init --deploy-systemd     # Setup + auto systemd deployment
tenangdb config                    # Show config paths and active config
```

<div align="center">

### **Operations**
</div>

```bash
tenangdb backup                    # Interactive backup with confirmation
tenangdb backup --yes              # Skip confirmations (automated mode)
tenangdb backup --force            # Skip frequency checks
tenangdb restore -b /path -d db    # Restore with safety checks
tenangdb cleanup                   # Clean old backups
```

<div align="center">

### **Systemd Management** *(after --deploy-systemd)*
</div>

```bash
sudo systemctl status tenangdb.timer     # Check backup schedule
sudo systemctl start tenangdb.service    # Manual backup
sudo journalctl -u tenangdb.service -f   # View logs
curl http://localhost:8080/metrics       # Prometheus metrics (if enabled)
```

---

## 🏗️ Advanced Configuration

### **Custom Deployment**
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

### **MySQL User Setup**
```sql
CREATE USER 'tenangdb'@'%' IDENTIFIED BY 'secure_password';
GRANT SELECT, SHOW DATABASES, LOCK TABLES, EVENT, TRIGGER ON *.* TO 'tenangdb'@'%';
GRANT REPLICATION CLIENT ON *.* TO 'tenangdb'@'%';
FLUSH PRIVILEGES;
```

### **Cloud Storage**
```bash
rclone config  # Setup remote (S3, GCS, Azure, etc.)
# Wizard will guide you through cloud setup
```

---

## 📊 Monitoring & Metrics

<div align="center">

### **Built-in Metrics** *(enabled with `--deploy-systemd`)*

✅ Prometheus metrics on `:8080/metrics`  
✅ Health check endpoint `:8080/health`  
✅ Centralized logging via `journalctl`  
✅ Service status monitoring

</div>

### **Key Metrics:**
- `tenangdb_backup_duration_seconds` - Backup execution time
- `tenangdb_backup_success_total` - Successful backups counter
- `tenangdb_upload_duration_seconds` - Cloud upload time
- `tenangdb_cleanup_files_removed_total` - Cleaned up files

### **Grafana Dashboard:** [Import from examples/](grafana/dashboard.json)

---

## 🔧 Troubleshooting

<div align="center">

### **Common Issues & Solutions**

</div>

```bash
# Permission denied on config file
./tenangdb init                    # Uses user config (~/.config/tenangdb/)
sudo ./tenangdb init --deploy-systemd  # Uses system config (/etc/tenangdb/)

# Metrics server port conflict
# Edit config: metrics.port: "8081" (or disable: metrics.enabled: false)
netstat -tlnp | grep :8080        # Check what's using port 8080

# Systemd service won't start
sudo systemctl status tenangdb.service
sudo journalctl -u tenangdb.service -f
# Common fix: MySQL service name mismatch (now auto-handled)

# Partial backup failures
# Check individual database permissions and disk space
./tenangdb backup --log-level debug

# Non-root user issues
./tenangdb config                  # Shows active config path
# TenangDB automatically uses user-appropriate paths
```

---

## 🌐 Compatibility

<div align="center">

| **Platforms** | **MySQL** | **Cloud** |
|---------------|-----------|-----------|
| Linux (systemd) | 5.7+ | S3, GCS, Azure |
| macOS | 8.0+ | 40+ providers |
| Docker | MariaDB 10.3+ | via rclone |

</div>

---

## 👥 Built by

<div align="center">

[![Abdullah Ainun Najib](https://github.com/abdullahainun.png?size=50)](https://github.com/abdullahainun)

**[Abdullah Ainun Najib](https://github.com/abdullahainun)**  
*Creator & Maintainer*

</div>

---

## 🤝 Contributing

We welcome contributions! Here's how you can help:

1. 🍴 Fork the repository
2. 🌿 Create your feature branch (`git checkout -b feature/amazing-feature`)
3. 💾 Commit your changes (`git commit -m 'Add amazing feature'`)
4. 📤 Push to the branch (`git push origin feature/amazing-feature`)
5. 🔄 Open a Pull Request

---

## 📄 License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details.

---

<div align="center">

### 🔗 Links

**📚 [Documentation](config.yaml.example)** • **🐛 [Issues](https://github.com/abdullahainun/tenangdb/issues)** • **💬 [Discussions](https://github.com/abdullahainun/tenangdb/discussions)**

---

*Made with ❤️ for the MySQL community*

</div>