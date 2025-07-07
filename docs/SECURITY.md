# Security Guide

This document outlines security best practices and guidelines for deploying and using TenangDB in production environments.

## 🔐 Security Overview

TenangDB is designed with security as a core principle. This guide covers essential security measures to protect your database backups and ensure secure operations.

## 🛡️ Production Security Hardening

### 1. User Privileges & Access Control

**Create Dedicated User:**
```bash
# Create tenangdb user with minimal privileges
sudo useradd -r -s /bin/false -d /opt/tenangdb tenangdb
sudo mkdir -p /opt/tenangdb
sudo chown tenangdb:tenangdb /opt/tenangdb
```

**Database User Permissions:**
```sql
-- Create backup user with minimal required privileges
CREATE USER 'tenangdb_backup'@'localhost' IDENTIFIED BY 'strong_password_here';

-- Grant only necessary permissions
GRANT SELECT, LOCK TABLES, SHOW VIEW, EVENT, TRIGGER ON *.* TO 'tenangdb_backup'@'localhost';
GRANT RELOAD, SUPER ON *.* TO 'tenangdb_backup'@'localhost';

-- For specific databases only (recommended)
GRANT SELECT, LOCK TABLES, SHOW VIEW, EVENT, TRIGGER ON database_name.* TO 'tenangdb_backup'@'localhost';

FLUSH PRIVILEGES;
```

### 2. File System Security

**Note:** The `install.sh` script creates and configures the necessary directories. The commands below are for verification or manual setup.

**Directory Permissions:**
```bash
# Secure configuration directory
sudo mkdir -p /etc/tenangdb
sudo chown root:tenangdb /etc/tenangdb
sudo chmod 750 /etc/tenangdb

# Secure backup directory
sudo mkdir -p /var/backups/tenangdb
sudo chown tenangdb:tenangdb /var/backups/tenangdb
sudo chmod 750 /var/backups/tenangdb

# Set configuration file permissions
sudo chmod 640 /etc/tenangdb/config.yaml
sudo chown root:tenangdb /etc/tenangdb/config.yaml
```

**Binary Security:**
```bash
# Place binary in secure location
sudo cp tenangdb /opt/tenangdb/
sudo chown root:root /opt/tenangdb/tenangdb
sudo chmod 755 /opt/tenangdb/tenangdb

# Verify binary integrity (optional)
sha256sum /opt/tenangdb/tenangdb
```

### 3. Systemd Security Configuration

**Enhanced Service Security:**
```ini
[Unit]
Description=TenangDB Backup Service
After=network.target mysqld.service
Requires=mysqld.service

[Service]
Type=oneshot
User=tenangdb
Group=tenangdb
WorkingDirectory=/opt/tenangdb
ExecStart=/opt/tenangdb/tenangdb backup --config /etc/tenangdb/config.yaml

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/backups/tenangdb /var/log/tenangdb
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictSUIDSGID=true
RestrictRealtime=true
ProtectHostname=true
ProtectClock=true

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

# Network security
PrivateNetwork=false
RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX

[Install]
WantedBy=multi-user.target
```

## 🔑 Configuration Security

### 1. Credential Management

**Secure Password Storage:**
```yaml
# Use strong passwords (minimum 16 characters)
database:
  username: tenangdb_backup
  password: "Use-Strong-Password-With-Special-Characters-123!"
```

**Environment Variables (Recommended):**
```bash
# Set sensitive data via environment variables
export TENANGDB_DB_PASSWORD="your-secure-password"
export TENANGDB_ENCRYPTION_KEY="your-encryption-key"
```

**Configuration in code:**
```yaml
database:
  username: tenangdb_backup
  password: "${TENANGDB_DB_PASSWORD}"
```

### 2. MySQL Configuration Security

**Secure MySQL Defaults File:**
```bash
# Create secure defaults file
sudo cat > /etc/tenangdb/.my.cnf << 'EOF'
[client]
user=tenangdb_backup
password=your-secure-password
host=localhost
port=3306

[mydumper]
user=tenangdb_backup
password=your-secure-password
EOF

# Secure the file
sudo chown root:tenangdb /etc/tenangdb/.my.cnf
sudo chmod 640 /etc/tenangdb/.my.cnf
```

### 3. Network Security

**Database Connection Security:**
```yaml
database:
  host: localhost  # Use localhost when possible
  port: 3306
  # Enable SSL if available
  ssl_mode: "PREFERRED"
  ssl_ca: "/path/to/ca.pem"
  ssl_cert: "/path/to/client-cert.pem"
  ssl_key: "/path/to/client-key.pem"
```

## 🔒 Backup Encryption & Storage

### 1. Local Backup Security

**Encrypt Backup Files:**
```bash
# Enable compression with encryption
backup:
  directory: /var/backups/tenangdb
  compression: gzip
  encryption: true
  encryption_key: "${TENANGDB_ENCRYPTION_KEY}"
```

**Secure Backup Directory:**
```bash
# Set restrictive permissions on backup files
find /var/backups/tenangdb -type f -exec chmod 640 {} \;
find /var/backups/tenangdb -type d -exec chmod 750 {} \;
```

### 2. Cloud Storage Security

**Secure Rclone Configuration:**
```bash
# Create encrypted rclone config
sudo mkdir -p /etc/tenangdb
sudo rclone config create s3backup s3 \
  provider AWS \
  access_key_id your-access-key \
  secret_access_key your-secret-key \
  region us-east-1 \
  server_side_encryption AES256

# Secure rclone config
sudo chown root:tenangdb /etc/tenangdb/rclone.conf
sudo chmod 640 /etc/tenangdb/rclone.conf
```

**Upload Configuration:**
```yaml
upload:
  enabled: true
  rclone_config_path: /etc/tenangdb/rclone.conf
  destination: "s3backup:your-bucket/database-backups/"
  encryption: true
  verify_upload: true
```

## 🚨 Monitoring & Alerting

### 1. Security Monitoring

**Log Analysis:**
```bash
# Monitor for security events
sudo journalctl -u tenangdb.service -f | grep -E "(FAILED|ERROR|UNAUTHORIZED)"

# Set up log rotation
sudo cat > /etc/logrotate.d/tenangdb << 'EOF'
/var/log/tenangdb/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 640 tenangdb tenangdb
}
EOF
```

**Prometheus Metrics:**
```yaml
metrics:
  enabled: true
  port: 9090
  path: /metrics
  # Secure metrics endpoint
  basic_auth:
    username: prometheus
    password: "${PROMETHEUS_PASSWORD}"
```

### 2. Backup Verification

**Integrity Checks:**
```yaml
backup:
  verify_backup: true
  checksum_algorithm: sha256
  test_restore: true
```

**Automated Testing:**
```bash
#!/bin/bash
# Test backup integrity
tenangdb verify --backup-path /var/backups/tenangdb/latest
if [ $? -ne 0 ]; then
    echo "ALERT: Backup verification failed!" | mail admin@company.com
fi
```

## 🔧 Security Configuration Checklist

### Pre-Production Checklist

- [ ] **Database User**: Created with minimal required privileges
- [ ] **File Permissions**: All files secured with proper ownership/permissions
- [ ] **Service User**: Running as non-root user with restricted capabilities
- [ ] **Network**: Database connections use localhost or secure networks
- [ ] **Encryption**: Backup files encrypted at rest and in transit
- [ ] **Credentials**: No plaintext passwords in configuration files
- [ ] **Logging**: Security events logged and monitored
- [ ] **Monitoring**: Metrics secured and alerts configured
- [ ] **Updates**: System and dependencies up to date

### Runtime Security Checks

```bash
# Verify service security
sudo systemctl show tenangdb.service | grep -E "(User|Group|NoNewPrivileges|ProtectSystem)"

# Check file permissions
ls -la /etc/tenangdb/
ls -la /var/backups/tenangdb/

# Verify database permissions
mysql -u tenangdb_backup -p -e "SHOW GRANTS;"

# Test backup encryption
file /var/backups/tenangdb/latest/*.sql.gz
```

## 🚨 Incident Response

### 1. Security Breach Response

**Immediate Actions:**
1. **Stop the service**: `sudo systemctl stop tenangdb.service`
2. **Isolate the system**: Review network connections
3. **Check logs**: `sudo journalctl -u tenangdb.service --since "1 hour ago"`
4. **Verify backup integrity**: Check recent backup files
5. **Rotate credentials**: Change database passwords immediately

### 2. Recovery Procedures

**Backup Verification:**
```bash
# Verify backup integrity
tenangdb verify --backup-path /var/backups/tenangdb/
mysqldump --single-transaction --routines --triggers database_name | gzip > test_backup.sql.gz
```

**Emergency Restore:**
```bash
# Emergency restore from secure backup
tenangdb restore --backup-path /var/backups/tenangdb/verified_backup/
```

## 📞 Security Contacts

### Reporting Security Issues

If you discover a security vulnerability in TenangDB:

1. **DO NOT** open a public GitHub issue
2. Email security concerns to: [security@tenangdb.com] (if available)
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact assessment
   - Suggested fix (if any)

### Security Updates

- Subscribe to security announcements: [GitHub Releases](https://github.com/abdullahainun/tenangdb/releases)
- Monitor the repository for security patches
- Enable automated dependency updates

## 📚 Additional Resources

### Security Standards & Compliance

- **OWASP Database Security**: [Database Security Guidelines](https://owasp.org/www-project-database-security/)
- **MySQL Security**: [MySQL Security Best Practices](https://dev.mysql.com/doc/refman/8.0/en/security-guidelines.html)
- **Systemd Security**: [Systemd Service Hardening](https://www.freedesktop.org/software/systemd/man/systemd.exec.html)

### Tools & Resources

- **Security Scanner**: `lynis` for system security auditing
- **File Integrity**: `aide` for file integrity monitoring
- **Network Security**: `fail2ban` for intrusion prevention
- **Backup Testing**: Regular restore testing procedures

---

## 🔐 Security Best Practices Summary

1. **Principle of Least Privilege**: Grant minimal necessary permissions
2. **Defense in Depth**: Multiple layers of security controls
3. **Regular Updates**: Keep all components updated
4. **Monitoring**: Continuous security monitoring and alerting
5. **Encryption**: Encrypt data at rest and in transit
6. **Access Control**: Strict file and network permissions
7. **Audit Trails**: Comprehensive logging for security events
8. **Incident Response**: Prepared procedures for security incidents

**Remember**: Security is an ongoing process, not a one-time setup. Regularly review and update your security measures.

---

**Last Updated**: 2025-01-06
**Version**: 1.0
**Maintainer**: Abdullah Ainun Najib