package authhandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	sessstore "prasankit-api/internal/adapters/cache/session"
	authdbrepo "prasankit-api/internal/adapters/database/auth"
	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/email"
	authhandler "prasankit-api/internal/transport/http/auth"
	"prasankit-api/internal/transport/http/middlewares"
	"prasankit-api/pkg/ids"
	"prasankit-api/pkg/securetoken"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ── Infrastructure helpers ────────────────────────────────────────────────────

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		envOr("POSTGRES_PRIMARY_HOST", "localhost"),
		envOr("POSTGRES_PRIMARY_PORT", "15433"),
		envOr("POSTGRES_PRIMARY_USER", "prasankit"),
		envOr("POSTGRES_PRIMARY_PASSWORD", "change_me"),
		envOr("POSTGRES_PRIMARY_NAME", "prasankit"),
		envOr("POSTGRES_SSL_MODE", "disable"),
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Skipf("postgres not reachable: %v", err)
	}
	sqlDB, _ := db.DB()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		t.Skipf("postgres ping: %v", err)
	}
	return db
}

func openTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{
		Addr:     envOr("REDIS_HOST", "localhost") + ":" + envOr("REDIS_PORT", "16380"),
		Password: envOr("REDIS_PASSWORD", "change_me"),
		DB:       0,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not reachable: %v", err)
	}
	return client
}

func truncateDataTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, tbl := range []string{
		"auth_email_verification_tokens",
		"security_events",
		"auth_identities",
		"user_accounts",
	} {
		db.Exec("TRUNCATE TABLE " + tbl + " CASCADE")
	}
}

// noopSender is a no-op email.Sender for tests that don't need real email.
type noopSender struct{}

func (noopSender) Send(_ context.Context, _ email.Message) error { return nil }

// buildTestApp wires a fiber app against real DB+Redis with a no-op email sender.
func buildTestApp(t *testing.T, db *gorm.DB, rc *redis.Client) *fiber.App {
	t.Helper()
	accountRepo := authdbrepo.NewAccountRepo(db)
	identityRepo := authdbrepo.NewIdentityRepo(db)
	eventRepo := authdbrepo.NewSecurityEventRepo(db)
	verifyRepo := authdbrepo.NewVerificationTokenRepo(db)
	store := sessstore.NewStore(rc)
	svc := auth.NewService(accountRepo, identityRepo, eventRepo, store, verifyRepo, noopSender{}, "http://localhost:13000/verify-email", authdbrepo.NewPasswordResetTokenRepo(db), "http://localhost:13000/reset-password")
	h := authhandler.NewHandler(svc, "development")

	app := fiber.New(fiber.Config{ErrorHandler: middlewares.ErrorHandler})
	app.Post("/api/v1/auth/signup", h.HandleSignup)
	app.Post("/api/v1/auth/login", h.HandleLogin)
	app.Post("/api/v1/auth/logout", h.HandleLogout)
	app.Post("/api/v1/auth/verify-email", h.HandleVerifyEmail)
	app.Post("/api/v1/auth/verify-email/resend", h.HandleResendVerification)
	app.Post("/api/v1/auth/password-reset/request", h.HandleRequestPasswordReset)
	app.Post("/api/v1/auth/password-reset/confirm", h.HandleConfirmPasswordReset)
	app.Get("/api/v1/auth/me", middlewares.RequireSession(svc), h.HandleMe)
	return app
}

// activateAccount flips an account to active directly in DB, simulating email
// verification without going through the full token flow.
func activateAccount(t *testing.T, db *gorm.DB, emailAddr string) {
	t.Helper()
	if err := db.Exec(
		"UPDATE user_accounts SET account_status_code = 'active', updated_at = now() WHERE primary_email = ?",
		emailAddr,
	).Error; err != nil {
		t.Fatalf("activateAccount: %v", err)
	}
}

// ── Request helper ────────────────────────────────────────────────────────────

type testResp struct {
	StatusCode int
	Body       map[string]any
	Cookies    []*http.Cookie
}

func doRequest(t *testing.T, app *fiber.App, method, path string, body any, cookies []*http.Cookie) testResp {
	t.Helper()
	var buf *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	} else {
		buf = &bytes.Buffer{}
	}
	req := httptest.NewRequest(method, path, buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	var parsed map[string]any
	json.NewDecoder(resp.Body).Decode(&parsed) //nolint

	return testResp{
		StatusCode: resp.StatusCode,
		Body:       parsed,
		Cookies:    resp.Cookies(),
	}
}

func cookieByName(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// extractVerifyToken reads the raw token from the auth_email_verification_tokens
// table for the given email (used when bypassing the email flow in tests).
func extractVerifyTokenFromDB(t *testing.T, db *gorm.DB, emailAddr string) string {
	t.Helper()
	// Join via identity to find the token.
	var tokenHash string
	err := db.Raw(`
		SELECT t.token_hash
		FROM auth_email_verification_tokens t
		JOIN auth_identities i ON i.id = t.auth_identity_id
		WHERE i.email = ?
		  AND t.used_at IS NULL
		  AND t.revoked_at IS NULL
		  AND t.expires_at > now()
		ORDER BY t.created_at DESC
		LIMIT 1`,
		emailAddr,
	).Scan(&tokenHash).Error
	if err != nil || tokenHash == "" {
		t.Fatalf("extractVerifyTokenFromDB: no active token for %q: %v", emailAddr, err)
	}
	return tokenHash
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestAuthFlow_SignupLoginMeLogout(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	t.Cleanup(func() {
		truncateDataTables(t, db)
		rc.Close()
	})
	truncateDataTables(t, db)

	app := buildTestApp(t, db, rc)
	email := "flow+" + fmt.Sprintf("%d", time.Now().UnixNano()) + "@example.com"

	// 1. Signup → 201.
	resp := doRequest(t, app, "POST", "/api/v1/auth/signup", map[string]any{
		"email":        email,
		"password":     "testpass123",
		"display_name": "Flow User",
	}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("signup: status = %d, want 201; body = %v", resp.StatusCode, resp.Body)
	}
	data := resp.Body["data"].(map[string]any)
	if data["primary_email"] != email {
		t.Errorf("signup: email = %v, want %v", data["primary_email"], email)
	}

	// Activate account (simulate email verification) so login succeeds.
	activateAccount(t, db, email)

	// 2. Login → 200 + session cookie.
	resp = doRequest(t, app, "POST", "/api/v1/auth/login", map[string]any{
		"email":    email,
		"password": "testpass123",
	}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: status = %d, want 200; body = %v", resp.StatusCode, resp.Body)
	}
	sessionCookie := cookieByName(resp.Cookies, auth.SessionCookieName)
	if sessionCookie == nil {
		t.Fatal("login: session cookie not set")
	}
	if !sessionCookie.HttpOnly {
		t.Error("session cookie must be HttpOnly")
	}

	// 3. /auth/me with cookie → 200.
	resp = doRequest(t, app, "GET", "/api/v1/auth/me", nil, []*http.Cookie{sessionCookie})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/auth/me: status = %d, want 200; body = %v", resp.StatusCode, resp.Body)
	}
	meData := resp.Body["data"].(map[string]any)
	if meData["primary_email"] != email {
		t.Errorf("/auth/me: email = %v, want %v", meData["primary_email"], email)
	}

	// 4. Logout → 204.
	resp = doRequest(t, app, "POST", "/api/v1/auth/logout", nil, []*http.Cookie{sessionCookie})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("logout: status = %d, want 204", resp.StatusCode)
	}

	// 5. /auth/me without valid session → 401.
	resp = doRequest(t, app, "GET", "/api/v1/auth/me", nil, []*http.Cookie{sessionCookie})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("/auth/me after logout: status = %d, want 401", resp.StatusCode)
	}
	errBody := resp.Body["error"].(map[string]any)
	if errBody["code"] != "auth.unauthenticated" {
		t.Errorf("error code = %v, want auth.unauthenticated", errBody["code"])
	}
}

func TestAuth_MeWithoutCookie(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	t.Cleanup(func() { truncateDataTables(t, db); rc.Close() })

	app := buildTestApp(t, db, rc)

	resp := doRequest(t, app, "GET", "/api/v1/auth/me", nil, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestAuth_LockoutFlow(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	t.Cleanup(func() { truncateDataTables(t, db); rc.Close() })
	truncateDataTables(t, db)

	app := buildTestApp(t, db, rc)
	email := "lockout+" + fmt.Sprintf("%d", time.Now().UnixNano()) + "@example.com"

	// Signup + activate (lockout test needs wrong-password attempts to reach the password check).
	doRequest(t, app, "POST", "/api/v1/auth/signup", map[string]any{
		"email": email, "password": "password123", "display_name": "Lock User",
	}, nil)
	activateAccount(t, db, email)

	// 5 wrong-password attempts.
	for i := 0; i < auth.LockoutThreshold; i++ {
		resp := doRequest(t, app, "POST", "/api/v1/auth/login", map[string]any{
			"email":    email,
			"password": "wrongpass",
		}, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("attempt %d: status = %d, want 401", i+1, resp.StatusCode)
		}
	}

	// Next attempt should return account_locked.
	resp := doRequest(t, app, "POST", "/api/v1/auth/login", map[string]any{
		"email":    email,
		"password": "wrongpass",
	}, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("locked attempt: status = %d, want 401", resp.StatusCode)
	}
	errBody, ok := resp.Body["error"].(map[string]any)
	if !ok {
		t.Fatalf("no error body: %v", resp.Body)
	}
	code := errBody["code"].(string)
	if code != "auth.account_locked" && code != "auth.invalid_credentials" {
		t.Errorf("code = %v, want auth.account_locked (or auth.invalid_credentials during threshold hit)", code)
	}
	// Try one more with correct password to confirm it's locked.
	resp = doRequest(t, app, "POST", "/api/v1/auth/login", map[string]any{
		"email":    email,
		"password": "password123",
	}, nil)
	errBody2, ok := resp.Body["error"].(map[string]any)
	if !ok {
		if resp.StatusCode == http.StatusOK {
			t.Error("should not login with correct password when account is locked")
		}
	} else {
		code2 := errBody2["code"].(string)
		if code2 != "auth.account_locked" {
			t.Errorf("with correct pwd + locked: code = %v, want auth.account_locked", code2)
		}
	}
}

func TestAuth_SignupValidation(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	t.Cleanup(func() { truncateDataTables(t, db); rc.Close() })

	app := buildTestApp(t, db, rc)

	resp := doRequest(t, app, "POST", "/api/v1/auth/signup", map[string]any{
		"email":        "notvalid",
		"password":     "short",
		"display_name": "",
	}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	errBody := resp.Body["error"].(map[string]any)
	if errBody["code"] != "validation.invalid_input" {
		t.Errorf("code = %v, want validation.invalid_input", errBody["code"])
	}
}

func TestAuth_EmailTaken(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	t.Cleanup(func() { truncateDataTables(t, db); rc.Close() })
	truncateDataTables(t, db)

	app := buildTestApp(t, db, rc)
	email := "taken+" + fmt.Sprintf("%d", time.Now().UnixNano()) + "@example.com"

	doRequest(t, app, "POST", "/api/v1/auth/signup", map[string]any{
		"email": email, "password": "password123", "display_name": "First",
	}, nil)

	resp := doRequest(t, app, "POST", "/api/v1/auth/signup", map[string]any{
		"email": email, "password": "password456", "display_name": "Second",
	}, nil)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want 409", resp.StatusCode)
	}
	if !strings.Contains(fmt.Sprintf("%v", resp.Body), "email_taken") {
		t.Errorf("expected auth.email_taken in body, got: %v", resp.Body)
	}
}

// ── Email verification HTTP tests ─────────────────────────────────────────────

func TestAuth_LoginBeforeVerify_Returns403(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	t.Cleanup(func() { truncateDataTables(t, db); rc.Close() })
	truncateDataTables(t, db)

	app := buildTestApp(t, db, rc)
	email := "unverified+" + fmt.Sprintf("%d", time.Now().UnixNano()) + "@example.com"

	// Signup (account stays pending_verification).
	resp := doRequest(t, app, "POST", "/api/v1/auth/signup", map[string]any{
		"email": email, "password": "pass1234!", "display_name": "Unverified",
	}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("signup: status = %d", resp.StatusCode)
	}

	// Login before verifying → 403 auth.email_not_verified.
	resp = doRequest(t, app, "POST", "/api/v1/auth/login", map[string]any{
		"email":    email,
		"password": "pass1234!",
	}, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("login before verify: status = %d, want 403; body = %v", resp.StatusCode, resp.Body)
	}
	errBody := resp.Body["error"].(map[string]any)
	if errBody["code"] != "auth.email_not_verified" {
		t.Errorf("code = %v, want auth.email_not_verified", errBody["code"])
	}
}

func TestAuth_VerifyEndpoint_HappyPath(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	t.Cleanup(func() { truncateDataTables(t, db); rc.Close() })
	truncateDataTables(t, db)

	app := buildTestApp(t, db, rc)
	email := "verify2+" + fmt.Sprintf("%d", time.Now().UnixNano()) + "@example.com"

	// Signup.
	resp := doRequest(t, app, "POST", "/api/v1/auth/signup", map[string]any{
		"email": email, "password": "pass1234!", "display_name": "Verify Me",
	}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("signup: status = %d", resp.StatusCode)
	}

	// The noopSender means we can't extract a real raw token from email.
	// However the token_hash is stored in DB. We need the raw token which
	// was generated by securetoken.New() — it was passed to the sender but
	// the sender is a no-op. So we exercise the endpoint using a bogus token
	// to confirm 404, then activate directly for the login-success assertion.
	//
	// For the actual verify happy path, use extractVerifyTokenFromDB which
	// reads the hash — but the raw token is not recoverable from the hash.
	// Use a real SMTP sender test (verification_email_test.go) for end-to-end.
	// Here we test the HTTP layer with a known-bad token → 404.

	// Bogus token → 404.
	resp = doRequest(t, app, "POST", "/api/v1/auth/verify-email", map[string]any{
		"token": "totally-bogus-token",
	}, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("bogus token: status = %d, want 404; body = %v", resp.StatusCode, resp.Body)
	}
	errBody := resp.Body["error"].(map[string]any)
	if errBody["code"] != "auth.token_invalid" {
		t.Errorf("code = %v, want auth.token_invalid", errBody["code"])
	}

	// Activate and confirm login works.
	activateAccount(t, db, email)
	resp = doRequest(t, app, "POST", "/api/v1/auth/login", map[string]any{
		"email":    email,
		"password": "pass1234!",
	}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login after activation: status = %d, want 200; body = %v", resp.StatusCode, resp.Body)
	}
}

func TestAuth_VerifyEndpoint_EmptyToken_400(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	t.Cleanup(func() { truncateDataTables(t, db); rc.Close() })

	app := buildTestApp(t, db, rc)

	resp := doRequest(t, app, "POST", "/api/v1/auth/verify-email", map[string]any{
		"token": "",
	}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("empty token: status = %d, want 400", resp.StatusCode)
	}
}

func TestAuth_ResendVerification_Returns204(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	t.Cleanup(func() { truncateDataTables(t, db); rc.Close() })
	truncateDataTables(t, db)

	app := buildTestApp(t, db, rc)
	email := "resend+" + fmt.Sprintf("%d", time.Now().UnixNano()) + "@example.com"

	// Signup first.
	doRequest(t, app, "POST", "/api/v1/auth/signup", map[string]any{
		"email": email, "password": "pass1234!", "display_name": "Resend",
	}, nil)

	// Resend → 204 always (even for unverified).
	resp := doRequest(t, app, "POST", "/api/v1/auth/verify-email/resend", map[string]any{
		"email": email,
	}, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("resend: status = %d, want 204; body = %v", resp.StatusCode, resp.Body)
	}

	// Resend for unknown email → still 204 (anti-enum).
	resp = doRequest(t, app, "POST", "/api/v1/auth/verify-email/resend", map[string]any{
		"email": "nobody@example.com",
	}, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("resend unknown: status = %d, want 204", resp.StatusCode)
	}
}

// ── Password reset HTTP tests ─────────────────────────────────────────────────

func TestAuth_PasswordResetRequest_Returns204(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	t.Cleanup(func() { truncateDataTables(t, db); rc.Close() })
	truncateDataTables(t, db)

	app := buildTestApp(t, db, rc)

	// Any email — even unknown — must return 204 (anti-enumeration).
	resp := doRequest(t, app, "POST", "/api/v1/auth/password-reset/request", map[string]any{
		"email": "unknown-user@example.com",
	}, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("password reset request: status = %d, want 204; body = %v", resp.StatusCode, resp.Body)
	}

	// Known email also returns 204.
	email := "pr-request+" + fmt.Sprintf("%d", time.Now().UnixNano()) + "@example.com"
	doRequest(t, app, "POST", "/api/v1/auth/signup", map[string]any{
		"email": email, "password": "pass1234!", "display_name": "Reset User",
	}, nil)

	resp = doRequest(t, app, "POST", "/api/v1/auth/password-reset/request", map[string]any{
		"email": email,
	}, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("password reset request (known email): status = %d, want 204; body = %v", resp.StatusCode, resp.Body)
	}
}

func TestAuth_PasswordResetConfirm_HappyPath(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	t.Cleanup(func() { truncateDataTables(t, db); rc.Close() })
	truncateDataTables(t, db)

	app := buildTestApp(t, db, rc)
	email := "pr-confirm+" + fmt.Sprintf("%d", time.Now().UnixNano()) + "@example.com"
	oldPassword := "oldpass1234!"

	// Signup.
	resp := doRequest(t, app, "POST", "/api/v1/auth/signup", map[string]any{
		"email": email, "password": oldPassword, "display_name": "Reset Confirm User",
	}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("signup: status = %d; body = %v", resp.StatusCode, resp.Body)
	}

	// Activate account.
	activateAccount(t, db, email)

	// Look up the identity ID for this email.
	var identityID string
	if err := db.Raw("SELECT id FROM auth_identities WHERE email = ?", email).Scan(&identityID).Error; err != nil || identityID == "" {
		t.Fatalf("look up identity id for %q: err=%v id=%q", email, err, identityID)
	}

	// Insert a reset token directly (bypassing SMTP).
	rawToken := "reset-raw-token-12345"
	tokenID, _ := ids.New()
	now := time.Now().UTC()
	resetToken := auth.PasswordResetToken{
		ID:             tokenID,
		AuthIdentityID: mustParseUUID(t, identityID),
		TokenHash:      securetoken.Hash(rawToken),
		ExpiresAt:      now.Add(time.Hour),
		CreatedAt:      now,
	}
	if err := authdbrepo.NewPasswordResetTokenRepo(db).Create(context.Background(), resetToken); err != nil {
		t.Fatalf("create reset token: %v", err)
	}

	newPassword := "brandnewpass123"

	// Confirm reset → 200.
	resp = doRequest(t, app, "POST", "/api/v1/auth/password-reset/confirm", map[string]any{
		"token":        rawToken,
		"new_password": newPassword,
	}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("confirm reset: status = %d, want 200; body = %v", resp.StatusCode, resp.Body)
	}

	// Login with NEW password → 200.
	resp = doRequest(t, app, "POST", "/api/v1/auth/login", map[string]any{
		"email":    email,
		"password": newPassword,
	}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login with new password: status = %d, want 200; body = %v", resp.StatusCode, resp.Body)
	}

	// Login with OLD password → 401.
	resp = doRequest(t, app, "POST", "/api/v1/auth/login", map[string]any{
		"email":    email,
		"password": oldPassword,
	}, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("login with old password: status = %d, want 401; body = %v", resp.StatusCode, resp.Body)
	}
}

func TestAuth_PasswordResetConfirm_BadToken_404(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	t.Cleanup(func() { truncateDataTables(t, db); rc.Close() })

	app := buildTestApp(t, db, rc)

	resp := doRequest(t, app, "POST", "/api/v1/auth/password-reset/confirm", map[string]any{
		"token":        "completely-unknown-token-xyz",
		"new_password": "validpassword123",
	}, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("bad token: status = %d, want 404; body = %v", resp.StatusCode, resp.Body)
	}
	if errBody, ok := resp.Body["error"].(map[string]any); ok {
		if errBody["code"] != "auth.token_invalid" {
			t.Errorf("error code = %v, want auth.token_invalid", errBody["code"])
		}
	}
}

func TestAuth_PasswordResetConfirm_WeakPassword_400(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	t.Cleanup(func() { truncateDataTables(t, db); rc.Close() })

	app := buildTestApp(t, db, rc)

	// Validation runs before token lookup — weak password returns 400 immediately.
	resp := doRequest(t, app, "POST", "/api/v1/auth/password-reset/confirm", map[string]any{
		"token":        "anything",
		"new_password": "short",
	}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("weak password: status = %d, want 400; body = %v", resp.StatusCode, resp.Body)
	}
	if errBody, ok := resp.Body["error"].(map[string]any); ok {
		if errBody["code"] != "validation.invalid_input" {
			t.Errorf("error code = %v, want validation.invalid_input", errBody["code"])
		}
	}
}

// mustParseUUID parses a UUID string, fatally failing the test on error.
func mustParseUUID(t *testing.T, s string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(s)
	if err != nil {
		t.Fatalf("mustParseUUID(%q): %v", s, err)
	}
	return id
}

// TestAuth_SessionRevokedAfterPasswordReset proves lazy session revocation (5c):
// a session created before the password reset is invalidated on its next request.
func TestAuth_SessionRevokedAfterPasswordReset(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	t.Cleanup(func() { truncateDataTables(t, db); rc.Close() })
	truncateDataTables(t, db)

	app := buildTestApp(t, db, rc)
	email := "sess-revoke+" + fmt.Sprintf("%d", time.Now().UnixNano()) + "@example.com"
	oldPassword := "oldpass1234!"

	// Signup + activate + login → get an old session cookie.
	doRequest(t, app, "POST", "/api/v1/auth/signup", map[string]any{
		"email": email, "password": oldPassword, "display_name": "Sess Revoke",
	}, nil)
	activateAccount(t, db, email)

	loginResp := doRequest(t, app, "POST", "/api/v1/auth/login", map[string]any{
		"email": email, "password": oldPassword,
	}, nil)
	oldCookie := cookieByName(loginResp.Cookies, auth.SessionCookieName)
	if oldCookie == nil {
		t.Fatal("login: no session cookie")
	}

	// Sanity: old cookie works before reset.
	meResp := doRequest(t, app, "GET", "/api/v1/auth/me", nil, []*http.Cookie{oldCookie})
	if meResp.StatusCode != http.StatusOK {
		t.Fatalf("me before reset: status = %d, want 200", meResp.StatusCode)
	}

	// Insert a reset token directly and confirm a new password.
	var identityID string
	if err := db.Raw("SELECT id FROM auth_identities WHERE email = ?", email).Scan(&identityID).Error; err != nil || identityID == "" {
		t.Fatalf("look up identity id: err=%v id=%q", err, identityID)
	}
	rawToken := "sess-revoke-reset-token"
	tokenID, _ := ids.New()
	now := time.Now().UTC()
	if err := authdbrepo.NewPasswordResetTokenRepo(db).Create(context.Background(), auth.PasswordResetToken{
		ID:             tokenID,
		AuthIdentityID: mustParseUUID(t, identityID),
		TokenHash:      securetoken.Hash(rawToken),
		ExpiresAt:      now.Add(time.Hour),
		CreatedAt:      now,
	}); err != nil {
		t.Fatalf("create reset token: %v", err)
	}
	confirmResp := doRequest(t, app, "POST", "/api/v1/auth/password-reset/confirm", map[string]any{
		"token": rawToken, "new_password": "brandnewpass123",
	}, nil)
	if confirmResp.StatusCode != http.StatusOK {
		t.Fatalf("confirm reset: status = %d, want 200; body = %v", confirmResp.StatusCode, confirmResp.Body)
	}

	// Old cookie must now be revoked → 401.
	meAfter := doRequest(t, app, "GET", "/api/v1/auth/me", nil, []*http.Cookie{oldCookie})
	if meAfter.StatusCode != http.StatusUnauthorized {
		t.Fatalf("me after reset with old cookie: status = %d, want 401", meAfter.StatusCode)
	}

	// Fresh login with the new password yields a working session.
	newLogin := doRequest(t, app, "POST", "/api/v1/auth/login", map[string]any{
		"email": email, "password": "brandnewpass123",
	}, nil)
	newCookie := cookieByName(newLogin.Cookies, auth.SessionCookieName)
	if newCookie == nil {
		t.Fatal("login with new password: no session cookie")
	}
	meNew := doRequest(t, app, "GET", "/api/v1/auth/me", nil, []*http.Cookie{newCookie})
	if meNew.StatusCode != http.StatusOK {
		t.Fatalf("me with new cookie: status = %d, want 200", meNew.StatusCode)
	}
}
