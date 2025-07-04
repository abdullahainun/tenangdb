package config

import (
	"fmt"
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
	Directory   string        `mapstructure:"directory"`
	Databases   []string      `mapstructure:"databases"`
	BatchSize   int           `mapstructure:"batch_size"`
	Concurrency int           `mapstructure:"concurrency"`
	Timeout     time.Duration `mapstructure:"timeout"`
	RetryCount  int           `mapstructure:"retry_count"`
	RetryDelay  time.Duration `mapstructure:"retry_delay"`
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
	Enabled     bool   `mapstructure:"enabled"`
	RclonePath  string `mapstructure:"rclone_path"`
	Destination string `mapstructure:"destination"`
	Timeout     int    `mapstructure:"timeout"`
	RetryCount  int    `mapstructure:"retry_count"`
}

type LoggingConfig struct {
	Level    string `mapstructure:"level"`
	Format   string `mapstructure:"format"`
	FilePath string `mapstructure:"file_path"`
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
	Enabled bool   `mapstructure:"enabled"`
	Port    string `mapstructure:"port"`
}

func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Set default values
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
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

func setDefaults() {
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 3306)
	viper.SetDefault("database.timeout", 30)

	viper.SetDefault("backup.directory", "/tmp/db-backups")
	viper.SetDefault("backup.batch_size", 10)
	viper.SetDefault("backup.concurrency", 3)
	viper.SetDefault("backup.timeout", "30m")
	viper.SetDefault("backup.retry_count", 3)
	viper.SetDefault("backup.retry_delay", "10s")

	// Mydumper defaults
	viper.SetDefault("database.mydumper.enabled", true)
	viper.SetDefault("database.mydumper.binary_path", "/usr/bin/mydumper")
	viper.SetDefault("database.mydumper.threads", 4)
	viper.SetDefault("database.mydumper.chunk_filesize", 100)
	viper.SetDefault("database.mydumper.compress_method", "gzip")
	viper.SetDefault("database.mydumper.build_empty_files", false)
	viper.SetDefault("database.mydumper.use_defer", true)
	viper.SetDefault("database.mydumper.single_table", false)
	viper.SetDefault("database.mydumper.no_schemas", false)
	viper.SetDefault("database.mydumper.no_data", false)

	// Myloader defaults
	viper.SetDefault("database.mydumper.myloader.enabled", true)
	viper.SetDefault("database.mydumper.myloader.binary_path", "/usr/bin/myloader")
	viper.SetDefault("database.mydumper.myloader.threads", 4)

	viper.SetDefault("upload.enabled", true)
	viper.SetDefault("upload.rclone_path", "/usr/bin/rclone")
	viper.SetDefault("upload.timeout", 300)
	viper.SetDefault("upload.retry_count", 3)

	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.file_path", "/var/log/db-backup.log")

	viper.SetDefault("cleanup.enabled", true)
	viper.SetDefault("cleanup.cleanup_uploaded_files", true)
	viper.SetDefault("cleanup.remote_retention_days", 30)
	viper.SetDefault("cleanup.weekend_only", true)
	viper.SetDefault("cleanup.age_based_cleanup", false)
	viper.SetDefault("cleanup.max_age_days", 7)
	viper.SetDefault("cleanup.verify_cloud_exists", true)

	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.port", "8080")
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
