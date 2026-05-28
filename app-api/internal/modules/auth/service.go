package auth

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"

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

// Service implements the auth use cases.
type Service struct {
	accounts   AccountRepository
	identities IdentityRepository
	events     SecurityEventRepository
	sessions   SessionStore
}

// NewService constructs an auth Service wired with the provided ports.
func NewService(
	accounts AccountRepository,
	identities IdentityRepository,
	events SecurityEventRepository,
	sessions SessionStore,
) *Service {
	return &Service{
		accounts:   accounts,
		identities: identities,
		events:     events,
		sessions:   sessions,
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

	// Step 5: check account status (do NOT reveal suspended/disabled — return generic).
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

// mustNewID panics on rand failure (security events are best-effort, but id gen failure is fatal).
func mustNewID() uuid.UUID {
	id, err := ids.New()
	if err != nil {
		panic("ids.New failed: " + err.Error())
	}
	return id
}
