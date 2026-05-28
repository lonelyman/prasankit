// Package authdbrepo implements the auth domain ports against Postgres via GORM.
// GORM model structs live here; the domain structs in modules/auth are pure.
package authdbrepo

import (
	"time"

	"github.com/google/uuid"
)

// accountModel is the GORM model for user_accounts.
type accountModel struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey"`
	PrimaryEmail      string     `gorm:"column:primary_email;not null"`
	DisplayName       string     `gorm:"column:display_name;not null"`
	AccountStatusCode string     `gorm:"column:account_status_code;not null"`
	FailedLoginCount  int        `gorm:"column:failed_login_count;not null;default:0"`
	LockedUntil       *time.Time `gorm:"column:locked_until"`
	LastLoginAt       *time.Time `gorm:"column:last_login_at"`
	CreatedAt         time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;not null"`
	DeletedAt         *time.Time `gorm:"column:deleted_at"`
}

func (accountModel) TableName() string { return "user_accounts" }

// identityModel is the GORM model for auth_identities.
type identityModel struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserAccountID    uuid.UUID  `gorm:"type:uuid;column:user_account_id;not null"`
	IdentityTypeCode string     `gorm:"column:identity_type_code;not null"`
	Email            string     `gorm:"column:email"`
	EmailVerifiedAt  *time.Time `gorm:"column:email_verified_at"`
	PasswordHash     string     `gorm:"column:password_hash"`
	LastUsedAt       *time.Time `gorm:"column:last_used_at"`
	CreatedAt        time.Time  `gorm:"column:created_at;not null"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;not null"`
	DeletedAt        *time.Time `gorm:"column:deleted_at"`
}

func (identityModel) TableName() string { return "auth_identities" }

// securityEventModel is the GORM model for security_events.
type securityEventModel struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserAccountID *uuid.UUID `gorm:"type:uuid;column:user_account_id"`
	EventType     string     `gorm:"column:event_type;not null"`
	Severity      string     `gorm:"column:severity;not null;default:'info'"`
	IPAddress     string     `gorm:"column:ip_address;type:inet"`
	UserAgent     string     `gorm:"column:user_agent"`
	Metadata      string     `gorm:"column:metadata;type:jsonb;not null;default:'{}'"`
	CreatedAt     time.Time  `gorm:"column:created_at;not null"`
}

func (securityEventModel) TableName() string { return "security_events" }

// emailVerificationTokenModel is the GORM model for auth_email_verification_tokens.
type emailVerificationTokenModel struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey"`
	AuthIdentityID uuid.UUID  `gorm:"type:uuid;column:auth_identity_id;not null"`
	TokenHash      string     `gorm:"column:token_hash;not null"`
	ExpiresAt      time.Time  `gorm:"column:expires_at;not null"`
	UsedAt         *time.Time `gorm:"column:used_at"`
	RevokedAt      *time.Time `gorm:"column:revoked_at"`
	CreatedAt      time.Time  `gorm:"column:created_at;not null"`
}

func (emailVerificationTokenModel) TableName() string {
	return "auth_email_verification_tokens"
}
