package auth

import (
	"time"

	"github.com/google/uuid"
)

type UserAccountStatus string

const (
	UserAccountStatusPendingVerification UserAccountStatus = "pending_verification"
	UserAccountStatusActive              UserAccountStatus = "active"
	UserAccountStatusSuspended           UserAccountStatus = "suspended"
	UserAccountStatusDisabled            UserAccountStatus = "disabled"
	UserAccountStatusDeleted             UserAccountStatus = "deleted"
)

type UserAccount struct {
	ID               uuid.UUID
	PrimaryEmail     string
	Status           UserAccountStatus
	LastLoginAt      *time.Time
	FailedLoginCount int
	LockedUntil      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
	DeletedBy        *uuid.UUID
}

type AuthIdentityType string

const (
	AuthIdentityTypeEmailPassword AuthIdentityType = "email_password"
	AuthIdentityTypeOAuth         AuthIdentityType = "oauth"
)

type AuthProvider string

const (
	AuthProviderEmail     AuthProvider = "email"
	AuthProviderGoogle    AuthProvider = "google"
	AuthProviderFacebook  AuthProvider = "facebook"
	AuthProviderMicrosoft AuthProvider = "microsoft"
	AuthProviderLine      AuthProvider = "line"
	AuthProviderGitHub    AuthProvider = "github"
)

type AuthIdentity struct {
	ID                uuid.UUID
	UserAccountID     uuid.UUID
	IdentityType      AuthIdentityType
	Provider          AuthProvider
	ProviderUserID    string
	Email             string
	EmailVerifiedAt   *time.Time
	PasswordHash      string
	PasswordChangedAt *time.Time
	LastUsedAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}

type AuthSessionStatus string

const (
	AuthSessionStatusActive  AuthSessionStatus = "active"
	AuthSessionStatusRevoked AuthSessionStatus = "revoked"
	AuthSessionStatusExpired AuthSessionStatus = "expired"
)

type AuthSession struct {
	ID             uuid.UUID
	UserAccountID  uuid.UUID
	SessionKeyHash string
	Status         AuthSessionStatus
	IPAddress      string
	UserAgent      string
	DeviceLabel    string
	CreatedAt      time.Time
	LastSeenAt     *time.Time
	ExpiresAt      time.Time
	RevokedAt      *time.Time
	RevokedReason  string
	MetadataJSON   map[string]any
}

type LoginAttempt struct {
	ID            uuid.UUID
	UserAccountID *uuid.UUID
	Email         string
	Success       bool
	FailureReason string
	IPAddress     string
	UserAgent     string
	CreatedAt     time.Time
}

type SecurityEventSeverity string

const (
	SecurityEventSeverityInfo     SecurityEventSeverity = "info"
	SecurityEventSeverityWarning  SecurityEventSeverity = "warning"
	SecurityEventSeverityCritical SecurityEventSeverity = "critical"
)

type SecurityEvent struct {
	ID            uuid.UUID
	UserAccountID *uuid.UUID
	EventType     string
	Severity      SecurityEventSeverity
	IPAddress     string
	UserAgent     string
	MetadataJSON  map[string]any
	CreatedAt     time.Time
}
