# TenangDB

🛡️ **Backup yang Bikin Tenang** - MySQL backup solution with auto-discovery and cloud integration.

*Zero-configuration backup system that just works.*

## ⚡ Installation

```bash
# One-liner install
curl -sSL https://go.ainun.cloud/tenangdb-install.sh | bash

# Install dependencies
brew install mydumper rclone mysql-client  # macOS
sudo apt install mydumper rclone mysql-client  # Ubuntu
sudo dnf install mydumper rclone mysql  # CentOS/Fedora
```

## 🚀 Quick Start

```bash
# 1. Get config template
curl -L https://go.ainun.cloud/tenangdb-config.yaml.example -o config.yaml

# 2. Edit with your database credentials
nano config.yaml

# 3. Run backup
tenangdb backup
```

## ⚙️ Basic Config

```yaml
database:
  host: your_mysql_host
  username: your_username
  password: your_password
  mydumper:
    enabled: true

backup:
  databases:
    - your_database1
    - your_database2

# Optional: Cloud upload
upload:
  enabled: false
  destination: "your-remote:backup-folder"
```

**Auto-Discovery Features:**
- Binary paths (mydumper, myloader, rclone, mysql)
- Backup directories (platform-specific)
- Log locations and optimal settings

## 📋 Commands

```bash
tenangdb backup                    # Backup all databases
tenangdb backup --databases db1,db2   # Backup specific databases
tenangdb restore --backup-path /path/to/backup --database target_db
tenangdb cleanup --force           # Clean old backups
tenangdb config                   # Show configuration
```

## 🔧 Advanced

<details>
<summary><strong>MySQL User Setup</strong></summary>

```sql
-- Create dedicated user
CREATE USER 'tenangdb'@'%' IDENTIFIED BY 'secure_password';
GRANT SELECT, SHOW DATABASES, SHOW VIEW, LOCK TABLES, EVENT, TRIGGER, ROUTINE, RELOAD, REPLICATION CLIENT ON *.* TO 'tenangdb'@'%';
GRANT INSERT, UPDATE, DELETE, CREATE, DROP, ALTER, INDEX, REFERENCES, CREATE TEMPORARY TABLES, CREATE VIEW ON *.* TO 'tenangdb'@'%';
FLUSH PRIVILEGES;
```
</details>

<details>
<summary><strong>Cloud Storage Setup</strong></summary>

```bash
# Configure rclone
rclone config

# Test connection
rclone lsf your-remote:

# Enable in config.yaml
upload:
  enabled: true
  destination: "your-remote:database-backups"
```
</details>

<details>
<summary><strong>Production Deployment</strong></summary>

```bash
# Install system-wide
curl -L https://go.ainun.cloud/tenangdb-latest -o tenangdb
sudo mv tenangdb /usr/local/bin/ && sudo chmod +x /usr/local/bin/tenangdb

# Setup config
sudo mkdir -p /etc/tenangdb
curl -L https://go.ainun.cloud/tenangdb-config.yaml.example | sudo tee /etc/tenangdb/config.yaml
sudo nano /etc/tenangdb/config.yaml
```
</details>

## 📋 Details

<details>
<summary><strong>Directory Structure</strong></summary>

```
Backups: ~/Library/Application Support/TenangDB/backups/ (macOS)
         ~/.local/share/tenangdb/backups/ (Linux)

Structure: {backup-dir}/{database}/{YYYY-MM}/{backup-timestamp}/
Cloud:     {destination}/{database}/{YYYY-MM}/{backup-timestamp}/
```
</details>

<details>
<summary><strong>Compatibility</strong></summary>

**mydumper:** v0.9.1+ (Ubuntu 18.04) to v0.19.3+ (macOS Homebrew)  
**MySQL:** 5.7+, 8.0+, MariaDB 10.3+  
**Platforms:** macOS (Intel/Apple Silicon), Linux (Ubuntu/CentOS/Debian/Fedora)
</details>

## 📚 Links

**Documentation:** [Installation Guide](INSTALL.md) • [MySQL Setup](MYSQL_USER_SETUP.md) • [Production Deployment](PRODUCTION_DEPLOYMENT.md) • [Config Reference](config.yaml.example)

<details>
<summary><strong>Troubleshooting</strong></summary>

**Binary not found:**
```bash
which tenangdb || curl -sSL https://go.ainun.cloud/tenangdb-install.sh | bash
```

**Dependencies missing:**
```bash
which mydumper myloader rclone mysql
# Install: brew/apt/dnf install mydumper rclone mysql-client
```

**Connection failed:**
```bash
mysql -h your_host -u your_user -p
SHOW GRANTS FOR 'your_user'@'%';
```

**Debug mode:**
```bash
tenangdb backup --log-level debug --dry-run
```
</details>

---

**Support:** [Issues](https://github.com/abdullahainun/tenangdb/issues) • **License:** [MIT](LICENSE)