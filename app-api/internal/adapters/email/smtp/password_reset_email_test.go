package smtp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	sessionstore "prasankit-api/internal/adapters/cache/session"
	authdbrepo "prasankit-api/internal/adapters/database/auth"
	smtpadapter "prasankit-api/internal/adapters/email/smtp"
	"prasankit-api/internal/modules/auth"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// truncatePasswordResetTables cleans all data tables used by the password reset email test.
func truncatePasswordResetTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, tbl := range []string{
		"auth_password_reset_tokens",
		"security_events",
		"auth_identities",
		"user_accounts",
	} {
		db.Exec("TRUNCATE TABLE " + tbl + " CASCADE")
	}
}

// openPasswordResetTestRedis opens Redis for the password reset email integration test.
func openPasswordResetTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr:     invEnvOr("REDIS_HOST", "localhost") + ":" + invEnvOr("REDIS_PORT", "16380"),
		Password: invEnvOr("REDIS_PASSWORD", "change_me"),
		DB:       0,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not reachable (%v): skipping password reset email integration test", err)
	}
	return client
}

// TestRequestPasswordReset_EmailArrivesInMailpit drives the real auth.Service.RequestPasswordReset
// with real repos + real SMTP sender pointed at Mailpit, and asserts the
// password reset email lands in Mailpit addressed to the user, subject contains
// "reset" (case-insensitive), and body contains ?token= and the reset base URL.
func TestRequestPasswordReset_EmailArrivesInMailpit(t *testing.T) {
	smtpPort := mailpitSMTPPort()
	uiPort := mailpitUIPort()

	ensureMailpit(t, smtpPort)
	clearMailpit(t, uiPort)

	db := openInviteTestDB(t)
	truncatePasswordResetTables(t, db)
	rc := openPasswordResetTestRedis(t)

	t.Cleanup(func() {
		truncatePasswordResetTables(t, db)
		clearMailpit(t, uiPort)
		rc.Close()
	})

	// Wire real repos and real SMTP sender.
	accountRepo := authdbrepo.NewAccountRepo(db)
	identityRepo := authdbrepo.NewIdentityRepo(db)
	eventRepo := authdbrepo.NewSecurityEventRepo(db)
	verifyRepo := authdbrepo.NewVerificationTokenRepo(db)
	resetRepo := authdbrepo.NewPasswordResetTokenRepo(db)
	sessStore := sessionstore.NewStore(rc)

	resetBaseURL := "http://localhost:13000/reset-password"
	sender := smtpadapter.New(smtpadapter.Config{
		Host:        "localhost",
		Port:        smtpPort,
		FromAddress: "noreply@prasankit.local",
		FromName:    "Prasankit",
	})

	svc := auth.NewService(
		accountRepo,
		identityRepo,
		eventRepo,
		sessStore,
		verifyRepo,
		sender,
		"http://localhost:13000/verify-email",
		resetRepo,
		resetBaseURL,
	)

	// Seed an account with a known email.
	userEmail := fmt.Sprintf("reset-test+%d@example.com", time.Now().UnixNano())
	createInviteTestAccount(t, db, userEmail)

	// Activate the account so the service can find and process it.
	if err := db.Exec(
		"UPDATE user_accounts SET account_status_code = 'active', updated_at = now() WHERE primary_email = ?",
		userEmail,
	).Error; err != nil {
		t.Fatalf("activate account: %v", err)
	}

	if err := svc.RequestPasswordReset(context.Background(), userEmail); err != nil {
		t.Fatalf("RequestPasswordReset: %v", err)
	}

	// Send is synchronous — email is present immediately. Small pause for Mailpit to index.
	time.Sleep(300 * time.Millisecond)

	// Assert via Mailpit API.
	apiURL := fmt.Sprintf("http://localhost:%s/api/v1/messages", uiPort)
	resp, err := http.Get(apiURL)
	if err != nil {
		t.Fatalf("Mailpit API GET failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Mailpit API returned status %d", resp.StatusCode)
	}

	var listResult struct {
		Messages []struct {
			ID      string `json:"ID"`
			Subject string `json:"Subject"`
			To      []struct {
				Address string `json:"Address"`
			} `json:"To"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResult); err != nil {
		t.Fatalf("decode Mailpit response: %v", err)
	}
	if len(listResult.Messages) == 0 {
		t.Fatal("expected at least 1 message in Mailpit, got 0")
	}

	var msgID, msgSubject string
	for _, msg := range listResult.Messages {
		if len(msg.To) == 0 {
			continue
		}
		if strings.EqualFold(msg.To[0].Address, userEmail) {
			msgID = msg.ID
			msgSubject = msg.Subject
			break
		}
	}
	if msgID == "" {
		t.Fatalf("no message addressed to %q found in Mailpit (got %d messages)", userEmail, len(listResult.Messages))
	}

	// Subject must contain "reset".
	if !strings.Contains(strings.ToLower(msgSubject), "reset") {
		t.Errorf("subject %q does not contain 'reset'", msgSubject)
	}

	// Fetch body and assert ?token= and reset base URL are present.
	bodyURL := fmt.Sprintf("http://localhost:%s/api/v1/message/%s", uiPort, msgID)
	bodyResp, err := http.Get(bodyURL)
	if err != nil {
		t.Fatalf("Mailpit body GET failed: %v", err)
	}
	defer bodyResp.Body.Close()
	var bodyResult struct {
		Text string `json:"Text"`
	}
	if err := json.NewDecoder(bodyResp.Body).Decode(&bodyResult); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if !strings.Contains(bodyResult.Text, "?token=") {
		t.Errorf("password reset email body missing '?token=': %s", bodyResult.Text)
	}
	if !strings.Contains(bodyResult.Text, resetBaseURL) {
		t.Errorf("password reset email body missing base URL %q: %s", resetBaseURL, bodyResult.Text)
	}

	t.Logf("password reset email to %q arrived in Mailpit — test passed", userEmail)
}
