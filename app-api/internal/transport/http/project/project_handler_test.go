package projecthandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/project"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/internal/transport/http/middlewares"
	projecthandler "prasankit-api/internal/transport/http/project"
	workspacehandler "prasankit-api/internal/transport/http/workspace"
	"prasankit-api/pkg/ids"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// ── fake repos ────────────────────────────────────────────────────────────────

type fakeProjectRepo struct {
	mu             sync.Mutex
	lastListOpts   project.ListOptions
	lastListWS     uuid.UUID
	createErr      error
	updateErr      error
	deleteErr      error
	statusErr      error
	createReturn   error
	listRows       []project.Project
	listTotal      int64
	findResult     *project.Project
	findReturnsNil bool
}

func (r *fakeProjectRepo) CreateWithAudit(ctx context.Context, p project.Project, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.createReturn != nil {
		return r.createReturn
	}
	return r.createErr
}
func (r *fakeProjectRepo) CreateWithOwner(ctx context.Context, p project.Project, owner project.OwnerMemberSeed, projectEntry, memberEntry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.createReturn != nil {
		return r.createReturn
	}
	return r.createErr
}
func (r *fakeProjectRepo) FindByIDForWorkspace(ctx context.Context, workspaceID, projectID uuid.UUID) (*project.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.findReturnsNil {
		return nil, nil
	}
	if r.findResult != nil {
		cp := *r.findResult
		return &cp, nil
	}
	return &project.Project{
		ID:                projectID,
		WorkspaceID:       workspaceID,
		ProjectName:       "Existing",
		ProjectTypeCode:   "client",
		ProjectStatusCode: "planning",
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}, nil
}
func (r *fakeProjectRepo) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, opts project.ListOptions) ([]project.Project, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastListOpts = opts
	r.lastListWS = workspaceID
	return r.listRows, r.listTotal, nil
}
func (r *fakeProjectRepo) UpdateWithAudit(ctx context.Context, p project.Project, entry audit.Entry) error {
	return r.updateErr
}
func (r *fakeProjectRepo) SoftDeleteWithAudit(ctx context.Context, workspaceID, projectID, deletedBy uuid.UUID, entry audit.Entry) error {
	return r.deleteErr
}
func (r *fakeProjectRepo) ChangeStatusWithAudit(ctx context.Context, workspaceID, projectID uuid.UUID, newStatusCode string, updatedBy uuid.UUID, entry audit.Entry) error {
	return r.statusErr
}

type fakeMasterRepo struct {
	statusActive bool
	typeActive   bool
	roleActive   bool
}

func (r *fakeMasterRepo) IsActiveProjectStatusCode(ctx context.Context, code string) (bool, error) {
	return r.statusActive, nil
}
func (r *fakeMasterRepo) IsActiveProjectTypeCode(ctx context.Context, code string) (bool, error) {
	return r.typeActive, nil
}
func (r *fakeMasterRepo) IsActiveProjectRoleCode(ctx context.Context, code string) (bool, error) {
	return r.roleActive, nil
}

// ── test app builder ──────────────────────────────────────────────────────────

func buildTestApp(t *testing.T, tc *workspace.TenantContext, fr *fakeProjectRepo, fm *fakeMasterRepo) *fiber.App {
	t.Helper()
	if fr == nil {
		fr = &fakeProjectRepo{listRows: []project.Project{}, listTotal: 0}
	}
	if fm == nil {
		fm = &fakeMasterRepo{statusActive: true, typeActive: true, roleActive: true}
	}
	svc := project.NewService(fr, fm)
	h := projecthandler.NewHandler(svc)

	app := fiber.New()

	// Inject TenantContext (or skip when tc is nil) via a setup middleware.
	withTenant := func(c fiber.Ctx) error {
		if tc != nil {
			c.Locals(workspacehandler.LocalsKeyTenant, tc)
		}
		return c.Next()
	}

	// Permission middlewares — same as router wiring.
	app.Post("/api/v1/workspaces/projects",
		withTenant,
		middlewares.RequireProjectPermission(project.PermissionCreateProject),
		h.HandleCreate,
	)
	app.Get("/api/v1/workspaces/projects",
		withTenant,
		middlewares.RequireProjectPermission(project.PermissionReadProject),
		h.HandleList,
	)
	app.Get("/api/v1/workspaces/projects/:id",
		withTenant,
		middlewares.RequireProjectPermission(project.PermissionReadProject),
		h.HandleGet,
	)
	app.Put("/api/v1/workspaces/projects/:id",
		withTenant,
		middlewares.RequireProjectPermission(project.PermissionUpdateProject),
		h.HandleUpdate,
	)
	app.Delete("/api/v1/workspaces/projects/:id",
		withTenant,
		middlewares.RequireProjectPermission(project.PermissionDeleteProject),
		h.HandleDelete,
	)
	app.Post("/api/v1/workspaces/projects/:id/status",
		withTenant,
		middlewares.RequireProjectPermission(project.PermissionChangeProjectStatus),
		h.HandleChangeStatus,
	)
	return app
}

func newTC(orgRole string) *workspace.TenantContext {
	wsID, _ := ids.New()
	accID, _ := ids.New()
	return &workspace.TenantContext{
		WorkspaceID: wsID,
		AccountID:   accID,
		OrgRoleCode: orgRole,
	}
}

func doJSON(t *testing.T, app *fiber.App, method, path string, body any) (int, map[string]any, []byte) {
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
	return resp.StatusCode, parsed, raw
}

func doRaw(t *testing.T, app *fiber.App, method, path string, body any) *http.Response {
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
	return resp
}

func validCreateBody() map[string]any {
	return map[string]any{
		"project_name":        "Test Project",
		"slug":                "test-project",
		"project_type_code":   "client",
		"project_status_code": "planning",
	}
}

// ── Permission gating ─────────────────────────────────────────────────────────

func TestProjectHandler_CreatePermission_OrgRoleOwner_201(t *testing.T) {
	tc := newTC(workspace.OrgRoleOwner)
	app := buildTestApp(t, tc, nil, nil)
	status, body, _ := doJSON(t, app, "POST", "/api/v1/workspaces/projects", validCreateBody())
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %v", status, body)
	}
	data := body["data"].(map[string]any)
	if data["id"] == nil || data["id"] == "" {
		t.Errorf("missing data.id; body = %v", body)
	}
}

func TestProjectHandler_CreatePermission_OrgRoleAdmin_201(t *testing.T) {
	tc := newTC(workspace.OrgRoleAdmin)
	app := buildTestApp(t, tc, nil, nil)
	status, body, _ := doJSON(t, app, "POST", "/api/v1/workspaces/projects", validCreateBody())
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %v", status, body)
	}
}

func TestProjectHandler_CreatePermission_OrgRoleExecutive_403(t *testing.T) {
	tc := newTC(workspace.OrgRoleExecutive)
	app := buildTestApp(t, tc, nil, nil)
	status, body, _ := doJSON(t, app, "POST", "/api/v1/workspaces/projects", validCreateBody())
	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body = %v", status, body)
	}
	errObj := body["error"].(map[string]any)
	if errObj["code"] != "tenant.permission_denied" {
		t.Errorf("code = %v, want tenant.permission_denied", errObj["code"])
	}
}

func TestProjectHandler_CreatePermission_OrgRoleUser_403(t *testing.T) {
	tc := newTC(workspace.OrgRoleUser)
	app := buildTestApp(t, tc, nil, nil)
	status, _, _ := doJSON(t, app, "POST", "/api/v1/workspaces/projects", validCreateBody())
	if status != http.StatusForbidden {
		t.Errorf("status = %d, want 403", status)
	}
}

func TestProjectHandler_ReadPermission_OrgRoleUser_200(t *testing.T) {
	tc := newTC(workspace.OrgRoleUser)
	app := buildTestApp(t, tc, nil, nil)
	status, _, _ := doJSON(t, app, "GET", "/api/v1/workspaces/projects", nil)
	if status != http.StatusOK {
		t.Errorf("status = %d, want 200", status)
	}
}

func TestProjectHandler_ReadPermission_OrgRoleExecutive_200(t *testing.T) {
	tc := newTC(workspace.OrgRoleExecutive)
	app := buildTestApp(t, tc, nil, nil)
	status, _, _ := doJSON(t, app, "GET", "/api/v1/workspaces/projects", nil)
	if status != http.StatusOK {
		t.Errorf("status = %d, want 200", status)
	}
}

func TestProjectHandler_MissingTenantContext_401(t *testing.T) {
	app := buildTestApp(t, nil, nil, nil)
	status, body, _ := doJSON(t, app, "POST", "/api/v1/workspaces/projects", validCreateBody())
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body = %v", status, body)
	}
	errObj := body["error"].(map[string]any)
	if errObj["code"] != "auth.unauthenticated" {
		t.Errorf("code = %v, want auth.unauthenticated", errObj["code"])
	}
}

// ── Body / path validation ────────────────────────────────────────────────────

func TestProjectHandler_InvalidJSONBody_400(t *testing.T) {
	tc := newTC(workspace.OrgRoleOwner)
	app := buildTestApp(t, tc, nil, nil)
	req := httptest.NewRequest("POST", "/api/v1/workspaces/projects", bytes.NewReader([]byte("{not json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)
	errObj := body["error"].(map[string]any)
	if errObj["code"] != "validation.invalid_input" {
		t.Errorf("code = %v, want validation.invalid_input", errObj["code"])
	}
}

func TestProjectHandler_InvalidUUIDInPath_400(t *testing.T) {
	tc := newTC(workspace.OrgRoleOwner)
	app := buildTestApp(t, tc, nil, nil)
	status, body, _ := doJSON(t, app, "GET", "/api/v1/workspaces/projects/not-a-uuid", nil)
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %v", status, body)
	}
	errObj := body["error"].(map[string]any)
	if errObj["code"] != "validation.invalid_input" {
		t.Errorf("code = %v, want validation.invalid_input", errObj["code"])
	}
}

func TestProjectHandler_PaginationDefaults(t *testing.T) {
	tc := newTC(workspace.OrgRoleOwner)
	fr := &fakeProjectRepo{}
	app := buildTestApp(t, tc, fr, nil)
	status, _, _ := doJSON(t, app, "GET", "/api/v1/workspaces/projects", nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	fr.mu.Lock()
	defer fr.mu.Unlock()
	if fr.lastListOpts.Page != 1 {
		t.Errorf("page = %d, want 1", fr.lastListOpts.Page)
	}
	if fr.lastListOpts.Limit != 10 {
		t.Errorf("limit = %d, want 10", fr.lastListOpts.Limit)
	}
}

func TestProjectHandler_Pagination_LimitTooHigh_400(t *testing.T) {
	tc := newTC(workspace.OrgRoleOwner)
	app := buildTestApp(t, tc, nil, nil)
	status, body, _ := doJSON(t, app, "GET", "/api/v1/workspaces/projects?limit=101", nil)
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %v", status, body)
	}
}

func TestProjectHandler_Pagination_NonNumericPage_400(t *testing.T) {
	tc := newTC(workspace.OrgRoleOwner)
	app := buildTestApp(t, tc, nil, nil)
	status, body, _ := doJSON(t, app, "GET", "/api/v1/workspaces/projects?page=abc", nil)
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %v", status, body)
	}
}

// ── Service error mapping (table-driven) ──────────────────────────────────────

func TestProjectHandler_ServiceErrorMapping(t *testing.T) {
	tc := newTC(workspace.OrgRoleOwner)

	cases := []struct {
		name     string
		setup    func(*fakeProjectRepo, *fakeMasterRepo)
		body     map[string]any
		method   string
		path     string
		wantCode int
		wantMC   string
	}{
		{
			name: "ErrSlugTaken",
			setup: func(fr *fakeProjectRepo, fm *fakeMasterRepo) {
				fr.createReturn = project.ErrSlugTaken
			},
			body:     validCreateBody(),
			method:   "POST",
			path:     "/api/v1/workspaces/projects",
			wantCode: http.StatusConflict,
			wantMC:   "project.slug_taken",
		},
		{
			name: "ErrInvalidStatusCode",
			setup: func(fr *fakeProjectRepo, fm *fakeMasterRepo) {
				fm.statusActive = false
			},
			body:     validCreateBody(),
			method:   "POST",
			path:     "/api/v1/workspaces/projects",
			wantCode: http.StatusUnprocessableEntity,
			wantMC:   "project.invalid_master_code",
		},
		{
			name: "ErrInvalidTypeCode",
			setup: func(fr *fakeProjectRepo, fm *fakeMasterRepo) {
				fm.typeActive = false
			},
			body:     validCreateBody(),
			method:   "POST",
			path:     "/api/v1/workspaces/projects",
			wantCode: http.StatusUnprocessableEntity,
			wantMC:   "project.invalid_master_code",
		},
		{
			name:  "ValidationError",
			setup: func(fr *fakeProjectRepo, fm *fakeMasterRepo) {},
			body: map[string]any{
				"project_name":        "",
				"slug":                "ok-slug",
				"project_type_code":   "client",
				"project_status_code": "planning",
			},
			method:   "POST",
			path:     "/api/v1/workspaces/projects",
			wantCode: http.StatusBadRequest,
			wantMC:   "validation.invalid_input",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fr := &fakeProjectRepo{}
			fm := &fakeMasterRepo{statusActive: true, typeActive: true, roleActive: true}
			if c.setup != nil {
				c.setup(fr, fm)
			}
			app := buildTestApp(t, tc, fr, fm)
			status, body, _ := doJSON(t, app, c.method, c.path, c.body)
			if status != c.wantCode {
				t.Fatalf("status = %d, want %d; body = %v", status, c.wantCode, body)
			}
			errObj, _ := body["error"].(map[string]any)
			if errObj == nil || errObj["code"] != c.wantMC {
				t.Errorf("code = %v, want %s", errObj["code"], c.wantMC)
			}
		})
	}

	// ErrProjectNotFound exercised by setting fakeProjectRepo.findResult so the
	// repo returns nil — service then maps to ErrProjectNotFound.
	t.Run("ErrProjectNotFound", func(t *testing.T) {
		fr := &fakeProjectRepo{findReturnsNil: true}
		app := buildTestApp(t, tc, fr, nil)
		id, _ := ids.New()
		status, body, _ := doJSON(t, app, "GET", "/api/v1/workspaces/projects/"+id.String(), nil)
		if status != http.StatusNotFound {
			t.Fatalf("status = %d, want 404; body = %v", status, body)
		}
		errObj := body["error"].(map[string]any)
		if errObj["code"] != "project.not_found" {
			t.Errorf("code = %v, want project.not_found", errObj["code"])
		}
	})
}

// ── DELETE 204 ────────────────────────────────────────────────────────────────

func TestProjectHandler_Delete_204_EmptyBody(t *testing.T) {
	tc := newTC(workspace.OrgRoleOwner)
	app := buildTestApp(t, tc, nil, nil)

	id, _ := ids.New()
	resp := doRaw(t, app, "DELETE", "/api/v1/workspaces/projects/"+id.String(), nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	raw, _ := io.ReadAll(resp.Body)
	if len(raw) != 0 {
		t.Errorf("body bytes = %d, want 0; body = %q", len(raw), string(raw))
	}
	// fiber may not always set Content-Length explicitly for empty bodies;
	// the byte-length 0 check above is the authoritative check.
}

// ── Update body does NOT carry status code ────────────────────────────────────

func TestProjectHandler_Update_NoProjectStatusCodeField(t *testing.T) {
	tc := newTC(workspace.OrgRoleOwner)
	fr := &fakeProjectRepo{}
	app := buildTestApp(t, tc, fr, nil)

	id, _ := ids.New()
	body := map[string]any{
		"project_name":        "Renamed",
		"slug":                "renamed",
		"project_type_code":   "client",
		"project_status_code": "active", // should be silently ignored
	}
	status, respBody, _ := doJSON(t, app, "PUT", "/api/v1/workspaces/projects/"+id.String(), body)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %v", status, respBody)
	}
	// Service constructs the update by carrying existing.ProjectStatusCode unchanged.
	// Our fake FindByIDForWorkspace returns ProjectStatusCode='planning'.
	// The returned project should still have ProjectStatusCode='planning' (not 'active').
	data := respBody["data"].(map[string]any)
	if data["project_status_code"] != "planning" {
		t.Errorf("project_status_code = %v, want planning (body's value must be ignored)", data["project_status_code"])
	}
}

// ── helper to demonstrate generic interface compliance ────────────────────────

var _ project.ProjectRepository = (*fakeProjectRepo)(nil)
var _ project.MasterRepository = (*fakeMasterRepo)(nil)
