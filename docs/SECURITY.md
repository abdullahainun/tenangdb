# Security Guide

## Production Security Hardening

### 1. User Privileges & Access Control

**Database User Permissions:**
```sql
CREATE USER 'tenangdb_backup'@'localhost' IDENTIFIED BY 'strong_password_here';
GRANT SELECT, LOCK TABLES, SHOW VIEW, EVENT, TRIGGER ON *.* TO 'tenangdb_backup'@'localhost';
GRANT RELOAD, SUPER ON *.* TO 'tenangdb_backup'@'localhost';
FLUSH PRIVILEGES;
```

### 2. File System Security

**Directory Permissions:**
```bash
sudo chmod 640 /etc/tenangdb/config.yaml
```

**Binary Security:**
```bash
sha256sum /tenangdb
```

### 3. Docker Security

**Container runs as non-root user (uid 1001) by default.**

**Read-only mounts:**
```yaml
volumes:
  - ./config.yaml:/config.yaml:ro
```

**Resource limits in docker-compose:**
```yaml
services:
  tenangdb:
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
```

## Configuration Security

### Credential Management

```yaml
database:
  username: tenangdb_backup
  password: "${TENANGDB_DB_PASSWORD}"
```

### MySQL Configuration Security

```bash
sudo chmod 640 /etc/tenangdb/.my.cnf
```

### Network Security

```yaml
database:
  host: localhost
  ssl_mode: "PREFERRED"
```

## Monitoring & Alerting

### Log Analysis

```bash
# Monitor for errors
tail -f /var/log/tenangdb/tenangdb.log | grep -E "(FAILED|ERROR)"
```

### Prometheus Metrics

```yaml
metrics:
  enabled: true
  port: 8080
```

## Security Checklist

- [ ] **Database User**: Created with minimal required privileges
- [ ] **File Permissions**: All files secured
- [ ] **Container**: Runs as non-root user
- [ ] **Network**: Database connections use localhost or secure networks
- [ ] **Credentials**: No plaintext passwords in config files

## Incident Response

```bash
# Stop the container
docker compose down

# Check logs
docker compose logs tenangdb

# Rotate credentials
# Change database passwords immediately
```

## Additional Resources

- [MySQL Security Best Practices](https://dev.mysql.com/doc/refman/8.0/en/security-guidelines.html)
- [Docker Security](https://docs.docker.com/engine/security/)

---

**Last Updated**: 2025-01-06
**Version**: 1.0
