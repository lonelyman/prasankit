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
