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
	ID                uuid.UUID  `gorm:"column:id;type:uuid"`
	Email             string     `gorm:"column:email"`
	PasswordHash      string     `gorm:"column:password_hash"`
	Status            string     `gorm:"column:status"`
	EmailVerifiedAt   *time.Time `gorm:"column:email_verified_at"`
	PasswordChangedAt *time.Time `gorm:"column:password_changed_at"`
	LastLoginAt       *time.Time `gorm:"column:last_login_at"`
	FailedLoginCount  int        `gorm:"column:failed_login_count"`
	LockedUntil       *time.Time `gorm:"column:locked_until"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
	DeletedAt         *time.Time `gorm:"column:deleted_at"`
	DeletedBy         *uuid.UUID `gorm:"column:deleted_by"`
}

func (userAccountRow) TableName() string {
	return "user_accounts"
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
		Where("email = ? AND deleted_at IS NULL", email).
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
		ID:                account.ID,
		Email:             account.Email,
		PasswordHash:      account.PasswordHash,
		Status:            string(account.Status),
		EmailVerifiedAt:   account.EmailVerifiedAt,
		PasswordChangedAt: account.PasswordChangedAt,
		LastLoginAt:       account.LastLoginAt,
		FailedLoginCount:  account.FailedLoginCount,
		LockedUntil:       account.LockedUntil,
		CreatedAt:         account.CreatedAt,
		UpdatedAt:         account.UpdatedAt,
		DeletedAt:         account.DeletedAt,
		DeletedBy:         account.DeletedBy,
	}

	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}

	account.ID = row.ID
	account.CreatedAt = row.CreatedAt
	account.UpdatedAt = row.UpdatedAt
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
		ID:                r.ID,
		Email:             r.Email,
		PasswordHash:      r.PasswordHash,
		Status:            auth.UserAccountStatus(r.Status),
		EmailVerifiedAt:   r.EmailVerifiedAt,
		PasswordChangedAt: r.PasswordChangedAt,
		LastLoginAt:       r.LastLoginAt,
		FailedLoginCount:  r.FailedLoginCount,
		LockedUntil:       r.LockedUntil,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
		DeletedAt:         r.DeletedAt,
		DeletedBy:         r.DeletedBy,
	}
}
