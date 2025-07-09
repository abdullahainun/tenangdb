# TenangDB v1.1.1 Release Notes

## 🐛 Bug Fixes

### Metrics Configuration Fix
- **Fixed metrics files creation when disabled**: TenangDB now properly respects the `metrics.enabled: false` configuration
- **Prevented unwanted directory creation**: No longer creates `/var/lib/tenangdb` directory when metrics are disabled
- **Enhanced privacy/security**: Eliminates unwanted file system modifications when metrics are not needed

## 🔧 Technical Changes

### Conditional Metrics Initialization
- MetricsStorage is only initialized when `cfg.Metrics.Enabled` is true
- All metrics operations are wrapped in conditional checks
- Prometheus metrics recording is now conditional
- Added proper null checking to prevent runtime errors

### Impact
- **Before**: TenangDB created metrics files and directories even when `metrics.enabled: false`
- **After**: TenangDB respects the configuration and creates no metrics-related files when disabled

## 📋 What's Changed

- Only initialize metrics storage when explicitly enabled
- Conditional metrics updates for cleanup and restore operations
- Conditional Prometheus metrics recording
- Proper null safety for metrics operations

## 🚀 Installation

### Using Install Script
```bash
curl -fsSL https://raw.githubusercontent.com/abdullahainun/tenangdb/main/scripts/install-dependencies.sh | bash
```

### Download Binary
Download the latest binary from [GitHub Releases](https://github.com/abdullahainun/tenangdb/releases/tag/v1.1.1)

## 🙏 Contributors

- @abdullahainun - Bug fix and implementation

---

**Full Changelog**: https://github.com/abdullahainun/tenangdb/compare/v1.1.0...v1.1.1