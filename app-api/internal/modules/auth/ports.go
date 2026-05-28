package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AccountRepository defines storage operations on user_accounts.
type AccountRepository interface {
	// CreateWithIdentity atomically creates a user_account + auth_identity in a single transaction.
	CreateWithIdentity(ctx context.Context, account Account, identity Identity) error

	// FindByID loads an account by PK.
	FindByID(ctx context.Context, id uuid.UUID) (*Account, error)

	// IncrementFailedLogin atomically increments failed_login_count and applies
	// locked_until when the threshold is reached. Must use a single UPDATE — no read-modify-write.
	IncrementFailedLogin(ctx context.Context, accountID uuid.UUID, threshold int, lockDuration time.Duration) error

	// ResetFailedLogin clears failed_login_count and locked_until, and sets last_login_at.
	ResetFailedLogin(ctx context.Context, accountID uuid.UUID, now time.Time) error
}

// IdentityRepository defines storage operations on auth_identities.
type IdentityRepository interface {
	// FindByEmail returns the identity (email_password type) for the given email.
	// Returns nil, nil when not found.
	FindByEmail(ctx context.Context, email string) (*Identity, error)

	// FindByID returns the identity for the given ID.
	// Returns nil, nil when not found.
	FindByID(ctx context.Context, id uuid.UUID) (*Identity, error)
}

// EmailVerificationTokenRepository defines storage operations on
// auth_email_verification_tokens.
type EmailVerificationTokenRepository interface {
	// Create inserts a new verification token row.
	Create(ctx context.Context, token EmailVerificationToken) error

	// RevokeActiveByIdentity sets revoked_at=now on all currently-active tokens
	// of the identity (used_at IS NULL AND revoked_at IS NULL AND expires_at > now).
	RevokeActiveByIdentity(ctx context.Context, identityID uuid.UUID, now time.Time) error

	// FindByTokenHash returns the token row for the given hash regardless of state
	// (nil, nil if absent). Caller derives state via IsActive.
	FindByTokenHash(ctx context.Context, tokenHash string) (*EmailVerificationToken, error)

	// ConfirmTx atomically consumes the token and verifies the account.
	// It sets used_at on the token, email_verified_at on the identity, and
	// conditionally flips account_status_code to active when it is
	// pending_verification. Returns ErrTokenExpired when the token was already
	// consumed by a concurrent caller.
	ConfirmTx(ctx context.Context, tokenID, identityID, accountID uuid.UUID, now time.Time) error
}

// SecurityEventRepository defines append-only logging to security_events.
type SecurityEventRepository interface {
	// Log records a security event. Failures should not abort the main request.
	Log(ctx context.Context, event SecurityEvent) error
}

// SecurityEvent is the domain struct for a security_events row.
type SecurityEvent struct {
	ID            uuid.UUID
	UserAccountID *uuid.UUID // nullable (login fail for unknown email)
	EventType     string
	Severity      string
	IPAddress     string
	UserAgent     string
	Metadata      map[string]any
}

// SessionStore defines Redis session operations.
type SessionStore interface {
	// Create stores a session record keyed by the token hash with given TTL.
	Create(ctx context.Context, hash string, record SessionRecord, ttl time.Duration) error

	// Get retrieves a session record by hash. Returns nil, nil, nil when not found.
	Get(ctx context.Context, hash string) (*SessionRecord, error)

	// Delete removes a session by hash. Idempotent — missing key is not an error.
	Delete(ctx context.Context, hash string) error
}

// SessionRecord holds the data stored in Redis per session (§6, D16 — no role/membership).
type SessionRecord struct {
	AccountID  uuid.UUID `json:"account_id"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
}
