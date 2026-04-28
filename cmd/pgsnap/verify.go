package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify [backup-id]",
	Short: "Verify backup integrity",
	Long: `Verify the integrity of a backup.

This checks that all backup files are present and their checksums match
the recorded values.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("verify: not implemented")
		return nil
	},
}

func init() {
	// Verify-specific flags will be added here
}
