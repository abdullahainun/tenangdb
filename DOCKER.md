# 🐳 Docker Guide

## Quick Start

### Pull Image
```bash
docker pull ghcr.io/abdullahainun/tenangdb:latest
```

### Method 1: Direct Backup

Create config:
```bash
mkdir tenangdb-docker && cd tenangdb-docker

echo "database:
  host: mysql-host
  username: your-user
  password: your-pass
backup:
  databases: [your-db]
  directory: /backups" > config.yaml
```

Run backup:
```bash
docker run --rm \
  -v $(pwd)/config.yaml:/config.yaml:ro \
  -v $(pwd)/backups:/backups \
  ghcr.io/abdullahainun/tenangdb:latest backup --config /config.yaml --yes
```

### Method 2: Docker Compose

Download compose file:
```bash
curl -L https://raw.githubusercontent.com/abdullahainun/tenangdb/main/docker-compose.yml -o docker-compose.yml
```

Start services:
```bash
docker-compose up -d  # Includes MySQL, TenangDB, and metrics exporter
```

## Interactive Setup

Run setup wizard in container:
```bash
docker run -it --rm \
  -v $(pwd):/workspace \
  ghcr.io/abdullahainun/tenangdb:latest init --config /workspace/config.yaml
```

## Networking

### Link to MySQL Container
```bash
# Start MySQL
docker run --name mysql-db -e MYSQL_ROOT_PASSWORD=pass -d mysql:8.0

# Run backup with link
docker run --rm --link mysql-db \
  -v $(pwd)/config.yaml:/config.yaml:ro \
  -v $(pwd)/backups:/backups \
  ghcr.io/abdullahainun/tenangdb:latest backup --yes
```

### Use Docker Network
```bash
# Create network
docker network create tenangdb-net

# Start MySQL on network
docker run --name mysql-db --network tenangdb-net \
  -e MYSQL_ROOT_PASSWORD=pass -d mysql:8.0

# Run backup on same network
docker run --rm --network tenangdb-net \
  -v $(pwd)/config.yaml:/config.yaml:ro \
  -v $(pwd)/backups:/backups \
  ghcr.io/abdullahainun/tenangdb:latest backup --yes
```

## Environment Variables

```bash
docker run --rm \
  -e TZ=Asia/Jakarta \
  -e MYSQL_HOST=mysql-server \
  -e MYSQL_USER=backup_user \
  -e MYSQL_PASSWORD=secure_pass \
  -v $(pwd)/backups:/backups \
  ghcr.io/abdullahainun/tenangdb:latest backup --yes
```

## Volume Mounts

### Essential Mounts
- **Config**: `-v $(pwd)/config.yaml:/config.yaml:ro`
- **Backups**: `-v $(pwd)/backups:/backups`
- **Logs**: `-v $(pwd)/logs:/logs`

### Optional Mounts
- **Metrics**: `-v $(pwd)/metrics:/var/lib/tenangdb`
- **rclone config**: `-v ~/.config/rclone:/root/.config/rclone:ro`

## Multi-Architecture

Image supports:
- `linux/amd64`
- `linux/arm64`

Docker automatically pulls the correct architecture.

## Production with systemd

```bash
docker run -d --privileged \
  -v /sys/fs/cgroup:/sys/fs/cgroup:ro \
  --name tenangdb-prod \
  ghcr.io/abdullahainun/tenangdb:latest /sbin/init
```

## Health Checks

Built-in health check:
```yaml
healthcheck:
  test: ["CMD", "/tenangdb", "version"]
  interval: 30s
  timeout: 10s
  retries: 3
```

## Security

Container runs as non-root user (uid 1001) by default for security.