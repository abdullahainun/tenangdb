# Backup Frequency Check Feature

Fitur ini mencegah backup yang tidak disengaja atau terlalu sering dengan mengecek kapan terakhir kali backup dilakukan.

## Konfigurasi

Tambahkan konfigurasi berikut ke `config.yaml`:

```yaml
backup:
  # ... konfigurasi backup lainnya
  
  # Backup frequency check configuration
  check_last_backup_time: true      # Enable backup frequency checking
  min_backup_interval: 1h           # Minimum interval between backups
  skip_confirmation: false          # Set to true to skip confirmation prompts
```

## Penggunaan

### 1. Backup Normal (dengan pengecekan)
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

## Konfigurasi Interval

Anda dapat mengatur interval minimum dengan berbagai format:

```yaml
backup:
  min_backup_interval: 30m    # 30 menit
  min_backup_interval: 1h     # 1 jam
  min_backup_interval: 2h30m  # 2 jam 30 menit
  min_backup_interval: 24h    # 1 hari
```

## Tracking File

Sistem akan membuat file `.tenangdb_backup_tracking.json` di dalam backup directory untuk melacak waktu backup terakhir:

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

Untuk menonaktifkan fitur ini, set:

```yaml
backup:
  check_last_backup_time: false
```

## Contoh Workflow

1. **Backup pertama** - Berjalan normal tanpa konfirmasi
2. **Backup kedua dalam 1 jam** - Muncul konfirmasi
3. **User pilih 'y'** - Backup dilanjutkan
4. **Backup ketiga dengan --force** - Langsung jalan tanpa konfirmasi
5. **Backup setelah 1 jam** - Berjalan normal tanpa konfirmasi

## Log Messages

- `✅ Backup confirmed by user` - User approved backup
- `🔄 Backup forced by user` - User forced backup
- `❌ Backup cancelled by user` - User cancelled backup
- `⏭️ dbname backup skipped` - Backup was skipped