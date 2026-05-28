// Package securetoken generates and hashes opaque tokens used for sessions,
// email verification, and password reset (docs/04-data-model.md §2.6, §6).
package securetoken

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// New generates a cryptographically-random 32-byte token.
// raw is the base64url-encoded token (sent to the client / email).
// hash is the hex-encoded SHA-256 of raw (stored in the database).
func New() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", fmt.Errorf("securetoken: read random bytes: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	hash = Hash(raw)
	return raw, hash, nil
}

// Hash returns the hex-encoded SHA-256 of raw. Use this to derive the stored
// hash from a token the client presents (lookup by hash, never by raw value).
func Hash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum)
}
