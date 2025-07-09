# Workflow Overview

## 📋 **Documentation Structure**

### **Quick Reference**
- **GITHUB_ACTIONS_SUMMARY.md** - What happens on PRs
- **PR_WORKFLOW_GUIDE.md** - Pull request process
- **GIT_FLOW_GUIDE.md** - Development workflow

### **Detailed Documentation**
- **.github/workflows/README.md** - Complete workflow details
- **.github/workflows/BRANCH_STRATEGY.md** - Main-only strategy

## 🚀 **Quick Start**

### **Development**
```bash
git checkout -b feature/new-feature
# Develop, commit, push
# Create PR to main
# GitHub Actions tests automatically
```

### **Testing**
```bash
make build && make test && make check-deps
```

### **Release**
```bash
git tag v1.1.0
git push origin v1.1.0
# Automated cross-platform release
```

## ✅ **Key Features**

- **Automated Testing**: 16+ jobs per PR
- **Cross-Platform**: Linux, macOS, Windows
- **Go Compatibility**: 1.22-1.25
- **Security Scanning**: gosec + govulncheck
- **Main-Only Strategy**: Simple, efficient workflow

---

**🎯 Clean, automated development workflow with comprehensive quality gates.**