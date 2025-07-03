package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"db-backup-tool/internal/config"
	"db-backup-tool/internal/logger"
	"db-backup-tool/internal/backup"

	"github.com/spf13/cobra"
)

var (
	configFile string
	logLevel   string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "db-backup-tool",
		Short: "A robust database backup tool with batch processing and cloud upload",
		Long:  `A Go-based database backup tool that supports batch processing, cloud uploads via rclone, and graceful error handling.`,
		Run:   run,
	}

	rootCmd.PersistentFlags().StringVar(&configFile, "config", "configs/config.yaml", "config file path")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")

	// Add cleanup subcommand
	rootCmd.AddCommand(newCleanupCommand())

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

	// Initialize logger
	log := logger.NewLogger(logLevel)
	
	// Load configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.WithError(err).Fatal("Failed to load configuration")
	}

	// Initialize backup service
	backupService, err := backup.NewService(cfg, log)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize backup service")
	}

	// Start backup process
	go func() {
		if err := backupService.Run(ctx); err != nil {
			log.WithError(err).Error("Backup process failed")
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Info("Received shutdown signal, gracefully shutting down...")
	cancel()
}

func newCleanupCommand() *cobra.Command {
	var configFile string
	var logLevel string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Cleanup uploaded backup files",
		Long:  `Remove local backup files that have been successfully uploaded to cloud storage.`,
		Run: func(cmd *cobra.Command, args []string) {
			runCleanup(configFile, logLevel, dryRun)
		},
	}

	cmd.Flags().StringVar(&configFile, "config", "configs/config.yaml", "config file path")
	cmd.Flags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be deleted without actually deleting")

	return cmd
}

func runCleanup(configFile, logLevel string, dryRun bool) {
	ctx := context.Background()

	// Initialize logger
	log := logger.NewLogger(logLevel)
	
	// Load configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.WithError(err).Fatal("Failed to load configuration")
	}

	// Check if today is weekend (Saturday or Sunday)
	today := time.Now().Weekday()
	if today != time.Saturday && today != time.Sunday {
		log.Info("Cleanup only runs on weekends. Skipping cleanup.")
		return
	}

	log.Info("Starting weekend cleanup process")

	// Initialize backup service to access uploaded files tracking
	backupService, err := backup.NewService(cfg, log)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize backup service")
	}

	if dryRun {
		log.Info("DRY RUN MODE: No files will be actually deleted")
		showFilesToCleanup(backupService, log)
		return
	}

	// Perform cleanup of uploaded files
	if err := backupService.CleanupUploadedFiles(ctx); err != nil {
		log.WithError(err).Error("Cleanup process failed")
		os.Exit(1)
	}

	log.Info("Weekend cleanup completed successfully")
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