# Pull Request Workflow Guide

## 🚀 What Happens When You Create a Pull Request

When you create a pull request to `main` or `develop` branches, the following GitHub Actions workflows are automatically triggered:

### ✅ **Always Runs on PRs:**

#### 1. **CI Workflow (`ci.yml`)**
**Triggers:** ALL pull requests to main/develop
**Duration:** ~15-20 minutes
**What it does:**

```yaml
on:
  pull_request:
    branches: [ main, develop ]
```

**Jobs that run:**
- **test-matrix** (6 jobs): Tests on Ubuntu/Windows/macOS with Go 1.22-1.25
- **build-matrix** (5 jobs): Cross-compilation for all platforms
- **dependency-test** (2 jobs): Tests dependency installation
- **security-scan** (1 job): Runs gosec and govulncheck
- **lint** (1 job): Code linting with golangci-lint  
- **integration-test** (1 job): Full MySQL integration tests
- **status-check** (1 job): Final status verification

**Total:** ~16 parallel jobs

#### 2. **Status Badge Workflow (`status-badge.yml`)**
**Triggers:** Pull requests to main branch only
**Duration:** ~5 minutes
**What it does:**
- Updates build status badges
- Generates coverage reports
- Updates platform support badges

### 🎯 **Conditionally Runs on PRs:**

#### 3. **Dependency Test Workflow (`dependency-test.yml`)**
**Triggers:** Only when these files change:
- `scripts/install-dependencies.sh`
- `.github/workflows/dependency-test.yml`

```yaml
on:
  pull_request:
    branches: [ main, develop ]
    paths:
      - 'scripts/install-dependencies.sh'
      - '.github/workflows/dependency-test.yml'
```

**Jobs that run:**
- **test-ubuntu** (3 jobs): Ubuntu 20.04, 22.04, latest
- **test-debian** (3 jobs): Debian 10, 11, 12 (containerized)
- **test-macos** (3 jobs): macOS 12, 13, 14
- **test-go-compatibility** (4 jobs): Go 1.21-1.24
- **test-dependency-options** (1 job): Script options testing
- **test-make-targets** (1 job): Make targets testing

**Total:** ~15 additional jobs (only when dependency files change)

## 📊 **PR Check Status Display**

When you create a PR, you'll see checks like this in the GitHub UI:

```
✅ CI / test-matrix (ubuntu-latest, 1.23)
✅ CI / test-matrix (ubuntu-latest, 1.24)  
✅ CI / test-matrix (windows-latest, 1.23)
✅ CI / test-matrix (macos-latest, 1.23)
✅ CI / build-matrix (linux, amd64)
✅ CI / build-matrix (darwin, arm64)
✅ CI / security-scan
✅ CI / lint
✅ CI / integration-test
✅ CI / status-check
```

## 🔄 **PR Workflow Process**

### 1. **PR Creation**
```bash
git checkout -b feature/new-feature
git commit -m "Add new feature"
git push origin feature/new-feature
# Create PR via GitHub UI
```

### 2. **Automatic Checks Start**
- GitHub immediately starts running workflows
- You'll see "Some checks haven't completed yet" status
- Each job runs independently in parallel

### 3. **Check Results**
- **✅ Green checkmark**: All tests passed
- **❌ Red X**: Tests failed - click for details
- **🟡 Yellow circle**: Tests still running
- **⚠️ Orange warning**: Some tests skipped

### 4. **PR Merge Requirements**
Based on the workflows, these checks must pass:
- All CI jobs must complete successfully
- Security scan must pass
- Lint checks must pass
- Integration tests must pass

## 🛠️ **Testing Your Changes Locally**

Before creating a PR, you can run the same checks locally:

```bash
# Run the same tests that CI runs
make test

# Run linting
make lint

# Run security scan
make security

# Check dependencies
make check-deps

# Build for all platforms (like CI does)
make build
GOOS=linux GOARCH=amd64 go build ./cmd
GOOS=darwin GOARCH=arm64 go build ./cmd
GOOS=windows GOARCH=amd64 go build ./cmd

# If you modified the dependency script
chmod +x scripts/install-dependencies.sh
./scripts/install-dependencies.sh --check-only
```

## 🚨 **Common PR Check Failures**

### **Test Failures**
```
❌ CI / test-matrix (ubuntu-latest, 1.23)
```
**Fix:** Check test logs, fix failing tests, push new commit

### **Build Failures**
```
❌ CI / build-matrix (linux, amd64)
```
**Fix:** Check compilation errors, fix code, push new commit

### **Lint Failures**
```
❌ CI / lint
```
**Fix:** Run `make lint` locally, fix issues, push new commit

### **Security Issues**
```
❌ CI / security-scan
```
**Fix:** Review security warnings, fix vulnerabilities, push new commit

### **Dependency Issues**
```
❌ Dependency Installation Test / test-ubuntu (20.04)
```
**Fix:** Test dependency script locally, fix issues, push new commit

## 📝 **PR Check Examples**

### **Simple Code Change PR**
```
Files changed: internal/backup/service.go
Workflows triggered: ci.yml only
Jobs: ~16 jobs
Duration: ~15 minutes
```

### **Dependency Script Change PR**
```
Files changed: scripts/install-dependencies.sh
Workflows triggered: ci.yml + dependency-test.yml
Jobs: ~31 jobs  
Duration: ~25 minutes
```

### **Documentation-only PR**
```
Files changed: README.md
Workflows triggered: ci.yml only
Jobs: ~16 jobs (but most will be very fast)
Duration: ~10 minutes
```

## 🎯 **Tips for Successful PRs**

1. **Test locally first:**
   ```bash
   make test
   make build
   make check-deps
   ```

2. **Small, focused changes:**
   - Easier to review
   - Faster CI runs
   - Easier to debug failures

3. **Watch the CI status:**
   - Fix failures quickly
   - Don't merge until all checks pass

4. **Check specific job logs:**
   - Click on failed jobs to see details
   - Look for specific error messages

## 📋 **PR Checklist**

Before creating a PR, ensure:
- [ ] Code builds locally (`make build`)
- [ ] Tests pass locally (`make test`)
- [ ] Linting passes (`make lint`)
- [ ] Dependencies work (`make check-deps`)
- [ ] Security scan passes (`make security`)
- [ ] Changes are well-documented
- [ ] Commit messages are clear

## 🔄 **After PR Merge**

Once your PR is merged to `main`:
1. **CI workflow** runs again on main branch
2. **Status badges** are updated
3. **Nightly builds** will include your changes
4. **Release workflow** will include your changes in next tag

## 🆘 **Troubleshooting**

### **"Some checks haven't completed yet"**
- Wait for jobs to finish (can take 15-30 minutes)
- Check individual job status

### **"All checks have failed"**
- Click on failed job for details
- Fix issues and push new commit
- Checks will re-run automatically

### **"Workflow run was cancelled"**
- Usually due to new commits
- Latest commit will trigger new run

### **Need help?**
- Check workflow logs in Actions tab
- Review this guide
- Ask in PR comments for help