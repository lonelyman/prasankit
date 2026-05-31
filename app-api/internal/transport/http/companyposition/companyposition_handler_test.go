package companypositionhandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/companyposition"
	"prasankit-api/internal/modules/workspace"
	companypositionhandler "prasankit-api/internal/transport/http/companyposition"
	"prasankit-api/internal/transport/http/middlewares"
	workspacehandler "prasankit-api/internal/transport/http/workspace"
	"prasankit-api/pkg/ids"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// ── fakes (build the REAL companyposition.Service) ──────────────────────────────

type fakeCompanyPositionRepo struct {
	createErr  error
	findResult *companyposition.CompanyPosition
}

func (r *fakeCompanyPositionRepo) CreateWithAudit(ctx context.Context, p companyposition.CompanyPosition, entry audit.Entry) error {
	return r.createErr
}
func (r *fakeCompanyPositionRepo) FindByCodeForWorkspace(ctx context.Context, workspaceID uuid.UUID, code string) (*companyposition.CompanyPosition, error) {
	if r.findResult != nil {
		cp := *r.findResult
		return &cp, nil
	}
	return nil, nil
}
func (r *fakeCompanyPositionRepo) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, opts companyposition.ListOptions) ([]companyposition.CompanyPosition, int64, error) {
	return nil, 0, nil
}
func (r *fakeCompanyPositionRepo) UpdateWithAudit(ctx context.Context, p companyposition.CompanyPosition, entry audit.Entry) error {
	return nil
}
func (r *fakeCompanyPositionRepo) DeprecateWithAudit(ctx context.Context, workspaceID uuid.UUID, code string, updatedBy uuid.UUID, entry audit.Entry) error {
	return nil
}

type fakeMaster struct{ active bool }

func (r *fakeMaster) IsActiveCompanyPositionCode(ctx context.Context, workspaceID uuid.UUID, code string) (bool, error) {
	return r.active, nil
}

type fakeMembership struct{ setErr error }

func (r *fakeMembership) SetCompanyPositionWithAudit(ctx context.Context, workspaceID, membershipID uuid.UUID, code *string, updatedBy uuid.UUID, entry audit.Entry) error {
	return r.setErr
}

var _ companyposition.CompanyPositionRepository = (*fakeCompanyPositionRepo)(nil)
var _ companyposition.CompanyPositionMasterRepository = (*fakeMaster)(nil)
var _ companyposition.MembershipPositionRepository = (*fakeMembership)(nil)

// ── test app builder ──────────────────────────────────────────────────────────

func buildTestApp(t *testing.T, tc *workspace.TenantContext, repo *fakeCompanyPositionRepo, master *fakeMaster, mem *fakeMembership) *fiber.App {
	t.Helper()
	if repo == nil {
		repo = &fakeCompanyPositionRepo{}
	}
	if master == nil {
		master = &fakeMaster{active: true}
	}
	if mem == nil {
		mem = &fakeMembership{}
	}
	svc := companyposition.NewService(repo, master, mem)
	h := companypositionhandler.NewHandler(svc)

	app := fiber.New()
	withTenant := func(c fiber.Ctx) error {
		if tc != nil {
			c.Locals(workspacehandler.LocalsKeyTenant, tc)
		}
		return c.Next()
	}

	app.Post("/api/v1/workspaces/company-positions",
		withTenant,
		middlewares.RequireCompanyPositionPermission(companyposition.PermissionCreateCompanyPosition),
		h.HandleCreate,
	)
	app.Get("/api/v1/workspaces/company-positions",
		withTenant,
		middlewares.RequireCompanyPositionPermission(companyposition.PermissionReadCompanyPosition),
		h.HandleList,
	)
	app.Put("/api/v1/workspaces/company-positions/:code",
		withTenant,
		middlewares.RequireCompanyPositionPermission(companyposition.PermissionUpdateCompanyPosition),
		h.HandleUpdate,
	)
	app.Post("/api/v1/workspaces/company-positions/:code/deprecate",
		withTenant,
		middlewares.RequireCompanyPositionPermission(companyposition.PermissionDeprecateCompanyPosition),
		h.HandleDeprecate,
	)
	app.Put("/api/v1/workspaces/memberships/:membershipId/company-position",
		withTenant,
		middlewares.RequireCompanyPositionPermission(companyposition.PermissionAssignCompanyPosition),
		h.HandleSetCompanyPosition,
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

func validCreateBody() map[string]any {
	return map[string]any{"code": "engineer", "label_th": "วิศวกร", "label_en": "Engineer", "sort_order": 10}
}

// ── tests ───────────────────────────────────────────────────────────────────────

func TestCompanyPositionHandler_ServiceErrorMapping(t *testing.T) {
	existingActive := func(tc *workspace.TenantContext) *companyposition.CompanyPosition {
		id, _ := ids.New()
		return &companyposition.CompanyPosition{ID: id, WorkspaceID: tc.WorkspaceID, Code: "engineer", LabelTH: "a", LabelEN: "b", Status: "active"}
	}

	tc := newTC(workspace.OrgRoleOwner)

	// code_taken
	app := buildTestApp(t, tc, &fakeCompanyPositionRepo{findResult: existingActive(tc)}, nil, nil)
	if status, body := doJSON(t, app, "POST", "/api/v1/workspaces/company-positions", validCreateBody()); status != http.StatusConflict {
		t.Errorf("code_taken: status = %d, want 409; body=%v", status, body)
	}

	// invalid_position_code (ws-attach, master inactive)
	mID, _ := ids.New()
	app = buildTestApp(t, tc, nil, &fakeMaster{active: false}, nil)
	if status, body := doJSON(t, app, "PUT", "/api/v1/workspaces/memberships/"+mID.String()+"/company-position",
		map[string]any{"company_position_code": "ghost"}); status != http.StatusUnprocessableEntity {
		t.Errorf("invalid_position_code: status = %d, want 422; body=%v", status, body)
	}

	// membership_not_found (ws-attach)
	app = buildTestApp(t, tc, nil, &fakeMaster{active: true}, &fakeMembership{setErr: companyposition.ErrMembershipNotFound})
	if status, body := doJSON(t, app, "PUT", "/api/v1/workspaces/memberships/"+mID.String()+"/company-position",
		map[string]any{"company_position_code": "engineer"}); status != http.StatusNotFound {
		t.Errorf("membership_not_found: status = %d, want 404; body=%v", status, body)
	}
}

func TestCompanyPositionHandler_Create_201(t *testing.T) {
	tc := newTC(workspace.OrgRoleOwner)
	app := buildTestApp(t, tc, nil, nil, nil)
	if status, body := doJSON(t, app, "POST", "/api/v1/workspaces/company-positions", validCreateBody()); status != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %v", status, body)
	}
}

func TestCompanyPositionHandler_SetCompanyPosition_200(t *testing.T) {
	tc := newTC(workspace.OrgRoleOwner)
	app := buildTestApp(t, tc, nil, &fakeMaster{active: true}, &fakeMembership{})
	mID, _ := ids.New()
	if status, body := doJSON(t, app, "PUT", "/api/v1/workspaces/memberships/"+mID.String()+"/company-position",
		map[string]any{"company_position_code": "engineer"}); status != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %v", status, body)
	}
}

func TestCompanyPositionHandler_Unauthorized_NoTenant_401(t *testing.T) {
	app := buildTestApp(t, nil, nil, nil, nil)
	status, body := doJSON(t, app, "POST", "/api/v1/workspaces/company-positions", validCreateBody())
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body = %v", status, body)
	}
}

func TestCompanyPositionHandler_Forbidden_WrongRole_403(t *testing.T) {
	for _, role := range []string{workspace.OrgRoleExecutive, workspace.OrgRoleUser} {
		app := buildTestApp(t, newTC(role), nil, nil, nil)
		status, body := doJSON(t, app, "POST", "/api/v1/workspaces/company-positions", validCreateBody())
		if status != http.StatusForbidden {
			t.Errorf("role %s: status = %d, want 403; body = %v", role, status, body)
		}
	}
}
