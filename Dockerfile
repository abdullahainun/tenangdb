# Build stage
FROM golang:1.23-alpine AS builder

# Build arguments for version info
ARG VERSION=dev
ARG COMMIT_SHA=unknown

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build both applications with static linking and version info
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-extldflags '-static' -X main.version=${VERSION} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o tenangdb ./cmd

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-extldflags '-static' -X main.version=${VERSION} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o tenangdb-exporter ./cmd/tenangdb-exporter

# Runtime stage - use stable Ubuntu base
FROM ubuntu:22.04

# Build arguments for labels
ARG VERSION=dev
ARG COMMIT_SHA=unknown

# Add container labels
LABEL org.opencontainers.image.title="TenangDB"
LABEL org.opencontainers.image.description="Backup yang Bikin Tenang - MySQL backup automation tool"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${COMMIT_SHA}"
LABEL org.opencontainers.image.vendor="Ainun Abdullah"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.source="https://github.com/abdullahainun/tenangdb"
LABEL org.opencontainers.image.documentation="https://tenangdb.ainun.cloud"

# Install dependencies including mydumper
RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
    mydumper \
    mysql-client \
    ca-certificates \
    tzdata \
    unzip \
    curl \
    bash \
    && rm -rf /var/lib/apt/lists/*

# Install rclone with multi-arch support
RUN ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "arm64" ]; then RCLONE_ARCH="arm64"; else RCLONE_ARCH="amd64"; fi && \
    curl -O https://downloads.rclone.org/rclone-current-linux-${RCLONE_ARCH}.zip && \
    unzip rclone-current-linux-${RCLONE_ARCH}.zip && \
    cp rclone-*/rclone /usr/bin/ && \
    chown root:root /usr/bin/rclone && \
    chmod 755 /usr/bin/rclone && \
    rm -rf rclone-*

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy both binaries
COPY --from=builder /app/tenangdb /tenangdb
COPY --from=builder /app/tenangdb-exporter /tenangdb-exporter

# Create non-root user for security
RUN useradd -u 1001 -m -s /bin/bash tenangdb

# Create intelligent entrypoint script that handles both binaries
RUN echo '#!/bin/bash\n\
if [ "$1" = "tenangdb-exporter" ] || [ "$1" = "exporter" ]; then\n\
    shift\n\
    exec /tenangdb-exporter "$@"\n\
elif [ "$1" = "tenangdb" ]; then\n\
    shift\n\
    exec /tenangdb "$@"\n\
else\n\
    exec /tenangdb "$@"\n\
fi' > /entrypoint.sh && chmod +x /entrypoint.sh

# Run as non-root by default, but can be overridden with --user
USER 1001:1001

# Expose metrics port for exporter
EXPOSE 9090

# Set entrypoint
ENTRYPOINT ["/entrypoint.sh"]