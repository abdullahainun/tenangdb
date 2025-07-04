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
			Name: "db_backup_duration_seconds",
			Help: "Duration of database backup operations in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600},
		},
		[]string{"database", "status"},
	)

	// Backup success counter
	BackupSuccessTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_backup_success_total",
			Help: "Total number of successful database backups",
		},
		[]string{"database"},
	)

	// Backup failure counter
	BackupFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_backup_failed_total",
			Help: "Total number of failed database backups",
		},
		[]string{"database"},
	)

	// Backup size metric
	BackupSizeBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_backup_size_bytes",
			Help: "Size of database backup in bytes",
		},
		[]string{"database"},
	)

	// Upload duration metric
	UploadDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "db_backup_upload_duration_seconds",
			Help: "Duration of backup upload operations in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600},
		},
		[]string{"database", "status"},
	)

	// Upload success counter
	UploadSuccessTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_backup_upload_success_total",
			Help: "Total number of successful backup uploads",
		},
		[]string{"database"},
	)

	// Upload failure counter
	UploadFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_backup_upload_failed_total",
			Help: "Total number of failed backup uploads",
		},
		[]string{"database"},
	)

	// Last backup timestamp
	LastBackupTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_backup_last_timestamp",
			Help: "Timestamp of the last backup operation",
		},
		[]string{"database"},
	)

	// Backup process running
	BackupProcessRunning = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_backup_process_running",
			Help: "Whether backup process is currently running (1 = running, 0 = stopped)",
		},
	)

	// Total databases configured
	TotalDatabases = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_backup_total_databases",
			Help: "Total number of databases configured for backup",
		},
	)
)

// Init initializes and registers all metrics
func Init() {
	prometheus.MustRegister(
		BackupDurationSeconds,
		BackupSuccessTotal,
		BackupFailedTotal,
		BackupSizeBytes,
		UploadDurationSeconds,
		UploadSuccessTotal,
		UploadFailedTotal,
		LastBackupTimestamp,
		BackupProcessRunning,
		TotalDatabases,
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

// RecordUploadStart records the start of an upload operation
func RecordUploadStart(database string) {
	// Can be used for tracking concurrent uploads if needed
}

// RecordUploadEnd records the end of an upload operation
func RecordUploadEnd(database string, duration time.Duration, success bool) {
	status := "success"
	if !success {
		status = "failed"
		UploadFailedTotal.WithLabelValues(database).Inc()
	} else {
		UploadSuccessTotal.WithLabelValues(database).Inc()
	}
	
	UploadDurationSeconds.WithLabelValues(database, status).Observe(duration.Seconds())
}

// SetTotalDatabases sets the total number of databases configured
func SetTotalDatabases(count int) {
	TotalDatabases.Set(float64(count))
}

// SetBackupProcessStopped marks the backup process as stopped
func SetBackupProcessStopped() {
	BackupProcessRunning.Set(0)
}

// StartMetricsServer starts HTTP server for Prometheus metrics
func StartMetricsServer(port string) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(":"+port, nil)
}