#!/bin/bash

# TenangDB Dependencies Test Script
# This script checks if all required dependencies are installed and working

set -e

echo "🔍 Testing TenangDB Dependencies..."
echo "=================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Function to print test result
print_result() {
    local test_name="$1"
    local result="$2"
    local message="$3"
    
    if [ "$result" = "PASS" ]; then
        echo -e "✅ ${GREEN}PASS${NC} - $test_name"
        ((TESTS_PASSED++))
    elif [ "$result" = "FAIL" ]; then
        echo -e "❌ ${RED}FAIL${NC} - $test_name: $message"
        ((TESTS_FAILED++))
    elif [ "$result" = "WARN" ]; then
        echo -e "⚠️  ${YELLOW}WARN${NC} - $test_name: $message"
    fi
}

# Test mydumper
echo "Testing mydumper..."
if command -v mydumper >/dev/null 2>&1; then
    VERSION=$(mydumper --version 2>/dev/null | head -n1 || echo "unknown")
    print_result "mydumper" "PASS" "Version: $VERSION"
else
    print_result "mydumper" "FAIL" "Not found. Install with: sudo apt install mydumper"
fi

# Test myloader
echo "Testing myloader..."
if command -v myloader >/dev/null 2>&1; then
    VERSION=$(myloader --version 2>/dev/null | head -n1 || echo "unknown")
    print_result "myloader" "PASS" "Version: $VERSION"
else
    print_result "myloader" "FAIL" "Not found. Install with: sudo apt install mydumper"
fi

# Test rclone
echo "Testing rclone..."
if command -v rclone >/dev/null 2>&1; then
    VERSION=$(rclone version | head -n1)
    print_result "rclone" "PASS" "Version: $VERSION"
else
    print_result "rclone" "WARN" "Not found. Install with: curl https://rclone.org/install.sh | sudo bash"
fi

# Test mysql client
echo "Testing mysql client..."
if command -v mysql >/dev/null 2>&1; then
    VERSION=$(mysql --version)
    print_result "mysql client" "PASS" "Version: $VERSION"
else
    print_result "mysql client" "FAIL" "Not found. Install with: sudo apt install mysql-client"
fi

# Test Go (for building)
echo "Testing Go..."
if command -v go >/dev/null 2>&1; then
    VERSION=$(go version)
    print_result "Go" "PASS" "Version: $VERSION"
else
    print_result "Go" "WARN" "Not found. Only needed for building from source"
fi

# Test configuration files
echo "Testing configuration files..."
if [ -f "$HOME/.my.cnf" ]; then
    print_result "~/.my.cnf" "PASS" "MySQL config file found"
else
    print_result "~/.my.cnf" "WARN" "MySQL config file not found. Create for mydumper authentication"
fi

if [ -f "$HOME/.my_restore.cnf" ]; then
    print_result "~/.my_restore.cnf" "PASS" "MySQL restore config file found"
else
    print_result "~/.my_restore.cnf" "WARN" "MySQL restore config file not found. Create for myloader authentication"
fi

# Test tenangdb binary
echo "Testing tenangdb binary..."
if [ -f "./tenangdb" ]; then
    print_result "tenangdb binary" "PASS" "Found in current directory"
elif command -v tenangdb >/dev/null 2>&1; then
    print_result "tenangdb binary" "PASS" "Found in PATH"
else
    print_result "tenangdb binary" "WARN" "Not found. Build with: make build"
fi

# Test directories
echo "Testing directories..."
BACKUP_DIR="/opt/tenangdb/backup"
LOG_DIR="/var/log/tenangdb"

if [ -d "$BACKUP_DIR" ]; then
    if [ -w "$BACKUP_DIR" ]; then
        print_result "Backup directory" "PASS" "$BACKUP_DIR (writable)"
    else
        print_result "Backup directory" "WARN" "$BACKUP_DIR (not writable)"
    fi
else
    print_result "Backup directory" "WARN" "$BACKUP_DIR not found. Create with: sudo mkdir -p $BACKUP_DIR"
fi

if [ -d "$LOG_DIR" ]; then
    if [ -w "$LOG_DIR" ]; then
        print_result "Log directory" "PASS" "$LOG_DIR (writable)"
    else
        print_result "Log directory" "WARN" "$LOG_DIR (not writable)"
    fi
else
    print_result "Log directory" "WARN" "$LOG_DIR not found. Create with: sudo mkdir -p $LOG_DIR"
fi

# Summary
echo ""
echo "=================================="
echo "📊 Test Summary"
echo "=================================="
echo -e "✅ Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "❌ Failed: ${RED}$TESTS_FAILED${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n🎉 ${GREEN}All critical dependencies are available!${NC}"
    echo "You can now run TenangDB backup operations."
    exit 0
else
    echo -e "\n⚠️  ${YELLOW}Some dependencies are missing.${NC}"
    echo "Please install the missing dependencies before running TenangDB."
    exit 1
fi