package authrepo

import (
	"context"
	"errors"
	"time"

	"prasankit-api/internal/modules/auth"
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

func (r *Repository) CreateUserAccount(context.Context, *auth.UserAccount) error {
	return ErrNotImplemented
}

func (r *Repository) CreateAuthSession(context.Context, *auth.AuthSession) error {
	return ErrNotImplemented
}

func (r *Repository) CreateLoginAttempt(ctx context.Context, attempt *auth.LoginAttempt) error {
	if attempt.ID == uuid.Nil {
		id, err := ids.NewUUID()
		if err != nil {
			return err
		}
		attempt.ID = id
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

func (r *Repository) CreateSecurityEvent(context.Context, *auth.SecurityEvent) error {
	return ErrNotImplemented
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
