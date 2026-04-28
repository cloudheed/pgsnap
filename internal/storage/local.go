package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalBackend implements Backend using the local filesystem.
type LocalBackend struct {
	basePath string
}

// NewLocalBackend creates a new LocalBackend with the given base path.
// The base path will be created if it does not exist.
func NewLocalBackend(basePath string) (*LocalBackend, error) {
	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	if err := os.MkdirAll(absPath, 0750); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &LocalBackend{basePath: absPath}, nil
}

// Put stores data from the reader under the given key.
func (b *LocalBackend) Put(ctx context.Context, key string, r io.Reader, size int64) error {
	if err := validateKey(key); err != nil {
		return err
	}

	fullPath := b.keyToPath(key)

	// Ensure parent directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to temp file first, then rename for atomicity
	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on error
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	// Copy data to temp file
	if _, err := io.Copy(tmpFile, r); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write data: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, fullPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	success = true
	return nil
}

// Get retrieves the object stored under the given key.
func (b *LocalBackend) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	fullPath := b.keyToPath(key)

	f, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return f, nil
}

// List returns information about objects matching the given prefix.
func (b *LocalBackend) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	var objects []ObjectInfo

	err := filepath.Walk(b.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Convert path to key
		relPath, err := filepath.Rel(b.basePath, path)
		if err != nil {
			return err
		}
		key := filepath.ToSlash(relPath)

		// Check prefix
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			return nil
		}

		objects = append(objects, ObjectInfo{
			Key:          key,
			Size:         info.Size(),
			LastModified: info.ModTime(),
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	return objects, nil
}

// Delete removes the object stored under the given key.
func (b *LocalBackend) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	fullPath := b.keyToPath(key)

	err := os.Remove(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// Stat returns information about the object stored under the given key.
func (b *LocalBackend) Stat(ctx context.Context, key string) (*ObjectInfo, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	fullPath := b.keyToPath(key)

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	return &ObjectInfo{
		Key:          key,
		Size:         info.Size(),
		LastModified: info.ModTime(),
	}, nil
}

// keyToPath converts a storage key to a filesystem path.
func (b *LocalBackend) keyToPath(key string) string {
	return filepath.Join(b.basePath, filepath.FromSlash(key))
}

// validateKey checks if a key is valid.
func validateKey(key string) error {
	if key == "" {
		return ErrInvalidKey
	}

	// Prevent path traversal
	if strings.Contains(key, "..") {
		return ErrInvalidKey
	}

	// Prevent absolute paths
	if filepath.IsAbs(key) {
		return ErrInvalidKey
	}

	return nil
}
