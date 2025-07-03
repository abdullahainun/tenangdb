# Database Backup Tool

A robust, production-ready database backup tool written in Go that addresses the limitations of traditional bash scripts. This tool provides batch processing, graceful error handling, cloud uploads, and comprehensive logging.

## Features

- **Dual Backup Engine**: Support for both mydumper (parallel) and mysqldump (traditional)
- **Batch Processing**: Process databases in configurable batches with controlled concurrency
- **Parallel Backups**: Mydumper provides multi-threaded backups for faster performance
- **Graceful Error Handling**: Continue processing even if individual databases fail
- **Cloud Upload**: Automatic upload to cloud storage using rclone
- **Retry Logic**: Configurable retry attempts for both backup and upload operations
- **Structured Logging**: JSON-formatted logs with detailed statistics
- **Systemd Integration**: Run as a systemd service with timer support
- **Resource Management**: Controlled concurrency to maintain server stability
- **Cleanup**: Automatic cleanup of old backup files (local and remote)
- **Compression**: Built-in compression support with mydumper (gzip/lz4)

## Project Structure

```
db-backup-tool/
├── cmd/
│   └── main.go                 # Application entry point
├── internal/
│   ├── backup/                 # Backup service implementation
│   │   ├── service.go         # Main backup service
│   │   └── cleanup.go         # Cleanup functionality
│   ├── config/                 # Configuration management
│   │   └── config.go
│   ├── logger/                 # Logging utilities
│   │   └── logger.go
│   └── upload/                 # Cloud upload service
│       └── service.go
├── pkg/
│   └── database/              # Database client
│       └── client.go
├── configs/
│   └── config.yaml            # Default configuration
├── scripts/
│   ├── db-backup.service      # Systemd service file
│   ├── db-backup.timer        # Systemd timer file
│   └── install.sh             # Installation script
├── Makefile                   # Build and deployment tasks
└── README.md
```

## Installation

1. **Prerequisites**:
   - Go 1.21 or later
   - MySQL client tools (`mysqldump` and/or `mydumper`)
   - rclone (for cloud uploads)
   - For mydumper: Install from your package manager or compile from source

2. **Build and Install**:
   ```bash
   git clone <repository-url>
   cd db-backup-tool
   make deps
   make install
   ```

3. **Configuration**:
   Edit `/etc/db-backup-tool/config.yaml` to match your environment:
   ```yaml
   database:
     host: localhost
     port: 3306
     username: root
     password: "your-password"
   
   backup:
     directory: /mnt/hdd/backup/databases
     databases:
       - database1
       - database2
     batch_size: 10
     concurrency: 3
   
   upload:
     enabled: true
     destination: "remote:path/to/backups/"
   ```

## Usage

### Manual Execution
```bash
# Run backup once
sudo systemctl start db-backup.service

# Check status
sudo systemctl status db-backup.service

# View logs
sudo journalctl -u db-backup.service -f
```

### Scheduled Execution
```bash
# Enable daily backups
sudo systemctl enable db-backup.timer
sudo systemctl start db-backup.timer

# Enable weekend cleanup
sudo systemctl enable db-backup-cleanup.timer
sudo systemctl start db-backup-cleanup.timer

# Check timer status
sudo systemctl status db-backup.timer
sudo systemctl status db-backup-cleanup.timer

# List active timers
sudo systemctl list-timers
```

### Direct Binary Execution
```bash
# Run backup
./db-backup-tool --config configs/config.yaml --log-level info

# Run cleanup (weekend only)
./db-backup-tool cleanup --config configs/config.yaml

# Test cleanup (dry-run)
./db-backup-tool cleanup --dry-run --config configs/config.yaml
```

## Configuration

### Database Configuration
```yaml
database:
  host: localhost          # MySQL host
  port: 3306              # MySQL port
  username: root          # MySQL username
  password: ""            # MySQL password
  timeout: 30             # Connection timeout in seconds
  mydumper:                # Mydumper configuration (optional)
    enabled: true          # Enable mydumper (faster, parallel backups)
    binary_path: /usr/bin/mydumper  # Path to mydumper binary
    threads: 4             # Number of threads for parallel processing
    chunk_filesize: 100    # Chunk size in MB
    compress_method: gzip  # Compression method (gzip, lz4, or empty)
    build_empty_files: false  # Build empty files for empty tables
    use_defer: true        # Use deferred transactions
    single_table: false    # Single table mode
    no_schemas: false      # Skip schema dump
    no_data: false         # Skip data dump
```

### Backup Configuration
```yaml
backup:
  directory: /backups           # Local backup directory
  databases:                    # List of databases to backup
    - db1
    - db2
  batch_size: 10               # Databases per batch
  concurrency: 3               # Concurrent backups per batch
  timeout: 30m                 # Backup timeout
  retry_count: 3               # Retry attempts
  retry_delay: 10s             # Delay between retries
```

### Upload Configuration
```yaml
upload:
  enabled: true                     # Enable cloud upload
  rclone_path: /usr/bin/rclone     # Path to rclone binary
  destination: "remote:path/"       # Rclone destination
  timeout: 300                     # Upload timeout in seconds
  retry_count: 3                   # Upload retry attempts
```

### Logging Configuration
```yaml
logging:
  level: info                      # Log level (debug, info, warn, error)
  format: json                     # Log format
  file_path: /var/log/backup.log   # Log file path
```

### Cleanup Configuration
```yaml
cleanup:
  enabled: true                    # Enable cleanup functionality
  cleanup_uploaded_files: true     # Cleanup local files after successful upload
  remote_retention_days: 3         # Keep remote files for 3 days
  weekend_only: true              # Only run cleanup on weekends
```

## Improvements Over Bash Script

1. **Fault Tolerance**: Individual database failures don't stop the entire process
2. **Resource Management**: Controlled concurrency prevents server overload
3. **Batch Processing**: Process 300+ databases efficiently in manageable batches
4. **Better Error Handling**: Detailed error reporting and retry logic
5. **Structured Logging**: JSON logs with comprehensive statistics
6. **Graceful Shutdown**: Proper signal handling for clean shutdowns
7. **Configuration Management**: YAML-based configuration with validation
8. **Systemd Integration**: Proper service management and scheduling
9. **Smart Cleanup**: Only removes files after successful upload, weekend-only schedule
10. **Parallel Processing**: Mydumper provides 3-5x faster backups for large databases
11. **Automatic Compression**: Built-in compression reduces storage requirements
12. **Flexible Output**: Choose between single-file (mysqldump) or multi-file (mydumper) output

## Monitoring

The tool provides comprehensive logging and statistics:

```json
{
  "level": "info",
  "msg": "Backup process completed",
  "statistics": {
    "total_databases": 300,
    "successful_backups": 298,
    "failed_backups": 2,
    "successful_uploads": 296,
    "failed_uploads": 2,
    "duration": "45m32s",
    "start_time": "2024-01-01T10:00:00Z",
    "end_time": "2024-01-01T10:45:32Z"
  }
}
```

## Development

### Building
```bash
make build           # Build binary
make build-prod      # Build for production
make test            # Run tests
make fmt             # Format code
make lint            # Lint code
```

### Testing
```bash
# Run with test configuration
./db-backup-tool --config configs/config.yaml --log-level debug
```

## Security

The systemd service runs with security hardening:
- `NoNewPrivileges=true`
- `PrivateTmp=true`
- `ProtectSystem=strict`
- `ReadWritePaths` limited to backup directories

## Troubleshooting

### Common Issues

1. **Permission Denied**: Ensure the service user has read/write access to backup directories
2. **MySQL Connection Failed**: Check database credentials and network connectivity
3. **Rclone Upload Failed**: Verify rclone configuration and remote access
4. **Backup File Empty**: Check MySQL user permissions and disk space

### Logs
```bash
# Service logs
sudo journalctl -u db-backup.service

# Application logs
sudo tail -f /var/log/db-backup.log
```

## License

This project is licensed under the MIT License.