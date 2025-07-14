# 📖 TenangDB Installation Guide

## 🐳 Docker Installation (Recommended)

```bash
# Pull and run
docker pull ghcr.io/abdullahainun/tenangdb:latest
docker run -v $(pwd)/config.yaml:/config.yaml ghcr.io/abdullahainun/tenangdb:latest backup

# Or use docker-compose
curl -L https://go.ainun.cloud/tenangdb-docker-compose.yml -o docker-compose.yml
docker-compose up -d
```

## 📦 Binary Installation

### Download Release Binary
```bash
# One-liner install
curl -sSL https://go.ainun.cloud/tenangdb-install.sh | bash

# Or manual download
curl -L https://github.com/abdullahainun/tenangdb/releases/latest/download/tenangdb-linux-amd64 -o tenangdb
chmod +x tenangdb && sudo mv tenangdb /usr/local/bin/
```

### Dependencies (Binary Only)
```bash
# Ubuntu/Debian
sudo apt install mydumper rclone mysql-client

# macOS
brew install mydumper rclone mysql-client
```

## ⚙️ Configuration

### Download Config Template
```bash
curl -L https://go.ainun.cloud/tenangdb-config.yaml.example -o config.yaml
nano config.yaml
```

### Basic Config
```yaml
database:
  host: 127.0.0.1
  username: backup_user
  password: "secure_password"

backup:
  databases:
    - your_database_1
    - your_database_2
```

### Test Installation
```bash
# With Docker
docker run -v $(pwd)/config.yaml:/config.yaml ghcr.io/abdullahainun/tenangdb:latest backup --dry-run

# With Binary
tenangdb backup --config config.yaml --dry-run
```

---

## 🚀 Production Setup

### System Service
```bash
# Install with systemd
sudo ./scripts/install.sh

# Enable timers
sudo systemctl enable tenangdb.timer
sudo systemctl start tenangdb.timer
```

### Docker Production
```bash
# Use docker-compose
docker-compose up -d

# Or schedule with cron
0 2 * * * docker run -v /path/to/config.yaml:/config.yaml ghcr.io/abdullahainun/tenangdb:latest backup
```

---

For detailed configuration options, see [config.yaml.example](config.yaml.example) and [DOCKER.md](DOCKER.md).