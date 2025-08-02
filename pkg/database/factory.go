package database

import (
	"fmt"
)

// DefaultProviderFactory implements ProviderFactory
type DefaultProviderFactory struct{}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() ProviderFactory {
	return &DefaultProviderFactory{}
}

// CreateProvider creates a database provider based on the configuration
func (f *DefaultProviderFactory) CreateProvider(config *ProviderConfig) (Provider, error) {
	if config == nil {
		return nil, fmt.Errorf("provider config cannot be nil")
	}

	// Auto-detect database type if not specified (backward compatibility)
	if config.Type == "" {
		config.Type = f.detectDatabaseType(config)
	}

	switch config.Type {
	case MySQL:
		return NewMySQLProvider(config)
	case PostgreSQL:
		// Will be implemented in v1.2.0
		return nil, fmt.Errorf("PostgreSQL support coming in v1.2.0")
	default:
		return nil, fmt.Errorf("unsupported database type: %s", config.Type)
	}
}

// GetSupportedTypes returns the list of supported database types
func (f *DefaultProviderFactory) GetSupportedTypes() []DatabaseType {
	return []DatabaseType{
		MySQL,
		// PostgreSQL will be added in v1.2.0
	}
}

// detectDatabaseType attempts to auto-detect the database type from configuration
// This provides backward compatibility with existing MySQL-only configs
func (f *DefaultProviderFactory) detectDatabaseType(config *ProviderConfig) DatabaseType {
	// Check for explicit MySQL configuration
	if config.MySQL != nil {
		return MySQL
	}
	
	// Check for explicit PostgreSQL configuration (future)
	if config.PostgreSQL != nil {
		return PostgreSQL
	}
	
	// Check default ports
	switch config.Port {
	case 3306:
		return MySQL
	case 5432:
		return PostgreSQL
	}
	
	// Default to MySQL for backward compatibility
	return MySQL
}

// ValidateConfig validates the provider configuration
func ValidateConfig(config *ProviderConfig) error {
	if config.Host == "" {
		return fmt.Errorf("database host is required")
	}
	
	if config.Username == "" {
		return fmt.Errorf("database username is required")
	}
	
	if config.Port <= 0 {
		// Set default ports based on database type
		switch config.Type {
		case MySQL:
			config.Port = 3306
		case PostgreSQL:
			config.Port = 5432
		default:
			return fmt.Errorf("invalid port and unknown database type")
		}
	}
	
	return nil
}