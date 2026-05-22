# Installation

## Docker (Recommended)

See [DOCKER.md](DOCKER.md) for full setup guide.

```bash
git clone https://github.com/abdullahainun/tenangdb.git
cd tenangdb
cp config.yaml.example config.yaml
make docker-build
```

## Development

```bash
go build -o tenangdb ./cmd
./tenangdb --help
```
