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

# Build the application with static linking and version info
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-extldflags '-static' -X main.Version=${VERSION} -X main.CommitSHA=${COMMIT_SHA}" \
    -o tenangdb cmd/main.go

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

# Install rclone
RUN curl -O https://downloads.rclone.org/rclone-current-linux-amd64.zip && \
    unzip rclone-current-linux-amd64.zip && \
    cp rclone-*/rclone /usr/bin/ && \
    chown root:root /usr/bin/rclone && \
    chmod 755 /usr/bin/rclone && \
    rm -rf rclone-*

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /app/tenangdb /tenangdb

# Create non-root user for security
RUN useradd -u 1001 -m -s /bin/bash tenangdb

# Create simple entrypoint script
RUN echo '#!/bin/bash\nexec /tenangdb "$@"' > /entrypoint.sh && chmod +x /entrypoint.sh

# Run as non-root by default, but can be overridden with --user
USER 1001:1001

# Expose port if needed (adjust according to your app)
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["/entrypoint.sh"]