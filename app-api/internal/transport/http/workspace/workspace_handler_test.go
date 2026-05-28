package workspacehandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	sessstore "prasankit-api/internal/adapters/cache/session"
	auditdbrepo "prasankit-api/internal/adapters/database/audit"
	authdbrepo "prasankit-api/internal/adapters/database/auth"
	workspacedbrepo "prasankit-api/internal/adapters/database/workspace"
	smtpadapter "prasankit-api/internal/adapters/email/smtp"
	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/email"
	"prasankit-api/internal/modules/workspace"
	authhandler "prasankit-api/internal/transport/http/auth"
	"prasankit-api/internal/transport/http/middlewares"
	workspacehandler "prasankit-api/internal/transport/http/workspace"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// noopEmailSender is a no-op email.Sender for tests that don't need real email.
type noopEmailSender struct{}

func (noopEmailSender) Send(_ context.Context, _ email.Message) error { return nil }

// ── Infrastructure ────────────────────────────────────────────────────────────

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
		"audit_logs",
		"workspace_invitations",
		"workspace_memberships",
		"workspaces",
		"auth_email_verification_tokens",
		"security_events",
		"auth_identities",
		"user_accounts",
	} {
		db.Exec("TRUNCATE TABLE " + tbl + " CASCADE")
	}
}

// buildTestApp wires a full fiber app for workspace handler tests.
func buildTestApp(t *testing.T, db *gorm.DB, rc *redis.Client) *fiber.App {
	t.Helper()
	// Auth wiring.
	accountRepo := authdbrepo.NewAccountRepo(db)
	identityRepo := authdbrepo.NewIdentityRepo(db)
	eventRepo := authdbrepo.NewSecurityEventRepo(db)
	verifyRepo := authdbrepo.NewVerificationTokenRepo(db)
	store := sessstore.NewStore(rc)
	authSvc := auth.NewService(accountRepo, identityRepo, eventRepo, store, verifyRepo, noopEmailSender{}, "http://localhost:13000/verify-email")
	authH := authhandler.NewHandler(authSvc, "development")

	// Workspace wiring.
	auditRepo := auditdbrepo.NewAuditRepo(db)
	wsRepo := workspacedbrepo.NewWorkspaceRepo(db, auditRepo)
	memberRepo := workspacedbrepo.NewMembershipRepo(db)
	inviteRepo := workspacedbrepo.NewInvitationRepo(db, auditRepo)
	emailSender := smtpadapter.New(smtpadapter.Config{
		Host:        envOr("MAIL_SMTP_HOST", "localhost"),
		Port:        envOr("MAIL_SMTP_EXTERNAL_PORT", "11025"),
		FromAddress: "test@prasankit.local",
	})
	workspaceSvc := workspace.NewService(wsRepo, memberRepo, inviteRepo, emailSender, "http://localhost:13000/invitations/accept")
	workspaceH := workspacehandler.NewHandler(workspaceSvc)

	requireSession := middlewares.RequireSession(authSvc)
	requireTenant := middlewares.RequireTenantContext(wsRepo, memberRepo)

	app := fiber.New(fiber.Config{ErrorHandler: middlewares.ErrorHandler})
	// Auth routes.
	app.Post("/api/v1/auth/signup", authH.HandleSignup)
	app.Post("/api/v1/auth/login", authH.HandleLogin)
	// Workspace routes.
	app.Post("/api/v1/workspaces", requireSession, workspaceH.HandleCreate)
	app.Get("/api/v1/workspaces", requireSession, workspaceH.HandleList)
	app.Get("/api/v1/workspaces/current", requireSession, requireTenant, workspaceH.HandleGetCurrent)
	return app
}

// ── Request helpers ───────────────────────────────────────────────────────────

type testResp struct {
	StatusCode int
	Body       map[string]any
	Cookies    []*http.Cookie
}

func doRequest(t *testing.T, app *fiber.App, method, path string, body any, cookies []*http.Cookie, headers map[string]string) testResp {
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
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	var parsed map[string]any
	json.NewDecoder(resp.Body).Decode(&parsed) //nolint
	return testResp{StatusCode: resp.StatusCode, Body: parsed, Cookies: resp.Cookies()}
}

func cookieByName(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// signupAndLogin signs up, activates the account (bypassing email verification),
// then logs in and returns the session cookie.
func signupAndLogin(t *testing.T, app *fiber.App, emailAddr string) *http.Cookie {
	t.Helper()
	doRequest(t, app, "POST", "/api/v1/auth/signup", map[string]any{
		"email":        emailAddr,
		"password":     "testpass123",
		"display_name": "Test User",
	}, nil, nil)
	// Activate account directly so login succeeds (email verification is required).
	db := openTestDB(t)
	if err := db.Exec(
		"UPDATE user_accounts SET account_status_code = 'active', updated_at = now() WHERE primary_email = ?",
		emailAddr,
	).Error; err != nil {
		t.Fatalf("signupAndLogin: activate account: %v", err)
	}
	resp := doRequest(t, app, "POST", "/api/v1/auth/login", map[string]any{
		"email":    emailAddr,
		"password": "testpass123",
	}, nil, nil)
	cookie := cookieByName(resp.Cookies, auth.SessionCookieName)
	if cookie == nil {
		t.Fatalf("login: session cookie not set for %s", emailAddr)
	}
	return cookie
}

// uniqueEmail generates a unique email per test run.
func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s+%d@example.com", prefix, time.Now().UnixNano())
}

// uniqueSlug generates a unique workspace slug.
func uniqueSlug(prefix string) string {
	nano := time.Now().UnixNano() % 1_000_000
	return fmt.Sprintf("%s-%d", prefix, nano)
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestWorkspaceFlow_CreateAndList(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	t.Cleanup(func() { truncateDataTables(t, db); rc.Close() })
	truncateDataTables(t, db)

	app := buildTestApp(t, db, rc)
	email := uniqueEmail("ws-flow")
	cookie := signupAndLogin(t, app, email)

	slug := uniqueSlug("acme")

	// POST /workspaces → 201.
	resp := doRequest(t, app, "POST", "/api/v1/workspaces", map[string]any{
		"name":          "Acme Corp",
		"slug":          slug,
		"contact_email": "admin@acme.com",
	}, []*http.Cookie{cookie}, nil)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: status = %d, want 201; body = %v", resp.StatusCode, resp.Body)
	}
	data := resp.Body["data"].(map[string]any)
	if data["slug"] != slug {
		t.Errorf("create: slug = %v, want %v", data["slug"], slug)
	}
	if data["workspace_status_code"] != "active" {
		t.Errorf("create: status = %v, want active", data["workspace_status_code"])
	}

	// GET /workspaces → 200 with exactly 1 item.
	resp = doRequest(t, app, "GET", "/api/v1/workspaces", nil, []*http.Cookie{cookie}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: status = %d, want 200; body = %v", resp.StatusCode, resp.Body)
	}
	listData := resp.Body["data"].(map[string]any)
	items := listData["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("list: got %d items, want 1", len(items))
	}
	item := items[0].(map[string]any)
	if item["org_role_code"] != "owner" {
		t.Errorf("list: org_role_code = %v, want owner", item["org_role_code"])
	}
}

func TestWorkspaceGetCurrent_Success(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	t.Cleanup(func() { truncateDataTables(t, db); rc.Close() })
	truncateDataTables(t, db)

	app := buildTestApp(t, db, rc)
	email := uniqueEmail("ws-current")
	cookie := signupAndLogin(t, app, email)
	slug := uniqueSlug("current")

	// Create workspace.
	doRequest(t, app, "POST", "/api/v1/workspaces", map[string]any{
		"name":          "Current Test",
		"slug":          slug,
		"contact_email": "test@current.com",
	}, []*http.Cookie{cookie}, nil)

	// GET /workspaces/current with correct X-Workspace-Slug → 200, role=owner.
	resp := doRequest(t, app, "GET", "/api/v1/workspaces/current", nil, []*http.Cookie{cookie},
		map[string]string{"X-Workspace-Slug": slug})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("current: status = %d, want 200; body = %v", resp.StatusCode, resp.Body)
	}
	d := resp.Body["data"].(map[string]any)
	if d["slug"] != slug {
		t.Errorf("current: slug = %v, want %v", d["slug"], slug)
	}
	if d["org_role_code"] != "owner" {
		t.Errorf("current: org_role_code = %v, want owner", d["org_role_code"])
	}
}

func TestWorkspaceGetCurrent_NoHeader_400(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	t.Cleanup(func() { truncateDataTables(t, db); rc.Close() })
	truncateDataTables(t, db)

	app := buildTestApp(t, db, rc)
	email := uniqueEmail("ws-noheader")
	cookie := signupAndLogin(t, app, email)

	// No X-Workspace-Slug header → 400.
	resp := doRequest(t, app, "GET", "/api/v1/workspaces/current", nil, []*http.Cookie{cookie}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("no header: status = %d, want 400", resp.StatusCode)
	}
	errBody := resp.Body["error"].(map[string]any)
	if errBody["code"] != "tenant.workspace_required" {
		t.Errorf("code = %v, want tenant.workspace_required", errBody["code"])
	}
}

func TestWorkspaceGetCurrent_BogusSlug_404(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	t.Cleanup(func() { truncateDataTables(t, db); rc.Close() })
	truncateDataTables(t, db)

	app := buildTestApp(t, db, rc)
	email := uniqueEmail("ws-bogus")
	cookie := signupAndLogin(t, app, email)

	// Non-existent slug → 404.
	resp := doRequest(t, app, "GET", "/api/v1/workspaces/current", nil, []*http.Cookie{cookie},
		map[string]string{"X-Workspace-Slug": "does-not-exist-ever"})
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("bogus slug: status = %d, want 404; body = %v", resp.StatusCode, resp.Body)
	}
	errBody := resp.Body["error"].(map[string]any)
	if errBody["code"] != "tenant.workspace_not_found" {
		t.Errorf("code = %v, want tenant.workspace_not_found", errBody["code"])
	}
}

func TestWorkspaceGetCurrent_NotMember_403(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	t.Cleanup(func() { truncateDataTables(t, db); rc.Close() })
	truncateDataTables(t, db)

	app := buildTestApp(t, db, rc)

	// User A creates workspace.
	emailA := uniqueEmail("ws-owner")
	cookieA := signupAndLogin(t, app, emailA)
	slug := uniqueSlug("owned-by-a")
	doRequest(t, app, "POST", "/api/v1/workspaces", map[string]any{
		"name":          "Owned By A",
		"slug":          slug,
		"contact_email": "a@example.com",
	}, []*http.Cookie{cookieA}, nil)

	// User B tries to access the workspace → 403.
	emailB := uniqueEmail("ws-nonmember")
	cookieB := signupAndLogin(t, app, emailB)
	resp := doRequest(t, app, "GET", "/api/v1/workspaces/current", nil, []*http.Cookie{cookieB},
		map[string]string{"X-Workspace-Slug": slug})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("non-member: status = %d, want 403; body = %v", resp.StatusCode, resp.Body)
	}
	errBody := resp.Body["error"].(map[string]any)
	if errBody["code"] != "tenant.forbidden" {
		t.Errorf("code = %v, want tenant.forbidden", errBody["code"])
	}
}

func TestWorkspace_Create_SlugReserved_400(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	t.Cleanup(func() { truncateDataTables(t, db); rc.Close() })
	truncateDataTables(t, db)

	app := buildTestApp(t, db, rc)
	email := uniqueEmail("ws-reserved")
	cookie := signupAndLogin(t, app, email)

	resp := doRequest(t, app, "POST", "/api/v1/workspaces", map[string]any{
		"name":          "Admin",
		"slug":          "admin",
		"contact_email": "admin@example.com",
	}, []*http.Cookie{cookie}, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("reserved slug: status = %d, want 400; body = %v", resp.StatusCode, resp.Body)
	}
	errBody := resp.Body["error"].(map[string]any)
	if errBody["code"] != "workspace.slug_reserved" {
		t.Errorf("code = %v, want workspace.slug_reserved", errBody["code"])
	}
}

func TestWorkspace_Create_SlugTaken_409(t *testing.T) {
	db := openTestDB(t)
	rc := openTestRedis(t)
	t.Cleanup(func() { truncateDataTables(t, db); rc.Close() })
	truncateDataTables(t, db)

	app := buildTestApp(t, db, rc)
	email := uniqueEmail("ws-slug-taken")
	cookie := signupAndLogin(t, app, email)
	slug := uniqueSlug("taken-slug")

	doRequest(t, app, "POST", "/api/v1/workspaces", map[string]any{
		"name":          "First",
		"slug":          slug,
		"contact_email": "first@example.com",
	}, []*http.Cookie{cookie}, nil)

	// Second attempt with same slug → 409.
	resp := doRequest(t, app, "POST", "/api/v1/workspaces", map[string]any{
		"name":          "Second",
		"slug":          slug,
		"contact_email": "second@example.com",
	}, []*http.Cookie{cookie}, nil)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("slug taken: status = %d, want 409; body = %v", resp.StatusCode, resp.Body)
	}
	errBody := resp.Body["error"].(map[string]any)
	if errBody["code"] != "workspace.slug_taken" {
		t.Errorf("code = %v, want workspace.slug_taken", errBody["code"])
	}
}
