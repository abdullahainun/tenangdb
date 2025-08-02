package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/abdullahainun/tenangdb/internal/logger"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLProvider implements the Provider interface for MySQL databases
type MySQLProvider struct {
	config *ProviderConfig
	db     *sql.DB
	logger *logger.Logger
}

// NewMySQLProvider creates a new MySQL provider
func NewMySQLProvider(config *ProviderConfig) (Provider, error) {
	if err := ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid MySQL config: %w", err)
	}

	// Set default MySQL configuration if not provided
	if config.MySQL == nil {
		config.MySQL = &MySQLConfig{
			UseMyDumper:       true,
			SingleTransaction: true,
			LockTables:        true,
			RoutinesAndEvents: true,
		}
	}

	provider := &MySQLProvider{
		config: config,
		logger: logger.NewLogger("mysql-provider"),
	}

	// Establish database connection
	if err := provider.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	return provider, nil
}

// connect establishes a connection to the MySQL database
func (p *MySQLProvider) connect() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/",
		p.config.Username,
		p.config.Password,
		p.config.Host,
		p.config.Port,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Set connection timeouts
	timeout := 30 * time.Second
	if p.config.Timeout != "" {
		if parsedTimeout, err := time.ParseDuration(p.config.Timeout); err == nil {
			timeout = parsedTimeout
		}
	}

	db.SetConnMaxLifetime(timeout)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	p.db = db
	return nil
}

// TestConnection tests the database connection
func (p *MySQLProvider) TestConnection(ctx context.Context) error {
	if p.db == nil {
		return fmt.Errorf("database connection not established")
	}

	return p.db.PingContext(ctx)
}

// Close closes the database connection
func (p *MySQLProvider) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// ListDatabases returns a list of all databases
func (p *MySQLProvider) ListDatabases(ctx context.Context) ([]*DatabaseInfo, error) {
	rows, err := p.db.QueryContext(ctx, "SHOW DATABASES")
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}
	defer rows.Close()

	var databases []*DatabaseInfo
	systemDBs := p.GetSystemDatabases()

	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			continue
		}

		isSystem := false
		for _, sysDB := range systemDBs {
			if dbName == sysDB {
				isSystem = true
				break
			}
		}

		dbInfo := &DatabaseInfo{
			Name:     dbName,
			IsSystem: isSystem,
		}

		// Get additional database info
		if size, err := p.getDatabaseSize(ctx, dbName); err == nil {
			dbInfo.Size = size
		}

		databases = append(databases, dbInfo)
	}

	return databases, rows.Err()
}

// DatabaseExists checks if a database exists
func (p *MySQLProvider) DatabaseExists(ctx context.Context, dbName string) (bool, error) {
	var exists bool
	query := "SELECT 1 FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?"
	err := p.db.QueryRowContext(ctx, query, dbName).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return exists, err
}

// CreateBackup creates backups for the specified databases
func (p *MySQLProvider) CreateBackup(ctx context.Context, opts *BackupOptions) ([]*BackupResult, error) {
	var results []*BackupResult

	for _, dbName := range opts.Databases {
		result := &BackupResult{
			Database: dbName,
			Format:   opts.Format,
		}

		start := time.Now()

		// Choose backup method
		var backupPath string
		var err error

		if p.config.MySQL.UseMyDumper && p.hasMyDumper() {
			backupPath, err = p.createMyDumperBackup(ctx, dbName, opts)
		} else {
			backupPath, err = p.createMySQLDumpBackup(ctx, dbName, opts)
		}

		result.Duration = time.Since(start)
		result.BackupPath = backupPath

		if err != nil {
			result.Error = err
			result.Success = false
		} else {
			result.Success = true
			if stat, err := os.Stat(backupPath); err == nil {
				result.Size = stat.Size()
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// RestoreBackup restores a database from backup
func (p *MySQLProvider) RestoreBackup(ctx context.Context, opts *RestoreOptions) error {
	// Implementation will use mysql client or myloader based on backup format
	// This is a placeholder for the full implementation
	return fmt.Errorf("restore functionality will be implemented in the refactoring")
}

// GetAvailableTools returns available MySQL tools
func (p *MySQLProvider) GetAvailableTools() []string {
	var tools []string

	if p.hasMySQLDump() {
		tools = append(tools, "mysqldump")
	}
	if p.hasMySQL() {
		tools = append(tools, "mysql")
	}
	if p.hasMyDumper() {
		tools = append(tools, "mydumper")
	}
	if p.hasMyLoader() {
		tools = append(tools, "myloader")
	}

	return tools
}

// ValidateTools validates that required tools are available
func (p *MySQLProvider) ValidateTools() error {
	if !p.hasMySQLDump() && !p.hasMyDumper() {
		return fmt.Errorf("neither mysqldump nor mydumper found")
	}
	return nil
}

// GetProviderType returns the provider type
func (p *MySQLProvider) GetProviderType() DatabaseType {
	return MySQL
}

// GetDefaultPort returns the default MySQL port
func (p *MySQLProvider) GetDefaultPort() int {
	return 3306
}

// GetSystemDatabases returns MySQL system databases
func (p *MySQLProvider) GetSystemDatabases() []string {
	return []string{"information_schema", "performance_schema", "mysql", "sys"}
}

// Helper methods

func (p *MySQLProvider) getDatabaseSize(ctx context.Context, dbName string) (int64, error) {
	query := `
		SELECT ROUND(SUM(data_length + index_length), 0) as size_bytes
		FROM information_schema.tables 
		WHERE table_schema = ?
	`
	var size sql.NullInt64
	err := p.db.QueryRowContext(ctx, query, dbName).Scan(&size)
	if err != nil {
		return 0, err
	}
	if !size.Valid {
		return 0, nil
	}
	return size.Int64, nil
}

func (p *MySQLProvider) hasMySQLDump() bool {
	_, err := exec.LookPath("mysqldump")
	return err == nil
}

func (p *MySQLProvider) hasMySQL() bool {
	_, err := exec.LookPath("mysql")
	return err == nil
}

func (p *MySQLProvider) hasMyDumper() bool {
	_, err := exec.LookPath("mydumper")
	return err == nil
}

func (p *MySQLProvider) hasMyLoader() bool {
	_, err := exec.LookPath("myloader")
	return err == nil
}

func (p *MySQLProvider) createMySQLDumpBackup(ctx context.Context, dbName string, opts *BackupOptions) (string, error) {
	// This will contain the existing mysqldump logic from client.go
	// Placeholder implementation
	fileName := fmt.Sprintf("backup-%s-%s.sql", dbName, opts.Timestamp)
	backupPath := filepath.Join(opts.Directory, fileName)

	args := []string{
		"--single-transaction",
		"--routines",
		"--events",
		"--triggers",
		"--create-options",
		"--extended-insert",
		"--quick",
		"--lock-tables=false",
		"--add-drop-database",
		"--databases", dbName,
		fmt.Sprintf("--host=%s", p.config.Host),
		fmt.Sprintf("--port=%d", p.config.Port),
		fmt.Sprintf("--user=%s", p.config.Username),
		fmt.Sprintf("--password=%s", p.config.Password),
	}

	cmd := exec.CommandContext(ctx, "mysqldump", args...)
	
	output, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer output.Close()

	cmd.Stdout = output

	if err := cmd.Run(); err != nil {
		os.Remove(backupPath) // Clean up on failure
		return "", fmt.Errorf("mysqldump failed: %w", err)
	}

	return backupPath, nil
}

func (p *MySQLProvider) createMyDumperBackup(ctx context.Context, dbName string, opts *BackupOptions) (string, error) {
	// This will contain the existing mydumper logic from client.go
	// Placeholder implementation
	backupDir := filepath.Join(opts.Directory, fmt.Sprintf("%s-%s", dbName, opts.Timestamp))

	args := []string{
		fmt.Sprintf("--host=%s", p.config.Host),
		fmt.Sprintf("--port=%d", p.config.Port),
		fmt.Sprintf("--user=%s", p.config.Username),
		fmt.Sprintf("--password=%s", p.config.Password),
		"--single-transaction",
		"--routines",
		"--events",
		"--triggers",
		"--databases", dbName,
		"--outputdir", backupDir,
	}

	cmd := exec.CommandContext(ctx, "mydumper", args...)

	if err := cmd.Run(); err != nil {
		os.RemoveAll(backupDir) // Clean up on failure
		return "", fmt.Errorf("mydumper failed: %w", err)
	}

	return backupDir, nil
}