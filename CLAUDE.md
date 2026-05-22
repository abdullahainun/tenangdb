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
make check-deps   # verify mysqldump/mydumper/rclone present
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
      → database.NewClient()     # connects to MySQL, pings on init
      → upload.NewService()      # wraps rclone, only if upload.enabled
      → compression.NewCompressor()
  → service.Run()
      → processDatabasesBatch()  # respects batch_size + concurrency via semaphore
          → createBackupWithRetry()    # retry_count attempts
          → compressor.CompressBackup()
          → uploader.Upload()          # rclone copy
      → metricsStorage.Update*()      # write to metrics.json
```

### Two parallel database code paths

**Active path** — `pkg/database/client.go` (`database.Client`): used by all production commands. Contains full mysqldump and mydumper implementations including version-aware mydumper args (`isMydumperVersionCompatible()` detects v0.9.x legacy vs v0.19.x modern via `--help` output parsing).

**WIP path** — `pkg/database/mysql_provider.go` + `provider.go` + `factory.go`: a provider-interface refactor (supports future PostgreSQL via `PostgreSQLConfig`). Currently contains placeholder implementations and is **not wired into any command**. `RestoreBackup()` here returns a hardcoded error.

When modifying backup or restore logic, always work in `client.go`, not `mysql_provider.go`.

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

Only `tar.gz` is fully implemented despite config accepting `tar.zst` and `tar.xz`. All three formats call `createTarGz()` internally — the output file extension changes but the content is always gzip.

### Backup directory structure

Backups are organised as: `{backup.directory}/{database}/{YYYY-MM}/{database}-{timestamp}/` (mydumper) or `{backup.directory}/{database}/{YYYY-MM}/{database}-{timestamp}.sql` (mysqldump). Upload service parses this structure in `extractBackupInfo()` to replicate it in cloud storage.
