package upload

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
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

func (s *Service) Upload(ctx context.Context, filePath string) error {
	if !s.config.Enabled {
		return nil
	}

	fileName := filepath.Base(filePath)
	log := s.logger.WithField("backup_file", fileName)

	log.Info("Starting cloud upload")

	// Upload with retry logic
	var lastErr error
	for attempt := 1; attempt <= s.config.RetryCount; attempt++ {
		if attempt > 1 {
			log.WithField("attempt", attempt).Info("Retrying upload")
			time.Sleep(time.Second * 10)
		}

		if err := s.uploadFile(ctx, filePath); err == nil {
			log.Info("Upload completed successfully")
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

	// Build rclone command
	args := []string{
		"copy",
		filePath,
		s.config.Destination,
		"--progress",
		"--stats", "10s",
		"--checksum",
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

	cmd := exec.CommandContext(cleanupCtx, s.config.RclonePath, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rclone cleanup failed: %w (output: %s)", err, string(output))
	}

	s.logger.WithField("output", string(output)).Info("Remote cleanup completed")
	return nil
}
