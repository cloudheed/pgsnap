// Package verify provides backup verification utilities.
package verify

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/cloudheed/pgsnap/internal/compress"
	"github.com/cloudheed/pgsnap/internal/crypto"
	"github.com/cloudheed/pgsnap/internal/storage"
)

// Result contains the verification result.
type Result struct {
	BackupID       string
	StorageKey     string
	Size           int64
	Readable       bool
	Decryptable    bool
	Decompressible bool
	ValidFormat    bool
	Checksum       string
	Error          error
}

// Options configures the verification.
type Options struct {
	Backend            storage.Backend
	StorageKey         string
	DecryptionPassword string
	ExpectedChecksum   string // Optional: verify against known checksum
}

// Run verifies a backup's integrity.
func Run(ctx context.Context, opts Options) (*Result, error) {
	result := &Result{
		StorageKey: opts.StorageKey,
		BackupID:   extractID(opts.StorageKey),
	}

	// Get backup from storage
	rc, err := opts.Backend.Get(ctx, opts.StorageKey)
	if err != nil {
		result.Error = fmt.Errorf("failed to read backup: %w", err)
		return result, nil
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		result.Error = fmt.Errorf("failed to read data: %w", err)
		return result, nil
	}

	result.Size = int64(len(data))
	result.Readable = true

	// Decrypt if encrypted
	isEncrypted := strings.HasSuffix(opts.StorageKey, ".enc")
	if isEncrypted {
		if opts.DecryptionPassword == "" {
			result.Error = fmt.Errorf("decryption password required")
			return result, nil
		}

		decrypted, err := crypto.DecryptWithPassword(data, opts.DecryptionPassword)
		if err != nil {
			result.Error = fmt.Errorf("decryption failed: %w", err)
			return result, nil
		}
		data = decrypted
		result.Decryptable = true
	} else {
		result.Decryptable = true // Not encrypted, so trivially "decryptable"
	}

	// Decompress if compressed
	isCompressed := strings.HasSuffix(strings.TrimSuffix(opts.StorageKey, ".enc"), ".gz")
	if isCompressed {
		decompressed, err := compress.DecompressBytes(data)
		if err != nil {
			result.Error = fmt.Errorf("decompression failed: %w", err)
			return result, nil
		}
		data = decompressed
		result.Decompressible = true
	} else {
		result.Decompressible = true // Not compressed, so trivially "decompressible"
	}

	// Calculate checksum of raw dump
	checksum := sha256.Sum256(data)
	result.Checksum = hex.EncodeToString(checksum[:])

	// Verify checksum if expected value provided
	if opts.ExpectedChecksum != "" && result.Checksum != opts.ExpectedChecksum {
		result.Error = fmt.Errorf("checksum mismatch: expected %s, got %s", opts.ExpectedChecksum, result.Checksum)
		return result, nil
	}

	// Validate pg_dump format (custom format starts with "PGDMP")
	if len(data) >= 5 && string(data[:5]) == "PGDMP" {
		result.ValidFormat = true
	} else if len(data) > 0 && (data[0] == '-' || bytes.HasPrefix(data, []byte("--"))) {
		// Plain SQL format starts with comments
		result.ValidFormat = true
	} else if len(data) == 0 {
		result.Error = fmt.Errorf("backup is empty")
		return result, nil
	} else {
		// Try to detect if it looks like SQL
		if bytes.Contains(data[:min(1000, len(data))], []byte("CREATE")) ||
			bytes.Contains(data[:min(1000, len(data))], []byte("INSERT")) {
			result.ValidFormat = true
		} else {
			result.Error = fmt.Errorf("unrecognized backup format")
			return result, nil
		}
	}

	return result, nil
}

// IsValid returns true if all verification checks passed.
func (r *Result) IsValid() bool {
	return r.Readable && r.Decryptable && r.Decompressible && r.ValidFormat && r.Error == nil
}

func extractID(key string) string {
	id := key
	for _, ext := range []string{".enc", ".gz", ".dump"} {
		id = strings.TrimSuffix(id, ext)
	}
	return id
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
