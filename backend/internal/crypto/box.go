// Package crypto provides application-level encryption (AES-256-GCM)
// with a versioned key id to allow rotation.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

const (
	// CurrentKeyID is written into new ciphertexts.
	CurrentKeyID byte = 1
	keySize           = 32
	nonceSize         = 12
)

var (
	ErrInvalidKey    = errors.New("encryption key must be 32 bytes")
	ErrInvalidBlob   = errors.New("invalid ciphertext blob")
	ErrUnknownKeyID  = errors.New("unknown encryption key id")
	ErrCipherTooShort = errors.New("ciphertext too short")
)

// Box encrypts/decrypts sensitive fields (IBAN, addresses, …).
type Box struct {
	keys    map[byte][]byte
	current byte
}

// NewBox creates a box with a single current key (id = CurrentKeyID).
func NewBox(key []byte) (*Box, error) {
	if len(key) != keySize {
		return nil, ErrInvalidKey
	}
	k := make([]byte, keySize)
	copy(k, key)
	return &Box{
		keys:    map[byte][]byte{CurrentKeyID: k},
		current: CurrentKeyID,
	}, nil
}

// NewBoxFromBase64 decodes a standard base64 32-byte key.
func NewBoxFromBase64(encoded string) (*Box, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode ENCRYPTION_KEY: %w", err)
	}
	return NewBox(raw)
}

// Encrypt returns keyID || nonce || ciphertext.
func (b *Box) Encrypt(plaintext []byte) ([]byte, error) {
	key, ok := b.keys[b.current]
	if !ok {
		return nil, ErrUnknownKeyID
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	out := make([]byte, 1+nonceSize, 1+nonceSize+len(plaintext)+gcm.Overhead())
	out[0] = b.current
	copy(out[1:], nonce)
	out = gcm.Seal(out, nonce, plaintext, nil)
	return out, nil
}

// Decrypt opens a blob produced by Encrypt (any known key id).
func (b *Box) Decrypt(blob []byte) ([]byte, error) {
	if len(blob) < 1+nonceSize+1 {
		return nil, ErrCipherTooShort
	}
	keyID := blob[0]
	key, ok := b.keys[keyID]
	if !ok {
		return nil, ErrUnknownKeyID
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := blob[1 : 1+nonceSize]
	ciphertext := blob[1+nonceSize:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrInvalidBlob
	}
	return plain, nil
}

// EncryptString is a convenience wrapper around Encrypt.
func (b *Box) EncryptString(s string) ([]byte, error) {
	return b.Encrypt([]byte(s))
}

// DecryptString decrypts and returns a UTF-8 string.
func (b *Box) DecryptString(blob []byte) (string, error) {
	plain, err := b.Decrypt(blob)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
