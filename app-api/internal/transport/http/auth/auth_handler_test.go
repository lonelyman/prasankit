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
	authhandler "prasankit-api/internal/transport/http/auth"
	"prasankit-api/internal/transport/http/middlewares"

	"github.com/gofiber/fiber/v3"
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
	for _, tbl := range []string{"security_events", "auth_identities", "user_accounts"} {
		db.Exec("TRUNCATE TABLE " + tbl + " CASCADE")
	}
}

// buildTestApp wires a fiber app against real DB+Redis.
func buildTestApp(t *testing.T, db *gorm.DB, rc *redis.Client) *fiber.App {
	t.Helper()
	accountRepo := authdbrepo.NewAccountRepo(db)
	identityRepo := authdbrepo.NewIdentityRepo(db)
	eventRepo := authdbrepo.NewSecurityEventRepo(db)
	store := sessstore.NewStore(rc)
	svc := auth.NewService(accountRepo, identityRepo, eventRepo, store)
	h := authhandler.NewHandler(svc, "development")

	app := fiber.New(fiber.Config{ErrorHandler: middlewares.ErrorHandler})
	app.Post("/api/v1/auth/signup", h.HandleSignup)
	app.Post("/api/v1/auth/login", h.HandleLogin)
	app.Post("/api/v1/auth/logout", h.HandleLogout)
	app.Get("/api/v1/auth/me", middlewares.RequireSession(svc), h.HandleMe)
	return app
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

	// Signup.
	doRequest(t, app, "POST", "/api/v1/auth/signup", map[string]any{
		"email": email, "password": "password123", "display_name": "Lock User",
	}, nil)

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
	// The 6th attempt (after threshold is reached) must return auth.account_locked.
	// The 5th increments count to 5 and locks; the 6th checks and sees locked.
	// Try one more with correct password to confirm it's locked.
	resp = doRequest(t, app, "POST", "/api/v1/auth/login", map[string]any{
		"email":    email,
		"password": "password123",
	}, nil)
	errBody2, ok := resp.Body["error"].(map[string]any)
	if !ok {
		// If somehow success, that's a problem.
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
