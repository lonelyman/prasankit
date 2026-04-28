package authsvc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"prasankit-api/internal/modules/auth"

	"github.com/google/uuid"
)

type fakeHasher struct {
	hash string
	err  error
}

func (h fakeHasher) Hash(string) (string, error) {
	if h.err != nil {
		return "", h.err
	}
	return h.hash, nil
}

type fakeRepository struct {
	existingAccount   *auth.UserAccount
	findErr           error
	transactionCalled bool
	account           *auth.UserAccount
	identity          *auth.AuthIdentity
	emailToken        *auth.EmailVerificationToken
	securityEvent     *auth.SecurityEvent
	securityEvents    []*auth.SecurityEvent
}

func (r *fakeRepository) WithinTransaction(ctx context.Context, fn func(context.Context, auth.Repository) error) error {
	r.transactionCalled = true
	return fn(ctx, r)
}

func (r *fakeRepository) FindUserAccountByEmail(context.Context, string) (*auth.UserAccount, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	if r.existingAccount != nil {
		return r.existingAccount, nil
	}
	return nil, auth.ErrUserAccountNotFound
}

func (r *fakeRepository) CreateUserAccount(_ context.Context, account *auth.UserAccount) error {
	account.ID = uuid.Must(uuid.NewV7())
	r.account = account
	return nil
}

func (r *fakeRepository) CreateAuthIdentity(_ context.Context, identity *auth.AuthIdentity) error {
	identity.ID = uuid.Must(uuid.NewV7())
	r.identity = identity
	return nil
}

func (r *fakeRepository) RevokeActiveEmailVerificationTokens(context.Context, uuid.UUID) error {
	return nil
}

func (r *fakeRepository) CreateEmailVerificationToken(_ context.Context, token *auth.EmailVerificationToken) error {
	token.ID = uuid.Must(uuid.NewV7())
	r.emailToken = token
	return nil
}

func (r *fakeRepository) CreateAuthSession(context.Context, *auth.AuthSession) error {
	return nil
}

func (r *fakeRepository) CreateLoginAttempt(context.Context, *auth.LoginAttempt) error {
	return nil
}

func (r *fakeRepository) CreateSecurityEvent(_ context.Context, event *auth.SecurityEvent) error {
	event.ID = uuid.Must(uuid.NewV7())
	r.securityEvent = event
	r.securityEvents = append(r.securityEvents, event)
	return nil
}

type fakeEmailSender struct {
	toEmail         string
	verificationURL string
	err             error
}

func (s *fakeEmailSender) SendVerificationEmail(_ context.Context, toEmail string, verificationURL string) error {
	s.toEmail = toEmail
	s.verificationURL = verificationURL
	return s.err
}

type fakeRateLimiter struct {
	allowed bool
	key     string
	limit   int
	window  time.Duration
	err     error
}

func (l *fakeRateLimiter) Allow(_ context.Context, key string, limit int, window time.Duration) (bool, error) {
	l.key = key
	l.limit = limit
	l.window = window
	if l.err != nil {
		return false, l.err
	}
	return l.allowed, nil
}

func newTestService(repo *fakeRepository, email *fakeEmailSender, limiter *fakeRateLimiter) *Service {
	if email == nil {
		email = &fakeEmailSender{}
	}
	if limiter == nil {
		limiter = &fakeRateLimiter{allowed: true}
	}
	return NewService(repo, fakeHasher{hash: "hashed-password"}, email, limiter, ServiceConfig{
		VerificationBaseURL:  "https://app.example.test/auth/verify-email",
		VerificationTokenTTL: 30 * time.Minute,
		VerificationIPLimit:  5,
		VerificationIPWindow: 10 * time.Minute,
	})
}

func TestRegisterEmailPassword(t *testing.T) {
	repo := &fakeRepository{}
	emailSender := &fakeEmailSender{}
	limiter := &fakeRateLimiter{allowed: true}
	service := newTestService(repo, emailSender, limiter)

	result, err := service.RegisterEmailPassword(context.Background(), RegisterEmailPasswordInput{
		Email:     " Owner@Example.Test ",
		Password:  "correct-password",
		IPAddress: "127.0.0.1",
		UserAgent: "service unit test",
	})
	if err != nil {
		t.Fatalf("RegisterEmailPassword: %v", err)
	}
	if result.Account.PrimaryEmail != "owner@example.test" {
		t.Fatalf("PrimaryEmail = %s, want owner@example.test", result.Account.PrimaryEmail)
	}
	if !result.VerificationEmailSent {
		t.Fatal("VerificationEmailSent = false, want true")
	}
	if !repo.transactionCalled {
		t.Fatal("transaction was not used")
	}
	if repo.account == nil {
		t.Fatal("account was not created")
	}
	if repo.identity == nil {
		t.Fatal("identity was not created")
	}
	if repo.identity.UserAccountID != repo.account.ID {
		t.Fatalf("identity.UserAccountID = %s, want %s", repo.identity.UserAccountID, repo.account.ID)
	}
	if repo.identity.IdentityType != auth.AuthIdentityTypeEmailPassword {
		t.Fatalf("identity.IdentityType = %s, want %s", repo.identity.IdentityType, auth.AuthIdentityTypeEmailPassword)
	}
	if repo.identity.Provider != auth.AuthProviderEmail {
		t.Fatalf("identity.Provider = %s, want %s", repo.identity.Provider, auth.AuthProviderEmail)
	}
	if repo.identity.PasswordHash != "hashed-password" {
		t.Fatalf("identity.PasswordHash = %s, want hashed-password", repo.identity.PasswordHash)
	}
	if repo.emailToken == nil {
		t.Fatal("email verification token was not created")
	}
	if repo.emailToken.AuthIdentityID != repo.identity.ID {
		t.Fatalf("token.AuthIdentityID = %s, want %s", repo.emailToken.AuthIdentityID, repo.identity.ID)
	}
	if repo.emailToken.TokenHash == "" {
		t.Fatal("token.TokenHash was not set")
	}
	if repo.emailToken.ExpiresAt.IsZero() {
		t.Fatal("token.ExpiresAt was not set")
	}
	if repo.securityEvent == nil {
		t.Fatal("security event was not created")
	}
	if len(repo.securityEvents) != 2 {
		t.Fatalf("security events count = %d, want 2", len(repo.securityEvents))
	}
	if repo.securityEvents[0].EventType != "auth.account_registered" {
		t.Fatalf("first event type = %s, want auth.account_registered", repo.securityEvents[0].EventType)
	}
	if repo.securityEvents[1].EventType != "auth.email_verification_sent" {
		t.Fatalf("second event type = %s, want auth.email_verification_sent", repo.securityEvents[1].EventType)
	}
	if emailSender.toEmail != "owner@example.test" {
		t.Fatalf("email to = %s, want owner@example.test", emailSender.toEmail)
	}
	if !strings.HasPrefix(emailSender.verificationURL, "https://app.example.test/auth/verify-email?token=") {
		t.Fatalf("verification URL = %s", emailSender.verificationURL)
	}
	if limiter.key != "rate:auth:verify_email:ip:127.0.0.1" {
		t.Fatalf("limiter key = %s, want rate:auth:verify_email:ip:127.0.0.1", limiter.key)
	}
}

func TestRegisterEmailPasswordRejectsInvalidInput(t *testing.T) {
	service := newTestService(&fakeRepository{}, nil, nil)

	_, err := service.RegisterEmailPassword(context.Background(), RegisterEmailPasswordInput{
		Email:    "not-an-email",
		Password: "correct-password",
	})
	if !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("err = %v, want ErrInvalidEmail", err)
	}

	_, err = service.RegisterEmailPassword(context.Background(), RegisterEmailPasswordInput{
		Email:    "owner@example.test",
		Password: "short",
	})
	if !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("err = %v, want ErrPasswordTooShort", err)
	}
}

func TestRegisterEmailPasswordRejectsDuplicateEmail(t *testing.T) {
	repo := &fakeRepository{
		existingAccount: &auth.UserAccount{
			ID:           uuid.Must(uuid.NewV7()),
			PrimaryEmail: "owner@example.test",
		},
	}
	limiter := &fakeRateLimiter{allowed: true}
	service := newTestService(repo, nil, limiter)

	_, err := service.RegisterEmailPassword(context.Background(), RegisterEmailPasswordInput{
		Email:    "owner@example.test",
		Password: "correct-password",
	})
	if !errors.Is(err, auth.ErrEmailAlreadyRegistered) {
		t.Fatalf("err = %v, want ErrEmailAlreadyRegistered", err)
	}
	if repo.transactionCalled {
		t.Fatal("transaction should not run for duplicate email")
	}
	if limiter.key != "" {
		t.Fatalf("rate limiter should not run for duplicate email, got key %s", limiter.key)
	}
}

func TestRegisterEmailPasswordRequiresPasswordHasher(t *testing.T) {
	service := NewService(&fakeRepository{}, nil, &fakeEmailSender{}, &fakeRateLimiter{allowed: true}, ServiceConfig{
		VerificationBaseURL:  "https://app.example.test/auth/verify-email",
		VerificationTokenTTL: 30 * time.Minute,
		VerificationIPLimit:  5,
		VerificationIPWindow: 10 * time.Minute,
	})

	_, err := service.RegisterEmailPassword(context.Background(), RegisterEmailPasswordInput{
		Email:    "owner@example.test",
		Password: "correct-password",
	})
	if !errors.Is(err, ErrPasswordHasherRequired) {
		t.Fatalf("err = %v, want ErrPasswordHasherRequired", err)
	}
}

func TestRegisterEmailPasswordRequiresEmailSender(t *testing.T) {
	service := NewService(&fakeRepository{}, fakeHasher{hash: "hashed-password"}, nil, &fakeRateLimiter{allowed: true}, ServiceConfig{
		VerificationBaseURL:  "https://app.example.test/auth/verify-email",
		VerificationTokenTTL: 30 * time.Minute,
		VerificationIPLimit:  5,
		VerificationIPWindow: 10 * time.Minute,
	})

	_, err := service.RegisterEmailPassword(context.Background(), RegisterEmailPasswordInput{
		Email:    "owner@example.test",
		Password: "correct-password",
	})
	if !errors.Is(err, ErrEmailSenderRequired) {
		t.Fatalf("err = %v, want ErrEmailSenderRequired", err)
	}
}

func TestRegisterEmailPasswordRateLimited(t *testing.T) {
	service := newTestService(&fakeRepository{}, nil, &fakeRateLimiter{allowed: false})

	_, err := service.RegisterEmailPassword(context.Background(), RegisterEmailPasswordInput{
		Email:    "owner@example.test",
		Password: "correct-password",
	})
	if !errors.Is(err, ErrVerificationEmailRateLimited) {
		t.Fatalf("err = %v, want ErrVerificationEmailRateLimited", err)
	}
}

func TestRegisterEmailPasswordReturnsEmailSendFailure(t *testing.T) {
	service := newTestService(&fakeRepository{}, &fakeEmailSender{err: errors.New("smtp down")}, nil)

	_, err := service.RegisterEmailPassword(context.Background(), RegisterEmailPasswordInput{
		Email:    "owner@example.test",
		Password: "correct-password",
	})
	if !errors.Is(err, ErrVerificationEmailSendFailed) {
		t.Fatalf("err = %v, want ErrVerificationEmailSendFailed", err)
	}
}
