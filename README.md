# 🛡️ TenangDB

**Backup yang Bikin Tenang** - Secure MySQL backup with intelligent automation.

[![GitHub release](https://img.shields.io/github/release/abdullahainun/tenangdb.svg)](https://github.com/abdullahainun/tenangdb/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/abdullahainun/tenangdb)](https://goreportcard.com/report/github.com/abdullahainun/tenangdb)

## ⚡ Quick Start

### Docker (Recommended)

```bash
# Clone the repository
git clone https://github.com/abdullahainun/tenangdb.git
cd tenangdb

# Copy and edit config
cp configs/config.yaml config.yaml
# ...edit config.yaml with your MySQL credentials...

# Build and run
make docker-build
make docker-up

# Run backup once
docker compose exec tenangdb backup

# Or use the init wizard
docker compose run --rm tenangdb init
```

Or pull the pre-built image:

```bash
docker pull ghcr.io/abdullahainun/tenangdb:latest
docker run --rm -v ./config.yaml:/config.yaml:ro ghcr.io/abdullahainun/tenangdb:latest backup
```

### Development (Go)

```bash
go build -o tenangdb ./cmd
./tenangdb --help
```

## 🔧 Key Features

- **🧙‍♂️ Interactive Setup**: 2-minute configuration wizard
- **📊 Built-in Monitoring**: Prometheus metrics + health checks
- **☁️ Cloud Integration**: S3, GCS, Azure via rclone
- **⚡ Fast & Smart**: mydumper + intelligent error handling
- **🧩 Compression**: tar.gz, tar.zst, tar.xz support

## 📚 Documentation

- **[Commands Reference](docs/COMMANDS.md)** - Complete command list
- **[Configuration](configs/config.yaml)** - Full config examples
- **[Docker Compose](docker-compose.yml)** - Container setup

## 📋 Compatibility

**Platforms:** Linux, macOS (development), Docker (production)  
**MySQL:** 5.7+, 8.0+, MariaDB 10.3+  
**Cloud:** 40+ providers via rclone

---

**📚 [Full Documentation](docs/)** • **🐛 [Issues](https://github.com/abdullahainun/tenangdb/issues)** • **📄 [License](LICENSE)**

Built by [Ainun Abdullah](https://github.com/abdullahainun)
