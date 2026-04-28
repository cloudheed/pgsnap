package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudheed/pgsnap/internal/storage"
	"github.com/cloudheed/pgsnap/internal/verify"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify <backup-id>",
	Short: "Verify backup integrity",
	Long: `Verify the integrity of a backup.

This checks that the backup file is readable, can be decrypted (if encrypted),
can be decompressed (if compressed), and has a valid pg_dump format.`,
	Args: cobra.ExactArgs(1),
	RunE: runVerify,
}

func runVerify(cmd *cobra.Command, args []string) error {
	backupID := args[0]

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Create storage backend
	backend, err := createBackend(ctx)
	if err != nil {
		return fmt.Errorf("failed to create storage backend: %w", err)
	}

	// Find the backup file
	storageKey, err := findBackupKey(ctx, backend, backupID)
	if err != nil {
		return err
	}

	// Get decryption password if needed
	var password string
	if isEncrypted(storageKey) {
		password = os.Getenv("PGSNAP_ENCRYPTION_PASSWORD")
		if password == "" {
			return fmt.Errorf("PGSNAP_ENCRYPTION_PASSWORD environment variable required for encrypted backup")
		}
	}

	fmt.Printf("Verifying backup '%s'...\n", backupID)

	result, err := verify.Run(ctx, verify.Options{
		Backend:            backend,
		StorageKey:         storageKey,
		DecryptionPassword: password,
	})
	if err != nil {
		return err
	}

	fmt.Printf("\nVerification Results:\n")
	fmt.Printf("  Backup ID:      %s\n", result.BackupID)
	fmt.Printf("  Storage Key:    %s\n", result.StorageKey)
	fmt.Printf("  Size:           %s\n", formatSize(result.Size))
	fmt.Printf("  Readable:       %s\n", boolToStatus(result.Readable))
	fmt.Printf("  Decryptable:    %s\n", boolToStatus(result.Decryptable))
	fmt.Printf("  Decompressible: %s\n", boolToStatus(result.Decompressible))
	fmt.Printf("  Valid Format:   %s\n", boolToStatus(result.ValidFormat))
	fmt.Printf("  Checksum:       %s\n", result.Checksum[:16]+"...")

	if result.Error != nil {
		fmt.Printf("\n  Error: %v\n", result.Error)
		return fmt.Errorf("verification failed")
	}

	if result.IsValid() {
		fmt.Printf("\nBackup is valid!\n")
	} else {
		fmt.Printf("\nBackup verification failed!\n")
		return fmt.Errorf("backup is invalid")
	}

	return nil
}

func boolToStatus(b bool) string {
	if b {
		return "OK"
	}
	return "FAILED"
}

func findBackupKey(ctx context.Context, backend storage.Backend, backupID string) (string, error) {
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
	// Verify-specific flags can be added here
}
