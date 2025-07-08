package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/abdullahainun/tenangdb/internal/config"
	"github.com/abdullahainun/tenangdb/internal/logger"
	"github.com/abdullahainun/tenangdb/internal/metrics"
	"github.com/abdullahainun/tenangdb/internal/upload"
	"github.com/abdullahainun/tenangdb/pkg/database"
)

type Service struct {
	config         *config.Config
	logger         *logger.Logger
	dbClient       *database.Client
	uploader       *upload.Service
	stats          *Statistics
	uploadedFiles  map[string]time.Time // Track uploaded files with timestamp
	metricsStorage *metrics.MetricsStorage
	backupTracker  *BackupTracker
	mu             sync.RWMutex
}

type Statistics struct {
	TotalDatabases    int
	SuccessfulBackups int
	FailedBackups     int
	SuccessfulUploads int
	FailedUploads     int
	StartTime         time.Time
	EndTime           time.Time
}

func NewService(cfg *config.Config, log *logger.Logger) (*Service, error) {
	// Initialize database client
	dbClient, err := database.NewClient(&cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to create database client: %w", err)
	}

	// Initialize uploader if enabled
	var uploader *upload.Service
	if cfg.Upload.Enabled {
		uploader = upload.NewService(&cfg.Upload, log)
	}

	// Initialize metrics storage
	metricsPath := "/var/lib/tenangdb/metrics.json"
	if cfg.Metrics.Enabled && cfg.Metrics.StoragePath != "" {
		metricsPath = cfg.Metrics.StoragePath
	}
	metricsStorage := metrics.NewMetricsStorage(metricsPath)

	// Initialize backup tracker
	backupTracker, err := NewBackupTracker(cfg.Backup.Directory)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup tracker: %w", err)
	}

	return &Service{
		config:         cfg,
		logger:         log,
		dbClient:       dbClient,
		uploader:       uploader,
		uploadedFiles:  make(map[string]time.Time),
		metricsStorage: metricsStorage,
		backupTracker:  backupTracker,
		stats: &Statistics{
			TotalDatabases: len(cfg.Backup.Databases),
		},
	}, nil
}

func (s *Service) Run(ctx context.Context) error {
	s.mu.Lock()
	s.stats.StartTime = time.Now()
	s.mu.Unlock()

	// Initialize metrics
	metrics.SetTotalDatabases(s.stats.TotalDatabases)
	metrics.RecordBackupStart("")

	// Update metrics storage
	s.metricsStorage.SetTotalDatabases(s.stats.TotalDatabases)
	s.metricsStorage.SetBackupProcessActive(true)

	s.logger.Debug("🚀 Starting database backup process")
	s.logger.WithField("total_databases", s.stats.TotalDatabases).Debug("📊 Backup statistics")

	// Create backup directory if it doesn't exist
	if err := s.createBackupDirectory(); err != nil {
		metrics.SetBackupProcessStopped()
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Process databases in batches
	if err := s.processDatabasesBatch(ctx); err != nil {
		metrics.SetBackupProcessStopped()
		s.metricsStorage.SetBackupProcessActive(false)
		return fmt.Errorf("batch processing failed: %w", err)
	}

	s.mu.Lock()
	s.stats.EndTime = time.Now()
	s.mu.Unlock()

	metrics.SetBackupProcessStopped()
	s.metricsStorage.SetBackupProcessActive(false)
	s.logFinalStatistics()
	return nil
}

func (s *Service) processDatabasesBatch(ctx context.Context) error {
	databases := s.config.Backup.Databases
	batchSize := s.config.Backup.BatchSize
	concurrency := s.config.Backup.Concurrency

	for i := 0; i < len(databases); i += batchSize {
		end := i + batchSize
		if end > len(databases) {
			end = len(databases)
		}

		batch := databases[i:end]
		s.logger.WithField("batch", fmt.Sprintf("%d-%d", i+1, end)).Debug("⚙️ Processing batch")

		if err := s.processBatch(ctx, batch, concurrency); err != nil {
			s.logger.WithError(err).Error("Batch processing failed")
			continue
		}

		// Add delay between batches to reduce system load
		if end < len(databases) {
			time.Sleep(time.Second * 5)
		}
	}

	return nil
}

func (s *Service) processBatch(ctx context.Context, databases []string, concurrency int) error {
	semaphore := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for _, dbName := range databases {
		wg.Add(1)
		go func(database string) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			s.processDatabase(ctx, database)
		}(dbName)
	}

	wg.Wait()
	return nil
}

func (s *Service) processDatabase(ctx context.Context, dbName string) {
	log := s.logger.WithDatabase(dbName)
	log.Debug("🔄 Starting database backup")

	backupStartTime := time.Now()

	// Create backup with retry logic
	backupPath, err := s.createBackupWithRetry(ctx, dbName)
	backupDuration := time.Since(backupStartTime)

	if err != nil {
		log.Error("❌ " + dbName + " backup failed: " + err.Error())
		s.incrementFailedBackups()
		metrics.RecordBackupEnd(dbName, backupDuration, false, 0)
		s.metricsStorage.UpdateBackupMetrics(dbName, backupDuration, false, 0)
		return
	}

	// Get backup size
	backupSize, sizeErr := s.getBackupSize(backupPath)
	if sizeErr != nil {
		log.WithError(sizeErr).Warn("Failed to get backup size")
		backupSize = 0
	}

	log.Info("✅ " + dbName + " backup completed")

	// Show backup location to user
	relativeBackupPath := s.getRelativeBackupPath(backupPath)
	s.logger.Info("📁 Backup saved: " + relativeBackupPath)

	s.incrementSuccessfulBackups()
	metrics.RecordBackupEnd(dbName, backupDuration, true, backupSize)
	s.metricsStorage.UpdateBackupMetrics(dbName, backupDuration, true, backupSize)

	// Upload to cloud if enabled
	if s.uploader != nil {
		uploadStartTime := time.Now()
		if err := s.uploadBackup(ctx, backupPath); err != nil {
			log.Error("❌ " + dbName + " upload failed: " + err.Error())
			s.incrementFailedUploads()
			metrics.RecordUploadEnd(dbName, "rclone", time.Since(uploadStartTime), false, 0)
			s.metricsStorage.UpdateUploadMetrics(dbName, time.Since(uploadStartTime), false, 0)
		} else {
			log.Info("✅ " + dbName + " upload completed")
			s.incrementSuccessfulUploads()
			metrics.RecordUploadEnd(dbName, "rclone", time.Since(uploadStartTime), true, backupSize)
			s.metricsStorage.UpdateUploadMetrics(dbName, time.Since(uploadStartTime), true, backupSize)

			// Mark backup as uploaded for potential cleanup
			s.markFileAsUploaded(backupPath)
		}
	}
}

func (s *Service) createBackupWithRetry(ctx context.Context, dbName string) (string, error) {
	var lastErr error
	retryCount := s.config.Backup.RetryCount
	retryDelay := s.config.Backup.RetryDelay

	for attempt := 1; attempt <= retryCount; attempt++ {
		if attempt > 1 {
			s.logger.WithDatabase(dbName).WithField("attempt", attempt).Info("Retrying backup")
			time.Sleep(retryDelay)
		}

		backupPath, err := s.dbClient.CreateBackup(ctx, dbName, s.config.Backup.Directory)
		if err == nil {
			return backupPath, nil
		}

		lastErr = err
		s.logger.WithDatabase(dbName).WithError(err).WithField("attempt", attempt).Warn("Backup attempt failed")
	}

	return "", fmt.Errorf("backup failed after %d attempts: %w", retryCount, lastErr)
}

func (s *Service) uploadBackup(ctx context.Context, backupPath string) error {
	// Check if backupPath is a directory (mydumper) or file (mysqldump)
	info, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("failed to stat backup path: %w", err)
	}

	if info.IsDir() {
		// For mydumper directories, upload all files in the directory
		return s.uploadDirectory(ctx, backupPath)
	} else {
		// For mysqldump files, upload single file
		return s.uploader.Upload(ctx, backupPath)
	}
}

func (s *Service) uploadDirectory(ctx context.Context, dirPath string) error {
	// Get all files in the directory
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Upload each file
	for _, file := range files {
		if file.IsDir() {
			continue // Skip subdirectories
		}

		filePath := filepath.Join(dirPath, file.Name())
		if err := s.uploader.Upload(ctx, filePath); err != nil {
			return fmt.Errorf("failed to upload file %s: %w", file.Name(), err)
		}
	}

	return nil
}

func (s *Service) createBackupDirectory() error {
	return s.dbClient.CreateDirectory(s.config.Backup.Directory)
}

func (s *Service) incrementSuccessfulBackups() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.SuccessfulBackups++
}

func (s *Service) incrementFailedBackups() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.FailedBackups++
}

func (s *Service) incrementSuccessfulUploads() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.SuccessfulUploads++
}

func (s *Service) incrementFailedUploads() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.FailedUploads++
}

func (s *Service) logFinalStatistics() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	duration := s.stats.EndTime.Sub(s.stats.StartTime)

	s.logger.WithField("statistics", map[string]interface{}{
		"total_databases":    s.stats.TotalDatabases,
		"successful_backups": s.stats.SuccessfulBackups,
		"failed_backups":     s.stats.FailedBackups,
		"successful_uploads": s.stats.SuccessfulUploads,
		"failed_uploads":     s.stats.FailedUploads,
		"duration":           duration.String(),
		"start_time":         s.stats.StartTime.Format(time.RFC3339),
		"end_time":           s.stats.EndTime.Format(time.RFC3339),
	}).Info("🗂️ " + fmt.Sprintf("%d databases backed up in %v", s.stats.SuccessfulBackups, duration.Round(time.Millisecond*100)))
}

func (s *Service) GetStatistics() Statistics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return *s.stats
}

// markFileAsUploaded marks a file as successfully uploaded
func (s *Service) markFileAsUploaded(filePath string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.uploadedFiles[filePath] = time.Now()
}

// GetUploadedFiles returns list of files that were successfully uploaded
func (s *Service) GetUploadedFiles() map[string]time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]time.Time)
	for k, v := range s.uploadedFiles {
		result[k] = v
	}
	return result
}

// CleanupUploadedFiles removes local files that have been successfully uploaded
func (s *Service) CleanupUploadedFiles(ctx context.Context) error {
	s.mu.RLock()
	uploadedFiles := make(map[string]time.Time)
	for k, v := range s.uploadedFiles {
		uploadedFiles[k] = v
	}
	s.mu.RUnlock()

	if len(uploadedFiles) == 0 {
		s.logger.Info("No uploaded files to cleanup")
		return nil
	}

	s.logger.WithField("files_to_cleanup", len(uploadedFiles)).Info("Starting cleanup of uploaded files")

	var cleanedFiles []string
	var totalSize int64

	for filePath, uploadTime := range uploadedFiles {
		// Only cleanup files that were uploaded more than 1 hour ago (safety buffer)
		if time.Since(uploadTime) < time.Hour {
			continue
		}

		if err := s.removeBackupFile(filePath); err != nil {
			s.logger.WithError(err).WithField("file", filePath).Error("Failed to remove uploaded file")
			continue
		}

		cleanedFiles = append(cleanedFiles, filePath)
		s.logger.WithField("file", filePath).Info("Removed uploaded backup file")
	}

	// Remove cleaned files from tracking
	s.mu.Lock()
	for _, filePath := range cleanedFiles {
		delete(s.uploadedFiles, filePath)
	}
	s.mu.Unlock()

	s.logger.WithField("cleanup_stats", map[string]interface{}{
		"files_cleaned": len(cleanedFiles),
		"total_size_mb": totalSize / (1024 * 1024),
	}).Info("Cleanup of uploaded files completed")

	return nil
}

// removeBackupFile safely removes a backup file with size calculation
func (s *Service) removeBackupFile(backupPath string) error {
	info, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("failed to stat backup path: %w", err)
	}

	var totalSize int64
	if info.IsDir() {
		// For mydumper directories, calculate total size and remove directory
		totalSize, err = s.calculateDirectorySize(backupPath)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to calculate directory size")
			totalSize = 0
		}

		if err := os.RemoveAll(backupPath); err != nil {
			return fmt.Errorf("failed to remove directory: %w", err)
		}
	} else {
		// For mysqldump files, remove single file
		totalSize = info.Size()
		if err := os.Remove(backupPath); err != nil {
			return fmt.Errorf("failed to remove file: %w", err)
		}
	}

	s.logger.WithField("backup_size_mb", totalSize/(1024*1024)).Debug("Backup removed successfully")
	return nil
}

func (s *Service) calculateDirectorySize(dirPath string) (int64, error) {
	var totalSize int64

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return totalSize, err
}

// getBackupSize calculates the size of a backup file or directory
func (s *Service) getBackupSize(backupPath string) (int64, error) {
	info, err := os.Stat(backupPath)
	if err != nil {
		return 0, fmt.Errorf("failed to stat backup path: %w", err)
	}

	if info.IsDir() {
		// For mydumper directories, calculate total size
		return s.calculateDirectorySize(backupPath)
	} else {
		// For mysqldump files, return file size
		return info.Size(), nil
	}
}

// getRelativeBackupPath converts absolute backup path to relative path from backup directory
func (s *Service) getRelativeBackupPath(backupPath string) string {
	// Get the backup directory from config
	backupDir := s.config.Backup.Directory

	// Try to make the path relative to the backup directory
	if relPath, err := filepath.Rel(backupDir, backupPath); err == nil {
		return filepath.Join(filepath.Base(backupDir), relPath)
	}

	// If relative path conversion fails, return just the path relative to current directory
	if relPath, err := filepath.Rel(".", backupPath); err == nil {
		return relPath
	}

	// Fallback to absolute path
	return backupPath
}
