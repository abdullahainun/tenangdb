# TenangDB

**Backup yang Bikin Tenang** — MySQL & PostgreSQL backup automation tool.

```bash
# Quick start
cp config.yaml.example config.yaml          # edit with your database credentials
docker compose pull                         # pull pre-built image
docker compose run --rm tenangdb backup --yes
```

## Features

- **Interactive setup** — `docker compose run --rm tenangdb init`
- **Metrics exporter** — Prometheus endpoint at `:9090`
- **Cloud upload** — S3, GCS, Azure via rclone
- **Compression** — tar.gz, tar.zst, tar.xz
- **Scheduling** — cron or systemd timer on the host

## Docs

| Guide | Description |
|-------|-------------|
| [Docker](DOCKER.md) | Setup, networking, volumes |
| [Production](PRODUCTION_DEPLOYMENT.md) | Scheduling, security, monitoring |
| [Commands](docs/COMMANDS.md) | CLI reference |
| [Configuration](config.yaml.example) | Quick config example |
| [Full Reference](docs/CONFIGURATION.md) | All config options |

## Compatibility

**Platform:** Linux, macOS (dev), Docker (prod)  
**MySQL:** 5.7+, 8.0+, MariaDB 10.3+  
**PostgreSQL:** 12+  
**Cloud:** 40+ providers via rclone
