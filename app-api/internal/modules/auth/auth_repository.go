package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrUserAccountNotFound = errors.New("user account not found")
var ErrAuthSessionExpiresAtRequired = errors.New("auth session expires_at is required")
var ErrEmailVerificationTokenExpiresAtRequired = errors.New("email verification token expires_at is required")
var ErrEmailAlreadyRegistered = errors.New("email is already registered")

type Repository interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context, repo Repository) error) error
	FindUserAccountByEmail(ctx context.Context, email string) (*UserAccount, error)
	CreateUserAccount(ctx context.Context, account *UserAccount) error
	CreateAuthIdentity(ctx context.Context, identity *AuthIdentity) error
	RevokeActiveEmailVerificationTokens(ctx context.Context, authIdentityID uuid.UUID) error
	CreateEmailVerificationToken(ctx context.Context, token *EmailVerificationToken) error
	CreateAuthSession(ctx context.Context, session *AuthSession) error
	CreateLoginAttempt(ctx context.Context, attempt *LoginAttempt) error
	CreateSecurityEvent(ctx context.Context, event *SecurityEvent) error
}
