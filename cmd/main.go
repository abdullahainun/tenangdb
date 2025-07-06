package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/abdullahainun/tenangdb/internal/backup"
	"github.com/abdullahainun/tenangdb/internal/config"
	"github.com/abdullahainun/tenangdb/internal/logger"
	"github.com/abdullahainun/tenangdb/internal/metrics"
	"github.com/abdullahainun/tenangdb/pkg/database"

	"github.com/spf13/cobra"
)

var (
	configFile string
	logLevel   string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "tenangdb",
		Short: "A robust database backup tool with batch processing and cloud upload",
		Long:  `A Go-based database backup tool that supports batch processing, cloud uploads via rclone, and graceful error handling.`,
		Run:   run,
	}

	rootCmd.PersistentFlags().StringVar(&configFile, "config", "configs/config.yaml", "config file path")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")

	// Add cleanup subcommand
	rootCmd.AddCommand(newCleanupCommand())

	// Add restore subcommand
	rootCmd.AddCommand(newRestoreCommand())

	// Add exporter subcommand
	rootCmd.AddCommand(newExporterCommand())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Load configuration first to get log file path
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		// Use basic logger if config fails
		log := logger.NewLogger(logLevel)
		log.WithError(err).Fatal("Failed to load configuration")
	}

	// Determine effective log level: CLI flag overrides config
	effectiveLogLevel := logLevel
	if logLevel == "info" && cfg.Logging.Level != "" {
		// If CLI uses default "info" and config has a level set, use config
		effectiveLogLevel = cfg.Logging.Level
	}

	// Initialize file logger with effective log level
	log, err := logger.NewFileLogger(effectiveLogLevel, cfg.Logging.FilePath)
	if err != nil {
		// Fallback to stdout logger
		log = logger.NewLogger(effectiveLogLevel)
		log.WithError(err).Warn("Failed to initialize file logger, using stdout")
	}

	// Initialize Prometheus metrics if enabled
	if cfg.Metrics.Enabled {
		metrics.Init()
		go func() {
			log.WithField("port", cfg.Metrics.Port).Info("Starting Prometheus metrics server")
			if err := metrics.StartMetricsServer(cfg.Metrics.Port); err != nil {
				log.WithError(err).Error("Failed to start metrics server")
			}
		}()
	}

	// Initialize backup service
	backupService, err := backup.NewService(cfg, log)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize backup service")
	}

	// Start backup process
	done := make(chan error, 1)
	go func() {
		done <- backupService.Run(ctx)
	}()

	// Wait for backup completion or shutdown signal
	select {
	case err := <-done:
		if err != nil {
			log.WithError(err).Error("Backup process failed")
			os.Exit(1)
		}
		log.Info("Backup process completed successfully")
	case <-sigChan:
		log.Info("Received shutdown signal, gracefully shutting down...")
		cancel()
		// Wait for backup to finish gracefully
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			log.Warn("Backup process did not finish within 30 seconds, forcing exit")
		}
	}
}

func newCleanupCommand() *cobra.Command {
	var configFile string
	var logLevel string
	var dryRun bool
	var force bool
	var databases string

	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Cleanup uploaded backup files",
		Long:  `Remove local backup files that have been successfully uploaded to cloud storage.`,
		Run: func(cmd *cobra.Command, args []string) {
			runCleanup(configFile, logLevel, dryRun, force, databases)
		},
	}

	cmd.Flags().StringVar(&configFile, "config", "configs/config.yaml", "config file path")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be deleted without actually deleting")
	cmd.Flags().BoolVar(&force, "force", false, "force cleanup regardless of day (bypass weekend-only restriction)")
	cmd.Flags().StringVar(&databases, "databases", "", "comma-separated list of databases to cleanup (overrides config)")

	return cmd
}

func runCleanup(configFile, logLevel string, dryRun bool, force bool, databases string) {
	ctx := context.Background()

	// Load configuration first to get log file path
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		// Use basic logger if config fails
		log := logger.NewLogger(logLevel)
		log.WithError(err).Fatal("Failed to load configuration")
	}

	// Determine effective log level: CLI flag overrides config
	effectiveLogLevel := logLevel
	if logLevel == "info" && cfg.Logging.Level != "" {
		// If CLI uses default "info" and config has a level set, use config
		effectiveLogLevel = cfg.Logging.Level
	}

	// Initialize file logger with effective log level
	log, err := logger.NewFileLogger(effectiveLogLevel, cfg.Logging.FilePath)
	if err != nil {
		// Fallback to stdout logger
		log = logger.NewLogger(effectiveLogLevel)
		log.WithError(err).Warn("Failed to initialize file logger, using stdout")
	}

	// Check if today is weekend (Saturday or Sunday) unless force flag is used
	if !force {
		today := time.Now().Weekday()
		if today != time.Saturday && today != time.Sunday {
			log.Info("Cleanup only runs on weekends. Use --force to cleanup anytime. Skipping cleanup.")
			return
		}
	}

	if force {
		log.Info("Starting forced cleanup process")
	} else {
		log.Info("Starting weekend cleanup process")
	}

	// Parse databases from command line and merge with config
	var selectedDatabases []string
	if databases != "" {
		// Command line overrides config
		selectedDatabases = strings.Split(databases, ",")
		for i, db := range selectedDatabases {
			selectedDatabases[i] = strings.TrimSpace(db)
		}
		log.Infof("Using databases from command line: %v", selectedDatabases)
	} else if len(cfg.Cleanup.Databases) > 0 {
		// Use config databases
		selectedDatabases = cfg.Cleanup.Databases
		log.Infof("Using databases from config: %v", selectedDatabases)
	} else {
		// No filter, cleanup all databases
		log.Info("No database filter specified, cleaning up all databases")
	}

	// Initialize backup service to access uploaded files tracking
	backupService, err := backup.NewService(cfg, log)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize backup service")
	}

	if dryRun {
		log.Info("DRY RUN MODE: No files will be actually deleted")
		showFilesToCleanup(backupService, log)
		
		// Show age-based cleanup files if enabled
		if cfg.Cleanup.AgeBasedCleanup {
			cleanupService := backup.NewCleanupService(&cfg.Cleanup, &cfg.Upload, log)
			showAgeBasedFilesToCleanup(cleanupService, cfg.Backup.Directory, selectedDatabases, log)
		}
		return
	}

	// Perform cleanup of uploaded files
	if err := backupService.CleanupUploadedFiles(ctx); err != nil {
		log.WithError(err).Error("Cleanup process failed")
		os.Exit(1)
	}

	// Perform age-based cleanup if enabled
	if cfg.Cleanup.AgeBasedCleanup {
		cleanupService := backup.NewCleanupService(&cfg.Cleanup, &cfg.Upload, log)
		if err := cleanupService.CleanupAgeBasedFiles(ctx, cfg.Backup.Directory, selectedDatabases); err != nil {
			log.WithError(err).Error("Age-based cleanup failed")
			os.Exit(1)
		}
	}

	if force {
		log.Info("Forced cleanup completed successfully")
	} else {
		log.Info("Weekend cleanup completed successfully")
	}
}

func showFilesToCleanup(service *backup.Service, log *logger.Logger) {
	uploadedFiles := service.GetUploadedFiles()

	if len(uploadedFiles) == 0 {
		log.Info("No uploaded files to cleanup")
		return
	}

	var filesToClean []string
	for filePath, uploadTime := range uploadedFiles {
		if time.Since(uploadTime) >= time.Hour {
			filesToClean = append(filesToClean, filePath)
		}
	}

	log.WithField("files_to_cleanup", len(filesToClean)).Info("Files that would be cleaned up:")
	for _, file := range filesToClean {
		log.WithField("file", file).Info("Would delete")
	}
}

func showAgeBasedFilesToCleanup(cleanupService *backup.CleanupService, backupDir string, selectedDatabases []string, log *logger.Logger) {
	// Get old files based on age
	oldFiles, err := cleanupService.GetOldFiles(backupDir, cleanupService.GetConfig().MaxAgeDays)
	if err != nil {
		log.WithError(err).Error("Failed to get old files for age-based cleanup")
		return
	}

	// Filter by selected databases if specified
	if len(selectedDatabases) > 0 {
		filteredFiles := []string{}
		for _, file := range oldFiles {
			if shouldCleanupFile(file, selectedDatabases) {
				filteredFiles = append(filteredFiles, file)
			}
		}
		oldFiles = filteredFiles
	}

	if len(oldFiles) == 0 {
		log.Info("No old files found for age-based cleanup")
		return
	}

	log.WithField("old_files_count", len(oldFiles)).Info("Age-based files that would be cleaned up:")
	for _, file := range oldFiles {
		log.WithField("file", file).Info("Would delete (age-based)")
	}
}

// shouldCleanupFile checks if a file should be cleaned up based on database filter
func shouldCleanupFile(filePath string, selectedDatabases []string) bool {
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

func newRestoreCommand() *cobra.Command {
	var configFile string
	var logLevel string
	var backupPath string
	var targetDatabase string

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore database from backup",
		Long:  `Restore a database from mydumper backup directory or SQL file.`,
		Run: func(cmd *cobra.Command, args []string) {
			runRestore(configFile, logLevel, backupPath, targetDatabase)
		},
	}

	cmd.Flags().StringVar(&configFile, "config", "configs/config.yaml", "config file path")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	cmd.Flags().StringVarP(&backupPath, "backup-path", "b", "", "path to backup directory or SQL file (required)")
	cmd.Flags().StringVarP(&targetDatabase, "database", "d", "", "target database name (required)")

	cmd.MarkFlagRequired("backup-path")
	cmd.MarkFlagRequired("database")

	return cmd
}

func runRestore(configFile, logLevel, backupPath, targetDatabase string) {
	ctx := context.Background()

	// Load configuration first to get log file path
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		// Use basic logger if config fails
		log := logger.NewLogger(logLevel)
		log.WithError(err).Fatal("Failed to load configuration")
	}

	// Determine effective log level: CLI flag overrides config
	effectiveLogLevel := logLevel
	if logLevel == "info" && cfg.Logging.Level != "" {
		// If CLI uses default "info" and config has a level set, use config
		effectiveLogLevel = cfg.Logging.Level
	}

	// Initialize file logger with effective log level
	log, err := logger.NewFileLogger(effectiveLogLevel, cfg.Logging.FilePath)
	if err != nil {
		// Fallback to stdout logger
		log = logger.NewLogger(effectiveLogLevel)
		log.WithError(err).Warn("Failed to initialize file logger, using stdout")
	}

	// Initialize database client
	dbClient, err := database.NewClient(&cfg.Database)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize database client")
	}
	defer dbClient.Close()

	log.WithField("backup_path", backupPath).WithField("target_database", targetDatabase).Info("Starting database restore")

	// Perform restore
	if err := dbClient.RestoreBackup(ctx, backupPath, targetDatabase); err != nil {
		log.WithError(err).Fatal("Database restore failed")
	}

	log.WithField("target_database", targetDatabase).Info("Database restore completed successfully")
}

func newExporterCommand() *cobra.Command {
	var configFile string
	var logLevel string
	var port string
	var metricsFile string

	cmd := &cobra.Command{
		Use:   "exporter",
		Short: "Start Prometheus metrics exporter",
		Long:  `Start HTTP server to expose tenangdb metrics for Prometheus scraping.`,
		Run: func(cmd *cobra.Command, args []string) {
			runExporter(configFile, logLevel, port, metricsFile)
		},
	}

	cmd.Flags().StringVar(&configFile, "config", "configs/config.yaml", "config file path")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	cmd.Flags().StringVar(&port, "port", "9090", "HTTP server port for metrics")
	cmd.Flags().StringVar(&metricsFile, "metrics-file", "/var/lib/tenangdb/metrics.json", "path to metrics storage file")

	return cmd
}

func runExporter(configFile, logLevel, port, metricsFile string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Load configuration first to get log file path
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		// Use basic logger if config fails
		log := logger.NewLogger(logLevel)
		log.WithError(err).Fatal("Failed to load configuration")
	}

	// Determine effective log level: CLI flag overrides config
	effectiveLogLevel := logLevel
	if logLevel == "info" && cfg.Logging.Level != "" {
		// If CLI uses default "info" and config has a level set, use config
		effectiveLogLevel = cfg.Logging.Level
	}

	// Initialize file logger with effective log level
	log, err := logger.NewFileLogger(effectiveLogLevel, cfg.Logging.FilePath)
	if err != nil {
		// Fallback to stdout logger
		log = logger.NewLogger(effectiveLogLevel)
		log.WithError(err).Warn("Failed to initialize file logger, using stdout")
	}

	log.WithField("port", port).WithField("metrics_file", metricsFile).Info("Starting tenangdb metrics exporter")

	// Start metrics exporter
	done := make(chan error, 1)
	go func() {
		done <- startMetricsExporter(ctx, port, metricsFile, log)
	}()

	// Wait for shutdown signal
	select {
	case err := <-done:
		if err != nil {
			log.WithError(err).Error("Metrics exporter failed")
			os.Exit(1)
		}
	case <-sigChan:
		log.Info("Received shutdown signal, gracefully shutting down...")
		cancel()
		// Wait for exporter to finish gracefully
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			log.Warn("Metrics exporter did not finish within 10 seconds, forcing exit")
		}
	}

	log.Info("Metrics exporter stopped")
}

func startMetricsExporter(ctx context.Context, port, metricsFile string, log *logger.Logger) error {
	return metrics.StartMetricsExporter(ctx, port, metricsFile, log)
}
