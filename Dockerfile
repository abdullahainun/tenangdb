# Build stage
FROM golang:1.25-alpine AS builder

ARG VERSION=dev
ARG COMMIT_SHA=unknown

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
COPY . ./
RUN if [ ! -d vendor ] || [ -z "$(ls -A vendor 2>/dev/null)" ]; then go mod vendor; fi

RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -a -installsuffix cgo \
    -ldflags "-extldflags '-static' -X main.version=${VERSION} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o tenangdb ./cmd

RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -a -installsuffix cgo \
    -ldflags "-extldflags '-static' -X main.version=${VERSION} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o tenangdb-exporter ./cmd/tenangdb-exporter

# Runtime stage
FROM ubuntu:22.04

ARG VERSION=dev
ARG COMMIT_SHA=unknown

LABEL org.opencontainers.image.title="TenangDB"
LABEL org.opencontainers.image.description="Backup yang Bikin Tenang - MySQL backup automation tool"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${COMMIT_SHA}"
LABEL org.opencontainers.image.vendor="Ainun Abdullah"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.source="https://github.com/abdullahainun/tenangdb"

RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
    mydumper \
    mysql-client \
    xz-utils \
    ca-certificates \
    tzdata \
    unzip \
    curl \
    bash \
    && rm -rf /var/lib/apt/lists/*

RUN ARCH=$(dpkg --print-architecture) && \
    if [ "$ARCH" = "arm64" ]; then RCLONE_ARCH="arm64"; else RCLONE_ARCH="amd64"; fi && \
    curl -sSL https://downloads.rclone.org/rclone-current-linux-${RCLONE_ARCH}.zip -o rclone.zip && \
    unzip -q rclone.zip && \
    cp rclone-*/rclone /usr/bin/ && \
    chown root:root /usr/bin/rclone && \
    chmod 755 /usr/bin/rclone && \
    rm -rf rclone-*

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/tenangdb /usr/local/bin/tenangdb
COPY --from=builder /app/tenangdb-exporter /usr/local/bin/tenangdb-exporter

RUN useradd -u 1001 -m -s /bin/bash tenangdb

EXPOSE 9090

ENTRYPOINT ["tenangdb"]
