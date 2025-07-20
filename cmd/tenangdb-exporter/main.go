package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/abdullahainun/tenangdb/internal/config"
	"github.com/abdullahainun/tenangdb/internal/logger"
	"github.com/abdullahainun/tenangdb/internal/metrics"

	"github.com/spf13/cobra"
)

var (
	version   string // Set via ldflags during build
	buildTime string // Set via ldflags during build
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "tenangdb-exporter",
		Short: "TenangDB Prometheus metrics exporter",
		Long:  `Standalone HTTP server to expose TenangDB metrics for Prometheus scraping.`,
		Run:   runExporter,
	}

	var configFile string
	var logLevel string
	var port string
	var metricsFile string
	var showVersionFlag bool

	rootCmd.Flags().StringVar(&configFile, "config", "", "config file path (auto-discovery if not specified)")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.Flags().StringVar(&port, "port", "9090", "HTTP server port for metrics")
	rootCmd.Flags().StringVar(&metricsFile, "metrics-file", "", "path to metrics storage file (auto-discovery if not specified)")
	rootCmd.Flags().BoolVar(&showVersionFlag, "version", false, "show version information")

	// Add version command
	rootCmd.AddCommand(newVersionCommand())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runExporter(cmd *cobra.Command, args []string) {
	// Check if version flag is set
	showVersionFlag, _ := cmd.Flags().GetBool("version")
	if showVersionFlag {
		showVersion()
		return
	}

	configFile, _ := cmd.Flags().GetString("config")
	logLevel, _ := cmd.Flags().GetString("log-level")
	port, _ := cmd.Flags().GetString("port")
	metricsFile, _ := cmd.Flags().GetString("metrics-file")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Try to load configuration (optional for exporter)
	var cfg *config.Config
	var log *logger.Logger
	
	if configFile != "" {
		// Config file explicitly specified, load it
		var err error
		cfg, err = config.LoadConfig(configFile)
		if err != nil {
			// Use basic logger if config fails
			log = logger.NewLogger(logLevel)
			log.WithError(err).Fatal("Failed to load configuration")
		}
	} else {
		// No config file specified, try auto-discovery (but don't fail if not found)
		var err error
		cfg, err = config.LoadConfig("")
		if err != nil {
			// Config not found or invalid, use defaults - this is OK for exporter
			cfg = nil
		}
	}

	// Determine effective log level: CLI flag overrides config
	effectiveLogLevel := logLevel
	var logFilePath, logFormat, logFileFormat string
	
	if cfg != nil {
		if logLevel == "info" && cfg.Logging.Level != "" {
			// If CLI uses default "info" and config has a level set, use config
			effectiveLogLevel = cfg.Logging.Level
		}
		logFilePath = cfg.Logging.FilePath
		logFormat = cfg.Logging.Format
		logFileFormat = cfg.Logging.FileFormat
	}

	// Initialize file logger with separate formats for stdout and file
	if logFilePath != "" {
		var err error
		log, err = logger.NewFileLoggerWithSeparateFormats(effectiveLogLevel, logFilePath, logFormat, logFileFormat)
		if err != nil {
			// Fallback to stdout logger
			log = logger.NewLogger(effectiveLogLevel)
			log.WithError(err).Warn("Failed to initialize file logger, using stdout")
		}
	} else {
		// No file logging configured, use stdout logger
		log = logger.NewLogger(effectiveLogLevel)
	}

	// Use config-based metrics file path if not specified
	if metricsFile == "" {
		if cfg != nil && cfg.Metrics.StoragePath != "" {
			metricsFile = cfg.Metrics.StoragePath
		} else {
			metricsFile = "/var/lib/tenangdb/metrics.json" // fallback
		}
	}

	log.WithField("port", port).WithField("metrics_file", metricsFile).Info("Starting tenangdb-exporter")

	// Start metrics exporter
	done := make(chan error, 1)
	go func() {
		done <- metrics.StartMetricsExporter(ctx, port, metricsFile, log)
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

	log.Info("TenangDB metrics exporter stopped")
}

func newVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Display the version and build information for TenangDB Exporter.`,
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
	fmt.Printf("TenangDB Exporter version %s\n", version)
	fmt.Printf("Build time: %s\n", buildTime)
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}