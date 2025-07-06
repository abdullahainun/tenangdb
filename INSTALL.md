# 📖 TenangDB Installation & Setup Guide

## 🛠️ Prerequisites & Dependencies

### **System Requirements**
- Linux/Unix system
- MySQL server access
- Go 1.23+ (for building from source)

### **Required Dependencies**

#### **1. Install mydumper & myloader**
```bash
# Ubuntu/Debian
sudo apt update && sudo apt install mydumper

# CentOS/RHEL/Fedora
sudo dnf install mydumper

# Verify installation
mydumper --version
myloader --version
```

#### **2. Install rclone (for cloud upload)**
```bash
# Official installer (recommended)
curl https://rclone.org/install.sh | sudo bash

# Verify installation
rclone version
```

#### **3. Install MySQL client**
```bash
# Ubuntu/Debian
sudo apt install mysql-client

# CentOS/RHEL/Fedora
sudo dnf install mysql

# Verify installation
mysql --version
```

---

## 🚀 TenangDB Installation

### **Option 1: Build from Source**
```bash
git clone https://github.com/abdullahainun/tenangdb.git
cd tenangdb
go build -o tenangdb cmd/main.go
sudo mv tenangdb /usr/local/bin/
```

### **Option 2: Using Make**
```bash
git clone https://github.com/abdullahainun/tenangdb.git
cd tenangdb
make build
sudo make install
```

### **Verify Installation**
```bash
tenangdb --help
```

---

## ⚙️ Configuration

### **1. Create Configuration Directory**
```bash
sudo mkdir -p /etc/tenangdb
sudo mkdir -p /var/log/tenangdb
sudo mkdir -p /opt/tenangdb/backup
```

### **2. Copy Configuration Template**
```bash
# Copy from project
sudo cp configs/config.yaml /etc/tenangdb/config.yaml

# Edit configuration
sudo nano /etc/tenangdb/config.yaml
```

### **3. Configure Database Connection**
Edit `/etc/tenangdb/config.yaml`:

```yaml
database:
  host: 192.168.1.100       # Your MySQL server IP
  port: 3306
  username: backup_user     # MySQL username
  password: "secure_password"
  timeout: 30

backup:
  directory: /opt/tenangdb/backup
  databases:
    - your_database_1
    - your_database_2

logging:
  level: info
  format: json
  file_path: /var/log/tenangdb/tenangdb.log
```

### **4. Create MySQL Configuration Files**

**For mydumper** (`~/.my.cnf`):
```ini
[client]
host=192.168.1.100
port=3306
user=backup_user
password=secure_password

[mydumper]
single-transaction=true
routines=true
triggers=true
events=true
```

**For myloader** (`~/.my_restore.cnf`):
```ini
[client]
host=192.168.1.100
port=3306
user=restore_user
password=restore_password

[myloader]
overwrite-tables=true
```

### **5. Setup Cloud Storage (Optional)**
```bash
# Configure rclone for your cloud provider
rclone config

# Test connection
rclone lsf your-remote:

# Update config.yaml to enable upload
upload:
  enabled: true
  destination: "your-remote:database-backups/"
```

---

## 🧪 Testing & Verification

### **1. Test Dependencies**
```bash
# Test all dependencies
./scripts/test-dependencies.sh

# Or manually:
mydumper --version
myloader --version
rclone version
mysql --version
```

### **2. Test TenangDB**
```bash
# Test configuration
tenangdb backup --config /etc/tenangdb/config.yaml --dry-run

# Test backup with debug logging
tenangdb backup --config /etc/tenangdb/config.yaml --log-level debug
```

---

## 🔧 Usage Examples

### **Basic Operations**
```bash
# Run backup
tenangdb backup --config /etc/tenangdb/config.yaml

# Run cleanup
tenangdb cleanup --config /etc/tenangdb/config.yaml --force

# Cleanup specific databases
tenangdb cleanup --databases mysql,sys --config /etc/tenangdb/config.yaml --force

# Restore database
tenangdb restore --config /etc/tenangdb/config.yaml --backup-path /path/to/backup --target-database restored_db
```

### **Advanced Usage**
```bash
# Debug mode
tenangdb backup --log-level debug --config /etc/tenangdb/config.yaml

# Dry run mode
tenangdb cleanup --dry-run --config /etc/tenangdb/config.yaml

# Force cleanup (bypass weekend-only)
tenangdb cleanup --force --config /etc/tenangdb/config.yaml

# Age-based cleanup for specific databases
tenangdb cleanup --databases app_db --max-age-days 3 --force
```

---

## 🚀 Production Setup

### **1. Install as System Service**
```bash
# Use installation script
sudo ./scripts/install.sh

# Or manually install systemd files
sudo cp scripts/tenangdb.service /etc/systemd/system/
sudo cp scripts/tenangdb.timer /etc/systemd/system/
sudo cp scripts/tenangdb-cleanup.service /etc/systemd/system/
sudo cp scripts/tenangdb-cleanup.timer /etc/systemd/system/
```

### **2. Create Service User**
```bash
# Create dedicated user (no login shell for security)
sudo useradd -r -s /bin/false -d /opt/tenangdb tenangdb

# Set proper ownership
sudo chown -R tenangdb:tenangdb /opt/tenangdb /var/log/tenangdb /backup
```

### **3. Enable and Start Services**
```bash
# Enable backup timer
sudo systemctl daemon-reload
sudo systemctl enable tenangdb.timer
sudo systemctl start tenangdb.timer

# Enable cleanup timer
sudo systemctl enable tenangdb-cleanup.timer
sudo systemctl start tenangdb-cleanup.timer

# Check status
sudo systemctl status tenangdb.timer
sudo systemctl status tenangdb-cleanup.timer
```

---

## 📊 Monitoring & Logs

### **View Logs**
```bash
# Application logs
tail -f /var/log/tenangdb/tenangdb.log

# Systemd logs
journalctl -u tenangdb -f
journalctl -u tenangdb-cleanup -f
```

### **Metrics Endpoint** (if enabled)
```bash
# View metrics
curl http://localhost:8080/metrics

# Grafana dashboard available in grafana/
```

---

## ❓ Troubleshooting

### **Common Issues**

**1. Permission Denied**
```bash
sudo chmod +x /usr/local/bin/tenangdb
sudo chown -R tenangdb:tenangdb /opt/tenangdb
```

**2. MySQL Connection Failed**
```bash
# Test connection manually
mysql -h192.168.1.100 -ubackup_user -p

# Check MySQL user permissions
GRANT SELECT, LOCK TABLES, SHOW VIEW ON *.* TO 'backup_user'@'%';
FLUSH PRIVILEGES;
```

**3. Dependency Not Found**
```bash
# Check paths in config
which mydumper  # Update binary_path if different
which myloader
which rclone

# Update config.yaml paths accordingly
```

**4. Log Level Issues**
```bash
# Test with explicit log level
tenangdb backup --log-level debug --config /etc/tenangdb/config.yaml

# Check config file log level setting
```

**5. Cloud Upload Issues**
```bash
# Test rclone configuration
rclone config show
rclone lsf your-remote:

# Check upload config in tenangdb
upload:
  enabled: true
  destination: "your-remote:backups/"
```

### **Debug Mode**
```bash
# Enable trace logging for maximum detail
tenangdb backup --log-level trace --config /etc/tenangdb/config.yaml

# Check all configuration values
tenangdb backup --config /etc/tenangdb/config.yaml --dry-run --log-level debug
```

### **Getting Help**
```bash
# Show help
tenangdb --help
tenangdb backup --help
tenangdb cleanup --help
tenangdb restore --help

# Check version
tenangdb version
```

---

## 🔧 Configuration Reference

### **Complete Configuration Example**
See [configs/config.yaml](configs/config.yaml) for a complete configuration example with all available options.

### **Log Levels**
- `panic` - Highest level, logs and then panics
- `fatal` - Logs and then calls os.Exit(1)
- `error` - Error conditions
- `warn` - Warning conditions
- `info` - General informational messages (default)
- `debug` - Debug-level messages
- `trace` - Very fine-grained informational events

### **Cleanup Options**
- `cleanup_uploaded_files` - Clean files after successful upload
- `age_based_cleanup` - Clean files based on age
- `max_age_days` - Maximum age before cleanup
- `verify_cloud_exists` - Verify file exists in cloud before deletion
- `weekend_only` - Only run cleanup on weekends

---

## 🎉 You're Ready!

TenangDB is now installed and configured. Your databases will be automatically backed up according to your schedule, with optional cloud upload and intelligent cleanup.

For production use, monitor the logs and metrics to ensure everything runs smoothly.

### **Next Steps**
1. Set up monitoring with the provided Grafana dashboard
2. Configure alerting for backup failures
3. Test restore procedures regularly
4. Schedule regular configuration reviews

### **Support**
- GitHub Issues: [Report bugs or request features](https://github.com/abdullahainun/tenangdb/issues)
- Documentation: Check the project README and wiki
- Examples: See the `examples/` directory for configuration samples