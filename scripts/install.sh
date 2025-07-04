#!/bin/bash

set -e

# Configuration
SERVICE_NAME="tenangdb"
INSTALL_DIR="/opt/tenangdb"
CONFIG_DIR="/etc/tenangdb"
LOG_DIR="/var/log/tenangdb"

echo "Installing TenangDB..."

# Create directories
sudo mkdir -p "$INSTALL_DIR"
sudo mkdir -p "$CONFIG_DIR"
sudo mkdir -p "$LOG_DIR"

# Copy binary
sudo cp ./tenangdb "$INSTALL_DIR/"
sudo chmod +x "$INSTALL_DIR/tenangdb"

# Copy configuration
sudo cp ./configs/config.yaml "$CONFIG_DIR/"

# Copy systemd files
sudo cp ./scripts/tenangdb.service /etc/systemd/system/
sudo cp ./scripts/tenangdb.timer /etc/systemd/system/
sudo cp ./scripts/tenangdb-cleanup.service /etc/systemd/system/
sudo cp ./scripts/tenangdb-cleanup.timer /etc/systemd/system/

# Set permissions
sudo chown -R root:root "$INSTALL_DIR"
sudo chown -R root:root "$CONFIG_DIR"
sudo chown -R root:root "$LOG_DIR"

# Reload systemd
sudo systemctl daemon-reload

# Enable and start timers
sudo systemctl enable tenangdb.timer
sudo systemctl start tenangdb.timer
sudo systemctl enable tenangdb-cleanup.timer
sudo systemctl start tenangdb-cleanup.timer

echo "Installation completed successfully!"
echo "Service status:"
sudo systemctl status tenangdb.timer
echo ""
echo "Cleanup service status:"
sudo systemctl status tenangdb-cleanup.timer

echo ""
echo "Commands:"
echo "  Run backup manually:        sudo systemctl start tenangdb.service"
echo "  Run cleanup manually:       sudo systemctl start tenangdb-cleanup.service"
echo "  Test cleanup (dry-run):     sudo /opt/tenangdb/tenangdb cleanup --dry-run"
echo ""
echo "Logs:"
echo "  Backup logs:                sudo journalctl -u tenangdb.service -f"
echo "  Cleanup logs:               sudo journalctl -u tenangdb-cleanup.service -f"
echo ""
echo "Configuration file: $CONFIG_DIR/config.yaml"
echo ""
echo "Schedule:"
echo "  Backup: Daily"
echo "  Cleanup: Weekend only (Saturday & Sunday at 2 AM)"