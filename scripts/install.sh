#!/bin/bash

set -e

# Configuration
SERVICE_NAME="db-backup"
INSTALL_DIR="/opt/db-backup-tool"
CONFIG_DIR="/etc/db-backup-tool"
LOG_DIR="/var/log/db-backup-tool"

echo "Installing Database Backup Tool..."

# Create directories
sudo mkdir -p "$INSTALL_DIR"
sudo mkdir -p "$CONFIG_DIR"
sudo mkdir -p "$LOG_DIR"

# Copy binary
sudo cp ./db-backup-tool "$INSTALL_DIR/"
sudo chmod +x "$INSTALL_DIR/db-backup-tool"

# Copy configuration
sudo cp ./configs/config.yaml "$CONFIG_DIR/"

# Copy systemd files
sudo cp ./scripts/db-backup.service /etc/systemd/system/
sudo cp ./scripts/db-backup.timer /etc/systemd/system/
sudo cp ./scripts/db-backup-cleanup.service /etc/systemd/system/
sudo cp ./scripts/db-backup-cleanup.timer /etc/systemd/system/

# Set permissions
sudo chown -R root:root "$INSTALL_DIR"
sudo chown -R root:root "$CONFIG_DIR"
sudo chown -R root:root "$LOG_DIR"

# Reload systemd
sudo systemctl daemon-reload

# Enable and start timers
sudo systemctl enable db-backup.timer
sudo systemctl start db-backup.timer
sudo systemctl enable db-backup-cleanup.timer
sudo systemctl start db-backup-cleanup.timer

echo "Installation completed successfully!"
echo "Service status:"
sudo systemctl status db-backup.timer
echo ""
echo "Cleanup service status:"
sudo systemctl status db-backup-cleanup.timer

echo ""
echo "Commands:"
echo "  Run backup manually:        sudo systemctl start db-backup.service"
echo "  Run cleanup manually:       sudo systemctl start db-backup-cleanup.service"
echo "  Test cleanup (dry-run):     sudo /opt/db-backup-tool/db-backup-tool cleanup --dry-run"
echo ""
echo "Logs:"
echo "  Backup logs:                sudo journalctl -u db-backup.service -f"
echo "  Cleanup logs:               sudo journalctl -u db-backup-cleanup.service -f"
echo ""
echo "Configuration file: $CONFIG_DIR/config.yaml"
echo ""
echo "Schedule:"
echo "  Backup: Daily"
echo "  Cleanup: Weekend only (Saturday & Sunday at 2 AM)"