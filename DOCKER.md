# Docker Guide

## Quick Start

Pull or build the image:

```bash
docker pull ghcr.io/abdullahainun/tenangdb:latest

# Or build locally
make docker-build
```

### Simple Backup

```bash
mkdir tenangdb && cd tenangdb

cat > config.yaml << 'EOF'
database:
  host: mysql-host
  username: backup_user
  password: your_password
backup:
  databases: [your_database]
  directory: /backups
EOF

docker run --rm \
  -v $(pwd)/config.yaml:/config.yaml:ro \
  -v $(pwd)/backups:/backups \
  ghcr.io/abdullahainun/tenangdb:latest backup --config /config.yaml --yes
```

### Docker Compose

```bash
git clone https://github.com/abdullahainun/tenangdb.git
cd tenangdb
cp configs/config.yaml config.yaml

make docker-up
```

## Interactive Setup

```bash
docker run -it --rm \
  -v $(pwd):/workspace \
  ghcr.io/abdullahainun/tenangdb:latest init
```

## Networking

### Docker Network

```bash
docker network create tenangdb-net

# Start MySQL
docker run --name mysql-db --network tenangdb-net \
  -e MYSQL_ROOT_PASSWORD=pass -d mysql:8.0

# Run backup
docker run --rm --network tenangdb-net \
  -v $(pwd)/config.yaml:/config.yaml:ro \
  -v $(pwd)/backups:/backups \
  ghcr.io/abdullahainun/tenangdb:latest backup --yes
```

## Volume Mounts

- **Config**: `-v $(pwd)/config.yaml:/config.yaml:ro`
- **Backups**: `-v $(pwd)/backups:/backups`
- **Logs**: `-v $(pwd)/logs:/logs`
- **Metrics**: `-v $(pwd)/metrics:/var/lib/tenangdb`
- **Rclone**: `-v ~/.config/rclone:/root/.config/rclone:ro`

## Multi-Architecture

Supports `linux/amd64` and `linux/arm64` — Docker pulls the correct architecture automatically.

## Health Checks

The docker-compose includes health checks:
```yaml
healthcheck:
  test: ["CMD", "tenangdb", "version"]
  interval: 30s
  timeout: 10s
  retries: 3
```
