// Package compress provides compression and decompression utilities.
package compress

import (
	"compress/gzip"
	"fmt"
	"io"
)

// Level represents compression level.
type Level int

const (
	NoCompression      Level = gzip.NoCompression
	BestSpeed          Level = gzip.BestSpeed
	BestCompression    Level = gzip.BestCompression
	DefaultCompression Level = gzip.DefaultCompression
)

// GzipWriter wraps gzip.Writer for backup compression.
type GzipWriter struct {
	*gzip.Writer
}

// NewGzipWriter creates a new gzip compressing writer.
func NewGzipWriter(w io.Writer, level Level) (*GzipWriter, error) {
	gw, err := gzip.NewWriterLevel(w, int(level))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip writer: %w", err)
	}
	return &GzipWriter{Writer: gw}, nil
}

// GzipReader wraps gzip.Reader for backup decompression.
type GzipReader struct {
	*gzip.Reader
}

// NewGzipReader creates a new gzip decompressing reader.
func NewGzipReader(r io.Reader) (*GzipReader, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	return &GzipReader{Reader: gr}, nil
}

// CompressBytes compresses data using gzip.
func CompressBytes(data []byte, level Level) ([]byte, error) {
	var buf []byte
	w, err := NewGzipWriter((*bytesWriter)(&buf), level)
	if err != nil {
		return nil, err
	}

	if _, err := w.Write(data); err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf, nil
}

// DecompressBytes decompresses gzip data.
func DecompressBytes(data []byte) ([]byte, error) {
	r, err := NewGzipReader((*bytesReader)(&data))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return io.ReadAll(r)
}

// bytesWriter implements io.Writer for a byte slice.
type bytesWriter []byte

func (b *bytesWriter) Write(p []byte) (int, error) {
	*b = append(*b, p...)
	return len(p), nil
}

// bytesReader implements io.Reader for a byte slice.
type bytesReader []byte

func (b *bytesReader) Read(p []byte) (int, error) {
	if len(*b) == 0 {
		return 0, io.EOF
	}
	n := copy(p, *b)
	*b = (*b)[n:]
	return n, nil
}
