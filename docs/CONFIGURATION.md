# Configuration Reference

## Quick Start

```yaml
# config.yaml
database:
  host: mysql
  port: 3306
  username: backup_user
  password: "your_password"
  timeout: 30

backup:
  databases:
    - my_database
  directory: /backups

upload:
  enabled: false

logging:
  level: info

cleanup:
  enabled: false

metrics:
  enabled: false
```

## Full Options

### database

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `host` | string | `localhost` | MySQL hostname |
| `port` | int | `3306` | MySQL port |
| `username` | string | — | MySQL username (required) |
| `password` | string | — | MySQL password (required) |
| `timeout` | int | `30` | Connection timeout in seconds |
| `mysqldump_path` | string | auto-detect | Path to mysqldump binary |
| `mysql_path` | string | auto-detect | Path to mysql client binary |
| `mydumper` | — | — | Mydumper config (see below) |

#### database.mydumper

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `false` | Enable mydumper for backups |
| `binary_path` | string | auto-detect | Path to mydumper binary |
| `defaults_file` | string | — | Path to my.cnf defaults file |
| `threads` | int | `4` | Number of parallel threads |
| `chunk_filesize` | int | `100` | Chunk size in MB |
| `compress_method` | string | `gzip` | Compression: `gzip`, `lz4`, or empty |
| `build_empty_files` | bool | `false` | Create empty files for empty tables |
| `use_defer` | bool | `true` | Use deferred inserts (deprecated) |
| `single_table` | bool | `false` | Single table mode (deprecated) |
| `no_schemas` | bool | `false` | Skip schema dump |
| `no_data` | bool | `false` | Skip data dump |
| `myloader` | — | — | Myloader config (see below) |

##### database.mydumper.myloader

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `false` | Enable myloader for restore |
| `binary_path` | string | auto-detect | Path to myloader binary |
| `defaults_file` | string | — | Path to my.cnf defaults file |
| `threads` | int | `4` | Number of restore threads |

### backup

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `directory` | string | platform-specific | Backup storage directory |
| `databases` | []string | `[]` | List of databases to backup |
| `batch_size` | int | `5` | Databases per batch |
| `concurrency` | int | `3` | Max concurrent backups |
| `timeout` | duration | `30m` | Per-backup timeout |
| `retry_count` | int | `3` | Failed backup retry count |
| `retry_delay` | duration | `10s` | Delay between retries |
| `check_last_backup_time` | bool | `true` | Skip backup if run too recently |
| `min_backup_interval` | duration | `1h` | Minimum time between backups |
| `skip_confirmation` | bool | `false` | Skip confirmation prompts |

#### backup.compression

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `false` | Enable compression |
| `format` | string | `tar.gz` | Format: `tar.gz`, `tar.zst`, `tar.xz` |
| `level` | int | `6` | Compression level (1-9) |
| `keep_original` | bool | `true` | Keep uncompressed backup locally |

### upload

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `false` | Enable cloud upload |
| `rclone_path` | string | auto-detect | Path to rclone binary |
| `rclone_config_path` | string | `~/.config/rclone/rclone.conf` | Rclone config path |
| `destination` | string | — | Remote destination (e.g. `remote:path`) |
| `timeout` | int | `300` | Upload timeout in seconds |
| `retry_count` | int | `3` | Failed upload retry count |

### logging

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `level` | string | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `format` | string | `clean` | Output format: `clean`, `json` |
| `file_format` | string | `text` | File format: `text`, `json` |
| `file_path` | string | platform-specific | Log file path |

### cleanup

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `false` | Enable cleanup |
| `cleanup_uploaded_files` | bool | `true` | Clean up files after upload |
| `remote_retention_days` | int | `30` | Remote file retention days |
| `weekend_only` | bool | `true` | Only run cleanup on weekends |
| `age_based_cleanup` | bool | `false` | Enable age-based cleanup |
| `max_age_days` | int | `7` | Delete files older than N days |
| `verify_cloud_exists` | bool | `true` | Verify file exists in cloud before deleting |
| `databases` | []string | `[]` | Filter cleanup to specific databases |

### metrics

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | `false` | Enable Prometheus metrics |
| `port` | string | `8080` | Metrics HTTP port |
| `storage_path` | string | platform-specific | Persistent metrics file path |

## Example: Full Config

```yaml
database:
  host: mysql
  port: 3306
  username: backup_user
  password: "secure_password"
  timeout: 30
  mydumper:
    enabled: true
    threads: 4
    chunk_filesize: 100
    compress_method: gzip
    myloader:
      enabled: true
      threads: 4

backup:
  directory: /backups
  databases:
    - app_db
    - analytics_db
  batch_size: 5
  concurrency: 3
  timeout: 30m
  retry_count: 3
  retry_delay: 10s
  check_last_backup_time: true
  min_backup_interval: 1h
  compression:
    enabled: true
    format: tar.gz
    level: 6
    keep_original: true

upload:
  enabled: true
  destination: "s3:tenangdb-backups"
  timeout: 300
  retry_count: 3

logging:
  level: info
  format: clean
  file_path: /logs/tenangdb.log

cleanup:
  enabled: true
  age_based_cleanup: true
  max_age_days: 7
  weekend_only: true

metrics:
  enabled: true
  port: "9090"
```
