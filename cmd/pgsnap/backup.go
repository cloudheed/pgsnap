// Package main provides the pgsnap CLI entrypoint.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudheed/pgsnap/internal/backup"
	"github.com/cloudheed/pgsnap/internal/crypto"
	"github.com/cloudheed/pgsnap/internal/pg"
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
	backend, err := createBackend()
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

	// Handle encryption key
	if opts.Encrypt {
		key, err := getEncryptionKey()
		if err != nil {
			return err
		}
		opts.EncryptionKey = key
	}

	fmt.Printf("Starting backup of database '%s'...\n", pgConfig.Database)

	result, err := backup.Run(ctx, opts)
	if err != nil {
		return err
	}

	fmt.Printf("Backup completed successfully!\n")
	fmt.Printf("  ID:         %s\n", result.ID)
	fmt.Printf("  Database:   %s\n", result.Database)
	fmt.Printf("  Size:       %d bytes\n", result.Size)
	fmt.Printf("  Compressed: %v\n", result.Compressed)
	fmt.Printf("  Encrypted:  %v\n", result.Encrypted)
	fmt.Printf("  Duration:   %s\n", result.CompletedAt.Sub(result.StartedAt))

	return nil
}

func createBackend() (storage.Backend, error) {
	switch cfg.Storage.Type {
	case "local":
		return storage.NewLocalBackend(cfg.Storage.Local.Path)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Storage.Type)
	}
}

func getEncryptionKey() ([]byte, error) {
	password := os.Getenv("PGSNAP_ENCRYPTION_PASSWORD")
	if password == "" {
		return nil, fmt.Errorf("PGSNAP_ENCRYPTION_PASSWORD environment variable required for encryption")
	}

	key, _, err := crypto.DeriveKey(password, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}

	return key, nil
}

func init() {
	// Backup-specific flags can be added here
}
