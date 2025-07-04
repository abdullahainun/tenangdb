package backup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/abdullahainun/tenangdb/internal/config"
	"github.com/abdullahainun/tenangdb/internal/logger"
)

type CleanupService struct {
	config       *config.CleanupConfig
	uploadConfig *config.UploadConfig
	logger       *logger.Logger
}

func NewCleanupService(config *config.CleanupConfig, uploadConfig *config.UploadConfig, logger *logger.Logger) *CleanupService {
	return &CleanupService{
		config:       config,
		uploadConfig: uploadConfig,
		logger:       logger,
	}
}

// CleanupLocal is now deprecated in favor of the new smart cleanup system
// This function is kept for backward compatibility but should not be used
func (c *CleanupService) CleanupLocal(ctx context.Context, backupDir string) error {
	c.logger.Info("CleanupLocal is deprecated. Use CleanupUploadedFiles instead for smart cleanup.")
	return nil
}

func (c *CleanupService) GetOldFiles(backupDir string, retentionDays int) ([]string, error) {
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	var oldFiles []string

	err := filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && info.ModTime().Before(cutoffTime) {
			oldFiles = append(oldFiles, path)
		}

		return nil
	})

	return oldFiles, err
}

// verifyFileExistsInCloud checks if a local file exists in cloud storage
func (c *CleanupService) verifyFileExistsInCloud(localPath, backupDir string) bool {
	if !c.config.VerifyCloudExists || !c.uploadConfig.Enabled {
		return false
	}

	// Convert local path to relative path from backup directory
	relPath, err := filepath.Rel(backupDir, localPath)
	if err != nil {
		c.logger.WithError(err).Warnf("Failed to get relative path for %s", localPath)
		return false
	}

	// Construct remote path
	remotePath := filepath.Join(c.uploadConfig.Destination, relPath)
	
	// Use rclone to check if file exists
	rclonePath := c.uploadConfig.RclonePath
	if rclonePath == "" {
		rclonePath = "/usr/bin/rclone"
	}

	cmd := exec.Command(rclonePath, "lsf", remotePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		c.logger.WithError(err).Debugf("File %s not found in cloud or rclone error", remotePath)
		return false
	}

	// Check if output contains the file
	outputStr := strings.TrimSpace(string(output))
	if outputStr != "" {
		c.logger.Debugf("File %s verified in cloud", remotePath)
		return true
	}

	return false
}

// CleanupAgeBasedFiles removes old files based on age with cloud verification
func (c *CleanupService) CleanupAgeBasedFiles(ctx context.Context, backupDir string, selectedDatabases []string) error {
	if !c.config.AgeBasedCleanup {
		c.logger.Debug("Age-based cleanup is disabled")
		return nil
	}

	c.logger.Infof("Starting age-based cleanup with max age: %d days", c.config.MaxAgeDays)

	cutoffTime := time.Now().AddDate(0, 0, -c.config.MaxAgeDays)
	var filesToDelete []string
	var totalSize int64

	err := filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file is old enough
		if info.ModTime().Before(cutoffTime) {
			// Check if file should be cleaned up based on database filter
			if !c.shouldCleanupFile(path, selectedDatabases) {
				return nil
			}

			// If cloud verification is enabled, verify file exists in cloud
			if c.config.VerifyCloudExists {
				if !c.verifyFileExistsInCloud(path, backupDir) {
					c.logger.Warnf("File %s is old but not found in cloud, skipping deletion for safety", path)
					return nil
				}
			}

			filesToDelete = append(filesToDelete, path)
			totalSize += info.Size()
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan backup directory: %w", err)
	}

	if len(filesToDelete) == 0 {
		c.logger.Info("No old files found for age-based cleanup")
		return nil
	}

	c.logger.Infof("Found %d old files to delete (total size: %d bytes)", len(filesToDelete), totalSize)

	// Delete files
	deletedCount := 0
	deletedSize := int64(0)
	for _, filePath := range filesToDelete {
		info, err := os.Stat(filePath)
		if err != nil {
			c.logger.WithError(err).Warnf("Failed to stat file %s", filePath)
			continue
		}

		if err := os.Remove(filePath); err != nil {
			c.logger.WithError(err).Errorf("Failed to delete file %s", filePath)
			continue
		}

		deletedCount++
		deletedSize += info.Size()
		c.logger.Infof("Deleted old file: %s (size: %d bytes)", filePath, info.Size())
	}

	c.logger.Infof("Age-based cleanup completed: deleted %d files, freed %d bytes", deletedCount, deletedSize)
	return nil
}

// GetConfig returns the cleanup configuration
func (c *CleanupService) GetConfig() *config.CleanupConfig {
	return c.config
}

// shouldCleanupFile checks if a file should be cleaned up based on database filter
func (c *CleanupService) shouldCleanupFile(filePath string, selectedDatabases []string) bool {
	if len(selectedDatabases) == 0 {
		return true // no filter, cleanup all
	}

	// Extract database name from file path
	// Expected format: /path/to/backup/database_name/file.sql.gz
	parts := strings.Split(filePath, "/")
	if len(parts) < 2 {
		return false
	}

	// Find database name in path
	var dbName string
	for _, part := range parts {
		if part != "" && part != "." && part != ".." {
			// Check if this part looks like a database name
			// by checking if it matches any of the selected databases
			for _, selectedDB := range selectedDatabases {
				if strings.Contains(part, selectedDB) {
					dbName = selectedDB
					break
				}
			}
			if dbName != "" {
				break
			}
		}
	}

	// If no database found in path, check filename
	if dbName == "" {
		filename := parts[len(parts)-1]
		for _, selectedDB := range selectedDatabases {
			if strings.HasPrefix(filename, selectedDB) {
				dbName = selectedDB
				break
			}
		}
	}

	// Check if database should be cleaned up
	for _, selectedDB := range selectedDatabases {
		if dbName == selectedDB {
			return true
		}
	}

	return false
}
