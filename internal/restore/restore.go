// Package restore handles PostgreSQL restore operations.
package restore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cloudheed/pgsnap/internal/compress"
	"github.com/cloudheed/pgsnap/internal/crypto"
	"github.com/cloudheed/pgsnap/internal/pg"
	"github.com/cloudheed/pgsnap/internal/storage"
)

// Options configures the restore operation.
type Options struct {
	PgConfig           *pg.Config
	RestoreOptions     pg.RestoreOptions
	Backend            storage.Backend
	StorageKey         string
	DecryptionPassword string
}

// Run executes a restore operation.
func Run(ctx context.Context, opts Options) error {
	if err := pg.CheckTools(); err != nil {
		return err
	}

	// Fetch backup from storage
	rc, err := opts.Backend.Get(ctx, opts.StorageKey)
	if err != nil {
		return fmt.Errorf("failed to get backup: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	// Decrypt if encrypted
	if strings.HasSuffix(opts.StorageKey, ".enc") {
		if opts.DecryptionPassword == "" {
			return fmt.Errorf("decryption password required for encrypted backup")
		}
		decrypted, err := crypto.DecryptWithPassword(data, opts.DecryptionPassword)
		if err != nil {
			return fmt.Errorf("decryption failed: %w", err)
		}
		data = decrypted
	}

	// Decompress if compressed
	if strings.HasSuffix(strings.TrimSuffix(opts.StorageKey, ".enc"), ".gz") {
		decompressed, err := compress.DecompressBytes(data)
		if err != nil {
			return fmt.Errorf("decompression failed: %w", err)
		}
		data = decompressed
	}

	// Run pg_restore
	if err := pg.Restore(ctx, opts.PgConfig, opts.RestoreOptions, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	return nil
}
