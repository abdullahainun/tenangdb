# TenangDB v1.1.2 Release Notes

## 🚀 Major Features

### Auto-Discovery System
- **Binary Path Discovery**: Automatically finds mydumper, myloader, rclone, and mysql binaries across platforms
- **Cross-Platform Compatibility**: Supports macOS (Homebrew), Linux (system packages), and manual installations
- **Version-Aware mydumper**: Detects mydumper versions (v0.9.1 - v0.19.3+) and uses appropriate parameters
- **Platform-Specific Defaults**: Auto-configures backup, log, and metrics paths based on OS

### Enhanced Cloud Upload Structure  
- **Organized Directory Structure**: Maintains consistent local/cloud structure: `{destination}/{database}/{YYYY-MM}/{backup-timestamp}/`
- **Directory Preservation**: Properly uploads mydumper directory backups with full structure
- **Improved Upload Logic**: Separate handling for files vs directories with proper path construction

### Configuration Improvements
- **Clean Configuration Files**: Simplified config.yaml.example with grouped comments and better organization
- **Auto-Discovery Documentation**: Clear explanation of what gets auto-discovered vs manual configuration
- **Minimal Required Config**: Only database credentials and database list required

## 🐛 Critical Bug Fixes

### Metrics Configuration Fix
- **Fixed metrics initialization**: All metrics functions now properly respect `metrics.enabled` configuration
- **Resolved directory creation**: No more unwanted `/var/lib/tenangdb` directory when metrics disabled
- **Permission error fixes**: Eliminated permission denied errors for metrics when disabled
- **Missing config handling**: Works correctly when metrics section is missing from config

### Cross-Platform mydumper Compatibility
- **Parameter Detection**: Automatically uses modern (`--sync-thread-lock-mode=AUTO`) or legacy (`--no-locks`) parameters
- **Version Compatibility**: Supports Ubuntu 18.04 (v0.9.1), modern Linux (v0.10.0+), and macOS (v0.19.3+)
- **Compression Support**: Added .zst file verification for compressed mydumper backups

## 📚 Documentation Overhaul

### User-Friendly Documentation
- **Simplified README**: Reduced from verbose to clean, actionable quick-start guide
- **Binary Installation Priority**: Release binaries as primary installation method
- **One-Liner Install**: `curl -sSL .../install.sh | bash` for instant setup
- **Progressive Disclosure**: Basic setup visible, advanced options in collapsible sections

### Updated Guides
- **Installation Guide**: Added macOS support, binary releases, and corrected command examples
- **MySQL User Setup**: Complete SQL scripts for proper user privileges
- **Production Deployment**: Updated for binary installations and auto-discovery
- **Configuration Examples**: All examples verified against actual codebase

## 🔧 Technical Improvements

### Code Quality & Structure
- **Auto-Discovery Functions**: Added `findRclonePath()`, `findMydumperPath()`, `findMyloaderPath()`, `findMysqldumpPath()`, `findMysqlPath()`
- **Version Detection**: Implemented `isMydumperVersionCompatible()` for parameter selection
- **Upload Service Refactor**: Split upload logic into `uploadFile()` and `uploadDirectory()` methods
- **Configuration Validation**: Enhanced config validation with auto-discovery support

### Performance & Reliability
- **Backup Verification**: Enhanced verification logic for .zst compressed files
- **Error Handling**: Improved error messages and graceful fallbacks
- **Logging Enhancement**: Better structured logging with emojis and progress indicators
- **Concurrent Operations**: Optimized backup and upload concurrency

## 🌍 Cross-Platform Support

### macOS Support
- **Homebrew Integration**: Auto-discovery for Homebrew installation paths
- **Apple Silicon**: Support for both Intel and ARM64 architectures
- **Platform-Specific Paths**: Proper macOS directory conventions

### Linux Compatibility
- **Package Manager Support**: Works with apt, dnf, and manual installations
- **Distribution Coverage**: Ubuntu 18.04+ to latest distributions
- **Legacy mydumper**: Backward compatibility with older mydumper versions

## 📦 Release Assets

### Binary Downloads
- `tenangdb-linux-amd64` - Linux 64-bit
- `tenangdb-darwin-amd64` - macOS Intel 64-bit  
- `tenangdb-darwin-arm64` - macOS Apple Silicon
- `install.sh` - One-liner installation script

## 🚀 Quick Start

### One-Liner Installation
```bash
curl -sSL https://raw.githubusercontent.com/abdullahainun/tenangdb/main/install.sh | bash
```

### Manual Installation
```bash
# Download for your platform
curl -L https://github.com/abdullahainun/tenangdb/releases/download/v1.1.2/tenangdb-linux-amd64 -o tenangdb
chmod +x tenangdb && sudo mv tenangdb /usr/local/bin/

# Get config and run
curl -L https://raw.githubusercontent.com/abdullahainun/tenangdb/main/config.yaml.example -o config.yaml
nano config.yaml  # Edit with your database credentials
tenangdb backup
```

## 🔄 Migration

### From v1.1.1
- **Backward Compatible**: All existing configurations continue to work
- **Auto-Discovery Benefits**: Existing hardcoded paths will be auto-discovered if removed
- **Enhanced Features**: Cloud upload structure and mydumper compatibility automatically improved

### Configuration Updates (Optional)
- Remove hardcoded binary paths to enable auto-discovery
- Update config.yaml.example for cleaner configuration
- Consider using new install script for easier deployment

## 🙏 Contributors

- @abdullahainun - Core development and cross-platform testing
- Community contributors - Testing across different platforms and mydumper versions

---

**Full Changelog**: https://github.com/abdullahainun/tenangdb/compare/v1.1.1...v1.1.2