# TenangDB

**Backup yang Bikin Tenang** — MySQL backup automation tool.

```bash
git clone https://github.com/abdullahainun/tenangdb.git
cd tenangdb
cp configs/config.yaml config.yaml   # edit with your MySQL credentials
make docker-build                    # build image
docker compose up -d mysql           # start MySQL (optional)
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
| [Configuration](configs/config.yaml) | Config example |

## Compatibility

**Platform:** Linux, macOS (dev), Docker (prod)  
**MySQL:** 5.7+, 8.0+, MariaDB 10.3+  
**Cloud:** 40+ providers via rclone
