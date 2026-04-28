package workspacehttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/auth/authsvc"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/internal/modules/workspace/workspacesvc"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type fakeWorkspaceService struct {
	checkResult    *workspacesvc.CheckSlugResult
	checkErr       error
	checkInput     workspacesvc.CheckSlugInput
	registerResult *workspacesvc.RegisterWorkspaceResult
	registerErr    error
	registerInput  workspacesvc.RegisterWorkspaceInput
}

func (s *fakeWorkspaceService) CheckSlug(_ context.Context, input workspacesvc.CheckSlugInput) (*workspacesvc.CheckSlugResult, error) {
	s.checkInput = input
	if s.checkErr != nil {
		return nil, s.checkErr
	}
	return s.checkResult, nil
}

func (s *fakeWorkspaceService) RegisterWorkspace(_ context.Context, input workspacesvc.RegisterWorkspaceInput) (*workspacesvc.RegisterWorkspaceResult, error) {
	s.registerInput = input
	if s.registerErr != nil {
		return nil, s.registerErr
	}
	return s.registerResult, nil
}

type fakeSessionService struct {
	result *authsvc.CurrentAccountResult
	err    error
	input  authsvc.CurrentAccountInput
}

func (s *fakeSessionService) CurrentAccount(_ context.Context, input authsvc.CurrentAccountInput) (*authsvc.CurrentAccountResult, error) {
	s.input = input
	if s.err != nil {
		return nil, s.err
	}
	return s.result, nil
}

func TestCheckSlug(t *testing.T) {
	service := &fakeWorkspaceService{
		checkResult: &workspacesvc.CheckSlugResult{
			Slug:      "team-one",
			Available: true,
		},
	}
	app := newWorkspaceTestApp(newTestHandler(service, nil))

	resp := workspaceTestRequest(t, app, http.MethodGet, "/api/v1/workspaces/check-slug?slug=team-one", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if service.checkInput.Slug != "team-one" {
		t.Fatalf("slug = %s, want team-one", service.checkInput.Slug)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data := body["data"].(map[string]any)
	if data["available"] != true {
		t.Fatalf("available = %v, want true", data["available"])
	}
}

func TestRegisterWorkspace(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	workspaceID := uuid.Must(uuid.NewV7())
	tenantID := uuid.Must(uuid.NewV7())
	membershipID := uuid.Must(uuid.NewV7())
	service := &fakeWorkspaceService{
		registerResult: &workspacesvc.RegisterWorkspaceResult{
			Workspace: workspace.Workspace{
				ID:           workspaceID,
				TenantID:     tenantID,
				Name:         "Team One",
				Slug:         "team-one",
				Mode:         workspace.WorkspaceModeDemo,
				Status:       workspace.WorkspaceStatusActive,
				ContactEmail: "owner@example.test",
			},
			Membership: workspace.Membership{
				ID:          membershipID,
				WorkspaceID: workspaceID,
				Role:        workspace.WorkspaceRoleOwner,
				Status:      workspace.MembershipStatusActive,
			},
		},
	}
	session := &fakeSessionService{
		result: &authsvc.CurrentAccountResult{
			Account: auth.UserAccount{
				ID:           accountID,
				PrimaryEmail: "owner@example.test",
				Status:       auth.UserAccountStatusActive,
			},
		},
	}
	app := newWorkspaceTestApp(newTestHandler(service, session))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/register", bytes.NewBufferString(`{"name":"Team One","slug":"team-one","contact_email":"owner@example.test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status code = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
	if session.input.SessionToken != "raw-session-token" {
		t.Fatalf("session token = %s, want raw-session-token", session.input.SessionToken)
	}
	if service.registerInput.Account.ID != accountID {
		t.Fatalf("account id = %s, want %s", service.registerInput.Account.ID, accountID)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data := body["data"].(map[string]any)
	workspaceData := data["workspace"].(map[string]any)
	if _, ok := workspaceData["tenant_id"]; ok {
		t.Fatalf("response must not expose tenant_id: %#v", workspaceData)
	}
	if workspaceData["slug"] != "team-one" {
		t.Fatalf("workspace.slug = %v, want team-one", workspaceData["slug"])
	}
}

func TestRegisterWorkspaceRequiresSession(t *testing.T) {
	app := newWorkspaceTestApp(newTestHandler(&fakeWorkspaceService{}, &fakeSessionService{err: authsvc.ErrSessionTokenRequired}))

	resp := workspaceTestRequest(t, app, http.MethodPost, "/api/v1/workspaces/register", map[string]string{
		"name":          "Team One",
		"slug":          "team-one",
		"contact_email": "owner@example.test",
	})
	defer resp.Body.Close()

	assertWorkspaceError(t, resp, http.StatusUnauthorized, "AUTH_SESSION_REQUIRED")
}

func TestRegisterWorkspaceMapsValidationErrors(t *testing.T) {
	app := newWorkspaceTestApp(newTestHandler(
		&fakeWorkspaceService{registerErr: workspacesvc.ErrWorkspaceSlugTaken},
		&fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}}},
	))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/register", bytes.NewBufferString(`{"name":"Team One","slug":"team-one","contact_email":"owner@example.test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertWorkspaceError(t, resp, http.StatusConflict, "WORKSPACE_SLUG_TAKEN")
}

func TestCheckSlugUnexpectedError(t *testing.T) {
	app := newWorkspaceTestApp(newTestHandler(&fakeWorkspaceService{checkErr: errors.New("database down")}, nil))

	resp := workspaceTestRequest(t, app, http.MethodGet, "/api/v1/workspaces/check-slug?slug=team-one", nil)
	defer resp.Body.Close()

	assertWorkspaceError(t, resp, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR")
}

func newTestHandler(workspaceService WorkspaceService, sessionService SessionService) Handler {
	return NewHandler(workspaceService, sessionService, CookieConfig{
		Name:     "prasankit_session",
		TTL:      24 * time.Hour,
		SameSite: "Lax",
	})
}

func newWorkspaceTestApp(handler Handler) *fiber.App {
	app := fiber.New()
	api := app.Group("/api/v1")
	handler.RegisterRoutes(api)
	return app
}

func workspaceTestRequest(t *testing.T, app *fiber.App, method string, path string, body any) *http.Response {
	t.Helper()

	var requestBody bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&requestBody).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, &requestBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, path, err)
	}
	return resp
}

func assertWorkspaceError(t *testing.T, resp *http.Response, wantStatus int, wantCode string) {
	t.Helper()

	if resp.StatusCode != wantStatus {
		t.Fatalf("status code = %d, want %d", resp.StatusCode, wantStatus)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	gotError, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("error envelope missing or invalid: %#v", body)
	}
	if gotError["code"] != wantCode {
		t.Fatalf("error.code = %v, want %s", gotError["code"], wantCode)
	}
}
