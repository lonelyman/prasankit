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
	ID                uuid.UUID
	Email             string
	PasswordHash      string
	Status            UserAccountStatus
	EmailVerifiedAt   *time.Time
	PasswordChangedAt *time.Time
	LastLoginAt       *time.Time
	FailedLoginCount  int
	LockedUntil       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
	DeletedBy         *uuid.UUID
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
