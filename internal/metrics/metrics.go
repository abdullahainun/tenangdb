package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Backup duration metric
	BackupDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "tenangdb_backup_duration_seconds",
			Help: "Duration of database backup operations in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600},
		},
		[]string{"database", "status"},
	)

	// Backup success counter
	BackupSuccessTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tenangdb_backup_success_total",
			Help: "Total number of successful database backups",
		},
		[]string{"database"},
	)

	// Backup failure counter
	BackupFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tenangdb_backup_failed_total",
			Help: "Total number of failed database backups",
		},
		[]string{"database"},
	)

	// Backup size metric
	BackupSizeBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tenangdb_backup_size_bytes",
			Help: "Size of database backup in bytes",
		},
		[]string{"database"},
	)

	// Upload duration metric
	UploadDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "tenangdb_upload_duration_seconds",
			Help: "Duration of backup upload operations in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600},
		},
		[]string{"database", "status"},
	)

	// Upload success counter
	UploadSuccessTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tenangdb_upload_success_total",
			Help: "Total number of successful backup uploads",
		},
		[]string{"database"},
	)

	// Upload failure counter
	UploadFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tenangdb_upload_failed_total",
			Help: "Total number of failed backup uploads",
		},
		[]string{"database"},
	)

	// Last backup timestamp
	LastBackupTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tenangdb_backup_last_timestamp",
			Help: "Timestamp of the last backup operation",
		},
		[]string{"database"},
	)

	// Backup process running
	BackupProcessRunning = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "tenangdb_backup_process_running",
			Help: "Whether backup process is currently running (1 = running, 0 = stopped)",
		},
	)

	// Total databases configured
	TotalDatabases = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "tenangdb_total_databases",
			Help: "Total number of databases configured for backup",
		},
	)

	// === RESTORE METRICS ===
	
	// Restore duration metric
	RestoreDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "tenangdb_restore_duration_seconds",
			Help: "Duration of database restore operations in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600},
		},
		[]string{"database", "status"},
	)

	// Restore success counter
	RestoreSuccessTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tenangdb_restore_success_total",
			Help: "Total number of successful database restores",
		},
		[]string{"database"},
	)

	// Restore failure counter
	RestoreFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tenangdb_restore_failed_total",
			Help: "Total number of failed database restores",
		},
		[]string{"database"},
	)

	// Last restore timestamp
	LastRestoreTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tenangdb_restore_last_timestamp",
			Help: "Timestamp of the last restore operation",
		},
		[]string{"database"},
	)

	// === UPLOAD METRICS ===
	
	// Upload bytes transferred
	UploadBytesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tenangdb_upload_bytes_total",
			Help: "Total bytes uploaded to cloud storage",
		},
		[]string{"database", "provider"},
	)

	// Upload active connections
	UploadActiveConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tenangdb_upload_active_connections",
			Help: "Number of active upload connections",
		},
		[]string{"provider"},
	)

	// === SYSTEM METRICS ===
	
	// System health status
	SystemHealthStatus = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "tenangdb_system_health_status",
			Help: "System health status (1 = healthy, 0 = unhealthy)",
		},
	)

	// Database connections
	DatabaseConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tenangdb_database_connections",
			Help: "Number of active database connections",
		},
		[]string{"database", "status"},
	)

	// Memory usage
	MemoryUsageBytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "tenangdb_memory_usage_bytes",
			Help: "Memory usage in bytes",
		},
	)

	// Disk usage
	DiskUsageBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tenangdb_disk_usage_bytes",
			Help: "Disk usage in bytes",
		},
		[]string{"path", "type"},
	)

	// Active operations
	ActiveOperations = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tenangdb_active_operations",
			Help: "Number of active operations",
		},
		[]string{"operation_type"},
	)
)

// Init initializes and registers all metrics
func Init() {
	prometheus.MustRegister(
		// Backup metrics
		BackupDurationSeconds,
		BackupSuccessTotal,
		BackupFailedTotal,
		BackupSizeBytes,
		LastBackupTimestamp,
		BackupProcessRunning,
		
		// Upload metrics
		UploadDurationSeconds,
		UploadSuccessTotal,
		UploadFailedTotal,
		UploadBytesTotal,
		UploadActiveConnections,
		
		// Restore metrics
		RestoreDurationSeconds,
		RestoreSuccessTotal,
		RestoreFailedTotal,
		LastRestoreTimestamp,
		
		// System metrics
		TotalDatabases,
		SystemHealthStatus,
		DatabaseConnections,
		MemoryUsageBytes,
		DiskUsageBytes,
		ActiveOperations,
	)
}

// RecordBackupStart records the start of a backup operation
func RecordBackupStart(database string) {
	BackupProcessRunning.Set(1)
}

// RecordBackupEnd records the end of a backup operation
func RecordBackupEnd(database string, duration time.Duration, success bool, sizeBytes int64) {
	status := "success"
	if !success {
		status = "failed"
		BackupFailedTotal.WithLabelValues(database).Inc()
	} else {
		BackupSuccessTotal.WithLabelValues(database).Inc()
		BackupSizeBytes.WithLabelValues(database).Set(float64(sizeBytes))
	}
	
	BackupDurationSeconds.WithLabelValues(database, status).Observe(duration.Seconds())
	LastBackupTimestamp.WithLabelValues(database).Set(float64(time.Now().Unix()))
}


// SetTotalDatabases sets the total number of databases configured
func SetTotalDatabases(count int) {
	TotalDatabases.Set(float64(count))
}

// SetBackupProcessStopped marks the backup process as stopped
func SetBackupProcessStopped() {
	BackupProcessRunning.Set(0)
}

// === RESTORE FUNCTIONS ===

// RecordRestoreStart records the start of a restore operation
func RecordRestoreStart(database string) {
	ActiveOperations.WithLabelValues("restore").Inc()
}

// RecordRestoreEnd records the end of a restore operation
func RecordRestoreEnd(database string, duration time.Duration, success bool) {
	status := "success"
	if !success {
		status = "failed"
		RestoreFailedTotal.WithLabelValues(database).Inc()
	} else {
		RestoreSuccessTotal.WithLabelValues(database).Inc()
	}
	
	RestoreDurationSeconds.WithLabelValues(database, status).Observe(duration.Seconds())
	LastRestoreTimestamp.WithLabelValues(database).Set(float64(time.Now().Unix()))
	ActiveOperations.WithLabelValues("restore").Dec()
}

// === UPLOAD FUNCTIONS ===

// RecordUploadBytes records bytes uploaded
func RecordUploadBytes(database, provider string, bytes int64) {
	UploadBytesTotal.WithLabelValues(database, provider).Add(float64(bytes))
}

// SetUploadActiveConnections sets the number of active upload connections
func SetUploadActiveConnections(provider string, count int) {
	UploadActiveConnections.WithLabelValues(provider).Set(float64(count))
}

// RecordUploadStart records the start of an upload operation
func RecordUploadStart(database, provider string) {
	ActiveOperations.WithLabelValues("upload").Inc()
	UploadActiveConnections.WithLabelValues(provider).Inc()
}

// RecordUploadEnd records the end of an upload operation
func RecordUploadEnd(database, provider string, duration time.Duration, success bool, bytesUploaded int64) {
	status := "success"
	if !success {
		status = "failed"
		UploadFailedTotal.WithLabelValues(database).Inc()
	} else {
		UploadSuccessTotal.WithLabelValues(database).Inc()
		UploadBytesTotal.WithLabelValues(database, provider).Add(float64(bytesUploaded))
	}
	
	UploadDurationSeconds.WithLabelValues(database, status).Observe(duration.Seconds())
	ActiveOperations.WithLabelValues("upload").Dec()
	UploadActiveConnections.WithLabelValues(provider).Dec()
}

// === SYSTEM FUNCTIONS ===

// SetSystemHealth sets the system health status
func SetSystemHealth(healthy bool) {
	if healthy {
		SystemHealthStatus.Set(1)
	} else {
		SystemHealthStatus.Set(0)
	}
}

// SetDatabaseConnections sets the number of database connections
func SetDatabaseConnections(database, status string, count int) {
	DatabaseConnections.WithLabelValues(database, status).Set(float64(count))
}

// SetMemoryUsage sets the memory usage in bytes
func SetMemoryUsage(bytes int64) {
	MemoryUsageBytes.Set(float64(bytes))
}

// SetDiskUsage sets the disk usage in bytes
func SetDiskUsage(path, usageType string, bytes int64) {
	DiskUsageBytes.WithLabelValues(path, usageType).Set(float64(bytes))
}

// SetActiveOperations sets the number of active operations
func SetActiveOperations(operationType string, count int) {
	ActiveOperations.WithLabelValues(operationType).Set(float64(count))
}

// StartMetricsServer starts HTTP server for Prometheus metrics
func StartMetricsServer(port string) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(":"+port, nil)
}