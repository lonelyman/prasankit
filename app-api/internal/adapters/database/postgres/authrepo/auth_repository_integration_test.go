package authrepo

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/pkg/ids"

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

	t.Run("find user account by email", func(t *testing.T) {
		email := "repo-test-" + uuid.NewString() + "@example.test"
		_, err := repo.FindUserAccountByEmail(ctx, email)
		if !errors.Is(err, auth.ErrUserAccountNotFound) {
			t.Fatalf("err = %v, want ErrUserAccountNotFound", err)
		}

		accountID, err := ids.NewUUID()
		if err != nil {
			t.Fatalf("new account id: %v", err)
		}
		if err := db.Exec(
			`INSERT INTO user_accounts (id, email, password_hash, status) VALUES (?, ?, ?, ?)`,
			accountID,
			email,
			"test-password-hash",
			string(auth.UserAccountStatusActive),
		).Error; err != nil {
			t.Fatalf("insert user account: %v", err)
		}
		t.Cleanup(func() {
			_ = db.Exec(`DELETE FROM user_accounts WHERE email = ?`, email).Error
		})

		account, err := repo.FindUserAccountByEmail(ctx, email)
		if err != nil {
			t.Fatalf("find user account: %v", err)
		}
		if account.Email != email {
			t.Fatalf("account.Email = %s, want %s", account.Email, email)
		}
		if account.Status != auth.UserAccountStatusActive {
			t.Fatalf("account.Status = %s, want %s", account.Status, auth.UserAccountStatusActive)
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
