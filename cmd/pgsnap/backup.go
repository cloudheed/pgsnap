// Package main provides the pgsnap CLI entrypoint.
package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create a database backup",
	Long: `Create a backup of the PostgreSQL database.

The backup will be stored in the configured storage backend with
optional compression and encryption.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("backup: not implemented")
		return nil
	},
}

func init() {
	// Backup-specific flags will be added here
	// backupCmd.Flags().StringP("database", "d", "", "database to backup")
}
