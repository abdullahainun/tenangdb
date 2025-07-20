package metrics

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/abdullahainun/tenangdb/internal/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ExporterMetrics holds the Prometheus metrics for the exporter
type ExporterMetrics struct {
	// Backup metrics
	backupDuration    *prometheus.GaugeVec
	backupSuccess     *prometheus.GaugeVec  // Changed to Gauge to allow setting exact values
	backupFailed      *prometheus.GaugeVec  // Changed to Gauge to allow setting exact values
	backupSize        *prometheus.GaugeVec
	backupTimestamp   *prometheus.GaugeVec
	
	// Upload metrics
	uploadDuration    *prometheus.GaugeVec
	uploadSuccess     *prometheus.GaugeVec  // Changed to Gauge to allow setting exact values
	uploadFailed      *prometheus.GaugeVec  // Changed to Gauge to allow setting exact values
	uploadBytes       *prometheus.GaugeVec  // Changed to Gauge to allow setting exact values
	uploadTimestamp   *prometheus.GaugeVec
	
	// Restore metrics
	restoreDuration   *prometheus.GaugeVec
	restoreSuccess    *prometheus.GaugeVec  // Changed to Gauge to allow setting exact values
	restoreFailed     *prometheus.GaugeVec  // Changed to Gauge to allow setting exact values
	restoreTimestamp  *prometheus.GaugeVec
	
	// Cleanup metrics
	cleanupDuration   prometheus.Gauge
	cleanupSuccess    prometheus.Gauge      // Changed to Gauge to allow setting exact values
	cleanupFailed     prometheus.Gauge      // Changed to Gauge to allow setting exact values
	cleanupFiles      prometheus.Gauge
	cleanupBytes      prometheus.Gauge
	cleanupTimestamp  prometheus.Gauge
	
	// System metrics
	totalDatabases    prometheus.Gauge
	processActive     prometheus.Gauge
	systemHealth      prometheus.Gauge
	lastProcessTime   prometheus.Gauge
	
	storage *MetricsStorage
}

// NewExporterMetrics creates a new ExporterMetrics instance
func NewExporterMetrics(storage *MetricsStorage) *ExporterMetrics {
	return &ExporterMetrics{
		backupDuration: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tenangdb_backup_duration_seconds",
				Help: "Duration of the last backup operation in seconds",
			},
			[]string{"database"},
		),
		backupSuccess: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tenangdb_backup_success_total",
				Help: "Total number of successful backups",
			},
			[]string{"database"},
		),
		backupFailed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tenangdb_backup_failed_total",
				Help: "Total number of failed backups",
			},
			[]string{"database"},
		),
		backupSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tenangdb_backup_size_bytes",
				Help: "Size of the last backup in bytes",
			},
			[]string{"database"},
		),
		backupTimestamp: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tenangdb_backup_last_timestamp",
				Help: "Timestamp of the last backup operation",
			},
			[]string{"database"},
		),
		uploadDuration: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tenangdb_upload_duration_seconds",
				Help: "Duration of the last upload operation in seconds",
			},
			[]string{"database"},
		),
		uploadSuccess: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tenangdb_upload_success_total",
				Help: "Total number of successful uploads",
			},
			[]string{"database"},
		),
		uploadFailed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tenangdb_upload_failed_total",
				Help: "Total number of failed uploads",
			},
			[]string{"database"},
		),
		uploadBytes: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tenangdb_upload_bytes_total",
				Help: "Total bytes uploaded",
			},
			[]string{"database"},
		),
		uploadTimestamp: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tenangdb_upload_last_timestamp",
				Help: "Timestamp of the last upload operation",
			},
			[]string{"database"},
		),
		restoreDuration: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tenangdb_restore_duration_seconds",
				Help: "Duration of the last restore operation in seconds",
			},
			[]string{"database"},
		),
		restoreSuccess: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tenangdb_restore_success_total",
				Help: "Total number of successful restores",
			},
			[]string{"database"},
		),
		restoreFailed: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tenangdb_restore_failed_total",
				Help: "Total number of failed restores",
			},
			[]string{"database"},
		),
		restoreTimestamp: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tenangdb_restore_last_timestamp",
				Help: "Timestamp of the last restore operation",
			},
			[]string{"database"},
		),
		cleanupDuration: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "tenangdb_cleanup_duration_seconds",
				Help: "Duration of the last cleanup operation in seconds",
			},
		),
		cleanupSuccess: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "tenangdb_cleanup_success_total",
				Help: "Total number of successful cleanup operations",
			},
		),
		cleanupFailed: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "tenangdb_cleanup_failed_total",
				Help: "Total number of failed cleanup operations",
			},
		),
		cleanupFiles: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "tenangdb_cleanup_files_removed_total",
				Help: "Total number of files removed by cleanup",
			},
		),
		cleanupBytes: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "tenangdb_cleanup_bytes_freed_total",
				Help: "Total bytes freed by cleanup operations",
			},
		),
		cleanupTimestamp: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "tenangdb_cleanup_last_timestamp",
				Help: "Timestamp of the last cleanup operation",
			},
		),
		totalDatabases: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "tenangdb_total_databases",
				Help: "Total number of databases configured",
			},
		),
		processActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "tenangdb_backup_process_active",
				Help: "Whether backup process is currently active (1 = active, 0 = inactive)",
			},
		),
		systemHealth: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "tenangdb_system_health",
				Help: "System health status (1 = healthy, 0 = unhealthy)",
			},
		),
		lastProcessTime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "tenangdb_last_process_timestamp",
				Help: "Timestamp of the last backup process",
			},
		),
		storage: storage,
	}
}

// Register registers all metrics with Prometheus
func (e *ExporterMetrics) Register() {
	prometheus.MustRegister(
		e.backupDuration,
		e.backupSuccess,
		e.backupFailed,
		e.backupSize,
		e.backupTimestamp,
		e.uploadDuration,
		e.uploadSuccess,
		e.uploadFailed,
		e.uploadBytes,
		e.uploadTimestamp,
		e.restoreDuration,
		e.restoreSuccess,
		e.restoreFailed,
		e.restoreTimestamp,
		e.cleanupDuration,
		e.cleanupSuccess,
		e.cleanupFailed,
		e.cleanupFiles,
		e.cleanupBytes,
		e.cleanupTimestamp,
		e.totalDatabases,
		e.processActive,
		e.systemHealth,
		e.lastProcessTime,
	)
}

// UpdateMetrics updates all metrics from storage
func (e *ExporterMetrics) UpdateMetrics() error {
	data, err := e.storage.LoadMetrics()
	if err != nil {
		return fmt.Errorf("failed to load metrics: %w", err)
	}
	
	// Update system metrics
	e.totalDatabases.Set(float64(data.System.TotalDatabases))
	if data.System.BackupProcessActive {
		e.processActive.Set(1)
	} else {
		e.processActive.Set(0)
	}
	if data.System.SystemHealthy {
		e.systemHealth.Set(1)
	} else {
		e.systemHealth.Set(0)
	}
	if !data.System.LastBackupProcess.IsZero() {
		e.lastProcessTime.Set(float64(data.System.LastBackupProcess.Unix()))
	}
	
	// Update backup metrics
	for _, backup := range data.Backups {
		e.backupDuration.WithLabelValues(backup.Database).Set(backup.DurationSeconds)
		e.backupSuccess.WithLabelValues(backup.Database).Set(float64(backup.SuccessCount))
		e.backupFailed.WithLabelValues(backup.Database).Set(float64(backup.FailureCount))
		e.backupSize.WithLabelValues(backup.Database).Set(float64(backup.SizeBytes))
		if !backup.LastBackup.IsZero() {
			e.backupTimestamp.WithLabelValues(backup.Database).Set(float64(backup.LastBackup.Unix()))
		}
	}
	
	// Update upload metrics
	for _, upload := range data.Uploads {
		e.uploadDuration.WithLabelValues(upload.Database).Set(upload.DurationSeconds)
		e.uploadSuccess.WithLabelValues(upload.Database).Set(float64(upload.SuccessCount))
		e.uploadFailed.WithLabelValues(upload.Database).Set(float64(upload.FailureCount))
		e.uploadBytes.WithLabelValues(upload.Database).Set(float64(upload.BytesUploaded))
		if !upload.LastUpload.IsZero() {
			e.uploadTimestamp.WithLabelValues(upload.Database).Set(float64(upload.LastUpload.Unix()))
		}
	}
	
	// Update restore metrics
	for _, restore := range data.Restores {
		e.restoreDuration.WithLabelValues(restore.Database).Set(restore.DurationSeconds)
		e.restoreSuccess.WithLabelValues(restore.Database).Set(float64(restore.SuccessCount))
		e.restoreFailed.WithLabelValues(restore.Database).Set(float64(restore.FailureCount))
		if !restore.LastRestore.IsZero() {
			e.restoreTimestamp.WithLabelValues(restore.Database).Set(float64(restore.LastRestore.Unix()))
		}
	}
	
	// Update cleanup metrics
	e.cleanupDuration.Set(data.Cleanup.DurationSeconds)
	e.cleanupSuccess.Set(float64(data.Cleanup.SuccessCount))
	e.cleanupFailed.Set(float64(data.Cleanup.FailureCount))
	e.cleanupFiles.Set(float64(data.Cleanup.FilesRemoved))
	e.cleanupBytes.Set(float64(data.Cleanup.BytesFreed))
	if !data.Cleanup.LastCleanup.IsZero() {
		e.cleanupTimestamp.Set(float64(data.Cleanup.LastCleanup.Unix()))
	}
	
	return nil
}

// getCurrentVersion returns version information for display
func getCurrentVersion() string {
	return "v1.1.3 (" + runtime.Version() + ")"
}

// StartMetricsExporter starts the metrics exporter HTTP server
func StartMetricsExporter(ctx context.Context, port, metricsFile string, log *logger.Logger) error {
	// Create metrics storage
	storage := NewMetricsStorage(metricsFile)
	
	// Create exporter metrics
	exporterMetrics := NewExporterMetrics(storage)
	exporterMetrics.Register()
	
	// Create HTTP server
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	
	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Try to load metrics to verify health
		_, err := storage.LoadMetrics()
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("UNHEALTHY: Cannot load metrics"))
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy","service":"tenangdb-exporter"}`))
	})
	
	// Add readiness check endpoint
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready","service":"tenangdb-exporter"}`))
	})
	
	// Add root endpoint with node exporter style dashboard
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		
		html := `<!DOCTYPE html>
<html>
<head>
    <title>TenangDB Exporter</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; line-height: 1.6; }
        h1 { color: #333; margin-bottom: 10px; }
        p { color: #666; margin-bottom: 20px; }
        ul { list-style: none; padding: 0; }
        li { margin: 8px 0; }
        a { color: #337ab7; text-decoration: none; }
        a:hover { text-decoration: underline; }
        .footer { margin-top: 40px; color: #999; font-size: 12px; }
    </style>
</head>
<body>
    <h1>TenangDB Exporter</h1>
    <p>MySQL Backup Metrics Exporter for Prometheus</p>
    
    <ul>
        <li><a href="/metrics">Metrics</a></li>
        <li><a href="/health">Health</a></li>
        <li><a href="/ready">Ready</a></li>
    </ul>
    
    <div class="footer">
        TenangDB ` + getCurrentVersion() + `
    </div>
</body>
</html>`
		
		_, _ = w.Write([]byte(html))
	})
	
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	
	// Start server in goroutine
	go func() {
		log.WithField("port", port).Info("Starting metrics HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Error("Metrics server failed")
		}
	}()
	
	// Update metrics periodically
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	// Initial metrics update
	if err := exporterMetrics.UpdateMetrics(); err != nil {
		log.WithError(err).Warn("Failed to update metrics")
	}
	
	for {
		select {
		case <-ctx.Done():
			log.Info("Shutting down metrics exporter...")
			
			// Shutdown server gracefully
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			
			if err := server.Shutdown(shutdownCtx); err != nil {
				log.WithError(err).Error("Failed to shutdown metrics server")
			}
			
			return nil
			
		case <-ticker.C:
			// Update metrics from storage
			if err := exporterMetrics.UpdateMetrics(); err != nil {
				log.WithError(err).Warn("Failed to update metrics")
			}
		}
	}
}