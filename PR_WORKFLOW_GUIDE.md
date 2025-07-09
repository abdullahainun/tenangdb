# Pull Request Workflow

## 🚀 **What Happens When You Create a PR**

When you create a pull request to `main`, GitHub Actions automatically runs comprehensive checks.

## ✅ **Automatic Checks**

### **CI Pipeline** (Always runs)
- **24+ parallel jobs** across Ubuntu 18.04+, macOS 12+
- **Go versions**: 1.22, 1.23, 1.24, 1.25
- **Platforms**: Linux, Darwin (AMD64/ARM64)
- **Security**: gosec + govulncheck scanning
- **Quality**: Linting and integration tests

### **Dependency Testing** (When scripts change)
- Tests on Ubuntu 18.04, 20.04, 22.04, latest
- Tests on Debian 10, 11, 12
- Tests on macOS 12, 13, latest
- Go compatibility validation (1.22-1.25)

## 📊 **PR Status Display**

In your GitHub PR, you'll see:
```
✅ CI / test-matrix (ubuntu-latest, 1.23)
✅ CI / build-matrix (linux, amd64)
✅ CI / security-scan
✅ CI / lint
✅ CI / integration-test
```

## 🔄 **Process**

1. **Create PR**: Feature branch → main
2. **GitHub Actions**: Runs automatically (~15-20 minutes)
3. **Code Review**: Team reviews your changes
4. **Merge**: All checks pass + approval = merge ready

## 🛠️ **Test Locally First**

```bash
make build && make test && make check-deps
```

## 🚨 **If Checks Fail**

1. Click failed job to see error details
2. Fix issue in your code
3. Commit and push fix
4. GitHub Actions re-runs automatically

## 📋 **Merge Requirements**

- ✅ All CI jobs pass
- ✅ Code review approved
- ✅ No merge conflicts
- ✅ Branch up-to-date with main

---

**🎯 Every PR is automatically tested across multiple platforms and Go versions before merge.**