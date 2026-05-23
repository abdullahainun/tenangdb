FROM golang:1.25-alpine AS builder

ARG VERSION=dev
ARG COMMIT_SHA=unknown

RUN apk add --no-cache git ca-certificates tzdata curl unzip

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN go mod vendor

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -mod=vendor -a -installsuffix cgo \
    -ldflags "-extldflags '-static' -X main.version=${VERSION} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o tenangdb ./cmd

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -mod=vendor -a -installsuffix cgo \
    -ldflags "-extldflags '-static' -X main.version=${VERSION} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o tenangdb-exporter ./cmd/tenangdb-exporter

RUN ARCH=$(uname -m) && \
    if [ "$ARCH" = "aarch64" ]; then RCLONE_ARCH="arm64"; else RCLONE_ARCH="amd64"; fi && \
    curl -sSL "https://downloads.rclone.org/rclone-current-linux-${RCLONE_ARCH}.zip" -o /tmp/rclone.zip && \
    unzip -q /tmp/rclone.zip -d /tmp && \
    mv /tmp/rclone-*/rclone /usr/bin/rclone && \
    chmod 755 /usr/bin/rclone && \
    rm -rf /tmp/rclone-*

FROM ubuntu:24.04

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
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    mydumper \
    mysql-client \
    lsb-release \
    ca-certificates \
    curl \
    gnupg \
    xz-utils \
    bash \
    && curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /etc/apt/trusted.gpg.d/postgresql.gpg \
    && echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list \
    && apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    postgresql-client-17 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /usr/bin/rclone /usr/bin/rclone
COPY --from=builder /app/tenangdb /usr/local/bin/tenangdb
COPY --from=builder /app/tenangdb-exporter /usr/local/bin/tenangdb-exporter

RUN useradd -u 1001 -m -s /bin/bash tenangdb

EXPOSE 9090

ENTRYPOINT ["tenangdb"]
