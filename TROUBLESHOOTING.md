# 🔧 Troubleshooting Guide

## Common Issues & Solutions

### Permission denied on config file
```bash
./tenangdb init                    # Uses user config (~/.config/tenangdb/)
sudo ./tenangdb init --deploy-systemd  # Uses system config (/etc/tenangdb/)
```

### Metrics server port conflict
```bash
# Edit config: metrics.port: "8081" (or disable: metrics.enabled: false)
netstat -tlnp | grep :8080        # Check what's using port 8080
```

### Systemd service won't start
```bash
sudo systemctl status tenangdb.service
sudo journalctl -u tenangdb.service -f
# Common fix: MySQL service name mismatch (now auto-handled)
```

### Partial backup failures
```bash
# Check individual database permissions and disk space
./tenangdb backup --log-level debug
```

### Non-root user issues
```bash
./tenangdb config                  # Shows active config path
# TenangDB automatically uses user-appropriate paths
```

### Docker Issues

#### Container can't connect to MySQL
```bash
# Make sure MySQL container is accessible
docker run --rm --link mysql-container tenangdb:latest backup
```

#### Permission issues with volumes
```bash
# Fix volume permissions
sudo chown -R $(id -u):$(id -g) ./backups
```

## MySQL Setup Issues

### Access denied errors
```sql
-- Grant proper permissions
CREATE USER 'tenangdb'@'%' IDENTIFIED BY 'secure_password';
GRANT SELECT, SHOW DATABASES, LOCK TABLES, EVENT, TRIGGER ON *.* TO 'tenangdb'@'%';
GRANT REPLICATION CLIENT ON *.* TO 'tenangdb'@'%';
FLUSH PRIVILEGES;
```

### Connection timeouts
```yaml
# Increase timeout in config
database:
  timeout: 30s
```

## Cloud Upload Issues

### rclone not configured
```bash
rclone config  # Run interactive setup
```

### Upload fails silently
```bash
# Test rclone connection
rclone ls your-remote:bucket
```

## Getting Help

If you're still having issues:

1. **Check the logs**: `sudo journalctl -u tenangdb.service -f`
2. **Run with debug**: `tenangdb backup --log-level debug`
3. **Report issues**: [GitHub Issues](https://github.com/abdullahainun/tenangdb/issues)