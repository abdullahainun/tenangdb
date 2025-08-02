#!/bin/bash

# TenangDB Uninstallation Script
# Safely removes TenangDB and all associated components
# 
# Usage:
#   curl -sSL https://go.ainun.cloud/tenangdb-uninstall.sh | bash
#   curl -sSL https://go.ainun.cloud/tenangdb-uninstall.sh | bash -s -- --force
#   curl -sSL https://go.ainun.cloud/tenangdb-uninstall.sh | bash -s -- --keep-backups

set -e

# Default options
FORCE=false
KEEP_BACKUPS=false
KEEP_CONFIG=false
DRY_RUN=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --force)
                FORCE=true
                shift
                ;;
            --keep-backups)
                KEEP_BACKUPS=true
                shift
                ;;
            --keep-config)
                KEEP_CONFIG=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                echo "Run with --help for usage information"
                exit 1
                ;;
        esac
    done
}

# Show help
show_help() {
    echo "🗑️ TenangDB Uninstallation Script"
    echo "================================="
    echo
    echo "Usage:"
    echo "  curl -sSL https://go.ainun.cloud/tenangdb-uninstall.sh | bash"
    echo "  curl -sSL https://go.ainun.cloud/tenangdb-uninstall.sh | bash -s -- [OPTIONS]"
    echo
    echo "Options:"
    echo "  --force         Skip confirmation prompts"
    echo "  --keep-backups  Don't remove backup files"
    echo "  --keep-config   Don't remove configuration files"
    echo "  --dry-run       Show what would be removed without actually removing"
    echo "  --help, -h      Show this help message"
    echo
    echo "Examples:"
    echo "  # Interactive uninstall"
    echo "  curl -sSL https://go.ainun.cloud/tenangdb-uninstall.sh | bash"
    echo
    echo "  # Force uninstall without prompts"
    echo "  curl -sSL https://go.ainun.cloud/tenangdb-uninstall.sh | bash -s -- --force"
    echo
    echo "  # Keep backup files"
    echo "  curl -sSL https://go.ainun.cloud/tenangdb-uninstall.sh | bash -s -- --keep-backups"
}

# Detect installation mode
detect_installation_mode() {
    local mode=""
    
    # Check for systemd services
    if systemctl list-unit-files | grep -q "tenangdb.service"; then
        mode="production"
        print_status "Detected: Production installation (systemd services)"
    elif [ -f "/etc/tenangdb/config.yaml" ]; then
        mode="production"
        print_status "Detected: Production installation (system config)"
    elif [ -f "$HOME/.config/tenangdb/config.yaml" ]; then
        mode="personal"
        print_status "Detected: Personal installation (user config)"
    else
        mode="unknown"
        print_warning "Could not detect installation mode"
    fi
    
    echo "$mode"
}

# Show what will be removed
show_removal_preview() {
    local mode="$1"
    
    echo
    print_status "📋 Uninstall Preview - Items to be removed:"
    echo
    
    # Binaries (always removed)
    echo "🔧 Binaries:"
    [ -f "/usr/local/bin/tenangdb" ] && echo "  ✓ /usr/local/bin/tenangdb"
    [ -f "/usr/local/bin/tenangdb-exporter" ] && echo "  ✓ /usr/local/bin/tenangdb-exporter"
    [ -f "/opt/tenangdb/tenangdb" ] && echo "  ✓ /opt/tenangdb/tenangdb"
    [ -f "/opt/tenangdb/tenangdb-exporter" ] && echo "  ✓ /opt/tenangdb/tenangdb-exporter"
    [ -d "/opt/tenangdb" ] && echo "  ✓ /opt/tenangdb/ (directory)"
    
    if [ "$mode" = "production" ]; then
        echo
        echo "🚀 Production Components:"
        
        # Systemd services
        if systemctl list-unit-files | grep -q "tenangdb"; then
            echo "  ⚙️ Systemd Services:"
            systemctl list-unit-files | grep tenangdb | while read -r service _; do
                echo "    ✓ $service"
            done
        fi
        
        # System user
        if id "tenangdb" &>/dev/null; then
            echo "  👤 System user: tenangdb"
        fi
        
        # System directories
        echo "  📁 System Directories:"
        [ -d "/var/log/tenangdb" ] && echo "    ✓ /var/log/tenangdb/"
        [ -d "/var/lib/tenangdb" ] && echo "    ✓ /var/lib/tenangdb/"
        
        if [ "$KEEP_CONFIG" = false ]; then
            [ -d "/etc/tenangdb" ] && echo "    ✓ /etc/tenangdb/ (config)"
        else
            echo "    ⚠️ /etc/tenangdb/ (config) - KEEPING"
        fi
        
        if [ "$KEEP_BACKUPS" = false ]; then
            [ -d "/var/backups/tenangdb" ] && echo "    ✓ /var/backups/tenangdb/ (backups)"
        else
            echo "    ⚠️ /var/backups/tenangdb/ (backups) - KEEPING"
        fi
        
    elif [ "$mode" = "personal" ]; then
        echo
        echo "👤 Personal Components:"
        echo "  📁 User Directories:"
        
        if [ "$KEEP_CONFIG" = false ]; then
            [ -d "$HOME/.config/tenangdb" ] && echo "    ✓ ~/.config/tenangdb/ (config)"
        else
            echo "    ⚠️ ~/.config/tenangdb/ (config) - KEEPING"
        fi
        
        [ -d "$HOME/.local/share/tenangdb" ] && echo "    ✓ ~/.local/share/tenangdb/"
        
        if [ "$KEEP_BACKUPS" = false ]; then
            [ -d "$HOME/backups" ] && echo "    ⚠️ ~/backups/ (might contain TenangDB backups)"
        fi
    fi
    
    echo
}

# Backup important data before removal
backup_data() {
    local mode="$1"
    local backup_dir="$HOME/tenangdb-uninstall-backup-$(date +%Y%m%d_%H%M%S)"
    
    print_status "📦 Creating backup of important data..."
    mkdir -p "$backup_dir"
    
    local backed_up=false
    
    if [ "$mode" = "production" ]; then
        if [ -d "/etc/tenangdb" ]; then
            cp -r "/etc/tenangdb" "$backup_dir/etc-tenangdb"
            backed_up=true
        fi
        if [ -f "/var/lib/tenangdb/metrics.json" ]; then
            cp "/var/lib/tenangdb/metrics.json" "$backup_dir/"
            backed_up=true
        fi
    elif [ "$mode" = "personal" ]; then
        if [ -d "$HOME/.config/tenangdb" ]; then
            cp -r "$HOME/.config/tenangdb" "$backup_dir/config"
            backed_up=true
        fi
        if [ -d "$HOME/.local/share/tenangdb" ]; then
            cp -r "$HOME/.local/share/tenangdb" "$backup_dir/data"
            backed_up=true
        fi
    fi
    
    if [ "$backed_up" = true ]; then
        print_success "Backup created: $backup_dir"
        echo "You can restore from this backup if needed."
    else
        print_warning "No configuration data found to backup"
        rmdir "$backup_dir" 2>/dev/null || true
    fi
    
    echo
}

# Remove systemd services
remove_systemd_services() {
    print_status "🚀 Removing systemd services..."
    
    local services=("tenangdb.service" "tenangdb.timer" "tenangdb-cleanup.service" "tenangdb-cleanup.timer" "tenangdb-exporter.service")
    
    for service in "${services[@]}"; do
        if systemctl list-unit-files | grep -q "$service"; then
            if [ "$DRY_RUN" = false ]; then
                print_status "Stopping and disabling $service"
                systemctl stop "$service" 2>/dev/null || true
                systemctl disable "$service" 2>/dev/null || true
                rm -f "/etc/systemd/system/$service"
            else
                echo "Would remove: $service"
            fi
        fi
    done
    
    if [ "$DRY_RUN" = false ]; then
        systemctl daemon-reload 2>/dev/null || true
        print_success "Systemd services removed"
    fi
}

# Remove system user
remove_system_user() {
    if id "tenangdb" &>/dev/null; then
        if [ "$DRY_RUN" = false ]; then
            print_status "👤 Removing system user 'tenangdb'"
            userdel tenangdb 2>/dev/null || true
            groupdel tenangdb 2>/dev/null || true
            print_success "System user removed"
        else
            echo "Would remove: system user 'tenangdb'"
        fi
    fi
}

# Remove binaries
remove_binaries() {
    print_status "🔧 Removing binaries..."
    
    local binaries=(
        "/usr/local/bin/tenangdb"
        "/usr/local/bin/tenangdb-exporter"
        "/opt/tenangdb/tenangdb"
        "/opt/tenangdb/tenangdb-exporter"
    )
    
    for binary in "${binaries[@]}"; do
        if [ -f "$binary" ]; then
            if [ "$DRY_RUN" = false ]; then
                rm -f "$binary"
                print_success "Removed $binary"
            else
                echo "Would remove: $binary"
            fi
        fi
    done
    
    # Remove /opt/tenangdb directory if empty
    if [ -d "/opt/tenangdb" ]; then
        if [ "$DRY_RUN" = false ]; then
            rmdir "/opt/tenangdb" 2>/dev/null || rm -rf "/opt/tenangdb"
            [ ! -d "/opt/tenangdb" ] && print_success "Removed /opt/tenangdb directory"
        else
            echo "Would remove: /opt/tenangdb directory"
        fi
    fi
}

# Remove directories
remove_directories() {
    local mode="$1"
    
    print_status "📁 Removing directories..."
    
    if [ "$mode" = "production" ]; then
        # System directories
        local dirs=(
            "/var/log/tenangdb"
            "/var/lib/tenangdb"
        )
        
        # Add conditional directories
        [ "$KEEP_CONFIG" = false ] && dirs+=("/etc/tenangdb")
        [ "$KEEP_BACKUPS" = false ] && dirs+=("/var/backups/tenangdb")
        
        for dir in "${dirs[@]}"; do
            if [ -d "$dir" ]; then
                if [ "$DRY_RUN" = false ]; then
                    rm -rf "$dir"
                    print_success "Removed $dir"
                else
                    echo "Would remove: $dir"
                fi
            fi
        done
        
    elif [ "$mode" = "personal" ]; then
        # User directories
        local dirs=(
            "$HOME/.local/share/tenangdb"
        )
        
        # Add conditional directories
        [ "$KEEP_CONFIG" = false ] && dirs+=("$HOME/.config/tenangdb")
        
        for dir in "${dirs[@]}"; do
            if [ -d "$dir" ]; then
                if [ "$DRY_RUN" = false ]; then
                    rm -rf "$dir"
                    print_success "Removed $dir"
                else
                    echo "Would remove: $dir"
                fi
            fi
        done
    fi
}

# Show confirmation prompt
show_confirmation() {
    local mode="$1"
    
    if [ "$FORCE" = true ] || [ "$DRY_RUN" = true ]; then
        return 0
    fi
    
    echo
    print_warning "⚠️  WARNING: This will permanently remove TenangDB!"
    print_warning "⚠️  This action cannot be undone!"
    
    if [ "$KEEP_BACKUPS" = false ]; then
        echo
        print_warning "🔥 BACKUP FILES WILL BE DELETED!"
        print_warning "Make sure you have exported any important databases!"
    fi
    
    echo
    echo -n "Are you sure you want to continue? Type 'yes' to confirm: "
    read -r confirmation
    
    if [ "$confirmation" != "yes" ]; then
        print_status "Uninstall cancelled by user"
        exit 0
    fi
    
    echo
}

# Verify removal
verify_removal() {
    print_status "🔍 Verifying removal..."
    
    local remaining_items=()
    
    # Check binaries
    [ -f "/usr/local/bin/tenangdb" ] && remaining_items+=("/usr/local/bin/tenangdb")
    [ -f "/usr/local/bin/tenangdb-exporter" ] && remaining_items+=("/usr/local/bin/tenangdb-exporter")
    [ -f "/opt/tenangdb/tenangdb" ] && remaining_items+=("/opt/tenangdb/tenangdb")
    
    # Check systemd services
    if systemctl list-unit-files | grep -q "tenangdb"; then
        remaining_items+=("systemd services")
    fi
    
    # Check system user
    if id "tenangdb" &>/dev/null; then
        remaining_items+=("system user 'tenangdb'")
    fi
    
    if [ ${#remaining_items[@]} -eq 0 ]; then
        print_success "✅ TenangDB completely removed!"
    else
        print_warning "⚠️  Some items may still remain:"
        for item in "${remaining_items[@]}"; do
            echo "  - $item"
        done
        echo
        print_warning "You may need to remove these manually"
    fi
}

# Main uninstall function
main() {
    echo "🗑️ TenangDB Uninstallation Script"
    echo "================================="
    echo
    
    # Parse arguments
    parse_args "$@"
    
    # Check if running as root for production uninstall
    local mode
    mode=$(detect_installation_mode)
    
    if [ "$mode" = "production" ] && [ "$EUID" -ne 0 ]; then
        print_error "Production uninstall requires root privileges"
        echo "Please run with sudo:"
        echo "  curl -sSL https://go.ainun.cloud/tenangdb-uninstall.sh | sudo bash"
        exit 1
    fi
    
    # Show what will be removed
    show_removal_preview "$mode"
    
    # Show confirmation
    show_confirmation "$mode"
    
    # Create backup if not dry run
    if [ "$DRY_RUN" = false ]; then
        backup_data "$mode"
    fi
    
    # Perform removal based on mode
    if [ "$mode" = "production" ]; then
        if command -v systemctl >/dev/null 2>&1; then
            remove_systemd_services
        fi
        remove_system_user
    fi
    
    remove_binaries
    remove_directories "$mode"
    
    if [ "$DRY_RUN" = true ]; then
        echo
        print_status "🔍 Dry run completed - no files were actually removed"
        print_status "Run without --dry-run to perform actual uninstall"
    else
        verify_removal
        echo
        print_success "🎉 TenangDB uninstallation completed!"
        echo
        print_status "Thank you for using TenangDB! 👋"
        echo "If you need to reinstall:"
        echo "  curl -sSL https://go.ainun.cloud/tenangdb-install.sh | sudo bash"
    fi
}

# Run main function with all arguments
main "$@"