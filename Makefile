BINARY_NAME=tenangdb
VERSION=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.1.0")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}"

.PHONY: build clean test install uninstall deps install-deps check-deps setup-ubuntu-18.04

# Build the application
build:
	@echo "🔍 Checking Go version compatibility..."
	@go version
	@echo "📦 Building TenangDB with Go modules..."
	GO111MODULE=on go build ${LDFLAGS} -o ${BINARY_NAME} ./cmd

# Build for production (with optimizations)
build-prod:
	@echo "🔍 Checking Go version compatibility..."
	@go version
	@echo "📦 Building TenangDB for production with Go modules..."
	GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo ${LDFLAGS} -o ${BINARY_NAME} ./cmd

# Clean build artifacts
clean:
	go clean
	rm -f ${BINARY_NAME}

# Run tests
test:
	go test -v ./...

# Install dependencies
deps:
	@echo "🔍 Checking Go version compatibility..."
	@go version
	@echo "📦 Installing Go dependencies..."
	GO111MODULE=on go mod tidy
	GO111MODULE=on go mod download

# Install the application
install: build
	sudo cp ${BINARY_NAME} /usr/local/bin/
	sudo mkdir -p /etc/tenangdb /var/log/tenangdb /var/backups
	sudo cp configs/config.yaml /etc/tenangdb/config.yaml.example
	sudo ./scripts/install.sh

# Uninstall the application
uninstall:
	sudo systemctl stop tenangdb.timer || true
	sudo systemctl disable tenangdb.timer || true
	sudo rm -f /etc/systemd/system/tenangdb.service
	sudo rm -f /etc/systemd/system/tenangdb.timer
	sudo rm -rf /opt/tenangdb
	sudo systemctl daemon-reload

# Run the application with default config
run: build
	./${BINARY_NAME} --config configs/config.yaml

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Check for security issues
security:
	gosec ./...

# Test dependencies
test-deps:
	./scripts/test-dependencies.sh

# Install dependencies automatically
install-deps:
	@echo "Installing TenangDB dependencies..."
	./scripts/install-dependencies.sh

# Check dependencies without installing
check-deps:
	@echo "Checking TenangDB dependencies..."
	./scripts/install-dependencies.sh --check-only

# Setup for Ubuntu 18.04 (handles Go version issues)
setup-ubuntu-18.04:
	@echo "Setting up TenangDB for Ubuntu 18.04..."
	./scripts/setup-ubuntu-18.04.sh

# Build Docker image
docker-build:
	docker build -t ${BINARY_NAME}:${VERSION} .

# Show help
help:
	@echo "Available targets:"
	@echo "  build      - Build the application"
	@echo "  build-prod - Build for production"
	@echo "  clean      - Clean build artifacts"
	@echo "  test       - Run tests"
	@echo "  deps       - Install dependencies"
	@echo "  install    - Install the application as systemd service"
	@echo "  uninstall  - Uninstall the application"
	@echo "  run        - Run the application"
	@echo "  fmt        - Format code"
	@echo "  lint       - Lint code"
	@echo "  security   - Check for security issues"
	@echo "  test-deps  - Test required dependencies"
	@echo "  install-deps - Install dependencies automatically"
	@echo "  check-deps - Check dependencies without installing"
	@echo "  setup-ubuntu-18.04 - Setup for Ubuntu 18.04 (fixes Go version)"
	@echo "  docker-build - Build Docker image"