package main

import (
	"context"
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

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Run database backup",
		Long:  `Backup databases to local directory with optional cloud upload.`,
		Run: func(cmd *cobra.Command, args []string) {
			runBackup(configFile, logLevel, dryRun, databases, force)
		},
	}

	cmd.Flags().StringVar(&configFile, "config", "", "config file path (auto-discovery if not specified)")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be backed up without actually running backup")
	cmd.Flags().StringVar(&databases, "databases", "", "comma-separated list of databases to backup (overrides config)")
	cmd.Flags().BoolVar(&force, "force", false, "skip backup frequency confirmation prompts")

	return cmd
}

func runBackup(configFile, logLevel string, dryRun bool, databases string, force bool) {
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
	
	// Override skip confirmation if force flag is used
	if force {
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
	runBackup(configFile, logLevel, dryRun, databases, false)
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

	cmd.Flags().StringVar(&configFile, "config", "", "config file path (auto-discovery if not specified)")
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
		metricsPath := "/var/lib/tenangdb/metrics.json"
		if cfg.Metrics.StoragePath != "" {
			metricsPath = cfg.Metrics.StoragePath
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

	// Perform age-based cleanup if enabled
	if cfg.Cleanup.AgeBasedCleanup {
		cleanupService := backup.NewCleanupService(&cfg.Cleanup, &cfg.Upload, log)
		if err := cleanupService.CleanupAgeBasedFiles(ctx, cfg.Backup.Directory, selectedDatabases); err != nil {
			log.WithError(err).Error("Age-based cleanup failed")
			cleanupDuration := time.Since(cleanupStartTime)
			if cfg.Metrics.Enabled && metricsStorage != nil {
				if err := metricsStorage.UpdateCleanupMetrics(cleanupDuration, false, totalFilesRemoved, totalBytesFreed); err != nil {
					log.WithError(err).Warn("Failed to update cleanup metrics")
				}
			}
			os.Exit(1)
		}
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

	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Restore database from backup",
		Long:  `Restore a database from mydumper backup directory or SQL file.`,
		Run: func(cmd *cobra.Command, args []string) {
			runRestore(configFile, logLevel, backupPath, targetDatabase)
		},
	}

	cmd.Flags().StringVar(&configFile, "config", "", "config file path (auto-discovery if not specified)")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	cmd.Flags().StringVarP(&backupPath, "backup-path", "b", "", "path to backup directory or SQL file (required)")
	cmd.Flags().StringVarP(&targetDatabase, "database", "d", "", "target database name (required)")

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
		metricsPath := "/var/lib/tenangdb/metrics.json"
		if cfg.Metrics.StoragePath != "" {
			metricsPath = cfg.Metrics.StoragePath
		}
		metricsStorage = metrics.NewMetricsStorage(metricsPath)
	}

	log.WithField("backup_path", backupPath).WithField("target_database", targetDatabase).Info("Starting database restore")

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

	// Initialize file logger with separate formats for stdout and file
	log, err := logger.NewFileLoggerWithSeparateFormats(effectiveLogLevel, cfg.Logging.FilePath, cfg.Logging.Format, cfg.Logging.FileFormat)
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
		fmt.Printf("  - Logs: ~/Library/Logs/TenangDB/\n")
	} else {
		fmt.Printf("Linux Notes:\n")
		fmt.Printf("  - System config: /etc/tenangdb/config.yaml\n")
		fmt.Printf("  - User config: ~/.config/tenangdb/config.yaml\n")
		fmt.Printf("  - Logs: /var/log/tenangdb/\n")
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
