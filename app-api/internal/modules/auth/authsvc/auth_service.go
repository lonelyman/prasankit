package authsvc

import (
	"context"
	"errors"
	"net"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/pkg/securetoken"
)

const minPasswordLength = 8

var ErrInvalidEmail = errors.New("invalid email")
var ErrPasswordTooShort = errors.New("password is too short")
var ErrPasswordHasherRequired = errors.New("password hasher is required")
var ErrEmailSenderRequired = errors.New("email sender is required")
var ErrRateLimiterRequired = errors.New("rate limiter is required")
var ErrVerificationEmailRateLimited = errors.New("verification email rate limited")
var ErrVerificationEmailSendFailed = errors.New("verification email send failed")
var ErrVerificationConfigInvalid = errors.New("verification email config is invalid")

type PasswordHasher interface {
	Hash(password string) (string, error)
}

type EmailSender interface {
	SendVerificationEmail(ctx context.Context, toEmail string, verificationURL string) error
}

type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

type ServiceConfig struct {
	VerificationBaseURL  string
	VerificationTokenTTL time.Duration
	VerificationIPLimit  int
	VerificationIPWindow time.Duration
}

type RegisterEmailPasswordInput struct {
	Email     string
	Password  string
	IPAddress string
	UserAgent string
}

type RegisterEmailPasswordResult struct {
	Account               auth.UserAccount
	VerificationEmailSent bool
}

type Service struct {
	repository auth.Repository
	hasher     PasswordHasher
	email      EmailSender
	limiter    RateLimiter
	config     ServiceConfig
}

func NewService(repository auth.Repository, hasher PasswordHasher, email EmailSender, limiter RateLimiter, config ServiceConfig) *Service {
	return &Service{
		repository: repository,
		hasher:     hasher,
		email:      email,
		limiter:    limiter,
		config:     config,
	}
}

func (s *Service) RegisterEmailPassword(ctx context.Context, input RegisterEmailPasswordInput) (*RegisterEmailPasswordResult, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return nil, err
	}
	if len(input.Password) < minPasswordLength {
		return nil, ErrPasswordTooShort
	}
	if s.hasher == nil {
		return nil, ErrPasswordHasherRequired
	}
	if s.email == nil {
		return nil, ErrEmailSenderRequired
	}
	if s.limiter == nil {
		return nil, ErrRateLimiterRequired
	}
	if err := s.validateVerificationConfig(); err != nil {
		return nil, err
	}

	existingAccount, err := s.repository.FindUserAccountByEmail(ctx, email)
	if err == nil && existingAccount != nil {
		return nil, auth.ErrEmailAlreadyRegistered
	}
	if err != nil && !errors.Is(err, auth.ErrUserAccountNotFound) {
		return nil, err
	}

	allowed, err := s.limiter.Allow(ctx, verificationRateLimitKey(input.IPAddress), s.config.VerificationIPLimit, s.config.VerificationIPWindow)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrVerificationEmailRateLimited
	}

	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return nil, err
	}

	account := auth.UserAccount{
		PrimaryEmail: email,
	}
	var identity auth.AuthIdentity
	var verificationToken string

	err = s.repository.WithinTransaction(ctx, func(ctx context.Context, repo auth.Repository) error {
		if err := repo.CreateUserAccount(ctx, &account); err != nil {
			return err
		}

		identity = auth.AuthIdentity{
			UserAccountID: account.ID,
			IdentityType:  auth.AuthIdentityTypeEmailPassword,
			Provider:      auth.AuthProviderEmail,
			Email:         email,
			PasswordHash:  passwordHash,
		}
		if err := repo.CreateAuthIdentity(ctx, &identity); err != nil {
			return err
		}

		if err := repo.RevokeActiveEmailVerificationTokens(ctx, identity.ID); err != nil {
			return err
		}

		tokenValue, tokenHash, err := securetoken.New()
		if err != nil {
			return err
		}
		verificationToken = tokenValue
		now := time.Now().UTC()
		emailToken := auth.EmailVerificationToken{
			AuthIdentityID: identity.ID,
			TokenHash:      tokenHash,
			ExpiresAt:      now.Add(s.config.VerificationTokenTTL),
			CreatedAt:      now,
		}
		if err := repo.CreateEmailVerificationToken(ctx, &emailToken); err != nil {
			return err
		}

		event := auth.SecurityEvent{
			UserAccountID: &account.ID,
			EventType:     "auth.account_registered",
			Severity:      auth.SecurityEventSeverityInfo,
			IPAddress:     input.IPAddress,
			UserAgent:     input.UserAgent,
			MetadataJSON: map[string]any{
				"identity_type":                 string(auth.AuthIdentityTypeEmailPassword),
				"provider":                      string(auth.AuthProviderEmail),
				"email_verification_token_id":   emailToken.ID.String(),
				"email_verification_expires_at": emailToken.ExpiresAt.Format(time.RFC3339),
			},
		}
		return repo.CreateSecurityEvent(ctx, &event)
	})
	if err != nil {
		return nil, err
	}

	verificationURL, err := s.buildVerificationURL(verificationToken)
	if err != nil {
		return nil, err
	}
	if err := s.email.SendVerificationEmail(ctx, email, verificationURL); err != nil {
		return nil, errors.Join(ErrVerificationEmailSendFailed, err)
	}

	_ = s.repository.CreateSecurityEvent(ctx, &auth.SecurityEvent{
		UserAccountID: &account.ID,
		EventType:     "auth.email_verification_sent",
		Severity:      auth.SecurityEventSeverityInfo,
		IPAddress:     input.IPAddress,
		UserAgent:     input.UserAgent,
		MetadataJSON: map[string]any{
			"identity_id": identity.ID.String(),
		},
	})

	return &RegisterEmailPasswordResult{
		Account:               account,
		VerificationEmailSent: true,
	}, nil
}

func (s *Service) validateVerificationConfig() error {
	if s.config.VerificationBaseURL == "" {
		return ErrVerificationConfigInvalid
	}
	if s.config.VerificationTokenTTL <= 0 {
		return ErrVerificationConfigInvalid
	}
	if s.config.VerificationIPLimit < 1 {
		return ErrVerificationConfigInvalid
	}
	if s.config.VerificationIPWindow <= 0 {
		return ErrVerificationConfigInvalid
	}
	return nil
}

func (s *Service) buildVerificationURL(token string) (string, error) {
	verifyURL, err := url.Parse(s.config.VerificationBaseURL)
	if err != nil {
		return "", err
	}
	query := verifyURL.Query()
	query.Set("token", token)
	verifyURL.RawQuery = query.Encode()
	return verifyURL.String(), nil
}

func verificationRateLimitKey(ipAddress string) string {
	if parsed := net.ParseIP(strings.TrimSpace(ipAddress)); parsed != nil {
		return "rate:auth:verify_email:ip:" + parsed.String()
	}
	return "rate:auth:verify_email:ip:unknown"
}

func normalizeEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	if email == "" {
		return "", ErrInvalidEmail
	}

	parsed, err := mail.ParseAddress(email)
	if err != nil {
		return "", ErrInvalidEmail
	}
	if parsed.Address != email {
		return "", ErrInvalidEmail
	}
	return email, nil
}
