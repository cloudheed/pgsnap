package storage

import "errors"

// Common storage errors.
var (
	// ErrNotFound is returned when a requested object does not exist.
	ErrNotFound = errors.New("object not found")

	// ErrInvalidKey is returned when a key contains invalid characters.
	ErrInvalidKey = errors.New("invalid key")
)
