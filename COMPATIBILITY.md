# TenangDB Compatibility Guide

🔍 **Complete compatibility information for TenangDB backup solution**

This document provides comprehensive compatibility information for TenangDB, including supported MySQL versions, operating systems, dependencies, and configuration requirements.

## 🗄️ Database Compatibility

### MySQL Versions

| MySQL Version | mysqldump Support | mydumper Support | Notes |
|---------------|------------------|------------------|--------|
| 8.0.x         | ✅ Full           | ✅ Full          | Recommended version |
| 5.7.x         | ✅ Full           | ✅ Full          | Fully supported |
| 5.6.x         | ✅ Full           | ✅ Limited       | Legacy support |
| 5.5.x         | ⚠️ Limited        | ❌ Not supported | End of life |

### MariaDB Versions

| MariaDB Version | mysqldump Support | mydumper Support | Notes |
|----------------|------------------|------------------|--------|
| 10.11.x        | ✅ Full           | ✅ Full          | Current LTS |
| 10.6.x         | ✅ Full           | ✅ Full          | Previous LTS |
| 10.5.x         | ✅ Full           | ✅ Full          | Supported |
| 10.4.x         | ✅ Full           | ⚠️ Limited       | Legacy support |

### Database Engine Support

- **InnoDB**: Full support (recommended)
- **MyISAM**: Full support
- **Memory**: Limited support (data loss on restart)
- **Archive**: Read-only support

## 🖥️ Operating System Compatibility

### Linux Distributions

| Distribution | Version | Support Level | Notes |
|-------------|---------|---------------|--------|
| Ubuntu      | 20.04+ | ✅ Full        | Recommended |
| Ubuntu      | 18.04  | ✅ Full        | Supported |
| Debian      | 11+    | ✅ Full        | Recommended |
| Debian      | 10     | ✅ Full        | Supported |
| CentOS      | 8+     | ✅ Full        | Supported |
| CentOS      | 7      | ⚠️ Limited     | Legacy support |
| RHEL        | 8+     | ✅ Full        | Supported |
| RHEL        | 7      | ⚠️ Limited     | Legacy support |
| Alpine      | 3.15+  | ✅ Full        | Docker optimized |

### macOS

| macOS Version | Support Level | Notes |
|--------------|---------------|--------|
| macOS 13+    | ✅ Full        | Recommended |
| macOS 12     | ✅ Full        | Supported |
| macOS 11     | ⚠️ Limited     | Basic support |

### Windows

| Windows Version | Support Level | Notes |
|----------------|---------------|--------|
| Windows 11     | ⚠️ Limited     | WSL2 recommended |
| Windows 10     | ⚠️ Limited     | WSL2 recommended |

## 🔧 Backup Tools Compatibility

### mysqldump (Default)

**Availability**: Built-in with MySQL/MariaDB installation

**Required Parameters**:
- `--source-data=2` (MySQL 8.0+)
- `--master-data=2` (MySQL 5.7 and earlier)
- `--single-transaction` (for InnoDB)
- `--routines`, `--triggers`, `--events`

**Version Compatibility**:
- MySQL 8.0+: Full support with modern parameters
- MySQL 5.7: Full support with legacy parameters
- MariaDB 10.6+: Full support

### mydumper (Optional, High Performance)

**Installation Required**: 
```bash
# Ubuntu/Debian
sudo apt install mydumper

# CentOS/RHEL
sudo yum install mydumper

# macOS
brew install mydumper
```

**Version Requirements**:
- mydumper 0.12.0+: Recommended
- mydumper 0.10.0+: Supported
- mydumper 0.9.x: Limited support

**Performance Benefits**:
- Parallel processing
- Faster for large databases (>1GB)
- Better compression options

## 🏗️ Build Requirements

### Go Version

| Go Version | Support Level | Notes |
|-----------|---------------|--------|
| 1.23.x    | ✅ Full        | Current version |
| 1.22.x    | ✅ Full        | Supported |
| 1.21.x    | ⚠️ Limited     | Minimum supported |
| 1.20.x    | ❌ Not supported | End of life |

### Build Dependencies

**Required**:
- Go 1.21+
- git
- make

**Optional**:
- golangci-lint (for development)
- gosec (for security scanning)

## 📦 Runtime Dependencies

### Core Dependencies

| Tool | Required | Purpose | Installation |
|------|----------|---------|-------------|
| mysqldump | ✅ Yes | Default backup engine | Included with MySQL |
| mysql client | ✅ Yes | Database connection | `apt install mysql-client` |
| mydumper | ❌ No | High-performance backup | `apt install mydumper` |
| myloader | ❌ No | High-performance restore | `apt install mydumper` |
| rclone | ❌ No | Cloud upload | `curl https://rclone.org/install.sh \| sudo bash` |

### System Requirements

**Minimum**:
- RAM: 512MB
- CPU: 1 core
- Disk: 100MB (+ backup storage)

**Recommended**:
- RAM: 2GB+
- CPU: 2+ cores
- Disk: SSD recommended for backup storage

## ⚙️ Configuration Compatibility

### Minimum Configuration

```yaml
database:
  host: localhost
  port: 3306
  username: backup_user
  password: secure_password
  timeout: 30

backup:
  directory: /var/backups
  databases:
    - your_database
```

### Default Behaviors

| Setting | Default Value | Notes |
|---------|---------------|-------|
| `backup.engine` | mysqldump | Changed from mydumper |
| `metrics.enabled` | false | Changed from true |
| `logging.format` | text | Changed from json |
| `cleanup.enabled` | false | Changed from true |
| `upload.enabled` | false | Remains false |

## 🔧 Common Compatibility Issues

### MySQL Parameter Issues

**Problem**: `--master-data is deprecated` warning
**Solution**: Automatic migration to `--source-data=2` for MySQL 8.0+

**Problem**: `unknown option '--skip-ssl'` error
**Solution**: Removed invalid SSL parameter from mysqldump command

### Permission Issues

**Problem**: `/var/log/db-backup.log: permission denied`
**Solutions**:
1. Run as root: `sudo ./tenangdb`
2. Change log path in config: `logging.file_path: "./tenangdb.log"`
3. Create writable directory: `sudo mkdir -p /var/log/tenangdb && sudo chown $USER /var/log/tenangdb`

### Port Conflicts

**Problem**: `bind: address already in use` (port 8080)
**Solutions**:
1. Disable metrics: `metrics.enabled: false`
2. Change port: `metrics.port: "8081"`
3. Kill conflicting process: `sudo lsof -ti:8080 | xargs sudo kill`

## 🧪 Testing Compatibility

### Automated Testing

Run the dependency checker:
```bash
make test-deps
```

### Manual Testing

1. **Test database connection**:
   ```bash
   mysql -h hostname -u username -p
   ```

2. **Test backup tools**:
   ```bash
   mysqldump --version
   mydumper --version  # if installed
   ```

3. **Test configuration**:
   ```bash
   ./tenangdb backup --config config.yaml --dry-run
   ```

## 📊 Feature Matrix

| Feature | mysqldump | mydumper | Notes |
|---------|-----------|----------|--------|
| Parallel processing | ❌ | ✅ | mydumper advantage |
| Compression | ✅ | ✅ | Both support gzip |
| Consistency | ✅ | ✅ | Both support transactions |
| Restore speed | ⚠️ Slow | ✅ Fast | mydumper advantage |
| Availability | ✅ Built-in | ❌ External | mysqldump advantage |
| Memory usage | ✅ Low | ⚠️ Higher | mysqldump advantage |

## 📚 Version History

### v1.0.0 (Current)
- Default backup engine: mysqldump
- Metrics disabled by default
- Text logging by default
- Cleanup disabled by default
- Fixed MySQL 8.0+ parameter compatibility

### v0.9.0 (Previous)
- Default backup engine: mydumper
- Metrics enabled by default
- JSON logging by default
- Cleanup enabled by default

## 🚀 Migration Guide

### From v0.9.0 to v1.0.0

**Configuration Changes**:
- No configuration changes required
- New defaults apply automatically

**Behavior Changes**:
- Backup engine changed from mydumper to mysqldump
- Metrics server disabled by default
- Logging format changed to text
- Cleanup disabled by default

**Migration Steps**:
1. Rebuild binary: `make build`
2. Test with existing config: `./tenangdb backup --config config.yaml`
3. Enable features if needed:
   ```yaml
   database:
     mydumper:
       enabled: true  # For mydumper
   metrics:
     enabled: true    # For metrics
   cleanup:
     enabled: true    # For cleanup
   ```

## 🆘 Support Matrix

| Issue Type | Support Level | Contact |
|-----------|---------------|---------|
| Installation | ✅ Full | GitHub Issues |
| Configuration | ✅ Full | GitHub Issues |
| Bug Reports | ✅ Full | GitHub Issues |
| Feature Requests | ✅ Full | GitHub Issues |
| Performance | ⚠️ Limited | GitHub Issues |
| Custom Integrations | ❌ Community | GitHub Discussions |

---

**Last Updated**: 2025-07-08  
**TenangDB Version**: 1.0.0  
**Maintainer**: @abdullahainun