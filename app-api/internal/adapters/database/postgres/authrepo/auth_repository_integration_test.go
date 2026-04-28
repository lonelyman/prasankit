package authrepo

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"prasankit-api/internal/modules/auth"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestRepositoryIntegration(t *testing.T) {
	dsn := os.Getenv("PRASANKIT_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("PRASANKIT_TEST_DB_DSN is not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	defer sqlDB.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	t.Run("create user account", func(t *testing.T) {
		email := "create-account-" + uuid.NewString() + "@example.test"
		account := &auth.UserAccount{
			PrimaryEmail: email,
		}

		if err := repo.CreateUserAccount(ctx, account); err != nil {
			t.Fatalf("create user account: %v", err)
		}
		t.Cleanup(func() {
			_ = db.Exec(`DELETE FROM user_accounts WHERE id = ?`, account.ID).Error
		})

		if account.ID == uuid.Nil {
			t.Fatal("account.ID was not set")
		}
		if account.ID.Version() != 7 {
			t.Fatalf("account.ID version = %d, want 7", account.ID.Version())
		}
		if account.Status != auth.UserAccountStatusPendingVerification {
			t.Fatalf("account.Status = %s, want %s", account.Status, auth.UserAccountStatusPendingVerification)
		}
		if account.CreatedAt.IsZero() {
			t.Fatal("account.CreatedAt was not set")
		}
		if account.UpdatedAt.IsZero() {
			t.Fatal("account.UpdatedAt was not set")
		}

		var storedPrimaryEmail string
		if err := db.Table("user_accounts").
			Select("primary_email").
			Where("id = ?", account.ID).
			Scan(&storedPrimaryEmail).
			Error; err != nil {
			t.Fatalf("query created user account: %v", err)
		}
		if storedPrimaryEmail != email {
			t.Fatalf("primary_email = %s, want %s", storedPrimaryEmail, email)
		}
	})

	t.Run("create auth identity", func(t *testing.T) {
		email := "identity-" + uuid.NewString() + "@example.test"
		account := &auth.UserAccount{
			PrimaryEmail: email,
		}
		if err := repo.CreateUserAccount(ctx, account); err != nil {
			t.Fatalf("create user account: %v", err)
		}
		t.Cleanup(func() {
			_ = db.Exec(`DELETE FROM auth_identities WHERE email = ?`, email).Error
			_ = db.Exec(`DELETE FROM user_accounts WHERE id = ?`, account.ID).Error
		})

		identity := &auth.AuthIdentity{
			UserAccountID: account.ID,
			Email:         email,
			PasswordHash:  "test-password-hash",
		}
		if err := repo.CreateAuthIdentity(ctx, identity); err != nil {
			t.Fatalf("create auth identity: %v", err)
		}

		if identity.ID == uuid.Nil {
			t.Fatal("identity.ID was not set")
		}
		if identity.ID.Version() != 7 {
			t.Fatalf("identity.ID version = %d, want 7", identity.ID.Version())
		}
		if identity.IdentityType != auth.AuthIdentityTypeEmailPassword {
			t.Fatalf("identity.IdentityType = %s, want %s", identity.IdentityType, auth.AuthIdentityTypeEmailPassword)
		}
		if identity.Provider != auth.AuthProviderEmail {
			t.Fatalf("identity.Provider = %s, want %s", identity.Provider, auth.AuthProviderEmail)
		}
		if identity.CreatedAt.IsZero() {
			t.Fatal("identity.CreatedAt was not set")
		}
		if identity.UpdatedAt.IsZero() {
			t.Fatal("identity.UpdatedAt was not set")
		}
	})

	t.Run("find user account by email", func(t *testing.T) {
		email := "repo-test-" + uuid.NewString() + "@example.test"
		_, err := repo.FindUserAccountByEmail(ctx, email)
		if !errors.Is(err, auth.ErrUserAccountNotFound) {
			t.Fatalf("err = %v, want ErrUserAccountNotFound", err)
		}

		account := &auth.UserAccount{
			PrimaryEmail: email,
			Status:       auth.UserAccountStatusActive,
		}
		if err := repo.CreateUserAccount(ctx, account); err != nil {
			t.Fatalf("create user account: %v", err)
		}
		identity := &auth.AuthIdentity{
			UserAccountID: account.ID,
			Email:         email,
			PasswordHash:  "test-password-hash",
		}
		if err := repo.CreateAuthIdentity(ctx, identity); err != nil {
			t.Fatalf("create auth identity: %v", err)
		}
		t.Cleanup(func() {
			_ = db.Exec(`DELETE FROM auth_identities WHERE email = ?`, email).Error
			_ = db.Exec(`DELETE FROM user_accounts WHERE id = ?`, account.ID).Error
		})

		account, err = repo.FindUserAccountByEmail(ctx, email)
		if err != nil {
			t.Fatalf("find user account: %v", err)
		}
		if account.PrimaryEmail != email {
			t.Fatalf("account.PrimaryEmail = %s, want %s", account.PrimaryEmail, email)
		}
		if account.Status != auth.UserAccountStatusActive {
			t.Fatalf("account.Status = %s, want %s", account.Status, auth.UserAccountStatusActive)
		}
	})

	t.Run("create auth session", func(t *testing.T) {
		email := "session-" + uuid.NewString() + "@example.test"
		account := &auth.UserAccount{
			PrimaryEmail: email,
			Status:       auth.UserAccountStatusActive,
		}
		if err := repo.CreateUserAccount(ctx, account); err != nil {
			t.Fatalf("create user account: %v", err)
		}
		t.Cleanup(func() {
			_ = db.Exec(`DELETE FROM auth_sessions WHERE user_account_id = ?`, account.ID).Error
			_ = db.Exec(`DELETE FROM user_accounts WHERE id = ?`, account.ID).Error
		})

		session := &auth.AuthSession{
			UserAccountID:  account.ID,
			SessionKeyHash: "session-hash-" + uuid.NewString(),
			IPAddress:      "127.0.0.1",
			UserAgent:      "repository integration test",
			DeviceLabel:    "test browser",
			ExpiresAt:      time.Now().UTC().Add(24 * time.Hour),
			MetadataJSON: map[string]any{
				"source": "integration_test",
			},
		}

		if err := repo.CreateAuthSession(ctx, session); err != nil {
			t.Fatalf("create auth session: %v", err)
		}

		if session.ID == uuid.Nil {
			t.Fatal("session.ID was not set")
		}
		if session.ID.Version() != 7 {
			t.Fatalf("session.ID version = %d, want 7", session.ID.Version())
		}
		if session.Status != auth.AuthSessionStatusActive {
			t.Fatalf("session.Status = %s, want %s", session.Status, auth.AuthSessionStatusActive)
		}
		if session.CreatedAt.IsZero() {
			t.Fatal("session.CreatedAt was not set")
		}

		var stored struct {
			Status string
			Source string
		}
		if err := db.Raw(
			`SELECT status, metadata_json->>'source' AS source FROM auth_sessions WHERE id = ?`,
			session.ID,
		).Scan(&stored).Error; err != nil {
			t.Fatalf("query auth session: %v", err)
		}
		if stored.Status != string(auth.AuthSessionStatusActive) {
			t.Fatalf("stored status = %s, want %s", stored.Status, auth.AuthSessionStatusActive)
		}
		if stored.Source != "integration_test" {
			t.Fatalf("stored source = %s, want integration_test", stored.Source)
		}
	})

	t.Run("create auth session requires expires_at", func(t *testing.T) {
		session := &auth.AuthSession{
			UserAccountID:  uuid.New(),
			SessionKeyHash: "session-hash-" + uuid.NewString(),
		}

		err := repo.CreateAuthSession(ctx, session)
		if !errors.Is(err, auth.ErrAuthSessionExpiresAtRequired) {
			t.Fatalf("err = %v, want ErrAuthSessionExpiresAtRequired", err)
		}
	})

	t.Run("create login attempt", func(t *testing.T) {
		email := "login-attempt-" + uuid.NewString() + "@example.test"
		attempt := &auth.LoginAttempt{
			Email:         email,
			Success:       false,
			FailureReason: "invalid_password",
			IPAddress:     "127.0.0.1",
			UserAgent:     "repository integration test",
		}

		if err := repo.CreateLoginAttempt(ctx, attempt); err != nil {
			t.Fatalf("create login attempt: %v", err)
		}
		t.Cleanup(func() {
			_ = db.Exec(`DELETE FROM auth_login_attempts WHERE email = ?`, email).Error
		})

		if attempt.ID == uuid.Nil {
			t.Fatal("attempt.ID was not set")
		}
		if attempt.ID.Version() != 7 {
			t.Fatalf("attempt.ID version = %d, want 7", attempt.ID.Version())
		}
		if attempt.CreatedAt.IsZero() {
			t.Fatal("attempt.CreatedAt was not set")
		}

		var count int64
		if err := db.Table("auth_login_attempts").Where("email = ?", email).Count(&count).Error; err != nil {
			t.Fatalf("count login attempts: %v", err)
		}
		if count != 1 {
			t.Fatalf("login attempt count = %d, want 1", count)
		}
	})

	t.Run("create security event", func(t *testing.T) {
		eventType := "repository.integration." + uuid.NewString()
		event := &auth.SecurityEvent{
			EventType: eventType,
			Severity:  auth.SecurityEventSeverityWarning,
			IPAddress: "127.0.0.1",
			UserAgent: "repository integration test",
			MetadataJSON: map[string]any{
				"reason": "invalid_password",
			},
			CreatedAt: time.Now().UTC(),
		}

		if err := repo.CreateSecurityEvent(ctx, event); err != nil {
			t.Fatalf("create security event: %v", err)
		}
		t.Cleanup(func() {
			_ = db.Exec(`DELETE FROM security_events WHERE event_type = ?`, eventType).Error
		})

		if event.ID == uuid.Nil {
			t.Fatal("event.ID was not set")
		}
		if event.ID.Version() != 7 {
			t.Fatalf("event.ID version = %d, want 7", event.ID.Version())
		}

		var stored struct {
			Severity string
			Reason   string
		}
		if err := db.Raw(
			`SELECT severity, metadata_json->>'reason' AS reason FROM security_events WHERE id = ?`,
			event.ID,
		).Scan(&stored).Error; err != nil {
			t.Fatalf("query security event: %v", err)
		}
		if stored.Severity != string(auth.SecurityEventSeverityWarning) {
			t.Fatalf("stored severity = %s, want %s", stored.Severity, auth.SecurityEventSeverityWarning)
		}
		if stored.Reason != "invalid_password" {
			t.Fatalf("stored reason = %s, want invalid_password", stored.Reason)
		}
	})
}
