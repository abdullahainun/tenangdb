# TenangDB GitHub Actions Workflows

This directory contains GitHub Actions workflows for continuous integration, testing, and deployment of TenangDB.

## Workflows Overview

### 1. `ci.yml` - Continuous Integration
**Triggers:** Push/PR to main/develop branches
**Purpose:** Comprehensive testing across multiple platforms and Go versions

**Jobs:**
- **test-matrix**: Tests on Ubuntu, Windows, macOS with Go 1.22-1.25
- **build-matrix**: Cross-compilation for Linux/Darwin/Windows on AMD64/ARM64
- **dependency-test**: Tests dependency installation on Ubuntu/macOS
- **security-scan**: Runs gosec and govulncheck
- **lint**: Code linting with golangci-lint
- **integration-test**: Full integration tests with MySQL database

### 2. `dependency-test.yml` - Dependency Installation Testing
**Triggers:** Changes to install-dependencies.sh, weekly schedule
**Purpose:** Ensures dependency installation works across all supported platforms

**Jobs:**
- **test-ubuntu**: Tests on Ubuntu 20.04, 22.04, latest
- **test-debian**: Tests on Debian 10, 11, 12 (containerized)
- **test-macos**: Tests on macOS 12, 13, 14
- **test-go-compatibility**: Tests Go versions 1.21-1.24
- **test-dependency-options**: Tests script options (--help, --check-only, etc.)
- **test-make-targets**: Tests all Makefile targets

### 3. `nightly.yml` - Nightly Builds
**Triggers:** Daily at 2:00 AM UTC, manual dispatch
**Purpose:** Catches issues early with comprehensive nightly testing

**Jobs:**
- **nightly-build**: Builds on all platforms with nightly version
- **performance-test**: Performance benchmarking with MySQL
- **security-nightly**: Comprehensive security scanning
- **dependency-update-check**: Checks for outdated dependencies

### 4. `release.yml` - Release Automation
**Triggers:** Git tags (v*)
**Purpose:** Automated release creation with multi-platform binaries

**Jobs:**
- **build**: Creates release binaries for all supported platforms
- **create-release**: Creates GitHub release with binaries and checksums

### 5. `status-badge.yml` - Status Badges
**Triggers:** Push to main branch
**Purpose:** Generates dynamic badges for README

**Badges:**
- Build status
- Test coverage
- Platform support
- Dependency status
- Go version

## Supported Platforms

### Operating Systems
- **Ubuntu**: 20.04, 22.04, latest
- **Debian**: 10 (Buster), 11 (Bullseye), 12 (Bookworm)
- **macOS**: 12, 13, 14
- **Windows**: latest (limited dependency support)

### Go Versions
- **Primary**: 1.23, 1.24
- **Tested**: 1.21, 1.22, 1.25
- **Minimum**: 1.23 (required for TenangDB)

### Build Targets
- **linux/amd64**: Primary Linux target
- **linux/arm64**: ARM64 Linux (servers, Raspberry Pi)
- **darwin/amd64**: Intel Mac
- **darwin/arm64**: Apple Silicon Mac
- **windows/amd64**: Windows 64-bit

## Usage

### Running Workflows Locally
You can test the dependency installation locally:

```bash
# Test dependency checker
./scripts/install-dependencies.sh --check-only

# Install dependencies
./scripts/install-dependencies.sh -y

# Test specific options
./scripts/install-dependencies.sh --no-go --no-rclone --check-only
```

### Make Targets
All workflows use standardized Make targets:

```bash
make check-deps     # Check dependencies
make install-deps   # Install dependencies
make build          # Build TenangDB
make test           # Run tests
make deps           # Install Go dependencies
```

### Manual Workflow Dispatch
Some workflows can be triggered manually:

1. Go to **Actions** tab in GitHub
2. Select the workflow (e.g., "Nightly Build")
3. Click **Run workflow**
4. Configure options if available

## Configuration

### Environment Variables
- `GITHUB_TOKEN`: Automatically provided by GitHub
- `CODECOV_TOKEN`: For coverage reporting (optional)

### Secrets
- `GITHUB_TOKEN`: For release creation and badge generation
- Additional secrets can be added in repository settings

### Customization
You can customize workflows by:

1. **Adding new OS versions**: Edit matrix strategy
2. **Adding new Go versions**: Update go-version matrix
3. **Modifying test databases**: Update service configurations
4. **Adding new build targets**: Update GOOS/GOARCH combinations

## Badge Integration

Add these badges to your README:

```markdown
[![CI](https://github.com/username/tenangdb/workflows/CI/badge.svg)](https://github.com/username/tenangdb/actions)
[![Dependency Test](https://github.com/username/tenangdb/workflows/Dependency%20Installation%20Test/badge.svg)](https://github.com/username/tenangdb/actions)
[![Nightly Build](https://github.com/username/tenangdb/workflows/Nightly%20Build/badge.svg)](https://github.com/username/tenangdb/actions)
[![Release](https://github.com/username/tenangdb/workflows/Release/badge.svg)](https://github.com/username/tenangdb/actions)
```

## Troubleshooting

### Common Issues

1. **Dependency installation fails**: Check OS compatibility in scripts
2. **Build fails on specific platform**: Verify Go version compatibility
3. **Tests timeout**: Increase timeout in workflow configuration
4. **MySQL connection fails**: Check service configuration

### Debug Tips

1. **Enable debug logging**: Add `--log-level debug` to TenangDB commands
2. **Check dependency script**: Run `./scripts/install-dependencies.sh --check-only`
3. **Verify build**: Test local build with `make build`
4. **Check artifacts**: Download build artifacts from Actions tab

## Contributing

When adding new workflows:

1. **Test locally first**: Verify scripts work on your platform
2. **Use matrix strategy**: Test multiple versions/platforms
3. **Add proper error handling**: Workflows should fail gracefully
4. **Update documentation**: Update this README with new workflows
5. **Test with secrets**: Ensure workflows work with/without secrets

## Performance Considerations

- **Caching**: All workflows use Go module caching
- **Parallel execution**: Jobs run in parallel where possible
- **Artifact retention**: Build artifacts kept for 7 days, summaries for 30 days
- **Schedule optimization**: Nightly builds run during low-traffic hours

## Security

- **Dependency scanning**: Automated vulnerability checks
- **Secret management**: Secrets are never logged
- **Least privilege**: Minimal permissions for each job
- **Container security**: Debian containers use official images only