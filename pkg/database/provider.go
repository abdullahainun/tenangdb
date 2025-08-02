package database

import (
	"context"
	"fmt"
	"time"
)

// DatabaseType represents the type of database
type DatabaseType string

const (
	MySQL      DatabaseType = "mysql"
	PostgreSQL DatabaseType = "postgresql"
)

// BackupFormat represents the format of backup files
type BackupFormat string

const (
	SQL    BackupFormat = "sql"    // Plain SQL format
	Custom BackupFormat = "custom" // Database-specific custom format
	Binary BackupFormat = "binary" // Binary/physical backup
)

// BackupOptions contains options for backup operations
type BackupOptions struct {
	Databases     []string
	Directory     string
	Timestamp     string
	UseParallel   bool      // Use parallel tools (mydumper/pg_dump parallel)
	Format        BackupFormat
	Compression   bool
	IncludeData   bool
	IncludeSchema bool
	Timeout       time.Duration
	ExtraArgs     []string // Database-specific arguments
}

// RestoreOptions contains options for restore operations  
type RestoreOptions struct {
	BackupPath    string
	TargetDB      string
	DropIfExists  bool
	Timeout       time.Duration
	ExtraArgs     []string
}

// DatabaseInfo contains metadata about a database
type DatabaseInfo struct {
	Name        string
	Size        int64  // Size in bytes
	TableCount  int
	IsSystem    bool   // Whether it's a system database
	Charset     string // Character set/encoding
}

// BackupResult contains the result of a backup operation
type BackupResult struct {
	Database     string
	BackupPath   string
	Size         int64
	Duration     time.Duration
	Success      bool
	Error        error
	Format       BackupFormat
	Compression  bool
}

// Provider defines the interface that all database providers must implement
type Provider interface {
	// Connection management
	TestConnection(ctx context.Context) error
	Close() error

	// Database operations
	ListDatabases(ctx context.Context) ([]*DatabaseInfo, error)
	DatabaseExists(ctx context.Context, dbName string) (bool, error)

	// Backup operations
	CreateBackup(ctx context.Context, opts *BackupOptions) ([]*BackupResult, error)
	
	// Restore operations
	RestoreBackup(ctx context.Context, opts *RestoreOptions) error
	
	// Tool availability
	GetAvailableTools() []string
	ValidateTools() error
	
	// Provider metadata
	GetProviderType() DatabaseType
	GetDefaultPort() int
	GetSystemDatabases() []string
}

// ProviderConfig contains common configuration for all providers
type ProviderConfig struct {
	Type            DatabaseType `yaml:"type"`
	Host            string       `yaml:"host"`
	Port            int          `yaml:"port"`
	Username        string       `yaml:"username"`
	Password        string       `yaml:"password"`
	SSLMode         string       `yaml:"ssl_mode,omitempty"`
	Timeout         string       `yaml:"timeout,omitempty"`
	
	// Tool paths (auto-discovered if empty)
	DumpToolPath    string `yaml:"dump_tool_path,omitempty"`
	ClientToolPath  string `yaml:"client_tool_path,omitempty"`
	ParallelToolPath string `yaml:"parallel_tool_path,omitempty"`
	
	// Provider-specific options
	MySQL      *MySQLConfig      `yaml:"mysql,omitempty"`
	PostgreSQL *PostgreSQLConfig `yaml:"postgresql,omitempty"`
}

// MySQLConfig contains MySQL-specific configuration
type MySQLConfig struct {
	// Existing MySQL configuration will be moved here
	UseMyDumper      bool   `yaml:"use_mydumper"`
	MyDumperPath     string `yaml:"mydumper_path,omitempty"`
	MyLoaderPath     string `yaml:"myloader_path,omitempty"`
	SingleTransaction bool  `yaml:"single_transaction"`
	LockTables       bool   `yaml:"lock_tables"`
	RoutinesAndEvents bool  `yaml:"routines_and_events"`
}

// PostgreSQLConfig contains PostgreSQL-specific configuration  
type PostgreSQLConfig struct {
	// Will be implemented in v1.2.0
	UsePgDumpParallel bool   `yaml:"use_pg_dump_parallel"`
	PgDumpPath        string `yaml:"pg_dump_path,omitempty"`
	PsqlPath          string `yaml:"psql_path,omitempty"`
	Format            string `yaml:"format"` // plain, custom, directory, tar
}

// ProviderFactory creates database providers based on configuration
type ProviderFactory interface {
	CreateProvider(config *ProviderConfig) (Provider, error)
	GetSupportedTypes() []DatabaseType
}

// LegacyDatabaseConfig represents the old MySQL-only configuration for backward compatibility
type LegacyDatabaseConfig struct {
	Host          string `yaml:"host"`
	Port          int    `yaml:"port"`
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
	Timeout       int    `yaml:"timeout"`
	MysqldumpPath string `yaml:"mysqldump_path"`
	MysqlPath     string `yaml:"mysql_path"`
}

// MigrateFromLegacyConfig converts legacy config to new provider config
func MigrateFromLegacyConfig(legacy *LegacyDatabaseConfig) *ProviderConfig {
	config := &ProviderConfig{
		Type:           MySQL, // Legacy configs are always MySQL
		Host:           legacy.Host,
		Port:           legacy.Port,
		Username:       legacy.Username,
		Password:       legacy.Password,
		DumpToolPath:   legacy.MysqldumpPath,
		ClientToolPath: legacy.MysqlPath,
	}
	
	if legacy.Timeout > 0 {
		config.Timeout = fmt.Sprintf("%ds", legacy.Timeout)
	}
	
	// Set default MySQL configuration
	config.MySQL = &MySQLConfig{
		UseMyDumper:       true,
		SingleTransaction: true,
		LockTables:        true,
		RoutinesAndEvents: true,
	}
	
	// Set default port if not specified
	if config.Port == 0 {
		config.Port = 3306
	}
	
	return config
}