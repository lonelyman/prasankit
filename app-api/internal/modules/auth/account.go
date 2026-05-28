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
// Login requires an active (verified) account. Suspended, disabled, deleted, and
// pending_verification accounts are not allowed to log in.
func (a *Account) IsLoginAllowed() bool {
	return a.AccountStatusCode == AccountStatusActive
}
