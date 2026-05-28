package auth_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/email"
	"prasankit-api/pkg/passwordhash"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ── Fake implementations of ports ────────────────────────────────────────────

type fakeAccountRepo struct {
	mu           sync.Mutex
	accounts     map[uuid.UUID]*auth.Account
	identityRepo *fakeIdentityRepo // set after construction so CreateWithIdentity can also store the identity
}

func newFakeAccountRepo() *fakeAccountRepo {
	return &fakeAccountRepo{accounts: map[uuid.UUID]*auth.Account{}}
}

func (r *fakeAccountRepo) CreateWithIdentity(ctx context.Context, account auth.Account, identity auth.Identity) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, a := range r.accounts {
		if a.PrimaryEmail == account.PrimaryEmail {
			return auth.ErrEmailTaken
		}
	}
	cp := account
	r.accounts[account.ID] = &cp
	// Also persist the identity so FindByID/FindByEmail work after Signup.
	if r.identityRepo != nil {
		icp := identity
		r.identityRepo.mu.Lock()
		r.identityRepo.identities[identity.Email] = &icp
		r.identityRepo.byID[identity.ID] = &icp
		r.identityRepo.mu.Unlock()
	}
	return nil
}

func (r *fakeAccountRepo) FindByID(ctx context.Context, id uuid.UUID) (*auth.Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.accounts[id]
	if !ok {
		return nil, nil
	}
	cp := *a
	return &cp, nil
}

func (r *fakeAccountRepo) IncrementFailedLogin(ctx context.Context, accountID uuid.UUID, threshold int, lockDuration time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.accounts[accountID]
	if !ok {
		return nil
	}
	a.FailedLoginCount++
	if a.FailedLoginCount >= threshold {
		until := time.Now().Add(lockDuration)
		a.LockedUntil = &until
	}
	return nil
}

func (r *fakeAccountRepo) ResetFailedLogin(ctx context.Context, accountID uuid.UUID, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.accounts[accountID]
	if !ok {
		return nil
	}
	a.FailedLoginCount = 0
	a.LockedUntil = nil
	a.LastLoginAt = &now
	return nil
}

type fakeIdentityRepo struct {
	mu         sync.Mutex
	identities map[string]*auth.Identity // keyed by email
	byID       map[uuid.UUID]*auth.Identity
}

func newFakeIdentityRepo() *fakeIdentityRepo {
	return &fakeIdentityRepo{
		identities: map[string]*auth.Identity{},
		byID:       map[uuid.UUID]*auth.Identity{},
	}
}

func (r *fakeIdentityRepo) FindByEmail(ctx context.Context, email string) (*auth.Identity, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	i, ok := r.identities[email]
	if !ok {
		return nil, nil
	}
	cp := *i
	return &cp, nil
}

func (r *fakeIdentityRepo) FindByID(ctx context.Context, id uuid.UUID) (*auth.Identity, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	i, ok := r.byID[id]
	if !ok {
		return nil, nil
	}
	cp := *i
	return &cp, nil
}

func (r *fakeIdentityRepo) FindByUserAccountID(ctx context.Context, accountID uuid.UUID) (*auth.Identity, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, i := range r.byID {
		if i.UserAccountID == accountID {
			cp := *i
			return &cp, nil
		}
	}
	return nil, nil
}

type fakeEventRepo struct {
	mu     sync.Mutex
	events []auth.SecurityEvent
}

func (r *fakeEventRepo) Log(ctx context.Context, event auth.SecurityEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
	return nil
}

type fakeSessionStore struct {
	mu       sync.Mutex
	sessions map[string]auth.SessionRecord
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{sessions: map[string]auth.SessionRecord{}}
}

func (s *fakeSessionStore) Create(ctx context.Context, hash string, record auth.SessionRecord, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[hash] = record
	return nil
}

func (s *fakeSessionStore) Get(ctx context.Context, hash string) (*auth.SessionRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.sessions[hash]
	if !ok {
		return nil, nil
	}
	cp := rec
	return &cp, nil
}

func (s *fakeSessionStore) Delete(ctx context.Context, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, hash)
	return nil
}

// fakeVerificationRepo implements auth.EmailVerificationTokenRepository in memory.
type fakeVerificationRepo struct {
	mu     sync.Mutex
	tokens map[string]*auth.EmailVerificationToken // keyed by token hash

	// back-references so ConfirmTx can mutate sibling fakes
	accountRepo  *fakeAccountRepo
	identityRepo *fakeIdentityRepo

	// mirrors of confirmed state for assertions
	usedTokens    map[uuid.UUID]time.Time  // tokenID → usedAt
	verifiedIdent map[uuid.UUID]*time.Time // identityID → verifiedAt
}

func newFakeVerificationRepo() *fakeVerificationRepo {
	return &fakeVerificationRepo{
		tokens:        map[string]*auth.EmailVerificationToken{},
		usedTokens:    map[uuid.UUID]time.Time{},
		verifiedIdent: map[uuid.UUID]*time.Time{},
	}
}

func (r *fakeVerificationRepo) Create(ctx context.Context, token auth.EmailVerificationToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := token
	r.tokens[token.TokenHash] = &cp
	return nil
}

func (r *fakeVerificationRepo) RevokeActiveByIdentity(ctx context.Context, identityID uuid.UUID, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.tokens {
		if t.AuthIdentityID == identityID && t.IsActive(now) {
			cp := now
			t.RevokedAt = &cp
		}
	}
	return nil
}

func (r *fakeVerificationRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*auth.EmailVerificationToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tokens[tokenHash]
	if !ok {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}

func (r *fakeVerificationRepo) ConfirmTx(ctx context.Context, tokenID, identityID, accountID uuid.UUID, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Find token by ID.
	var tok *auth.EmailVerificationToken
	for _, t := range r.tokens {
		if t.ID == tokenID {
			tok = t
			break
		}
	}
	if tok == nil || tok.UsedAt != nil || tok.RevokedAt != nil {
		return auth.ErrTokenExpired
	}
	tok.UsedAt = &now
	r.usedTokens[tokenID] = now
	r.verifiedIdent[identityID] = &now

	// Flip account status pending_verification → active (idempotent, mirrors the real repo).
	if r.accountRepo != nil {
		r.accountRepo.mu.Lock()
		if a, ok := r.accountRepo.accounts[accountID]; ok && a.AccountStatusCode == auth.AccountStatusPendingVerification {
			a.AccountStatusCode = auth.AccountStatusActive
		}
		r.accountRepo.mu.Unlock()
	}
	return nil
}

// fakeResetRepo implements auth.PasswordResetTokenRepository in memory.
type fakeResetRepo struct {
	mu     sync.Mutex
	tokens map[string]*auth.PasswordResetToken // keyed by token hash

	// back-references so ConfirmTx can mutate sibling fakes
	accountRepo  *fakeAccountRepo
	identityRepo *fakeIdentityRepo

	// mirrors of confirmed state for assertions
	usedTokens      map[uuid.UUID]time.Time // tokenID → usedAt
	newPasswordHash map[uuid.UUID]string    // identityID → new password hash
	lockoutCleared  map[uuid.UUID]bool      // accountID → lockout cleared
}

func newFakeResetRepo() *fakeResetRepo {
	return &fakeResetRepo{
		tokens:          map[string]*auth.PasswordResetToken{},
		usedTokens:      map[uuid.UUID]time.Time{},
		newPasswordHash: map[uuid.UUID]string{},
		lockoutCleared:  map[uuid.UUID]bool{},
	}
}

func (r *fakeResetRepo) Create(ctx context.Context, token auth.PasswordResetToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := token
	r.tokens[token.TokenHash] = &cp
	return nil
}

func (r *fakeResetRepo) RevokeActiveByIdentity(ctx context.Context, identityID uuid.UUID, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.tokens {
		if t.AuthIdentityID == identityID && t.IsActive(now) {
			cp := now
			t.RevokedAt = &cp
		}
	}
	return nil
}

func (r *fakeResetRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*auth.PasswordResetToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tokens[tokenHash]
	if !ok {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}

func (r *fakeResetRepo) ConfirmTx(ctx context.Context, tokenID, identityID, accountID uuid.UUID, newPasswordHash string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Find token by ID.
	var tok *auth.PasswordResetToken
	for _, t := range r.tokens {
		if t.ID == tokenID {
			tok = t
			break
		}
	}
	if tok == nil || tok.UsedAt != nil || tok.RevokedAt != nil {
		return auth.ErrTokenExpired
	}
	tok.UsedAt = &now
	r.usedTokens[tokenID] = now
	r.newPasswordHash[identityID] = newPasswordHash

	// Update identity password hash in the fake identity repo.
	if r.identityRepo != nil {
		r.identityRepo.mu.Lock()
		if ident, ok := r.identityRepo.byID[identityID]; ok {
			ident.PasswordHash = newPasswordHash
		}
		r.identityRepo.mu.Unlock()
	}

	// Clear account lockout.
	if r.accountRepo != nil {
		r.accountRepo.mu.Lock()
		if a, ok := r.accountRepo.accounts[accountID]; ok {
			a.FailedLoginCount = 0
			a.LockedUntil = nil
		}
		r.accountRepo.mu.Unlock()
	}
	r.lockoutCleared[accountID] = true
	return nil
}

// fakeEmailSender records sent messages.
type fakeEmailSender struct {
	mu       sync.Mutex
	messages []email.Message
}

func (s *fakeEmailSender) Send(ctx context.Context, msg email.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, msg)
	return nil
}

func (s *fakeEmailSender) Sent() []email.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]email.Message, len(s.messages))
	copy(cp, s.messages)
	return cp
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func buildSvc() (*auth.Service, *fakeAccountRepo, *fakeIdentityRepo, *fakeEventRepo, *fakeSessionStore, *fakeVerificationRepo, *fakeEmailSender) {
	accounts := newFakeAccountRepo()
	identities := newFakeIdentityRepo()
	verifyRepo := newFakeVerificationRepo()
	resetRepo := newFakeResetRepo()
	// Wire back-references so the fakes stay consistent across Signup/ConfirmTx.
	accounts.identityRepo = identities
	verifyRepo.accountRepo = accounts
	verifyRepo.identityRepo = identities
	resetRepo.accountRepo = accounts
	resetRepo.identityRepo = identities
	events := &fakeEventRepo{}
	sessions := newFakeSessionStore()
	sender := &fakeEmailSender{}
	svc := auth.NewService(accounts, identities, events, sessions, verifyRepo, sender, "http://localhost:13000/verify-email", resetRepo, "http://localhost:13000/reset-password")
	return svc, accounts, identities, events, sessions, verifyRepo, sender
}

// seedAccount inserts a pre-built account+identity directly into the fakes.
func seedAccount(accounts *fakeAccountRepo, identities *fakeIdentityRepo, email, plainPwd string, status string) *auth.Account {
	hash, _ := bcrypt.GenerateFromPassword([]byte(plainPwd), bcrypt.MinCost)
	id, _ := uuid.NewV7()
	iid, _ := uuid.NewV7()
	now := time.Now().UTC()
	a := &auth.Account{
		ID:                id,
		PrimaryEmail:      email,
		DisplayName:       "Test User",
		AccountStatusCode: status,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	accounts.mu.Lock()
	accounts.accounts[id] = a
	accounts.mu.Unlock()

	ident := &auth.Identity{
		ID:               iid,
		UserAccountID:    id,
		IdentityTypeCode: auth.IdentityTypeEmailPassword,
		Email:            email,
		PasswordHash:     string(hash),
	}
	identities.mu.Lock()
	identities.identities[email] = ident
	identities.byID[iid] = ident
	identities.mu.Unlock()

	return a
}

// tokenHash mirrors securetoken.Hash without importing the adapter package.
func tokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum)
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestSignup_Success(t *testing.T) {
	svc, accounts, _, events, _, verifyRepo, sender := buildSvc()
	account, err := svc.Signup(context.Background(), auth.SignupInput{
		Email:       "alice@example.com",
		Password:    "securepass",
		DisplayName: "Alice",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if account.PrimaryEmail != "alice@example.com" {
		t.Errorf("primary_email = %q, want alice@example.com", account.PrimaryEmail)
	}
	if account.AccountStatusCode != auth.AccountStatusPendingVerification {
		t.Errorf("status = %q, want pending_verification", account.AccountStatusCode)
	}
	stored, _ := accounts.FindByID(context.Background(), account.ID)
	if stored == nil {
		t.Fatal("account not found in repo after signup")
	}

	// Security event for account.created should be logged.
	hasCreated := false
	for _, e := range events.events {
		if e.EventType == auth.SecurityEventAccountCreated {
			hasCreated = true
		}
	}
	if !hasCreated {
		t.Errorf("security event account.created not logged: %v", events.events)
	}

	// Verification token should be issued.
	tokens := verifyRepo.tokens
	if len(tokens) != 1 {
		t.Fatalf("expected 1 verification token, got %d", len(tokens))
	}

	// Verification email should be sent with a ?token= link.
	msgs := sender.Sent()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 email sent, got %d", len(msgs))
	}
	if !strings.Contains(msgs[0].TextBody, "?token=") {
		t.Errorf("verification email body missing '?token=': %s", msgs[0].TextBody)
	}
}

func TestSignup_DuplicateEmail(t *testing.T) {
	svc, accounts, identities, _, _, _, _ := buildSvc()
	seedAccount(accounts, identities, "bob@example.com", "password1", auth.AccountStatusActive)

	_, err := svc.Signup(context.Background(), auth.SignupInput{
		Email:       "bob@example.com",
		Password:    "otherpass",
		DisplayName: "Bob2",
	})
	if !errors.Is(err, auth.ErrEmailTaken) {
		t.Errorf("err = %v, want ErrEmailTaken", err)
	}
}

func TestSignup_ValidationFailures(t *testing.T) {
	cases := []struct {
		name  string
		input auth.SignupInput
		field string
	}{
		{"bad email", auth.SignupInput{Email: "notanemail", Password: "pass1234", DisplayName: "X"}, "email"},
		{"short password", auth.SignupInput{Email: "x@x.com", Password: "short", DisplayName: "X"}, "password"},
		{"empty display name", auth.SignupInput{Email: "x@x.com", Password: "pass1234", DisplayName: "   "}, "display_name"},
		{"long display name", auth.SignupInput{Email: "x@x.com", Password: "pass1234", DisplayName: string(make([]byte, 101))}, "display_name"},
	}
	svc, _, _, _, _, _, _ := buildSvc()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Signup(context.Background(), tc.input)
			var ve *auth.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("err = %v, want ValidationError", err)
			}
			found := false
			for _, fe := range ve.Fields {
				if fe.Field == tc.field {
					found = true
				}
			}
			if !found {
				t.Errorf("expected field %q in errors: %v", tc.field, ve.Fields)
			}
		})
	}
}

func TestLogin_Success(t *testing.T) {
	svc, accounts, identities, events, sessions, _, _ := buildSvc()
	seedAccount(accounts, identities, "carol@example.com", "mypassword", auth.AccountStatusActive)

	out, err := svc.Login(context.Background(), auth.LoginInput{
		Email:    "carol@example.com",
		Password: "mypassword",
		IP:       "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.RawToken == "" {
		t.Error("raw token should not be empty")
	}
	hash := tokenHash(out.RawToken)
	rec, _ := sessions.Get(context.Background(), hash)
	if rec == nil {
		t.Error("session not found in store after login")
	}
	hasSuccess := false
	for _, e := range events.events {
		if e.EventType == auth.SecurityEventLoginSuccess {
			hasSuccess = true
		}
	}
	if !hasSuccess {
		t.Error("login.success event not logged")
	}
}

func TestLogin_PendingVerification_Returns_ErrEmailNotVerified(t *testing.T) {
	svc, accounts, identities, _, _, _, _ := buildSvc()
	seedAccount(accounts, identities, "unverified@example.com", "mypassword", auth.AccountStatusPendingVerification)

	_, err := svc.Login(context.Background(), auth.LoginInput{
		Email:    "unverified@example.com",
		Password: "mypassword",
	})
	if !errors.Is(err, auth.ErrEmailNotVerified) {
		t.Errorf("err = %v, want ErrEmailNotVerified", err)
	}
}

func TestLogin_WrongPassword_CounterIncrements(t *testing.T) {
	svc, accounts, identities, _, _, _, _ := buildSvc()
	a := seedAccount(accounts, identities, "dave@example.com", "rightpass", auth.AccountStatusActive)

	for i := 0; i < 3; i++ {
		_, err := svc.Login(context.Background(), auth.LoginInput{
			Email:    "dave@example.com",
			Password: "wrongpass",
		})
		if !errors.Is(err, auth.ErrInvalidCredentials) {
			t.Fatalf("attempt %d: err = %v, want ErrInvalidCredentials", i+1, err)
		}
	}

	stored, _ := accounts.FindByID(context.Background(), a.ID)
	if stored.FailedLoginCount != 3 {
		t.Errorf("failed_login_count = %d, want 3", stored.FailedLoginCount)
	}
}

func TestLogin_Lockout_AfterThreshold(t *testing.T) {
	svc, accounts, identities, _, _, _, _ := buildSvc()
	a := seedAccount(accounts, identities, "eve@example.com", "rightpass", auth.AccountStatusActive)

	for i := 0; i < auth.LockoutThreshold; i++ {
		_, _ = svc.Login(context.Background(), auth.LoginInput{
			Email:    "eve@example.com",
			Password: "wrong",
		})
	}

	stored, _ := accounts.FindByID(context.Background(), a.ID)
	if stored.LockedUntil == nil {
		t.Fatal("account should be locked after threshold")
	}
	if !stored.LockedUntil.After(time.Now()) {
		t.Error("locked_until should be in the future")
	}
}

func TestLogin_LockedAccount(t *testing.T) {
	svc, accounts, identities, _, _, _, _ := buildSvc()
	a := seedAccount(accounts, identities, "frank@example.com", "pw123456", auth.AccountStatusActive)

	until := time.Now().Add(15 * time.Minute)
	accounts.mu.Lock()
	accounts.accounts[a.ID].LockedUntil = &until
	accounts.mu.Unlock()

	_, err := svc.Login(context.Background(), auth.LoginInput{
		Email:    "frank@example.com",
		Password: "pw123456",
	})
	if !errors.Is(err, auth.ErrAccountLocked) {
		t.Errorf("err = %v, want ErrAccountLocked", err)
	}
}

func TestLogin_NonActiveStatus(t *testing.T) {
	svc, accounts, identities, _, _, _, _ := buildSvc()
	seedAccount(accounts, identities, "grace@example.com", "pw123456", auth.AccountStatusSuspended)

	_, err := svc.Login(context.Background(), auth.LoginInput{
		Email:    "grace@example.com",
		Password: "pw123456",
	})
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("err = %v, want ErrInvalidCredentials (no status reveal)", err)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	svc, _, _, _, _, _, _ := buildSvc()

	_, err := svc.Login(context.Background(), auth.LoginInput{
		Email:    "nobody@example.com",
		Password: "anything",
	})
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("err = %v, want ErrInvalidCredentials", err)
	}
}

func TestLogout(t *testing.T) {
	svc, accounts, identities, _, sessions, _, _ := buildSvc()
	a := seedAccount(accounts, identities, "heidi@example.com", "pw123456", auth.AccountStatusActive)

	out, err := svc.Login(context.Background(), auth.LoginInput{
		Email:    "heidi@example.com",
		Password: "pw123456",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	token := out.RawToken
	hash := tokenHash(token)

	rec, _ := sessions.Get(context.Background(), hash)
	if rec == nil {
		t.Fatal("session not found before logout")
	}

	if err := svc.Logout(context.Background(), token, &a.ID); err != nil {
		t.Fatalf("logout: %v", err)
	}

	rec, _ = sessions.Get(context.Background(), hash)
	if rec != nil {
		t.Error("session should be deleted after logout")
	}
}

// ── ConfirmEmailVerification tests ───────────────────────────────────────────

func TestConfirmEmailVerification_ValidToken(t *testing.T) {
	svc, accounts, identities, _, _, verifyRepo, sender := buildSvc()

	// Signup creates account + issues token.
	account, err := svc.Signup(context.Background(), auth.SignupInput{
		Email:       "verify@example.com",
		Password:    "securepass",
		DisplayName: "Verify Me",
	})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}

	// Extract raw token from the sent email.
	msgs := sender.Sent()
	if len(msgs) == 0 {
		t.Fatal("no verification email sent")
	}
	body := msgs[0].TextBody
	idx := strings.Index(body, "?token=")
	if idx < 0 {
		t.Fatalf("no ?token= in email body: %s", body)
	}
	rawToken := strings.TrimSpace(strings.SplitN(body[idx+len("?token="):], "\n", 2)[0])

	// Confirm.
	if err := svc.ConfirmEmailVerification(context.Background(), rawToken); err != nil {
		t.Fatalf("ConfirmEmailVerification: %v", err)
	}

	// Token should be used.
	hash := tokenHash(rawToken)
	verifyRepo.mu.Lock()
	tok := verifyRepo.tokens[hash]
	verifyRepo.mu.Unlock()
	if tok == nil || tok.UsedAt == nil {
		t.Error("token should have used_at set after confirm")
	}

	// Account should be active.
	accounts.mu.Lock()
	acct := accounts.accounts[account.ID]
	accounts.mu.Unlock()
	if acct.AccountStatusCode != auth.AccountStatusActive {
		t.Errorf("account status = %q, want active", acct.AccountStatusCode)
	}

	// Identity verified_at should be set (keyed by identityID in fakeVerificationRepo).
	identities.mu.Lock()
	identID := identities.identities["verify@example.com"].ID
	identities.mu.Unlock()
	verifyRepo.mu.Lock()
	verifiedAt, ok := verifyRepo.verifiedIdent[identID]
	verifyRepo.mu.Unlock()
	if !ok || verifiedAt == nil {
		t.Error("identity email_verified_at should be set")
	}
}

func TestConfirmEmailVerification_UnknownToken(t *testing.T) {
	svc, _, _, _, _, _, _ := buildSvc()
	err := svc.ConfirmEmailVerification(context.Background(), "completelyunknowntoken")
	if !errors.Is(err, auth.ErrTokenInvalid) {
		t.Errorf("err = %v, want ErrTokenInvalid", err)
	}
}

func TestConfirmEmailVerification_ExpiredToken(t *testing.T) {
	accounts2 := newFakeAccountRepo()
	identities2 := newFakeIdentityRepo()
	verifyRepo2 := newFakeVerificationRepo()

	eml := "expired@example.com"
	id, _ := uuid.NewV7()
	iid, _ := uuid.NewV7()
	now := time.Now().UTC()
	a := &auth.Account{
		ID:                id,
		PrimaryEmail:      eml,
		AccountStatusCode: auth.AccountStatusPendingVerification,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	accounts2.mu.Lock()
	accounts2.accounts[id] = a
	accounts2.mu.Unlock()

	ident := &auth.Identity{
		ID:            iid,
		UserAccountID: id,
		Email:         eml,
	}
	identities2.mu.Lock()
	identities2.identities[eml] = ident
	identities2.byID[iid] = ident
	identities2.mu.Unlock()

	// Insert an already-expired token directly.
	rawToken := "expiredrawtoken"
	h := tokenHash(rawToken)
	expiredTime := now.Add(-1 * time.Hour) // in the past
	tok := &auth.EmailVerificationToken{
		ID:             mustUUID(),
		AuthIdentityID: iid,
		TokenHash:      h,
		ExpiresAt:      expiredTime,
		CreatedAt:      now,
	}
	verifyRepo2.mu.Lock()
	verifyRepo2.tokens[h] = tok
	verifyRepo2.mu.Unlock()

	svc := auth.NewService(accounts2, identities2, &fakeEventRepo{}, newFakeSessionStore(), verifyRepo2, &fakeEmailSender{}, "http://localhost:13000/verify-email", newFakeResetRepo(), "http://localhost:13000/reset-password")

	err := svc.ConfirmEmailVerification(context.Background(), rawToken)
	if !errors.Is(err, auth.ErrTokenExpired) {
		t.Errorf("err = %v, want ErrTokenExpired", err)
	}
}

func TestConfirmEmailVerification_AlreadyUsedToken(t *testing.T) {
	svc, accounts, identities, _, _, verifyRepo, _ := buildSvc()

	// Signup to get a real token.
	_, err := svc.Signup(context.Background(), auth.SignupInput{
		Email:       "used@example.com",
		Password:    "securepass",
		DisplayName: "Used Token",
	})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}

	// Find the token hash.
	verifyRepo.mu.Lock()
	var rawHash string
	for h := range verifyRepo.tokens {
		rawHash = h
	}
	verifyRepo.mu.Unlock()

	// Mark it as used directly.
	now := time.Now().UTC()
	verifyRepo.mu.Lock()
	verifyRepo.tokens[rawHash].UsedAt = &now
	verifyRepo.mu.Unlock()

	// Get the identity ID.
	identities.mu.Lock()
	ident := identities.identities["used@example.com"]
	identities.mu.Unlock()

	// Manually build a raw token that hashes to rawHash — we can't recover original,
	// so instead insert a new token with a known raw token.
	rawToken2 := "knownrawtoken12345"
	h2 := tokenHash(rawToken2)
	verifyRepo.mu.Lock()
	verifyRepo.tokens[h2] = &auth.EmailVerificationToken{
		ID:             mustUUID(),
		AuthIdentityID: ident.ID,
		TokenHash:      h2,
		ExpiresAt:      now.Add(1 * time.Hour),
		UsedAt:         &now, // already used
		CreatedAt:      now,
	}
	verifyRepo.mu.Unlock()

	err = svc.ConfirmEmailVerification(context.Background(), rawToken2)
	if !errors.Is(err, auth.ErrTokenExpired) {
		t.Errorf("already-used token: err = %v, want ErrTokenExpired", err)
	}
	_ = accounts
}

// ── ResendVerification tests ──────────────────────────────────────────────────

func TestResendVerification_UnknownEmail_Nil(t *testing.T) {
	svc, _, _, _, _, verifyRepo, sender := buildSvc()
	err := svc.ResendVerification(context.Background(), "nobody@example.com")
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if len(verifyRepo.tokens) != 0 {
		t.Error("no token should be issued for unknown email")
	}
	if len(sender.Sent()) != 0 {
		t.Error("no email should be sent for unknown email")
	}
}

func TestResendVerification_AlreadyVerified_Nil(t *testing.T) {
	svc, accounts, identities, _, _, verifyRepo, sender := buildSvc()
	a := seedAccount(accounts, identities, "verified@example.com", "pw", auth.AccountStatusActive)
	// Mark identity as verified.
	now := time.Now().UTC()
	identities.mu.Lock()
	identities.identities["verified@example.com"].EmailVerifiedAt = &now
	identities.mu.Unlock()
	_ = a

	err := svc.ResendVerification(context.Background(), "verified@example.com")
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if len(verifyRepo.tokens) != 0 {
		t.Error("no new token for already-verified identity")
	}
	if len(sender.Sent()) != 0 {
		t.Error("no email for already-verified identity")
	}
}

func TestResendVerification_Unverified_RevokesAndReissues(t *testing.T) {
	svc, accounts, identities, _, _, verifyRepo, sender := buildSvc()

	// Signup first to have an existing pending token.
	_, err := svc.Signup(context.Background(), auth.SignupInput{
		Email:       "resend@example.com",
		Password:    "securepass",
		DisplayName: "Resend Me",
	})
	if err != nil {
		t.Fatalf("signup: %v", err)
	}

	initialTokenCount := len(verifyRepo.tokens)
	initialEmailCount := len(sender.Sent())

	// Resend.
	err = svc.ResendVerification(context.Background(), "resend@example.com")
	if err != nil {
		t.Errorf("ResendVerification: %v", err)
	}

	// A new token should have been issued.
	if len(verifyRepo.tokens) != initialTokenCount+1 {
		t.Errorf("expected %d tokens, got %d", initialTokenCount+1, len(verifyRepo.tokens))
	}
	// A new email should have been sent.
	if len(sender.Sent()) != initialEmailCount+1 {
		t.Errorf("expected %d emails, got %d", initialEmailCount+1, len(sender.Sent()))
	}

	// Prior active tokens should be revoked.
	now := time.Now().UTC()
	revokedCount := 0
	verifyRepo.mu.Lock()
	for _, tok := range verifyRepo.tokens {
		if tok.RevokedAt != nil && !tok.IsActive(now) {
			revokedCount++
		}
	}
	verifyRepo.mu.Unlock()
	if revokedCount < 1 {
		t.Error("expected at least one revoked token after resend")
	}
	_ = accounts
	_ = identities
}

// ── RequestPasswordReset tests ────────────────────────────────────────────────

func TestRequestPasswordReset_UnknownEmail_NilNoTokenNoEmail(t *testing.T) {
	svc, _, _, _, _, _, sender := buildSvc()
	// Access resetRepo via a dedicated build with visible reset repo.
	accounts := newFakeAccountRepo()
	identities := newFakeIdentityRepo()
	verifyRepo := newFakeVerificationRepo()
	resetRepo := newFakeResetRepo()
	accounts.identityRepo = identities
	verifyRepo.accountRepo = accounts
	verifyRepo.identityRepo = identities
	resetRepo.accountRepo = accounts
	resetRepo.identityRepo = identities
	events := &fakeEventRepo{}
	sessions := newFakeSessionStore()
	_ = sender
	svc2 := auth.NewService(accounts, identities, events, sessions, verifyRepo, &fakeEmailSender{}, "http://localhost:13000/verify-email", resetRepo, "http://localhost:13000/reset-password")

	err := svc2.RequestPasswordReset(context.Background(), "nobody@example.com")
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	resetRepo.mu.Lock()
	count := len(resetRepo.tokens)
	resetRepo.mu.Unlock()
	if count != 0 {
		t.Errorf("expected 0 reset tokens, got %d", count)
	}
	_ = svc
}

func TestRequestPasswordReset_ExistingIdentity_RevokesAndIssues(t *testing.T) {
	accounts := newFakeAccountRepo()
	identities := newFakeIdentityRepo()
	verifyRepo := newFakeVerificationRepo()
	resetRepo := newFakeResetRepo()
	accounts.identityRepo = identities
	verifyRepo.accountRepo = accounts
	verifyRepo.identityRepo = identities
	resetRepo.accountRepo = accounts
	resetRepo.identityRepo = identities
	events := &fakeEventRepo{}
	sessions := newFakeSessionStore()
	sender := &fakeEmailSender{}
	svc := auth.NewService(accounts, identities, events, sessions, verifyRepo, sender, "http://localhost:13000/verify-email", resetRepo, "http://localhost:13000/reset-password")

	seedAccount(accounts, identities, "reset@example.com", "pass1234", auth.AccountStatusActive)

	// First call — issues token + sends email.
	if err := svc.RequestPasswordReset(context.Background(), "reset@example.com"); err != nil {
		t.Fatalf("RequestPasswordReset: %v", err)
	}

	resetRepo.mu.Lock()
	count1 := len(resetRepo.tokens)
	resetRepo.mu.Unlock()
	if count1 != 1 {
		t.Fatalf("expected 1 reset token after first request, got %d", count1)
	}
	msgs1 := sender.Sent()
	if len(msgs1) != 1 {
		t.Fatalf("expected 1 email after first request, got %d", len(msgs1))
	}
	if !strings.Contains(msgs1[0].TextBody, "?token=") {
		t.Errorf("email body missing '?token=': %s", msgs1[0].TextBody)
	}

	// Second call — prior active token should be revoked, new one issued.
	if err := svc.RequestPasswordReset(context.Background(), "reset@example.com"); err != nil {
		t.Fatalf("second RequestPasswordReset: %v", err)
	}

	resetRepo.mu.Lock()
	count2 := len(resetRepo.tokens)
	now := time.Now().UTC()
	revokedCount := 0
	for _, tok := range resetRepo.tokens {
		if tok.RevokedAt != nil && !tok.IsActive(now) {
			revokedCount++
		}
	}
	resetRepo.mu.Unlock()

	if count2 != 2 {
		t.Errorf("expected 2 reset tokens total, got %d", count2)
	}
	if revokedCount < 1 {
		t.Error("expected at least 1 revoked token after second request")
	}
	msgs2 := sender.Sent()
	if len(msgs2) != 2 {
		t.Errorf("expected 2 emails total, got %d", len(msgs2))
	}
}

// ── ConfirmPasswordReset tests ─────────────────────────────────────────────────

func TestConfirmPasswordReset_ValidToken_UpdatesPasswordAndClearsLockout(t *testing.T) {
	accounts := newFakeAccountRepo()
	identities := newFakeIdentityRepo()
	verifyRepo := newFakeVerificationRepo()
	resetRepo := newFakeResetRepo()
	accounts.identityRepo = identities
	verifyRepo.accountRepo = accounts
	verifyRepo.identityRepo = identities
	resetRepo.accountRepo = accounts
	resetRepo.identityRepo = identities
	events := &fakeEventRepo{}
	sessions := newFakeSessionStore()
	sender := &fakeEmailSender{}
	svc := auth.NewService(accounts, identities, events, sessions, verifyRepo, sender, "http://localhost:13000/verify-email", resetRepo, "http://localhost:13000/reset-password")

	a := seedAccount(accounts, identities, "confpwd@example.com", "oldpassword", auth.AccountStatusActive)

	// Set a lockout on the account to verify it gets cleared.
	until := time.Now().Add(10 * time.Minute)
	accounts.mu.Lock()
	accounts.accounts[a.ID].LockedUntil = &until
	accounts.accounts[a.ID].FailedLoginCount = 3
	accounts.mu.Unlock()

	// Request a reset so we have a token.
	if err := svc.RequestPasswordReset(context.Background(), "confpwd@example.com"); err != nil {
		t.Fatalf("RequestPasswordReset: %v", err)
	}

	// Extract raw token from email.
	msgs := sender.Sent()
	if len(msgs) == 0 {
		t.Fatal("no reset email sent")
	}
	body := msgs[0].TextBody
	idx := strings.Index(body, "?token=")
	if idx < 0 {
		t.Fatalf("no ?token= in email body: %s", body)
	}
	rawToken := strings.TrimSpace(strings.SplitN(body[idx+len("?token="):], "\n", 2)[0])

	// Confirm with new password.
	newPwd := "newpassword123"
	if err := svc.ConfirmPasswordReset(context.Background(), rawToken, newPwd); err != nil {
		t.Fatalf("ConfirmPasswordReset: %v", err)
	}

	// Token should be marked used.
	h := tokenHash(rawToken)
	resetRepo.mu.Lock()
	tok := resetRepo.tokens[h]
	resetRepo.mu.Unlock()
	if tok == nil || tok.UsedAt == nil {
		t.Error("token should have used_at set after confirm")
	}

	// Identity password hash should be updated — old password should not verify.
	identities.mu.Lock()
	ident := identities.identities["confpwd@example.com"]
	newHash := ident.PasswordHash
	identities.mu.Unlock()

	if err := passwordhash.Verify(newHash, newPwd); err != nil {
		t.Errorf("new password does not verify against stored hash: %v", err)
	}
	if err := passwordhash.Verify(newHash, "oldpassword"); err == nil {
		t.Error("old password should not verify against new hash")
	}

	// Account lockout should be cleared.
	accounts.mu.Lock()
	acct := accounts.accounts[a.ID]
	accounts.mu.Unlock()
	if acct.FailedLoginCount != 0 {
		t.Errorf("failed_login_count = %d, want 0 after reset", acct.FailedLoginCount)
	}
	if acct.LockedUntil != nil {
		t.Error("locked_until should be nil after reset")
	}
}

func TestConfirmPasswordReset_WeakPassword_ValidationError(t *testing.T) {
	svc, _, _, _, _, _, _ := buildSvc()
	err := svc.ConfirmPasswordReset(context.Background(), "anytoken", "short")
	var ve *auth.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
	found := false
	for _, fe := range ve.Fields {
		if fe.Field == "new_password" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected field 'new_password' in errors: %v", ve.Fields)
	}
}

func TestConfirmPasswordReset_UnknownToken_ErrTokenInvalid(t *testing.T) {
	svc, _, _, _, _, _, _ := buildSvc()
	err := svc.ConfirmPasswordReset(context.Background(), "completelyunknowntoken", "newpassword123")
	if !errors.Is(err, auth.ErrTokenInvalid) {
		t.Errorf("err = %v, want ErrTokenInvalid", err)
	}
}

func TestConfirmPasswordReset_ExpiredToken_ErrTokenExpired(t *testing.T) {
	accounts := newFakeAccountRepo()
	identities := newFakeIdentityRepo()
	verifyRepo := newFakeVerificationRepo()
	resetRepo := newFakeResetRepo()
	accounts.identityRepo = identities
	verifyRepo.accountRepo = accounts
	verifyRepo.identityRepo = identities
	resetRepo.accountRepo = accounts
	resetRepo.identityRepo = identities
	events := &fakeEventRepo{}
	sessions := newFakeSessionStore()

	a := seedAccount(accounts, identities, "expreset@example.com", "pass1234", auth.AccountStatusActive)

	// Find the identity ID.
	identities.mu.Lock()
	ident := identities.identities["expreset@example.com"]
	identities.mu.Unlock()

	// Insert an already-expired reset token.
	rawToken := "expiredresettoken"
	h := tokenHash(rawToken)
	now := time.Now().UTC()
	expiredTok := &auth.PasswordResetToken{
		ID:             mustUUID(),
		AuthIdentityID: ident.ID,
		TokenHash:      h,
		ExpiresAt:      now.Add(-1 * time.Hour), // in the past
		CreatedAt:      now,
	}
	resetRepo.mu.Lock()
	resetRepo.tokens[h] = expiredTok
	resetRepo.mu.Unlock()

	svc := auth.NewService(accounts, identities, events, sessions, verifyRepo, &fakeEmailSender{}, "http://localhost:13000/verify-email", resetRepo, "http://localhost:13000/reset-password")

	err := svc.ConfirmPasswordReset(context.Background(), rawToken, "newpassword123")
	if !errors.Is(err, auth.ErrTokenExpired) {
		t.Errorf("err = %v, want ErrTokenExpired", err)
	}
	_ = a
}

func TestConfirmPasswordReset_AlreadyUsedToken_ErrTokenExpired(t *testing.T) {
	accounts := newFakeAccountRepo()
	identities := newFakeIdentityRepo()
	verifyRepo := newFakeVerificationRepo()
	resetRepo := newFakeResetRepo()
	accounts.identityRepo = identities
	verifyRepo.accountRepo = accounts
	verifyRepo.identityRepo = identities
	resetRepo.accountRepo = accounts
	resetRepo.identityRepo = identities
	events := &fakeEventRepo{}
	sessions := newFakeSessionStore()

	seedAccount(accounts, identities, "usedreset@example.com", "pass1234", auth.AccountStatusActive)
	identities.mu.Lock()
	ident := identities.identities["usedreset@example.com"]
	identities.mu.Unlock()

	// Insert an already-used reset token.
	rawToken := "usedresettoken12345"
	h := tokenHash(rawToken)
	now := time.Now().UTC()
	usedTok := &auth.PasswordResetToken{
		ID:             mustUUID(),
		AuthIdentityID: ident.ID,
		TokenHash:      h,
		ExpiresAt:      now.Add(1 * time.Hour),
		UsedAt:         &now, // already used
		CreatedAt:      now,
	}
	resetRepo.mu.Lock()
	resetRepo.tokens[h] = usedTok
	resetRepo.mu.Unlock()

	svc := auth.NewService(accounts, identities, events, sessions, verifyRepo, &fakeEmailSender{}, "http://localhost:13000/verify-email", resetRepo, "http://localhost:13000/reset-password")

	err := svc.ConfirmPasswordReset(context.Background(), rawToken, "newpassword123")
	if !errors.Is(err, auth.ErrTokenExpired) {
		t.Errorf("already-used token: err = %v, want ErrTokenExpired", err)
	}
}

// ── ResolveSessionAccount (lazy session revocation, 5c) ────────────────────────

func TestResolveSessionAccount_NoPasswordChange_ReturnsAccount(t *testing.T) {
	svc, accounts, identities, _, sessions, _, _ := buildSvc()
	a := seedAccount(accounts, identities, "sess1@example.com", "pw", auth.AccountStatusActive)

	raw := "raw-token-1"
	sessions.sessions[tokenHash(raw)] = auth.SessionRecord{AccountID: a.ID, CreatedAt: time.Now().UTC().Add(-time.Hour)}

	got, err := svc.ResolveSessionAccount(context.Background(), raw)
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got == nil || got.ID != a.ID {
		t.Errorf("account mismatch")
	}
}

func TestResolveSessionAccount_SessionOlderThanPasswordChange_RevokesAnd401(t *testing.T) {
	svc, accounts, identities, _, sessions, _, _ := buildSvc()
	a := seedAccount(accounts, identities, "sess2@example.com", "pw", auth.AccountStatusActive)

	now := time.Now().UTC()
	// password changed AFTER the session was created → stale session.
	changed := now
	identities.identities["sess2@example.com"].PasswordChangedAt = &changed

	raw := "raw-token-2"
	hash := tokenHash(raw)
	sessions.sessions[hash] = auth.SessionRecord{AccountID: a.ID, CreatedAt: now.Add(-time.Hour)}

	_, err := svc.ResolveSessionAccount(context.Background(), raw)
	if !errors.Is(err, auth.ErrUnauthenticated) {
		t.Fatalf("err = %v, want ErrUnauthenticated", err)
	}
	// Session must have been deleted (lazy revoke).
	sessions.mu.Lock()
	_, exists := sessions.sessions[hash]
	sessions.mu.Unlock()
	if exists {
		t.Error("stale session should have been deleted")
	}
}

func TestResolveSessionAccount_SessionNewerThanPasswordChange_ReturnsAccount(t *testing.T) {
	svc, accounts, identities, _, sessions, _, _ := buildSvc()
	a := seedAccount(accounts, identities, "sess3@example.com", "pw", auth.AccountStatusActive)

	now := time.Now().UTC()
	changed := now.Add(-time.Hour) // password changed BEFORE session created
	identities.identities["sess3@example.com"].PasswordChangedAt = &changed

	raw := "raw-token-3"
	sessions.sessions[tokenHash(raw)] = auth.SessionRecord{AccountID: a.ID, CreatedAt: now}

	got, err := svc.ResolveSessionAccount(context.Background(), raw)
	if err != nil || got == nil {
		t.Fatalf("err = %v, account = %v; want account, nil err", err, got)
	}
}

func TestResolveSessionAccount_NoSession_Unauthenticated(t *testing.T) {
	svc, _, _, _, _, _, _ := buildSvc()
	_, err := svc.ResolveSessionAccount(context.Background(), "does-not-exist")
	if !errors.Is(err, auth.ErrUnauthenticated) {
		t.Errorf("err = %v, want ErrUnauthenticated", err)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func mustUUID() uuid.UUID {
	id, _ := uuid.NewV7()
	return id
}
