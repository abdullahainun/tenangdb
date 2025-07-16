# Backup Frequency Check Feature

This feature prevents accidental or too frequent backups by checking when the last backup was performed.

## Configuration

Add the following configuration to `config.yaml`:

```yaml
backup:
  # ... other backup configurations
  
  # Backup frequency check configuration
  check_last_backup_time: true      # Enable backup frequency checking
  min_backup_interval: 1h           # Minimum interval between backups
  skip_confirmation: false          # Set to true to skip confirmation prompts
```

## Usage

### 1. Normal Backup (with checking)
```bash
./tenangdb backup --config config.yaml
```

If the last backup is too recent (less than `min_backup_interval`), a confirmation prompt will appear:
```
⚠️  last backup was 10 minutes ago, are you sure you want to run backup again?
Continue backup? (y/n/force): 
```

### 2. Force Backup (skip confirmation)
```bash
./tenangdb backup --config config.yaml --force
```

### 3. Confirmation Options
- `y` or `yes` - Continue backup
- `n` or `no` - Cancel backup
- `force` or `f` - Force backup without confirmation

## Interval Configuration

You can set the minimum interval with various formats:

```yaml
backup:
  min_backup_interval: 30m    # 30 minutes
  min_backup_interval: 1h     # 1 hour
  min_backup_interval: 2h30m  # 2 hours 30 minutes
  min_backup_interval: 24h    # 1 day
```

## Tracking File

The system creates a `.tenangdb_backup_tracking_*.json` file in a persistent location to track last backup times:

### Location Strategy:
- **macOS**: `~/Library/Application Support/TenangDB/`
- **Linux (regular)**: `~/.local/share/tenangdb/`
- **Docker container**: `/tmp/tenangdb/`
- **Fallback**: Backup directory

### Docker Usage:
For Docker containers, mount `/tmp` volume to persist tracking data:
```bash
docker run -v $(pwd)/tmp:/tmp ghcr.io/abdullahainun/tenangdb:latest backup
```

```json
{
  "database_backups": {
    "testdb1": "2025-07-08T13:45:30Z",
    "testdb2": "2025-07-08T13:45:45Z"
  },
  "last_updated": "2025-07-08T13:45:45Z"
}
```

## Disable Feature

To disable this feature, set:

```yaml
backup:
  check_last_backup_time: false
```

## Example Workflow

1. **First backup** - Runs normally without confirmation
2. **Second backup within 1 hour** - Shows confirmation prompt
3. **User selects 'y'** - Backup continues
4. **Third backup with --force** - Runs immediately without confirmation
5. **Backup after 1 hour** - Runs normally without confirmation

## Log Messages

- `✅ Backup confirmed by user` - User approved backup
- `🔄 Backup forced by user` - User forced backup
- `❌ Backup cancelled by user` - User cancelled backup
- `⏭️ dbname backup skipped` - Backup was skipped