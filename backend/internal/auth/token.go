package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// NewToken returns a URL-safe random token and its SHA-256 hex hash for storage.
func NewToken(nbytes int) (raw string, hash string, err error) {
	if nbytes < 16 {
		nbytes = 32
	}
	buf := make([]byte, nbytes)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	hash = HashToken(raw)
	return raw, hash, nil
}

// HashToken returns a hex-encoded SHA-256 of the raw token.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
