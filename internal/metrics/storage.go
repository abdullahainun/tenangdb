package metrics

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MetricsStorage handles persistent storage of metrics data
type MetricsStorage struct {
	filePath string
	mu       sync.RWMutex
}

// BackupMetrics represents metrics for a single database backup
type BackupMetrics struct {
	Database        string    `json:"database"`
	LastBackup      time.Time `json:"last_backup"`
	SizeBytes       int64     `json:"size_bytes"`
	DurationSeconds float64   `json:"duration_seconds"`
	Status          string    `json:"status"`
	SuccessCount    int64     `json:"success_count"`
	FailureCount    int64     `json:"failure_count"`
}

// UploadMetrics represents metrics for upload operations
type UploadMetrics struct {
	Database        string    `json:"database"`
	LastUpload      time.Time `json:"last_upload"`
	DurationSeconds float64   `json:"duration_seconds"`
	Status          string    `json:"status"`
	BytesUploaded   int64     `json:"bytes_uploaded"`
	SuccessCount    int64     `json:"success_count"`
	FailureCount    int64     `json:"failure_count"`
}

// RestoreMetrics represents metrics for restore operations
type RestoreMetrics struct {
	Database        string    `json:"database"`
	LastRestore     time.Time `json:"last_restore"`
	DurationSeconds float64   `json:"duration_seconds"`
	Status          string    `json:"status"`
	SuccessCount    int64     `json:"success_count"`
	FailureCount    int64     `json:"failure_count"`
}

// CleanupMetrics represents metrics for cleanup operations
type CleanupMetrics struct {
	LastCleanup     time.Time `json:"last_cleanup"`
	FilesRemoved    int64     `json:"files_removed"`
	BytesFreed      int64     `json:"bytes_freed"`
	DurationSeconds float64   `json:"duration_seconds"`
	Status          string    `json:"status"`
	SuccessCount    int64     `json:"success_count"`
	FailureCount    int64     `json:"failure_count"`
}

// SystemMetrics represents system-level metrics
type SystemMetrics struct {
	TotalDatabases      int       `json:"total_databases"`
	LastBackupProcess   time.Time `json:"last_backup_process"`
	BackupProcessActive bool      `json:"backup_process_active"`
	SystemHealthy       bool      `json:"system_healthy"`
}

// MetricsData represents the complete metrics data structure
type MetricsData struct {
	System   SystemMetrics             `json:"system"`
	Backups  map[string]BackupMetrics  `json:"backups"`
	Uploads  map[string]UploadMetrics  `json:"uploads"`
	Restores map[string]RestoreMetrics `json:"restores"`
	Cleanup  CleanupMetrics            `json:"cleanup"`
}

// NewMetricsStorage creates a new metrics storage instance
func NewMetricsStorage(filePath string) *MetricsStorage {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		// Log the error but continue - we'll handle it when trying to write
		log.Printf("Warning: failed to create directory %s: %v", dir, err)
	}
	
	return &MetricsStorage{
		filePath: filePath,
	}
}

// LoadMetrics loads metrics from storage file
func (s *MetricsStorage) LoadMetrics() (*MetricsData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Initialize default data
	data := &MetricsData{
		System: SystemMetrics{
			SystemHealthy: true,
		},
		Backups:  make(map[string]BackupMetrics),
		Uploads:  make(map[string]UploadMetrics),
		Restores: make(map[string]RestoreMetrics),
		Cleanup:  CleanupMetrics{},
	}
	
	// Check if file exists
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		return data, nil
	}
	
	// Read file
	fileData, err := os.ReadFile(s.filePath)
	if err != nil {
		return data, fmt.Errorf("failed to read metrics file: %w", err)
	}
	
	// Parse JSON
	if err := json.Unmarshal(fileData, data); err != nil {
		return data, fmt.Errorf("failed to parse metrics file: %w", err)
	}
	
	return data, nil
}

// SaveMetrics saves metrics to storage file
func (s *MetricsStorage) SaveMetrics(data *MetricsData) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Marshal to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics data: %w", err)
	}
	
	// Write to temp file first
	tempFile := s.filePath + ".tmp"
	if err := os.WriteFile(tempFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write temp metrics file: %w", err)
	}
	
	// Atomic rename
	if err := os.Rename(tempFile, s.filePath); err != nil {
		return fmt.Errorf("failed to rename metrics file: %w", err)
	}
	
	return nil
}

// UpdateBackupMetrics updates backup metrics for a database
func (s *MetricsStorage) UpdateBackupMetrics(database string, duration time.Duration, success bool, sizeBytes int64) error {
	data, err := s.LoadMetrics()
	if err != nil {
		return err
	}
	
	// Get existing metrics or create new
	backup, exists := data.Backups[database]
	if !exists {
		backup = BackupMetrics{
			Database: database,
		}
	}
	
	// Update metrics
	backup.LastBackup = time.Now()
	backup.DurationSeconds = duration.Seconds()
	backup.SizeBytes = sizeBytes
	
	if success {
		backup.Status = "success"
		backup.SuccessCount++
	} else {
		backup.Status = "failed"
		backup.FailureCount++
	}
	
	data.Backups[database] = backup
	data.System.LastBackupProcess = time.Now()
	
	return s.SaveMetrics(data)
}

// UpdateUploadMetrics updates upload metrics for a database
func (s *MetricsStorage) UpdateUploadMetrics(database string, duration time.Duration, success bool, bytesUploaded int64) error {
	data, err := s.LoadMetrics()
	if err != nil {
		return err
	}
	
	// Get existing metrics or create new
	upload, exists := data.Uploads[database]
	if !exists {
		upload = UploadMetrics{
			Database: database,
		}
	}
	
	// Update metrics
	upload.LastUpload = time.Now()
	upload.DurationSeconds = duration.Seconds()
	upload.BytesUploaded = bytesUploaded
	
	if success {
		upload.Status = "success"
		upload.SuccessCount++
	} else {
		upload.Status = "failed"
		upload.FailureCount++
	}
	
	data.Uploads[database] = upload
	
	return s.SaveMetrics(data)
}

// SetBackupProcessActive sets the backup process status
func (s *MetricsStorage) SetBackupProcessActive(active bool) error {
	data, err := s.LoadMetrics()
	if err != nil {
		return err
	}
	
	data.System.BackupProcessActive = active
	if !active {
		data.System.LastBackupProcess = time.Now()
	}
	
	return s.SaveMetrics(data)
}

// UpdateRestoreMetrics updates restore metrics for a database
func (s *MetricsStorage) UpdateRestoreMetrics(database string, duration time.Duration, success bool) error {
	data, err := s.LoadMetrics()
	if err != nil {
		return err
	}
	
	// Get existing metrics or create new
	restore, exists := data.Restores[database]
	if !exists {
		restore = RestoreMetrics{
			Database: database,
		}
	}
	
	// Update metrics
	restore.LastRestore = time.Now()
	restore.DurationSeconds = duration.Seconds()
	
	if success {
		restore.Status = "success"
		restore.SuccessCount++
	} else {
		restore.Status = "failed"
		restore.FailureCount++
	}
	
	data.Restores[database] = restore
	
	return s.SaveMetrics(data)
}

// UpdateCleanupMetrics updates cleanup metrics
func (s *MetricsStorage) UpdateCleanupMetrics(duration time.Duration, success bool, filesRemoved int64, bytesFreed int64) error {
	data, err := s.LoadMetrics()
	if err != nil {
		return err
	}
	
	// Update cleanup metrics
	data.Cleanup.LastCleanup = time.Now()
	data.Cleanup.DurationSeconds = duration.Seconds()
	data.Cleanup.FilesRemoved += filesRemoved
	data.Cleanup.BytesFreed += bytesFreed
	
	if success {
		data.Cleanup.Status = "success"
		data.Cleanup.SuccessCount++
	} else {
		data.Cleanup.Status = "failed"
		data.Cleanup.FailureCount++
	}
	
	return s.SaveMetrics(data)
}

// SetTotalDatabases sets the total number of databases
func (s *MetricsStorage) SetTotalDatabases(count int) error {
	data, err := s.LoadMetrics()
	if err != nil {
		return err
	}
	
	data.System.TotalDatabases = count
	
	return s.SaveMetrics(data)
}