package database

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/abdullahainun/tenangdb/internal/config"
)

type PostgreSQLClient struct {
	config *config.DatabaseConfig
}

func NewPostgreSQLClient(cfg *config.DatabaseConfig) (*PostgreSQLClient, error) {
	return &PostgreSQLClient{config: cfg}, nil
}

func (c *PostgreSQLClient) pgDumpPath() string {
	return c.findBinary("pg_dump")
}

func (c *PostgreSQLClient) pgRestorePath() string {
	return c.findBinary("pg_restore")
}

func (c *PostgreSQLClient) psqlPath() string {
	return c.findBinary("psql")
}

func (c *PostgreSQLClient) findBinary(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return name
	}
	return path
}

func (c *PostgreSQLClient) envVars() []string {
	return []string{
		fmt.Sprintf("PGHOST=%s", c.config.Host),
		fmt.Sprintf("PGPORT=%d", c.config.Port),
		fmt.Sprintf("PGUSER=%s", c.config.Username),
		fmt.Sprintf("PGPASSWORD=%s", c.config.Password),
	}
}

func (c *PostgreSQLClient) CreateBackup(ctx context.Context, dbName, backupDir string) (string, error) {
	now := time.Now()
	timestamp := now.Format("2006-01-02_15-04-05")
	yearMonth := now.Format("2006-01")
	organizedBackupDir := filepath.Join(backupDir, dbName, yearMonth)

	if err := os.MkdirAll(organizedBackupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	fileName := fmt.Sprintf("%s-%s.dump", dbName, timestamp)
	backupPath := filepath.Join(organizedBackupDir, fileName)

	args := []string{
		fmt.Sprintf("--host=%s", c.config.Host),
		fmt.Sprintf("--port=%d", c.config.Port),
		fmt.Sprintf("--username=%s", c.config.Username),
		"--format=custom",
		"--compress=9",
		fmt.Sprintf("--file=%s", backupPath),
		dbName,
	}

	cmd := exec.CommandContext(ctx, c.pgDumpPath(), args...)
	cmd.Env = append(os.Environ(), c.envVars()...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		os.Remove(backupPath)
		return "", fmt.Errorf("pg_dump failed: %w, stderr: %s", err, stderr.String())
	}

	if err := c.verifyDumpFile(backupPath); err != nil {
		os.Remove(backupPath)
		return "", fmt.Errorf("backup verification failed: %w", err)
	}

	return backupPath, nil
}

func (c *PostgreSQLClient) verifyDumpFile(backupPath string) error {
	info, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("dump file not found: %w", err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("dump file is empty")
	}
	return nil
}

func (c *PostgreSQLClient) CreateDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

func (c *PostgreSQLClient) RestoreBackup(ctx context.Context, backupPath, dbName string) error {
	args := []string{
		fmt.Sprintf("--host=%s", c.config.Host),
		fmt.Sprintf("--port=%d", c.config.Port),
		fmt.Sprintf("--username=%s", c.config.Username),
		"--clean",
		"--if-exists",
		"--dbname=" + dbName,
		backupPath,
	}

	cmd := exec.CommandContext(ctx, c.pgRestorePath(), args...)
	cmd.Env = append(os.Environ(), c.envVars()...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			// Exit code 1 means non-fatal errors (e.g. unknown GUC settings
			// from newer pg_dump) — data was restored. Log and continue.
			if !isFatalRestoreError(stderr.String()) {
				fmt.Fprintf(os.Stderr, "pg_restore: non-fatal warnings: %s", stderr.String())
				return nil
			}
		}
		return fmt.Errorf("pg_restore failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// isFatalRestoreError checks if stderr contains a genuinely fatal error
// vs benign warnings like unknown GUC settings from version mismatch.
func isFatalRestoreError(stderr string) bool {
	fatal := []string{
		"could not open input file",
		"connection to server",
		"out of memory",
		"permission denied",
		"does not exist",
	}
	for _, s := range fatal {
		if strings.Contains(stderr, s) {
			return true
		}
	}
	return false
}

func (c *PostgreSQLClient) Close() error {
	return nil
}

func (c *PostgreSQLClient) ListDatabases(ctx context.Context) ([]string, error) {
	args := []string{
		fmt.Sprintf("--host=%s", c.config.Host),
		fmt.Sprintf("--port=%d", c.config.Port),
		fmt.Sprintf("--username=%s", c.config.Username),
		"-l",
		"-q",
		"-A",
		"-t",
	}

	cmd := exec.CommandContext(ctx, c.psqlPath(), args...)
	cmd.Env = append(os.Environ(), c.envVars()...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("psql failed: %w, stderr: %s", err, stderr.String())
	}

	var databases []string
	for _, line := range strings.Split(stdout.String(), "\n") {
		dbName := strings.TrimSpace(line)
		if dbName == "" {
			continue
		}
		if strings.HasPrefix(dbName, "template") || dbName == "postgres" {
			continue
		}
		databases = append(databases, dbName)
	}

	return databases, nil
}
