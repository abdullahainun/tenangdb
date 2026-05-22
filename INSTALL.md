# TenangDB Installation Guide

## Docker (Recommended)

```bash
# Clone the repository
git clone https://github.com/abdullahainun/tenangdb.git
cd tenangdb

# Copy and configure
cp configs/config.yaml config.yaml
# Edit config.yaml with your MySQL credentials

# Build and run
make docker-build
make docker-up

# Run setup wizard (first time only)
docker compose run --rm tenangdb init

# Run backup
docker compose exec tenangdb backup
```

Or pull the pre-built image:

```bash
docker run -it --rm \
  -v $(pwd)/config.yaml:/config.yaml:ro \
  -v $(pwd)/backups:/backups \
  ghcr.io/abdullahainun/tenangdb:latest init

docker run --rm \
  -v $(pwd)/config.yaml:/config.yaml:ro \
  -v $(pwd)/backups:/backups \
  ghcr.io/abdullahainun/tenangdb:latest backup
```

## Development (Go)

```bash
# Build from source
go build -o tenangdb ./cmd
go build -o tenangdb-exporter ./cmd/tenangdb-exporter

# Run
./tenangdb init
./tenangdb backup
```

## Verify Installation

```bash
./tenangdb --version
./tenangdb backup --dry-run
```

**Next Steps:** See [PRODUCTION_DEPLOYMENT.md](PRODUCTION_DEPLOYMENT.md) for production setup.
