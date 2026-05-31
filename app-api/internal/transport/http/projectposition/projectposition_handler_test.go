package projectpositionhandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/projectposition"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/internal/transport/http/middlewares"
	projectpositionhandler "prasankit-api/internal/transport/http/projectposition"
	workspacehandler "prasankit-api/internal/transport/http/workspace"
	"prasankit-api/pkg/ids"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// ── fake repo (build the REAL projectposition.Service) ──────────────────────────

type fakeProjectPositionRepo struct {
	createErr  error
	findResult *projectposition.ProjectPosition
}

func (r *fakeProjectPositionRepo) CreateWithAudit(ctx context.Context, p projectposition.ProjectPosition, entry audit.Entry) error {
	return r.createErr
}
func (r *fakeProjectPositionRepo) FindByCodeForWorkspace(ctx context.Context, workspaceID uuid.UUID, code string) (*projectposition.ProjectPosition, error) {
	if r.findResult != nil {
		cp := *r.findResult
		return &cp, nil
	}
	return nil, nil
}
func (r *fakeProjectPositionRepo) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, opts projectposition.ListOptions) ([]projectposition.ProjectPosition, int64, error) {
	return nil, 0, nil
}
func (r *fakeProjectPositionRepo) UpdateWithAudit(ctx context.Context, p projectposition.ProjectPosition, entry audit.Entry) error {
	return nil
}
func (r *fakeProjectPositionRepo) DeprecateWithAudit(ctx context.Context, workspaceID uuid.UUID, code string, updatedBy uuid.UUID, entry audit.Entry) error {
	return nil
}

var _ projectposition.ProjectPositionRepository = (*fakeProjectPositionRepo)(nil)

// ── test app builder ──────────────────────────────────────────────────────────

func buildTestApp(t *testing.T, tc *workspace.TenantContext, repo *fakeProjectPositionRepo) *fiber.App {
	t.Helper()
	if repo == nil {
		repo = &fakeProjectPositionRepo{}
	}
	svc := projectposition.NewService(repo)
	h := projectpositionhandler.NewHandler(svc)

	app := fiber.New()
	withTenant := func(c fiber.Ctx) error {
		if tc != nil {
			c.Locals(workspacehandler.LocalsKeyTenant, tc)
		}
		return c.Next()
	}

	app.Post("/api/v1/workspaces/project-positions",
		withTenant,
		middlewares.RequireProjectPositionPermission(projectposition.PermissionCreateProjectPosition),
		h.HandleCreate,
	)
	app.Get("/api/v1/workspaces/project-positions",
		withTenant,
		middlewares.RequireProjectPositionPermission(projectposition.PermissionReadProjectPosition),
		h.HandleList,
	)
	app.Put("/api/v1/workspaces/project-positions/:code",
		withTenant,
		middlewares.RequireProjectPositionPermission(projectposition.PermissionUpdateProjectPosition),
		h.HandleUpdate,
	)
	app.Post("/api/v1/workspaces/project-positions/:code/deprecate",
		withTenant,
		middlewares.RequireProjectPositionPermission(projectposition.PermissionDeprecateProjectPosition),
		h.HandleDeprecate,
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
	return map[string]any{
		"code":       "tech_lead",
		"label_th":   "หัวหน้าเทคนิค",
		"label_en":   "Technical Lead",
		"sort_order": 10,
	}
}

// ── tests ───────────────────────────────────────────────────────────────────────

func TestProjectPositionHandler_ServiceErrorMapping(t *testing.T) {
	existingActive := func(tc *workspace.TenantContext) *projectposition.ProjectPosition {
		id, _ := ids.New()
		return &projectposition.ProjectPosition{ID: id, WorkspaceID: tc.WorkspaceID, Code: "tech_lead", LabelTH: "a", LabelEN: "b", Status: "active"}
	}
	systemRow := func(tc *workspace.TenantContext) *projectposition.ProjectPosition {
		p := existingActive(tc)
		p.IsSystem = true
		return p
	}

	type tcase struct {
		name     string
		method   string
		path     string
		body     any
		repo     func(tc *workspace.TenantContext) *fakeProjectPositionRepo
		wantCode int
		wantMC   string
	}
	cases := []tcase{
		{
			name:   "code_taken",
			method: "POST", path: "/api/v1/workspaces/project-positions", body: validCreateBody(),
			repo: func(tc *workspace.TenantContext) *fakeProjectPositionRepo {
				return &fakeProjectPositionRepo{findResult: existingActive(tc)}
			},
			wantCode: http.StatusConflict, wantMC: "project_position.code_taken",
		},
		{
			name:   "code_immutable",
			method: "PUT", path: "/api/v1/workspaces/project-positions/tech_lead",
			body: map[string]any{"code": "other", "label_th": "a", "label_en": "b"},
			repo: func(tc *workspace.TenantContext) *fakeProjectPositionRepo {
				return &fakeProjectPositionRepo{findResult: existingActive(tc)}
			},
			wantCode: http.StatusUnprocessableEntity, wantMC: "project_position.code_immutable",
		},
		{
			name:   "system_immutable",
			method: "PUT", path: "/api/v1/workspaces/project-positions/tech_lead",
			body: map[string]any{"label_th": "a", "label_en": "b"},
			repo: func(tc *workspace.TenantContext) *fakeProjectPositionRepo {
				return &fakeProjectPositionRepo{findResult: systemRow(tc)}
			},
			wantCode: http.StatusUnprocessableEntity, wantMC: "project_position.system_immutable",
		},
		{
			name:   "not_found",
			method: "PUT", path: "/api/v1/workspaces/project-positions/ghost",
			body: map[string]any{"label_th": "a", "label_en": "b"},
			repo: func(tc *workspace.TenantContext) *fakeProjectPositionRepo {
				return &fakeProjectPositionRepo{findResult: nil}
			},
			wantCode: http.StatusNotFound, wantMC: "project_position.not_found",
		},
		{
			name:   "validation",
			method: "POST", path: "/api/v1/workspaces/project-positions",
			body:     map[string]any{"code": "", "label_th": "a", "label_en": "b"},
			repo:     func(tc *workspace.TenantContext) *fakeProjectPositionRepo { return &fakeProjectPositionRepo{} },
			wantCode: http.StatusBadRequest, wantMC: "validation.invalid_input",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tc := newTC(workspace.OrgRoleOwner)
			app := buildTestApp(t, tc, c.repo(tc))
			status, body := doJSON(t, app, c.method, c.path, c.body)
			if status != c.wantCode {
				t.Fatalf("status = %d, want %d; body = %v", status, c.wantCode, body)
			}
			errObj, _ := body["error"].(map[string]any)
			if errObj == nil || errObj["code"] != c.wantMC {
				t.Errorf("code = %v, want %s", errObj["code"], c.wantMC)
			}
		})
	}
}

func TestProjectPositionHandler_Create_201(t *testing.T) {
	tc := newTC(workspace.OrgRoleOwner)
	app := buildTestApp(t, tc, &fakeProjectPositionRepo{})
	status, body := doJSON(t, app, "POST", "/api/v1/workspaces/project-positions", validCreateBody())
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %v", status, body)
	}
}

func TestProjectPositionHandler_Unauthorized_NoTenant_401(t *testing.T) {
	app := buildTestApp(t, nil, &fakeProjectPositionRepo{})
	status, body := doJSON(t, app, "POST", "/api/v1/workspaces/project-positions", validCreateBody())
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body = %v", status, body)
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj == nil || errObj["code"] != "auth.unauthenticated" {
		t.Errorf("code = %v, want auth.unauthenticated", errObj["code"])
	}
}

func TestProjectPositionHandler_Forbidden_WrongRole_403(t *testing.T) {
	for _, role := range []string{workspace.OrgRoleExecutive, workspace.OrgRoleUser} {
		app := buildTestApp(t, newTC(role), &fakeProjectPositionRepo{})
		status, body := doJSON(t, app, "POST", "/api/v1/workspaces/project-positions", validCreateBody())
		if status != http.StatusForbidden {
			t.Errorf("role %s: status = %d, want 403; body = %v", role, status, body)
		}
	}
}
