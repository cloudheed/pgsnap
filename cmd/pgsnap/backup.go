// Package main provides the pgsnap CLI entrypoint.
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
	"github.com/cloudheed/pgsnap/internal/storage"
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create a database backup",
	Long: `Create a backup of the PostgreSQL database.

The backup will be stored in the configured storage backend with
optional compression and encryption.`,
	RunE: runBackup,
}

func runBackup(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create storage backend
	backend, err := createBackend(ctx)
	if err != nil {
		return fmt.Errorf("failed to create storage backend: %w", err)
	}

	// Build PostgreSQL config
	pgConfig := &pg.Config{
		Host:     cfg.Postgres.Host,
		Port:     cfg.Postgres.Port,
		User:     cfg.Postgres.User,
		Password: cfg.Postgres.Password,
		Database: cfg.Postgres.Database,
		SSLMode:  cfg.Postgres.SSLMode,
	}

	// Build backup options
	opts := backup.Options{
		PgConfig:    pgConfig,
		DumpOptions: pg.DefaultDumpOptions(),
		Backend:     backend,
		Compress:    cfg.Backup.Compress,
		Encrypt:     cfg.Backup.Encrypt,
	}

	// Handle encryption password
	if opts.Encrypt {
		password := os.Getenv("PGSNAP_ENCRYPTION_PASSWORD")
		if password == "" {
			return fmt.Errorf("PGSNAP_ENCRYPTION_PASSWORD environment variable required for encryption")
		}
		opts.EncryptionPassword = password
	}

	fmt.Printf("Starting backup of database '%s'...\n", pgConfig.Database)

	result, err := backup.Run(ctx, opts)
	if err != nil {
		return err
	}

	fmt.Printf("Backup completed successfully!\n")
	fmt.Printf("  ID:         %s\n", result.ID)
	fmt.Printf("  Database:   %s\n", result.Database)
	fmt.Printf("  Size:       %s\n", formatSize(result.Size))
	fmt.Printf("  Compressed: %v\n", result.Compressed)
	fmt.Printf("  Encrypted:  %v\n", result.Encrypted)
	fmt.Printf("  Checksum:   %s\n", result.Checksum[:16]+"...")
	fmt.Printf("  Duration:   %s\n", result.CompletedAt.Sub(result.StartedAt))

	// Apply retention policy if configured
	if cfg.Backup.RetentionDays > 0 {
		fmt.Printf("\nApplying retention policy (%d days)...\n", cfg.Backup.RetentionDays)

		policy := retention.Policy{
			MaxAge: time.Duration(cfg.Backup.RetentionDays) * 24 * time.Hour,
		}

		retResult, err := retention.Apply(ctx, backend, policy)
		if err != nil {
			fmt.Printf("Warning: retention policy failed: %v\n", err)
		} else if len(retResult.Deleted) > 0 {
			fmt.Printf("Deleted %d old backup(s)\n", len(retResult.Deleted))
		}
	}

	return nil
}

func createBackend(ctx context.Context) (storage.Backend, error) {
	switch cfg.Storage.Type {
	case "local":
		return storage.NewLocalBackend(cfg.Storage.Local.Path)
	case "s3":
		return storage.NewS3Backend(ctx, storage.S3Options{
			Bucket:    cfg.Storage.S3.Bucket,
			Region:    cfg.Storage.S3.Region,
			Endpoint:  cfg.Storage.S3.Endpoint,
			AccessKey: cfg.Storage.S3.AccessKey,
			SecretKey: cfg.Storage.S3.SecretKey,
		})
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Storage.Type)
	}
}

func init() {
	// Backup-specific flags can be added here
}
