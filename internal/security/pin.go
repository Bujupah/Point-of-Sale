// Package security owns password/PIN hashing, session tokens, and the
// permission-checking middleware shared by every API route. It is the only
// package allowed to touch bcrypt or generate session tokens.
package security

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPIN hashes a numeric PIN (or password) for storage. Never store or log
// the raw value.
func HashPIN(pin string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash pin: %w", err)
	}
	return string(hash), nil
}

// VerifyPIN reports whether pin matches the stored bcrypt hash.
func VerifyPIN(hash, pin string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pin)) == nil
}

// NewToken generates a random opaque session token (not a JWT — nothing here
// needs to be verified offline by a third party, so an unguessable random
// string plus a server-side sessions table is simpler and smaller).
func NewToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
