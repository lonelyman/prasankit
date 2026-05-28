// Package passwordhash wraps bcrypt for password hashing and verification.
// Used by the auth service (docs/04-data-model.md §4.1 auth_identities.password_hash).
package passwordhash

import "golang.org/x/crypto/bcrypt"

// Hash returns the bcrypt hash of password at the default cost.
func Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Verify returns nil if hash matches password, or a non-nil error otherwise.
// The error is bcrypt.ErrMismatchedHashAndPassword on a wrong password.
func Verify(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
