package backup

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"db-backup-tool/internal/config"
	"db-backup-tool/internal/logger"
)

type CleanupService struct {
	config *config.CleanupConfig
	logger *logger.Logger
}

func NewCleanupService(config *config.CleanupConfig, logger *logger.Logger) *CleanupService {
	return &CleanupService{
		config: config,
		logger: logger,
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
