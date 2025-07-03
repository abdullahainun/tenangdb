package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"db-backup-tool/internal/config"

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
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	
	// Check if mydumper is enabled in config
	if c.config.Mydumper != nil && c.config.Mydumper.Enabled {
		return c.createMydumperBackup(ctx, dbName, backupDir, timestamp)
	}
	
	// Fallback to mysqldump
	return c.createMysqldumpBackup(ctx, dbName, backupDir, timestamp)
}

func (c *Client) createMydumperBackup(ctx context.Context, dbName, backupDir, timestamp string) (string, error) {
	// Create database-specific directory
	dbBackupDir := filepath.Join(backupDir, fmt.Sprintf("%s-%s", dbName, timestamp))
	if err := os.MkdirAll(dbBackupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Build mydumper command
	args := []string{
		"--single-transaction",
		"--routines",
		"--triggers",
		"--events",
		"--use-savepoints",
		fmt.Sprintf("--host=%s", c.config.Host),
		fmt.Sprintf("--port=%d", c.config.Port),
		fmt.Sprintf("--user=%s", c.config.Username),
		fmt.Sprintf("--outputdir=%s", dbBackupDir),
		fmt.Sprintf("--database=%s", dbName),
		fmt.Sprintf("--threads=%d", c.config.Mydumper.Threads),
		fmt.Sprintf("--chunk-filesize=%d", c.config.Mydumper.ChunkFilesize),
	}

	if c.config.Password != "" {
		args = append(args, fmt.Sprintf("--password=%s", c.config.Password))
	}

	if c.config.Mydumper.CompressMethod != "" {
		args = append(args, fmt.Sprintf("--compress-method=%s", c.config.Mydumper.CompressMethod))
	}

	if c.config.Mydumper.BuildEmptyFiles {
		args = append(args, "--build-empty-files")
	}

	if c.config.Mydumper.UseDefer {
		args = append(args, "--use-defer")
	}

	if c.config.Mydumper.SingleTable {
		args = append(args, "--single-table")
	}

	if c.config.Mydumper.NoSchemas {
		args = append(args, "--no-schemas")
	}

	if c.config.Mydumper.NoData {
		args = append(args, "--no-data")
	}

	cmd := exec.CommandContext(ctx, c.config.Mydumper.BinaryPath, args...)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Remove failed backup directory
		os.RemoveAll(dbBackupDir)
		return "", fmt.Errorf("mydumper failed: %w", err)
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

	// Build mysqldump command
	args := []string{
		"--single-transaction",
		"--routines",
		"--triggers",
		"--events",
		"--flush-logs",
		"--master-data=2",
		"--skip-ssl",
		fmt.Sprintf("--host=%s", c.config.Host),
		fmt.Sprintf("--port=%d", c.config.Port),
		fmt.Sprintf("--user=%s", c.config.Username),
	}

	if c.config.Password != "" {
		args = append(args, fmt.Sprintf("--password=%s", c.config.Password))
	}

	args = append(args, dbName)

	cmd := exec.CommandContext(ctx, "mysqldump", args...)
	
	// Create output file
	outFile, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer outFile.Close()

	cmd.Stdout = outFile
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Remove failed backup file
		os.Remove(backupPath)
		return "", fmt.Errorf("mysqldump failed: %w", err)
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
		if filepath.Ext(file.Name()) == ".sql" || 
		   filepath.Ext(file.Name()) == ".gz" ||
		   filepath.Ext(file.Name()) == ".lz4" {
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

func (c *Client) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}