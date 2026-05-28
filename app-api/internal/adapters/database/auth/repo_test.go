package authdbrepo_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	authdbrepo "prasankit-api/internal/adapters/database/auth"
	"prasankit-api/internal/modules/auth"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// openTestDB opens a GORM DB using the env vars from .env.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	host := envOr("POSTGRES_PRIMARY_HOST", "localhost")
	port := envOr("POSTGRES_PRIMARY_PORT", "15433")
	user := envOr("POSTGRES_PRIMARY_USER", "prasankit")
	password := envOr("POSTGRES_PRIMARY_PASSWORD", "change_me")
	name := envOr("POSTGRES_PRIMARY_NAME", "prasankit")
	sslMode := envOr("POSTGRES_SSL_MODE", "disable")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, name, sslMode,
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skipf("postgres not reachable (%v): skipping integration test", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Skipf("get sqlDB: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		t.Skipf("postgres ping failed (%v): skipping integration test", err)
	}
	return db
}

// truncateTestTables cleans data tables; never truncates master/seed tables.
func truncateTestTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	tables := []string{
		"auth_email_verification_tokens",
		"security_events",
		"auth_identities",
		"user_accounts",
	}
	for _, tbl := range tables {
		if err := db.Exec("TRUNCATE TABLE " + tbl + " CASCADE").Error; err != nil {
			t.Logf("truncate %s: %v", tbl, err)
		}
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func buildAccount(email string) auth.Account {
	id, _ := ids.New()
	now := time.Now().UTC()
	return auth.Account{
		ID:                id,
		PrimaryEmail:      email,
		DisplayName:       "Test User",
		AccountStatusCode: auth.AccountStatusPendingVerification,
		FailedLoginCount:  0,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func TestAccountRepo_CreateWithIdentity_Atomic(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	accountRepo := authdbrepo.NewAccountRepo(db)
	identityRepo := authdbrepo.NewIdentityRepo(db)

	email := "test+" + fmt.Sprintf("%d", time.Now().UnixNano()) + "@example.com"
	account := buildAccount(email)
	identity := buildIdentityFor(account, email)

	if err := accountRepo.CreateWithIdentity(context.Background(), account, identity); err != nil {
		t.Fatalf("CreateWithIdentity: %v", err)
	}

	// Account is findable.
	found, err := accountRepo.FindByID(context.Background(), account.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found == nil {
		t.Fatal("account not found after create")
	}
	if found.PrimaryEmail != email {
		t.Errorf("email = %q, want %q", found.PrimaryEmail, email)
	}

	// Identity is findable.
	ident, err := identityRepo.FindByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("FindByEmail: %v", err)
	}
	if ident == nil {
		t.Fatal("identity not found after create")
	}
	if ident.UserAccountID != account.ID {
		t.Errorf("identity.user_account_id = %v, want %v", ident.UserAccountID, account.ID)
	}
}

func TestAccountRepo_DuplicateEmail_UniqueConstraint(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	accountRepo := authdbrepo.NewAccountRepo(db)

	email := "dup+" + fmt.Sprintf("%d", time.Now().UnixNano()) + "@example.com"
	account1 := buildAccount(email)
	identity1 := buildIdentityFor(account1, email)

	if err := accountRepo.CreateWithIdentity(context.Background(), account1, identity1); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	account2 := buildAccount(email) // same email
	identity2 := buildIdentityFor(account2, email)

	err := accountRepo.CreateWithIdentity(context.Background(), account2, identity2)
	if !errors.Is(err, auth.ErrEmailTaken) {
		t.Errorf("err = %v, want ErrEmailTaken", err)
	}
}

func TestAccountRepo_IncrementFailedLogin(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	accountRepo := authdbrepo.NewAccountRepo(db)

	email := "incr+" + fmt.Sprintf("%d", time.Now().UnixNano()) + "@example.com"
	account := buildAccount(email)
	identity := buildIdentityFor(account, email)

	if err := accountRepo.CreateWithIdentity(context.Background(), account, identity); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Increment below threshold — no lockout.
	for i := 0; i < 3; i++ {
		if err := accountRepo.IncrementFailedLogin(context.Background(), account.ID, 5, 15*time.Minute); err != nil {
			t.Fatalf("IncrementFailedLogin: %v", err)
		}
	}
	found, _ := accountRepo.FindByID(context.Background(), account.ID)
	if found.FailedLoginCount != 3 {
		t.Errorf("failed_login_count = %d, want 3", found.FailedLoginCount)
	}
	if found.LockedUntil != nil {
		t.Error("should not be locked yet")
	}

	// Increment to threshold — should lock.
	for i := 3; i < 5; i++ {
		if err := accountRepo.IncrementFailedLogin(context.Background(), account.ID, 5, 15*time.Minute); err != nil {
			t.Fatalf("IncrementFailedLogin: %v", err)
		}
	}
	found, _ = accountRepo.FindByID(context.Background(), account.ID)
	if found.LockedUntil == nil {
		t.Error("account should be locked after reaching threshold")
	}
}

// buildIdentityFor constructs an Identity with the correct UserAccountID.
func buildIdentityFor(account auth.Account, email string) auth.Identity {
	iid, _ := ids.New()
	now := time.Now().UTC()
	return auth.Identity{
		ID:               iid,
		UserAccountID:    account.ID,
		IdentityTypeCode: auth.IdentityTypeEmailPassword,
		Email:            email,
		PasswordHash:     "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// buildVerificationToken creates a test verification token for the given identity.
func buildVerificationToken(identityID uuid.UUID, ttl time.Duration) auth.EmailVerificationToken {
	tokenID, _ := ids.New()
	now := time.Now().UTC()
	return auth.EmailVerificationToken{
		ID:             tokenID,
		AuthIdentityID: identityID,
		TokenHash:      fmt.Sprintf("testhash-%d", time.Now().UnixNano()),
		ExpiresAt:      now.Add(ttl),
		CreatedAt:      now,
	}
}

// setupAccountWithIdentity creates an account + identity and returns both.
func setupAccountWithIdentity(t *testing.T, accountRepo *authdbrepo.AccountRepo, email string) (auth.Account, auth.Identity) {
	t.Helper()
	account := buildAccount(email)
	identity := buildIdentityFor(account, email)
	if err := accountRepo.CreateWithIdentity(context.Background(), account, identity); err != nil {
		t.Fatalf("setupAccountWithIdentity: %v", err)
	}
	return account, identity
}

func TestVerificationTokenRepo_CreateAndFind(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	accountRepo := authdbrepo.NewAccountRepo(db)
	verifyRepo := authdbrepo.NewVerificationTokenRepo(db)

	email := "vtok+" + fmt.Sprintf("%d", time.Now().UnixNano()) + "@example.com"
	_, identity := setupAccountWithIdentity(t, accountRepo, email)

	tok := buildVerificationToken(identity.ID, 24*time.Hour)
	if err := verifyRepo.Create(context.Background(), tok); err != nil {
		t.Fatalf("Create: %v", err)
	}

	found, err := verifyRepo.FindByTokenHash(context.Background(), tok.TokenHash)
	if err != nil {
		t.Fatalf("FindByTokenHash: %v", err)
	}
	if found == nil {
		t.Fatal("expected token, got nil")
	}
	if found.ID != tok.ID {
		t.Errorf("id = %v, want %v", found.ID, tok.ID)
	}
	if found.AuthIdentityID != identity.ID {
		t.Errorf("auth_identity_id = %v, want %v", found.AuthIdentityID, identity.ID)
	}
}

func TestVerificationTokenRepo_FindByTokenHash_NotFound(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	verifyRepo := authdbrepo.NewVerificationTokenRepo(db)
	found, err := verifyRepo.FindByTokenHash(context.Background(), "nonexistenthash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Error("expected nil for unknown hash")
	}
}

func TestVerificationTokenRepo_RevokeActiveByIdentity(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	accountRepo := authdbrepo.NewAccountRepo(db)
	verifyRepo := authdbrepo.NewVerificationTokenRepo(db)

	email := "revoke+" + fmt.Sprintf("%d", time.Now().UnixNano()) + "@example.com"
	_, identity := setupAccountWithIdentity(t, accountRepo, email)

	// Create two active tokens.
	tok1 := buildVerificationToken(identity.ID, 24*time.Hour)
	tok2 := buildVerificationToken(identity.ID, 24*time.Hour)
	if err := verifyRepo.Create(context.Background(), tok1); err != nil {
		t.Fatalf("create tok1: %v", err)
	}
	if err := verifyRepo.Create(context.Background(), tok2); err != nil {
		t.Fatalf("create tok2: %v", err)
	}

	now := time.Now().UTC()
	if err := verifyRepo.RevokeActiveByIdentity(context.Background(), identity.ID, now); err != nil {
		t.Fatalf("RevokeActiveByIdentity: %v", err)
	}

	// Both tokens should now be inactive.
	found1, _ := verifyRepo.FindByTokenHash(context.Background(), tok1.TokenHash)
	found2, _ := verifyRepo.FindByTokenHash(context.Background(), tok2.TokenHash)
	if found1 == nil || found2 == nil {
		t.Fatal("tokens should still exist after revoke")
	}
	if found1.IsActive(now) {
		t.Error("tok1 should not be active after revoke")
	}
	if found2.IsActive(now) {
		t.Error("tok2 should not be active after revoke")
	}
}

func TestVerificationTokenRepo_ConfirmTx_HappyPath(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	accountRepo := authdbrepo.NewAccountRepo(db)
	identityRepo := authdbrepo.NewIdentityRepo(db)
	verifyRepo := authdbrepo.NewVerificationTokenRepo(db)

	email := "confirm+" + fmt.Sprintf("%d", time.Now().UnixNano()) + "@example.com"
	account, identity := setupAccountWithIdentity(t, accountRepo, email)

	tok := buildVerificationToken(identity.ID, 24*time.Hour)
	if err := verifyRepo.Create(context.Background(), tok); err != nil {
		t.Fatalf("create token: %v", err)
	}

	now := time.Now().UTC()
	if err := verifyRepo.ConfirmTx(context.Background(), tok.ID, identity.ID, account.ID, now); err != nil {
		t.Fatalf("ConfirmTx: %v", err)
	}

	// Token should be used.
	found, _ := verifyRepo.FindByTokenHash(context.Background(), tok.TokenHash)
	if found == nil || found.UsedAt == nil {
		t.Error("token should have used_at set after ConfirmTx")
	}

	// Identity should be verified.
	ident, _ := identityRepo.FindByID(context.Background(), identity.ID)
	if ident == nil || ident.EmailVerifiedAt == nil {
		t.Error("identity email_verified_at should be set after ConfirmTx")
	}

	// Account should be active.
	acct, _ := accountRepo.FindByID(context.Background(), account.ID)
	if acct == nil || acct.AccountStatusCode != auth.AccountStatusActive {
		t.Errorf("account status = %q, want active", acct.AccountStatusCode)
	}
}

func TestVerificationTokenRepo_ConfirmTx_ConcurrentLoser(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	accountRepo := authdbrepo.NewAccountRepo(db)
	verifyRepo := authdbrepo.NewVerificationTokenRepo(db)

	email := "concurrent+" + fmt.Sprintf("%d", time.Now().UnixNano()) + "@example.com"
	account, identity := setupAccountWithIdentity(t, accountRepo, email)

	tok := buildVerificationToken(identity.ID, 24*time.Hour)
	if err := verifyRepo.Create(context.Background(), tok); err != nil {
		t.Fatalf("create token: %v", err)
	}

	now := time.Now().UTC()
	// First confirm succeeds.
	if err := verifyRepo.ConfirmTx(context.Background(), tok.ID, identity.ID, account.ID, now); err != nil {
		t.Fatalf("first ConfirmTx: %v", err)
	}

	// Second confirm should return ErrTokenExpired (concurrent-loser case).
	err := verifyRepo.ConfirmTx(context.Background(), tok.ID, identity.ID, account.ID, now)
	if !errors.Is(err, auth.ErrTokenExpired) {
		t.Errorf("second ConfirmTx: err = %v, want ErrTokenExpired", err)
	}
}

func TestVerificationTokenRepo_ConfirmTx_DoesNotFlipSuspended(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	accountRepo := authdbrepo.NewAccountRepo(db)
	identityRepo := authdbrepo.NewIdentityRepo(db)
	verifyRepo := authdbrepo.NewVerificationTokenRepo(db)

	// Build account directly in suspended status.
	email := "suspended+" + fmt.Sprintf("%d", time.Now().UnixNano()) + "@example.com"
	account := buildAccount(email)
	account.AccountStatusCode = auth.AccountStatusSuspended
	identity := buildIdentityFor(account, email)
	if err := accountRepo.CreateWithIdentity(context.Background(), account, identity); err != nil {
		t.Fatalf("setup: %v", err)
	}

	tok := buildVerificationToken(identity.ID, 24*time.Hour)
	if err := verifyRepo.Create(context.Background(), tok); err != nil {
		t.Fatalf("create token: %v", err)
	}

	now := time.Now().UTC()
	if err := verifyRepo.ConfirmTx(context.Background(), tok.ID, identity.ID, account.ID, now); err != nil {
		t.Fatalf("ConfirmTx: %v", err)
	}

	// Identity should be verified.
	ident, _ := identityRepo.FindByID(context.Background(), identity.ID)
	if ident == nil || ident.EmailVerifiedAt == nil {
		t.Error("identity email_verified_at should be set")
	}

	// Account must remain suspended — not flipped to active.
	acct, _ := accountRepo.FindByID(context.Background(), account.ID)
	if acct == nil || acct.AccountStatusCode != auth.AccountStatusSuspended {
		t.Errorf("account status = %q, want suspended (must not be flipped)", acct.AccountStatusCode)
	}
}
