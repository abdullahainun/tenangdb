#!/bin/bash

# TenangDB Installation Script
# Automatically detects platform and installs the latest release

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# GitHub repository
REPO="abdullahainun/tenangdb"
BINARY_NAME="tenangdb"
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

# Download and install binary
download_and_install() {
    local binary_name="${BINARY_NAME}-${PLATFORM}"
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
        mv "$temp_file" "$INSTALL_DIR/$BINARY_NAME"
    else
        print_status "Moving binary requires sudo permissions..."
        sudo mv "$temp_file" "$INSTALL_DIR/$BINARY_NAME"
    fi
    
    print_success "TenangDB installed successfully to $INSTALL_DIR/$BINARY_NAME"
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
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        local installed_version=$($BINARY_NAME version 2>/dev/null | head -n1 || echo "unknown")
        print_success "Installation verified: $installed_version"
    else
        print_warning "Binary installed but not found in PATH"
        if [ "$INSTALL_DIR" != "$HOME/.local/bin" ]; then
            print_warning "Add $INSTALL_DIR to your PATH or use full path: $INSTALL_DIR/$BINARY_NAME"
        fi
    fi
}

# Show next steps
show_next_steps() {
    echo
    print_success "TenangDB installation completed!"
    echo
    echo "Next steps:"
    echo "1. Install dependencies:"
    echo "   # macOS (Homebrew)"
    echo "   brew install mydumper rclone mysql-client"
    echo
    echo "   # Ubuntu/Debian"
    echo "   sudo apt update && sudo apt install mydumper rclone mysql-client"
    echo
    echo "   # CentOS/RHEL/Fedora"
    echo "   sudo dnf install mydumper rclone mysql"
    echo
    echo "2. Download example configuration:"
    echo "   curl -L https://go.ainun.cloud/tenangdb-config.yaml.example -o config.yaml"
    echo
    echo "3. Edit configuration with your database credentials:"
    echo "   nano config.yaml"
    echo
    echo "4. Run your first backup:"
    echo "   $BINARY_NAME backup"
    echo
    echo "5. Get help:"
    echo "   $BINARY_NAME --help"
    echo
    print_success "Documentation: https://github.com/$REPO"
}

# Main installation process
main() {
    echo "TenangDB Installation Script"
    echo "============================"
    echo
    
    detect_platform
    check_permissions
    get_latest_version
    download_and_install
    add_to_path
    verify_installation
    show_next_steps
}

# Run main function
main "$@"