package storage

import (
	"context"
	"io"
	"time"
)

// ObjectInfo contains metadata about a stored object.
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	ContentType  string
	Metadata     map[string]string
}

// Backend defines the interface for storage backends.
// Implementations must be safe for concurrent use.
type Backend interface {
	// Put stores data from the reader under the given key.
	// If an object with the same key exists, it will be overwritten.
	Put(ctx context.Context, key string, r io.Reader, size int64) error

	// Get retrieves the object stored under the given key.
	// The caller is responsible for closing the returned ReadCloser.
	// Returns ErrNotFound if the key does not exist.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// List returns information about objects matching the given prefix.
	// An empty prefix lists all objects.
	List(ctx context.Context, prefix string) ([]ObjectInfo, error)

	// Delete removes the object stored under the given key.
	// Returns ErrNotFound if the key does not exist.
	Delete(ctx context.Context, key string) error

	// Stat returns information about the object stored under the given key.
	// Returns ErrNotFound if the key does not exist.
	Stat(ctx context.Context, key string) (*ObjectInfo, error)
}
