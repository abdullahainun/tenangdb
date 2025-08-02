#!/bin/bash

# TenangDB Installation Script
# Automatically detects platform and installs the latest release
# 
# Usage:
#   curl -sSL https://go.ainun.cloud/tenangdb-install.sh | bash
#   curl -sSL https://go.ainun.cloud/tenangdb-install.sh | bash -s -- --production
#   curl -sSL https://go.ainun.cloud/tenangdb-install.sh | bash -s -- --personal
#   curl -sSL https://go.ainun.cloud/tenangdb-install.sh | bash -s -- --install-only

set -e

# Default options
AUTO_SETUP=""
SKIP_DEPS=false
FORCE_PRODUCTION=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# GitHub repository
REPO="abdullahainun/tenangdb"
BINARY_NAME="tenangdb"
EXPORTER_BINARY_NAME="tenangdb-exporter"
INSTALL_DIR="/usr/local/bin"

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

# Detect platform and architecture
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)
    
    case $os in
        linux*)
            OS="linux"
            ;;
        darwin*)
            OS="darwin"
            ;;
        *)
            print_error "Unsupported operating system: $os"
            exit 1
            ;;
    esac
    
    case $arch in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            print_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
    
    PLATFORM="${OS}-${ARCH}"
    print_status "Detected platform: $PLATFORM"
}

# Check if running as root for system-wide installation
check_permissions() {
    if [ "$EUID" -eq 0 ]; then
        INSTALL_DIR="/usr/local/bin"
        print_status "Installing system-wide to $INSTALL_DIR"
    else
        # Check if /usr/local/bin is writable
        if [ -w "/usr/local/bin" ]; then
            INSTALL_DIR="/usr/local/bin"
            print_status "Installing to $INSTALL_DIR"
        else
            # Fallback to user directory
            INSTALL_DIR="$HOME/.local/bin"
            mkdir -p "$INSTALL_DIR"
            print_warning "Installing to user directory: $INSTALL_DIR"
            print_warning "Make sure $INSTALL_DIR is in your PATH"
        fi
    fi
}

# Get latest release version
get_latest_version() {
    print_status "Fetching latest release information..."
    
    # Try to get latest release from GitHub API
    if command -v curl >/dev/null 2>&1; then
        VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command -v wget >/dev/null 2>&1; then
        VERSION=$(wget -qO- "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        print_error "Neither curl nor wget found. Please install one of them."
        exit 1
    fi
    
    if [ -z "$VERSION" ]; then
        print_error "Failed to get latest version"
        exit 1
    fi
    
    print_status "Latest version: $VERSION"
}

# Download and install a single binary
download_binary() {
    local bin_name="$1"
    local install_name="$2"
    local binary_name="${bin_name}-${PLATFORM}"
    local download_url="https://github.com/$REPO/releases/download/$VERSION/$binary_name"
    local temp_file="/tmp/$binary_name"
    
    print_status "Downloading $binary_name..."
    
    # Download binary
    if command -v curl >/dev/null 2>&1; then
        curl -L "$download_url" -o "$temp_file"
    elif command -v wget >/dev/null 2>&1; then
        wget "$download_url" -O "$temp_file"
    else
        print_error "Neither curl nor wget found"
        exit 1
    fi
    
    # Check if download was successful
    if [ ! -f "$temp_file" ]; then
        print_error "Failed to download $binary_name"
        exit 1
    fi
    
    # Make executable and move to install directory
    chmod +x "$temp_file"
    
    # Move to install directory
    if [ "$EUID" -eq 0 ] || [ -w "$(dirname "$INSTALL_DIR")" ]; then
        mv "$temp_file" "$INSTALL_DIR/$install_name"
    else
        print_status "Moving binary requires sudo permissions..."
        sudo mv "$temp_file" "$INSTALL_DIR/$install_name"
    fi
    
    print_success "$install_name installed successfully to $INSTALL_DIR/$install_name"
}

# Download and install both binaries
download_and_install() {
    download_binary "$BINARY_NAME" "$BINARY_NAME"
    download_binary "$EXPORTER_BINARY_NAME" "$EXPORTER_BINARY_NAME"
}

# Add to PATH if needed
add_to_path() {
    if [ "$INSTALL_DIR" = "$HOME/.local/bin" ]; then
        # Check if ~/.local/bin is already in PATH
        if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
            print_status "Adding $HOME/.local/bin to PATH..."
            
            # Detect shell and add to appropriate rc file
            if [ -n "$ZSH_VERSION" ] || [ "$SHELL" = "/bin/zsh" ] || [ "$SHELL" = "/usr/bin/zsh" ]; then
                # Zsh
                if [ -f "$HOME/.zshrc" ]; then
                    echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.zshrc"
                    print_success "Added to ~/.zshrc"
                fi
            elif [ -n "$BASH_VERSION" ] || [ "$SHELL" = "/bin/bash" ] || [ "$SHELL" = "/usr/bin/bash" ]; then
                # Bash
                if [ -f "$HOME/.bashrc" ]; then
                    echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.bashrc"
                    print_success "Added to ~/.bashrc"
                fi
            else
                # Try to detect from SHELL variable or add to both
                case "$SHELL" in
                    *zsh*)
                        if [ -f "$HOME/.zshrc" ]; then
                            echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.zshrc"
                            print_success "Added to ~/.zshrc"
                        fi
                        ;;
                    *bash*)
                        if [ -f "$HOME/.bashrc" ]; then
                            echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.bashrc"
                            print_success "Added to ~/.bashrc"
                        fi
                        ;;
                    *)
                        # Fallback: try to add to both if they exist
                        if [ -f "$HOME/.bashrc" ]; then
                            echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.bashrc"
                            print_success "Added to ~/.bashrc"
                        fi
                        if [ -f "$HOME/.zshrc" ]; then
                            echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.zshrc"
                            print_success "Added to ~/.zshrc"
                        fi
                        ;;
                esac
            fi
            
            print_warning "Please run 'source ~/.bashrc' or 'source ~/.zshrc' to apply changes"
            print_warning "Or restart your terminal session"
        fi
    fi
}

# Verify installation
verify_installation() {
    local verified_main=false
    local verified_exporter=false
    
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        local installed_version=$($BINARY_NAME version 2>/dev/null | head -n1 || echo "unknown")
        print_success "Main binary verified: $installed_version"
        verified_main=true
    else
        print_warning "Main binary installed but not found in PATH"
    fi
    
    if command -v "$EXPORTER_BINARY_NAME" >/dev/null 2>&1; then
        local exporter_version=$($EXPORTER_BINARY_NAME version 2>/dev/null | head -n1 || echo "unknown")
        print_success "Exporter binary verified: $exporter_version"
        verified_exporter=true
    else
        print_warning "Exporter binary installed but not found in PATH"
    fi
    
    if [ "$verified_main" = false ] || [ "$verified_exporter" = false ]; then
        if [ "$INSTALL_DIR" != "$HOME/.local/bin" ]; then
            print_warning "Add $INSTALL_DIR to your PATH or use full paths:"
            print_warning "  Main: $INSTALL_DIR/$BINARY_NAME"
            print_warning "  Exporter: $INSTALL_DIR/$EXPORTER_BINARY_NAME"
        fi
    fi
}

# Check and install dependencies
install_dependencies() {
    print_status "Checking for required dependencies..."
    
    local missing_deps=()
    local optional_deps=()
    
    # Check for required tools
    if ! command -v mysqldump >/dev/null 2>&1 && ! command -v mysql >/dev/null 2>&1; then
        missing_deps+=(mysql-client)
    fi
    
    if ! command -v mydumper >/dev/null 2>&1; then
        optional_deps+=(mydumper)
    fi
    
    if ! command -v rclone >/dev/null 2>&1; then
        optional_deps+=(rclone)
    fi
    
    # Show status
    local total_missing=$((${#missing_deps[@]} + ${#optional_deps[@]}))
    if [ $total_missing -eq 0 ]; then
        print_success "All dependencies are already installed!"
        return 0
    fi
    
    echo
    echo "📦 Dependency Status:"
    if [ ${#missing_deps[@]} -gt 0 ]; then
        echo "   Required: ${missing_deps[*]} (needed for basic functionality)"
    fi
    if [ ${#optional_deps[@]} -gt 0 ]; then
        echo "   Optional: ${optional_deps[*]} (enhances performance/features)"
    fi
    echo
    
    # Ask user permission for installation
    if [ "$AUTO_SETUP" = "" ]; then
        echo "Install missing dependencies automatically? [Y/n]: "
        read -r response
        if [[ "$response" =~ ^[Nn]$ ]]; then
            print_warning "Skipping dependency installation. You may need to install them manually."
            show_manual_dep_instructions "${missing_deps[@]}" "${optional_deps[@]}"
            return 0
        fi
    fi
    
    # Auto-install dependencies based on platform
    local install_success=true
    
    if [ "$OS" = "darwin" ] && command -v brew >/dev/null 2>&1; then
        print_status "Installing dependencies with Homebrew..."
        install_deps_macos "${missing_deps[@]}" "${optional_deps[@]}"
        install_success=$?
    elif [ "$OS" = "linux" ]; then
        if command -v apt >/dev/null 2>&1; then
            print_status "Installing dependencies with APT..."
            install_deps_apt "${missing_deps[@]}" "${optional_deps[@]}"
            install_success=$?
        elif command -v dnf >/dev/null 2>&1; then
            print_status "Installing dependencies with DNF..."
            install_deps_dnf "${missing_deps[@]}" "${optional_deps[@]}"
            install_success=$?
        elif command -v yum >/dev/null 2>&1; then
            print_status "Installing dependencies with YUM..."
            install_deps_yum "${missing_deps[@]}" "${optional_deps[@]}"
            install_success=$?
        else
            print_warning "No supported package manager found."
            show_manual_dep_instructions "${missing_deps[@]}" "${optional_deps[@]}"
            return 1
        fi
    fi
    
    # Verify installation
    echo
    verify_dependencies_installed
}

# Install dependencies on macOS
install_deps_macos() {
    local deps=("$@")
    local failed_deps=()
    
    for dep in "${deps[@]}"; do
        case $dep in
            mysql-client)
                if ! brew install mysql-client 2>/dev/null; then
                    failed_deps+=(mysql-client)
                fi
                ;;
            mydumper)
                if ! brew install mydumper 2>/dev/null; then
                    print_warning "mydumper not available in Homebrew, trying alternative..."
                    if ! brew tap mydumper/mydumper && brew install mydumper 2>/dev/null; then
                        failed_deps+=(mydumper)
                    fi
                fi
                ;;
            rclone)
                if ! brew install rclone 2>/dev/null; then
                    failed_deps+=(rclone)
                fi
                ;;
        esac
    done
    
    if [ ${#failed_deps[@]} -gt 0 ]; then
        print_warning "Failed to install: ${failed_deps[*]}"
        show_manual_dep_instructions "${failed_deps[@]}"
        return 1
    fi
    return 0
}

# Install dependencies on Ubuntu/Debian
install_deps_apt() {
    local deps=("$@")
    local failed_deps=()
    
    # Update package list
    if ! sudo apt update >/dev/null 2>&1; then
        print_warning "Failed to update package list"
    fi
    
    for dep in "${deps[@]}"; do
        case $dep in
            mysql-client)
                if ! sudo apt install -y mysql-client 2>/dev/null; then
                    # Try mariadb-client as fallback
                    if ! sudo apt install -y mariadb-client 2>/dev/null; then
                        failed_deps+=(mysql-client)
                    fi
                fi
                ;;
            mydumper)
                if ! sudo apt install -y mydumper 2>/dev/null; then
                    print_warning "mydumper not available in repositories, trying manual installation..."
                    if ! install_mydumper_manual_apt; then
                        failed_deps+=(mydumper)
                    fi
                fi
                ;;
            rclone)
                if ! sudo apt install -y rclone 2>/dev/null; then
                    failed_deps+=(rclone)
                fi
                ;;
        esac
    done
    
    if [ ${#failed_deps[@]} -gt 0 ]; then
        print_warning "Failed to install: ${failed_deps[*]}"
        show_manual_dep_instructions "${failed_deps[@]}"
        return 1
    fi
    return 0
}

# Install dependencies on CentOS/RHEL/Fedora (DNF)
install_deps_dnf() {
    local deps=("$@")
    local failed_deps=()
    
    for dep in "${deps[@]}"; do
        case $dep in
            mysql-client)
                if ! sudo dnf install -y mysql 2>/dev/null; then
                    # Try mariadb as fallback
                    if ! sudo dnf install -y mariadb 2>/dev/null; then
                        failed_deps+=(mysql-client)
                    fi
                fi
                ;;
            mydumper)
                if ! sudo dnf install -y mydumper 2>/dev/null; then
                    print_warning "mydumper not available in repositories, trying EPEL..."
                    if ! sudo dnf install -y epel-release && sudo dnf install -y mydumper 2>/dev/null; then
                        failed_deps+=(mydumper)
                    fi
                fi
                ;;
            rclone)
                if ! sudo dnf install -y rclone 2>/dev/null; then
                    failed_deps+=(rclone)
                fi
                ;;
        esac
    done
    
    if [ ${#failed_deps[@]} -gt 0 ]; then
        print_warning "Failed to install: ${failed_deps[*]}"
        show_manual_dep_instructions "${failed_deps[@]}"
        return 1
    fi
    return 0
}

# Install dependencies on older CentOS/RHEL (YUM)
install_deps_yum() {
    local deps=("$@")
    local failed_deps=()
    
    for dep in "${deps[@]}"; do
        case $dep in
            mysql-client)
                if ! sudo yum install -y mysql 2>/dev/null; then
                    # Try mariadb as fallback
                    if ! sudo yum install -y mariadb 2>/dev/null; then
                        failed_deps+=(mysql-client)
                    fi
                fi
                ;;
            mydumper)
                if ! sudo yum install -y mydumper 2>/dev/null; then
                    print_warning "mydumper not available in repositories, trying EPEL..."
                    if ! sudo yum install -y epel-release && sudo yum install -y mydumper 2>/dev/null; then
                        failed_deps+=(mydumper)
                    fi
                fi
                ;;
            rclone)
                if ! sudo yum install -y rclone 2>/dev/null; then
                    failed_deps+=(rclone)
                fi
                ;;
        esac
    done
    
    if [ ${#failed_deps[@]} -gt 0 ]; then
        print_warning "Failed to install: ${failed_deps[*]}"
        show_manual_dep_instructions "${failed_deps[@]}"
        return 1
    fi
    return 0
}

# Manual mydumper installation for APT systems
install_mydumper_manual_apt() {
    local temp_dir=$(mktemp -d)
    cd "$temp_dir"
    
    # Try to download mydumper .deb package
    local ubuntu_version=$(lsb_release -rs 2>/dev/null | cut -d. -f1)
    local deb_url=""
    
    case $ubuntu_version in
        20|21|22|23|24)
            deb_url="https://github.com/mydumper/mydumper/releases/download/v0.15.2-2/mydumper_0.15.2-2.jammy_amd64.deb"
            ;;
        18|19)
            deb_url="https://github.com/mydumper/mydumper/releases/download/v0.15.2-2/mydumper_0.15.2-2.bionic_amd64.deb"
            ;;
        *)
            return 1
            ;;
    esac
    
    if command -v curl >/dev/null 2>&1; then
        curl -L "$deb_url" -o mydumper.deb
    elif command -v wget >/dev/null 2>&1; then
        wget "$deb_url" -O mydumper.deb
    else
        return 1
    fi
    
    if sudo dpkg -i mydumper.deb 2>/dev/null; then
        print_success "mydumper installed from .deb package"
        cd - >/dev/null
        rm -rf "$temp_dir"
        return 0
    else
        cd - >/dev/null
        rm -rf "$temp_dir"
        return 1
    fi
}

# Verify dependencies after installation
verify_dependencies_installed() {
    print_status "Verifying installed dependencies..."
    
    local verified=0
    local total=0
    
    # Check mysql client
    total=$((total + 1))
    if command -v mysqldump >/dev/null 2>&1; then
        local mysql_version=$(mysqldump --version 2>/dev/null | head -n1)
        print_success "✅ mysqldump: $mysql_version"
        verified=$((verified + 1))
    elif command -v mysql >/dev/null 2>&1; then
        local mysql_version=$(mysql --version 2>/dev/null | head -n1)
        print_success "✅ mysql: $mysql_version"
        verified=$((verified + 1))
    else
        print_warning "❌ mysql client: Not found"
    fi
    
    # Check mydumper (optional)
    total=$((total + 1))
    if command -v mydumper >/dev/null 2>&1; then
        local mydumper_version=$(mydumper --version 2>/dev/null | head -n1)
        print_success "✅ mydumper: $mydumper_version"
        verified=$((verified + 1))
    else
        print_warning "⚠️  mydumper: Not found (optional - falls back to mysqldump)"
    fi
    
    # Check rclone (optional)
    total=$((total + 1))
    if command -v rclone >/dev/null 2>&1; then
        local rclone_version=$(rclone version 2>/dev/null | head -n1)
        print_success "✅ rclone: $rclone_version"
        verified=$((verified + 1))
    else
        print_warning "⚠️  rclone: Not found (optional - for cloud uploads)"
    fi
    
    echo
    if [ $verified -eq $total ]; then
        print_success "🎉 All dependencies verified successfully!"
    elif [ $verified -gt 0 ]; then
        print_warning "⚠️  $verified/$total dependencies available. TenangDB will work with limited functionality."
    else
        print_error "❌ Critical dependencies missing. Manual installation required."
        return 1
    fi
}

# Show manual installation instructions
show_manual_dep_instructions() {
    local deps=("$@")
    
    if [ ${#deps[@]} -eq 0 ]; then
        return 0
    fi
    
    echo
    print_warning "📖 Manual Installation Required:"
    echo
    
    for dep in "${deps[@]}"; do
        case $dep in
            mysql-client)
                echo "MySQL Client:"
                echo "  Ubuntu/Debian: sudo apt install mysql-client"
                echo "  CentOS/RHEL:   sudo dnf install mysql"
                echo "  macOS:         brew install mysql-client"
                echo
                ;;
            mydumper)
                echo "MyDumper (optional - for faster parallel backups):"
                echo "  Ubuntu/Debian: sudo apt install mydumper"
                echo "  CentOS/RHEL:   sudo dnf install mydumper"
                echo "  macOS:         brew install mydumper"
                echo "  Manual:        https://github.com/mydumper/mydumper/releases"
                echo
                ;;
            rclone)
                echo "Rclone (optional - for cloud uploads):"
                echo "  Ubuntu/Debian: sudo apt install rclone"
                echo "  CentOS/RHEL:   sudo dnf install rclone"
                echo "  macOS:         brew install rclone"
                echo "  Manual:        https://rclone.org/install/"
                echo
                ;;
        esac
    done
}

# Prompt for setup wizard
prompt_setup_wizard() {
    echo
    print_status "🛡️ TenangDB is now installed!"
    echo
    echo "Choose your setup option:"
    echo "1. 🚀 Quick Production Setup (systemd service + 2-minute wizard)"
    echo "2. 👤 Personal Setup (user configuration)"
    echo "3. 📖 Manual Setup (see documentation)"
    echo "4. ⏭️  Skip setup (install only)"
    echo
    
    while true; do
        read -p "Enter your choice (1-4): " choice
        case $choice in
            1)
                run_production_setup
                break
                ;;
            2)
                run_personal_setup
                break
                ;;
            3)
                show_manual_steps
                break
                ;;
            4)
                print_status "Installation complete. Run 'tenangdb --help' to get started."
                break
                ;;
            *)
                print_warning "Please enter 1, 2, 3, or 4"
                ;;
        esac
    done
}

# Run production setup with systemd
run_production_setup() {
    echo
    print_status "🚀 Starting production setup..."
    print_warning "This will:"
    print_warning "- Create system user 'tenangdb'"
    print_warning "- Install systemd services"
    print_warning "- Set up automated daily backups"
    print_warning "- Configure secure directories"
    echo
    
    # Check if we have a TTY available for interactive input
    if [ -t 0 ] && [ -c /dev/tty ]; then
        # TTY available, run interactive setup
        if [ "$EUID" -ne 0 ]; then
            print_status "Switching to sudo for system setup..."
            sudo "$INSTALL_DIR/$BINARY_NAME" init --deploy-systemd < /dev/tty
        else
            "$INSTALL_DIR/$BINARY_NAME" init --deploy-systemd < /dev/tty
        fi
    else
        # No TTY available (piped from curl), show manual instructions
        print_warning "No interactive terminal detected (likely running via curl pipe)."
        print_warning "Please run the setup manually after installation:"
        echo
        if [ "$EUID" -ne 0 ]; then
            echo "   sudo $BINARY_NAME init --deploy-systemd"
        else
            echo "   $BINARY_NAME init --deploy-systemd"
        fi
        echo
        echo "Or download and run locally:"
        echo "   curl -O https://go.ainun.cloud/tenangdb-install.sh"
        echo "   chmod +x tenangdb-install.sh"
        if [ "$EUID" -ne 0 ]; then
            echo "   sudo ./tenangdb-install.sh"
        else
            echo "   ./tenangdb-install.sh"
        fi
        echo
        print_status "Installation completed. Run the setup command above to configure TenangDB."
        return 0
    fi
    
    if [ $? -eq 0 ]; then
        echo
        print_success "🎉 Production setup completed!"
        echo
        echo "Your TenangDB is now running as a systemd service:"
        echo "  sudo systemctl status tenangdb.timer"
        echo "  sudo journalctl -u tenangdb.service -f"
        echo
    fi
}

# Run personal setup
run_personal_setup() {
    echo
    print_status "👤 Starting personal setup..."
    
    "$INSTALL_DIR/$BINARY_NAME" init
    
    if [ $? -eq 0 ]; then
        echo
        print_success "🎉 Personal setup completed!"
        echo
        echo "Run your first backup:"
        echo "  $BINARY_NAME backup"
        echo
    fi
}

# Show manual setup steps
show_manual_steps() {
    echo
    print_status "📖 Manual Setup Instructions"
    echo
    echo "1. View configuration paths:"
    echo "   $BINARY_NAME config"
    echo
    echo "2. Run interactive setup wizard:"
    echo "   $BINARY_NAME init"
    echo
    echo "3. Or create config manually and run backup:"
    echo "   $BINARY_NAME backup --config /path/to/config.yaml"
    echo
    echo "4. For production with systemd:"
    echo "   sudo $BINARY_NAME init --deploy-systemd"
    echo
    echo "5. View all available commands:"
    echo "   $BINARY_NAME --help"
    echo
    print_success "Documentation: https://github.com/$REPO"
}

# Detect installation mode from execution context
detect_installation_mode() {
    # Check if running with sudo or as root
    if [ "$EUID" -eq 0 ] || [ "$SUDO_USER" != "" ]; then
        AUTO_SETUP="production"
        print_status "🚀 Production mode detected (running with elevated privileges)"
        print_status "   → Will install systemd service and system-wide configuration"
    else
        AUTO_SETUP="personal"
        print_status "👤 Personal mode detected (running as regular user)"
        print_status "   → Will install for current user only"
        
        echo
        echo "💡 For production setup with systemd service, re-run with sudo:"
        echo "   sudo bash <(curl -sSL https://go.ainun.cloud/tenangdb-install.sh)"
        echo
    fi
}

# Parse command line arguments (for backwards compatibility and advanced options)
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --production)
                AUTO_SETUP="production"
                FORCE_PRODUCTION=true
                shift
                ;;
            --personal)
                AUTO_SETUP="personal"
                shift
                ;;
            --install-only)
                AUTO_SETUP="skip"
                shift
                ;;
            --skip-deps)
                SKIP_DEPS=true
                shift
                ;;
            --help|-h)
                echo "🛡️ TenangDB Installation Script"
                echo "==============================="
                echo
                echo "Smart installer that detects your setup intent automatically!"
                echo
                echo "Simple Usage (Recommended):"
                echo "  curl -sSL https://go.ainun.cloud/tenangdb-install.sh | sudo bash    # Production"
                echo "  curl -sSL https://go.ainun.cloud/tenangdb-install.sh | bash         # Personal"
                echo
                echo "Advanced Options:"
                echo "  --production    Force production setup (systemd service)"
                echo "  --personal      Force personal setup (user config)"
                echo "  --install-only  Install binaries only, skip setup"
                echo "  --skip-deps     Skip dependency installation"
                echo "  --help, -h      Show this help message"
                echo
                echo "Examples:"
                echo "  # Automatic mode detection (recommended)"
                echo "  curl -sSL https://go.ainun.cloud/tenangdb-install.sh | sudo bash"
                echo "  curl -sSL https://go.ainun.cloud/tenangdb-install.sh | bash"
                echo
                echo "  # Advanced usage"
                echo "  curl -sSL https://go.ainun.cloud/tenangdb-install.sh | bash -s -- --skip-deps"
                echo "  curl -sSL https://go.ainun.cloud/tenangdb-install.sh | bash -s -- --install-only"
                exit 0
                ;;
            *)
                print_warning "Unknown option: $1"
                shift
                ;;
        esac
    done
}

# Main installation process
main() {
    echo "🛡️ TenangDB Installation Script"
    echo "==============================="
    echo "Secure MySQL backup with intelligent automation"
    echo
    
    # Parse arguments first (for advanced options)
    parse_args "$@"
    
    # Auto-detect installation mode if not explicitly set
    if [ "$AUTO_SETUP" = "" ]; then
        detect_installation_mode
    fi
    
    detect_platform
    check_permissions
    get_latest_version
    download_and_install
    add_to_path
    verify_installation
    
    # Install dependencies unless skipped
    if [ "$SKIP_DEPS" = false ]; then
        install_dependencies
    fi
    
    # Handle setup based on detected/specified mode
    case "$AUTO_SETUP" in
        "production")
            run_production_setup
            ;;
        "personal")
            run_personal_setup
            ;;
        "skip")
            print_status "Installation complete. Run 'tenangdb --help' to get started."
            ;;
        *)
            # Fallback to interactive wizard (shouldn't happen with auto-detection)
            prompt_setup_wizard
            ;;
    esac
}

# Run main function
main "$@"