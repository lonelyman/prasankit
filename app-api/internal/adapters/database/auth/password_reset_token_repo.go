package authdbrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"prasankit-api/internal/modules/auth"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PasswordResetTokenRepo implements auth.PasswordResetTokenRepository against
// Postgres via GORM.
type PasswordResetTokenRepo struct {
	db *gorm.DB
}

// NewPasswordResetTokenRepo constructs a PasswordResetTokenRepo holding only
// the base *gorm.DB.
func NewPasswordResetTokenRepo(db *gorm.DB) *PasswordResetTokenRepo {
	return &PasswordResetTokenRepo{db: db}
}

// Create inserts a new password reset token row.
func (r *PasswordResetTokenRepo) Create(ctx context.Context, token auth.PasswordResetToken) error {
	m := passwordResetTokenToModel(token)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("create password reset token: %w", err)
	}
	return nil
}

// RevokeActiveByIdentity sets revoked_at = now on all currently-active reset
// tokens for the identity (used_at IS NULL AND revoked_at IS NULL AND expires_at > now).
func (r *PasswordResetTokenRepo) RevokeActiveByIdentity(ctx context.Context, identityID uuid.UUID, now time.Time) error {
	err := r.db.WithContext(ctx).Exec(
		`UPDATE auth_password_reset_tokens
		 SET revoked_at = ?
		 WHERE auth_identity_id = ?
		   AND used_at IS NULL
		   AND revoked_at IS NULL
		   AND expires_at > ?`,
		now, identityID, now,
	).Error
	if err != nil {
		return fmt.Errorf("revoke active password reset tokens: %w", err)
	}
	return nil
}

// FindByTokenHash returns the token row for the hash regardless of state.
// Returns nil, nil when not found. Caller derives state via IsActive.
func (r *PasswordResetTokenRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*auth.PasswordResetToken, error) {
	var m passwordResetTokenModel
	err := r.db.WithContext(ctx).
		Where("token_hash = ?", tokenHash).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find password reset token by hash: %w", err)
	}
	t := modelToPasswordResetToken(m)
	return &t, nil
}

// ConfirmTx atomically consumes the password reset token, updates the identity
// password hash + password_changed_at, and clears the account lockout.
//
// Inside a single transaction it:
//  1. Guards the token consume with a WHERE used_at IS NULL AND revoked_at IS NULL
//     check — if 0 rows are affected a concurrent caller already won, so it returns
//     auth.ErrTokenExpired (the tx is rolled back automatically by GORM).
//  2. Sets password_hash and password_changed_at on the identity row via raw SQL
//     (password_changed_at is not in the GORM identity model — handled in DB only).
//  3. Clears failed_login_count and locked_until on the account (lockout recovery).
//     Does NOT change account_status_code — reset does not verify email.
func (r *PasswordResetTokenRepo) ConfirmTx(ctx context.Context, tokenID, identityID, accountID uuid.UUID, newPasswordHash string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Step 1: guarded token consume (concurrency guard).
		res := tx.Exec(
			`UPDATE auth_password_reset_tokens
			 SET used_at = ?
			 WHERE id = ?
			   AND used_at IS NULL
			   AND revoked_at IS NULL`,
			now, tokenID,
		)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// Another concurrent confirm already consumed this token.
			return auth.ErrTokenExpired
		}

		// Step 2: update password hash and password_changed_at on identity.
		if err := tx.Exec(
			`UPDATE auth_identities
			 SET password_hash = ?, password_changed_at = ?, updated_at = ?
			 WHERE id = ?`,
			newPasswordHash, now, now, identityID,
		).Error; err != nil {
			return err
		}

		// Step 3: clear account lockout (recovery path).
		// Does NOT change account_status_code — reset does not verify email.
		if err := tx.Exec(
			`UPDATE user_accounts
			 SET failed_login_count = 0, locked_until = NULL, updated_at = ?
			 WHERE id = ?`,
			now, accountID,
		).Error; err != nil {
			return err
		}

		return nil
	})
}
