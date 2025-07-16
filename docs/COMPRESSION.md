# 🗜️ Backup Compression Guide

TenangDB supports optional backup compression to save storage space and reduce upload times.

## 🎯 Compression Benefits

### **Storage Efficiency**
- **50-80% size reduction** depending on database content
- **Faster cloud uploads** due to smaller file sizes
- **Reduced storage costs** for cloud storage

### **Hybrid Approach**
- **Local**: Keep uncompressed backups for fast restore
- **Cloud**: Compress before upload to save bandwidth/storage
- **Auto-decompression**: Seamless restore from compressed backups

## ⚙️ Configuration

```yaml
backup:
  compression:
    enabled: true           # Enable/disable compression
    format: "tar.gz"        # Compression format
    level: 6                # Compression level (1-9)
    keep_original: true     # Keep uncompressed backup locally
    compress_upload: true   # Only compress for cloud upload
```

## 📝 Configuration Options

### **enabled**
- **Type**: Boolean
- **Default**: `false`
- **Description**: Enable or disable backup compression

### **format**
- **Type**: String
- **Default**: `"tar.gz"`
- **Options**: `"tar.gz"`, `"tar.zst"`, `"tar.xz"`
- **Description**: Compression format to use

### **level**
- **Type**: Integer
- **Default**: `6`
- **Range**: 1-9
- **Description**: Compression level (1=fastest, 9=best compression)

### **keep_original**
- **Type**: Boolean
- **Default**: `true`
- **Description**: Keep uncompressed backup locally for fast restore

### **compress_upload**
- **Type**: Boolean
- **Default**: `true`
- **Description**: Only compress for cloud upload (hybrid approach)

## 🚀 Usage Examples

### **Basic Compression**
```yaml
backup:
  compression:
    enabled: true
    format: "tar.gz"
    level: 6
```

### **Maximum Compression**
```yaml
backup:
  compression:
    enabled: true
    format: "tar.xz"      # Best compression ratio
    level: 9              # Maximum compression
    keep_original: false  # Save local space
```

### **Hybrid Approach (Recommended)**
```yaml
backup:
  compression:
    enabled: true
    format: "tar.gz"
    level: 6
    keep_original: true     # Fast local restore
    compress_upload: true   # Efficient cloud storage
```

### **Upload-Only Compression**
```yaml
backup:
  compression:
    enabled: true
    compress_upload: true   # Only compress for upload
    keep_original: true     # Keep local uncompressed
```

## 📊 Performance Comparison

| Format | Speed | Compression Ratio | CPU Usage |
|--------|-------|-------------------|-----------|
| `tar.gz` | Fast | Good (60-70%) | Low |
| `tar.zst` | Fast | Better (65-75%) | Medium |
| `tar.xz` | Slow | Best (70-80%) | High |

## 🔄 Backup & Restore Flow

### **Backup Process**
1. **Mydumper backup** → `/backups/db-2025-01-10_10-30-15/`
2. **Compression** → `/backups/db-2025-01-10_10-30-15.tar.gz`
3. **Upload** → Cloud storage (compressed file)
4. **Cleanup** → Remove original if `keep_original: false`

### **Restore Process**
1. **Auto-detection** → Detects compressed backup
2. **Decompression** → Temporary directory
3. **Restore** → Myloader/MySQL restore
4. **Cleanup** → Remove temporary decompressed files

## 💡 Best Practices

### **For Production**
```yaml
backup:
  compression:
    enabled: true
    format: "tar.gz"        # Good balance of speed/compression
    level: 6                # Default level
    keep_original: true     # Fast local restore
    compress_upload: true   # Efficient cloud storage
```

### **For Development**
```yaml
backup:
  compression:
    enabled: false          # Disable for faster local backups
```

### **For Archive Storage**
```yaml
backup:
  compression:
    enabled: true
    format: "tar.xz"        # Maximum compression
    level: 9                # Best compression
    keep_original: false    # Save local space
```

## 🔧 Command Line Usage

```bash
# Backup with compression (if enabled in config)
tenangdb backup

# Restore from compressed backup (auto-decompression)
tenangdb restore --backup-path /backup/db-2025-01-10.tar.gz --database restored_db

# Restore from uncompressed backup
tenangdb restore --backup-path /backup/db-2025-01-10/ --database restored_db
```

## 📋 Compression Logs

### **Backup Compression**
```
🗜️ Compressing backup
✅ Backup compression completed
   Original size: 125.3 MB
   Compressed size: 45.2 MB
   Compression ratio: 36.1%
   Duration: 2.3s
```

### **Restore Decompression**
```
🗜️ Decompressing backup for restore
✅ Backup decompressed successfully
   Decompressed path: /tmp/restore-db-2025-01-10/
🚀 Starting myloader restore
✅ Myloader restore completed successfully
🗑️ Cleaned up decompressed backup
```

## 🆘 Troubleshooting

### **Compression Failed**
```
⚠️ Backup compression failed, continuing with uncompressed backup
```
**Solution**: Check disk space, compression format, and permissions.

### **Decompression Failed**
```
❌ Failed to decompress backup: archive corrupted
```
**Solution**: Try with original backup or re-download from cloud.

### **Unsupported Format**
```
❌ Unsupported compression format: tar.bz2
```
**Solution**: Use supported formats: `tar.gz`, `tar.zst`, `tar.xz`.

## 🔍 Storage Usage

### **Without Compression**
```
/backups/
├── app_db-2025-01-10_10-30-15/     125.3 MB
├── logs_db-2025-01-10_10-30-15/    89.7 MB
└── user_db-2025-01-10_10-30-15/    67.2 MB
Total: 282.2 MB
```

### **With Compression (keep_original: false)**
```
/backups/
├── app_db-2025-01-10_10-30-15.tar.gz    45.2 MB
├── logs_db-2025-01-10_10-30-15.tar.gz   32.1 MB
└── user_db-2025-01-10_10-30-15.tar.gz   24.8 MB
Total: 102.1 MB (64% savings)
```

### **Hybrid Approach (keep_original: true)**
```
/backups/
├── app_db-2025-01-10_10-30-15/          125.3 MB
├── app_db-2025-01-10_10-30-15.tar.gz    45.2 MB
├── logs_db-2025-01-10_10-30-15/         89.7 MB
├── logs_db-2025-01-10_10-30-15.tar.gz   32.1 MB
└── user_db-2025-01-10_10-30-15/         67.2 MB
└── user_db-2025-01-10_10-30-15.tar.gz   24.8 MB
Total: 384.3 MB (uncompressed + compressed)
```

## 🎉 Ready to Use!

Compression feature is now available in TenangDB. Enable it in your configuration and enjoy the benefits of efficient backup storage!