# Installation

## Docker (Recommended)

See [DOCKER.md](DOCKER.md) for full setup guide.

```bash
mkdir tenangdb && cd tenangdb
cp /path/to/config.yaml.example config.yaml
docker compose pull
```

## Development

```bash
git clone https://github.com/abdullahainun/tenangdb.git
cd tenangdb
go build -o tenangdb ./cmd
./tenangdb --help
```
