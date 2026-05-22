# Production Deployment Guide

## Docker Compose (Recommended)

```bash
# Clone and configure
git clone https://github.com/abdullahainun/tenangdb.git
cd tenangdb
cp configs/config.yaml config.yaml

# Build the image
make docker-build

# Start all services (TenangDB + metrics exporter + MySQL)
make docker-up

# Run backup
docker compose exec tenangdb backup
```

### Scheduled Backups

For automated daily backups, add a cron entry on the host:

```bash
# Edit crontab
crontab -e

# Run backup daily at 2 AM
0 2 * * * cd /path/to/tenangdb && docker compose exec -T tenangdb backup --yes >> /var/log/tenangdb-cron.log 2>&1

# Run cleanup weekly on weekends
0 3 * * 6 cd /path/to/tenangdb && docker compose exec -T tenangdb cleanup --yes --force >> /var/log/tenangdb-cron.log 2>&1
```

### With Systemd Timer (Alternative to cron)

```ini
# /etc/systemd/system/tenangdb-docker.service
[Unit]
Description=TenangDB Backup (Docker)
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
ExecStart=/usr/bin/docker compose -f /opt/tenangdb/docker-compose.yml exec -T tenangdb backup --yes
WorkingDirectory=/opt/tenangdb
```

```ini
# /etc/systemd/system/tenangdb-docker.timer
[Unit]
Description=Daily TenangDB Backup

[Timer]
OnCalendar=daily
Persistent=true

[Install]
WantedBy=timers.target
```

## Kubernetes

See [k8s/README.md](k8s/README.md) for Kubernetes deployment with CronJob.

## Directory Structure

```
tenangdb/
├── config.yaml          # Main configuration
├── docker-compose.yml   # Docker services
├── backups/             # Backup files
│   ├── database1/
│   └── database2/
├── logs/                # Application logs
│   └── tenangdb.log
└── metrics/             # Metrics tracking
    └── metrics.json
```

## Security

1. **Non-root user**: Container runs as uid 1001 by default
2. **Read-only config**: Mount config as read-only (`:ro`)
3. **MySQL credentials**: Use environment variables for sensitive data
4. **Network isolation**: Use Docker networks to isolate MySQL traffic

## Monitoring

### Metrics Exporter

The docker-compose includes a metrics exporter:

```bash
curl http://localhost:9090/metrics
```

### Grafana Dashboard

See [grafana/README.md](grafana/README.md) for the Grafana dashboard setup.
