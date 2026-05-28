package auth

import (
	"time"

	"github.com/google/uuid"
)

// Account is the pure domain representation of user_accounts.
// No gorm tags — mapping happens in adapters/database/auth.
type Account struct {
	ID                uuid.UUID
	PrimaryEmail      string
	DisplayName       string
	AccountStatusCode string
	FailedLoginCount  int
	LockedUntil       *time.Time
	LastLoginAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}

// IsLocked returns true if the account is currently under a failed-login lockout.
func (a *Account) IsLocked(now time.Time) bool {
	return a.LockedUntil != nil && a.LockedUntil.After(now)
}

// IsLoginAllowed returns true if the account status permits login attempts.
// Per spec: pending_verification and active are allowed (dispatch 3 adds email-verify enforcement).
func (a *Account) IsLoginAllowed() bool {
	return a.AccountStatusCode == AccountStatusPendingVerification ||
		a.AccountStatusCode == AccountStatusActive
}
