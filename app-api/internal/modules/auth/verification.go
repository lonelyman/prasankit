package auth

import (
	"time"

	"github.com/google/uuid"
)

// EmailVerificationToken is the pure domain representation of
// auth_email_verification_tokens. State is derived from timestamps — there is
// no status column (mirrors the invitation pattern, D25).
type EmailVerificationToken struct {
	ID             uuid.UUID
	AuthIdentityID uuid.UUID
	TokenHash      string
	ExpiresAt      time.Time
	UsedAt         *time.Time
	RevokedAt      *time.Time
	CreatedAt      time.Time
}

// IsActive returns true when the token has not been used, not been revoked,
// and has not yet expired.
func (t *EmailVerificationToken) IsActive(now time.Time) bool {
	return t.UsedAt == nil && t.RevokedAt == nil && t.ExpiresAt.After(now)
}
