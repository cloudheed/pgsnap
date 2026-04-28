package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available backups",
	Long: `List all available backups in the configured storage backend.

The output includes backup ID, timestamp, size, and status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("list: not implemented")
		return nil
	},
}

func init() {
	// List-specific flags will be added here
	// listCmd.Flags().IntP("limit", "n", 10, "number of backups to show")
}
