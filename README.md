# 🛡️ TenangDB

**Backup yang Bikin Tenang** - Secure MySQL backup with intelligent automation.

[![GitHub release](https://img.shields.io/github/release/abdullahainun/tenangdb.svg)](https://github.com/abdullahainun/tenangdb/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/abdullahainun/tenangdb)](https://goreportcard.com/report/github.com/abdullahainun/tenangdb)

*2-minute setup wizard. Production-ready systemd service. Zero configuration headaches.*

## 🎬 Live Demo

[![TenangDB Demo](https://asciinema.org/a/731101.svg)](https://asciinema.org/a/731101)

## ⚡ Quick Start

### Production Setup (Recommended)
```bash
curl -sSL https://go.ainun.cloud/tenangdb-install.sh | sudo bash
```

### Personal Setup
```bash
curl -sSL https://go.ainun.cloud/tenangdb-install.sh | bash
```

### Docker
```bash
docker pull ghcr.io/abdullahainun/tenangdb:latest
docker run --rm ghcr.io/abdullahainun/tenangdb:latest --help
```

## 🔧 Key Features

- **🧙‍♂️ Interactive Setup**: 2-minute wizard with database testing
- **🚀 Auto Deployment**: One-command systemd service installation  
- **📊 Built-in Monitoring**: Prometheus metrics + health checks
- **☁️ Cloud Integration**: S3, GCS, Azure via rclone
- **⚡ Fast & Smart**: mydumper + intelligent error handling

## 📚 Documentation

- **[Installation Guide](INSTALL.md)** - Detailed setup instructions
- **[Commands Reference](docs/COMMANDS.md)** - Complete command list
- **[Configuration](config.yaml.example)** - Full config examples
- **[Production Deployment](PRODUCTION_DEPLOYMENT.md)** - systemd setup
- **[Troubleshooting](TROUBLESHOOTING.md)** - Common issues & solutions
- **[Docker Guide](DOCKER.md)** - Container usage

## 🚀 Basic Usage

```bash
# Interactive setup
tenangdb init

# Run backup
tenangdb backup

# Check status (if systemd deployed)
sudo systemctl status tenangdb.timer
```

## 📋 Compatibility

**Platforms:** Linux, macOS, Docker  
**MySQL:** 5.7+, 8.0+, MariaDB 10.3+  
**Cloud:** 40+ providers via rclone

---

**📚 [Full Documentation](docs/)** • **🐛 [Issues](https://github.com/abdullahainun/tenangdb/issues)** • **📄 [License](LICENSE)**

Built by [Ainun Abdullah](https://github.com/abdullahainun)
