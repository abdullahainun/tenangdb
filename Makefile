BINARY_NAME=tenangdb
EXPORTER_BINARY_NAME=tenangdb-exporter
VERSION=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.1.0")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}"

.PHONY: build build-exporter build-all clean test deps fmt lint security docker-build docker-up docker-down help

build:
	go build ${LDFLAGS} -o ${BINARY_NAME} ./cmd

build-exporter:
	go build ${LDFLAGS} -o ${EXPORTER_BINARY_NAME} ./cmd/tenangdb-exporter

build-all: build build-exporter

clean:
	go clean
	rm -f ${BINARY_NAME} ${EXPORTER_BINARY_NAME}

test:
	go test -v ./...

deps:
	go mod tidy
	go mod download

fmt:
	go fmt ./...

lint:
	golangci-lint run

security:
	gosec ./...

docker-build:
	docker build -t ${BINARY_NAME}:${VERSION} .

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

help:
	@echo "Available targets:"
	@echo "  build       - Build the main tenangdb application"
	@echo "  build-exporter - Build the tenangdb-exporter application"
	@echo "  build-all   - Build both applications"
	@echo "  clean       - Clean build artifacts"
	@echo "  test        - Run tests"
	@echo "  deps        - Install Go dependencies"
	@echo "  fmt         - Format code"
	@echo "  lint        - Lint code"
	@echo "  security    - Check for security issues"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-up    - Start services with docker compose"
	@echo "  docker-down  - Stop services"
	@echo "  docker-logs  - Follow logs"
