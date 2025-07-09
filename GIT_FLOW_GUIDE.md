# Git Flow Guide

## 🎯 **Simple Main-Only Strategy**

TenangDB uses a **main-only** branching strategy for clean, efficient development.

## 🔄 **Development Process**

### **1. Feature Development**
```bash
# Start from main
git checkout main
git pull origin main

# Create feature branch
git checkout -b feature/awesome-feature

# Develop and commit
git commit -m "feat: add awesome feature"
git push origin feature/awesome-feature
```

### **2. Pull Request**
```bash
# Create PR: feature/awesome-feature → main
# GitHub Actions automatically runs:
# - 16+ parallel jobs across platforms
# - Go version compatibility tests
# - Security and quality checks
```

### **3. Code Review & Merge**
```bash
# After review and all checks pass:
# - Merge via GitHub UI
# - Feature branch deleted automatically
# - Main branch always production-ready
```

### **4. Release**
```bash
# Tag main when ready for release
git checkout main
git pull origin main
git tag v1.1.0
git push origin v1.1.0

# GitHub Actions automatically:
# - Builds cross-platform binaries
# - Creates GitHub release
# - Uploads release assets
```

## 🚨 **Hotfix Process**

```bash
# For critical production issues
git checkout main
git checkout -b hotfix/critical-fix
git commit -m "hotfix: fix critical issue"
git push origin hotfix/critical-fix
# Fast-track PR → merge → tag v1.0.1
```

## 📊 **Branch Structure**

```
main (production-ready)
├── feature/backup-encryption
├── feature/performance-optimization
├── fix/memory-leak
└── hotfix/security-fix (if needed)
```

## ✅ **Benefits**

- **Simple**: Single main branch, no complex synchronization
- **Fast**: Direct feature → main → release cycle
- **Quality**: Every PR tested across 16+ jobs
- **Automated**: GitHub Actions handles testing and releases

## 🛠️ **Local Testing**

```bash
# Test before creating PR
make build          # Build check
make test           # Run tests
make check-deps     # Dependency check
```

## 📋 **Best Practices**

### **Branch Naming**
- `feature/descriptive-name`
- `fix/bug-description`
- `hotfix/urgent-fix`

### **Commit Messages**
- `feat: add new feature`
- `fix: resolve bug`
- `docs: update documentation`

### **PR Guidelines**
- Small, focused changes
- Clear description
- All tests passing
- Code review approval

---

**🎯 Main-only strategy provides simple, efficient development with comprehensive quality gates.**