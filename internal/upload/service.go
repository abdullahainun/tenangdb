package upload

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/abdullahainun/tenangdb/internal/config"
	"github.com/abdullahainun/tenangdb/internal/logger"
)

type Service struct {
	config *config.UploadConfig
	logger *logger.Logger
}

func NewService(config *config.UploadConfig, logger *logger.Logger) *Service {
	return &Service{
		config: config,
		logger: logger,
	}
}

// extractBackupInfo extracts database name and date from backup file path
// Expected path format: {baseDir}/{database}/{YYYY-MM}/{filename}
func extractBackupInfo(filePath string) (database, date string) {
	// Split the path into parts
	parts := strings.Split(filepath.Clean(filePath), string(filepath.Separator))
	
	// Find the backup directory structure
	// Look for the pattern: {database}/{YYYY-MM}/{filename}
	for i := len(parts) - 3; i >= 0; i-- {
		if len(parts) > i+2 {
			// Check if the pattern matches database/YYYY-MM/filename
			datePattern := parts[i+1]
			if len(datePattern) == 7 && datePattern[4] == '-' {
				// Looks like YYYY-MM format
				database = parts[i]
				date = datePattern
				return
			}
		}
	}
	
	// Fallback: extract database from filename if pattern not found
	filename := filepath.Base(filePath)
	if dashIndex := strings.Index(filename, "-"); dashIndex > 0 {
		database = filename[:dashIndex]
	}
	
	return
}

func (s *Service) Upload(ctx context.Context, filePath string) error {
	if !s.config.Enabled {
		return nil
	}

	fileName := filepath.Base(filePath)
	log := s.logger.WithField("backup_file", fileName)

	log.Info("☁️  Uploading " + fileName + " to cloud")

	// Upload with retry logic
	var lastErr error
	for attempt := 1; attempt <= s.config.RetryCount; attempt++ {
		if attempt > 1 {
			log.WithField("attempt", attempt).Info("Retrying upload")
			time.Sleep(time.Second * 10)
		}

		if err := s.uploadFile(ctx, filePath); err == nil {
			log.Info("☁️  Upload completed successfully")
			return nil
		} else {
			lastErr = err
			log.WithError(err).WithField("attempt", attempt).Warn("Upload attempt failed")
		}
	}

	return fmt.Errorf("upload failed after %d attempts: %w", s.config.RetryCount, lastErr)
}

func (s *Service) uploadFile(ctx context.Context, filePath string) error {
	// Create context with timeout
	uploadCtx, cancel := context.WithTimeout(ctx, time.Duration(s.config.Timeout)*time.Second)
	defer cancel()

	// Extract database and date from backup path
	database, date := extractBackupInfo(filePath)
	
	// Construct organized destination path
	destination := s.config.Destination
	if database != "" {
		destination = strings.TrimSuffix(destination, "/") + "/" + database
		if date != "" {
			destination = destination + "/" + date
		}
	}

	// Build rclone command
	args := []string{
		"copy",
		filePath,
		destination,
		"--progress",
		"--stats", "10s",
		"--checksum",
	}

	// Add config path if specified
	if s.config.RcloneConfigPath != "" {
		args = append(args, "--config", s.config.RcloneConfigPath)
	}

	cmd := exec.CommandContext(uploadCtx, s.config.RclonePath, args...)

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rclone command failed: %w (output: %s)", err, string(output))
	}

	return nil
}

func (s *Service) CleanupRemote(ctx context.Context, retentionDays int) error {
	if !s.config.Enabled {
		return nil
	}

	s.logger.WithField("retention_days", retentionDays).Info("Starting remote cleanup")

	// Create context with timeout
	cleanupCtx, cancel := context.WithTimeout(ctx, time.Duration(s.config.Timeout)*time.Second)
	defer cancel()

	// Build rclone delete command
	args := []string{
		"delete",
		s.config.Destination,
		"--min-age", fmt.Sprintf("%dd", retentionDays),
		"--dry-run", // Remove this flag in production
	}

	// Add config path if specified
	if s.config.RcloneConfigPath != "" {
		args = append(args, "--config", s.config.RcloneConfigPath)
	}

	cmd := exec.CommandContext(cleanupCtx, s.config.RclonePath, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rclone cleanup failed: %w (output: %s)", err, string(output))
	}

	s.logger.WithField("output", string(output)).Info("Remote cleanup completed")
	return nil
}
