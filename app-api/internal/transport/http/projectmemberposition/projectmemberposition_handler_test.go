package projectmemberpositionhandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/projectmember"
	"prasankit-api/internal/modules/projectmemberposition"
	"prasankit-api/internal/modules/projectposition"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/internal/transport/http/middlewares"
	projectmemberpositionhandler "prasankit-api/internal/transport/http/projectmemberposition"
	workspacehandler "prasankit-api/internal/transport/http/workspace"
	"prasankit-api/pkg/ids"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// ── fakes (build the REAL projectmemberposition.Service) ────────────────────────

type fakeJunctionRepo struct {
	assignErr   error
	unassignErr error
}

func (r *fakeJunctionRepo) AssignWithAudit(ctx context.Context, mp projectmemberposition.ProjectMemberPosition, entry audit.Entry) error {
	return r.assignErr
}
func (r *fakeJunctionRepo) UnassignWithAudit(ctx context.Context, workspaceID, projectMemberID uuid.UUID, positionCode string, entry audit.Entry) error {
	return r.unassignErr
}
func (r *fakeJunctionRepo) ListByMember(ctx context.Context, workspaceID, projectMemberID uuid.UUID) ([]projectmemberposition.PositionWithLabel, error) {
	return nil, nil
}

// fakeMemberRepo stubs ALL 8 projectmember.ProjectMemberRepository methods (incl. FindByID).
type fakeMemberRepo struct{}

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
	return nil, nil
}
func (r *fakeMemberRepo) FindActiveByID(ctx context.Context, workspaceID, projectID, memberID uuid.UUID) (*projectmember.ProjectMember, error) {
	return &projectmember.ProjectMember{ID: memberID, WorkspaceID: workspaceID, ProjectID: projectID}, nil
}
func (r *fakeMemberRepo) FindByID(ctx context.Context, workspaceID, projectID, memberID uuid.UUID) (*projectmember.ProjectMember, error) {
	// active member (RemovedAt nil) so the guard passes in gating tests.
	return &projectmember.ProjectMember{ID: memberID, WorkspaceID: workspaceID, ProjectID: projectID}, nil
}
func (r *fakeMemberRepo) IsActiveWorkspaceMember(ctx context.Context, workspaceID, membershipID uuid.UUID) (bool, error) {
	return true, nil
}
func (r *fakeMemberRepo) ProjectExistsForWorkspace(ctx context.Context, workspaceID, projectID uuid.UUID) (bool, error) {
	return true, nil
}

type fakePositionMasterRepo struct{}

func (r *fakePositionMasterRepo) IsActiveProjectPositionCode(ctx context.Context, workspaceID uuid.UUID, code string) (bool, error) {
	return true, nil
}

// Three explicit compile-time interface assertions.
var _ projectmemberposition.ProjectMemberPositionRepository = (*fakeJunctionRepo)(nil)
var _ projectmember.ProjectMemberRepository = (*fakeMemberRepo)(nil)
var _ projectposition.PositionMasterRepository = (*fakePositionMasterRepo)(nil)

// ── test app builder ──────────────────────────────────────────────────────────

func buildTestApp(t *testing.T, tc *workspace.TenantContext) *fiber.App {
	t.Helper()
	svc := projectmemberposition.NewService(&fakeJunctionRepo{}, &fakeMemberRepo{}, &fakePositionMasterRepo{})
	h := projectmemberpositionhandler.NewHandler(svc)

	app := fiber.New()
	withTenant := func(c fiber.Ctx) error {
		if tc != nil {
			c.Locals(workspacehandler.LocalsKeyTenant, tc)
		}
		return c.Next()
	}

	app.Post("/api/v1/workspaces/projects/:id/members/:memberId/positions",
		withTenant,
		middlewares.RequireProjectMemberPositionPermission(projectmemberposition.PermissionAssignPosition),
		h.HandleAssign,
	)
	app.Get("/api/v1/workspaces/projects/:id/members/:memberId/positions",
		withTenant,
		middlewares.RequireProjectMemberPositionPermission(projectmemberposition.PermissionReadPosition),
		h.HandleList,
	)
	app.Delete("/api/v1/workspaces/projects/:id/members/:memberId/positions/:code",
		withTenant,
		middlewares.RequireProjectMemberPositionPermission(projectmemberposition.PermissionUnassignPosition),
		h.HandleUnassign,
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

func positionsPath() string {
	pid, _ := ids.New()
	mid, _ := ids.New()
	return "/api/v1/workspaces/projects/" + pid.String() + "/members/" + mid.String() + "/positions"
}

// ── tests ───────────────────────────────────────────────────────────────────────

func TestProjectMemberPositionHandler_Assign_AllowedForOwnerAdmin(t *testing.T) {
	for _, role := range []string{workspace.OrgRoleOwner, workspace.OrgRoleAdmin} {
		app := buildTestApp(t, newTC(role))
		req := httptest.NewRequest("POST", positionsPath(), bytes.NewReader(mustJSON(map[string]any{"project_position_code": "tech_lead"})))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("role %s: status = %d, want 201", role, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

func TestProjectMemberPositionHandler_Assign_ForbiddenForUser_403(t *testing.T) {
	app := buildTestApp(t, newTC(workspace.OrgRoleUser))
	req := httptest.NewRequest("POST", positionsPath(), bytes.NewReader(mustJSON(map[string]any{"project_position_code": "tech_lead"})))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want 403", resp.StatusCode)
	}
}

func TestProjectMemberPositionHandler_Unauthorized_NoTenant_401(t *testing.T) {
	app := buildTestApp(t, nil)
	req := httptest.NewRequest("POST", positionsPath(), bytes.NewReader(mustJSON(map[string]any{"project_position_code": "tech_lead"})))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestProjectMemberPositionHandler_Unassign_204_EmptyBody(t *testing.T) {
	app := buildTestApp(t, newTC(workspace.OrgRoleOwner))
	pid, _ := ids.New()
	mid, _ := ids.New()
	path := "/api/v1/workspaces/projects/" + pid.String() + "/members/" + mid.String() + "/positions/tech_lead"
	req := httptest.NewRequest("DELETE", path, nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	if len(raw) != 0 {
		t.Errorf("body length = %d, want 0 (empty 204)", len(raw))
	}
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
