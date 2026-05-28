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

// VerificationTokenRepo implements auth.EmailVerificationTokenRepository against
// Postgres via GORM.
type VerificationTokenRepo struct {
	db *gorm.DB
}

// NewVerificationTokenRepo constructs a VerificationTokenRepo holding only the
// base *gorm.DB.
func NewVerificationTokenRepo(db *gorm.DB) *VerificationTokenRepo {
	return &VerificationTokenRepo{db: db}
}

// Create inserts a new verification token row.
func (r *VerificationTokenRepo) Create(ctx context.Context, token auth.EmailVerificationToken) error {
	m := emailVerificationTokenToModel(token)
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("create verification token: %w", err)
	}
	return nil
}

// RevokeActiveByIdentity sets revoked_at = now on all currently-active tokens
// for the identity (used_at IS NULL AND revoked_at IS NULL AND expires_at > now).
func (r *VerificationTokenRepo) RevokeActiveByIdentity(ctx context.Context, identityID uuid.UUID, now time.Time) error {
	err := r.db.WithContext(ctx).Exec(
		`UPDATE auth_email_verification_tokens
		 SET revoked_at = ?
		 WHERE auth_identity_id = ?
		   AND used_at IS NULL
		   AND revoked_at IS NULL
		   AND expires_at > ?`,
		now, identityID, now,
	).Error
	if err != nil {
		return fmt.Errorf("revoke active verification tokens: %w", err)
	}
	return nil
}

// FindByTokenHash returns the token row for the hash regardless of state.
// Returns nil, nil when not found. Caller derives state via IsActive.
func (r *VerificationTokenRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*auth.EmailVerificationToken, error) {
	var m emailVerificationTokenModel
	err := r.db.WithContext(ctx).
		Where("token_hash = ?", tokenHash).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find verification token by hash: %w", err)
	}
	t := modelToEmailVerificationToken(m)
	return &t, nil
}

// ConfirmTx atomically consumes the verification token and verifies the account.
//
// Inside a single transaction it:
//  1. Guards the token consume with a WHERE used_at IS NULL AND revoked_at IS NULL
//     check — if 0 rows are affected a concurrent caller already won, so it returns
//     auth.ErrTokenExpired (the tx is rolled back automatically by GORM).
//  2. Sets email_verified_at on the identity row.
//  3. Conditionally flips account_status_code to active only when it is currently
//     pending_verification (idempotent; will not downgrade suspended/disabled accounts).
func (r *VerificationTokenRepo) ConfirmTx(ctx context.Context, tokenID, identityID, accountID uuid.UUID, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Step 1: guarded token consume (concurrency guard).
		res := tx.Exec(
			`UPDATE auth_email_verification_tokens
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

		// Step 2: set email_verified_at on the identity.
		if err := tx.Exec(
			`UPDATE auth_identities
			 SET email_verified_at = ?, updated_at = ?
			 WHERE id = ?`,
			now, now, identityID,
		).Error; err != nil {
			return err
		}

		// Step 3: conditional account flip — only from pending_verification to active.
		if err := tx.Exec(
			`UPDATE user_accounts
			 SET account_status_code = ?, updated_at = ?
			 WHERE id = ?
			   AND account_status_code = ?`,
			auth.AccountStatusActive, now, accountID, auth.AccountStatusPendingVerification,
		).Error; err != nil {
			return err
		}

		return nil
	})
}
