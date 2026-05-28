// Package auth contains the domain and service layer for authentication.
// Domain structs and interfaces are pure — no gorm tags, no fiber, no redis imports.
package auth

// Account status codes — FK by code, must match account_statuses.code seed.
const (
	AccountStatusPendingVerification = "pending_verification"
	AccountStatusActive              = "active"
	AccountStatusSuspended           = "suspended"
	AccountStatusDisabled            = "disabled"
	AccountStatusDeleted             = "deleted"
)

// Identity type codes — FK by code, must match auth_identity_types.code seed.
const (
	IdentityTypeEmailPassword = "email_password"
)

// Security event types (machine codes, §4.4 — no DB FK, controlled by domain constant).
const (
	SecurityEventAccountCreated             = "account.created"
	SecurityEventLoginSuccess               = "login.success"
	SecurityEventLoginFailure               = "login.failure"
	SecurityEventAccountLocked              = "account.locked"
	SecurityEventLogout                     = "logout"
	SecurityEventEmailVerificationRequested = "email.verification_requested"
	SecurityEventEmailVerified              = "email.verified"
	SecurityEventPasswordResetRequested     = "password.reset_requested"
	SecurityEventPasswordReset              = "password.reset"
	SecurityEventSessionRevoked             = "session.revoked"
)

// Security event severity codes (§4.4).
const (
	SecuritySeverityInfo     = "info"
	SecuritySeverityWarning  = "warning"
	SecuritySeverityCritical = "critical"
)

// Lockout policy (named constants, no new env vars per spec).
const (
	LockoutThreshold  = 5  // consecutive failures before lockout
	SessionTTL        = 24 // hours
	SessionCookieName = "prasankit_session"
)
