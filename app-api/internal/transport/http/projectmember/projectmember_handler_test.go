package projectmemberhandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/project"
	"prasankit-api/internal/modules/projectmember"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/internal/transport/http/middlewares"
	projectmemberhandler "prasankit-api/internal/transport/http/projectmember"
	workspacehandler "prasankit-api/internal/transport/http/workspace"
	"prasankit-api/pkg/ids"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// ── fake repos (build the REAL projectmember.Service, mirroring 6a) ─────────────

type fakeMemberRepo struct {
	listRows []projectmember.MemberWithDisplayName
}

func (r *fakeMemberRepo) AddWithAudit(ctx context.Context, m projectmember.ProjectMember, entry audit.Entry) error {
	return nil
}
func (r *fakeMemberRepo) RemoveWithAudit(ctx context.Context, workspaceID, projectID, memberID, removedBy uuid.UUID, entry audit.Entry) error {
	return nil
}
func (r *fakeMemberRepo) ChangeRoleWithAudit(ctx context.Context, workspaceID, projectID, memberID uuid.UUID, newRoleCode string, updatedBy uuid.UUID, entry audit.Entry) error {
	return nil
}
func (r *fakeMemberRepo) ListByProject(ctx context.Context, workspaceID, projectID uuid.UUID) ([]projectmember.MemberWithDisplayName, error) {
	return r.listRows, nil
}
func (r *fakeMemberRepo) FindActiveByID(ctx context.Context, workspaceID, projectID, memberID uuid.UUID) (*projectmember.ProjectMember, error) {
	return &projectmember.ProjectMember{ID: memberID, WorkspaceID: workspaceID, ProjectID: projectID, ProjectRoleCode: "member"}, nil
}
func (r *fakeMemberRepo) IsActiveWorkspaceMember(ctx context.Context, workspaceID, membershipID uuid.UUID) (bool, error) {
	return true, nil
}
func (r *fakeMemberRepo) ProjectExistsForWorkspace(ctx context.Context, workspaceID, projectID uuid.UUID) (bool, error) {
	return true, nil
}

type fakeMasterRepo struct{}

func (r *fakeMasterRepo) IsActiveProjectStatusCode(ctx context.Context, code string) (bool, error) {
	return true, nil
}
func (r *fakeMasterRepo) IsActiveProjectTypeCode(ctx context.Context, code string) (bool, error) {
	return true, nil
}
func (r *fakeMasterRepo) IsActiveProjectRoleCode(ctx context.Context, code string) (bool, error) {
	return true, nil
}

type fakeProjectRepo struct{}

func (r *fakeProjectRepo) CreateWithAudit(ctx context.Context, p project.Project, entry audit.Entry) error {
	return nil
}
func (r *fakeProjectRepo) CreateWithOwner(ctx context.Context, p project.Project, owner project.OwnerMemberSeed, projectEntry, memberEntry audit.Entry) error {
	return nil
}
func (r *fakeProjectRepo) FindByIDForWorkspace(ctx context.Context, workspaceID, projectID uuid.UUID) (*project.Project, error) {
	// No owner set → owner-guard never triggers in handler gating tests.
	return &project.Project{ID: projectID, WorkspaceID: workspaceID}, nil
}
func (r *fakeProjectRepo) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, opts project.ListOptions) ([]project.Project, int64, error) {
	return nil, 0, nil
}
func (r *fakeProjectRepo) UpdateWithAudit(ctx context.Context, p project.Project, entry audit.Entry) error {
	return nil
}
func (r *fakeProjectRepo) SoftDeleteWithAudit(ctx context.Context, workspaceID, projectID, deletedBy uuid.UUID, entry audit.Entry) error {
	return nil
}
func (r *fakeProjectRepo) ChangeStatusWithAudit(ctx context.Context, workspaceID, projectID uuid.UUID, newStatusCode string, updatedBy uuid.UUID, entry audit.Entry) error {
	return nil
}

var _ projectmember.ProjectMemberRepository = (*fakeMemberRepo)(nil)
var _ project.MasterRepository = (*fakeMasterRepo)(nil)
var _ project.ProjectRepository = (*fakeProjectRepo)(nil)

// ── test app builder ──────────────────────────────────────────────────────────

func buildTestApp(t *testing.T, tc *workspace.TenantContext) *fiber.App {
	t.Helper()
	svc := projectmember.NewService(&fakeMemberRepo{}, &fakeMasterRepo{}, &fakeProjectRepo{})
	h := projectmemberhandler.NewHandler(svc)

	app := fiber.New()
	withTenant := func(c fiber.Ctx) error {
		if tc != nil {
			c.Locals(workspacehandler.LocalsKeyTenant, tc)
		}
		return c.Next()
	}

	app.Post("/api/v1/workspaces/projects/:id/members",
		withTenant,
		middlewares.RequireProjectMemberPermission(projectmember.PermissionAddProjectMember),
		h.HandleAdd,
	)
	app.Get("/api/v1/workspaces/projects/:id/members",
		withTenant,
		middlewares.RequireProjectMemberPermission(projectmember.PermissionReadProjectMember),
		h.HandleList,
	)
	app.Put("/api/v1/workspaces/projects/:id/members/:memberId/role",
		withTenant,
		middlewares.RequireProjectMemberPermission(projectmember.PermissionChangeProjectMemberRole),
		h.HandleChangeRole,
	)
	app.Delete("/api/v1/workspaces/projects/:id/members/:memberId",
		withTenant,
		middlewares.RequireProjectMemberPermission(projectmember.PermissionRemoveProjectMember),
		h.HandleRemove,
	)
	return app
}

func newTC(orgRole string) *workspace.TenantContext {
	wsID, _ := ids.New()
	accID, _ := ids.New()
	mID, _ := ids.New()
	return &workspace.TenantContext{
		WorkspaceID:  wsID,
		MembershipID: mID,
		AccountID:    accID,
		OrgRoleCode:  orgRole,
	}
}

func doJSON(t *testing.T, app *fiber.App, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var buf io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var parsed map[string]any
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &parsed)
	}
	return resp.StatusCode, parsed
}

func newProjectPath() string {
	id, _ := ids.New()
	return "/api/v1/workspaces/projects/" + id.String() + "/members"
}

func validAddBody() map[string]any {
	memID, _ := ids.New()
	return map[string]any{
		"workspace_membership_id": memID.String(),
		"project_role_code":       "member",
	}
}

// ── Add ───────────────────────────────────────────────────────────────────────

func TestHandler_Add_AllowedForOwnerAdmin(t *testing.T) {
	for _, role := range []string{workspace.OrgRoleOwner, workspace.OrgRoleAdmin} {
		app := buildTestApp(t, newTC(role))
		status, body := doJSON(t, app, "POST", newProjectPath(), validAddBody())
		if status != http.StatusCreated {
			t.Errorf("role %s: status = %d, want 201; body = %v", role, status, body)
		}
	}
}

func TestHandler_Add_ForbiddenForExecutiveUser(t *testing.T) {
	for _, role := range []string{workspace.OrgRoleExecutive, workspace.OrgRoleUser} {
		app := buildTestApp(t, newTC(role))
		status, body := doJSON(t, app, "POST", newProjectPath(), validAddBody())
		if status != http.StatusForbidden {
			t.Errorf("role %s: status = %d, want 403; body = %v", role, status, body)
		}
	}
}

func TestHandler_Remove_ForbiddenForUser(t *testing.T) {
	app := buildTestApp(t, newTC(workspace.OrgRoleUser))
	memID, _ := ids.New()
	pid, _ := ids.New()
	path := "/api/v1/workspaces/projects/" + pid.String() + "/members/" + memID.String()
	status, body := doJSON(t, app, "DELETE", path, nil)
	if status != http.StatusForbidden {
		t.Errorf("status = %d, want 403; body = %v", status, body)
	}
}

func TestHandler_ChangeRole_ForbiddenForExecutive(t *testing.T) {
	app := buildTestApp(t, newTC(workspace.OrgRoleExecutive))
	memID, _ := ids.New()
	pid, _ := ids.New()
	path := "/api/v1/workspaces/projects/" + pid.String() + "/members/" + memID.String() + "/role"
	status, body := doJSON(t, app, "PUT", path, map[string]any{"new_role_code": "finance"})
	if status != http.StatusForbidden {
		t.Errorf("status = %d, want 403; body = %v", status, body)
	}
}

func TestHandler_List_AllowedForAllOrgRoles(t *testing.T) {
	for _, role := range []string{workspace.OrgRoleOwner, workspace.OrgRoleAdmin, workspace.OrgRoleExecutive, workspace.OrgRoleUser} {
		app := buildTestApp(t, newTC(role))
		status, body := doJSON(t, app, "GET", newProjectPath(), nil)
		if status != http.StatusOK {
			t.Errorf("role %s: status = %d, want 200; body = %v", role, status, body)
		}
	}
}

func TestHandler_Add_BadUUIDMembershipID_400(t *testing.T) {
	app := buildTestApp(t, newTC(workspace.OrgRoleOwner))
	body := map[string]any{
		"workspace_membership_id": "not-a-uuid",
		"project_role_code":       "member",
	}
	status, respBody := doJSON(t, app, "POST", newProjectPath(), body)
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %v", status, respBody)
	}
	errObj := respBody["error"].(map[string]any)
	if errObj["code"] != "validation.invalid_input" {
		t.Errorf("code = %v, want validation.invalid_input", errObj["code"])
	}
}

func TestHandler_Unauthenticated_401_WhenNoTenantContext(t *testing.T) {
	app := buildTestApp(t, nil) // no tenant context injected
	status, respBody := doJSON(t, app, "POST", newProjectPath(), validAddBody())
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body = %v", status, respBody)
	}
	errObj := respBody["error"].(map[string]any)
	if errObj["code"] != "auth.unauthenticated" {
		t.Errorf("code = %v, want auth.unauthenticated", errObj["code"])
	}
}
