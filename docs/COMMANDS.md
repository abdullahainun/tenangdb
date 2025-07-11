# 🔧 Commands Reference

Complete reference for all TenangDB commands and options.

## 📋 Command Overview

```bash
tenangdb [command] [options]
```

### Available Commands
- Default (no subcommand) - Run database backup
- `restore` - Restore database from backup
- `cleanup` - Clean up old backup files
- `exporter` - Start Prometheus metrics exporter
- `version` - Show version information
- `help` - Show help information

## 🔄 Default Backup Command

### Basic Usage
```bash
# Backup all configured databases
./tenangdb backup --config config.yaml

# Backup with specific log level
./tenangdb backup --config config.yaml --log-level debug

# Dry run (preview only)
./tenangdb backup --config config.yaml --dry-run
```

### Options
| Option | Description | Default |
|--------|-------------|---------|
| `--config` | Path to configuration file | `config.yaml` |
| `--log-level` | Log level (panic, fatal, error, warn, info, debug, trace) | `info` |
| `--dry-run` | Preview actions without executing | `false` |
| `--databases` | Comma-separated list of databases to backup | All from config |

### Examples
```bash
# Backup specific databases
./tenangdb --databases app_db,user_db --config config.yaml

# Debug mode with verbose output
./tenangdb --log-level trace --config config.yaml

# Test configuration without running backup
./tenangdb --dry-run --config config.yaml
```

## 🚀 Restore Command

### Basic Usage
```bash
# Restore database from backup
./tenangdb restore --backup-path /backup/db-2025-07-05_10-30-15 --target-database restored_db

# Restore with custom config
./tenangdb restore --backup-path /backup/db-2025-07-05_10-30-15 --target-database restored_db --config config.yaml
```

### Options
| Option | Description | Required |
|--------|-------------|----------|
| `--backup-path` | Path to backup directory | ✅ |
| `--target-database` | Target database name | ✅ |
| `--config` | Path to configuration file | ❌ |
| `--log-level` | Log level | ❌ |
| `--dry-run` | Preview actions without executing | ❌ |

### Examples
```bash
# Restore with different name
./tenangdb restore --backup-path /backup/prod_db-2025-07-05_10-30-15 --target-database prod_db_restored

# Restore from cloud backup (download first)
rclone copy minio:backups/db-2025-07-05_10-30-15 /tmp/restore/
./tenangdb restore --backup-path /tmp/restore/db-2025-07-05_10-30-15 --target-database restored_db
```

## 🧹 Cleanup Command

### Basic Usage
```bash
# Cleanup old backups
./tenangdb cleanup --config config.yaml

# Force cleanup (bypass weekend-only restriction)
./tenangdb cleanup --force --config config.yaml

# Preview cleanup actions
./tenangdb cleanup --dry-run --config config.yaml
```

### Options
| Option | Description | Default |
|--------|-------------|---------|
| `--config` | Path to configuration file | `config.yaml` |
| `--force` | Force cleanup (bypass weekend-only) | `false` |
| `--dry-run` | Preview actions without executing | `false` |
| `--databases` | Comma-separated list of databases to clean | All from config |
| `--max-age-days` | Override max age from config | From config |
| `--log-level` | Log level | `info` |

### Examples
```bash
# Cleanup specific databases
./tenangdb cleanup --databases app_db,logs_db --force --config config.yaml

# Cleanup with custom age limit
./tenangdb cleanup --max-age-days 3 --force --config config.yaml

# Preview cleanup for all databases
./tenangdb cleanup --dry-run --config config.yaml

# Force cleanup with debug logging
./tenangdb cleanup --force --log-level debug --config config.yaml
```

## 📊 Version & Help

### Version Information
```bash
# Show version
./tenangdb version

# Show build information
./tenangdb version --build-info
```

### Help
```bash
# General help
./tenangdb help

# Command-specific help
./tenangdb --help
./tenangdb restore --help
./tenangdb cleanup --help
```

## 🔧 Global Options

These options work with all commands:

| Option | Description | Default |
|--------|-------------|---------|
| `--config` | Configuration file path | `config.yaml` |
| `--log-level` | Logging level | `info` |
| `--help` | Show help for command | - |
| `--version` | Show version information | - |

## 📋 Exit Codes

| Code | Description |
|------|-------------|
| `0` | Success |
| `1` | General error |
| `2` | Configuration error |
| `3` | Database connection error |
| `4` | Backup/restore operation failed |
| `5` | Cloud upload failed |
| `6` | Cleanup operation failed |

## 💡 Usage Tips

### Production Workflows
```bash
# Daily backup with upload
./tenangdb backup --config /etc/tenangdb/config.yaml

# Weekly cleanup
./tenangdb cleanup --force --config /etc/tenangdb/config.yaml

# Monthly restore test
./tenangdb restore --backup-path /backup/latest --target-database test_restore
```

### Development Workflows
```bash
# Quick backup for development
./tenangdb --databases dev_db --log-level debug

# Restore from production backup
./tenangdb restore --backup-path /backup/prod-2025-07-05 --target-database dev_db_copy
```

### Monitoring Integration
```bash
# Export metrics to file
curl -s localhost:8080/metrics > /tmp/tenangdb_metrics.txt

# Check backup status via metrics
curl -s localhost:8080/metrics | grep tenangdb_backup_status
```

## 🆘 Troubleshooting Commands

### Debug Connection Issues
```bash
# Test with maximum logging
./tenangdb --log-level trace --dry-run

# Test specific database
./tenangdb --databases test_db --log-level debug
```

### Verify Configuration
```bash
# Validate config file
./tenangdb backup --config config.yaml --dry-run

# Test with different config
./tenangdb backup --config /path/to/test-config.yaml --dry-run
```

### Check Dependencies
```bash
# Test system dependencies
make test-deps

# Manual dependency check
mydumper --version
myloader --version
rclone version
```