# 🔐 Security Guide

TenangDB implements multiple layers of security to protect your database backups and system.

## 🛡️ System Security

### Dedicated User Account
TenangDB runs under a dedicated `tenangdb` user (not root) with:
- **System user**: `useradd -r` (no login capability)
- **No shell**: `/bin/false` (prevents interactive login)
- **Minimal permissions**: Only access to required directories
- **Home directory**: `/opt/tenangdb` (isolated environment)

### Directory Permissions
```bash
# Application files
/opt/tenangdb/          → tenangdb:tenangdb (755)
/etc/tenangdb/          → tenangdb:tenangdb (750)
/var/log/tenangdb/      → tenangdb:tenangdb (755)
/backup/                → tenangdb:tenangdb (755)
```

## 🔒 Systemd Security Hardening

### Enabled Security Features
```ini
# Prevent privilege escalation
NoNewPrivileges=true

# Isolated temporary directory
PrivateTmp=true

# Read-only system directories
ProtectSystem=strict

# No access to user home directories
ProtectHome=true

# Specific write access only
ReadWritePaths=/backup /var/log/tenangdb /tmp

# Protect kernel interfaces
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
```

### Benefits
- **Privilege Isolation**: Cannot escalate to root
- **File System Protection**: Limited write access
- **Kernel Protection**: Cannot modify kernel parameters
- **Process Isolation**: Isolated from other services

## 🔑 Credential Management

### MySQL Credentials
**❌ DON'T:** Store passwords in config files
```yaml
# AVOID THIS
database:
  password: "plaintext_password"  # Visible in process list
```

**✅ DO:** Use MySQL defaults files
```bash
# Create ~/.my.cnf for tenangdb user
sudo -u tenangdb cat > /opt/tenangdb/.my.cnf << 'EOF'
[client]
host=192.168.1.100
port=3306
user=backup_user
password=secure_password

[mydumper]
single-transaction=true
routines=true
triggers=true
events=true
EOF

# Secure the file
sudo chmod 600 /opt/tenangdb/.my.cnf
```

### Configuration in TenangDB
```yaml
database:
  mydumper:
    defaults_file: /opt/tenangdb/.my.cnf  # Secure credential storage
```

## 🌐 Network Security

### Database User Permissions
Create a dedicated backup user with minimal privileges:
```sql
-- Create backup user
CREATE USER 'backup_user'@'%' IDENTIFIED BY 'secure_password';

-- Grant only necessary permissions
GRANT SELECT, LOCK TABLES, SHOW VIEW ON *.* TO 'backup_user'@'%';
GRANT RELOAD, PROCESS ON *.* TO 'backup_user'@'%';
FLUSH PRIVILEGES;
```

### rclone Security
```bash
# Secure rclone config
sudo -u tenangdb rclone config  # Configure as tenangdb user
sudo chmod 600 /opt/tenangdb/.config/rclone/rclone.conf
```

## 📊 Audit & Monitoring

### Log Security
- **Structured logging**: JSON format prevents log injection
- **No sensitive data**: Passwords never logged
- **Rotation**: Automatic log rotation prevents disk filling
- **Centralized**: All logs go to systemd journal

### Monitoring Access
```bash
# Metrics endpoint (if enabled)
# Bind to localhost only
metrics:
  listen_addr: "127.0.0.1:8080"  # Not 0.0.0.0:8080
```

## 🚨 Security Checklist

### Before Production Deployment
- [ ] Created dedicated `tenangdb` user
- [ ] Set proper file permissions (750/755, not 777)
- [ ] Configured MySQL credentials in defaults file
- [ ] Enabled systemd security hardening
- [ ] Created minimal-privilege database user
- [ ] Secured rclone configuration
- [ ] Tested backup/restore as non-root user
- [ ] Configured log rotation
- [ ] Restricted metrics endpoint to localhost

### Regular Security Maintenance
- [ ] Rotate database passwords quarterly
- [ ] Review systemd service permissions
- [ ] Monitor backup logs for suspicious activity
- [ ] Update rclone and mydumper regularly
- [ ] Check file permissions monthly

## 🆘 Security Issues

### Common Vulnerabilities to Avoid
1. **Running as root**: Use dedicated user
2. **World-readable config**: Proper file permissions
3. **Passwords in logs**: Use defaults files
4. **Overprivileged DB user**: Minimal permissions only
5. **Unencrypted cloud storage**: Enable encryption in rclone

### Reporting Security Issues
For security vulnerabilities, please:
1. **DO NOT** open public GitHub issues
2. Email security concerns privately
3. Include reproduction steps
4. Allow time for patches before disclosure

## 📚 References

- [Systemd Security Features](https://systemd.io/SECURITY/)
- [MySQL User Privileges](https://dev.mysql.com/doc/refman/8.0/en/privileges-provided.html)
- [rclone Security](https://rclone.org/docs/#configuration-encryption)
- [Go Security Best Practices](https://golang.org/security/)