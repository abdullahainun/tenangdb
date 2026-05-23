# Production Deployment

## Docker Compose

```bash
mkdir -p /opt/tenangdb && cd /opt/tenangdb
cp /path/to/config.yaml.example config.yaml   # configure your databases
docker compose pull                            # pull pre-built image
```

For local builds: uncomment `build:` in `docker-compose.yml`, then run `make docker-build`.

Start the metrics exporter:

```bash
docker compose up -d tenangdb-exporter
```

## Scheduling with Cron

```bash
crontab -e

# Backup daily at 2 AM
0 2 * * * cd /opt/tenangdb && docker compose run --rm tenangdb backup --yes >> /var/log/tenangdb.log 2>&1

# Cleanup weekly on Saturday at 3 AM
0 3 * * 6 cd /opt/tenangdb && docker compose run --rm tenangdb cleanup --yes --force >> /var/log/tenangdb.log 2>&1
```

## Scheduling with Systemd

`/etc/systemd/system/tenangdb.service`:

```ini
[Unit]
Description=TenangDB Backup
After=docker.service

[Service]
Type=oneshot
ExecStart=/usr/bin/docker compose -f /opt/tenangdb/docker-compose.yml run --rm tenangdb backup --yes
WorkingDirectory=/opt/tenangdb
```

`/etc/systemd/system/tenangdb.timer`:

```ini
[Unit]
Description=Daily TenangDB Backup

[Timer]
OnCalendar=daily
Persistent=true

[Install]
WantedBy=timers.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now tenangdb.timer
```

## Monitoring

Metrics exporter at `http://localhost:9090/metrics`.

Prometheus scrape config:

```yaml
scrape_configs:
  - job_name: tenangdb
    static_configs:
      - targets: ['localhost:9090']
```

See [grafana/README.md](grafana/README.md) for the Grafana dashboard.

## Security

- Container runs as non-root (uid 1001)
- Mount config as read-only (`:ro`)
- Use environment variables or Docker secrets for passwords
- Isolate the database on a dedicated Docker network

## Kubernetes

See [k8s/README.md](k8s/README.md) for CronJob deployment.
