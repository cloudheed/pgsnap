package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudheed/pgsnap/internal/pg"
	"github.com/cloudheed/pgsnap/internal/restore"
	"github.com/cloudheed/pgsnap/internal/storage"
	"github.com/spf13/cobra"
)

var restoreCmd = &cobra.Command{
	Use:   "restore <backup-id>",
	Short: "Restore a database backup",
	Long: `Restore a database from a previously created backup.

Specify the backup ID (e.g., 20240101-120000) to restore.`,
	Args: cobra.ExactArgs(1),
	RunE: runRestore,
}

func runRestore(cmd *cobra.Command, args []string) error {
	backupID := args[0]

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create storage backend
	backend, err := createBackend()
	if err != nil {
		return fmt.Errorf("failed to create storage backend: %w", err)
	}

	// Find the backup file
	storageKey, err := findBackupKey(ctx, backend, backupID)
	if err != nil {
		return err
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

	// Build restore options
	opts := restore.Options{
		PgConfig:       pgConfig,
		RestoreOptions: pg.DefaultRestoreOptions(),
		Backend:        backend,
		StorageKey:     storageKey,
	}

	// Handle decryption key if backup is encrypted
	if isEncrypted(storageKey) {
		key, err := getEncryptionKey()
		if err != nil {
			return err
		}
		opts.DecryptionKey = key
	}

	fmt.Printf("Restoring backup '%s' to database '%s'...\n", backupID, pgConfig.Database)

	if err := restore.Run(ctx, opts); err != nil {
		return err
	}

	fmt.Println("Restore completed successfully!")

	return nil
}

func findBackupKey(ctx context.Context, backend interface {
	Stat(context.Context, string) (*storage.ObjectInfo, error)
}, backupID string) (string, error) {
	// Try common extensions in order of preference
	extensions := []string{
		".dump.gz.enc",
		".dump.gz",
		".dump.enc",
		".dump",
	}

	for _, ext := range extensions {
		key := backupID + ext
		if _, err := backend.Stat(ctx, key); err == nil {
			return key, nil
		}
	}

	return "", fmt.Errorf("backup not found: %s", backupID)
}

func isEncrypted(key string) bool {
	return len(key) > 4 && key[len(key)-4:] == ".enc"
}

func init() {
	// Restore-specific flags can be added here
}
