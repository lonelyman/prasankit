package auth

import (
	"context"
	"errors"
)

var ErrUserAccountNotFound = errors.New("user account not found")

type Repository interface {
	FindUserAccountByEmail(ctx context.Context, email string) (*UserAccount, error)
	CreateUserAccount(ctx context.Context, account *UserAccount) error
	CreateAuthIdentity(ctx context.Context, identity *AuthIdentity) error
	CreateAuthSession(ctx context.Context, session *AuthSession) error
	CreateLoginAttempt(ctx context.Context, attempt *LoginAttempt) error
	CreateSecurityEvent(ctx context.Context, event *SecurityEvent) error
}
