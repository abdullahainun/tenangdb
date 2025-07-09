package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Database DatabaseConfig `mapstructure:"database"`
	Backup   BackupConfig   `mapstructure:"backup"`
	Upload   UploadConfig   `mapstructure:"upload"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Cleanup  CleanupConfig  `mapstructure:"cleanup"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
}

type DatabaseConfig struct {
	Host     string          `mapstructure:"host"`
	Port     int             `mapstructure:"port"`
	Username string          `mapstructure:"username"`
	Password string          `mapstructure:"password"`
	Timeout  int             `mapstructure:"timeout"`
	Mydumper *MydumperConfig `mapstructure:"mydumper"`
}

type BackupConfig struct {
	Directory             string        `mapstructure:"directory"`
	Databases             []string      `mapstructure:"databases"`
	BatchSize             int           `mapstructure:"batch_size"`
	Concurrency           int           `mapstructure:"concurrency"`
	Timeout               time.Duration `mapstructure:"timeout"`
	RetryCount            int           `mapstructure:"retry_count"`
	RetryDelay            time.Duration `mapstructure:"retry_delay"`
	CheckLastBackupTime   bool          `mapstructure:"check_last_backup_time"`
	MinBackupInterval     time.Duration `mapstructure:"min_backup_interval"`
	SkipConfirmation      bool          `mapstructure:"skip_confirmation"`
}

type MydumperConfig struct {
	Enabled         bool            `mapstructure:"enabled"`
	BinaryPath      string          `mapstructure:"binary_path"`
	DefaultsFile    string          `mapstructure:"defaults_file"`
	Threads         int             `mapstructure:"threads"`
	ChunkFilesize   int             `mapstructure:"chunk_filesize"`
	CompressMethod  string          `mapstructure:"compress_method"`
	BuildEmptyFiles bool            `mapstructure:"build_empty_files"`
	UseDefer        bool            `mapstructure:"use_defer"`
	SingleTable     bool            `mapstructure:"single_table"`
	NoSchemas       bool            `mapstructure:"no_schemas"`
	NoData          bool            `mapstructure:"no_data"`
	Myloader        *MyloaderConfig `mapstructure:"myloader"`
}

type MyloaderConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	BinaryPath   string `mapstructure:"binary_path"`
	DefaultsFile string `mapstructure:"defaults_file"`
	Threads      int    `mapstructure:"threads"`
}

type UploadConfig struct {
	Enabled          bool   `mapstructure:"enabled"`
	RclonePath       string `mapstructure:"rclone_path"`
	RcloneConfigPath string `mapstructure:"rclone_config_path"`
	Destination      string `mapstructure:"destination"`
	Timeout          int    `mapstructure:"timeout"`
	RetryCount       int    `mapstructure:"retry_count"`
}

type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	FileFormat string `mapstructure:"file_format"`
	FilePath   string `mapstructure:"file_path"`
}

type CleanupConfig struct {
	Enabled              bool     `mapstructure:"enabled"`
	CleanupUploadedFiles bool     `mapstructure:"cleanup_uploaded_files"`
	RemoteRetention      int      `mapstructure:"remote_retention_days"`
	WeekendOnly          bool     `mapstructure:"weekend_only"`
	AgeBasedCleanup      bool     `mapstructure:"age_based_cleanup"`
	MaxAgeDays           int      `mapstructure:"max_age_days"`
	VerifyCloudExists    bool     `mapstructure:"verify_cloud_exists"`
	Databases            []string `mapstructure:"databases"`
}

type MetricsConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	Port        string `mapstructure:"port"`
	StoragePath string `mapstructure:"storage_path"`
}

func LoadConfig(configPath string) (*Config, error) {
	// Set default values first
	setDefaults()

	// If specific config path is provided, use it directly
	if configPath != "" {
		viper.SetConfigFile(configPath)
		viper.SetConfigType("yaml")

		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}
	} else {
		// Auto-discover config file using multi-platform paths
		foundPath, err := findConfigFile()
		if err != nil {
			return nil, fmt.Errorf("failed to find config file: %w", err)
		}

		viper.SetConfigFile(foundPath)
		viper.SetConfigType("yaml")

		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", foundPath, err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// findConfigFile searches for config file in platform-specific locations
func findConfigFile() (string, error) {
	configPaths := getConfigPaths()

	for _, path := range configPaths {
		// Expand ~ to home directory
		expandedPath := expandHomeDir(path)
		
		if _, err := os.Stat(expandedPath); err == nil {
			return expandedPath, nil
		}
	}

	return "", fmt.Errorf("no config file found in any of the standard locations: %v", configPaths)
}

// getConfigPaths returns platform-specific config file paths in priority order
func getConfigPaths() []string {
	if runtime.GOOS == "darwin" {
		// macOS specific paths
		return []string{
			"/usr/local/etc/tenangdb/config.yaml",                  // Homebrew system-wide
			"/etc/tenangdb/config.yaml",                            // System fallback
			"~/Library/Application Support/TenangDB/config.yaml",   // macOS user config
			"~/.config/tenangdb/config.yaml",                      // XDG fallback
			"./config.yaml",                                        // Current dir
			"./tenangdb.yaml",                                      // Current dir alt
		}
	} else {
		// Linux/Unix paths
		return []string{
			"/etc/tenangdb/config.yaml",         // System-wide
			"~/.config/tenangdb/config.yaml",    // User-specific
			"./config.yaml",                     // Current dir
			"./tenangdb.yaml",                   // Current dir alt
		}
	}
}

// expandHomeDir expands ~ to the user's home directory
func expandHomeDir(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path // Return original path if home dir can't be determined
	}

	return filepath.Join(homeDir, path[2:])
}

func setDefaults() {
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 3306)
	viper.SetDefault("database.timeout", 30)

	viper.SetDefault("backup.directory", "backups")
	viper.SetDefault("backup.batch_size", 5)
	viper.SetDefault("backup.concurrency", 3)
	viper.SetDefault("backup.timeout", "30m")
	viper.SetDefault("backup.retry_count", 3)
	viper.SetDefault("backup.retry_delay", "10s")
	viper.SetDefault("backup.check_last_backup_time", true)
	viper.SetDefault("backup.min_backup_interval", "1h")
	viper.SetDefault("backup.skip_confirmation", false)

	// Platform-specific binary paths
	if runtime.GOOS == "darwin" {
		// macOS defaults (Homebrew)
		viper.SetDefault("database.mydumper.binary_path", "/usr/local/bin/mydumper")
		viper.SetDefault("database.mydumper.myloader.binary_path", "/usr/local/bin/myloader")
		viper.SetDefault("upload.rclone_path", "/usr/local/bin/rclone")
		viper.SetDefault("upload.rclone_config_path", "~/.config/rclone/rclone.conf")
		viper.SetDefault("logging.file_path", "~/Library/Logs/TenangDB/tenangdb.log")
	} else {
		// Linux/Unix defaults
		viper.SetDefault("database.mydumper.binary_path", "/usr/bin/mydumper")
		viper.SetDefault("database.mydumper.myloader.binary_path", "/usr/bin/myloader")
		viper.SetDefault("upload.rclone_path", "/usr/bin/rclone")
		viper.SetDefault("upload.rclone_config_path", "~/.config/rclone/rclone.conf")
		viper.SetDefault("logging.file_path", "/var/log/tenangdb/tenangdb.log")
	}

	// Mydumper defaults
	viper.SetDefault("database.mydumper.enabled", false)
	viper.SetDefault("database.mydumper.threads", 4)
	viper.SetDefault("database.mydumper.chunk_filesize", 100)
	viper.SetDefault("database.mydumper.compress_method", "gzip")
	viper.SetDefault("database.mydumper.build_empty_files", false)
	viper.SetDefault("database.mydumper.use_defer", true)
	viper.SetDefault("database.mydumper.single_table", false)
	viper.SetDefault("database.mydumper.no_schemas", false)
	viper.SetDefault("database.mydumper.no_data", false)

	// Myloader defaults
	viper.SetDefault("database.mydumper.myloader.enabled", false)
	viper.SetDefault("database.mydumper.myloader.threads", 4)

	viper.SetDefault("upload.enabled", false)
	viper.SetDefault("upload.timeout", 300)
	viper.SetDefault("upload.retry_count", 3)

	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "clean")
	viper.SetDefault("logging.file_format", "text")

	viper.SetDefault("cleanup.enabled", false)
	viper.SetDefault("cleanup.cleanup_uploaded_files", true)
	viper.SetDefault("cleanup.remote_retention_days", 30)
	viper.SetDefault("cleanup.weekend_only", true)
	viper.SetDefault("cleanup.age_based_cleanup", false)
	viper.SetDefault("cleanup.max_age_days", 7)
	viper.SetDefault("cleanup.verify_cloud_exists", true)

	viper.SetDefault("metrics.enabled", false)
	viper.SetDefault("metrics.port", "8080")
}

// GetConfigPaths returns the config paths for the current platform (for CLI help)
func GetConfigPaths() []string {
	return getConfigPaths()
}

// GetActiveConfigPath returns the path of the config file that would be used
func GetActiveConfigPath() (string, error) {
	return findConfigFile()
}

func validateConfig(config *Config) error {
	if config.Database.Username == "" {
		return fmt.Errorf("database username is required")
	}

	if len(config.Backup.Databases) == 0 {
		return fmt.Errorf("at least one database must be specified")
	}

	if config.Backup.BatchSize <= 0 {
		return fmt.Errorf("batch size must be greater than 0")
	}

	if config.Backup.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be greater than 0")
	}

	if config.Upload.Enabled && config.Upload.Destination == "" {
		return fmt.Errorf("upload destination is required when upload is enabled")
	}

	// Mydumper validation
	if config.Database.Mydumper != nil && config.Database.Mydumper.Enabled {
		if config.Database.Mydumper.Threads <= 0 {
			return fmt.Errorf("mydumper threads must be greater than 0")
		}
		if config.Database.Mydumper.ChunkFilesize <= 0 {
			return fmt.Errorf("mydumper chunk filesize must be greater than 0")
		}
		if config.Database.Mydumper.CompressMethod != "" &&
			config.Database.Mydumper.CompressMethod != "gzip" &&
			config.Database.Mydumper.CompressMethod != "lz4" {
			return fmt.Errorf("mydumper compress method must be 'gzip', 'lz4', or empty")
		}

		// Myloader validation
		if config.Database.Mydumper.Myloader != nil && config.Database.Mydumper.Myloader.Enabled {
			if config.Database.Mydumper.Myloader.Threads <= 0 {
				return fmt.Errorf("myloader threads must be greater than 0")
			}
		}
	}

	return nil
}
