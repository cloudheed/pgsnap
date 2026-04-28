package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalBackend(t *testing.T) {
	// Create temp directory for tests
	tmpDir, err := os.MkdirTemp("", "pgsnap-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	backend, err := NewLocalBackend(tmpDir)
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	ctx := context.Background()

	t.Run("Put and Get", func(t *testing.T) {
		tests := []struct {
			name    string
			key     string
			content []byte
		}{
			{
				name:    "simple key",
				key:     "test.txt",
				content: []byte("hello world"),
			},
			{
				name:    "nested key",
				key:     "path/to/file.txt",
				content: []byte("nested content"),
			},
			{
				name:    "empty content",
				key:     "empty.txt",
				content: []byte{},
			},
			{
				name:    "binary content",
				key:     "binary.bin",
				content: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Put
				err := backend.Put(ctx, tt.key, bytes.NewReader(tt.content), int64(len(tt.content)))
				if err != nil {
					t.Fatalf("Put failed: %v", err)
				}

				// Get
				rc, err := backend.Get(ctx, tt.key)
				if err != nil {
					t.Fatalf("Get failed: %v", err)
				}
				defer rc.Close()

				got, err := io.ReadAll(rc)
				if err != nil {
					t.Fatalf("ReadAll failed: %v", err)
				}

				if !bytes.Equal(got, tt.content) {
					t.Errorf("content mismatch: got %v, want %v", got, tt.content)
				}
			})
		}
	})

	t.Run("Get non-existent", func(t *testing.T) {
		_, err := backend.Get(ctx, "nonexistent.txt")
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("Stat", func(t *testing.T) {
		key := "stat-test.txt"
		content := []byte("stat test content")

		err := backend.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		info, err := backend.Stat(ctx, key)
		if err != nil {
			t.Fatalf("Stat failed: %v", err)
		}

		if info.Key != key {
			t.Errorf("key mismatch: got %q, want %q", info.Key, key)
		}

		if info.Size != int64(len(content)) {
			t.Errorf("size mismatch: got %d, want %d", info.Size, len(content))
		}
	})

	t.Run("Stat non-existent", func(t *testing.T) {
		_, err := backend.Stat(ctx, "nonexistent.txt")
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("List", func(t *testing.T) {
		// Create a fresh directory for list tests
		listDir := filepath.Join(tmpDir, "list-test")
		listBackend, err := NewLocalBackend(listDir)
		if err != nil {
			t.Fatalf("failed to create backend: %v", err)
		}

		// Create test files
		files := map[string][]byte{
			"a.txt":         []byte("a"),
			"b.txt":         []byte("b"),
			"prefix/c.txt":  []byte("c"),
			"prefix/d.txt":  []byte("d"),
			"other/e.txt":   []byte("e"),
		}

		for key, content := range files {
			err := listBackend.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
			if err != nil {
				t.Fatalf("Put failed for %s: %v", key, err)
			}
		}

		tests := []struct {
			name     string
			prefix   string
			expected int
		}{
			{
				name:     "list all",
				prefix:   "",
				expected: 5,
			},
			{
				name:     "list with prefix",
				prefix:   "prefix/",
				expected: 2,
			},
			{
				name:     "list with non-matching prefix",
				prefix:   "nonexistent/",
				expected: 0,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				objects, err := listBackend.List(ctx, tt.prefix)
				if err != nil {
					t.Fatalf("List failed: %v", err)
				}

				if len(objects) != tt.expected {
					t.Errorf("count mismatch: got %d, want %d", len(objects), tt.expected)
				}
			})
		}
	})

	t.Run("Delete", func(t *testing.T) {
		key := "delete-test.txt"
		content := []byte("to be deleted")

		err := backend.Put(ctx, key, bytes.NewReader(content), int64(len(content)))
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		// Verify it exists
		_, err = backend.Stat(ctx, key)
		if err != nil {
			t.Fatalf("Stat failed: %v", err)
		}

		// Delete
		err = backend.Delete(ctx, key)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify it's gone
		_, err = backend.Stat(ctx, key)
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound after delete, got %v", err)
		}
	})

	t.Run("Delete non-existent", func(t *testing.T) {
		err := backend.Delete(ctx, "nonexistent-delete.txt")
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("Invalid keys", func(t *testing.T) {
		tests := []struct {
			name string
			key  string
		}{
			{"empty key", ""},
			{"path traversal", "../escape.txt"},
			{"nested path traversal", "foo/../../../etc/passwd"},
			{"absolute path", "/etc/passwd"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := backend.Put(ctx, tt.key, bytes.NewReader([]byte("test")), 4)
				if err != ErrInvalidKey {
					t.Errorf("Put: expected ErrInvalidKey, got %v", err)
				}

				_, err = backend.Get(ctx, tt.key)
				if err != ErrInvalidKey {
					t.Errorf("Get: expected ErrInvalidKey, got %v", err)
				}

				_, err = backend.Stat(ctx, tt.key)
				if err != ErrInvalidKey {
					t.Errorf("Stat: expected ErrInvalidKey, got %v", err)
				}

				err = backend.Delete(ctx, tt.key)
				if err != ErrInvalidKey {
					t.Errorf("Delete: expected ErrInvalidKey, got %v", err)
				}
			})
		}
	})

	t.Run("Overwrite existing", func(t *testing.T) {
		key := "overwrite.txt"
		content1 := []byte("original content")
		content2 := []byte("new content")

		// Put original
		err := backend.Put(ctx, key, bytes.NewReader(content1), int64(len(content1)))
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}

		// Overwrite
		err = backend.Put(ctx, key, bytes.NewReader(content2), int64(len(content2)))
		if err != nil {
			t.Fatalf("Put (overwrite) failed: %v", err)
		}

		// Verify new content
		rc, err := backend.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		defer rc.Close()

		got, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("ReadAll failed: %v", err)
		}

		if !bytes.Equal(got, content2) {
			t.Errorf("content mismatch after overwrite: got %q, want %q", got, content2)
		}
	})
}
