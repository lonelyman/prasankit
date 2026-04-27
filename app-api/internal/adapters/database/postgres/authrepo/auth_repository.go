package authrepo

import (
	"context"
	"errors"
	"time"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/pkg/dbtypes"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrNotImplemented = errors.New("auth repository method is not implemented")

type Repository struct {
	db *gorm.DB
}

type userAccountRow struct {
	ID               uuid.UUID  `gorm:"column:id;type:uuid"`
	PrimaryEmail     string     `gorm:"column:primary_email"`
	Status           string     `gorm:"column:status"`
	LastLoginAt      *time.Time `gorm:"column:last_login_at"`
	FailedLoginCount int        `gorm:"column:failed_login_count"`
	LockedUntil      *time.Time `gorm:"column:locked_until"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
	DeletedAt        *time.Time `gorm:"column:deleted_at"`
	DeletedBy        *uuid.UUID `gorm:"column:deleted_by"`
}

func (userAccountRow) TableName() string {
	return "user_accounts"
}

type authIdentityRow struct {
	ID                uuid.UUID  `gorm:"column:id;type:uuid"`
	UserAccountID     uuid.UUID  `gorm:"column:user_account_id"`
	IdentityType      string     `gorm:"column:identity_type"`
	Provider          string     `gorm:"column:provider"`
	ProviderUserID    *string    `gorm:"column:provider_user_id"`
	Email             string     `gorm:"column:email"`
	EmailVerifiedAt   *time.Time `gorm:"column:email_verified_at"`
	PasswordHash      string     `gorm:"column:password_hash"`
	PasswordChangedAt *time.Time `gorm:"column:password_changed_at"`
	LastUsedAt        *time.Time `gorm:"column:last_used_at"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
	DeletedAt         *time.Time `gorm:"column:deleted_at"`
}

func (authIdentityRow) TableName() string {
	return "auth_identities"
}

type loginAttemptRow struct {
	ID            uuid.UUID  `gorm:"column:id;type:uuid"`
	UserAccountID *uuid.UUID `gorm:"column:user_account_id"`
	Email         string     `gorm:"column:email"`
	Success       bool       `gorm:"column:success"`
	FailureReason string     `gorm:"column:failure_reason"`
	IPAddress     string     `gorm:"column:ip_address"`
	UserAgent     string     `gorm:"column:user_agent"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
}

func (loginAttemptRow) TableName() string {
	return "auth_login_attempts"
}

type securityEventRow struct {
	ID            uuid.UUID     `gorm:"column:id;type:uuid"`
	UserAccountID *uuid.UUID    `gorm:"column:user_account_id"`
	EventType     string        `gorm:"column:event_type"`
	Severity      string        `gorm:"column:severity"`
	IPAddress     *string       `gorm:"column:ip_address"`
	UserAgent     string        `gorm:"column:user_agent"`
	MetadataJSON  dbtypes.JSONB `gorm:"column:metadata_json;type:jsonb"`
	CreatedAt     time.Time     `gorm:"column:created_at"`
}

func (securityEventRow) TableName() string {
	return "security_events"
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) FindUserAccountByEmail(ctx context.Context, email string) (*auth.UserAccount, error) {
	var row userAccountRow
	err := r.db.WithContext(ctx).
		Table("user_accounts").
		Select("user_accounts.*").
		Joins("JOIN auth_identities ON auth_identities.user_account_id = user_accounts.id").
		Where("auth_identities.identity_type = ?", string(auth.AuthIdentityTypeEmailPassword)).
		Where("auth_identities.provider = ?", string(auth.AuthProviderEmail)).
		Where("auth_identities.email = ?", email).
		Where("auth_identities.deleted_at IS NULL").
		Where("user_accounts.deleted_at IS NULL").
		First(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, auth.ErrUserAccountNotFound
	}
	if err != nil {
		return nil, err
	}

	return row.toDomain(), nil
}

func (r *Repository) CreateUserAccount(ctx context.Context, account *auth.UserAccount) error {
	if err := ensureUUID(&account.ID); err != nil {
		return err
	}
	if account.Status == "" {
		account.Status = auth.UserAccountStatusPendingVerification
	}
	if account.CreatedAt.IsZero() {
		account.CreatedAt = time.Now().UTC()
	}
	if account.UpdatedAt.IsZero() {
		account.UpdatedAt = account.CreatedAt
	}

	row := userAccountRow{
		ID:               account.ID,
		PrimaryEmail:     account.PrimaryEmail,
		Status:           string(account.Status),
		LastLoginAt:      account.LastLoginAt,
		FailedLoginCount: account.FailedLoginCount,
		LockedUntil:      account.LockedUntil,
		CreatedAt:        account.CreatedAt,
		UpdatedAt:        account.UpdatedAt,
		DeletedAt:        account.DeletedAt,
		DeletedBy:        account.DeletedBy,
	}

	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}

	account.ID = row.ID
	account.CreatedAt = row.CreatedAt
	account.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *Repository) CreateAuthIdentity(ctx context.Context, identity *auth.AuthIdentity) error {
	if err := ensureUUID(&identity.ID); err != nil {
		return err
	}
	if identity.IdentityType == "" {
		identity.IdentityType = auth.AuthIdentityTypeEmailPassword
	}
	if identity.Provider == "" {
		identity.Provider = auth.AuthProviderEmail
	}
	if identity.CreatedAt.IsZero() {
		identity.CreatedAt = time.Now().UTC()
	}
	if identity.UpdatedAt.IsZero() {
		identity.UpdatedAt = identity.CreatedAt
	}

	row := authIdentityRow{
		ID:                identity.ID,
		UserAccountID:     identity.UserAccountID,
		IdentityType:      string(identity.IdentityType),
		Provider:          string(identity.Provider),
		ProviderUserID:    stringPtrOrNil(identity.ProviderUserID),
		Email:             identity.Email,
		EmailVerifiedAt:   identity.EmailVerifiedAt,
		PasswordHash:      identity.PasswordHash,
		PasswordChangedAt: identity.PasswordChangedAt,
		LastUsedAt:        identity.LastUsedAt,
		CreatedAt:         identity.CreatedAt,
		UpdatedAt:         identity.UpdatedAt,
		DeletedAt:         identity.DeletedAt,
	}

	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}

	identity.ID = row.ID
	identity.CreatedAt = row.CreatedAt
	identity.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *Repository) CreateAuthSession(context.Context, *auth.AuthSession) error {
	return ErrNotImplemented
}

func (r *Repository) CreateLoginAttempt(ctx context.Context, attempt *auth.LoginAttempt) error {
	if err := ensureUUID(&attempt.ID); err != nil {
		return err
	}
	if attempt.CreatedAt.IsZero() {
		attempt.CreatedAt = time.Now().UTC()
	}

	row := loginAttemptRow{
		ID:            attempt.ID,
		UserAccountID: attempt.UserAccountID,
		Email:         attempt.Email,
		Success:       attempt.Success,
		FailureReason: attempt.FailureReason,
		IPAddress:     attempt.IPAddress,
		UserAgent:     attempt.UserAgent,
		CreatedAt:     attempt.CreatedAt,
	}

	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}

	attempt.ID = row.ID
	attempt.CreatedAt = row.CreatedAt
	return nil
}

func (r *Repository) CreateSecurityEvent(ctx context.Context, event *auth.SecurityEvent) error {
	if err := ensureUUID(&event.ID); err != nil {
		return err
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if event.Severity == "" {
		event.Severity = auth.SecurityEventSeverityInfo
	}
	if event.MetadataJSON == nil {
		event.MetadataJSON = map[string]any{}
	}

	row := securityEventRow{
		ID:            event.ID,
		UserAccountID: event.UserAccountID,
		EventType:     event.EventType,
		Severity:      string(event.Severity),
		IPAddress:     stringPtrOrNil(event.IPAddress),
		UserAgent:     event.UserAgent,
		MetadataJSON:  dbtypes.NewJSONB(event.MetadataJSON),
		CreatedAt:     event.CreatedAt,
	}

	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	return nil
}

func ensureUUID(id *uuid.UUID) error {
	if *id != uuid.Nil {
		return nil
	}
	newID, err := ids.NewUUID()
	if err != nil {
		return err
	}
	*id = newID
	return nil
}

func stringPtrOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func (r userAccountRow) toDomain() *auth.UserAccount {
	return &auth.UserAccount{
		ID:               r.ID,
		PrimaryEmail:     r.PrimaryEmail,
		Status:           auth.UserAccountStatus(r.Status),
		LastLoginAt:      r.LastLoginAt,
		FailedLoginCount: r.FailedLoginCount,
		LockedUntil:      r.LockedUntil,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
		DeletedAt:        r.DeletedAt,
		DeletedBy:        r.DeletedBy,
	}
}
