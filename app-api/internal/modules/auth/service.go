package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"strings"
	"time"

	"prasankit-api/internal/modules/email"
	"prasankit-api/pkg/ids"
	"prasankit-api/pkg/passwordhash"
	"prasankit-api/pkg/securetoken"

	"github.com/google/uuid"
)

// lockDuration is the window an account is locked after hitting the threshold.
const lockDuration = 15 * time.Minute

// dummyHash is a pre-computed bcrypt hash used to equalise timing when a
// login attempt is made for an email that does not exist (anti-enumeration).
// The plaintext is arbitrary; it will never match any real password.
const dummyHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

// emailVerificationTTL is how long a verification token remains valid.
const emailVerificationTTL = 24 * time.Hour

// passwordResetTTL is how long a password reset token remains valid.
const passwordResetTTL = 1 * time.Hour

// Service implements the auth use cases.
type Service struct {
	accounts      AccountRepository
	identities    IdentityRepository
	events        SecurityEventRepository
	sessions      SessionStore
	verifyTokens  EmailVerificationTokenRepository
	emailSender   email.Sender
	verifyBaseURL string
	resetTokens   PasswordResetTokenRepository
	resetBaseURL  string
}

// NewService constructs an auth Service wired with the provided ports.
func NewService(
	accounts AccountRepository,
	identities IdentityRepository,
	events SecurityEventRepository,
	sessions SessionStore,
	verifyTokens EmailVerificationTokenRepository,
	emailSender email.Sender,
	verifyBaseURL string,
	resetTokens PasswordResetTokenRepository,
	resetBaseURL string,
) *Service {
	return &Service{
		accounts:      accounts,
		identities:    identities,
		events:        events,
		sessions:      sessions,
		verifyTokens:  verifyTokens,
		emailSender:   emailSender,
		verifyBaseURL: verifyBaseURL,
		resetTokens:   resetTokens,
		resetBaseURL:  resetBaseURL,
	}
}

// SignupInput carries validated signup parameters.
type SignupInput struct {
	Email       string
	Password    string
	DisplayName string
}

// LoginInput carries login parameters.
type LoginInput struct {
	Email     string
	Password  string
	IP        string
	UserAgent string
}

// SessionOutput is returned on successful login.
type SessionOutput struct {
	Account  Account
	RawToken string // raw session token to set as cookie
}

// Signup creates a new account + identity, does NOT auto-login (no session).
// Returns the created Account on success.
func (s *Service) Signup(ctx context.Context, in SignupInput) (*Account, error) {
	// Validate input.
	if err := validateSignupInput(in); err != nil {
		return nil, err
	}

	accountID, err := ids.New()
	if err != nil {
		return nil, fmt.Errorf("generate account id: %w", err)
	}
	identityID, err := ids.New()
	if err != nil {
		return nil, fmt.Errorf("generate identity id: %w", err)
	}

	hash, err := passwordhash.Hash(in.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC()

	account := Account{
		ID:                accountID,
		PrimaryEmail:      strings.ToLower(strings.TrimSpace(in.Email)),
		DisplayName:       strings.TrimSpace(in.DisplayName),
		AccountStatusCode: AccountStatusPendingVerification,
		FailedLoginCount:  0,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	identity := Identity{
		ID:               identityID,
		UserAccountID:    accountID,
		IdentityTypeCode: IdentityTypeEmailPassword,
		Email:            strings.ToLower(strings.TrimSpace(in.Email)),
		PasswordHash:     hash,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.accounts.CreateWithIdentity(ctx, account, identity); err != nil {
		return nil, err // adapter maps unique-constraint to ErrEmailTaken
	}

	// Log security event — best-effort, not fatal.
	_ = s.events.Log(ctx, SecurityEvent{
		ID:            mustNewID(),
		UserAccountID: &accountID,
		EventType:     SecurityEventAccountCreated,
		Severity:      SecuritySeverityInfo,
	})

	// Issue + send verification email — best-effort, not fatal.
	// Signup still returns 201 even if email issuance fails.
	s.issueAndSendVerification(ctx, identity, account.PrimaryEmail)

	return &account, nil
}

// Login authenticates by email+password. On success it creates a session and
// returns the raw token (to be placed in a cookie) plus the account.
// The ordering matches the spec anti-enumeration rules exactly.
func (s *Service) Login(ctx context.Context, in LoginInput) (*SessionOutput, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))

	// Step 1: look up identity by email.
	identity, err := s.identities.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("find identity: %w", err)
	}

	// Step 2: if not found → dummy compare (equalise timing) then reject.
	if identity == nil {
		_ = passwordhash.Verify(dummyHash, in.Password)
		return nil, ErrInvalidCredentials
	}

	// Step 3: load account; check lockout.
	account, err := s.accounts.FindByID(ctx, identity.UserAccountID)
	if err != nil {
		return nil, fmt.Errorf("find account: %w", err)
	}

	if account.IsLocked(time.Now().UTC()) {
		return nil, ErrAccountLocked
	}

	// Step 4: verify password. On failure → atomic increment.
	if err := passwordhash.Verify(identity.PasswordHash, in.Password); err != nil {
		_ = s.accounts.IncrementFailedLogin(ctx, account.ID, LockoutThreshold, lockDuration)
		_ = s.events.Log(ctx, SecurityEvent{
			ID:            mustNewID(),
			UserAccountID: &account.ID,
			EventType:     SecurityEventLoginFailure,
			Severity:      SecuritySeverityWarning,
			IPAddress:     in.IP,
			UserAgent:     in.UserAgent,
		})
		return nil, ErrInvalidCredentials
	}

	// Step 5: check account status.
	// pending_verification returns a distinct error (only reachable after the password
	// verified in step 4, so this is NOT an enumeration oracle).
	if account.AccountStatusCode == AccountStatusPendingVerification {
		return nil, ErrEmailNotVerified
	}
	// suspended/disabled/deleted → generic (no status reveal).
	if !account.IsLoginAllowed() {
		return nil, ErrInvalidCredentials
	}

	// Step 6: success — reset counter, create session.
	now := time.Now().UTC()
	if err := s.accounts.ResetFailedLogin(ctx, account.ID, now); err != nil {
		return nil, fmt.Errorf("reset failed login: %w", err)
	}

	rawToken, hash, err := securetoken.New()
	if err != nil {
		return nil, fmt.Errorf("generate session token: %w", err)
	}

	sessionRec := SessionRecord{
		AccountID:  account.ID,
		CreatedAt:  now,
		LastSeenAt: now,
		IP:         in.IP,
		UserAgent:  in.UserAgent,
	}
	ttl := SessionTTL * time.Hour
	if err := s.sessions.Create(ctx, hash, sessionRec, ttl); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	_ = s.events.Log(ctx, SecurityEvent{
		ID:            mustNewID(),
		UserAccountID: &account.ID,
		EventType:     SecurityEventLoginSuccess,
		Severity:      SecuritySeverityInfo,
		IPAddress:     in.IP,
		UserAgent:     in.UserAgent,
	})

	// Refresh account with updated last_login_at (return fresh state).
	account.LastLoginAt = &now
	account.FailedLoginCount = 0
	account.LockedUntil = nil

	return &SessionOutput{Account: *account, RawToken: rawToken}, nil
}

// Logout deletes the session identified by the raw cookie token.
// Idempotent — missing or already-expired token is not an error.
func (s *Service) Logout(ctx context.Context, rawToken string, accountID *uuid.UUID) error {
	hash := securetoken.Hash(rawToken)
	if err := s.sessions.Delete(ctx, hash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	if accountID != nil {
		_ = s.events.Log(ctx, SecurityEvent{
			ID:            mustNewID(),
			UserAccountID: accountID,
			EventType:     SecurityEventLogout,
			Severity:      SecuritySeverityInfo,
		})
	}

	return nil
}

// GetSession resolves a raw token to its SessionRecord.
func (s *Service) GetSession(ctx context.Context, rawToken string) (*SessionRecord, error) {
	hash := securetoken.Hash(rawToken)
	rec, err := s.sessions.Get(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	return rec, nil
}

// GetAccountByID loads an account by ID — used by requireSession to get fresh data.
func (s *Service) GetAccountByID(ctx context.Context, id uuid.UUID) (*Account, error) {
	return s.accounts.FindByID(ctx, id)
}

// validateSignupInput returns a *ValidationError when any field is invalid.
func validateSignupInput(in SignupInput) error {
	var fields []FieldError

	if _, err := mail.ParseAddress(in.Email); err != nil {
		fields = append(fields, FieldError{Field: "email", Message: "must be a valid email address"})
	}

	if len(in.Password) < 8 {
		fields = append(fields, FieldError{Field: "password", Message: "must be at least 8 characters"})
	}

	trimmed := strings.TrimSpace(in.DisplayName)
	if trimmed == "" {
		fields = append(fields, FieldError{Field: "display_name", Message: "must not be empty"})
	} else if len(trimmed) > 100 {
		fields = append(fields, FieldError{Field: "display_name", Message: "must be 100 characters or fewer"})
	}

	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	return nil
}

// issueAndSendVerification generates a verification token, stores it, and
// sends the verification email. It is entirely best-effort — all errors are
// logged and swallowed; the caller is never interrupted.
func (s *Service) issueAndSendVerification(ctx context.Context, identity Identity, recipientEmail string) {
	rawToken, hash, err := securetoken.New()
	if err != nil {
		log.Printf("auth: issueAndSendVerification: generate token: %v", err)
		return
	}

	now := time.Now().UTC()
	tok := EmailVerificationToken{
		ID:             mustNewID(),
		AuthIdentityID: identity.ID,
		TokenHash:      hash,
		ExpiresAt:      now.Add(emailVerificationTTL),
		CreatedAt:      now,
	}
	if err := s.verifyTokens.Create(ctx, tok); err != nil {
		log.Printf("auth: issueAndSendVerification: store token: %v", err)
		return
	}

	link := fmt.Sprintf("%s?token=%s", s.verifyBaseURL, rawToken)
	msg := email.Message{
		To:      recipientEmail,
		Subject: "Please verify your email address",
		TextBody: fmt.Sprintf(
			"Hello,\n\nPlease verify your email address by clicking the link below:\n\n%s\n\n"+
				"This link expires in 24 hours.\n\nIf you did not create an account, please ignore this email.\n",
			link,
		),
	}
	if err := s.emailSender.Send(ctx, msg); err != nil {
		log.Printf("auth: issueAndSendVerification: send email to %q: %v", recipientEmail, err)
	}

	_ = s.events.Log(ctx, SecurityEvent{
		ID:            mustNewID(),
		UserAccountID: &identity.UserAccountID,
		EventType:     SecurityEventEmailVerificationRequested,
		Severity:      SecuritySeverityInfo,
	})
}

// ConfirmEmailVerification consumes a raw verification token, marks the
// identity as verified, and flips the account status from pending_verification
// to active.
func (s *Service) ConfirmEmailVerification(ctx context.Context, rawToken string) error {
	hash := securetoken.Hash(rawToken)
	tok, err := s.verifyTokens.FindByTokenHash(ctx, hash)
	if err != nil {
		return fmt.Errorf("find verification token: %w", err)
	}
	if tok == nil {
		return ErrTokenInvalid
	}

	now := time.Now().UTC()
	if !tok.IsActive(now) {
		return ErrTokenExpired
	}

	ident, err := s.identities.FindByID(ctx, tok.AuthIdentityID)
	if err != nil {
		return fmt.Errorf("find identity: %w", err)
	}
	if ident == nil {
		return ErrTokenInvalid
	}

	if err := s.verifyTokens.ConfirmTx(ctx, tok.ID, ident.ID, ident.UserAccountID, now); err != nil {
		if errors.Is(err, ErrTokenExpired) {
			return ErrTokenExpired
		}
		return fmt.Errorf("confirm verification tx: %w", err)
	}

	_ = s.events.Log(ctx, SecurityEvent{
		ID:            mustNewID(),
		UserAccountID: &ident.UserAccountID,
		EventType:     SecurityEventEmailVerified,
		Severity:      SecuritySeverityInfo,
	})

	return nil
}

// ResendVerification re-issues a verification email for the given email
// address. It always returns nil to the caller on the "no-op" paths to avoid
// enumeration (no timing padding — see D32 rationale).
func (s *Service) ResendVerification(ctx context.Context, emailAddr string) error {
	normalized := strings.ToLower(strings.TrimSpace(emailAddr))

	ident, err := s.identities.FindByEmail(ctx, normalized)
	if err != nil {
		return fmt.Errorf("find identity for resend: %w", err)
	}
	if ident == nil {
		return nil // unknown email — silent
	}
	if ident.EmailVerifiedAt != nil {
		return nil // already verified — silent
	}

	now := time.Now().UTC()
	if err := s.verifyTokens.RevokeActiveByIdentity(ctx, ident.ID, now); err != nil {
		log.Printf("auth: ResendVerification: revoke active tokens: %v", err)
	}

	s.issueAndSendVerification(ctx, *ident, emailAddr)
	return nil
}

// issueAndSendReset generates a password reset token, stores it, and sends
// the reset email. It is entirely best-effort — all errors are logged and
// swallowed; the caller is never interrupted.
func (s *Service) issueAndSendReset(ctx context.Context, identity Identity, recipientEmail string) {
	rawToken, hash, err := securetoken.New()
	if err != nil {
		log.Printf("auth: issueAndSendReset: generate token: %v", err)
		return
	}

	now := time.Now().UTC()
	tok := PasswordResetToken{
		ID:             mustNewID(),
		AuthIdentityID: identity.ID,
		TokenHash:      hash,
		ExpiresAt:      now.Add(passwordResetTTL),
		CreatedAt:      now,
	}
	if err := s.resetTokens.Create(ctx, tok); err != nil {
		log.Printf("auth: issueAndSendReset: store token: %v", err)
		return
	}

	link := fmt.Sprintf("%s?token=%s", s.resetBaseURL, rawToken)
	msg := email.Message{
		To:      recipientEmail,
		Subject: "Reset your password",
		TextBody: fmt.Sprintf(
			"Hello,\n\nYou requested a password reset. Click the link below to set a new password:\n\n%s\n\n"+
				"This link expires in 1 hour.\n\nIf you did not request a password reset, please ignore this email.\n",
			link,
		),
	}
	if err := s.emailSender.Send(ctx, msg); err != nil {
		log.Printf("auth: issueAndSendReset: send email to %q: %v", recipientEmail, err)
	}

	_ = s.events.Log(ctx, SecurityEvent{
		ID:            mustNewID(),
		UserAccountID: &identity.UserAccountID,
		EventType:     SecurityEventPasswordResetRequested,
		Severity:      SecuritySeverityInfo,
	})
}

// RequestPasswordReset issues a password reset email for the given email
// address. It always returns nil on "no-op" paths to avoid enumeration
// (no timing padding — see D34 rationale).
func (s *Service) RequestPasswordReset(ctx context.Context, emailAddr string) error {
	normalized := strings.ToLower(strings.TrimSpace(emailAddr))

	ident, err := s.identities.FindByEmail(ctx, normalized)
	if err != nil {
		return fmt.Errorf("find identity for password reset: %w", err)
	}
	if ident == nil {
		return nil // unknown email — silent (anti-enum)
	}

	now := time.Now().UTC()
	if err := s.resetTokens.RevokeActiveByIdentity(ctx, ident.ID, now); err != nil {
		log.Printf("auth: RequestPasswordReset: revoke active tokens: %v", err)
	}

	s.issueAndSendReset(ctx, *ident, emailAddr)
	return nil
}

// ConfirmPasswordReset validates the token and sets a new password.
func (s *Service) ConfirmPasswordReset(ctx context.Context, rawToken, newPassword string) error {
	// Step 1: validate password length.
	if len(newPassword) < 8 {
		return &ValidationError{Fields: []FieldError{{Field: "new_password", Message: "must be at least 8 characters"}}}
	}

	// Step 2: look up token.
	hash := securetoken.Hash(rawToken)
	tok, err := s.resetTokens.FindByTokenHash(ctx, hash)
	if err != nil {
		return fmt.Errorf("find reset token: %w", err)
	}
	if tok == nil {
		return ErrTokenInvalid
	}

	// Step 3: check token is active.
	now := time.Now().UTC()
	if !tok.IsActive(now) {
		return ErrTokenExpired
	}

	// Step 4: load identity.
	ident, err := s.identities.FindByID(ctx, tok.AuthIdentityID)
	if err != nil {
		return fmt.Errorf("find identity: %w", err)
	}
	if ident == nil {
		return ErrTokenInvalid
	}

	// Step 5: hash the new password.
	newHash, err := passwordhash.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}

	// Step 6: atomic confirm in transaction.
	if err := s.resetTokens.ConfirmTx(ctx, tok.ID, ident.ID, ident.UserAccountID, newHash, now); err != nil {
		if errors.Is(err, ErrTokenExpired) {
			return ErrTokenExpired
		}
		return fmt.Errorf("confirm password reset tx: %w", err)
	}

	// Step 7: log security event best-effort.
	_ = s.events.Log(ctx, SecurityEvent{
		ID:            mustNewID(),
		UserAccountID: &ident.UserAccountID,
		EventType:     SecurityEventPasswordReset,
		Severity:      SecuritySeverityInfo,
	})

	return nil
}

// mustNewID panics on rand failure (security events are best-effort, but id gen failure is fatal).
func mustNewID() uuid.UUID {
	id, err := ids.New()
	if err != nil {
		panic("ids.New failed: " + err.Error())
	}
	return id
}
