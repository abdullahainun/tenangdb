BINARY_NAME=db-backup-tool
VERSION=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.1.0")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}"

.PHONY: build clean test install uninstall deps

# Build the application
build:
	go build ${LDFLAGS} -o ${BINARY_NAME} cmd/main.go

# Build for production (with optimizations)
build-prod:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo ${LDFLAGS} -o ${BINARY_NAME} cmd/main.go

# Clean build artifacts
clean:
	go clean
	rm -f ${BINARY_NAME}

# Run tests
test:
	go test -v ./...

# Install dependencies
deps:
	go mod tidy
	go mod download

# Install the application
install: build
	sudo ./scripts/install.sh

# Uninstall the application
uninstall:
	sudo systemctl stop db-backup.timer || true
	sudo systemctl disable db-backup.timer || true
	sudo rm -f /etc/systemd/system/db-backup.service
	sudo rm -f /etc/systemd/system/db-backup.timer
	sudo rm -rf /opt/db-backup-tool
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
	@echo "  docker-build - Build Docker image"