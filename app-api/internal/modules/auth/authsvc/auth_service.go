package authsvc

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/pkg/ids"
	"prasankit-api/pkg/securetoken"

	"github.com/google/uuid"
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
var ErrVerificationTokenRequired = errors.New("verification token is required")
var ErrVerificationTokenInvalid = errors.New("verification token is invalid")
var ErrVerificationTokenExpired = errors.New("verification token is expired")
var ErrVerificationTokenAlreadyUsed = errors.New("verification token is already used")
var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrEmailNotVerified = errors.New("email is not verified")
var ErrAccountInactive = errors.New("account is inactive")
var ErrSessionStoreRequired = errors.New("session store is required")
var ErrSessionConfigInvalid = errors.New("session config is invalid")
var ErrSessionTokenRequired = errors.New("session token is required")
var ErrSessionNotFound = errors.New("session not found")
var ErrSessionInvalid = errors.New("session is invalid")
var ErrSessionExpired = errors.New("session is expired")

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash string, password string) error
}

type EmailSender interface {
	SendVerificationEmail(ctx context.Context, toEmail string, verificationURL string) error
}

type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

type SessionStore interface {
	Save(ctx context.Context, session SessionRecord, ttl time.Duration) error
	Get(ctx context.Context, sessionKeyHash string) (*SessionRecord, error)
	Delete(ctx context.Context, sessionKeyHash string) error
}

type SessionRecord struct {
	SessionID      uuid.UUID `json:"session_id"`
	UserAccountID  uuid.UUID `json:"user_account_id"`
	SessionKeyHash string    `json:"session_key_hash"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

type ServiceConfig struct {
	VerificationBaseURL  string
	VerificationTokenTTL time.Duration
	VerificationIPLimit  int
	VerificationIPWindow time.Duration
	SessionSecret        string
	SessionTTL           time.Duration
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

type VerifyEmailInput struct {
	Token     string
	IPAddress string
	UserAgent string
}

type VerifyEmailResult struct {
	Account auth.UserAccount
}

type ResendVerificationEmailInput struct {
	Email     string
	IPAddress string
	UserAgent string
}

type ResendVerificationEmailResult struct {
	VerificationEmailSent bool
}

type LoginEmailPasswordInput struct {
	Email     string
	Password  string
	IPAddress string
	UserAgent string
}

type LoginEmailPasswordResult struct {
	Account          auth.UserAccount
	SessionID        uuid.UUID
	SessionToken     string
	SessionExpiresAt time.Time
}

type CurrentAccountInput struct {
	SessionToken string
	IPAddress    string
	UserAgent    string
}

type CurrentAccountResult struct {
	Account auth.UserAccount
	Session SessionRecord
}

type LogoutCurrentSessionInput struct {
	SessionToken string
	IPAddress    string
	UserAgent    string
}

type LogoutCurrentSessionResult struct {
	Status string
}

type LogoutAllSessionsInput struct {
	SessionToken string
	IPAddress    string
	UserAgent    string
}

type LogoutAllSessionsResult struct {
	Status string
}

type Service struct {
	repository auth.Repository
	hasher     PasswordHasher
	email      EmailSender
	limiter    RateLimiter
	sessions   SessionStore
	config     ServiceConfig
}

func NewService(repository auth.Repository, hasher PasswordHasher, email EmailSender, limiter RateLimiter, sessions SessionStore, config ServiceConfig) *Service {
	return &Service{
		repository: repository,
		hasher:     hasher,
		email:      email,
		limiter:    limiter,
		sessions:   sessions,
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

func (s *Service) VerifyEmail(ctx context.Context, input VerifyEmailInput) (*VerifyEmailResult, error) {
	tokenValue := strings.TrimSpace(input.Token)
	if tokenValue == "" {
		return nil, ErrVerificationTokenRequired
	}

	emailToken, err := s.repository.FindEmailVerificationTokenByHash(ctx, securetoken.Hash(tokenValue))
	if errors.Is(err, auth.ErrEmailVerificationTokenNotFound) {
		return nil, ErrVerificationTokenInvalid
	}
	if err != nil {
		return nil, err
	}

	switch emailToken.Status {
	case auth.EmailVerificationTokenStatusActive:
	case auth.EmailVerificationTokenStatusUsed:
		return nil, ErrVerificationTokenAlreadyUsed
	default:
		return nil, ErrVerificationTokenInvalid
	}

	now := time.Now().UTC()
	if !emailToken.ExpiresAt.After(now) {
		return nil, ErrVerificationTokenExpired
	}

	identity, err := s.repository.FindAuthIdentityByID(ctx, emailToken.AuthIdentityID)
	if errors.Is(err, auth.ErrAuthIdentityNotFound) {
		return nil, ErrVerificationTokenInvalid
	}
	if err != nil {
		return nil, err
	}

	account := auth.UserAccount{
		ID:           identity.UserAccountID,
		PrimaryEmail: identity.Email,
		Status:       auth.UserAccountStatusActive,
		UpdatedAt:    now,
	}

	err = s.repository.WithinTransaction(ctx, func(ctx context.Context, repo auth.Repository) error {
		if err := repo.MarkEmailVerificationTokenUsed(ctx, emailToken.ID, now); err != nil {
			if errors.Is(err, auth.ErrEmailVerificationTokenNotFound) {
				return ErrVerificationTokenInvalid
			}
			return err
		}
		if err := repo.MarkAuthIdentityEmailVerified(ctx, identity.ID, now); err != nil {
			return err
		}
		if err := repo.ActivateUserAccount(ctx, identity.UserAccountID, now); err != nil {
			return err
		}
		return repo.CreateSecurityEvent(ctx, &auth.SecurityEvent{
			UserAccountID: &identity.UserAccountID,
			EventType:     "auth.email_verified",
			Severity:      auth.SecurityEventSeverityInfo,
			IPAddress:     input.IPAddress,
			UserAgent:     input.UserAgent,
			MetadataJSON: map[string]any{
				"identity_id":                 identity.ID.String(),
				"email_verification_token_id": emailToken.ID.String(),
			},
		})
	})
	if err != nil {
		return nil, err
	}

	return &VerifyEmailResult{
		Account: account,
	}, nil
}

func (s *Service) ResendVerificationEmail(ctx context.Context, input ResendVerificationEmailInput) (*ResendVerificationEmailResult, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return nil, err
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

	allowed, err := s.limiter.Allow(ctx, verificationRateLimitKey(input.IPAddress), s.config.VerificationIPLimit, s.config.VerificationIPWindow)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrVerificationEmailRateLimited
	}

	allowed, err = s.limiter.Allow(ctx, verificationEmailRateLimitKey(email), s.config.VerificationIPLimit, s.config.VerificationIPWindow)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrVerificationEmailRateLimited
	}

	account, err := s.repository.FindUserAccountByEmail(ctx, email)
	if errors.Is(err, auth.ErrUserAccountNotFound) {
		return &ResendVerificationEmailResult{}, nil
	}
	if err != nil {
		return nil, err
	}
	if account.Status == auth.UserAccountStatusActive {
		return &ResendVerificationEmailResult{}, nil
	}

	identity, err := s.repository.FindAuthIdentityByEmail(ctx, email)
	if errors.Is(err, auth.ErrAuthIdentityNotFound) {
		return &ResendVerificationEmailResult{}, nil
	}
	if err != nil {
		return nil, err
	}
	if identity.EmailVerifiedAt != nil {
		return &ResendVerificationEmailResult{}, nil
	}

	var verificationToken string
	var emailToken auth.EmailVerificationToken
	err = s.repository.WithinTransaction(ctx, func(ctx context.Context, repo auth.Repository) error {
		if err := repo.RevokeActiveEmailVerificationTokens(ctx, identity.ID); err != nil {
			return err
		}

		tokenValue, tokenHash, err := securetoken.New()
		if err != nil {
			return err
		}
		verificationToken = tokenValue
		now := time.Now().UTC()
		emailToken = auth.EmailVerificationToken{
			AuthIdentityID: identity.ID,
			TokenHash:      tokenHash,
			ExpiresAt:      now.Add(s.config.VerificationTokenTTL),
			CreatedAt:      now,
		}
		if err := repo.CreateEmailVerificationToken(ctx, &emailToken); err != nil {
			return err
		}

		return repo.CreateSecurityEvent(ctx, &auth.SecurityEvent{
			UserAccountID: &account.ID,
			EventType:     "auth.email_verification_resent",
			Severity:      auth.SecurityEventSeverityInfo,
			IPAddress:     input.IPAddress,
			UserAgent:     input.UserAgent,
			MetadataJSON: map[string]any{
				"identity_id":                   identity.ID.String(),
				"email_verification_token_id":   emailToken.ID.String(),
				"email_verification_expires_at": emailToken.ExpiresAt.Format(time.RFC3339),
			},
		})
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

	return &ResendVerificationEmailResult{
		VerificationEmailSent: true,
	}, nil
}

func (s *Service) LoginEmailPassword(ctx context.Context, input LoginEmailPasswordInput) (*LoginEmailPasswordResult, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if strings.TrimSpace(input.Password) == "" {
		return nil, ErrInvalidCredentials
	}
	if s.hasher == nil {
		return nil, ErrPasswordHasherRequired
	}
	if s.sessions == nil {
		return nil, ErrSessionStoreRequired
	}
	if err := s.validateSessionConfig(); err != nil {
		return nil, err
	}

	account, err := s.repository.FindUserAccountByEmail(ctx, email)
	if errors.Is(err, auth.ErrUserAccountNotFound) {
		_ = s.recordLoginFailure(ctx, nil, email, input, "invalid_credentials")
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	identity, err := s.repository.FindAuthIdentityByEmail(ctx, email)
	if errors.Is(err, auth.ErrAuthIdentityNotFound) {
		_ = s.recordLoginFailure(ctx, &account.ID, email, input, "invalid_credentials")
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if identity.PasswordHash == "" || s.hasher.Compare(identity.PasswordHash, input.Password) != nil {
		_ = s.recordLoginFailure(ctx, &account.ID, email, input, "invalid_credentials")
		return nil, ErrInvalidCredentials
	}

	if identity.EmailVerifiedAt == nil || account.Status == auth.UserAccountStatusPendingVerification {
		_ = s.recordLoginFailure(ctx, &account.ID, email, input, "email_not_verified")
		return nil, ErrEmailNotVerified
	}

	if account.Status != auth.UserAccountStatusActive {
		_ = s.recordLoginFailure(ctx, &account.ID, email, input, "account_inactive")
		return nil, ErrAccountInactive
	}

	sessionID, err := ids.NewUUID()
	if err != nil {
		return nil, err
	}
	sessionToken, _, err := securetoken.New()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	expiresAt := now.Add(s.config.SessionTTL)
	sessionKeyHash := hashSessionKey(sessionToken, s.config.SessionSecret)
	session := auth.AuthSession{
		ID:             sessionID,
		UserAccountID:  account.ID,
		SessionKeyHash: sessionKeyHash,
		IPAddress:      input.IPAddress,
		UserAgent:      input.UserAgent,
		CreatedAt:      now,
		ExpiresAt:      expiresAt,
		MetadataJSON: map[string]any{
			"identity_id": identity.ID.String(),
			"provider":    string(identity.Provider),
		},
	}

	sessionRecord := SessionRecord{
		SessionID:      sessionID,
		UserAccountID:  account.ID,
		SessionKeyHash: sessionKeyHash,
		CreatedAt:      now,
		ExpiresAt:      expiresAt,
	}
	if err := s.sessions.Save(ctx, sessionRecord, s.config.SessionTTL); err != nil {
		return nil, err
	}

	err = s.repository.WithinTransaction(ctx, func(ctx context.Context, repo auth.Repository) error {
		if err := repo.CreateAuthSession(ctx, &session); err != nil {
			return err
		}
		if err := repo.CreateLoginAttempt(ctx, &auth.LoginAttempt{
			UserAccountID: &account.ID,
			Email:         email,
			Success:       true,
			IPAddress:     input.IPAddress,
			UserAgent:     input.UserAgent,
			CreatedAt:     now,
		}); err != nil {
			return err
		}
		if err := repo.UpdateUserAccountLoginSuccess(ctx, account.ID, now); err != nil {
			return err
		}
		if err := repo.MarkAuthIdentityLastUsed(ctx, identity.ID, now); err != nil {
			return err
		}
		return repo.CreateSecurityEvent(ctx, &auth.SecurityEvent{
			UserAccountID: &account.ID,
			EventType:     "auth.login_success",
			Severity:      auth.SecurityEventSeverityInfo,
			IPAddress:     input.IPAddress,
			UserAgent:     input.UserAgent,
			MetadataJSON: map[string]any{
				"identity_id": identity.ID.String(),
				"session_id":  sessionID.String(),
			},
			CreatedAt: now,
		})
	})
	if err != nil {
		_ = s.sessions.Delete(ctx, sessionKeyHash)
		return nil, err
	}

	account.LastLoginAt = &now
	account.FailedLoginCount = 0
	account.LockedUntil = nil
	account.UpdatedAt = now

	return &LoginEmailPasswordResult{
		Account:          *account,
		SessionID:        sessionID,
		SessionToken:     sessionToken,
		SessionExpiresAt: expiresAt,
	}, nil
}

func (s *Service) CurrentAccount(ctx context.Context, input CurrentAccountInput) (*CurrentAccountResult, error) {
	sessionToken := strings.TrimSpace(input.SessionToken)
	if sessionToken == "" {
		return nil, ErrSessionTokenRequired
	}
	if s.sessions == nil {
		return nil, ErrSessionStoreRequired
	}
	if err := s.validateSessionConfig(); err != nil {
		return nil, err
	}

	sessionKeyHash := hashSessionKey(sessionToken, s.config.SessionSecret)
	session, err := s.sessions.Get(ctx, sessionKeyHash)
	if errors.Is(err, ErrSessionNotFound) {
		return nil, ErrSessionInvalid
	}
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if !session.ExpiresAt.After(now) {
		_ = s.sessions.Delete(ctx, session.SessionKeyHash)
		return nil, ErrSessionExpired
	}

	dbSession, err := s.repository.FindActiveAuthSessionByHash(ctx, session.SessionKeyHash)
	if errors.Is(err, auth.ErrAuthSessionNotFound) {
		_ = s.sessions.Delete(ctx, session.SessionKeyHash)
		return nil, ErrSessionInvalid
	}
	if err != nil {
		return nil, err
	}
	if dbSession.ExpiresAt.Before(now) || dbSession.ExpiresAt.Equal(now) {
		_ = s.sessions.Delete(ctx, session.SessionKeyHash)
		return nil, ErrSessionExpired
	}
	if dbSession.UserAccountID != session.UserAccountID || dbSession.ID != session.SessionID {
		_ = s.sessions.Delete(ctx, session.SessionKeyHash)
		return nil, ErrSessionInvalid
	}

	account, err := s.repository.FindUserAccountByID(ctx, session.UserAccountID)
	if errors.Is(err, auth.ErrUserAccountNotFound) {
		_ = s.sessions.Delete(ctx, session.SessionKeyHash)
		return nil, ErrSessionInvalid
	}
	if err != nil {
		return nil, err
	}
	if account.Status != auth.UserAccountStatusActive {
		return nil, ErrAccountInactive
	}

	return &CurrentAccountResult{
		Account: *account,
		Session: *session,
	}, nil
}

func (s *Service) LogoutCurrentSession(ctx context.Context, input LogoutCurrentSessionInput) (*LogoutCurrentSessionResult, error) {
	sessionToken := strings.TrimSpace(input.SessionToken)
	if sessionToken == "" {
		return nil, ErrSessionTokenRequired
	}
	if s.sessions == nil {
		return nil, ErrSessionStoreRequired
	}
	if err := s.validateSessionConfig(); err != nil {
		return nil, err
	}

	sessionKeyHash := hashSessionKey(sessionToken, s.config.SessionSecret)
	session, err := s.sessions.Get(ctx, sessionKeyHash)
	if errors.Is(err, ErrSessionNotFound) {
		return nil, ErrSessionInvalid
	}
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	err = s.repository.WithinTransaction(ctx, func(ctx context.Context, repo auth.Repository) error {
		if err := repo.RevokeAuthSessionByHash(ctx, session.SessionKeyHash, now, "user_logout"); err != nil {
			if errors.Is(err, auth.ErrAuthSessionNotFound) {
				return ErrSessionInvalid
			}
			return err
		}
		return repo.CreateSecurityEvent(ctx, &auth.SecurityEvent{
			UserAccountID: &session.UserAccountID,
			EventType:     "auth.logout",
			Severity:      auth.SecurityEventSeverityInfo,
			IPAddress:     input.IPAddress,
			UserAgent:     input.UserAgent,
			MetadataJSON: map[string]any{
				"session_id": session.SessionID.String(),
			},
			CreatedAt: now,
		})
	})
	if err != nil {
		return nil, err
	}

	if err := s.sessions.Delete(ctx, session.SessionKeyHash); err != nil {
		return nil, err
	}

	return &LogoutCurrentSessionResult{
		Status: "ok",
	}, nil
}

func (s *Service) LogoutAllSessions(ctx context.Context, input LogoutAllSessionsInput) (*LogoutAllSessionsResult, error) {
	sessionToken := strings.TrimSpace(input.SessionToken)
	if sessionToken == "" {
		return nil, ErrSessionTokenRequired
	}
	if s.sessions == nil {
		return nil, ErrSessionStoreRequired
	}
	if err := s.validateSessionConfig(); err != nil {
		return nil, err
	}

	sessionKeyHash := hashSessionKey(sessionToken, s.config.SessionSecret)
	current, err := s.sessions.Get(ctx, sessionKeyHash)
	if errors.Is(err, ErrSessionNotFound) {
		return nil, ErrSessionInvalid
	}
	if err != nil {
		return nil, err
	}

	dbSession, err := s.repository.FindActiveAuthSessionByHash(ctx, current.SessionKeyHash)
	if errors.Is(err, auth.ErrAuthSessionNotFound) {
		_ = s.sessions.Delete(ctx, current.SessionKeyHash)
		return nil, ErrSessionInvalid
	}
	if err != nil {
		return nil, err
	}

	activeSessions, err := s.repository.ListActiveAuthSessionsByUserAccountID(ctx, dbSession.UserAccountID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	err = s.repository.WithinTransaction(ctx, func(ctx context.Context, repo auth.Repository) error {
		if err := repo.RevokeActiveAuthSessionsByUserAccountID(ctx, dbSession.UserAccountID, now, "user_logout_all"); err != nil {
			return err
		}
		return repo.CreateSecurityEvent(ctx, &auth.SecurityEvent{
			UserAccountID: &dbSession.UserAccountID,
			EventType:     "auth.logout_all",
			Severity:      auth.SecurityEventSeverityInfo,
			IPAddress:     input.IPAddress,
			UserAgent:     input.UserAgent,
			MetadataJSON: map[string]any{
				"current_session_id": dbSession.ID.String(),
				"revoked_count":      len(activeSessions),
			},
			CreatedAt: now,
		})
	})
	if err != nil {
		return nil, err
	}

	for _, session := range activeSessions {
		_ = s.sessions.Delete(ctx, session.SessionKeyHash)
	}

	return &LogoutAllSessionsResult{
		Status: "ok",
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

func (s *Service) validateSessionConfig() error {
	if s.config.SessionSecret == "" {
		return ErrSessionConfigInvalid
	}
	if s.config.SessionTTL <= 0 {
		return ErrSessionConfigInvalid
	}
	return nil
}

func (s *Service) recordLoginFailure(ctx context.Context, accountID *uuid.UUID, email string, input LoginEmailPasswordInput, reason string) error {
	if err := s.repository.CreateLoginAttempt(ctx, &auth.LoginAttempt{
		UserAccountID: accountID,
		Email:         email,
		Success:       false,
		FailureReason: reason,
		IPAddress:     input.IPAddress,
		UserAgent:     input.UserAgent,
	}); err != nil {
		return err
	}

	return s.repository.CreateSecurityEvent(ctx, &auth.SecurityEvent{
		UserAccountID: accountID,
		EventType:     "auth.login_failed",
		Severity:      auth.SecurityEventSeverityWarning,
		IPAddress:     input.IPAddress,
		UserAgent:     input.UserAgent,
		MetadataJSON: map[string]any{
			"reason": reason,
			"email":  email,
		},
	})
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

func verificationEmailRateLimitKey(email string) string {
	return "rate:auth:verify_email:email:" + strings.ToLower(strings.TrimSpace(email))
}

func hashSessionKey(token string, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
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
