// Package crypto provides encryption and decryption utilities.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// KeySize is the AES-256 key size in bytes.
	KeySize = 32

	// NonceSize is the GCM nonce size in bytes.
	NonceSize = 12

	// SaltSize is the PBKDF2 salt size in bytes.
	SaltSize = 16

	// PBKDF2Iterations is the number of iterations for key derivation.
	PBKDF2Iterations = 100000

	// HeaderSize is salt + nonce size.
	HeaderSize = SaltSize + NonceSize

	// MagicBytes identifies encrypted files.
	MagicBytes uint32 = 0x50475345 // "PGSE" - PGSnap Encrypted
)

// Errors
var (
	ErrInvalidKey       = errors.New("invalid key size")
	ErrDecryptionFailed = errors.New("decryption failed")
	ErrInvalidFormat    = errors.New("invalid encrypted file format")
)

// DeriveKey derives a key from a password using PBKDF2.
// Returns the derived key and the salt used.
func DeriveKey(password string, salt []byte) ([]byte, []byte, error) {
	if salt == nil {
		salt = make([]byte, SaltSize)
		if _, err := rand.Read(salt); err != nil {
			return nil, nil, fmt.Errorf("failed to generate salt: %w", err)
		}
	}

	key := pbkdf2.Key([]byte(password), salt, PBKDF2Iterations, KeySize, sha256.New)
	return key, salt, nil
}

// EncryptWithPassword encrypts data with a password.
// Format: [magic:4][salt:16][nonce:12][ciphertext...]
func EncryptWithPassword(plaintext []byte, password string) ([]byte, error) {
	// Derive key with new salt
	key, salt, err := DeriveKey(password, nil)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// Build output: magic + salt + nonce + ciphertext
	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	result := make([]byte, 4+SaltSize+NonceSize+len(ciphertext))
	binary.BigEndian.PutUint32(result[0:4], MagicBytes)
	copy(result[4:4+SaltSize], salt)
	copy(result[4+SaltSize:4+SaltSize+NonceSize], nonce)
	copy(result[4+SaltSize+NonceSize:], ciphertext)

	return result, nil
}

// DecryptWithPassword decrypts data with a password.
// Reads salt from the encrypted data header.
func DecryptWithPassword(ciphertext []byte, password string) ([]byte, error) {
	minSize := 4 + SaltSize + NonceSize + 16 // magic + salt + nonce + min auth tag
	if len(ciphertext) < minSize {
		return nil, ErrInvalidFormat
	}

	// Verify magic bytes
	magic := binary.BigEndian.Uint32(ciphertext[0:4])
	if magic != MagicBytes {
		return nil, ErrInvalidFormat
	}

	// Extract salt and nonce
	salt := ciphertext[4 : 4+SaltSize]
	nonce := ciphertext[4+SaltSize : 4+SaltSize+NonceSize]
	encData := ciphertext[4+SaltSize+NonceSize:]

	// Derive key with extracted salt
	key, _, err := DeriveKey(password, salt)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := aead.Open(nil, nonce, encData, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// EncryptBytes encrypts data with the given key (legacy, for direct key use).
func EncryptBytes(plaintext, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptBytes decrypts data with the given key (legacy, for direct key use).
func DecryptBytes(ciphertext, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}

	if len(ciphertext) < NonceSize {
		return nil, ErrDecryptionFailed
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := ciphertext[:NonceSize]
	ciphertext = ciphertext[NonceSize:]

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// Encrypter wraps a writer with AES-256-GCM encryption.
type Encrypter struct {
	dst    io.Writer
	aead   cipher.AEAD
	nonce  []byte
	buf    []byte
	closed bool
}

// NewEncrypter creates a new encrypting writer.
// The key must be 32 bytes (AES-256).
// Writes salt and nonce to dst before encrypted data.
func NewEncrypter(dst io.Writer, key []byte) (*Encrypter, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Write nonce to output
	if _, err := dst.Write(nonce); err != nil {
		return nil, fmt.Errorf("failed to write nonce: %w", err)
	}

	return &Encrypter{
		dst:   dst,
		aead:  aead,
		nonce: nonce,
		buf:   make([]byte, 0, 64*1024), // 64KB buffer
	}, nil
}

// Write buffers data for encryption.
func (e *Encrypter) Write(p []byte) (int, error) {
	if e.closed {
		return 0, errors.New("encrypter is closed")
	}
	e.buf = append(e.buf, p...)
	return len(p), nil
}

// Close encrypts all buffered data and writes to destination.
func (e *Encrypter) Close() error {
	if e.closed {
		return nil
	}
	e.closed = true

	ciphertext := e.aead.Seal(nil, e.nonce, e.buf, nil)
	_, err := e.dst.Write(ciphertext)
	return err
}

// Decrypter wraps a reader with AES-256-GCM decryption.
type Decrypter struct {
	plaintext []byte
	offset    int
}

// NewDecrypter creates a new decrypting reader.
// Reads nonce from src, then decrypts the rest.
func NewDecrypter(src io.Reader, key []byte) (*Decrypter, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Read nonce
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(src, nonce); err != nil {
		return nil, fmt.Errorf("failed to read nonce: %w", err)
	}

	// Read ciphertext
	ciphertext, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("failed to read ciphertext: %w", err)
	}

	// Decrypt
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return &Decrypter{
		plaintext: plaintext,
		offset:    0,
	}, nil
}

// Read reads decrypted data.
func (d *Decrypter) Read(p []byte) (int, error) {
	if d.offset >= len(d.plaintext) {
		return 0, io.EOF
	}

	n := copy(p, d.plaintext[d.offset:])
	d.offset += n
	return n, nil
}
