#!/bin/bash

# TenangDB Auto Dependency Installer
# Support Ubuntu 18.04+ (Bionic, Focal, Jammy, Noble), Debian 10+ (Buster, Bullseye, Bookworm), and macOS 10.15+
# This script automatically installs all required dependencies for TenangDB

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script info
SCRIPT_VERSION="1.0.0"
SUPPORTED_UBUNTU_VERSIONS=("18.04" "20.04" "22.04" "24.04")
SUPPORTED_DEBIAN_VERSIONS=("10" "11" "12")
SUPPORTED_MACOS_VERSIONS=("10.15" "11.0" "12.0" "13.0" "13.1" "13.2" "13.3" "13.4" "13.5" "13.6" "13.7" "14.0" "14.1" "14.2" "14.3" "14.4" "14.5" "14.6" "15.0" "15.1" "15.2" "15.3" "15.4" "15.5" "15.6")

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

# Function to check if running as root
check_root() {
    # Skip root check in GitHub Actions or CI environments
    if [[ -n "$GITHUB_ACTIONS" || -n "$CI" ]]; then
        print_status "INFO" "Running in CI environment, skipping root check"
        return 0
    fi
    
    if [[ $EUID -eq 0 ]]; then
        print_status "ERROR" "This script should not be run as root. Please run as regular user."
        print_status "INFO" "The script will use sudo when needed."
        exit 1
    fi
}

# Function to check OS version (Ubuntu/Debian/macOS)
check_os_version() {
    # Check if running on macOS
    if [[ "$(uname)" == "Darwin" ]]; then
        OS_TYPE="macos"
        OS_VERSION=$(sw_vers -productVersion)
        OS_BUILD=$(sw_vers -buildVersion)
        print_status "INFO" "Detected macOS $OS_VERSION (Build: $OS_BUILD)"
        
        # Check if version is supported
        local version_supported=false
        local major_version=$(echo "$OS_VERSION" | cut -d'.' -f1)
        local minor_version=$(echo "$OS_VERSION" | cut -d'.' -f2)
        local current_version="$major_version.$minor_version"
        
        for supported_version in "${SUPPORTED_MACOS_VERSIONS[@]}"; do
            if [[ "$current_version" == "$supported_version"* ]] || [[ "$OS_VERSION" == "$supported_version"* ]]; then
                version_supported=true
                break
            fi
        done
        
        if [[ "$version_supported" == false ]]; then
            print_status "WARNING" "macOS $OS_VERSION is not officially tested."
            print_status "WARNING" "Supported versions: ${SUPPORTED_MACOS_VERSIONS[*]}"
            print_status "WARNING" "Continuing anyway, but some dependencies might fail..."
        fi
        
        return 0
    fi
    
    # Check Linux distributions
    if [[ ! -f /etc/os-release ]]; then
        print_status "ERROR" "Cannot determine OS version. This script is designed for Ubuntu/Debian/macOS."
        exit 1
    fi
    
    source /etc/os-release
    
    if [[ "$ID" == "ubuntu" ]]; then
        OS_TYPE="ubuntu"
        DISTRO="ubuntu"
        OS_VERSION=$(echo "$VERSION_ID")
        print_status "INFO" "Detected Ubuntu $OS_VERSION ($VERSION_CODENAME)"
        
        # Check if version is supported
        local version_supported=false
        for supported_version in "${SUPPORTED_UBUNTU_VERSIONS[@]}"; do
            if [[ "$OS_VERSION" == "$supported_version" ]]; then
                version_supported=true
                break
            fi
        done
        
        if [[ "$version_supported" == false ]]; then
            print_status "WARNING" "Ubuntu $OS_VERSION is not officially tested."
            print_status "WARNING" "Supported versions: ${SUPPORTED_UBUNTU_VERSIONS[*]}"
            print_status "WARNING" "Continuing anyway, but some dependencies might fail..."
        fi
        
    elif [[ "$ID" == "debian" ]]; then
        OS_TYPE="debian"
        DISTRO="debian"
        OS_VERSION=$(echo "$VERSION_ID")
        print_status "INFO" "Detected Debian $OS_VERSION ($VERSION_CODENAME)"
        
        # Handle Debian 10 (EOL) - update to archive.debian.org
        if [[ "$OS_VERSION" == "10" ]]; then
            print_status "WARNING" "Debian 10 is End-of-Life, updating sources to archive repository..."
            setup_debian_10_archives
        fi
        
        # Check if version is supported
        local version_supported=false
        for supported_version in "${SUPPORTED_DEBIAN_VERSIONS[@]}"; do
            if [[ "$OS_VERSION" == "$supported_version" ]]; then
                version_supported=true
                break
            fi
        done
        
        if [[ "$version_supported" == false ]]; then
            print_status "WARNING" "Debian $OS_VERSION is not officially tested."
            print_status "WARNING" "Supported versions: ${SUPPORTED_DEBIAN_VERSIONS[*]}"
            print_status "WARNING" "Continuing anyway, but some dependencies might fail..."
        fi
        
    else
        print_status "ERROR" "This script is designed for Ubuntu/Debian/macOS. Detected: $ID"
        print_status "INFO" "You may need to manually install dependencies for your OS."
        exit 1
    fi
}

# Function to setup Debian 10 archive repositories (EOL)
setup_debian_10_archives() {
    local sources_list="/etc/apt/sources.list"
    local backup_file="/etc/apt/sources.list.backup.$(date +%s)"
    
    print_status "INFO" "Backing up current sources.list to $backup_file"
    sudo cp "$sources_list" "$backup_file"
    
    print_status "INFO" "Updating sources.list to use archive.debian.org"
    
    # Create new sources.list for Debian 10 archive
    cat << 'EOF' | sudo tee "$sources_list" > /dev/null
# Debian 10 (Buster) - Archive repositories
deb http://archive.debian.org/debian buster main contrib non-free
deb http://archive.debian.org/debian-security buster/updates main contrib non-free
EOF
    
    # Remove Release file date checking for archive
    echo 'Acquire::Check-Valid-Until "false";' | sudo tee /etc/apt/apt.conf.d/99archive-check > /dev/null
    
    print_status "SUCCESS" "Debian 10 archive repositories configured"
}

# Function to check and install Homebrew on macOS
check_and_install_homebrew() {
    if [[ "$OS_TYPE" != "macos" ]]; then
        return 0
    fi
    
    print_status "INFO" "Checking for Homebrew..."
    
    if command -v brew >/dev/null 2>&1; then
        print_status "SUCCESS" "Homebrew is already installed"
        # Update Homebrew
        print_status "INFO" "Updating Homebrew..."
        brew update
        print_status "SUCCESS" "Homebrew updated"
        return 0
    fi
    
    print_status "INFO" "Homebrew not found. Installing Homebrew..."
    
    # Install Homebrew
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    
    # Add Homebrew to PATH for different shell profiles
    if [[ -f "/opt/homebrew/bin/brew" ]]; then
        # Apple Silicon Mac
        HOMEBREW_PREFIX="/opt/homebrew"
    elif [[ -f "/usr/local/bin/brew" ]]; then
        # Intel Mac
        HOMEBREW_PREFIX="/usr/local"
    else
        print_status "ERROR" "Homebrew installation failed"
        return 1
    fi
    
    # Add to current session
    export PATH="$HOMEBREW_PREFIX/bin:$PATH"
    
    # Add to shell profiles
    local shell_profiles=("$HOME/.zshrc" "$HOME/.bash_profile" "$HOME/.bashrc")
    for profile in "${shell_profiles[@]}"; do
        if [[ -f "$profile" ]] && ! grep -q "homebrew" "$profile"; then
            echo "# Add Homebrew to PATH" >> "$profile"
            echo "export PATH=\"$HOMEBREW_PREFIX/bin:\$PATH\"" >> "$profile"
        fi
    done
    
    # Verify installation
    if command -v brew >/dev/null 2>&1; then
        print_status "SUCCESS" "Homebrew installed successfully"
    else
        print_status "ERROR" "Homebrew installation verification failed"
        return 1
    fi
}

# Function to update package lists
update_package_lists() {
    if [[ "$OS_TYPE" == "macos" ]]; then
        check_and_install_homebrew
    else
        print_status "INFO" "Updating package lists..."
        sudo apt-get update -qq
        print_status "SUCCESS" "Package lists updated"
    fi
}

# Function to install basic dependencies
install_basic_deps() {
    print_status "INFO" "Installing basic dependencies..."
    
    if [[ "$OS_TYPE" == "macos" ]]; then
        # macOS basic dependencies
        local basic_packages=(
            "curl"
            "wget"
            "gnupg"
        )
        
        for package in "${basic_packages[@]}"; do
            if ! brew list "$package" &> /dev/null; then
                print_status "INFO" "Installing $package..."
                brew install "$package"
            else
                print_status "INFO" "$package is already installed"
            fi
        done
        
    else
        # Linux basic dependencies
        local basic_packages=(
            "curl"
            "wget"
            "software-properties-common"
            "apt-transport-https"
            "ca-certificates"
            "gnupg"
            "lsb-release"
        )
        
        for package in "${basic_packages[@]}"; do
            if ! dpkg -l "$package" &> /dev/null; then
                print_status "INFO" "Installing $package..."
                sudo apt-get install -y "$package"
            else
                print_status "INFO" "$package is already installed"
            fi
        done
    fi
    
    print_status "SUCCESS" "Basic dependencies installed"
}

# Function to install mydumper/myloader
install_mydumper() {
    print_status "INFO" "Checking mydumper/myloader..."
    
    if command -v mydumper >/dev/null 2>&1 && command -v myloader >/dev/null 2>&1; then
        local version=$(mydumper --version 2>/dev/null | head -n1 || echo "unknown")
        print_status "SUCCESS" "mydumper/myloader already installed: $version"
        return 0
    fi
    
    print_status "INFO" "Installing mydumper/myloader..."
    
    if [[ "$OS_TYPE" == "macos" ]]; then
        # macOS installation using Homebrew
        print_status "INFO" "Installing mydumper via Homebrew..."
        if ! brew install mydumper 2>/dev/null; then
            print_status "WARNING" "Failed to install mydumper via Homebrew, trying with tap..."
            if ! brew tap mydumper/homebrew-mydumper && brew install mydumper; then
                print_status "ERROR" "Failed to install mydumper via Homebrew"
                return 1
            fi
        fi
        
    elif [[ "$OS_TYPE" == "ubuntu" ]]; then
        # Ubuntu-specific installation
        if [[ "$OS_VERSION" == "18.04" ]]; then
            print_status "INFO" "Using special installation method for Ubuntu 18.04..."
            
            # Enable universe repository first
            sudo add-apt-repository universe -y
            sudo apt-get update -qq
            
            # Try to install from repos with universe enabled
            if ! sudo apt-get install -y mydumper; then
                print_status "WARNING" "Package mydumper not available in repos for Ubuntu 18.04"
                print_status "INFO" "Attempting to download from GitHub releases..."
                
                # Download mydumper binary for Ubuntu 18.04
                local mydumper_url="https://github.com/mydumper/mydumper/releases/download/v0.12.7-2/mydumper_0.12.7-2.bionic_amd64.deb"
                
                if curl -fsSL "$mydumper_url" -o /tmp/mydumper.deb; then
                    sudo dpkg -i /tmp/mydumper.deb || true
                    sudo apt-get install -f -y  # Fix dependencies
                    rm -f /tmp/mydumper.deb
                else
                    print_status "ERROR" "Failed to download mydumper for Ubuntu 18.04"
                    print_status "INFO" "You may need to install mydumper manually"
                    return 1
                fi
            fi
        else
            # For newer Ubuntu versions (20.04, 22.04, 24.04+)
            # Enable universe repository first
            sudo add-apt-repository universe -y
            sudo apt-get update -qq
            
            if ! sudo apt-get install -y mydumper; then
                print_status "WARNING" "mydumper not available in universe repository for Ubuntu $OS_VERSION"
                
                # Try different approaches based on version
                if [[ "$OS_VERSION" == "24.04" ]]; then
                    print_status "INFO" "Ubuntu 24.04 detected - trying alternative installation methods..."
                    # Try installing from official GitHub releases
                    install_mydumper_from_github_release
                else
                    print_status "INFO" "Attempting to install build dependencies and compile from source..."
                    install_mydumper_from_source
                fi
            fi
        fi
        
    elif [[ "$OS_TYPE" == "debian" ]]; then
        # Debian-specific installation
        if [[ "$OS_VERSION" == "10" ]]; then
            # Debian 10 (Buster) - mydumper might not be available in default repos
            print_status "INFO" "Installing mydumper for Debian 10..."
            
            if ! sudo apt-get install -y mydumper; then
                print_status "WARNING" "mydumper not available in default repos for Debian 10"
                print_status "INFO" "Trying to install from backports..."
                
                # Add backports repository using archive for Debian 10
                echo "deb http://archive.debian.org/debian buster-backports main" | sudo tee -a /etc/apt/sources.list
                sudo apt-get update -qq
                
                if ! sudo apt-get install -y -t buster-backports mydumper; then
                    print_status "WARNING" "mydumper not available in backports, trying to build from source..."
                    install_mydumper_from_source
                fi
            fi
            
        elif [[ "$OS_VERSION" == "11" ]]; then
            # Debian 11 (Bullseye)
            print_status "INFO" "Installing mydumper for Debian 11..."
            
            if ! sudo apt-get install -y mydumper; then
                print_status "WARNING" "mydumper not available in default repos, trying backports..."
                echo "deb http://deb.debian.org/debian bullseye-backports main" | sudo tee -a /etc/apt/sources.list
                sudo apt-get update -qq
                sudo apt-get install -y -t bullseye-backports mydumper || install_mydumper_from_source
            fi
            
        elif [[ "$OS_VERSION" == "12" ]]; then
            # Debian 12 (Bookworm)
            print_status "INFO" "Installing mydumper for Debian 12..."
            
            if ! sudo apt-get install -y mydumper; then
                print_status "WARNING" "mydumper not available in default repos, trying backports..."
                echo "deb http://deb.debian.org/debian bookworm-backports main" | sudo tee -a /etc/apt/sources.list
                sudo apt-get update -qq
                sudo apt-get install -y -t bookworm-backports mydumper || install_mydumper_from_source
            fi
        fi
    fi
    
    # Verify installation
    if command -v mydumper >/dev/null 2>&1 && command -v myloader >/dev/null 2>&1; then
        local version=$(mydumper --version 2>/dev/null | head -n1 || echo "unknown")
        print_status "SUCCESS" "mydumper/myloader installed: $version"
    else
        print_status "ERROR" "Failed to install mydumper/myloader"
        return 1
    fi
}

# Function to install mydumper from GitHub releases (for Ubuntu 24.04+)
install_mydumper_from_github_release() {
    print_status "INFO" "Installing mydumper from GitHub releases..."
    
    local arch=$(uname -m)
    local mydumper_url=""
    
    # Determine the correct download URL based on architecture
    if [[ "$arch" == "x86_64" ]]; then
        # Try to download the latest Ubuntu binary
        mydumper_url="https://github.com/mydumper/mydumper/releases/latest/download/mydumper_0.12.7-2.jammy_amd64.deb"
    else
        print_status "WARNING" "No pre-built binary available for architecture: $arch"
        print_status "INFO" "Falling back to source compilation..."
        install_mydumper_from_source
        return $?
    fi
    
    print_status "INFO" "Downloading mydumper from GitHub releases..."
    
    if curl -fsSL "$mydumper_url" -o /tmp/mydumper.deb; then
        print_status "INFO" "Installing downloaded mydumper package..."
        
        # Install package and fix dependencies
        if sudo dpkg -i /tmp/mydumper.deb 2>/dev/null; then
            print_status "SUCCESS" "mydumper installed successfully"
        else
            print_status "INFO" "Fixing dependencies..."
            sudo apt-get install -f -y
            
            # Try again
            if sudo dpkg -i /tmp/mydumper.deb; then
                print_status "SUCCESS" "mydumper installed successfully after fixing dependencies"
            else
                print_status "WARNING" "Failed to install downloaded package, trying source compilation..."
                install_mydumper_from_source
                return $?
            fi
        fi
        
        # Cleanup
        rm -f /tmp/mydumper.deb
        
    else
        print_status "WARNING" "Failed to download mydumper from GitHub releases"
        print_status "INFO" "Falling back to source compilation..."
        install_mydumper_from_source
        return $?
    fi
}

# Function to install mydumper from source (fallback)
install_mydumper_from_source() {
    print_status "INFO" "Installing mydumper from source..."
    
    # Install build dependencies
    sudo apt-get install -y build-essential cmake libmysqlclient-dev libglib2.0-dev libpcre3-dev zlib1g-dev
    
    # Download and build mydumper
    local mydumper_version="0.12.7-2"
    local mydumper_url="https://github.com/mydumper/mydumper/archive/refs/tags/v${mydumper_version}.tar.gz"
    
    cd /tmp
    if curl -fsSL "$mydumper_url" -o mydumper.tar.gz; then
        tar -xzf mydumper.tar.gz
        cd "mydumper-${mydumper_version}"
        
        cmake .
        make
        sudo make install
        
        # Clean up
        cd /
        rm -rf /tmp/mydumper*
        
        # Update library cache
        sudo ldconfig
    else
        print_status "ERROR" "Failed to download mydumper source"
        return 1
    fi
}

# Function to install MySQL client
install_mysql_client() {
    print_status "INFO" "Checking MySQL client..."
    
    if command -v mysql >/dev/null 2>&1; then
        local version=$(mysql --version 2>/dev/null || echo "unknown")
        print_status "SUCCESS" "MySQL client already installed: $version"
        return 0
    fi
    
    print_status "INFO" "Installing MySQL client..."
    
    if [[ "$OS_TYPE" == "macos" ]]; then
        # macOS installation using Homebrew
        print_status "INFO" "Installing MySQL client via Homebrew..."
        if ! brew install mysql-client 2>/dev/null; then
            print_status "WARNING" "Failed to install mysql-client, trying mysql..."
            if ! brew install mysql; then
                print_status "ERROR" "Failed to install MySQL client via Homebrew"
                return 1
            fi
        fi
        
        # Add mysql client to PATH
        local mysql_path="/opt/homebrew/opt/mysql-client/bin"
        if [[ ! -d "$mysql_path" ]]; then
            mysql_path="/usr/local/opt/mysql-client/bin"
        fi
        
        if [[ -d "$mysql_path" ]]; then
            export PATH="$mysql_path:$PATH"
            
            # Add to shell profiles
            local shell_profiles=("$HOME/.zshrc" "$HOME/.bash_profile" "$HOME/.bashrc")
            for profile in "${shell_profiles[@]}"; do
                if [[ -f "$profile" ]] && ! grep -q "mysql-client" "$profile"; then
                    echo "# Add MySQL client to PATH" >> "$profile"
                    echo "export PATH=\"$mysql_path:\$PATH\"" >> "$profile"
                fi
            done
        fi
        
    else
        # Linux installation
        # Handle different distributions and versions
        if [[ "$DISTRO" == "debian" ]]; then
            # Debian uses MariaDB packages
            local debian_packages=("mariadb-client" "default-mysql-client")
            local installed=false
            
            for package in "${debian_packages[@]}"; do
                if sudo apt-get install -y "$package" 2>/dev/null; then
                    installed=true
                    break
                fi
            done
            
            if [[ "$installed" == false ]]; then
                print_status "ERROR" "Failed to install MySQL/MariaDB client on Debian"
                return 1
            fi
            
        else
            # Ubuntu installation - try different package names
            local mysql_packages=("mysql-client" "mysql-client-core-8.0" "mysql-client-8.0" "default-mysql-client")
            local installed=false
            
            for package in "${mysql_packages[@]}"; do
                if sudo apt-get install -y "$package" 2>/dev/null; then
                    installed=true
                    break
                fi
            done
            
            if [[ "$installed" == false ]]; then
                print_status "ERROR" "Failed to install MySQL client"
                return 1
            fi
        fi
    fi
    
    # Verify installation
    if command -v mysql >/dev/null 2>&1; then
        local version=$(mysql --version 2>/dev/null || echo "unknown")
        print_status "SUCCESS" "MySQL client installed: $version"
    else
        print_status "ERROR" "MySQL client installation verification failed"
        return 1
    fi
}

# Function to install rclone
install_rclone() {
    print_status "INFO" "Checking rclone..."
    
    if command -v rclone >/dev/null 2>&1; then
        local version=$(rclone version 2>/dev/null | head -n1 || echo "unknown")
        print_status "SUCCESS" "rclone already installed: $version"
        return 0
    fi
    
    print_status "INFO" "Installing rclone..."
    
    if [[ "$OS_TYPE" == "macos" ]]; then
        # macOS installation using Homebrew
        print_status "INFO" "Installing rclone via Homebrew..."
        if ! brew install rclone 2>/dev/null; then
            print_status "ERROR" "Failed to install rclone via Homebrew"
            return 1
        fi
        
    else
        # Linux installation - use official rclone installer
        if ! curl -fsSL https://rclone.org/install.sh | sudo bash; then
            print_status "ERROR" "Failed to install rclone"
            return 1
        fi
    fi
    
    # Verify installation
    if command -v rclone >/dev/null 2>&1; then
        local version=$(rclone version 2>/dev/null | head -n1 || echo "unknown")
        print_status "SUCCESS" "rclone installed: $version"
    else
        print_status "ERROR" "rclone installation verification failed"
        return 1
    fi
}

# Function to install Go (optional, for building from source)
install_go() {
    print_status "INFO" "Checking Go..."
    
    if command -v go >/dev/null 2>&1; then
        local version=$(go version 2>/dev/null || echo "unknown")
        print_status "SUCCESS" "Go already installed: $version"
        return 0
    fi
    
    print_status "INFO" "Installing Go..."
    
    if [[ "$OS_TYPE" == "macos" ]]; then
        # macOS installation using Homebrew
        print_status "INFO" "Installing Go via Homebrew..."
        if ! brew install go 2>/dev/null; then
            print_status "ERROR" "Failed to install Go via Homebrew"
            return 1
        fi
        
    elif [[ "$OS_TYPE" == "ubuntu" ]]; then
        # Ubuntu-specific installation
        if [[ "$OS_VERSION" == "18.04" ]]; then
            # For Ubuntu 18.04, Go 1.10 is too old, need 1.23+
            print_status "WARNING" "Ubuntu 18.04 has Go 1.10 which is too old for TenangDB (requires Go 1.23+)"
            print_status "INFO" "Removing old Go version and installing newer version..."
            
            # Remove old Go version
            sudo apt-get remove -y golang-go || true
            
            # Install newer Go version
            if command -v snap >/dev/null 2>&1; then
                print_status "INFO" "Installing Go via snap..."
                sudo snap install go --classic
            else
                print_status "INFO" "Installing Go manually..."
                install_go_manually
            fi
        else
            # For newer Ubuntu versions, try package manager first
            if ! sudo apt-get install -y golang-go; then
                print_status "WARNING" "Go not available in default repos, trying snap..."
                if command -v snap >/dev/null 2>&1; then
                    sudo snap install go --classic
                else
                    install_go_manually
                fi
            fi
        fi
        
    elif [[ "$OS_TYPE" == "debian" ]]; then
        # Debian-specific installation
        if [[ "$OS_VERSION" == "10" ]]; then
            # Debian 10 (Buster) - older Go version in repos
            print_status "INFO" "Installing Go for Debian 10..."
            
            if ! sudo apt-get install -y golang-go; then
                print_status "WARNING" "Go not available in default repos, installing manually..."
                install_go_manually
            else
                # Check if version is too old using the same logic as above
                local go_version=$(go version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1)
                if [[ "$go_version" ]]; then
                    local major=$(echo "$go_version" | cut -d. -f1)
                    local minor=$(echo "$go_version" | cut -d. -f2)
                    local patch=$(echo "$go_version" | cut -d. -f3)
                    
                    if [[ -z "$patch" ]]; then
                        patch=0
                    fi
                    
                    local version_num=$(printf "%d%02d%02d" "$major" "$minor" "$patch")
                    local required_version=12300  # 1.23.0
                    
                    if [[ "$version_num" -lt "$required_version" ]]; then
                        print_status "WARNING" "Go version $go_version is too old for TenangDB (requires 1.23+)"
                        print_status "INFO" "Installing newer Go version manually..."
                        install_go_manually
                    fi
                fi
            fi
            
        elif [[ "$OS_VERSION" == "11" ]]; then
            # Debian 11 (Bullseye)
            print_status "INFO" "Installing Go for Debian 11..."
            
            if ! sudo apt-get install -y golang-go; then
                print_status "WARNING" "Go not available in default repos, trying backports..."
                echo "deb http://deb.debian.org/debian bullseye-backports main" | sudo tee -a /etc/apt/sources.list
                sudo apt-get update -qq
                sudo apt-get install -y -t bullseye-backports golang-go || install_go_manually
            fi
            
        elif [[ "$OS_VERSION" == "12" ]]; then
            # Debian 12 (Bookworm)
            print_status "INFO" "Installing Go for Debian 12..."
            
            if ! sudo apt-get install -y golang-go; then
                print_status "WARNING" "Go not available in default repos, installing manually..."
                install_go_manually
            fi
        fi
    fi
    
    # Verify installation
    if command -v go >/dev/null 2>&1; then
        local version=$(go version 2>/dev/null || echo "unknown")
        print_status "SUCCESS" "Go installed: $version"
        
        # Check if Go version is compatible with TenangDB (requires 1.23+)
        local go_version=$(go version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1)
        if [[ "$go_version" ]]; then
            # Convert version to comparable format (e.g., 1.23.1 -> 12301, 1.10.4 -> 11004)
            local major=$(echo "$go_version" | cut -d. -f1)
            local minor=$(echo "$go_version" | cut -d. -f2)
            local patch=$(echo "$go_version" | cut -d. -f3)
            
            # Default patch to 0 if not present
            if [[ -z "$patch" ]]; then
                patch=0
            fi
            
            # Create comparable version number (MMNNPP format)
            local version_num=$(printf "%d%02d%02d" "$major" "$minor" "$patch")
            local required_version=12300  # 1.23.0
            
            if [[ "$version_num" -lt "$required_version" ]]; then
                print_status "WARNING" "Go version $go_version is too old for TenangDB (requires 1.23+)"
                print_status "INFO" "Current version: $go_version (numeric: $version_num)"
                print_status "INFO" "Required version: 1.23+ (numeric: $required_version)"
                print_status "INFO" "Installing newer Go version..."
                
                # Force upgrade Go
                if [[ "$OS_TYPE" == "ubuntu" && "$OS_VERSION" == "18.04" ]]; then
                    # For Ubuntu 18.04, use the upgrade process
                    upgrade_go_ubuntu_18_04
                else
                    install_go_manually
                fi
            else
                print_status "SUCCESS" "Go version $go_version is compatible with TenangDB"
            fi
        fi
    else
        print_status "WARNING" "Go installation verification failed"
        print_status "INFO" "You may need to restart your shell or run: source /etc/profile"
        return 1
    fi
}

# Function to install Go manually
install_go_manually() {
    print_status "INFO" "Installing Go manually..."
    
    GO_VERSION="1.23.1"
    GO_URL="https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz"
    
    print_status "INFO" "Downloading Go $GO_VERSION..."
    if curl -fsSL "$GO_URL" -o /tmp/go.tar.gz; then
        sudo rm -rf /usr/local/go
        sudo tar -C /usr/local -xzf /tmp/go.tar.gz
        rm -f /tmp/go.tar.gz
        
        # Add to PATH
        echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee -a /etc/profile
        export PATH=$PATH:/usr/local/go/bin
    else
        print_status "ERROR" "Failed to download Go"
        return 1
    fi
}

# Function to upgrade Go on Ubuntu 18.04
upgrade_go_ubuntu_18_04() {
    print_status "INFO" "Upgrading Go for Ubuntu 18.04..."
    
    # Remove old Go version
    print_status "INFO" "Removing old Go version..."
    sudo apt-get remove -y golang-go golang-1.10 golang-1.10-go || true
    
    # Remove any existing Go installation
    sudo rm -rf /usr/local/go
    
    # Clean up PATH entries
    sudo sed -i '/\/usr\/local\/go\/bin/d' /etc/profile
    
    # Install new Go version
    if command -v snap >/dev/null 2>&1; then
        print_status "INFO" "Installing Go 1.23 via snap..."
        sudo snap install go --classic
        
        # Update PATH for snap Go
        export PATH=/snap/bin:$PATH
        
        # Add to user's bashrc
        if ! grep -q '/snap/bin' ~/.bashrc; then
            echo 'export PATH=/snap/bin:$PATH' >> ~/.bashrc
        fi
    else
        print_status "INFO" "Installing Go 1.23 manually..."
        install_go_manually
        
        # Update PATH for manual Go
        export PATH=$PATH:/usr/local/go/bin
        
        # Add to user's bashrc
        if ! grep -q '/usr/local/go/bin' ~/.bashrc; then
            echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
        fi
    fi
    
    print_status "SUCCESS" "Go upgrade completed"
}

# Function to check and upgrade Go version if needed
check_and_upgrade_go() {
    print_status "INFO" "Checking Go version compatibility..."
    
    if ! command -v go >/dev/null 2>&1; then
        print_status "WARNING" "Go is not installed"
        return 1
    fi
    
    local version=$(go version 2>/dev/null || echo "unknown")
    print_status "INFO" "Current Go version: $version"
    
    # Check if Go version is compatible with TenangDB (requires 1.23+)
    local go_version=$(go version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1)
    if [[ "$go_version" ]]; then
        # Convert version to comparable format
        local major=$(echo "$go_version" | cut -d. -f1)
        local minor=$(echo "$go_version" | cut -d. -f2)
        local patch=$(echo "$go_version" | cut -d. -f3)
        
        # Default patch to 0 if not present
        if [[ -z "$patch" ]]; then
            patch=0
        fi
        
        # Create comparable version number (MMNNPP format)
        local version_num=$(printf "%d%02d%02d" "$major" "$minor" "$patch")
        local required_version=12300  # 1.23.0
        
        if [[ "$version_num" -lt "$required_version" ]]; then
            print_status "WARNING" "Go version $go_version is too old for TenangDB (requires 1.23+)"
            print_status "INFO" "Current version: $go_version (numeric: $version_num)"
            print_status "INFO" "Required version: 1.23+ (numeric: $required_version)"
            print_status "INFO" "Upgrading Go version..."
            
            # Force upgrade Go
            if [[ "$OS_TYPE" == "ubuntu" && "$OS_VERSION" == "18.04" ]]; then
                upgrade_go_ubuntu_18_04
            else
                install_go_manually
            fi
            
            # Verify upgrade
            if command -v go >/dev/null 2>&1; then
                local new_version=$(go version 2>/dev/null || echo "unknown")
                print_status "SUCCESS" "Go upgraded to: $new_version"
            else
                print_status "ERROR" "Go upgrade failed"
                return 1
            fi
        else
            print_status "SUCCESS" "Go version $go_version is compatible with TenangDB"
        fi
    else
        print_status "ERROR" "Cannot determine Go version"
        return 1
    fi
    
    return 0
}

# Function to create required directories
create_directories() {
    print_status "INFO" "Creating required directories..."
    
    if [[ "$OS_TYPE" == "macos" ]]; then
        # macOS directories
        local directories=(
            "$HOME/Library/Application Support/TenangDB"
            "$HOME/Library/Logs/TenangDB"
            "/usr/local/etc/tenangdb"
        )
        
        for dir in "${directories[@]}"; do
            if [[ ! -d "$dir" ]]; then
                print_status "INFO" "Creating directory: $dir"
                if [[ "$dir" == "/usr/local/etc/tenangdb" ]]; then
                    sudo mkdir -p "$dir"
                    sudo chown $USER:staff "$dir" 2>/dev/null || true
                else
                    mkdir -p "$dir"
                fi
            else
                print_status "INFO" "Directory already exists: $dir"
            fi
        done
        
    else
        # Linux directories
        local directories=(
            "/var/backups"
            "/var/log/tenangdb"
            "/etc/tenangdb"
        )
        
        for dir in "${directories[@]}"; do
            if [[ ! -d "$dir" ]]; then
                print_status "INFO" "Creating directory: $dir"
                sudo mkdir -p "$dir"
                sudo chown $USER:$USER "$dir" 2>/dev/null || true
            else
                print_status "INFO" "Directory already exists: $dir"
            fi
        done
    fi
    
    print_status "SUCCESS" "Required directories created"
}

# Function to run final verification
final_verification() {
    print_status "HEADER" "Final Verification"
    
    local all_good=true
    
    # Check mydumper
    if command -v mydumper >/dev/null 2>&1; then
        print_status "SUCCESS" "mydumper: $(mydumper --version 2>/dev/null | head -n1)"
    else
        print_status "ERROR" "mydumper: NOT FOUND"
        all_good=false
    fi
    
    # Check myloader
    if command -v myloader >/dev/null 2>&1; then
        print_status "SUCCESS" "myloader: $(myloader --version 2>/dev/null | head -n1)"
    else
        print_status "ERROR" "myloader: NOT FOUND"
        all_good=false
    fi
    
    # Check mysql
    if command -v mysql >/dev/null 2>&1; then
        print_status "SUCCESS" "mysql: $(mysql --version 2>/dev/null)"
    else
        print_status "ERROR" "mysql: NOT FOUND"
        all_good=false
    fi
    
    # Check rclone
    if command -v rclone >/dev/null 2>&1; then
        print_status "SUCCESS" "rclone: $(rclone version 2>/dev/null | head -n1)"
    else
        print_status "WARNING" "rclone: NOT FOUND (optional for cloud upload)"
    fi
    
    # Check Go
    if command -v go >/dev/null 2>&1; then
        print_status "SUCCESS" "go: $(go version 2>/dev/null)"
    else
        print_status "WARNING" "go: NOT FOUND (optional, only needed for building)"
    fi
    
    if [[ "$all_good" == true ]]; then
        print_status "SUCCESS" "All required dependencies are installed!"
        print_status "INFO" "You can now build and run TenangDB"
        return 0
    else
        print_status "ERROR" "Some required dependencies are missing"
        return 1
    fi
}

# Function to show usage
show_usage() {
    echo "TenangDB Dependency Installer v$SCRIPT_VERSION"
    echo "Automatically installs dependencies for Ubuntu, Debian, and macOS systems"
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  -y, --yes      Automatic yes to prompts"
    echo "  --no-go        Skip Go installation"
    echo "  --no-rclone    Skip rclone installation"
    echo "  --check-only   Only check dependencies, don't install"
    echo ""
    echo "Supported Ubuntu versions: ${SUPPORTED_UBUNTU_VERSIONS[*]}"
    echo "Supported Debian versions: ${SUPPORTED_DEBIAN_VERSIONS[*]}"
    echo "Supported macOS versions: ${SUPPORTED_MACOS_VERSIONS[*]}"
}

# Parse command line arguments
ASSUME_YES=false
SKIP_GO=false
SKIP_RCLONE=false
CHECK_ONLY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_usage
            exit 0
            ;;
        -y|--yes)
            ASSUME_YES=true
            shift
            ;;
        --no-go)
            SKIP_GO=true
            shift
            ;;
        --no-rclone)
            SKIP_RCLONE=true
            shift
            ;;
        --check-only)
            CHECK_ONLY=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Main execution
main() {
    print_status "HEADER" "TenangDB Dependency Installer v$SCRIPT_VERSION"
    
    # Pre-flight checks
    check_root
    check_os_version
    
    if [[ "$CHECK_ONLY" == true ]]; then
        # In check-only mode, also check Go version compatibility
        if command -v go >/dev/null 2>&1; then
            print_status "INFO" "Checking Go version compatibility..."
            local go_version=$(go version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1)
            if [[ "$go_version" ]]; then
                local major=$(echo "$go_version" | cut -d. -f1)
                local minor=$(echo "$go_version" | cut -d. -f2)
                local patch=$(echo "$go_version" | cut -d. -f3)
                
                if [[ -z "$patch" ]]; then
                    patch=0
                fi
                
                local version_num=$(printf "%d%02d%02d" "$major" "$minor" "$patch")
                local required_version=12300  # 1.23.0
                
                if [[ "$version_num" -lt "$required_version" ]]; then
                    print_status "WARNING" "Go version $go_version is too old for TenangDB (requires 1.23+)"
                    print_status "INFO" "Run 'make install-deps' to upgrade Go version"
                fi
            fi
        fi
        
        final_verification
        exit $?
    fi
    
    # Ask for confirmation
    if [[ "$ASSUME_YES" == false ]]; then
        echo ""
        read -p "Do you want to install TenangDB dependencies? (y/N): " -n 1 -r
        echo ""
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_status "INFO" "Installation cancelled by user"
            exit 0
        fi
    fi
    
    # Install dependencies
    print_status "INFO" "Starting dependency installation..."
    
    update_package_lists
    install_basic_deps
    install_mydumper
    install_mysql_client
    
    if [[ "$SKIP_RCLONE" == false ]]; then
        install_rclone
    fi
    
    if [[ "$SKIP_GO" == false ]]; then
        # First try to install Go if not present
        if ! command -v go >/dev/null 2>&1; then
            install_go
        fi
        
        # Then check and upgrade Go version if needed
        check_and_upgrade_go
    fi
    
    create_directories
    
    # Final verification
    print_status "HEADER" "Installation Complete"
    final_verification
    
    if [[ $? -eq 0 ]]; then
        print_status "SUCCESS" "TenangDB dependencies successfully installed!"
        print_status "INFO" "Next steps:"
        print_status "INFO" "1. Build TenangDB: make build"
        print_status "INFO" "2. Install TenangDB: sudo make install"
        print_status "INFO" "3. Configure: /etc/tenangdb/config.yaml"
    else
        print_status "ERROR" "Some dependencies failed to install"
        exit 1
    fi
}

# Run main function
main "$@"