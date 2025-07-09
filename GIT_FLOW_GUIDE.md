# TenangDB Git Flow Guide: Development to Release

## 🎯 **Repository Structure Overview**

Your TenangDB repository follows a **main-only branching strategy** with feature branches, which is clean and efficient for your project size and team structure.

```
main (protected)
├── feature/backup-location
├── feature/install-dependencies
├── feature/versions
├── fix/makefile-linux
├── fix/mysqldump-params
└── hotfix/urgent-security (when needed)
```

## 🔄 **Complete Git Flow Process**

### **Phase 1: Development** 🛠️

#### **1.1 Starting New Development**
```bash
# Always start from latest main
git checkout main
git pull origin main

# Create feature branch with descriptive name
git checkout -b feature/backup-encryption
# or
git checkout -b fix/mysql-connection-timeout
# or  
git checkout -b enhancement/logging-improvements
```

#### **1.2 Development Work**
```bash
# Work on your feature
vim internal/backup/encryption.go
vim tests/backup_test.go

# Commit frequently with clear messages
git add .
git commit -m "feat: add AES-256 encryption for backup files"

git add .
git commit -m "test: add encryption unit tests"

git add .
git commit -m "docs: update encryption configuration examples"
```

#### **1.3 Push Feature Branch**
```bash
# Push feature branch to remote
git push origin feature/backup-encryption

# If you need to make more changes
git add .
git commit -m "fix: handle encryption key rotation"
git push origin feature/backup-encryption
```

### **Phase 2: Code Review & Testing** 🔍

#### **2.1 Create Pull Request**
```bash
# Via GitHub UI or GitHub CLI
gh pr create --title "feat: Add AES-256 encryption for backup files" \
             --body "## Summary
- Implements AES-256 encryption for backup files
- Adds configuration for encryption keys
- Includes comprehensive unit tests

## Testing
- [x] Unit tests pass
- [x] Integration tests with encrypted backups
- [x] Performance impact minimal (<5% overhead)

## Breaking Changes
- None - encryption is optional and disabled by default"
```

#### **2.2 Automatic GitHub Actions**
When you create the PR, GitHub Actions automatically trigger:

```yaml
🔄 CI Pipeline (16+ jobs running in parallel):
├── test-matrix (6 jobs)
│   ├── ✅ ubuntu-latest + Go 1.23
│   ├── ✅ ubuntu-latest + Go 1.24
│   ├── ✅ windows-latest + Go 1.23
│   ├── ✅ macos-latest + Go 1.23
│   ├── ✅ ubuntu-latest + Go 1.22
│   └── ✅ ubuntu-latest + Go 1.25
├── build-matrix (5 jobs)
│   ├── ✅ linux/amd64
│   ├── ✅ linux/arm64
│   ├── ✅ darwin/amd64
│   ├── ✅ darwin/arm64
│   └── ✅ windows/amd64
├── ✅ dependency-test (ubuntu + macos)
├── ✅ security-scan (gosec + govulncheck)
├── ✅ lint (golangci-lint)
├── ✅ integration-test (MySQL + real backup)
└── ✅ status-check (final verification)
```

#### **2.3 Code Review Process**
```bash
# Reviewers can:
# 1. Comment on specific lines
# 2. Request changes
# 3. Approve the PR

# If changes requested:
git checkout feature/backup-encryption
# Make requested changes
git add .
git commit -m "fix: address review comments on key management"
git push origin feature/backup-encryption
# GitHub Actions re-runs automatically
```

### **Phase 3: Integration** 🔀

#### **3.1 Pre-merge Verification**
```bash
# All these must be ✅ before merge:
✅ All CI jobs passed (16/16)
✅ Code review approved
✅ No merge conflicts
✅ Branch is up to date with main
```

#### **3.2 Merge to Main**
```bash
# Via GitHub UI (recommended) or CLI
gh pr merge --squash  # Squash commits for clean history
# or
gh pr merge --merge   # Keep all commits
# or
gh pr merge --rebase  # Rebase and merge
```

#### **3.3 Post-merge Actions**
```bash
# Automatic GitHub Actions on main:
main branch updated → CI pipeline runs again
                   → Status badges updated
                   → Integration tests run
                   → Security scans run
                   → Nightly build artifacts updated
```

#### **3.4 Cleanup**
```bash
# Delete merged feature branch
git checkout main
git pull origin main
git branch -d feature/backup-encryption
git push origin --delete feature/backup-encryption
```

### **Phase 4: Release Preparation** 🚀

#### **4.1 Pre-release Checklist**
```bash
# Ensure main is stable
git checkout main
git pull origin main

# Verify all tests pass
make test
make build
make check-deps

# Check version and changelog
./tenangdb version
# Should show: TenangDB version v1.0.0

# Review changes since last release
git log v1.0.0..HEAD --oneline
```

#### **4.2 Version Planning**
```bash
# Determine next version based on changes:
# - Major (2.0.0): Breaking changes
# - Minor (1.1.0): New features, backward compatible
# - Patch (1.0.1): Bug fixes only

# Example for new features:
NEXT_VERSION="v1.1.0"
```

#### **4.3 Release Preparation**
```bash
# Optional: Create release preparation branch
git checkout -b release/v1.1.0

# Update version-related files if needed
vim CHANGELOG.md  # Add release notes
vim README.md     # Update version references

# Commit release preparation
git add .
git commit -m "chore: prepare release v1.1.0"
git push origin release/v1.1.0

# Create PR for release preparation
gh pr create --title "chore: prepare release v1.1.0" \
             --body "## Release v1.1.0 Preparation
- Updated CHANGELOG.md with new features
- Updated README.md version references
- Ready for release tagging"
```

### **Phase 5: Release** 📦

#### **5.1 Create Release Tag**
```bash
# After release preparation is merged to main
git checkout main
git pull origin main

# Create and push release tag
git tag -a v1.1.0 -m "Release v1.1.0: Add backup encryption and dependency improvements"
git push origin v1.1.0
```

#### **5.2 Automatic Release Process**
```bash
# GitHub Actions release workflow triggers automatically:
🔄 Release Pipeline:
├── ✅ Build multi-platform binaries
│   ├── tenangdb-linux-amd64
│   ├── tenangdb-linux-arm64
│   ├── tenangdb-darwin-amd64
│   ├── tenangdb-darwin-arm64
│   └── tenangdb-windows-amd64.exe
├── ✅ Generate checksums
├── ✅ Run final tests
├── ✅ Create GitHub release
└── ✅ Upload release assets
```

#### **5.3 Release Verification**
```bash
# Check release was created successfully
gh release list

# Download and test release binary
wget https://github.com/username/tenangdb/releases/download/v1.1.0/tenangdb-linux-amd64
chmod +x tenangdb-linux-amd64
./tenangdb-linux-amd64 version
# Should show: TenangDB version v1.1.0
```

### **Phase 6: Post-Release** 🎉

#### **6.1 Release Announcement**
```bash
# Update documentation
# Announce in project channels
# Update deployment scripts
# Monitor for issues
```

#### **6.2 Hotfix Process (if needed)**
```bash
# For critical issues in production
git checkout main
git pull origin main
git checkout -b hotfix/critical-security-fix

# Make minimal fix
git add .
git commit -m "hotfix: fix critical security vulnerability"
git push origin hotfix/critical-security-fix

# Create urgent PR
gh pr create --title "hotfix: Critical security vulnerability" \
             --body "## Critical Hotfix
- Fixes security vulnerability in authentication
- Minimal change, low risk
- Needs immediate release"

# After merge and testing:
git tag -a v1.1.1 -m "Hotfix v1.1.1: Critical security fix"
git push origin v1.1.1
```

## 📊 **Git Flow Summary**

### **Branch Types & Purpose**
```
main              Production-ready code, always stable
├── feature/*     New features, enhancements
├── fix/*         Bug fixes, non-critical issues
├── hotfix/*      Critical production fixes
├── release/*     Release preparation (optional)
└── docs/*        Documentation updates
```

### **Workflow Timeline**
```
Development → Testing → Review → Integration → Release
    ↓           ↓         ↓          ↓           ↓
Feature     GitHub    Code       Merge to    Create
Branch      Actions   Review     Main        Tag
(1-7 days)  (15-20min)(1-2 days) (instant)  (planned)
```

### **Key Integrations**
- **GitHub Actions**: Automated testing and releases
- **Branch Protection**: Enforce CI checks before merge
- **Auto-merge**: Can be enabled for dependency updates
- **Status Checks**: Real-time CI feedback in PRs

## 🛠️ **Development Best Practices**

### **Branch Naming**
```bash
feature/feature-name      # New features
fix/bug-description       # Bug fixes
hotfix/urgent-fix         # Critical fixes
docs/documentation-update # Documentation
chore/maintenance-task    # Maintenance
```

### **Commit Messages**
```bash
feat: add new feature
fix: resolve bug in component
docs: update API documentation
test: add unit tests for backup
chore: update dependencies
BREAKING: change API signature
```

### **PR Best Practices**
- **Small PRs**: Easier to review and test
- **Clear descriptions**: Explain what and why
- **Link issues**: Reference related issues
- **Test coverage**: Include tests for new features
- **Documentation**: Update docs for user-facing changes

## 🚨 **Emergency Procedures**

### **Rollback Release**
```bash
# If v1.1.0 has critical issues
git checkout main
git revert <commit-hash>
git tag -a v1.1.1 -m "Rollback v1.1.0 due to critical issue"
git push origin v1.1.1
```

### **Fast Emergency Fix**
```bash
# For production-down scenarios
git checkout main
git checkout -b hotfix/emergency-fix
# Make minimal fix
git push origin hotfix/emergency-fix
# Create PR with "URGENT" label
# Override normal review process if needed
```

## 📈 **Monitoring & Metrics**

### **Release Health**
- **GitHub Actions**: Monitor CI success rates
- **Release frequency**: Track time between releases
- **Hotfix frequency**: Monitor production stability
- **PR metrics**: Review time, merge rate

### **Quality Metrics**
- **Test coverage**: Maintained by CI
- **Security scans**: Automated in every PR
- **Performance**: Tracked in nightly builds
- **Dependency health**: Weekly automated checks

---

**🎯 This git flow provides a robust, automated pipeline from development to release while maintaining high code quality and stability for TenangDB!**