# Build stage
FROM golang:1.23-alpine AS builder

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

# Build the application with static linking
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o tenangdb cmd/main.go

# Runtime stage with mydumper
FROM mydumper/mydumper:latest

# Install additional dependencies
RUN dnf update -y && \
    dnf install -y --allowerasing \
    mysql \
    ca-certificates \
    tzdata \
    unzip \
    curl \
    && dnf clean all

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

# Create non-root user and required directories
RUN useradd -u 1001 -m tenangdb && \
    mkdir -p /backups /logs /config && \
    chown -R 1001:1001 /backups /logs /config

USER 1001:1001

# Expose port if needed (adjust according to your app)
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["/tenangdb"]