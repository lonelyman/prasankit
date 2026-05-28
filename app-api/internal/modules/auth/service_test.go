package auth_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"prasankit-api/internal/modules/auth"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ── Fake implementations of ports ────────────────────────────────────────────

type fakeAccountRepo struct {
	mu       sync.Mutex
	accounts map[uuid.UUID]*auth.Account
}

func newFakeAccountRepo() *fakeAccountRepo {
	return &fakeAccountRepo{accounts: map[uuid.UUID]*auth.Account{}}
}

func (r *fakeAccountRepo) CreateWithIdentity(ctx context.Context, account auth.Account, _ auth.Identity) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, a := range r.accounts {
		if a.PrimaryEmail == account.PrimaryEmail {
			return auth.ErrEmailTaken
		}
	}
	cp := account
	r.accounts[account.ID] = &cp
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
}

func newFakeIdentityRepo() *fakeIdentityRepo {
	return &fakeIdentityRepo{identities: map[string]*auth.Identity{}}
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

// ── Helpers ───────────────────────────────────────────────────────────────────

func buildSvc() (*auth.Service, *fakeAccountRepo, *fakeIdentityRepo, *fakeEventRepo, *fakeSessionStore) {
	accounts := newFakeAccountRepo()
	identities := newFakeIdentityRepo()
	events := &fakeEventRepo{}
	sessions := newFakeSessionStore()
	svc := auth.NewService(accounts, identities, events, sessions)
	return svc, accounts, identities, events, sessions
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

	identities.mu.Lock()
	identities.identities[email] = &auth.Identity{
		ID:               iid,
		UserAccountID:    id,
		IdentityTypeCode: auth.IdentityTypeEmailPassword,
		Email:            email,
		PasswordHash:     string(hash),
	}
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
	svc, accounts, _, events, _ := buildSvc()
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
	if len(events.events) != 1 || events.events[0].EventType != auth.SecurityEventAccountCreated {
		t.Errorf("security event not logged: %v", events.events)
	}
}

func TestSignup_DuplicateEmail(t *testing.T) {
	svc, accounts, identities, _, _ := buildSvc()
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
	svc, _, _, _, _ := buildSvc()
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
	svc, accounts, identities, events, sessions := buildSvc()
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

func TestLogin_WrongPassword_CounterIncrements(t *testing.T) {
	svc, accounts, identities, _, _ := buildSvc()
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
	svc, accounts, identities, _, _ := buildSvc()
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
	svc, accounts, identities, _, _ := buildSvc()
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
	svc, accounts, identities, _, _ := buildSvc()
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
	svc, _, _, _, _ := buildSvc()

	_, err := svc.Login(context.Background(), auth.LoginInput{
		Email:    "nobody@example.com",
		Password: "anything",
	})
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("err = %v, want ErrInvalidCredentials", err)
	}
}

func TestLogout(t *testing.T) {
	svc, accounts, identities, _, sessions := buildSvc()
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
