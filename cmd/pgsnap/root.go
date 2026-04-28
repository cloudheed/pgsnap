package main

import (
	"fmt"

	"github.com/cloudheed/pgsnap/internal/config"
	"github.com/cloudheed/pgsnap/internal/logger"
	"github.com/spf13/cobra"
)

var (
	// Version is set at build time
	Version = "dev"

	// Configuration
	cfgFile string
	cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "pgsnap",
	Short: "PostgreSQL backup and restore tool",
	Long: `pgsnap is a fast, reliable PostgreSQL backup and restore tool.

It supports multiple storage backends (local, S3, GCS, Azure) and provides
features like compression, encryption, and incremental backups.`,
	Version: Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for help/version
		if cmd.Name() == "help" || cmd.Name() == "version" {
			return nil
		}

		var err error
		cfg, err = config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Initialize logger (will be used by subcommands)
		_ = logger.New(logger.Options{
			Level:  cfg.Log.Level,
			Format: cfg.Log.Format,
		})

		return nil
	},
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default: ./pgsnap.yaml)")

	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(verifyCmd)
}

