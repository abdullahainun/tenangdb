# 📖 TenangDB Installation Guide

## 🚀 Quick Install (Recommended)

**🎯 Production Setup (2 minutes)**
```bash
# 1. Install binary
curl -sSL https://go.ainun.cloud/tenangdb-install.sh | bash

# 2. Interactive setup wizard + systemd deployment
sudo tenangdb init --deploy-systemd

# 3. Done! ✅
sudo systemctl status tenangdb.timer
curl http://localhost:8080/metrics
```

**✨ What this does:**
- Downloads & installs binary + dependencies
- Interactive wizard (database config, backup setup)
- Auto-deploys as systemd service with security hardening
- Sets up daily backups + weekend cleanup
- Enables Prometheus metrics monitoring

---

## 🧙‍♂️ Interactive Setup Options

```bash
# Basic setup wizard
tenangdb init

# Setup + auto-deploy systemd services
tenangdb init --deploy-systemd

# Custom config location
tenangdb init --config /etc/tenangdb/config.yaml

# Custom systemd user
tenangdb init --deploy-systemd --systemd-user mybackup
```

**Wizard Features:**
- ✅ Database connection testing
- ✅ Dependency validation (mydumper, rclone, etc.)
- ✅ Smart defaults based on platform
- ✅ Cloud storage setup (optional)
- ✅ Metrics configuration

---

## 🐳 Docker Installation

**Quick Test:**
```bash
docker run -it --rm ghcr.io/abdullahainun/tenangdb:latest init
```

**Production with Docker:**
```bash
# Setup workspace
mkdir tenangdb && cd tenangdb

# Run setup wizard
docker run -it --user $(id -u):$(id -g) \
  -v $(pwd):/workspace \
  ghcr.io/abdullahainun/tenangdb:latest init

# Run backups
docker run --user $(id -u):$(id -g) \
  -v $(pwd)/config.yaml:/config.yaml \
  -v $(pwd)/backups:/backups \
  ghcr.io/abdullahainun/tenangdb:latest backup
```

---

## 📦 Manual Installation

**For Custom Setups Only**

```bash
# Download binaries
curl -L https://github.com/abdullahainun/tenangdb/releases/latest/download/tenangdb-linux-amd64 -o tenangdb
curl -L https://github.com/abdullahainun/tenangdb/releases/latest/download/tenangdb-exporter-linux-amd64 -o tenangdb-exporter
chmod +x tenangdb tenangdb-exporter
sudo mv tenangdb tenangdb-exporter /usr/local/bin/

# Install dependencies
sudo apt install mydumper rclone mysql-client  # Ubuntu/Debian
brew install mydumper rclone mysql-client      # macOS

# Manual config
curl -L https://go.ainun.cloud/tenangdb-config.yaml.example -o config.yaml
nano config.yaml
```

---

## ✅ Verify Installation

```bash
# Check binary
tenangdb --version

# Test config (dry run)
tenangdb backup --dry-run

# Check systemd services (if deployed)
sudo systemctl status tenangdb.timer
sudo systemctl status tenangdb-exporter.service
```

**Next Steps:** See [PRODUCTION_DEPLOYMENT.md](PRODUCTION_DEPLOYMENT.md) for advanced configuration.