package workspacehandler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	sessstore "prasankit-api/internal/adapters/cache/session"
	auditdbrepo "prasankit-api/internal/adapters/database/audit"
	authdbrepo "prasankit-api/internal/adapters/database/auth"
	workspacedbrepo "prasankit-api/internal/adapters/database/workspace"
	smtpadapter "prasankit-api/internal/adapters/email/smtp"
	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/workspace"
	authhandler "prasankit-api/internal/transport/http/auth"
	"prasankit-api/internal/transport/http/middlewares"
	workspacehandler "prasankit-api/internal/transport/http/workspace"

	"github.com/gofiber/fiber/v3"
)

// buildInviteTestApp wires a full fiber app including invitation routes.
// It connects to real Postgres + Redis. Mailpit is used as the SMTP sink.
func buildInviteTestApp(t *testing.T) (*fiber.App, func()) {
	t.Helper()
	db := openTestDB(t)
	rc := openTestRedis(t)

	truncateDataTables(t, db)

	uiPort := envOr("MAILPIT_UI_EXTERNAL_PORT", "18025")
	cleanup := func() {
		truncateDataTables(t, db)
		clearMailpitInHandler(uiPort)
		rc.Close()
	}

	accountRepo := authdbrepo.NewAccountRepo(db)
	identityRepo := authdbrepo.NewIdentityRepo(db)
	eventRepo := authdbrepo.NewSecurityEventRepo(db)
	store := sessstore.NewStore(rc)
	authSvc := auth.NewService(accountRepo, identityRepo, eventRepo, store)
	authH := authhandler.NewHandler(authSvc, "development")

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
	requireInvite := middlewares.RequireWorkspacePermission(workspace.PermissionInviteMember)

	app := fiber.New(fiber.Config{ErrorHandler: middlewares.ErrorHandler})
	app.Post("/api/v1/auth/signup", authH.HandleSignup)
	app.Post("/api/v1/auth/login", authH.HandleLogin)
	app.Post("/api/v1/workspaces", requireSession, workspaceH.HandleCreate)
	app.Get("/api/v1/workspaces", requireSession, workspaceH.HandleList)
	app.Post("/api/v1/workspaces/invitations", requireSession, requireTenant, requireInvite, workspaceH.HandleInvite)
	app.Post("/api/v1/invitations/accept", requireSession, workspaceH.HandleAcceptInvite)

	return app, cleanup
}

// mailpitGetMessages fetches the Mailpit message list. Returns nil on failure (non-fatal for flow tests).
func mailpitGetMessages(uiPort string) []struct {
	ID      string `json:"ID"`
	Subject string `json:"Subject"`
	To      []struct {
		Address string `json:"Address"`
	} `json:"To"`
} {
	url := fmt.Sprintf("http://localhost:%s/api/v1/messages", uiPort)
	resp, err := http.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var result struct {
		Messages []struct {
			ID      string `json:"ID"`
			Subject string `json:"Subject"`
			To      []struct {
				Address string `json:"Address"`
			} `json:"To"`
		} `json:"messages"`
	}
	json.NewDecoder(resp.Body).Decode(&result) //nolint
	return result.Messages
}

// mailpitGetText fetches the plain-text body of a specific Mailpit message.
func mailpitGetText(uiPort, msgID string) string {
	url := fmt.Sprintf("http://localhost:%s/api/v1/message/%s", uiPort, msgID)
	resp, err := http.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var result struct {
		Text string `json:"Text"`
	}
	json.NewDecoder(resp.Body).Decode(&result) //nolint
	return result.Text
}

// extractTokenFromMailpit finds the invite email for the given address and extracts the raw token.
func extractTokenFromMailpit(t *testing.T, uiPort, toEmail string) string {
	t.Helper()
	msgs := mailpitGetMessages(uiPort)
	for _, m := range msgs {
		for _, to := range m.To {
			if strings.EqualFold(to.Address, toEmail) {
				body := mailpitGetText(uiPort, m.ID)
				idx := strings.Index(body, "?token=")
				if idx < 0 {
					t.Fatalf("invite email to %q has no ?token= in body: %s", toEmail, body)
				}
				rest := body[idx+len("?token="):]
				end := strings.IndexAny(rest, "\r\n \t")
				if end >= 0 {
					return rest[:end]
				}
				return strings.TrimSpace(rest)
			}
		}
	}
	t.Fatalf("no Mailpit message found addressed to %q (got %d messages)", toEmail, len(msgs))
	return ""
}

// clearMailpitInHandler deletes all messages from Mailpit.
func clearMailpitInHandler(uiPort string) {
	url := fmt.Sprintf("http://localhost:%s/api/v1/messages", uiPort)
	req, _ := http.NewRequest(http.MethodDelete, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestInviteFlow_FullHappyPath: owner invites → invitee accepts → invitee sees workspace.
func TestInviteFlow_FullHappyPath(t *testing.T) {
	uiPort := envOr("MAILPIT_UI_EXTERNAL_PORT", "18025")
	clearMailpitInHandler(uiPort)

	app, cleanup := buildInviteTestApp(t)
	t.Cleanup(func() {
		cleanup()
		clearMailpitInHandler(uiPort)
	})

	ownerEmail := uniqueEmail("invite-owner")
	ownerCookie := signupAndLogin(t, app, ownerEmail)
	slug := uniqueSlug("invite-ws")

	// Create workspace.
	resp := doRequest(t, app, "POST", "/api/v1/workspaces", map[string]any{
		"name":          "Invite Test WS",
		"slug":          slug,
		"contact_email": ownerEmail,
	}, []*http.Cookie{ownerCookie}, nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create workspace: status=%d body=%v", resp.StatusCode, resp.Body)
	}

	inviteeEmail := uniqueEmail("invitee")

	// POST /workspaces/invitations → 201.
	resp = doRequest(t, app, "POST", "/api/v1/workspaces/invitations", map[string]any{
		"email":         inviteeEmail,
		"org_role_code": "user",
	}, []*http.Cookie{ownerCookie}, map[string]string{"X-Workspace-Slug": slug})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("invite: status=%d body=%v", resp.StatusCode, resp.Body)
	}
	invData := resp.Body["data"].(map[string]any)
	if invData["email"] != inviteeEmail {
		t.Errorf("invite email = %v, want %v", invData["email"], inviteeEmail)
	}

	// Allow brief Mailpit indexing.
	time.Sleep(300 * time.Millisecond)

	// Extract token from Mailpit.
	rawToken := extractTokenFromMailpit(t, uiPort, inviteeEmail)
	if rawToken == "" {
		t.Fatal("empty token from Mailpit")
	}

	// Invitee signs up + logs in.
	inviteeCookie := signupAndLogin(t, app, inviteeEmail)

	// POST /invitations/accept with the token → 200.
	resp = doRequest(t, app, "POST", "/api/v1/invitations/accept", map[string]any{
		"token": rawToken,
	}, []*http.Cookie{inviteeCookie}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("accept: status=%d body=%v", resp.StatusCode, resp.Body)
	}
	acceptData := resp.Body["data"].(map[string]any)
	if acceptData["org_role_code"] != "user" {
		t.Errorf("accepted role = %v, want user", acceptData["org_role_code"])
	}

	// Invitee GET /workspaces → sees the workspace with role=user.
	resp = doRequest(t, app, "GET", "/api/v1/workspaces", nil, []*http.Cookie{inviteeCookie}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: status=%d body=%v", resp.StatusCode, resp.Body)
	}
	listData := resp.Body["data"].(map[string]any)
	items := listData["items"].([]any)
	found := false
	for _, item := range items {
		m := item.(map[string]any)
		if m["slug"] == slug && m["org_role_code"] == "user" {
			found = true
		}
	}
	if !found {
		t.Errorf("invitee workspace list doesn't show invited workspace with role=user: %v", items)
	}
}

// TestInviteFlow_AcceptWrongEmail: accept with mismatched account email → 403.
func TestInviteFlow_AcceptWrongEmail(t *testing.T) {
	uiPort := envOr("MAILPIT_UI_EXTERNAL_PORT", "18025")
	clearMailpitInHandler(uiPort)

	app, cleanup := buildInviteTestApp(t)
	t.Cleanup(func() {
		cleanup()
		clearMailpitInHandler(uiPort)
	})

	ownerEmail := uniqueEmail("wrong-owner")
	ownerCookie := signupAndLogin(t, app, ownerEmail)
	slug := uniqueSlug("wrong-ws")

	doRequest(t, app, "POST", "/api/v1/workspaces", map[string]any{
		"name":          "Wrong Email WS",
		"slug":          slug,
		"contact_email": ownerEmail,
	}, []*http.Cookie{ownerCookie}, nil)

	inviteeEmail := uniqueEmail("right-invitee")
	doRequest(t, app, "POST", "/api/v1/workspaces/invitations", map[string]any{
		"email":         inviteeEmail,
		"org_role_code": "user",
	}, []*http.Cookie{ownerCookie}, map[string]string{"X-Workspace-Slug": slug})

	time.Sleep(300 * time.Millisecond)
	rawToken := extractTokenFromMailpit(t, uiPort, inviteeEmail)

	// WRONG account signs up (different email) and tries to accept.
	wrongEmail := uniqueEmail("wrong-accepter")
	wrongCookie := signupAndLogin(t, app, wrongEmail)

	resp := doRequest(t, app, "POST", "/api/v1/invitations/accept", map[string]any{
		"token": rawToken,
	}, []*http.Cookie{wrongCookie}, nil)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("wrong email accept: status=%d, want 403; body=%v", resp.StatusCode, resp.Body)
	}
	errBody := resp.Body["error"].(map[string]any)
	if errBody["code"] != "invitation.email_mismatch" {
		t.Errorf("code=%v, want invitation.email_mismatch", errBody["code"])
	}
}

// TestInviteFlow_NonAdminCannotInvite: a user-role member tries to invite → 403.
func TestInviteFlow_NonAdminCannotInvite(t *testing.T) {
	uiPort := envOr("MAILPIT_UI_EXTERNAL_PORT", "18025")
	clearMailpitInHandler(uiPort)

	app, cleanup := buildInviteTestApp(t)
	t.Cleanup(func() {
		cleanup()
		clearMailpitInHandler(uiPort)
	})

	ownerEmail := uniqueEmail("nonadmin-owner")
	ownerCookie := signupAndLogin(t, app, ownerEmail)
	slug := uniqueSlug("nonadmin-ws")

	doRequest(t, app, "POST", "/api/v1/workspaces", map[string]any{
		"name":          "NonAdmin WS",
		"slug":          slug,
		"contact_email": ownerEmail,
	}, []*http.Cookie{ownerCookie}, nil)

	// Invite a user-role member.
	userEmail := uniqueEmail("user-member")
	doRequest(t, app, "POST", "/api/v1/workspaces/invitations", map[string]any{
		"email":         userEmail,
		"org_role_code": "user",
	}, []*http.Cookie{ownerCookie}, map[string]string{"X-Workspace-Slug": slug})

	time.Sleep(300 * time.Millisecond)
	rawToken := extractTokenFromMailpit(t, uiPort, userEmail)

	userCookie := signupAndLogin(t, app, userEmail)
	doRequest(t, app, "POST", "/api/v1/invitations/accept", map[string]any{
		"token": rawToken,
	}, []*http.Cookie{userCookie}, nil)

	// The user-role member now tries to invite someone else → 403.
	thirdEmail := uniqueEmail("third-person")
	resp := doRequest(t, app, "POST", "/api/v1/workspaces/invitations", map[string]any{
		"email":         thirdEmail,
		"org_role_code": "user",
	}, []*http.Cookie{userCookie}, map[string]string{"X-Workspace-Slug": slug})

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("user invite: status=%d, want 403; body=%v", resp.StatusCode, resp.Body)
	}
	errBody := resp.Body["error"].(map[string]any)
	if errBody["code"] != "tenant.permission_denied" {
		t.Errorf("code=%v, want tenant.permission_denied", errBody["code"])
	}
}

// TestInviteFlow_InviteOwnerRole: inviting with org_role_code=owner → 400.
func TestInviteFlow_InviteOwnerRole(t *testing.T) {
	app, cleanup := buildInviteTestApp(t)
	t.Cleanup(cleanup)

	ownerEmail := uniqueEmail("owner-role-inv")
	ownerCookie := signupAndLogin(t, app, ownerEmail)
	slug := uniqueSlug("ownerrole-ws")

	doRequest(t, app, "POST", "/api/v1/workspaces", map[string]any{
		"name":          "OwnerRole WS",
		"slug":          slug,
		"contact_email": ownerEmail,
	}, []*http.Cookie{ownerCookie}, nil)

	resp := doRequest(t, app, "POST", "/api/v1/workspaces/invitations", map[string]any{
		"email":         uniqueEmail("owner-invitee"),
		"org_role_code": "owner",
	}, []*http.Cookie{ownerCookie}, map[string]string{"X-Workspace-Slug": slug})

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("owner role invite: status=%d, want 400; body=%v", resp.StatusCode, resp.Body)
	}
	errBody := resp.Body["error"].(map[string]any)
	if errBody["code"] != "invitation.role_not_allowed" {
		t.Errorf("code=%v, want invitation.role_not_allowed", errBody["code"])
	}
}

// TestInviteFlow_AcceptBogusToken: accept with a non-existent token → 404.
func TestInviteFlow_AcceptBogusToken(t *testing.T) {
	app, cleanup := buildInviteTestApp(t)
	t.Cleanup(cleanup)

	email := uniqueEmail("bogus-token")
	cookie := signupAndLogin(t, app, email)

	resp := doRequest(t, app, "POST", "/api/v1/invitations/accept", map[string]any{
		"token": "this-token-does-not-exist",
	}, []*http.Cookie{cookie}, nil)

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("bogus token: status=%d, want 404; body=%v", resp.StatusCode, resp.Body)
	}
	errBody := resp.Body["error"].(map[string]any)
	if errBody["code"] != "invitation.invalid" {
		t.Errorf("code=%v, want invitation.invalid", errBody["code"])
	}
}

// TestInviteFlow_InviteAlreadyMember: invite someone already in the workspace → 409.
func TestInviteFlow_InviteAlreadyMember(t *testing.T) {
	uiPort := envOr("MAILPIT_UI_EXTERNAL_PORT", "18025")
	clearMailpitInHandler(uiPort)

	app, cleanup := buildInviteTestApp(t)
	t.Cleanup(func() {
		cleanup()
		clearMailpitInHandler(uiPort)
	})

	ownerEmail := uniqueEmail("alreadymember-owner")
	ownerCookie := signupAndLogin(t, app, ownerEmail)
	slug := uniqueSlug("alreadymember-ws")

	doRequest(t, app, "POST", "/api/v1/workspaces", map[string]any{
		"name":          "Already Member WS",
		"slug":          slug,
		"contact_email": ownerEmail,
	}, []*http.Cookie{ownerCookie}, nil)

	// Invite and accept.
	memberEmail := uniqueEmail("already-member")
	doRequest(t, app, "POST", "/api/v1/workspaces/invitations", map[string]any{
		"email":         memberEmail,
		"org_role_code": "user",
	}, []*http.Cookie{ownerCookie}, map[string]string{"X-Workspace-Slug": slug})

	time.Sleep(300 * time.Millisecond)
	rawToken := extractTokenFromMailpit(t, uiPort, memberEmail)
	memberCookie := signupAndLogin(t, app, memberEmail)
	doRequest(t, app, "POST", "/api/v1/invitations/accept", map[string]any{"token": rawToken}, []*http.Cookie{memberCookie}, nil)

	// Clear Mailpit so the second invite lookup is clean.
	clearMailpitInHandler(uiPort)

	// Try to invite the same email again → 409 already_member (detected at invite time via IsActiveMemberByEmail).
	resp := doRequest(t, app, "POST", "/api/v1/workspaces/invitations", map[string]any{
		"email":         memberEmail,
		"org_role_code": "user",
	}, []*http.Cookie{ownerCookie}, map[string]string{"X-Workspace-Slug": slug})

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("re-invite member: status=%d, want 409; body=%v", resp.StatusCode, resp.Body)
	}
	errBody := resp.Body["error"].(map[string]any)
	if errBody["code"] != "invitation.already_member" {
		t.Errorf("code=%v, want invitation.already_member", errBody["code"])
	}
}

// TestInviteFlow_DuplicatePending: invite same email twice without accepting → 409.
func TestInviteFlow_DuplicatePending(t *testing.T) {
	app, cleanup := buildInviteTestApp(t)
	t.Cleanup(cleanup)

	ownerEmail := uniqueEmail("dup-pending-owner")
	ownerCookie := signupAndLogin(t, app, ownerEmail)
	slug := uniqueSlug("dup-pending-ws")

	doRequest(t, app, "POST", "/api/v1/workspaces", map[string]any{
		"name":          "Dup Pending WS",
		"slug":          slug,
		"contact_email": ownerEmail,
	}, []*http.Cookie{ownerCookie}, nil)

	pendingEmail := uniqueEmail("dup-pending-invitee")

	// First invite — should succeed.
	resp := doRequest(t, app, "POST", "/api/v1/workspaces/invitations", map[string]any{
		"email":         pendingEmail,
		"org_role_code": "user",
	}, []*http.Cookie{ownerCookie}, map[string]string{"X-Workspace-Slug": slug})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first invite: status=%d body=%v", resp.StatusCode, resp.Body)
	}

	// Second invite to same email (not yet accepted) → 409.
	resp = doRequest(t, app, "POST", "/api/v1/workspaces/invitations", map[string]any{
		"email":         pendingEmail,
		"org_role_code": "user",
	}, []*http.Cookie{ownerCookie}, map[string]string{"X-Workspace-Slug": slug})
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("dup pending: status=%d, want 409; body=%v", resp.StatusCode, resp.Body)
	}
	errBody := resp.Body["error"].(map[string]any)
	if errBody["code"] != "invitation.already_pending" {
		t.Errorf("code=%v, want invitation.already_pending", errBody["code"])
	}
}

// TestInviteFlow_UnauthenticatedInvite: no session → 401.
func TestInviteFlow_UnauthenticatedInvite(t *testing.T) {
	app, cleanup := buildInviteTestApp(t)
	t.Cleanup(cleanup)

	resp := doRequest(t, app, "POST", "/api/v1/workspaces/invitations", map[string]any{
		"email":         "anon@example.com",
		"org_role_code": "user",
	}, nil, map[string]string{"X-Workspace-Slug": "any-slug"})

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unauth invite: status=%d, want 401", resp.StatusCode)
	}
}

// ── helpers unique to this file ───────────────────────────────────────────────

// invUniqueEmail generates a unique email with a per-test nano suffix.
func invUniqueEmail(prefix string) string {
	return fmt.Sprintf("%s+%d@example.com", prefix, time.Now().UnixNano())
}
