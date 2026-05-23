# Docker Guide

## Setup

```bash
mkdir tenangdb && cd tenangdb
cp /path/to/config.yaml.example config.yaml   # edit with your database credentials
docker compose pull                            # pull pre-built image
```

For local builds (development): edit `docker-compose.yml` and uncomment the `build:` lines, then run:

```bash
make docker-build                             # go mod vendor + docker build
```

## Usage

Start the metrics exporter:

```bash
docker compose up -d tenangdb-exporter
```

Run a backup (oneshot):

```bash
docker compose run --rm tenangdb backup --yes
```

Run cleanup (oneshot):

```bash
docker compose run --rm tenangdb cleanup --yes --force
```

Interactive setup wizard:

```bash
docker compose run --rm tenangdb init
```

Check metrics:

```bash
curl http://localhost:9090/metrics
```

## Scheduling

Add to host crontab for automated backups:

```bash
0 2 * * * cd /path/to/tenangdb && docker compose run --rm tenangdb backup --yes
0 3 * * 6 cd /path/to/tenangdb && docker compose run --rm tenangdb cleanup --yes --force
```

## Configuration

Edit `config.yaml` to set:

```yaml
database:
  host: mysql              # container name in compose network (or postgres)
  port: 3306                # 5432 for PostgreSQL
  username: backup_user
  password: your_password

backup:
  databases:
    - production_db
    - analytics_db
  directory: /backups

upload:
  enabled: true
  rclone_config_path: /root/.config/rclone/rclone.conf
  destination: "s3:my-bucket/backups/"
```

## Networking

### Database di Host yang Sama (Linux)

Container secara default tidak bisa akses `localhost` host. Gunakan `host.docker.internal`:

```yaml
# config.yaml
database:
  host: host.docker.internal   # resolve ke IP host
  port: 3306                   # 5432 untuk PostgreSQL
```

```bash
# Butuh --add-host untuk Linux (Docker 20.10+)
docker compose run --rm --add-host host.docker.internal:host-gateway tenangdb backup --yes
```

Atau pake `--network host` (lebih simple, container share network host):

```yaml
# config.yaml
database:
  host: 127.0.0.1
```

```bash
docker compose run --rm --network host tenangdb backup --yes
```

> ⚠️ `--network host` tidak bisa bareng `--add-host` dan port mapping di compose.

### Database di Container Lain (Docker Compose)

Database dan TenangDB di service compose yang sama — cukup pake nama service:

```yaml
# docker-compose.yml
services:
  tenangdb:
    image: ghcr.io/abdullahainun/tenangdb:latest
    # ...

  mysql:
    image: mysql:8.0
    # ...
```

```yaml
# config.yaml
database:
  host: mysql    # nama service di compose
  port: 3306
```

### Database di Server Remote

Pake hostname/IP server langsung:

```yaml
# config.yaml
database:
  host: 192.168.1.100   # atau hostname seperti db.example.com
  port: 3306
```

```bash
# Kalo beda network, buat network dulu
docker network create tenangdb-net
docker compose run --rm --network tenangdb-net tenangdb backup --yes
```

## Volumes

| Mount | Purpose |
|-------|---------|
| `./config.yaml:/config.yaml:ro` | Config file |
| `./backups:/backups` | Backup output |
| `./logs:/logs` | Application logs |
| `./metrics:/var/lib/tenangdb` | Tracking data |
| `~/.config/rclone:/root/.config/rclone:ro` | Rclone config |
