package database

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// SetupWizard provides an interactive setup for database configuration
type SetupWizard struct {
	scanner *bufio.Scanner
}

// NewSetupWizard creates a new setup wizard
func NewSetupWizard() *SetupWizard {
	return &SetupWizard{
		scanner: bufio.NewScanner(os.Stdin),
	}
}

// SetupDatabaseConfig interactively configures database settings with multi-database support
func (w *SetupWizard) SetupDatabaseConfig() (*ProviderConfig, error) {
	fmt.Printf("💾 Database Configuration\n")
	fmt.Printf("=========================\n\n")

	// Step 1: Database Type Selection
	dbType := w.selectDatabaseType()
	
	config := &ProviderConfig{
		Type: dbType,
	}

	// Step 2: Connection Settings
	w.setupConnectionSettings(config)

	// Step 3: Database-specific Configuration
	switch dbType {
	case MySQL:
		w.setupMySQLConfig(config)
	case PostgreSQL:
		w.setupPostgreSQLConfig(config)
	}

	return config, nil
}

// selectDatabaseType allows user to choose database type
func (w *SetupWizard) selectDatabaseType() DatabaseType {
	fmt.Printf("🗄️  Select Database Type:\n")
	fmt.Printf("  1. MySQL (default)\n")
	fmt.Printf("  2. PostgreSQL (coming in v1.2.0)\n")
	fmt.Printf("  3. Auto-detect from port\n")
	fmt.Printf("\n")

	for {
		fmt.Print("Choose database type [1]: ")
		if w.scanner.Scan() {
			input := strings.TrimSpace(w.scanner.Text())
			
			switch input {
			case "", "1":
				fmt.Printf("✅ Selected: MySQL\n\n")
				return MySQL
			case "2":
				fmt.Printf("⚠️  PostgreSQL support is coming in v1.2.0. Using MySQL for now.\n\n")
				return MySQL
			case "3":
				fmt.Printf("✅ Will auto-detect based on port (3306=MySQL, 5432=PostgreSQL)\n\n")
				return "" // Will be set later based on port
			default:
				fmt.Printf("❌ Invalid choice. Please select 1, 2, or 3.\n")
				continue
			}
		}
	}
}

// setupConnectionSettings configures basic connection parameters
func (w *SetupWizard) setupConnectionSettings(config *ProviderConfig) {
	// Database host
	fmt.Print("Database host [localhost]: ")
	host := "localhost"
	if w.scanner.Scan() {
		if input := strings.TrimSpace(w.scanner.Text()); input != "" {
			host = input
		}
	}
	config.Host = host

	// Database port with auto-detection
	fmt.Print("Database port [auto-detect]: ")
	port := 0
	if w.scanner.Scan() {
		if input := strings.TrimSpace(w.scanner.Text()); input != "" {
			if p, err := strconv.Atoi(input); err == nil {
				port = p
			} else {
				fmt.Printf("Invalid port, will use default\n")
			}
		}
	}

	// Auto-detect database type from port if not already set
	if config.Type == "" {
		switch port {
		case 3306:
			config.Type = MySQL
			fmt.Printf("🔍 Auto-detected: MySQL (port 3306)\n")
		case 5432:
			config.Type = PostgreSQL
			fmt.Printf("🔍 Auto-detected: PostgreSQL (port 5432)\n")
			fmt.Printf("⚠️  PostgreSQL support coming in v1.2.0. Using MySQL for now.\n")
			config.Type = MySQL
			port = 3306
		default:
			// Default to MySQL
			config.Type = MySQL
			port = 3306
			fmt.Printf("🔍 No port specified, defaulting to MySQL (port 3306)\n")
		}
	} else {
		// Set default port based on database type
		if port == 0 {
			switch config.Type {
			case MySQL:
				port = 3306
			case PostgreSQL:
				port = 5432
			}
		}
	}
	config.Port = port

	// Database username
	fmt.Print("Database username: ")
	var username string
	if w.scanner.Scan() {
		username = strings.TrimSpace(w.scanner.Text())
	}
	for username == "" {
		fmt.Print("Username is required. Database username: ")
		if w.scanner.Scan() {
			username = strings.TrimSpace(w.scanner.Text())
		} else {
			// Handle EOF or input error - exit gracefully
			fmt.Printf("\nError: Unable to read input. Setup cancelled.\n")
			return
		}
	}
	config.Username = username

	// Database password
	fmt.Print("Database password: ")
	var password string
	if w.scanner.Scan() {
		password = w.scanner.Text() // Don't trim password, preserve spaces
	}
	config.Password = password

	// Connection timeout
	config.Timeout = "30s"
}

// setupMySQLConfig configures MySQL-specific settings
func (w *SetupWizard) setupMySQLConfig(config *ProviderConfig) {
	fmt.Printf("\n🔧 MySQL Configuration\n")
	fmt.Printf("=====================\n")

	config.MySQL = &MySQLConfig{
		UseMyDumper:       true,
		SingleTransaction: true,
		LockTables:        true,
		RoutinesAndEvents: true,
	}

	// Ask about parallel backup preference
	fmt.Print("Use mydumper for faster parallel backups? [Y/n]: ")
	if w.scanner.Scan() {
		input := strings.ToLower(strings.TrimSpace(w.scanner.Text()))
		if input == "n" || input == "no" {
			config.MySQL.UseMyDumper = false
			fmt.Printf("✅ Will use mysqldump for backups\n")
		} else {
			fmt.Printf("✅ Will use mydumper for faster parallel backups\n")
		}
	}
}

// setupPostgreSQLConfig configures PostgreSQL-specific settings (placeholder for v1.2.0)
func (w *SetupWizard) setupPostgreSQLConfig(config *ProviderConfig) {
	fmt.Printf("\n🔧 PostgreSQL Configuration\n")
	fmt.Printf("==========================\n")
	fmt.Printf("⚠️  PostgreSQL support will be fully implemented in v1.2.0\n")
	fmt.Printf("🔄 Converting to MySQL configuration for now...\n")
	
	// Convert to MySQL for now
	config.Type = MySQL
	config.Port = 3306
	w.setupMySQLConfig(config)
}

// GetSupportedTypesDisplay returns a display string of supported database types
func GetSupportedTypesDisplay() string {
	return "MySQL (PostgreSQL coming in v1.2.0)"
}

// ValidateAndSetDefaults validates the configuration and sets appropriate defaults
func ValidateAndSetDefaults(config *ProviderConfig) error {
	if err := ValidateConfig(config); err != nil {
		return err
	}

	// Set database-specific defaults
	switch config.Type {
	case MySQL:
		if config.MySQL == nil {
			config.MySQL = &MySQLConfig{
				UseMyDumper:       true,
				SingleTransaction: true,
				LockTables:        true,
				RoutinesAndEvents: true,
			}
		}
	case PostgreSQL:
		// Will be implemented in v1.2.0
		return fmt.Errorf("PostgreSQL support coming in v1.2.0")
	}

	return nil
}