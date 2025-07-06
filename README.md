# TenangDB

🛡️ **Backup yang Bikin Tenang** - A comprehensive MySQL backup solution that lets you sleep peacefully knowing your databases are safe, automatically backed up, uploaded to cloud storage, and ready for instant restore.

*No more worries about database disasters. TenangDB has got you covered.*

## Why Choose TenangDB?

✅ **Peace of Mind**: Automated daily backups with cloud redundancy  
✅ **Disaster Recovery**: One-command restore from any backup point  
✅ **Zero Maintenance**: Set it once, runs forever with intelligent cleanup  
✅ **Enterprise Grade**: Battle-tested with parallel processing & monitoring  

## 🚀 Quick Start

```bash
# 1. Install dependencies
make test-deps

# 2. Build TenangDB
git clone https://github.com/abdullahainun/tenangdb.git
cd tenangdb && make build

# 3. Configure
cp configs/config.yaml /etc/tenangdb/config.yaml
nano /etc/tenangdb/config.yaml  # Update your database credentials

# 4. Run your first backup
./tenangdb backup --config /etc/tenangdb/config.yaml
```

## 📋 Basic Commands

```bash
# Backup databases
./tenangdb backup --config config.yaml

# Restore database
./tenangdb restore --backup-path /backup/db-2025-07-05_10-30-15 --target-database restored_db

# Cleanup old backups
./tenangdb cleanup --config config.yaml
```

## 📚 Documentation

- 📖 **[Installation Guide](INSTALL.md)** - Complete setup instructions
- ⚙️ **[Configuration Guide](configs/README.md)** - Configuration options & examples
- 📊 **[Monitoring Setup](grafana/README.md)** - Grafana dashboard import
- 🔧 **[Commands Reference](docs/COMMANDS.md)** - All available commands & options

## 🎯 Key Features

- **🔄 Dual Backup Engine**: mydumper (parallel) + mysqldump (traditional)
- **📤 Cloud Integration**: Auto-upload to S3, Minio, or any rclone-supported storage
- **🚀 One-Click Restore**: Instant database recovery from any backup
- **🧹 Smart Cleanup**: Age-based cleanup with cloud verification
- **📊 Monitoring Ready**: Prometheus metrics + Grafana dashboard
- **⚙️ Production Ready**: Systemd services, structured logging, security hardening

## 🤝 Support

- 🐛 **Issues**: [GitHub Issues](https://github.com/abdullahainun/tenangdb/issues)
- 📖 **Documentation**: Check the files in `/docs` folder
- 💡 **Feature Requests**: Open an issue with enhancement label

## 📄 License

MIT License - See [LICENSE](LICENSE) file for details