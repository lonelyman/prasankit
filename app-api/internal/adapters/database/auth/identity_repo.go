package authdbrepo

import (
	"context"
	"errors"
	"fmt"

	"prasankit-api/internal/modules/auth"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IdentityRepo implements auth.IdentityRepository against Postgres via GORM.
type IdentityRepo struct {
	db *gorm.DB
}

// NewIdentityRepo constructs an IdentityRepo holding only the base *gorm.DB.
func NewIdentityRepo(db *gorm.DB) *IdentityRepo {
	return &IdentityRepo{db: db}
}

// FindByEmail returns the email_password identity for the given email (CITEXT,
// case-insensitive in Postgres). Returns nil, nil when not found.
func (r *IdentityRepo) FindByEmail(ctx context.Context, email string) (*auth.Identity, error) {
	var m identityModel
	err := r.db.WithContext(ctx).
		Where("email = ? AND identity_type_code = ? AND deleted_at IS NULL", email, auth.IdentityTypeEmailPassword).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find identity by email: %w", err)
	}
	i := modelToIdentity(m)
	return &i, nil
}

// FindByID returns the identity for the given ID. Returns nil, nil when not found.
func (r *IdentityRepo) FindByID(ctx context.Context, id uuid.UUID) (*auth.Identity, error) {
	var m identityModel
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find identity by id: %w", err)
	}
	i := modelToIdentity(m)
	return &i, nil
}
