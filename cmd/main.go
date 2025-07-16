package main

import (
	"bufio"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
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
	version    string // Set via ldflags during build
	buildTime  string // Set via ldflags during build
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "tenangdb",
		Short: "Secure automated MySQL backup with cloud integration",
		Long:  `Secure automated MySQL backup with cloud integration and intelligent cleanup.`,
		Run:   run,
	}

	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file path (auto-discovery if not specified)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	
	// Add version flag
	var showVersionFlag bool
	rootCmd.Flags().BoolVar(&showVersionFlag, "version", false, "show version information")
	
	// Add flags for backward compatibility with default command
	var dryRun bool
	var databases string
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be backed up without actually running backup (deprecated: use 'tenangdb backup --dry-run')")
	rootCmd.Flags().StringVar(&databases, "databases", "", "comma-separated list of databases to backup (deprecated: use 'tenangdb backup --databases')")

	// Add backup subcommand (new explicit command)
	rootCmd.AddCommand(newBackupCommand())

	// Add cleanup subcommand
	rootCmd.AddCommand(newCleanupCommand())

	// Add restore subcommand
	rootCmd.AddCommand(newRestoreCommand())

	// Add exporter subcommand
	rootCmd.AddCommand(newExporterCommand())

	// Add version command
	rootCmd.AddCommand(newVersionCommand())

	// Add config command
	rootCmd.AddCommand(newConfigCommand())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newBackupCommand() *cobra.Command {
	var configFile string
	var logLevel string
	var dryRun bool
	var databases string
	var force bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Run database backup",
		Long:  `Backup databases to local directory with optional cloud upload.`,
		Run: func(cmd *cobra.Command, args []string) {
			runBackup(configFile, logLevel, dryRun, databases, force, yes)
		},
	}

	cmd.Flags().StringVar(&configFile, "config", "", "config file path (auto-discovery if not specified)")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be backed up without actually running backup")
	cmd.Flags().StringVar(&databases, "databases", "", "comma-separated list of databases to backup (overrides config)")
	cmd.Flags().BoolVar(&force, "force", false, "skip backup frequency confirmation prompts")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompts (for automated mode)")

	return cmd
}

func runBackup(configFile, logLevel string, dryRun bool, databases string, force bool, yes bool) {
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

	// Override databases from command line if specified
	if databases != "" {
		selectedDatabases := strings.Split(databases, ",")
		for i, db := range selectedDatabases {
			selectedDatabases[i] = strings.TrimSpace(db)
		}
		cfg.Backup.Databases = selectedDatabases
		log := logger.NewLogger(logLevel)
		log.Infof("Using databases from command line: %v", selectedDatabases)
	}
	
	// Override skip confirmation if force or yes flag is used
	if force || yes {
		cfg.Backup.SkipConfirmation = true
	}

	// Determine effective log level: CLI flag overrides config
	effectiveLogLevel := logLevel
	if logLevel == "info" && cfg.Logging.Level != "" {
		// If CLI uses default "info" and config has a level set, use config
		effectiveLogLevel = cfg.Logging.Level
	}

	// Initialize file logger with separate formats for stdout and file
	log, err := logger.NewFileLoggerWithSeparateFormats(effectiveLogLevel, cfg.Logging.FilePath, cfg.Logging.Format, cfg.Logging.FileFormat)
	if err != nil {
		// Fallback to stdout logger
		log = logger.NewLogger(effectiveLogLevel)
		log.WithError(err).Warn("Failed to initialize file logger, using stdout")
	}

	if dryRun {
		log.Info("DRY RUN MODE: No actual backup will be performed")
		log.WithField("databases", cfg.Backup.Databases).Info("Would backup these databases")
		log.WithField("backup_directory", cfg.Backup.Directory).Info("Backup directory")
		if cfg.Upload.Enabled {
			log.WithField("upload_destination", cfg.Upload.Destination).Info("Would upload to")
		}
		return
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

	// Check backup frequency if enabled
	if cfg.Backup.CheckLastBackupTime && !force && !checkBackupFrequency(cfg, log) {
		log.Info("Backup cancelled due to frequency check")
		return
	}

	// Show confirmation prompt if not skipped
	if !cfg.Backup.SkipConfirmation && !showBackupConfirmation(cfg, log) {
		log.Info("Backup cancelled by user")
		return
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
		
		// Update last backup time tracking
		if err := updateLastBackupTime(cfg.Backup.Directory); err != nil {
			log.WithError(err).Warn("Failed to update backup timestamp")
		}
		
		log.Info("✅ All backup process completed successfully")
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

func run(cmd *cobra.Command, args []string) {
	// Check if version flag is set
	showVersionFlag, _ := cmd.Flags().GetBool("version")
	if showVersionFlag {
		showVersion()
		return
	}
	
	// Get flags from the command
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	databases, _ := cmd.Flags().GetString("databases")
	
	// Show deprecation notice for backward compatibility
	log := logger.NewLogger(logLevel)
	log.Debug("DEPRECATED: Running tenangdb without 'backup' subcommand is deprecated. Use 'tenangdb backup' instead.")
	
	// Call the new backup function for backward compatibility
	runBackup(configFile, logLevel, dryRun, databases, false, false)
}

func newCleanupCommand() *cobra.Command {
	var configFile string
	var logLevel string
	var dryRun bool
	var force bool
	var databases string
	var yes bool

	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Cleanup uploaded backup files",
		Long:  `Remove local backup files that have been successfully uploaded to cloud storage.`,
		Run: func(cmd *cobra.Command, args []string) {
			runCleanup(configFile, logLevel, dryRun, force, databases, yes)
		},
	}

	cmd.Flags().StringVar(&configFile, "config", "", "config file path (auto-discovery if not specified)")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be deleted without actually deleting")
	cmd.Flags().BoolVar(&force, "force", false, "force cleanup regardless of day (bypass weekend-only restriction)")
	cmd.Flags().StringVar(&databases, "databases", "", "comma-separated list of databases to cleanup (overrides config)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompts (for automated mode)")

	return cmd
}

func runCleanup(configFile, logLevel string, dryRun bool, force bool, databases string, yes bool) {
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

	// Initialize file logger with separate formats for stdout and file
	log, err := logger.NewFileLoggerWithSeparateFormats(effectiveLogLevel, cfg.Logging.FilePath, cfg.Logging.Format, cfg.Logging.FileFormat)
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

	// Initialize metrics storage only if metrics are enabled
	var metricsStorage *metrics.MetricsStorage
	if cfg.Metrics.Enabled {
		metricsPath := cfg.Metrics.StoragePath
		if metricsPath == "" {
			metricsPath = "/var/lib/tenangdb/metrics.json" // fallback
		}
		metricsStorage = metrics.NewMetricsStorage(metricsPath)
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

	// Show confirmation prompt if not skipped
	if !yes && !showCleanupConfirmation(backupService, &cfg.Cleanup, cfg.Backup.Directory, selectedDatabases, log) {
		log.Info("Cleanup cancelled by user")
		return
	}

	// Record cleanup start
	cleanupStartTime := time.Now()
	var totalFilesRemoved int64
	var totalBytesFreed int64

	// Perform cleanup of uploaded files
	if err := backupService.CleanupUploadedFiles(ctx); err != nil {
		log.WithError(err).Error("Cleanup process failed")
		cleanupDuration := time.Since(cleanupStartTime)
		if cfg.Metrics.Enabled && metricsStorage != nil {
			if err := metricsStorage.UpdateCleanupMetrics(cleanupDuration, false, totalFilesRemoved, totalBytesFreed); err != nil {
				log.WithError(err).Warn("Failed to update cleanup metrics")
			}
		}
		os.Exit(1)
	}

	// Perform age-based cleanup (always enabled for cleanup command)
	maxAgeDays := cfg.Cleanup.MaxAgeDays
	if maxAgeDays == 0 {
		maxAgeDays = 7 // Safe default: 7 days
	}
	
	if err := cleanupOldBackupFiles(cfg.Backup.Directory, selectedDatabases, maxAgeDays, log); err != nil {
		log.WithError(err).Error("Age-based cleanup failed")
		cleanupDuration := time.Since(cleanupStartTime)
		if cfg.Metrics.Enabled && metricsStorage != nil {
			if err := metricsStorage.UpdateCleanupMetrics(cleanupDuration, false, totalFilesRemoved, totalBytesFreed); err != nil {
				log.WithError(err).Warn("Failed to update cleanup metrics")
			}
		}
		os.Exit(1)
	}

	// Record successful cleanup
	cleanupDuration := time.Since(cleanupStartTime)
	if cfg.Metrics.Enabled && metricsStorage != nil {
		if err := metricsStorage.UpdateCleanupMetrics(cleanupDuration, true, totalFilesRemoved, totalBytesFreed); err != nil {
			log.WithError(err).Warn("Failed to update cleanup metrics")
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
	var yes bool

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore database from backup",
		Long:  `Restore a database from mydumper backup directory or SQL file.`,
		Run: func(cmd *cobra.Command, args []string) {
			runRestore(configFile, logLevel, backupPath, targetDatabase, yes)
		},
	}

	cmd.Flags().StringVar(&configFile, "config", "", "config file path (auto-discovery if not specified)")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	cmd.Flags().StringVarP(&backupPath, "backup-path", "b", "", "path to backup directory or SQL file (required)")
	cmd.Flags().StringVarP(&targetDatabase, "database", "d", "", "target database name (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation prompts (for automated mode)")

	if err := cmd.MarkFlagRequired("backup-path"); err != nil {
		fmt.Printf("Error: Failed to mark backup-path flag as required: %v\n", err)
		os.Exit(1)
	}
	if err := cmd.MarkFlagRequired("database"); err != nil {
		fmt.Printf("Error: Failed to mark database flag as required: %v\n", err)
		os.Exit(1)
	}

	return cmd
}

func runRestore(configFile, logLevel, backupPath, targetDatabase string, yes bool) {
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

	// Initialize file logger with separate formats for stdout and file
	log, err := logger.NewFileLoggerWithSeparateFormats(effectiveLogLevel, cfg.Logging.FilePath, cfg.Logging.Format, cfg.Logging.FileFormat)
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

	// Initialize metrics storage only if metrics are enabled
	var metricsStorage *metrics.MetricsStorage
	if cfg.Metrics.Enabled {
		metricsPath := cfg.Metrics.StoragePath
		if metricsPath == "" {
			metricsPath = "/var/lib/tenangdb/metrics.json" // fallback
		}
		metricsStorage = metrics.NewMetricsStorage(metricsPath)
	}

	log.WithField("backup_path", backupPath).WithField("target_database", targetDatabase).Info("Starting database restore")

	// Show confirmation prompt if not skipped
	if !yes && !showRestoreConfirmation(backupPath, targetDatabase, dbClient, ctx, log) {
		log.Info("Database restore cancelled by user")
		return
	}

	// Record restore start
	restoreStartTime := time.Now()
	if cfg.Metrics.Enabled {
		metrics.RecordRestoreStart(targetDatabase)
	}

	// Perform restore
	err = dbClient.RestoreBackup(ctx, backupPath, targetDatabase)
	restoreDuration := time.Since(restoreStartTime)

	if err != nil {
		log.WithError(err).Error("Database restore failed")
		if cfg.Metrics.Enabled {
			metrics.RecordRestoreEnd(targetDatabase, restoreDuration, false)
			if metricsStorage != nil {
				if err := metricsStorage.UpdateRestoreMetrics(targetDatabase, restoreDuration, false); err != nil {
					log.WithError(err).Warn("Failed to update restore metrics")
				}
			}
		}
		os.Exit(1)
	}

	// Record successful restore
	if cfg.Metrics.Enabled {
		metrics.RecordRestoreEnd(targetDatabase, restoreDuration, true)
		if metricsStorage != nil {
			if err := metricsStorage.UpdateRestoreMetrics(targetDatabase, restoreDuration, true); err != nil {
				log.WithError(err).Warn("Failed to update restore metrics")
			}
		}
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

	cmd.Flags().StringVar(&configFile, "config", "", "config file path (auto-discovery if not specified)")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	cmd.Flags().StringVar(&port, "port", "9090", "HTTP server port for metrics")
	cmd.Flags().StringVar(&metricsFile, "metrics-file", "", "path to metrics storage file (auto-discovery if not specified)")

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

	// Initialize file logger with separate formats for stdout and file
	log, err := logger.NewFileLoggerWithSeparateFormats(effectiveLogLevel, cfg.Logging.FilePath, cfg.Logging.Format, cfg.Logging.FileFormat)
	if err != nil {
		// Fallback to stdout logger
		log = logger.NewLogger(effectiveLogLevel)
		log.WithError(err).Warn("Failed to initialize file logger, using stdout")
	}

	// Use config-based metrics file path if not specified
	if metricsFile == "" {
		metricsFile = cfg.Metrics.StoragePath
		if metricsFile == "" {
			metricsFile = "/var/lib/tenangdb/metrics.json" // fallback
		}
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

func newVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Display the version and build information for TenangDB.`,
		Run: func(cmd *cobra.Command, args []string) {
			showVersion()
		},
	}

	return cmd
}

func showVersion() {
	// Set default values if not provided via ldflags
	if version == "" {
		version = "unknown"
	}
	if buildTime == "" {
		buildTime = "unknown"
	}

	// Format version output
	fmt.Printf("TenangDB version %s\n", version)
	fmt.Printf("Build time: %s\n", buildTime)
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show configuration information",
		Long:  `Display configuration file locations and current platform information.`,
		Run: func(cmd *cobra.Command, args []string) {
			showConfigInfo()
		},
	}

	return cmd
}

func showConfigInfo() {
	fmt.Printf("TenangDB Configuration\n")
	fmt.Printf("======================\n\n")
	
	fmt.Printf("Platform: %s/%s\n\n", runtime.GOOS, runtime.GOARCH)
	
	// Show active config file
	if activePath, err := config.GetActiveConfigPath(); err == nil {
		fmt.Printf("Active config file: %s\n\n", activePath)
	} else {
		fmt.Printf("No config file found\n\n")
	}
	
	fmt.Printf("Config file search order (first found will be used):\n")
	configPaths := config.GetConfigPaths()
	for i, path := range configPaths {
		// Check if file exists and mark it
		expandedPath := expandPath(path)
		if _, err := os.Stat(expandedPath); err == nil {
			fmt.Printf("  %d. %s ✓ (exists)\n", i+1, path)
		} else {
			fmt.Printf("  %d. %s\n", i+1, path)
		}
	}
	
	fmt.Printf("\nUsage:\n")
	fmt.Printf("  # Auto-discovery (recommended)\n")
	fmt.Printf("  tenangdb backup\n\n")
	fmt.Printf("  # Specific config file\n")
	fmt.Printf("  tenangdb backup --config /path/to/config.yaml\n\n")
	
	if runtime.GOOS == "darwin" {
		fmt.Printf("macOS Notes:\n")
		fmt.Printf("  - System config: /usr/local/etc/tenangdb/config.yaml (Homebrew)\n")
		fmt.Printf("  - User config: ~/Library/Application Support/TenangDB/config.yaml\n")
		fmt.Printf("  - User logs: ~/Library/Logs/TenangDB/\n")
		fmt.Printf("  - User backups: ~/Library/Application Support/TenangDB/backups/\n")
		fmt.Printf("  - User metrics: ~/Library/Application Support/TenangDB/metrics.json\n")
		fmt.Printf("  - System logs: /usr/local/var/log/tenangdb/\n")
		fmt.Printf("  - System backups: /usr/local/var/tenangdb/backups/\n")
		fmt.Printf("  - System metrics: /usr/local/var/tenangdb/metrics.json\n")
	} else {
		fmt.Printf("Linux Notes:\n")
		fmt.Printf("  - System config: /etc/tenangdb/config.yaml\n")
		fmt.Printf("  - User config: ~/.config/tenangdb/config.yaml\n")
		fmt.Printf("  - User logs: ~/.local/share/tenangdb/logs/\n")
		fmt.Printf("  - User backups: ~/.local/share/tenangdb/backups/\n")
		fmt.Printf("  - User metrics: ~/.local/share/tenangdb/metrics.json\n")
		fmt.Printf("  - System logs: /var/log/tenangdb/\n")
		fmt.Printf("  - System backups: /var/backups/tenangdb/\n")
		fmt.Printf("  - System metrics: /var/lib/tenangdb/metrics.json\n")
	}
}

func expandPath(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	
	return filepath.Join(homeDir, path[2:])
}

// showBackupConfirmation displays a confirmation prompt with backup summary
func showBackupConfirmation(cfg *config.Config, _ *logger.Logger) bool {
	// Display backup summary
	fmt.Printf("\n📋 Backup Summary\n")
	fmt.Printf("================\n\n")
	
	// Database list
	fmt.Printf("💾 Databases to backup:\n")
	for i, db := range cfg.Backup.Databases {
		fmt.Printf("  %d. %s\n", i+1, db)
	}
	
	fmt.Printf("\n📁 Backup directory: %s\n", cfg.Backup.Directory)
	
	// Upload information
	if cfg.Upload.Enabled {
		fmt.Printf("☁️  Upload enabled: %s\n", cfg.Upload.Destination)
		fmt.Printf("   Rclone config: %s\n", cfg.Upload.RcloneConfigPath)
	} else {
		fmt.Printf("☁️  Upload: Disabled (local backup only)\n")
	}
	
	// Backup options
	fmt.Printf("\n⚙️  Options:\n")
	fmt.Printf("   Concurrency: %d\n", cfg.Backup.Concurrency)
	fmt.Printf("   Batch size: %d\n", cfg.Backup.BatchSize)
	
	fmt.Printf("\n")
	
	// Confirmation prompt
	fmt.Print("Do you want to proceed with backup? [y/N]: ")
	
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		return response == "y" || response == "yes"
	}
	
	return false
}

// showCleanupConfirmation displays a confirmation prompt for cleanup operation
func showCleanupConfirmation(_ *backup.Service, cleanupCfg *config.CleanupConfig, backupDir string, selectedDatabases []string, _ *logger.Logger) bool {
	fmt.Printf("\n📋 Cleanup Summary\n")
	fmt.Printf("=================\n\n")
	
	// Set safe defaults for cleanup command
	maxAgeDays := cleanupCfg.MaxAgeDays
	if maxAgeDays == 0 {
		maxAgeDays = 7 // Safe default: 7 days
	}
	
	// Get all backup files in directory
	allBackupFiles := getBackupFiles(backupDir, selectedDatabases)
	
	if len(allBackupFiles) == 0 {
		fmt.Printf("✅ No backup files found in %s\n", backupDir)
		return false
	}
	
	// Categorize files by age
	var filesToDelete []BackupFileInfo
	var totalSizeToDelete int64
	
	for _, fileInfo := range allBackupFiles {
		ageDays := int(time.Since(fileInfo.ModTime).Hours() / 24)
		
		if ageDays >= maxAgeDays {
			filesToDelete = append(filesToDelete, fileInfo)
			totalSizeToDelete += fileInfo.Size
		}
	}
	
	// Display all files with age info
	fmt.Printf("📁 Backup files found:\n")
	for i, fileInfo := range allBackupFiles {
		if i >= 15 { // Show max 15 files
			fmt.Printf("   ... and %d more files\n", len(allBackupFiles)-15)
			break
		}
		
		ageDays := int(time.Since(fileInfo.ModTime).Hours() / 24)
		status := "✅ Keep"
		if ageDays >= maxAgeDays {
			status = "⚠️  Will delete"
		}
		
		fmt.Printf("  %d. %s (%d days old, %s) %s\n", 
			i+1, fileInfo.Name, ageDays, formatFileSize(fileInfo.Size), status)
	}
	
	fmt.Printf("\n📊 Files to delete: %d (%d+ days old)\n", len(filesToDelete), maxAgeDays)
	fmt.Printf("📊 Total space to free: %s\n", formatFileSize(totalSizeToDelete))
	fmt.Printf("⏰ Age threshold: %d days (configurable)\n", maxAgeDays)
	
	if len(filesToDelete) == 0 {
		fmt.Printf("\n✅ No files old enough to cleanup (all files are < %d days old)\n", maxAgeDays)
		return false
	}
	
	fmt.Printf("\n⚠️  WARNING: This action cannot be undone!\n")
	fmt.Printf("⚠️  Deleted backup files cannot be recovered!\n\n")
	
	// Confirmation prompt
	fmt.Print("Do you want to proceed with cleanup? [y/N]: ")
	
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		return response == "y" || response == "yes"
	}
	
	return false
}

// BackupFileInfo holds information about a backup file
type BackupFileInfo struct {
	Name    string
	Path    string
	Size    int64
	ModTime time.Time
}

// getBackupFiles scans backup directory and returns backup file information
func getBackupFiles(backupDir string, selectedDatabases []string) []BackupFileInfo {
	var backupFiles []BackupFileInfo
	
	// Read backup directory
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return backupFiles
	}
	
	for _, entry := range entries {
		// Skip non-directories and non-backup files
		if !entry.IsDir() && !strings.HasSuffix(entry.Name(), ".tar.gz") && 
		   !strings.HasSuffix(entry.Name(), ".tar.zst") && 
		   !strings.HasSuffix(entry.Name(), ".tar.xz") {
			continue
		}
		
		// Check if file should be included based on database filter
		if len(selectedDatabases) > 0 && !shouldCleanupFile(entry.Name(), selectedDatabases) {
			continue
		}
		
		// Get file info
		fullPath := filepath.Join(backupDir, entry.Name())
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}
		
		// Calculate size (for directories, get total size)
		var size int64
		if info.IsDir() {
			size, _ = getDirSize(fullPath)
		} else {
			size = info.Size()
		}
		
		backupFiles = append(backupFiles, BackupFileInfo{
			Name:    entry.Name(),
			Path:    fullPath,
			Size:    size,
			ModTime: info.ModTime(),
		})
	}
	
	return backupFiles
}

// checkBackupFrequency checks if enough time has passed since last backup
func checkBackupFrequency(cfg *config.Config, log *logger.Logger) bool {
	// Get last backup time
	lastBackupTime, err := getLastBackupTime(cfg.Backup.Directory)
	if err != nil {
		// If no tracking file exists, allow backup
		log.WithError(err).Debug("No previous backup timestamp found, allowing backup")
		return true
	}
	
	// Calculate time since last backup
	timeSinceLastBackup := time.Since(lastBackupTime)
	
	// Check if enough time has passed
	if timeSinceLastBackup < cfg.Backup.MinBackupInterval {
		// Show frequency warning
		fmt.Printf("\n⚠️  Backup Frequency Warning\n")
		fmt.Printf("==========================\n\n")
		fmt.Printf("📅 Last backup: %s\n", lastBackupTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("🕐 Time since last backup: %s\n", formatDuration(timeSinceLastBackup))
		fmt.Printf("⏰ Minimum interval: %s\n", formatDuration(cfg.Backup.MinBackupInterval))
		
		remaining := cfg.Backup.MinBackupInterval - timeSinceLastBackup
		fmt.Printf("⏳ Time remaining: %s\n", formatDuration(remaining))
		
		fmt.Printf("\n⚠️  Last backup was %s ago, are you sure you want to run backup again?\n", formatDuration(timeSinceLastBackup))
		fmt.Printf("💡 Use --force to skip this check\n\n")
		
		fmt.Print("Continue backup? (y/n/force): ")
		
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			response := strings.ToLower(strings.TrimSpace(scanner.Text()))
			return response == "y" || response == "yes" || response == "force" || response == "f"
		}
		
		return false
	}
	
	return true
}

// getLastBackupTime reads the last backup timestamp from tracking file
func getLastBackupTime(backupDir string) (time.Time, error) {
	trackingFile := getTrackingFilePath(backupDir)
	
	data, err := os.ReadFile(trackingFile)
	if err != nil {
		return time.Time{}, err
	}
	
	var tracking struct {
		LastBackupTime time.Time `json:"last_backup_time"`
	}
	
	if err := json.Unmarshal(data, &tracking); err != nil {
		return time.Time{}, err
	}
	
	return tracking.LastBackupTime, nil
}

// updateLastBackupTime updates the last backup timestamp in tracking file
func updateLastBackupTime(backupDir string) error {
	trackingFile := getTrackingFilePath(backupDir)
	
	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(trackingFile), 0755); err != nil {
		return fmt.Errorf("failed to create tracking directory: %w", err)
	}
	
	tracking := struct {
		LastBackupTime time.Time `json:"last_backup_time"`
	}{
		LastBackupTime: time.Now(),
	}
	
	data, err := json.Marshal(tracking)
	if err != nil {
		return err
	}
	
	return os.WriteFile(trackingFile, data, 0644)
}

// getTrackingFilePath returns the path for backup tracking file
// Uses a more persistent location to survive container restarts
func getTrackingFilePath(backupDir string) string {
	// Try to use a more persistent location based on platform
	var trackingDir string
	
	if runtime.GOOS == "darwin" {
		// macOS: Use Application Support directory
		if homeDir, err := os.UserHomeDir(); err == nil {
			trackingDir = filepath.Join(homeDir, "Library", "Application Support", "TenangDB")
		}
	} else {
		// Linux: Use /tmp for Docker containers, XDG for regular usage
		if _, err := os.Stat("/.dockerenv"); err == nil {
			// Running in Docker container - use /tmp which is more likely to be persistent
			trackingDir = "/tmp/tenangdb"
		} else if homeDir, err := os.UserHomeDir(); err == nil {
			// Regular Linux usage
			trackingDir = filepath.Join(homeDir, ".local", "share", "tenangdb")
		}
	}
	
	// Fallback to backup directory if we can't determine a better location
	if trackingDir == "" {
		trackingDir = backupDir
	}
	
	// Create a safe filename based on backup directory path
	// This allows multiple backup configs to have separate tracking files
	hash := md5.Sum([]byte(backupDir))
	hasher := fmt.Sprintf("%x", hash)[:8]
	
	trackingFile := fmt.Sprintf(".tenangdb_backup_tracking_%s.json", hasher)
	return filepath.Join(trackingDir, trackingFile)
}

// formatDuration formats duration in human readable format
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		if minutes == 0 {
			return fmt.Sprintf("%d hours", hours)
		}
		return fmt.Sprintf("%d hours %d minutes", hours, minutes)
	} else {
		days := int(d.Hours()) / 24
		hours := int(d.Hours()) % 24
		if hours == 0 {
			return fmt.Sprintf("%d days", days)
		}
		return fmt.Sprintf("%d days %d hours", days, hours)
	}
}

// cleanupOldBackupFiles removes backup files older than specified days
func cleanupOldBackupFiles(backupDir string, selectedDatabases []string, maxAgeDays int, log *logger.Logger) error {
	// Get all backup files
	allBackupFiles := getBackupFiles(backupDir, selectedDatabases)
	
	var filesToDelete []BackupFileInfo
	for _, fileInfo := range allBackupFiles {
		ageDays := int(time.Since(fileInfo.ModTime).Hours() / 24)
		if ageDays >= maxAgeDays {
			filesToDelete = append(filesToDelete, fileInfo)
		}
	}
	
	// Delete old files
	for _, fileInfo := range filesToDelete {
		log.WithField("file", fileInfo.Name).
			WithField("age_days", int(time.Since(fileInfo.ModTime).Hours()/24)).
			Info("🗑️ Deleting old backup file")
		
		if err := os.RemoveAll(fileInfo.Path); err != nil {
			log.WithError(err).WithField("file", fileInfo.Path).Error("Failed to delete backup file")
			return fmt.Errorf("failed to delete %s: %w", fileInfo.Path, err)
		}
	}
	
	log.WithField("deleted_files", len(filesToDelete)).Info("✅ Age-based cleanup completed")
	return nil
}

// formatFileSize formats file size in human readable format
func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	
	if size >= GB {
		return fmt.Sprintf("%.1f GB", float64(size)/GB)
	} else if size >= MB {
		return fmt.Sprintf("%.1f MB", float64(size)/MB)
	} else if size >= KB {
		return fmt.Sprintf("%.1f KB", float64(size)/KB)
	}
	
	return fmt.Sprintf("%d bytes", size)
}

// showRestoreConfirmation displays a confirmation prompt for restore operation
func showRestoreConfirmation(backupPath, targetDatabase string, dbClient *database.Client, ctx context.Context, log *logger.Logger) bool {
	fmt.Printf("\n⚠️  Database Restore Warning\n")
	fmt.Printf("===========================\n\n")
	
	// Display restore details
	fmt.Printf("🎯 Target database: %s\n", targetDatabase)
	fmt.Printf("📂 Backup source: %s\n", backupPath)
	
	// Get backup info
	if info, err := os.Stat(backupPath); err == nil {
		fmt.Printf("📅 Backup date: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
		
		// Show backup size
		if info.IsDir() {
			if size, err := getDirSize(backupPath); err == nil {
				fmt.Printf("📊 Backup size: %s\n", formatFileSize(size))
			}
		} else {
			fmt.Printf("📊 Backup size: %s\n", formatFileSize(info.Size()))
		}
	}
	
	// Check if target database exists
	databaseExists, err := checkDatabaseExists(dbClient, ctx, targetDatabase)
	if err != nil {
		log.WithError(err).Warn("Failed to check if database exists")
		databaseExists = false
	}
	
	fmt.Printf("\n")
	
	if databaseExists {
		fmt.Printf("🔴 **DANGER ZONE** 🔴\n")
		fmt.Printf("⚠️  WARNING: Database '%s' already exists!\n", targetDatabase)
		fmt.Printf("⚠️  This operation will COMPLETELY OVERWRITE the existing database!\n")
		fmt.Printf("⚠️  ALL existing data in '%s' will be PERMANENTLY LOST!\n", targetDatabase)
		fmt.Printf("⚠️  This action CANNOT be undone!\n")
		fmt.Printf("\n")
		fmt.Printf("💡 Recommendation: Create a backup of the existing database first\n")
		fmt.Printf("   tenangdb backup --databases %s\n", targetDatabase)
	} else {
		fmt.Printf("✅ Database '%s' does not exist - will be created\n", targetDatabase)
	}
	
	fmt.Printf("\n")
	
	// Different confirmation message based on whether database exists
	var prompt string
	if databaseExists {
		prompt = fmt.Sprintf("Are you ABSOLUTELY SURE you want to OVERWRITE database '%s'? [y/N]: ", targetDatabase)
	} else {
		prompt = fmt.Sprintf("Do you want to create and restore database '%s'? [y/N]: ", targetDatabase)
	}
	
	fmt.Print(prompt)
	
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		return response == "y" || response == "yes"
	}
	
	return false
}

// checkDatabaseExists checks if a database exists
func checkDatabaseExists(dbClient *database.Client, ctx context.Context, databaseName string) (bool, error) {
	databases, err := dbClient.ListDatabases(ctx)
	if err != nil {
		return false, err
	}
	
	for _, db := range databases {
		if db == databaseName {
			return true, nil
		}
	}
	
	return false, nil
}

// getDirSize calculates the total size of a directory
func getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}
