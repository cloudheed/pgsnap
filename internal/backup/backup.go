// Package backup handles PostgreSQL backup operations.
package backup

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/cloudheed/pgsnap/internal/compress"
	"github.com/cloudheed/pgsnap/internal/crypto"
	"github.com/cloudheed/pgsnap/internal/pg"
	"github.com/cloudheed/pgsnap/internal/storage"
)

// Backup represents a backup operation result.
type Backup struct {
	ID           string    `json:"id"`
	Database     string    `json:"database"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at"`
	Size         int64     `json:"size"`
	Compressed   bool      `json:"compressed"`
	Encrypted    bool      `json:"encrypted"`
	StorageKey   string    `json:"storage_key"`
	PgVersion    string    `json:"pg_version,omitempty"`
}

// Options configures the backup operation.
type Options struct {
	PgConfig       *pg.Config
	DumpOptions    pg.DumpOptions
	Backend        storage.Backend
	Compress       bool
	Encrypt        bool
	EncryptionKey  []byte // 32 bytes for AES-256
}

// Run executes a backup operation.
func Run(ctx context.Context, opts Options) (*Backup, error) {
	if err := pg.CheckTools(); err != nil {
		return nil, err
	}

	startedAt := time.Now()
	backupID := generateID(startedAt)

	// Get PostgreSQL version
	pgVersion, _ := pg.Version(ctx)

	// Run pg_dump to buffer
	var dumpBuf bytes.Buffer
	if err := pg.Dump(ctx, opts.PgConfig, opts.DumpOptions, &dumpBuf); err != nil {
		return nil, fmt.Errorf("dump failed: %w", err)
	}

	data := dumpBuf.Bytes()

	// Compress if enabled
	if opts.Compress {
		compressed, err := compress.CompressBytes(data, compress.DefaultCompression)
		if err != nil {
			return nil, fmt.Errorf("compression failed: %w", err)
		}
		data = compressed
	}

	// Encrypt if enabled
	if opts.Encrypt {
		if len(opts.EncryptionKey) != crypto.KeySize {
			return nil, fmt.Errorf("encryption key must be %d bytes", crypto.KeySize)
		}
		encrypted, err := crypto.EncryptBytes(data, opts.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("encryption failed: %w", err)
		}
		data = encrypted
	}

	// Store backup
	storageKey := buildStorageKey(backupID, opts.Compress, opts.Encrypt)
	if err := opts.Backend.Put(ctx, storageKey, bytes.NewReader(data), int64(len(data))); err != nil {
		return nil, fmt.Errorf("storage failed: %w", err)
	}

	completedAt := time.Now()

	return &Backup{
		ID:          backupID,
		Database:    opts.PgConfig.Database,
		StartedAt:   startedAt,
		CompletedAt: completedAt,
		Size:        int64(len(data)),
		Compressed:  opts.Compress,
		Encrypted:   opts.Encrypt,
		StorageKey:  storageKey,
		PgVersion:   pgVersion,
	}, nil
}

func generateID(t time.Time) string {
	return t.UTC().Format("20060102-150405")
}

func buildStorageKey(id string, compressed, encrypted bool) string {
	key := id + ".dump"
	if compressed {
		key += ".gz"
	}
	if encrypted {
		key += ".enc"
	}
	return key
}
