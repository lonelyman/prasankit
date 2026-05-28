package authdbrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"prasankit-api/internal/modules/auth"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AccountRepo implements auth.AccountRepository against Postgres via GORM.
type AccountRepo struct {
	db *gorm.DB
}

// NewAccountRepo constructs an AccountRepo holding only the base *gorm.DB.
func NewAccountRepo(db *gorm.DB) *AccountRepo {
	return &AccountRepo{db: db}
}

// CreateWithIdentity atomically inserts a user_account + auth_identity in a single tx.
// A unique constraint violation on either table is mapped to auth.ErrEmailTaken.
func (r *AccountRepo) CreateWithIdentity(ctx context.Context, account auth.Account, identity auth.Identity) error {
	accountM := accountToModel(account)
	identityM := identityToModel(identity)

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&accountM).Error; err != nil {
			return err
		}
		if err := tx.Create(&identityM).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if isUniqueViolation(err) {
			return auth.ErrEmailTaken
		}
		return fmt.Errorf("create account with identity: %w", err)
	}
	return nil
}

// FindByID loads an account by its PK. Returns nil, nil when not found.
func (r *AccountRepo) FindByID(ctx context.Context, id uuid.UUID) (*auth.Account, error) {
	var m accountModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find account by id: %w", err)
	}
	a := modelToAccount(m)
	return &a, nil
}

// IncrementFailedLogin performs a single atomic UPDATE to increment failed_login_count
// and conditionally set locked_until when the threshold is reached.
func (r *AccountRepo) IncrementFailedLogin(ctx context.Context, accountID uuid.UUID, threshold int, lockDuration time.Duration) error {
	sql := `UPDATE user_accounts
		SET
			failed_login_count = failed_login_count + 1,
			locked_until = CASE
				WHEN failed_login_count + 1 >= ?
				THEN now() + (? * interval '1 second')
				ELSE locked_until
			END,
			updated_at = now()
		WHERE id = ? AND deleted_at IS NULL`

	err := r.db.WithContext(ctx).Exec(sql, threshold, int(lockDuration.Seconds()), accountID).Error
	if err != nil {
		return fmt.Errorf("increment failed login: %w", err)
	}
	return nil
}

// ResetFailedLogin clears failed_login_count + locked_until and sets last_login_at.
func (r *AccountRepo) ResetFailedLogin(ctx context.Context, accountID uuid.UUID, now time.Time) error {
	err := r.db.WithContext(ctx).
		Model(&accountModel{}).
		Where("id = ? AND deleted_at IS NULL", accountID).
		Updates(map[string]any{
			"failed_login_count": 0,
			"locked_until":       nil,
			"last_login_at":      now,
			"updated_at":         now,
		}).Error
	if err != nil {
		return fmt.Errorf("reset failed login: %w", err)
	}
	return nil
}

// isUniqueViolation detects Postgres unique constraint violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}
