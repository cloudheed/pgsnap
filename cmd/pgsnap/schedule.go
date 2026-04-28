package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudheed/pgsnap/internal/backup"
	"github.com/cloudheed/pgsnap/internal/pg"
	"github.com/cloudheed/pgsnap/internal/retention"
	"github.com/cloudheed/pgsnap/internal/scheduler"
	"github.com/spf13/cobra"
)

var scheduleInterval string

var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Run scheduled backups",
	Long: `Run the backup scheduler in the foreground.

This will run backups at the specified interval and apply retention policies.
Use with a process manager (systemd, Docker, etc.) for production deployments.`,
	RunE: runSchedule,
}

func runSchedule(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Parse interval
	interval, err := time.ParseDuration(scheduleInterval)
	if err != nil {
		return fmt.Errorf("invalid interval: %w", err)
	}

	fmt.Printf("Starting backup scheduler (interval: %s)...\n", interval)
	fmt.Println("Press Ctrl+C to stop.\n")

	sched := scheduler.New()

	// Add backup job
	err = sched.Add("backup", scheduler.Every(interval), func(jobCtx context.Context) error {
		return runScheduledBackup(jobCtx)
	})
	if err != nil {
		return err
	}

	sched.Start()
	defer sched.Stop()

	// Run first backup immediately
	fmt.Println("Running initial backup...")
	if err := runScheduledBackup(ctx); err != nil {
		fmt.Printf("Initial backup failed: %v\n", err)
	}

	// Wait for shutdown signal
	<-ctx.Done()
	fmt.Println("\nShutting down scheduler...")

	return nil
}

func runScheduledBackup(ctx context.Context) error {
	backend, err := createBackend(ctx)
	if err != nil {
		return fmt.Errorf("failed to create storage backend: %w", err)
	}

	pgConfig := &pg.Config{
		Host:     cfg.Postgres.Host,
		Port:     cfg.Postgres.Port,
		User:     cfg.Postgres.User,
		Password: cfg.Postgres.Password,
		Database: cfg.Postgres.Database,
		SSLMode:  cfg.Postgres.SSLMode,
	}

	opts := backup.Options{
		PgConfig:    pgConfig,
		DumpOptions: pg.DefaultDumpOptions(),
		Backend:     backend,
		Compress:    cfg.Backup.Compress,
		Encrypt:     cfg.Backup.Encrypt,
	}

	if opts.Encrypt {
		password := os.Getenv("PGSNAP_ENCRYPTION_PASSWORD")
		if password == "" {
			return fmt.Errorf("PGSNAP_ENCRYPTION_PASSWORD required for encryption")
		}
		opts.EncryptionPassword = password
	}

	startTime := time.Now()
	fmt.Printf("[%s] Starting backup of '%s'...\n", startTime.Format(time.RFC3339), pgConfig.Database)

	result, err := backup.Run(ctx, opts)
	if err != nil {
		return err
	}

	fmt.Printf("[%s] Backup completed: %s (%s)\n",
		time.Now().Format(time.RFC3339),
		result.ID,
		formatSize(result.Size),
	)

	// Apply retention policy
	if cfg.Backup.RetentionDays > 0 {
		policy := retention.Policy{
			MaxAge: time.Duration(cfg.Backup.RetentionDays) * 24 * time.Hour,
		}

		retResult, err := retention.Apply(ctx, backend, policy)
		if err != nil {
			fmt.Printf("[%s] Retention policy failed: %v\n", time.Now().Format(time.RFC3339), err)
		} else if len(retResult.Deleted) > 0 {
			fmt.Printf("[%s] Deleted %d old backup(s)\n", time.Now().Format(time.RFC3339), len(retResult.Deleted))
		}
	}

	return nil
}

func init() {
	scheduleCmd.Flags().StringVar(&scheduleInterval, "interval", "1h", "Backup interval (e.g., 1h, 6h, 24h)")
}
