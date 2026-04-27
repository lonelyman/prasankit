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
			CreatedAt:     time.Now().UTC(),
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

		var count int64
		if err := db.Table("auth_login_attempts").Where("email = ?", email).Count(&count).Error; err != nil {
			t.Fatalf("count login attempts: %v", err)
		}
		if count != 1 {
			t.Fatalf("login attempt count = %d, want 1", count)
		}
	})
}
