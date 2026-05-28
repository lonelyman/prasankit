package auth

import (
	"time"

	"github.com/google/uuid"
)

// Identity is the pure domain representation of auth_identities.
type Identity struct {
	ID                uuid.UUID
	UserAccountID     uuid.UUID
	IdentityTypeCode  string
	Email             string
	EmailVerifiedAt   *time.Time
	PasswordHash      string
	PasswordChangedAt *time.Time
	LastUsedAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}
