package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudheed/pgsnap/internal/retention"
	"github.com/spf13/cobra"
)

var (
	pruneDryRun     bool
	pruneMaxAge     int
	pruneMaxCount   int
	pruneKeepDaily  int
	pruneKeepWeekly int
	pruneKeepMonthly int
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Delete old backups according to retention policy",
	Long: `Delete old backups according to the retention policy.

Use --dry-run to preview what would be deleted without actually deleting.`,
	RunE: runPrune,
}

func runPrune(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create storage backend
	backend, err := createBackend(ctx)
	if err != nil {
		return fmt.Errorf("failed to create storage backend: %w", err)
	}

	// Build retention policy
	policy := retention.Policy{}

	if pruneMaxAge > 0 {
		policy.MaxAge = time.Duration(pruneMaxAge) * 24 * time.Hour
	} else if cfg.Backup.RetentionDays > 0 {
		policy.MaxAge = time.Duration(cfg.Backup.RetentionDays) * 24 * time.Hour
	}

	if pruneMaxCount > 0 {
		policy.MaxCount = pruneMaxCount
	}

	if pruneKeepDaily > 0 {
		policy.KeepDaily = pruneKeepDaily
	}

	if pruneKeepWeekly > 0 {
		policy.KeepWeekly = pruneKeepWeekly
	}

	if pruneKeepMonthly > 0 {
		policy.KeepMonthly = pruneKeepMonthly
	}

	// If no policy specified, use defaults
	if policy.MaxAge == 0 && policy.MaxCount == 0 &&
		policy.KeepDaily == 0 && policy.KeepWeekly == 0 && policy.KeepMonthly == 0 {
		policy = retention.DefaultPolicy()
	}

	if pruneDryRun {
		fmt.Println("Dry run mode - no backups will be deleted\n")

		result, err := retention.Preview(ctx, backend, policy)
		if err != nil {
			return err
		}

		fmt.Printf("Backups to keep (%d):\n", len(result.Kept))
		for _, k := range result.Kept {
			fmt.Printf("  - %s\n", k)
		}

		fmt.Printf("\nBackups to delete (%d):\n", len(result.Deleted))
		for _, d := range result.Deleted {
			fmt.Printf("  - %s\n", d)
		}
	} else {
		fmt.Println("Applying retention policy...\n")

		result, err := retention.Apply(ctx, backend, policy)
		if err != nil {
			return err
		}

		fmt.Printf("Kept %d backup(s)\n", len(result.Kept))
		fmt.Printf("Deleted %d backup(s)\n", len(result.Deleted))

		for _, d := range result.Deleted {
			fmt.Printf("  - %s\n", d)
		}

		if len(result.Errors) > 0 {
			fmt.Printf("\nErrors:\n")
			for _, e := range result.Errors {
				fmt.Printf("  - %v\n", e)
			}
		}
	}

	return nil
}

func init() {
	pruneCmd.Flags().BoolVar(&pruneDryRun, "dry-run", false, "Preview what would be deleted")
	pruneCmd.Flags().IntVar(&pruneMaxAge, "max-age", 0, "Maximum age in days (0 = use config)")
	pruneCmd.Flags().IntVar(&pruneMaxCount, "max-count", 0, "Maximum number of backups to keep")
	pruneCmd.Flags().IntVar(&pruneKeepDaily, "keep-daily", 0, "Number of daily backups to keep")
	pruneCmd.Flags().IntVar(&pruneKeepWeekly, "keep-weekly", 0, "Number of weekly backups to keep")
	pruneCmd.Flags().IntVar(&pruneKeepMonthly, "keep-monthly", 0, "Number of monthly backups to keep")
}
