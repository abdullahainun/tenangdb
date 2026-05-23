# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
make build           # builds ./tenangdb binary
make build-exporter  # builds ./tenangdb-exporter binary
make build-all       # builds both
make build-prod      # CGO_ENABLED=0 static build for Linux

# Test
make test                                        # all tests
go test -v ./internal/backup/...                 # single package
go test -v -run TestCleanup ./internal/backup/  # single test

# Code quality
make fmt       # go fmt ./...
make lint      # golangci-lint run
make security  # gosec ./...

# Dependency checks
make check-deps   # verify mysqldump/mydumper/pg_dump/rclone present
make deps         # go mod tidy + download
```

## Architecture

Two binaries from one repo:
- `cmd/main.go` — main CLI (`tenangdb backup|cleanup|restore|init|config|version`)
- `cmd/tenangdb-exporter/main.go` — standalone Prometheus exporter (reads `metrics.json`, exposes `/metrics`)

### Backup execution flow

```
runBackup()
  → config.LoadConfig()          # platform-aware: /etc/tenangdb or ~/.config/tenangdb
  → backup.NewService()
      → database.NewClient()   # connects to MySQL/PostgreSQL based on config
      → upload.NewService()      # wraps rclone, only if upload.enabled
      → compression.NewCompressor()
  → service.Run()
      → processDatabasesBatch()  # respects batch_size + concurrency via semaphore
          → createBackupWithRetry()    # retry_count attempts
          → compressor.CompressBackup()
          → uploader.Upload()          # rclone copy
      → metricsStorage.Update*()      # write to metrics.json
```

### Database client architecture

`pkg/database/` uses a `DatabaseClient` interface with two implementations:

- **`MySQLClient`** (`client.go`): mysqldump + mydumper/myloader, including version-aware mydumper args (`isMydumperVersionCompatible()` detects v0.9.x legacy vs v0.19.x modern via `--help` output parsing).
- **`PostgreSQLClient`** (`postgres_client.go`): pg_dump/pg_restore/psql wrappers.

**Factory** — `NewClient(cfg)` at `client.go:58` dispatches based on `cfg.Type`:
- `"postgresql"` → `NewPostgreSQLClient(cfg)`
- default (including `"mysql"` or empty) → `NewMySQLClient(cfg)`

Config: `database.type` field (`mysql`|`postgresql`, default `mysql`).

### Metrics — two layers

1. **In-process Prometheus** (`internal/metrics/metrics.go`): counters/histograms registered via `metrics.Init()`. Must be called before any `metrics.Record*()` call. Currently only called in `runBackup()` — calling `runCleanup` or `runRestore` with `metrics.enabled: true` will panic.

2. **Persistent JSON** (`internal/metrics/storage.go`): atomic write via temp-file + rename to `metrics.json`. Read by the exporter binary between backup runs. Path is platform-aware via config defaults.

### Config discovery order

Non-root Linux: `~/.config/tenangdb/config.yaml` → `./config.yaml` → `/etc/tenangdb/config.yaml`
Root Linux: `/etc/tenangdb/config.yaml` → `~/.config/tenangdb/config.yaml` → `./config.yaml`
macOS paths swap in `~/Library/Application Support/TenangDB/`.

### Cleanup logic

`tenangdb cleanup` runs two passes:
1. `backupService.CleanupUploadedFiles()` — removes files tracked as uploaded this session (in-memory map, 1h safety buffer)
2. `cleanupOldBackupFiles()` — age-based deletion using `CleanupService.CleanupAgeBasedFiles()`

The `cleanup.weekend_only` config field exists but the actual weekend gate in `runCleanup()` is hardcoded — it does not read the config value.

### Compression

All three formats (`tar.gz`, `tar.zst`, `tar.xz`) are fully implemented with proper compression — gzip (Go stdlib), zstd (`klauspost/compress`), and xz (external binary).

### Backup directory structure

Backups are organised as: `{backup.directory}/{database}/{YYYY-MM}/{database}-{timestamp}/` (mydumper), `{backup.directory}/{database}/{YYYY-MM}/{database}-{timestamp}.sql` (mysqldump), or `{backup.directory}/{database}/{YYYY-MM}/{database}-{timestamp}.dump` (pg_dump custom format). Upload service parses this structure in `extractBackupInfo()` to replicate it in cloud storage.
