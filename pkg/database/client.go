package database

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/abdullahainun/tenangdb/internal/compression"
	"github.com/abdullahainun/tenangdb/internal/config"
	"github.com/abdullahainun/tenangdb/internal/logger"

	_ "github.com/go-sql-driver/mysql"
)

type Client struct {
	config *config.DatabaseConfig
	db     *sql.DB
}

func NewClient(config *config.DatabaseConfig) (*Client, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Set connection timeouts
	db.SetConnMaxLifetime(time.Duration(config.Timeout) * time.Second)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Timeout)*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Client{
		config: config,
		db:     db,
	}, nil
}

func (c *Client) CreateBackup(ctx context.Context, dbName, backupDir string) (string, error) {
	now := time.Now()
	timestamp := now.Format("2006-01-02_15-04-05")

	// Create organized directory structure: database-backup/dbname/YYYY-MM/
	yearMonth := now.Format("2006-01")
	organizedBackupDir := filepath.Join(backupDir, dbName, yearMonth)

	// Ensure the organized directory exists
	if err := os.MkdirAll(organizedBackupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create organized backup directory: %w", err)
	}

	// Check if mydumper is enabled in config
	if c.config.Mydumper != nil && c.config.Mydumper.Enabled {
		return c.createMydumperBackup(ctx, dbName, organizedBackupDir, timestamp)
	}

	// Fallback to mysqldump
	return c.createMysqldumpBackup(ctx, dbName, organizedBackupDir, timestamp)
}

func (c *Client) createMydumperBackup(ctx context.Context, dbName, backupDir, timestamp string) (string, error) {
	// Create database-specific directory
	dbBackupDir := filepath.Join(backupDir, fmt.Sprintf("%s-%s", dbName, timestamp))
	if err := os.MkdirAll(dbBackupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Build mydumper command with version-compatible arguments
	args := c.buildMydumperArgs(dbBackupDir, dbName)

	// Use defaults-file if specified, otherwise use individual connection parameters
	if c.config.Mydumper.DefaultsFile != "" {
		args = append(args, fmt.Sprintf("--defaults-file=%s", c.config.Mydumper.DefaultsFile))
	} else {
		args = append(args, fmt.Sprintf("--host=%s", c.config.Host))
		args = append(args, fmt.Sprintf("--port=%d", c.config.Port))
		args = append(args, fmt.Sprintf("--user=%s", c.config.Username))
		if c.config.Password != "" {
			args = append(args, fmt.Sprintf("--password=%s", c.config.Password))
		}
	}

	if c.config.Mydumper.CompressMethod != "" {
		args = append(args, "--compress")
	}

	if c.config.Mydumper.BuildEmptyFiles {
		args = append(args, "--build-empty-files")
	}

	// --use-defer is not a valid mydumper option, skip it
	// if c.config.Mydumper.UseDefer {
	//	args = append(args, "--use-defer")
	// }

	// --single-table is not a valid mydumper option, skip it
	// if c.config.Mydumper.SingleTable {
	//	args = append(args, "--single-table")
	// }

	if c.config.Mydumper.NoSchemas {
		args = append(args, "--no-schemas")
	}

	if c.config.Mydumper.NoData {
		args = append(args, "--no-data")
	}

	cmd := exec.CommandContext(ctx, c.config.Mydumper.BinaryPath, args...)

	// Capture both stdout and stderr for better error reporting
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr


	if err := cmd.Run(); err != nil {
		// Remove failed backup directory
		os.RemoveAll(dbBackupDir)
		return "", fmt.Errorf("mydumper failed: %w, stdout: %s, stderr: %s", err, stdout.String(), stderr.String())
	}

	// Verify backup directory was created and has content
	if err := c.verifyMydumperBackup(dbBackupDir); err != nil {
		os.RemoveAll(dbBackupDir)
		return "", fmt.Errorf("mydumper backup verification failed: %w", err)
	}

	return dbBackupDir, nil
}

func (c *Client) createMysqldumpBackup(ctx context.Context, dbName, backupDir, timestamp string) (string, error) {
	fileName := fmt.Sprintf("%s-%s.sql", dbName, timestamp)
	backupPath := filepath.Join(backupDir, fileName)

	// Build mysqldump command with maximum compatibility
	args := []string{
		"--single-transaction",
		"--skip-lock-tables",
		"--complete-insert",
		"--extended-insert",
		"--hex-blob",
		"--add-drop-table",
		"--disable-keys",
		fmt.Sprintf("--host=%s", c.config.Host),
		fmt.Sprintf("--port=%d", c.config.Port),
		fmt.Sprintf("--user=%s", c.config.Username),
	}

	if c.config.Password != "" {
		args = append(args, fmt.Sprintf("--password=%s", c.config.Password))
	}

	// Add database name
	args = append(args, dbName)

	cmd := exec.CommandContext(ctx, c.config.MysqldumpPath, args...)

	// Create output file
	outFile, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer outFile.Close()

	cmd.Stdout = outFile
	
	// Capture stderr to filter out warnings but keep errors
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Remove failed backup file
		os.Remove(backupPath)
		// Show actual errors
		stderrStr := stderr.String()
		if stderrStr != "" {
			return "", fmt.Errorf("mysqldump failed: %w\nOutput: %s", err, stderrStr)
		}
		return "", fmt.Errorf("mysqldump failed: %w", err)
	}
	
	// Log warnings only in debug mode (if needed)
	stderrStr := stderr.String()
	if stderrStr != "" {
		// Filter out common warnings that are not actual errors
		lines := strings.Split(stderrStr, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !isCommonWarning(line) {
				// Log non-warning messages as they might be important
				fmt.Printf("mysqldump notice: %s\n", line)
			}
		}
	}

	// Verify backup file was created and has content
	if err := c.verifyBackupFile(backupPath); err != nil {
		os.Remove(backupPath)
		return "", fmt.Errorf("backup verification failed: %w", err)
	}

	return backupPath, nil
}

func (c *Client) verifyBackupFile(backupPath string) error {
	info, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	if info.Size() == 0 {
		return fmt.Errorf("backup file is empty")
	}

	// Check if file contains SQL dump header
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	buffer := make([]byte, 100)
	n, err := file.Read(buffer)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	content := string(buffer[:n])
	if len(content) < 10 {
		return fmt.Errorf("backup file content too short")
	}

	return nil
}

func (c *Client) verifyMydumperBackup(backupDir string) error {
	// Check if metadata file exists
	metadataFile := filepath.Join(backupDir, "metadata")
	if _, err := os.Stat(metadataFile); err != nil {
		return fmt.Errorf("metadata file not found: %w", err)
	}

	// Check if backup directory has content
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("backup directory is empty")
	}

	// Check if at least one .sql file exists (excluding metadata)
	sqlFileFound := false
	for _, file := range files {
		ext := filepath.Ext(file.Name())
		if ext == ".sql" || ext == ".gz" || ext == ".lz4" || ext == ".zst" {
			sqlFileFound = true
			break
		}
	}

	if !sqlFileFound {
		return fmt.Errorf("no SQL dump files found in backup directory")
	}

	return nil
}

func (c *Client) CreateDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

func (c *Client) RestoreBackup(ctx context.Context, backupPath, dbName string) error {
	// Create a temporary logger for compression operations
	log := logger.NewLogger("info")
	
	// Auto-decompress if needed
	finalBackupPath := backupPath
	var cleanupPath string
	
	if c.isCompressedBackup(backupPath) {
		log.WithField("backup", backupPath).Info("🗜️ Decompressing backup for restore")
		
		// Create compressor for decompression
		compressionConfig := &config.CompressionConfig{
			Enabled: true,
			Format:  "tar.gz",
			Level:   6,
		}
		compressor := compression.NewCompressor(compressionConfig, log)
		
		// Decompress backup
		decompressedPath, err := compressor.DecompressBackup(backupPath)
		if err != nil {
			return fmt.Errorf("failed to decompress backup: %w", err)
		}
		
		finalBackupPath = decompressedPath
		cleanupPath = decompressedPath // Will be cleaned up after restore
		
		log.WithField("decompressed_path", decompressedPath).Info("✅ Backup decompressed successfully")
	}
	
	// Ensure cleanup happens after restore
	if cleanupPath != "" {
		defer func() {
			if err := os.RemoveAll(cleanupPath); err != nil {
				log.WithError(err).Warn("Failed to cleanup decompressed backup")
			} else {
				log.WithField("path", cleanupPath).Info("🗑️ Cleaned up decompressed backup")
			}
		}()
	}
	
	// Check if myloader is enabled and backup is from mydumper
	if c.config.Mydumper != nil && c.config.Mydumper.Enabled &&
		c.config.Mydumper.Myloader != nil && c.config.Mydumper.Myloader.Enabled {

		// Check if backup path is a directory (mydumper backup)
		if info, err := os.Stat(finalBackupPath); err == nil && info.IsDir() {
			return c.restoreWithMyloader(ctx, finalBackupPath, dbName)
		}
	}

	// Fallback to mysql restore for .sql files
	return c.restoreWithMysql(ctx, finalBackupPath, dbName)
}

func (c *Client) restoreWithMyloader(ctx context.Context, backupDir, dbName string) error {
	// Build myloader command
	args := []string{
		"--overwrite-tables",
		"--database", dbName,
		"--directory", backupDir,
		fmt.Sprintf("--threads=%d", c.config.Mydumper.Myloader.Threads),
	}

	// Use defaults-file if specified, otherwise use individual connection parameters
	if c.config.Mydumper.Myloader.DefaultsFile != "" {
		args = append(args, fmt.Sprintf("--defaults-file=%s", c.config.Mydumper.Myloader.DefaultsFile))
	} else {
		args = append(args, fmt.Sprintf("--host=%s", c.config.Host))
		args = append(args, fmt.Sprintf("--port=%d", c.config.Port))
		args = append(args, fmt.Sprintf("--user=%s", c.config.Username))
		if c.config.Password != "" {
			args = append(args, fmt.Sprintf("--password=%s", c.config.Password))
		}
	}

	cmd := exec.CommandContext(ctx, c.config.Mydumper.Myloader.BinaryPath, args...)

	// Capture stderr but don't display it unless there's an error
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = nil // Suppress stdout

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("myloader failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

func (c *Client) restoreWithMysql(ctx context.Context, backupPath, dbName string) error {
	// Build mysql command
	args := []string{
		fmt.Sprintf("--host=%s", c.config.Host),
		fmt.Sprintf("--port=%d", c.config.Port),
		fmt.Sprintf("--user=%s", c.config.Username),
		dbName,
	}

	if c.config.Password != "" {
		args = append(args, fmt.Sprintf("--password=%s", c.config.Password))
	}

	cmd := exec.CommandContext(ctx, c.config.MysqlPath, args...)

	// Open backup file
	backupFile, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer backupFile.Close()

	cmd.Stdin = backupFile

	// Capture stderr but don't display it unless there's an error
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = nil // Suppress stdout

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysql restore failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

func (c *Client) buildMydumperArgs(dbBackupDir, dbName string) []string {
	// Start with common arguments available in all supported mydumper versions
	// Supports: v0.9.1+ (Ubuntu 18.04), v0.10.0+ (most Linux distros), v0.19.3+ (macOS Homebrew)
	args := []string{
		"--routines",
		"--triggers",
		"--events",
		fmt.Sprintf("--outputdir=%s", dbBackupDir),
		fmt.Sprintf("--database=%s", dbName),
		fmt.Sprintf("--threads=%d", c.config.Mydumper.Threads),
		fmt.Sprintf("--chunk-filesize=%d", c.config.Mydumper.ChunkFilesize),
	}

	// Version-aware parameter selection for cross-platform compatibility
	if c.isMydumperVersionCompatible() {
		// Modern mydumper (v0.19.x+) - macOS Homebrew, newer Linux packages
		args = append(args, "--sync-thread-lock-mode=AUTO", "--trx-tables")
	} else {
		// Legacy mydumper (v0.9.1 - v0.10.x) - Ubuntu 18.04, CentOS, older Linux distros
		args = append(args, "--no-locks", "--trx-consistency-only")
	}

	return args
}

func (c *Client) isMydumperVersionCompatible() bool {
	// Detect mydumper version by checking for modern parameters in --help output
	// Returns true for v0.19.x+ (modern), false for v0.9.1-v0.10.x (legacy)
	// Tested versions:
	//   - v0.9.1 (Ubuntu 18.04) → legacy parameters
	//   - v0.10.0 (most Linux distros) → legacy parameters  
	//   - v0.19.3 (macOS Homebrew) → modern parameters
	cmd := exec.Command(c.config.Mydumper.BinaryPath, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If help fails, assume legacy version for safety
		return false
	}

	outputStr := string(output)
	// Check for modern parameters that exist in v0.19.x+
	return strings.Contains(outputStr, "--sync-thread-lock-mode") && 
		   strings.Contains(outputStr, "--trx-tables")
}

func (c *Client) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// ListDatabases returns a list of database names
func (c *Client) ListDatabases(ctx context.Context) ([]string, error) {
	query := "SHOW DATABASES"
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SHOW DATABASES: %w", err)
	}
	defer rows.Close()
	
	var databases []string
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			return nil, fmt.Errorf("failed to scan database name: %w", err)
		}
		
		// Skip system databases
		if dbName != "information_schema" && dbName != "performance_schema" && 
		   dbName != "mysql" && dbName != "sys" {
			databases = append(databases, dbName)
		}
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over database results: %w", err)
	}
	
	return databases, nil
}

// isCommonWarning checks if a stderr line is a common warning that can be safely ignored
func isCommonWarning(line string) bool {
	commonWarnings := []string{
		"Using a password on the command line interface can be insecure",
		"Warning] Using a password on the command line interface can be insecure",
		"[Warning] Using a password on the command line interface can be insecure",
		"mysqldump: [Warning] Using a password on the command line interface can be insecure",
	}
	
	for _, warning := range commonWarnings {
		if strings.Contains(line, warning) {
			return true
		}
	}
	return false
}

// isCompressedBackup checks if backup is compressed
func (c *Client) isCompressedBackup(backupPath string) bool {
	ext := strings.ToLower(filepath.Ext(backupPath))
	return ext == ".gz" || ext == ".zst" || ext == ".xz" || 
		   strings.HasSuffix(strings.ToLower(backupPath), ".tar.gz") ||
		   strings.HasSuffix(strings.ToLower(backupPath), ".tar.zst") ||
		   strings.HasSuffix(strings.ToLower(backupPath), ".tar.xz")
}
