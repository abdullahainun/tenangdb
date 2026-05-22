# Troubleshooting Guide

## Common Issues

### Permission denied on config file
```bash
./tenangdb init                    # Uses user config (~/.config/tenangdb/)
```

### Metrics server port conflict
```bash
# Edit config: metrics.port: "8081" (or disable: metrics.enabled: false)
netstat -tlnp | grep :8080        # Check what's using port 8080
```

### Partial backup failures
```bash
# Check individual database permissions and disk space
./tenangdb backup --log-level debug
```

### Non-root user issues
```bash
./tenangdb config                  # Shows active config path
```

### Docker Issues

#### Container can't connect to MySQL
```bash
# Make sure MySQL container is accessible on the same network
docker network ls
```

#### Permission issues with volumes
```bash
# Fix volume permissions
sudo chown -R $(id -u):$(id -g) ./backups
```

## MySQL Setup Issues

### Access denied errors
```sql
CREATE USER 'tenangdb'@'%' IDENTIFIED BY 'secure_password';
GRANT SELECT, SHOW DATABASES, LOCK TABLES, EVENT, TRIGGER ON *.* TO 'tenangdb'@'%';
GRANT REPLICATION CLIENT ON *.* TO 'tenangdb'@'%';
FLUSH PRIVILEGES;
```

### Connection timeouts
```yaml
database:
  timeout: 30s
```

## Cloud Upload Issues

### rclone not configured
```bash
rclone config
```

### Upload fails silently
```bash
rclone ls your-remote:bucket
```

## Getting Help

```bash
tenangdb backup --log-level debug
```

Report issues: [GitHub Issues](https://github.com/abdullahainun/tenangdb/issues)
