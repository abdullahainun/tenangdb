# 🔧 Commands Reference

Complete reference for all TenangDB commands and options.

## 📋 Command Overview

```bash
tenangdb [command] [options]
```

### Available Commands
- `init` - Interactive setup wizard (NEW!)
- `backup` - Run database backup (default)
- `restore` - Restore database from backup
- `cleanup` - Clean up old backup files
- `config` - Show configuration information
- `version` - Show version information
- `help` - Show help information

## 🧙‍♂️ Init Command (NEW!)

**The easiest way to set up TenangDB**

### Basic Usage
```bash
# Interactive setup wizard
tenangdb init

# Setup + deploy as systemd service
tenangdb init --deploy-systemd

# Custom config location
tenangdb init --config /etc/tenangdb/config.yaml

# Force overwrite existing config
tenangdb init --force
```

### Options
| Option | Description | Default |
|--------|-------------|---------|
| `--config` | Config file path (auto-discovery if not specified) | Auto-detect |
| `--deploy-systemd` | Automatically deploy as systemd service | `false` |
| `--systemd-user` | Systemd service user | `tenangdb` |
| `--force` | Overwrite existing config without confirmation | `false` |

### What Init Does
- ✅ **Dependency Check**: Validates mydumper, mysql, rclone availability
- ✅ **Database Testing**: Tests connection with provided credentials  
- ✅ **Smart Config**: Generates optimized config with privilege-aware paths
- ✅ **Directory Setup**: Creates backup, log, and metrics directories with proper ownership
- ✅ **Systemd Deploy**: (Optional) Installs and enables systemd services without MySQL dependency
- ✅ **Security Setup**: User isolation, proper permissions, root-owned config directory

### Examples
```bash
# Basic setup for development
tenangdb init --config ~/tenangdb-config.yaml

# Production deployment with systemd
sudo tenangdb init --deploy-systemd

# Multiple environments
sudo tenangdb init --deploy-systemd --config /etc/tenangdb/prod.yaml
sudo tenangdb init --deploy-systemd --config /etc/tenangdb/staging.yaml \
  --systemd-user tenangdb-staging
```

## 🔄 Default Backup Command

### Confirmation Feature

TenangDB now includes an interactive confirmation prompt before running backups. This shows:

```
📋 Backup Summary
================

💾 Databases to backup:
  1. app_db
  2. user_db
  3. logs_db

📁 Backup directory: /home/user/backups
☁️  Upload enabled: minio
   Rclone config: /home/user/.config/rclone/rclone.conf

⚙️  Options:
   Concurrency: 2
   Batch size: 5

Do you want to proceed with backup? [y/N]: 
```

**Skip confirmation:**
- `--yes` or `-y`: Skip all prompts (for automated/cron jobs)
- `--force`: Skip frequency checks and confirmations
- Config: `skip_confirmation: true`

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
| `--force` | Skip backup frequency confirmation prompts | `false` |
| `--yes, -y` | Skip all confirmation prompts (automated mode) | `false` |

### Examples
```bash
# Backup specific databases
./tenangdb backup --databases app_db,user_db --config config.yaml

# Debug mode with verbose output
./tenangdb backup --log-level trace --config config.yaml

# Test configuration without running backup
./tenangdb backup --dry-run --config config.yaml

# Skip confirmation prompts for automated mode
./tenangdb backup --yes --config config.yaml

# Force backup without frequency checks
./tenangdb backup --force --config config.yaml
```

## 🚀 Restore Command

### Confirmation Feature

TenangDB restore command includes a critical safety confirmation to prevent accidental database overwrites:

```
⚠️  Database Restore Warning
===========================

🎯 Target database: production_db
📂 Backup source: /backup/prod-2025-01-10_10-30-15/
📅 Backup date: 2025-01-10 10:30:15
📊 Backup size: 125.3 MB

🔴 **DANGER ZONE** 🔴
⚠️  WARNING: Database 'production_db' already exists!
⚠️  This operation will COMPLETELY OVERWRITE the existing database!
⚠️  ALL existing data in 'production_db' will be PERMANENTLY LOST!
⚠️  This action CANNOT be undone!

💡 Recommendation: Create a backup of the existing database first
   tenangdb backup --databases production_db

Are you ABSOLUTELY SURE you want to OVERWRITE database 'production_db'? [y/N]: 
```

**For new databases:**
```
✅ Database 'new_db' does not exist - will be created
Do you want to create and restore database 'new_db'? [y/N]: 
```

**Skip confirmation:**
- `--yes` or `-y`: Skip confirmation prompts (for automated mode)
- Essential for restore scripts and automation

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
| `--yes, -y` | Skip confirmation prompts (for automated mode) | ❌ |

### Examples
```bash
# Restore with different name
./tenangdb restore --backup-path /backup/prod_db-2025-07-05_10-30-15 --target-database prod_db_restored

# Restore from cloud backup (download first)
rclone copy minio:backups/db-2025-07-05_10-30-15 /tmp/restore/
./tenangdb restore --backup-path /tmp/restore/db-2025-07-05_10-30-15 --target-database restored_db

# Automated restore (skip confirmation)
./tenangdb restore --backup-path /backup/db-2025-07-05_10-30-15 --target-database restored_db --yes

# Restore from compressed backup (auto-decompression)
./tenangdb restore --backup-path /backup/db-2025-07-05_10-30-15.tar.gz --target-database restored_db
```

## 🧹 Cleanup Command

### Confirmation Feature

TenangDB cleanup command includes a safety confirmation prompt to prevent accidental file deletion:

```
📋 Cleanup Summary
=================

🗂️ Files to delete:
  1. /backups/app_db-2025-01-10_10-30-15/ (45.2 MB)
  2. /backups/logs_db-2025-01-09_10-30-15/ (128.7 MB)
  3. /backups/user_db-2025-01-08_10-30-15/ (23.1 MB)

📁 Total files: 3
📊 Total space to free: 196.9 MB
⏰ Max age: 7 days

⚠️  WARNING: This action cannot be undone!
⚠️  Deleted backup files cannot be recovered!

Do you want to proceed with cleanup? [y/N]: 
```

**Skip confirmation:**
- `--yes` or `-y`: Skip confirmation prompts (for automated/cron jobs)
- Useful for scheduled cleanup operations

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
| `--yes, -y` | Skip confirmation prompts (for automated mode) | `false` |

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

# Skip confirmation prompts for automated mode
./tenangdb cleanup --yes --force --config config.yaml
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

### Error Handling & Status Reporting

**TenangDB provides detailed status reporting for backup operations:**

```bash
# Successful backup (all databases)
✅ All backup process completed successfully

# Partial failure (some databases failed)
⚠️  Backup process completed with partial success (successful: 2, failed: 1, total: 3)

# Total failure (all databases failed)
❌ All database backups failed (failed: 3)
```

**Common Scenarios:**

```bash
# Permission issues
./tenangdb backup --log-level debug  # Shows detailed permission errors

# Port conflicts for metrics
# Edit ~/.config/tenangdb/config.yaml:
metrics:
  enabled: false  # Or change port: "8081"

# Non-root user setup
./tenangdb config  # Shows which config file is being used
# TenangDB automatically selects user-appropriate config paths

# Systemd deployment issues
sudo ./tenangdb init --deploy-systemd --force  # Redeploy with latest fixes
```