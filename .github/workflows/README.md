# GitHub Actions Workflows

## 🚀 **Overview**

TenangDB uses GitHub Actions for automated testing, building, and releasing across multiple platforms.

## 📋 **Workflows**

### **CI Pipeline** (`ci.yml`)
**Triggers:** Every PR to main, push to main  
**Duration:** ~15 minutes  
**Purpose:** Comprehensive testing and building

**What it does:**
- Tests on Ubuntu, macOS with Go 1.22-1.24
- Builds for Linux/Darwin/Windows (AMD64/ARM64)
- Runs security scans, linting, and integration tests

### **Dependency Testing** (`dependency-test.yml`)
**Triggers:** Changes to `scripts/install-dependencies.sh`  
**Duration:** ~25 minutes  
**Purpose:** Validates dependency installation across platforms

**What it does:**
- Tests Ubuntu 18.04, 20.04, 22.04, latest
- Tests Debian 10, 11, 12 (containerized)
- Tests macOS 12, 13, latest
- Validates Go compatibility (1.22-1.24)

### **Nightly Builds** (`nightly.yml`)
**Triggers:** Daily at 2:00 AM UTC  
**Duration:** ~30 minutes  
**Purpose:** Comprehensive quality checks

**What it does:**
- Builds nightly binaries for all platforms
- Runs performance tests with 10,000 record database
- Security scanning and dependency updates

### **Release** (`release.yml`)
**Triggers:** Git tags (`v*`)  
**Duration:** ~10 minutes  
**Purpose:** Automated releases

**What it does:**
- Builds binaries for all platforms
- Generates checksums and release notes
- Creates GitHub release with assets

### **Status Badges** (`status-badge.yml`)
**Triggers:** Push to main  
**Duration:** ~5 minutes  
**Purpose:** Updates README badges

**What it does:**
- Generates build status badges
- Updates test coverage badges
- Platform support indicators

### **Dependency Updates** (`dependency-update.yml`)
**Triggers:** Weekly on Mondays  
**Duration:** ~10 minutes  
**Purpose:** Automated dependency monitoring

**What it does:**
- Checks for Go and mydumper updates
- Creates GitHub issues for updates
- Tests compatibility with latest versions

## 🎯 **Pull Request Workflow**

When you create a PR to main:

1. **CI Pipeline** runs automatically (16+ jobs)
2. **Dependency Testing** runs if scripts changed
3. All checks must pass before merge
4. Status displayed in PR interface

## 🏷️ **Release Process**

```bash
# Create release
git tag v1.1.0
git push origin v1.1.0

# GitHub Actions automatically:
# - Builds cross-platform binaries
# - Creates GitHub release
# - Uploads assets with checksums
```

## 📊 **Supported Platforms**

**Operating Systems:**
- Ubuntu 18.04, 20.04, 22.04, latest
- Debian 10, 11, 12
- macOS 12, 13, latest (10.15+)

**Go Versions:**
- Primary: 1.23, 1.24
- Tested: 1.22, 1.23, 1.24
- Minimum: 1.23 (required for TenangDB)

**Build Targets:**
- linux/amd64, linux/arm64
- darwin/amd64, darwin/arm64

## 🔧 **Local Testing**

Test the same checks locally:

```bash
make build          # Build check
make test           # Run tests
make check-deps     # Dependency check
```

## 📈 **Monitoring**

- **Actions tab**: View workflow runs
- **Status badges**: Real-time build status
- **Releases**: Automatic binary distribution
- **Issues**: Automated dependency updates

---

**🎯 All workflows are optimized for the main-only branching strategy with comprehensive quality gates.**