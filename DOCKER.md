# Docker Guide

## Setup

```bash
git clone https://github.com/abdullahainun/tenangdb.git
cd tenangdb
cp configs/config.yaml config.yaml   # edit with your MySQL credentials
make docker-build                    # go mod vendor + docker build
```

## Usage

Start dependent services (MySQL + metrics exporter):

```bash
docker compose up -d mysql tenangdb-exporter
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
  host: mysql              # container name in compose network
  port: 3306
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

Connect to an external MySQL:

```bash
# Create network
docker network create tenangdb-net

# Run backup on that network
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
