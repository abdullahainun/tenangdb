#!/bin/bash

# TenangDB Setup Script for Ubuntu 18.04
# This script handles the specific requirements for building TenangDB on Ubuntu 18.04

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local status="$1"
    local message="$2"
    
    case "$status" in
        "INFO")
            echo -e "${BLUE}[INFO]${NC} $message"
            ;;
        "SUCCESS")
            echo -e "${GREEN}[SUCCESS]${NC} $message"
            ;;
        "WARNING")
            echo -e "${YELLOW}[WARNING]${NC} $message"
            ;;
        "ERROR")
            echo -e "${RED}[ERROR]${NC} $message"
            ;;
        "HEADER")
            echo -e "\n${BLUE}=====================================\n$message\n=====================================${NC}"
            ;;
    esac
}

# Function to check if running on Ubuntu 18.04
check_ubuntu_18_04() {
    if [[ ! -f /etc/os-release ]]; then
        print_status "ERROR" "Cannot determine OS version."
        exit 1
    fi
    
    source /etc/os-release
    
    if [[ "$ID" != "ubuntu" ]] || [[ "$VERSION_ID" != "18.04" ]]; then
        print_status "ERROR" "This script is designed specifically for Ubuntu 18.04."
        print_status "INFO" "Detected: $ID $VERSION_ID"
        print_status "INFO" "For other versions, use: ./scripts/install-dependencies.sh"
        exit 1
    fi
    
    print_status "SUCCESS" "Ubuntu 18.04 detected. Proceeding with setup..."
}

# Function to remove old Go version
remove_old_go() {
    print_status "INFO" "Removing old Go version (if present)..."
    
    # Remove system Go
    sudo apt-get remove -y golang-go golang-1.10 golang-1.10-go || true
    
    # Remove any existing Go installation
    sudo rm -rf /usr/local/go
    
    # Clean up PATH
    sudo sed -i '/\/usr\/local\/go\/bin/d' /etc/profile
    
    print_status "SUCCESS" "Old Go version removed"
}

# Function to install new Go version
install_new_go() {
    print_status "INFO" "Installing Go 1.23.1..."
    
    GO_VERSION="1.23.1"
    GO_URL="https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz"
    
    # Download Go
    print_status "INFO" "Downloading Go $GO_VERSION..."
    if curl -fsSL "$GO_URL" -o /tmp/go.tar.gz; then
        print_status "SUCCESS" "Go downloaded successfully"
    else
        print_status "ERROR" "Failed to download Go"
        exit 1
    fi
    
    # Extract Go
    print_status "INFO" "Extracting Go..."
    sudo tar -C /usr/local -xzf /tmp/go.tar.gz
    rm -f /tmp/go.tar.gz
    
    # Add to PATH
    echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee -a /etc/profile
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    
    # Set for current session
    export PATH=$PATH:/usr/local/go/bin
    
    print_status "SUCCESS" "Go $GO_VERSION installed successfully"
}

# Function to verify Go installation
verify_go_installation() {
    print_status "INFO" "Verifying Go installation..."
    
    if command -v go >/dev/null 2>&1; then
        local version=$(go version 2>/dev/null || echo "unknown")
        print_status "SUCCESS" "Go installed: $version"
        
        # Check if Go version is compatible
        local go_version=$(go version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+' | head -1)
        if [[ "$go_version" ]]; then
            local version_num=$(echo "$go_version" | sed 's/\.//g')
            if [[ "$version_num" -lt 123 ]]; then
                print_status "ERROR" "Go version $go_version is still too old for TenangDB (requires 1.23+)"
                return 1
            else
                print_status "SUCCESS" "Go version $go_version is compatible with TenangDB"
            fi
        fi
    else
        print_status "ERROR" "Go installation verification failed"
        print_status "INFO" "You may need to restart your shell or run: source ~/.bashrc"
        return 1
    fi
}

# Function to install other dependencies
install_other_dependencies() {
    print_status "INFO" "Installing other dependencies..."
    
    # Update package lists
    sudo apt-get update -qq
    
    # Install basic dependencies
    sudo apt-get install -y curl wget build-essential
    
    # Install mydumper (try different approaches)
    if ! sudo apt-get install -y mydumper; then
        print_status "WARNING" "mydumper not available in default repos"
        print_status "INFO" "Trying to install from universe repository..."
        
        sudo add-apt-repository universe -y
        sudo apt-get update -qq
        
        if ! sudo apt-get install -y mydumper; then
            print_status "WARNING" "mydumper still not available, trying manual installation..."
            
            # Try to download .deb package
            mydumper_url="https://github.com/mydumper/mydumper/releases/download/v0.12.7-2/mydumper_0.12.7-2.bionic_amd64.deb"
            
            if curl -fsSL "$mydumper_url" -o /tmp/mydumper.deb; then
                sudo dpkg -i /tmp/mydumper.deb || true
                sudo apt-get install -f -y  # Fix dependencies
                rm -f /tmp/mydumper.deb
                print_status "SUCCESS" "mydumper installed from .deb package"
            else
                print_status "ERROR" "Failed to install mydumper"
                print_status "INFO" "You may need to install mydumper manually"
            fi
        fi
    fi
    
    # Install MySQL client
    sudo apt-get install -y mysql-client
    
    # Install rclone
    print_status "INFO" "Installing rclone..."
    curl -fsSL https://rclone.org/install.sh | sudo bash
    
    print_status "SUCCESS" "Dependencies installed"
}

# Function to test build
test_build() {
    print_status "INFO" "Testing TenangDB build..."
    
    # Ensure we're in the right directory
    cd "$(dirname "$0")/.."
    
    # Install Go dependencies
    print_status "INFO" "Installing Go dependencies..."
    GO111MODULE=on go mod tidy
    GO111MODULE=on go mod download
    
    # Test build
    print_status "INFO" "Building TenangDB..."
    if GO111MODULE=on go build -o tenangdb ./cmd; then
        print_status "SUCCESS" "TenangDB build successful!"
        
        # Test the binary
        if ./tenangdb --help >/dev/null 2>&1; then
            print_status "SUCCESS" "TenangDB binary is working correctly"
        else
            print_status "WARNING" "TenangDB binary may have issues"
        fi
    else
        print_status "ERROR" "TenangDB build failed"
        return 1
    fi
}

# Function to show final instructions
show_final_instructions() {
    print_status "HEADER" "Setup Complete!"
    
    print_status "INFO" "TenangDB has been successfully set up on Ubuntu 18.04"
    print_status "INFO" ""
    print_status "INFO" "Next steps:"
    print_status "INFO" "1. Restart your terminal or run: source ~/.bashrc"
    print_status "INFO" "2. Verify Go version: go version"
    print_status "INFO" "3. Build TenangDB: make build"
    print_status "INFO" "4. Install TenangDB: sudo make install"
    print_status "INFO" "5. Configure: /etc/tenangdb/config.yaml"
    print_status "INFO" ""
    print_status "INFO" "If you encounter issues, try:"
    print_status "INFO" "- export PATH=\$PATH:/usr/local/go/bin"
    print_status "INFO" "- GO111MODULE=on go build -o tenangdb ./cmd"
}

# Main execution
main() {
    print_status "HEADER" "TenangDB Setup for Ubuntu 18.04"
    
    check_ubuntu_18_04
    
    # Ask for confirmation
    echo ""
    read -p "This will remove old Go version and install Go 1.23.1. Continue? (y/N): " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_status "INFO" "Setup cancelled by user"
        exit 0
    fi
    
    remove_old_go
    install_new_go
    verify_go_installation
    install_other_dependencies
    test_build
    show_final_instructions
    
    print_status "SUCCESS" "Ubuntu 18.04 setup completed successfully!"
}

# Check if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi