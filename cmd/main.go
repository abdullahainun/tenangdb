package main

import (
	"bufio"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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


	// Add version command
	rootCmd.AddCommand(newVersionCommand())

	// Add config command
	rootCmd.AddCommand(newConfigCommand())

	// Add init command
	rootCmd.AddCommand(newInitCommand())

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

func newInitCommand() *cobra.Command {
	var configPath string
	var force bool
	var deploySystemd bool
	var systemdUser string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize TenangDB configuration",
		Long:  `Interactive wizard to set up TenangDB configuration, create directories, and validate dependencies.`,
		Run: func(cmd *cobra.Command, args []string) {
			runInit(configPath, force, deploySystemd, systemdUser)
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path (auto-discovery if not specified)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing config file without confirmation")
	cmd.Flags().BoolVar(&deploySystemd, "deploy-systemd", false, "automatically deploy as systemd service")
	cmd.Flags().StringVar(&systemdUser, "systemd-user", "tenangdb", "systemd service user")

	return cmd
}

func runInit(configPath string, force bool, deploySystemd bool, systemdUser string) {
	fmt.Printf("\n🛡️ TenangDB Setup Wizard\n")
	fmt.Printf("========================\n\n")
	fmt.Printf("This wizard will help you set up TenangDB with your MySQL database.\n\n")

	// Determine config file path
	targetConfigPath := configPath
	if targetConfigPath == "" {
		// For init command, prioritize user-writable paths when not running as root
		configPaths := config.GetConfigPaths()
		if os.Geteuid() != 0 {
			// Not running as root, find first user-writable path
			for _, path := range configPaths {
				expandedPath := expandPath(path)
				// Check if we can write to the directory
				dir := filepath.Dir(expandedPath)
				if err := os.MkdirAll(dir, 0755); err == nil {
					// Test write permission
					testFile := filepath.Join(dir, ".tenangdb_write_test")
					if err := os.WriteFile(testFile, []byte("test"), 0644); err == nil {
						os.Remove(testFile) // Clean up test file
						targetConfigPath = expandedPath
						break
					}
				}
			}
			// If no writable path found, use user config as fallback
			if targetConfigPath == "" {
				if runtime.GOOS == "darwin" {
					homeDir, _ := os.UserHomeDir()
					targetConfigPath = filepath.Join(homeDir, "Library", "Application Support", "TenangDB", "config.yaml")
				} else {
					homeDir, _ := os.UserHomeDir()
					targetConfigPath = filepath.Join(homeDir, ".config", "tenangdb", "config.yaml")
				}
			}
		} else {
			// Running as root, use system-wide path
			targetConfigPath = expandPath(configPaths[0])
		}
	}

	// Check if config already exists
	if _, err := os.Stat(targetConfigPath); err == nil && !force {
		fmt.Printf("⚠️  Config file already exists: %s\n", targetConfigPath)
		fmt.Print("Do you want to overwrite it? [y/N]: ")
		
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			response := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if response != "y" && response != "yes" {
				fmt.Println("Setup cancelled.")
				return
			}
		} else {
			fmt.Println("Setup cancelled.")
			return
		}
	}

	fmt.Printf("📁 Config will be saved to: %s\n", targetConfigPath)
	if os.Geteuid() != 0 && deploySystemd {
		fmt.Printf("💡 Note: Run with 'sudo' to deploy systemd services system-wide\n")
	}
	fmt.Printf("\n")

	// Step 1: Validate dependencies
	fmt.Printf("🔍 Step 1: Checking dependencies...\n")
	deps := validateDependencies()
	
	// Step 2: Database configuration
	fmt.Printf("\n💾 Step 2: Database Configuration\n")
	fmt.Printf("=================================\n")
	dbConfig := setupDatabaseConfig()

	// Step 3: Test database connection
	fmt.Printf("\n🔗 Step 3: Testing database connection...\n")
	if !testDatabaseConnection(dbConfig) {
		fmt.Printf("❌ Database connection failed. Please check your settings and try again.\n")
		return
	}
	fmt.Printf("✅ Database connection successful!\n")

	// Step 4: Backup configuration
	fmt.Printf("\n📦 Step 4: Backup Configuration\n")
	fmt.Printf("===============================\n")
	backupConfig := setupBackupConfig(dbConfig)

	// Step 5: Upload configuration (optional)
	fmt.Printf("\n☁️ Step 5: Cloud Upload (Optional)\n")
	fmt.Printf("==================================\n")
	uploadConfig := setupUploadConfig(deps.rcloneAvailable)

	// Step 6: Logging and metrics
	fmt.Printf("\n📊 Step 6: Logging & Metrics\n")
	fmt.Printf("============================\n")
	loggingConfig, metricsConfig := setupLoggingAndMetrics()

	// Step 7: Generate and save config
	fmt.Printf("\n💾 Step 7: Generating configuration...\n")
	fullConfig := generateConfig(dbConfig, backupConfig, uploadConfig, loggingConfig, metricsConfig)
	
	if err := saveConfig(fullConfig, targetConfigPath); err != nil {
		fmt.Printf("❌ Failed to save config: %v\n", err)
		return
	}

	// Step 8: Create directories
	fmt.Printf("\n📁 Step 8: Creating directories...\n")
	createDirectories(backupConfig.Directory, loggingConfig.FilePath, metricsConfig.StoragePath)

	// Step 9: Systemd deployment (optional)
	if deploySystemd || (!deploySystemd && promptSystemdDeployment()) {
		fmt.Printf("\n🚀 Step 9: Deploying as systemd service...\n")
		if os.Geteuid() != 0 {
			fmt.Printf("❌ Systemd deployment requires root privileges\n")
			fmt.Printf("💡 Please run: sudo tenangdb init --deploy-systemd --config %s --force\n", targetConfigPath)
		} else {
			if err := deployAsSystemdService(targetConfigPath, systemdUser, metricsConfig.Port); err != nil {
				fmt.Printf("❌ Failed to deploy systemd service: %v\n", err)
				fmt.Printf("💡 You can deploy manually later using the script in scripts/install.sh\n")
			} else {
				fmt.Printf("✅ Systemd service deployed successfully!\n")
			}
		}
	}

	// Summary
	fmt.Printf("\n🎉 Setup Complete!\n")
	fmt.Printf("==================\n\n")
	fmt.Printf("✅ Configuration saved: %s\n", targetConfigPath)
	fmt.Printf("✅ Directories created\n")
	fmt.Printf("✅ Dependencies validated\n")
	if deploySystemd {
		fmt.Printf("✅ Systemd service deployed\n")
	}
	fmt.Printf("\n")
	
	fmt.Printf("🚀 Next steps:\n")
	if deploySystemd {
		fmt.Printf("  1. Check service status: sudo systemctl status tenangdb.timer\n")
		fmt.Printf("  2. View logs: sudo journalctl -u tenangdb.service -f\n")
		fmt.Printf("  3. Manual backup: sudo systemctl start tenangdb.service\n")
		if metricsConfig.Enabled {
			fmt.Printf("  4. View metrics: curl http://localhost:%s/metrics\n", metricsConfig.Port)
		}
	} else {
		fmt.Printf("  1. Run your first backup: tenangdb backup\n")
		if uploadConfig.Enabled {
			fmt.Printf("  2. Check cloud upload: rclone ls %s\n", uploadConfig.Destination)
		}
		if metricsConfig.Enabled {
			fmt.Printf("  3. View metrics: http://localhost:%s/metrics\n", metricsConfig.Port)
		}
		fmt.Printf("  4. Deploy as service: tenangdb init --deploy-systemd --force\n")
	}
	fmt.Printf("\n📚 Need help? Check: tenangdb --help\n\n")
}

type DependencyStatus struct {
	mysqldumpAvailable bool
	mysqlAvailable     bool
	mydumperAvailable  bool
	myloaderAvailable  bool
	rcloneAvailable    bool
}

func validateDependencies() DependencyStatus {
	deps := DependencyStatus{}
	
	// Check mysqldump
	if path := config.FindMysqldumpPath(); path != "" {
		if _, err := os.Stat(path); err == nil {
			deps.mysqldumpAvailable = true
			fmt.Printf("✅ mysqldump found: %s\n", path)
		}
	}
	if !deps.mysqldumpAvailable {
		fmt.Printf("❌ mysqldump not found (required for backup)\n")
	}

	// Check mysql
	if path := config.FindMysqlPath(); path != "" {
		if _, err := os.Stat(path); err == nil {
			deps.mysqlAvailable = true
			fmt.Printf("✅ mysql found: %s\n", path)
		}
	}
	if !deps.mysqlAvailable {
		fmt.Printf("⚠️  mysql client not found (required for restore)\n")
	}

	// Check mydumper (optional)
	if path := config.FindMydumperPath(); path != "" {
		if _, err := os.Stat(path); err == nil {
			deps.mydumperAvailable = true
			fmt.Printf("✅ mydumper found: %s (faster parallel backups)\n", path)
		}
	}
	if !deps.mydumperAvailable {
		fmt.Printf("⚠️  mydumper not found (optional, enables faster parallel backups)\n")
	}

	// Check myloader (optional)
	if path := config.FindMyloaderPath(); path != "" {
		if _, err := os.Stat(path); err == nil {
			deps.myloaderAvailable = true
			fmt.Printf("✅ myloader found: %s (faster parallel restores)\n", path)
		}
	}
	if !deps.myloaderAvailable && deps.mydumperAvailable {
		fmt.Printf("⚠️  myloader not found (optional, enables faster parallel restores)\n")
	}

	// Check rclone (optional)
	if path := config.FindRclonePath(); path != "" {
		if _, err := os.Stat(path); err == nil {
			deps.rcloneAvailable = true
			fmt.Printf("✅ rclone found: %s (cloud upload)\n", path)
		}
	}
	if !deps.rcloneAvailable {
		fmt.Printf("⚠️  rclone not found (optional, enables cloud upload)\n")
	}

	return deps
}

func setupDatabaseConfig() config.DatabaseConfig {
	scanner := bufio.NewScanner(os.Stdin)
	
	// Database host
	fmt.Print("Database host [localhost]: ")
	host := "localhost"
	if scanner.Scan() {
		if input := strings.TrimSpace(scanner.Text()); input != "" {
			host = input
		}
	}

	// Database port
	fmt.Print("Database port [3306]: ")
	port := 3306
	if scanner.Scan() {
		if input := strings.TrimSpace(scanner.Text()); input != "" {
			if p, err := fmt.Sscanf(input, "%d", &port); p != 1 || err != nil {
				fmt.Printf("Invalid port, using default: 3306\n")
				port = 3306
			}
		}
	}

	// Database username
	fmt.Print("Database username: ")
	var username string
	if scanner.Scan() {
		username = strings.TrimSpace(scanner.Text())
	}
	for username == "" {
		fmt.Print("Username is required. Database username: ")
		if scanner.Scan() {
			username = strings.TrimSpace(scanner.Text())
		}
	}

	// Database password
	fmt.Print("Database password: ")
	var password string
	if scanner.Scan() {
		password = scanner.Text() // Don't trim password, preserve spaces
	}

	return config.DatabaseConfig{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		Timeout:  30,
	}
}

func testDatabaseConnection(dbConfig config.DatabaseConfig) bool {
	// Create a minimal config for testing
	testConfig := &config.Config{
		Database: dbConfig,
	}
	
	dbClient, err := database.NewClient(&testConfig.Database)
	if err != nil {
		fmt.Printf("❌ Failed to create database client: %v\n", err)
		return false
	}
	defer dbClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test connection by listing databases
	databases, err := dbClient.ListDatabases(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to connect: %v\n", err)
		return false
	}

	fmt.Printf("✅ Found %d databases: %v\n", len(databases), databases)
	return true
}

func setupBackupConfig(dbConfig config.DatabaseConfig) config.BackupConfig {
	scanner := bufio.NewScanner(os.Stdin)
	
	// Get available databases for selection
	fmt.Printf("Getting list of available databases...\n")
	testConfig := &config.Config{Database: dbConfig}
	dbClient, err := database.NewClient(&testConfig.Database)
	if err != nil {
		fmt.Printf("❌ Could not connect to database: %v\n", err)
		fmt.Printf("You'll need to manually specify databases.\n")
	}
	
	var availableDatabases []string
	if dbClient != nil {
		defer dbClient.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		if dbs, err := dbClient.ListDatabases(ctx); err == nil {
			availableDatabases = dbs
		}
	}

	// Show available databases
	if len(availableDatabases) > 0 {
		fmt.Printf("\nAvailable databases:\n")
		for i, db := range availableDatabases {
			// Skip system databases by default
			if db == "information_schema" || db == "performance_schema" || db == "mysql" || db == "sys" {
				fmt.Printf("  %d. %s (system database)\n", i+1, db)
			} else {
				fmt.Printf("  %d. %s\n", i+1, db)
			}
		}
	}

	// Database selection
	fmt.Printf("\nWhich databases do you want to backup?\n")
	fmt.Printf("Enter database names separated by commas, or numbers from the list above.\n")
	fmt.Print("Databases to backup: ")
	
	var selectedDatabases []string
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			// Parse input - could be database names or numbers
			parts := strings.Split(input, ",")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				
				// Check if it's a number
				var num int
				if _, err := fmt.Sscanf(part, "%d", &num); err == nil && len(availableDatabases) > 0 {
					if num >= 1 && num <= len(availableDatabases) {
						selectedDatabases = append(selectedDatabases, availableDatabases[num-1])
						continue
					}
				}
				
				// Treat as database name
				if part != "" {
					selectedDatabases = append(selectedDatabases, part)
				}
			}
		}
	}

	// Ensure at least one database is selected
	for len(selectedDatabases) == 0 {
		fmt.Print("At least one database is required. Databases to backup: ")
		if scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			if input != "" {
				parts := strings.Split(input, ",")
				for _, part := range parts {
					part = strings.TrimSpace(part)
					if part != "" {
						selectedDatabases = append(selectedDatabases, part)
					}
				}
			}
		}
	}

	// Backup directory
	var defaultDir string
	if runtime.GOOS == "darwin" {
		if os.Geteuid() == 0 {
			defaultDir = "/usr/local/var/tenangdb/backups"
		} else {
			homeDir, _ := os.UserHomeDir()
			defaultDir = filepath.Join(homeDir, "Library", "Application Support", "TenangDB", "backups")
		}
	} else {
		if os.Geteuid() == 0 {
			defaultDir = "/var/backups/tenangdb"
		} else {
			homeDir, _ := os.UserHomeDir()
			defaultDir = filepath.Join(homeDir, ".local", "share", "tenangdb", "backups")
		}
	}

	fmt.Printf("Backup directory [%s]: ", defaultDir)
	backupDir := defaultDir
	if scanner.Scan() {
		if input := strings.TrimSpace(scanner.Text()); input != "" {
			backupDir = input
		}
	}

	return config.BackupConfig{
		Directory:           backupDir,
		Databases:           selectedDatabases,
		BatchSize:           5,
		Concurrency:         3,
		Timeout:             30 * time.Minute,
		RetryCount:          3,
		RetryDelay:          10 * time.Second,
		CheckLastBackupTime: true,
		MinBackupInterval:   1 * time.Hour,
		SkipConfirmation:    false,
	}
}

func setupUploadConfig(rcloneAvailable bool) config.UploadConfig {
	if !rcloneAvailable {
		fmt.Printf("⚠️  Rclone not available, skipping cloud upload setup.\n")
		return config.UploadConfig{Enabled: false}
	}

	scanner := bufio.NewScanner(os.Stdin)
	
	fmt.Print("Enable cloud upload? [y/N]: ")
	enabled := false
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		enabled = response == "y" || response == "yes"
	}

	if !enabled {
		return config.UploadConfig{Enabled: false}
	}

	// Get rclone destination
	fmt.Printf("\nRclone remotes (run 'rclone config' to set up remotes):\n")
	
	var destination string
	fmt.Print("Upload destination (e.g., 'mycloud:backup-folder'): ")
	if scanner.Scan() {
		destination = strings.TrimSpace(scanner.Text())
	}

	for destination == "" {
		fmt.Print("Destination is required. Upload destination: ")
		if scanner.Scan() {
			destination = strings.TrimSpace(scanner.Text())
		}
	}

	return config.UploadConfig{
		Enabled:     true,
		Destination: destination,
		Timeout:     300,
		RetryCount:  3,
	}
}

func setupLoggingAndMetrics() (config.LoggingConfig, config.MetricsConfig) {
	scanner := bufio.NewScanner(os.Stdin)
	
	// Logging level
	fmt.Print("Log level (debug/info/warn/error) [info]: ")
	logLevel := "info"
	if scanner.Scan() {
		if input := strings.TrimSpace(scanner.Text()); input != "" {
			logLevel = input
		}
	}

	// Metrics
	fmt.Print("Enable Prometheus metrics? [y/N]: ")
	metricsEnabled := false
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		metricsEnabled = response == "y" || response == "yes"
	}

	metricsPort := "8080"
	if metricsEnabled {
		fmt.Print("Metrics port [8080]: ")
		if scanner.Scan() {
			if input := strings.TrimSpace(scanner.Text()); input != "" {
				metricsPort = input
			}
		}
	}

	// Default paths
	var logPath, metricsPath string
	if runtime.GOOS == "darwin" {
		if os.Geteuid() == 0 {
			logPath = "/usr/local/var/log/tenangdb/tenangdb.log"
			metricsPath = "/usr/local/var/tenangdb/metrics.json"
		} else {
			homeDir, _ := os.UserHomeDir()
			logPath = filepath.Join(homeDir, "Library", "Logs", "TenangDB", "tenangdb.log")
			metricsPath = filepath.Join(homeDir, "Library", "Application Support", "TenangDB", "metrics.json")
		}
	} else {
		if os.Geteuid() == 0 {
			logPath = "/var/log/tenangdb/tenangdb.log"
			metricsPath = "/var/lib/tenangdb/metrics.json"
		} else {
			homeDir, _ := os.UserHomeDir()
			logPath = filepath.Join(homeDir, ".local", "share", "tenangdb", "logs", "tenangdb.log")
			metricsPath = filepath.Join(homeDir, ".local", "share", "tenangdb", "metrics.json")
		}
	}

	return config.LoggingConfig{
			Level:      logLevel,
			Format:     "clean",
			FileFormat: "text",
			FilePath:   logPath,
		}, config.MetricsConfig{
			Enabled:     metricsEnabled,
			Port:        metricsPort,
			StoragePath: metricsPath,
		}
}

func generateConfig(dbConfig config.DatabaseConfig, backupConfig config.BackupConfig, uploadConfig config.UploadConfig, loggingConfig config.LoggingConfig, metricsConfig config.MetricsConfig) string {
	var configBuilder strings.Builder
	
	configBuilder.WriteString("# TenangDB Configuration\n")
	configBuilder.WriteString("# Generated by: tenangdb init\n")
	configBuilder.WriteString(fmt.Sprintf("# Created: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	
	// Database section
	configBuilder.WriteString("database:\n")
	configBuilder.WriteString(fmt.Sprintf("  host: %s\n", dbConfig.Host))
	configBuilder.WriteString(fmt.Sprintf("  port: %d\n", dbConfig.Port))
	configBuilder.WriteString(fmt.Sprintf("  username: %s\n", dbConfig.Username))
	configBuilder.WriteString(fmt.Sprintf("  password: \"%s\"\n", dbConfig.Password))
	configBuilder.WriteString(fmt.Sprintf("  timeout: %d\n", dbConfig.Timeout))
	configBuilder.WriteString("\n")
	
	// Add mydumper if available
	if _, err := os.Stat(config.FindMydumperPath()); err == nil {
		configBuilder.WriteString("  mydumper:\n")
		configBuilder.WriteString("    enabled: true\n")
		configBuilder.WriteString("    threads: 4\n")
		configBuilder.WriteString("\n")
		
		if _, err := os.Stat(config.FindMyloaderPath()); err == nil {
			configBuilder.WriteString("    myloader:\n")
			configBuilder.WriteString("      enabled: true\n")
			configBuilder.WriteString("      threads: 4\n")
			configBuilder.WriteString("\n")
		}
	}
	
	// Backup section
	configBuilder.WriteString("backup:\n")
	configBuilder.WriteString(fmt.Sprintf("  directory: %s\n", backupConfig.Directory))
	configBuilder.WriteString("  databases:\n")
	for _, db := range backupConfig.Databases {
		configBuilder.WriteString(fmt.Sprintf("    - %s\n", db))
	}
	configBuilder.WriteString(fmt.Sprintf("  batch_size: %d\n", backupConfig.BatchSize))
	configBuilder.WriteString(fmt.Sprintf("  concurrency: %d\n", backupConfig.Concurrency))
	configBuilder.WriteString(fmt.Sprintf("  check_last_backup_time: %t\n", backupConfig.CheckLastBackupTime))
	configBuilder.WriteString(fmt.Sprintf("  min_backup_interval: %s\n", backupConfig.MinBackupInterval))
	configBuilder.WriteString("\n")
	
	// Upload section
	configBuilder.WriteString("upload:\n")
	configBuilder.WriteString(fmt.Sprintf("  enabled: %t\n", uploadConfig.Enabled))
	if uploadConfig.Enabled {
		configBuilder.WriteString(fmt.Sprintf("  destination: \"%s\"\n", uploadConfig.Destination))
		configBuilder.WriteString(fmt.Sprintf("  timeout: %d\n", uploadConfig.Timeout))
		configBuilder.WriteString(fmt.Sprintf("  retry_count: %d\n", uploadConfig.RetryCount))
	}
	configBuilder.WriteString("\n")
	
	// Logging section
	configBuilder.WriteString("logging:\n")
	configBuilder.WriteString(fmt.Sprintf("  level: %s\n", loggingConfig.Level))
	configBuilder.WriteString(fmt.Sprintf("  format: %s\n", loggingConfig.Format))
	configBuilder.WriteString(fmt.Sprintf("  file_path: %s\n", loggingConfig.FilePath))
	configBuilder.WriteString("\n")
	
	// Metrics section
	configBuilder.WriteString("metrics:\n")
	configBuilder.WriteString(fmt.Sprintf("  enabled: %t\n", metricsConfig.Enabled))
	if metricsConfig.Enabled {
		configBuilder.WriteString(fmt.Sprintf("  port: \"%s\"\n", metricsConfig.Port))
		configBuilder.WriteString(fmt.Sprintf("  storage_path: %s\n", metricsConfig.StoragePath))
	}
	configBuilder.WriteString("\n")
	
	// Cleanup section with safe defaults
	configBuilder.WriteString("cleanup:\n")
	configBuilder.WriteString("  enabled: false\n")
	configBuilder.WriteString("  age_based_cleanup: true\n")
	configBuilder.WriteString("  max_age_days: 7\n")
	
	return configBuilder.String()
}

func saveConfig(configContent, configPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write config file
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func createDirectories(backupDir, logPath, metricsPath string) {
	dirs := []string{
		backupDir,
		filepath.Dir(logPath),
	}
	
	if metricsPath != "" {
		dirs = append(dirs, filepath.Dir(metricsPath))
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("⚠️  Failed to create directory %s: %v\n", dir, err)
		} else {
			fmt.Printf("✅ Created directory: %s\n", dir)
		}
	}
}

func promptSystemdDeployment() bool {
	// Only prompt on Linux
	if runtime.GOOS != "linux" {
		return false
	}
	
	fmt.Printf("\n🚀 Systemd Deployment (Optional)\n")
	fmt.Printf("=================================\n")
	fmt.Printf("TenangDB can be deployed as a systemd service for:\n")
	fmt.Printf("  ✅ Automated daily backups\n")
	fmt.Printf("  ✅ Weekend cleanup\n")  
	fmt.Printf("  ✅ Always-on metrics server\n")
	fmt.Printf("  ✅ Auto-restart on failures\n\n")
	
	if os.Geteuid() != 0 {
		fmt.Printf("⚠️  Note: This requires sudo privileges (will show instructions)\n")
	}
	
	fmt.Print("Deploy as systemd service? [y/N]: ")
	
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		return response == "y" || response == "yes"
	}
	
	return false
}

func deployAsSystemdService(configPath, systemdUser, metricsPort string) error {
	// Check if running on Linux
	if runtime.GOOS != "linux" {
		return fmt.Errorf("systemd deployment is only supported on Linux")
	}
	
	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	
	// Create systemd user if doesn't exist
	if err := createSystemdUser(systemdUser); err != nil {
		return fmt.Errorf("failed to create systemd user: %w", err)
	}
	
	// Create system directories
	if err := createSystemDirectories(systemdUser); err != nil {
		return fmt.Errorf("failed to create system directories: %w", err)
	}
	
	// Install binary to system location
	if err := installBinary(execPath, systemdUser); err != nil {
		return fmt.Errorf("failed to install binary: %w", err)
	}
	
	// Copy config to system location
	if err := installConfig(configPath); err != nil {
		return fmt.Errorf("failed to install config: %w", err)
	}
	
	// Generate and install systemd service files
	if err := installSystemdServices(systemdUser, metricsPort); err != nil {
		return fmt.Errorf("failed to install systemd services: %w", err)
	}
	
	// Enable and start services
	if err := enableSystemdServices(); err != nil {
		return fmt.Errorf("failed to enable systemd services: %w", err)
	}
	
	return nil
}

func createSystemdUser(username string) error {
	fmt.Printf("Creating system user '%s'...\n", username)
	
	// Check if user exists
	if _, err := exec.LookPath("id"); err != nil {
		return fmt.Errorf("id command not found")
	}
	
	cmd := exec.Command("id", username)
	if cmd.Run() == nil {
		fmt.Printf("✅ User '%s' already exists\n", username)
		return nil
	}
	
	// Create group
	cmd = execCommand("groupadd", "-r", username)
	if err := cmd.Run(); err != nil {
		// Group might already exist, continue - this is expected
		fmt.Printf("Group creation result (expected if exists): %v\n", err)
	}
	
	// Create user
	cmd = execCommand("useradd", "-r", "-g", username, "-s", "/bin/false", "-d", "/opt/tenangdb", username)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	
	fmt.Printf("✅ Created system user '%s'\n", username)
	return nil
}

func createSystemDirectories(systemdUser string) error {
	fmt.Printf("Creating system directories...\n")
	
	directories := []string{
		"/opt/tenangdb",
		"/etc/tenangdb", 
		"/var/log/tenangdb",
		"/var/backups/tenangdb",
		"/var/lib/tenangdb",
	}
	
	for _, dir := range directories {
		cmd := execCommand("mkdir", "-p", dir)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		
		// Set ownership
		cmd = execCommand("chown", systemdUser+":"+systemdUser, dir)
		if err := cmd.Run(); err != nil {
			// Some directories might need different ownership, continue - this is expected
			fmt.Printf("Ownership setting result for %s (expected for some dirs): %v\n", dir, err)
		}
		
		// Set permissions
		cmd = execCommand("chmod", "755", dir)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set permissions for %s: %w", dir, err)
		}
	}
	
	fmt.Printf("✅ Created system directories\n")
	return nil
}

func installBinary(execPath, _ string) error {
	fmt.Printf("Installing binary to /opt/tenangdb/...\n")
	
	// Copy main binary
	cmd := execCommand("cp", execPath, "/opt/tenangdb/tenangdb")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}
	
	// Set permissions
	cmd = execCommand("chmod", "+x", "/opt/tenangdb/tenangdb")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set binary permissions: %w", err)
	}
	
	// Try to copy exporter binary if it exists in same directory
	execDir := filepath.Dir(execPath)
	exporterPath := filepath.Join(execDir, "tenangdb-exporter")
	if _, err := os.Stat(exporterPath); err == nil {
		cmd = execCommand("cp", exporterPath, "/opt/tenangdb/tenangdb-exporter")
		if err := cmd.Run(); err == nil {
			cmd = execCommand("chmod", "+x", "/opt/tenangdb/tenangdb-exporter")
			if err := cmd.Run(); err != nil {
				fmt.Printf("⚠️  Failed to set exporter permissions: %v\n", err)
			} else {
				fmt.Printf("✅ Installed tenangdb-exporter\n")
			}
		}
	}
	
	fmt.Printf("✅ Installed binary to /opt/tenangdb/tenangdb\n")
	return nil
}

// execCommand runs a command with or without sudo based on current privileges
func execCommand(args ...string) *exec.Cmd {
	if os.Geteuid() == 0 {
		// Already running as root, no need for sudo
		return exec.Command(args[0], args[1:]...)
	} else {
		// Not root, use sudo
		return exec.Command("sudo", args...)
	}
}

func installConfig(configPath string) error {
	fmt.Printf("Installing configuration to /etc/tenangdb/...\n")
	
	targetPath := "/etc/tenangdb/config.yaml"
	
	// Check if source and target are the same file
	if configPath == targetPath {
		fmt.Printf("✅ Configuration already at target location\n")
	} else {
		// Copy config file
		cmd := execCommand("cp", configPath, targetPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to copy config: %w", err)
		}
		fmt.Printf("✅ Copied configuration to %s\n", targetPath)
	}
	
	// Set permissions
	cmd := execCommand("chmod", "640", targetPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set config permissions: %w", err)
	}
	
	fmt.Printf("✅ Configuration permissions set\n")
	return nil
}

func installSystemdServices(systemdUser, metricsPort string) error {
	fmt.Printf("Installing systemd service files...\n")
	
	// Generate service file content
	services := map[string]string{
		"tenangdb.service": generateTenangDBService(systemdUser),
		"tenangdb.timer": generateTenangDBTimer(),
		"tenangdb-cleanup.service": generateCleanupService(systemdUser),
		"tenangdb-cleanup.timer": generateCleanupTimer(),
		"tenangdb-exporter.service": generateExporterService(systemdUser, metricsPort),
	}
	
	for filename, content := range services {
		// Write service file to temp location
		tempFile := filepath.Join("/tmp", filename)
		if err := os.WriteFile(tempFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
		
		// Copy to systemd directory
		cmd := execCommand("cp", tempFile, "/etc/systemd/system/"+filename)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install %s: %w", filename, err)
		}
		
		// Clean up temp file
		os.Remove(tempFile)
		
		fmt.Printf("✅ Installed %s\n", filename)
	}
	
	// Reload systemd
	cmd := execCommand("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}
	
	fmt.Printf("✅ Systemd daemon reloaded\n")
	return nil
}

func enableSystemdServices() error {
	fmt.Printf("Enabling and starting systemd services...\n")
	
	services := []string{
		"tenangdb.timer",
		"tenangdb-cleanup.timer", 
		"tenangdb-exporter.service",
	}
	
	for _, service := range services {
		// Enable service
		cmd := execCommand("systemctl", "enable", service)
		if err := cmd.Run(); err != nil {
			fmt.Printf("⚠️  Failed to enable %s: %v\n", service, err)
			continue
		}
		
		// Start service  
		cmd = execCommand("systemctl", "start", service)
		if err := cmd.Run(); err != nil {
			fmt.Printf("⚠️  Failed to start %s: %v\n", service, err)
			continue
		}
		
		fmt.Printf("✅ Enabled and started %s\n", service)
	}
	
	return nil
}

func generateTenangDBService(systemdUser string) string {
	return fmt.Sprintf(`[Unit]
Description=TenangDB Backup Service
After=network.target mysqld.service
Requires=mysqld.service

[Service]
Type=oneshot
User=%s
Group=%s
WorkingDirectory=/opt/tenangdb
ExecStart=/opt/tenangdb/tenangdb backup --config /etc/tenangdb/config.yaml --yes
StandardOutput=journal
StandardError=journal
TimeoutStartSec=3600
TimeoutStopSec=300

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/backups/tenangdb /var/log/tenangdb /var/lib/tenangdb
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

[Install]
WantedBy=multi-user.target
`, systemdUser, systemdUser)
}

func generateTenangDBTimer() string {
	return `[Unit]
Description=TenangDB Backup Timer
Requires=tenangdb.service

[Timer]
OnCalendar=daily
Persistent=true
RandomizedDelaySec=300

[Install]
WantedBy=timers.target
`
}

func generateCleanupService(systemdUser string) string {
	return fmt.Sprintf(`[Unit]
Description=TenangDB Cleanup Service
After=network.target

[Service]
Type=oneshot
User=%s
Group=%s
WorkingDirectory=/opt/tenangdb
ExecStart=/opt/tenangdb/tenangdb cleanup --config /etc/tenangdb/config.yaml --yes
StandardOutput=journal
StandardError=journal
TimeoutStartSec=1800
TimeoutStopSec=300

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/backups/tenangdb /var/log/tenangdb /var/lib/tenangdb
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

[Install]
WantedBy=multi-user.target
`, systemdUser, systemdUser)
}

func generateCleanupTimer() string {
	return `[Unit]
Description=TenangDB Cleanup Timer
Requires=tenangdb-cleanup.service

[Timer]
OnCalendar=Sat,Sun 02:00
Persistent=true
RandomizedDelaySec=600

[Install]
WantedBy=timers.target
`
}

func generateExporterService(systemdUser, metricsPort string) string {
	return fmt.Sprintf(`[Unit]
Description=TenangDB Metrics Exporter
Documentation=https://tenangdb.ainun.cloud
After=network.target
Wants=network.target

[Service]
Type=simple
User=%s
Group=%s
WorkingDirectory=/opt/tenangdb
ExecStart=/opt/tenangdb/tenangdb-exporter --config /etc/tenangdb/config.yaml --port %s
ExecReload=/bin/kill -HUP $MAINPID
Restart=always
RestartSec=5
KillMode=mixed
KillSignal=SIGTERM
TimeoutStopSec=30

# Output to journal
StandardOutput=journal
StandardError=journal
SyslogIdentifier=tenangdb-exporter

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/tenangdb /var/log/tenangdb
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
MemoryDenyWriteExecute=true
RestrictRealtime=true
RestrictSUIDSGID=true
RemoveIPC=true
PrivateDevices=true

# Network restrictions
RestrictAddressFamilies=AF_INET AF_INET6
IPAddressDeny=any
IPAddressAllow=localhost 
IPAddressAllow=127.0.0.0/8
IPAddressAllow=::1/128

[Install]
WantedBy=multi-user.target
`, systemdUser, systemdUser, metricsPort)
}
