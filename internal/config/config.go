// Package config provides configuration loading and management for pgsnap.
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for pgsnap.
type Config struct {
	// PostgreSQL connection settings
	Postgres PostgresConfig `mapstructure:"postgres"`

	// Storage backend settings
	Storage StorageConfig `mapstructure:"storage"`

	// Backup settings
	Backup BackupConfig `mapstructure:"backup"`

	// Logging settings
	Log LogConfig `mapstructure:"log"`
}

// PostgresConfig holds PostgreSQL connection settings.
type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	SSLMode  string `mapstructure:"sslmode"`
	Port     int    `mapstructure:"port"`
}

// StorageConfig holds storage backend settings.
type StorageConfig struct {
	Type string `mapstructure:"type"` // "local", "s3", "gcs", "azure"

	// Local storage settings
	Local LocalStorageConfig `mapstructure:"local"`

	// S3 storage settings (future)
	S3 S3StorageConfig `mapstructure:"s3"`
}

// LocalStorageConfig holds local filesystem storage settings.
type LocalStorageConfig struct {
	Path string `mapstructure:"path"`
}

// S3StorageConfig holds S3 storage settings.
type S3StorageConfig struct {
	Bucket    string `mapstructure:"bucket"`
	Region    string `mapstructure:"region"`
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
}

// BackupConfig holds backup operation settings.
type BackupConfig struct {
	Parallel      int  `mapstructure:"parallel"`
	RetentionDays int  `mapstructure:"retention_days"`
	Compress      bool `mapstructure:"compress"`
	Encrypt       bool `mapstructure:"encrypt"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string `mapstructure:"level"`  // debug, info, warn, error
	Format string `mapstructure:"format"` // json, console
}

// Load reads configuration from file and environment variables.
// It looks for config in:
// 1. Path specified by configFile parameter (if not empty)
// 2. ./pgsnap.yaml
// 3. ~/.pgsnap.yaml
// 4. /etc/pgsnap/pgsnap.yaml
//
// Environment variables with PGSNAP_ prefix override file settings.
func Load(configFile string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Config file settings
	v.SetConfigName("pgsnap")
	v.SetConfigType("yaml")

	if configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME")
		v.AddConfigPath("/etc/pgsnap")
	}

	// Environment variable settings
	v.SetEnvPrefix("PGSNAP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file (ignore if not found)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// PostgreSQL defaults
	v.SetDefault("postgres.host", "localhost")
	v.SetDefault("postgres.port", 5432)
	v.SetDefault("postgres.user", "postgres")
	v.SetDefault("postgres.database", "postgres")
	v.SetDefault("postgres.sslmode", "prefer")

	// Storage defaults
	v.SetDefault("storage.type", "local")
	v.SetDefault("storage.local.path", "./backups")

	// Backup defaults
	v.SetDefault("backup.compress", true)
	v.SetDefault("backup.encrypt", false)
	v.SetDefault("backup.parallel", 4)
	v.SetDefault("backup.retention_days", 30)

	// Log defaults
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "console")
}
