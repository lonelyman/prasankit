package projecthttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/auth/authsvc"
	"prasankit-api/internal/modules/project"
	"prasankit-api/internal/modules/project/projectsvc"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/internal/modules/workspace/workspacesvc"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type fakeProjectService struct {
	createResult *projectsvc.CreateProjectResult
	createErr    error
	createInput  projectsvc.CreateProjectInput
	listResult   *projectsvc.ListProjectsResult
	listErr      error
	listInput    projectsvc.ListProjectsInput
	getResult    *projectsvc.GetProjectResult
	getErr       error
	getInput     projectsvc.GetProjectInput
	updateResult *projectsvc.UpdateProjectResult
	updateErr    error
	updateInput  projectsvc.UpdateProjectInput
}

func (s *fakeProjectService) CreateProject(_ context.Context, input projectsvc.CreateProjectInput) (*projectsvc.CreateProjectResult, error) {
	s.createInput = input
	if s.createErr != nil {
		return nil, s.createErr
	}
	return s.createResult, nil
}

func (s *fakeProjectService) ListProjects(_ context.Context, input projectsvc.ListProjectsInput) (*projectsvc.ListProjectsResult, error) {
	s.listInput = input
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.listResult, nil
}

func (s *fakeProjectService) GetProject(_ context.Context, input projectsvc.GetProjectInput) (*projectsvc.GetProjectResult, error) {
	s.getInput = input
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.getResult, nil
}

func (s *fakeProjectService) UpdateProject(_ context.Context, input projectsvc.UpdateProjectInput) (*projectsvc.UpdateProjectResult, error) {
	s.updateInput = input
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	return s.updateResult, nil
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

type fakeTenantResolver struct {
	result *workspacesvc.ResolveTenantContextResult
	err    error
	input  workspacesvc.ResolveTenantContextInput
}

func (r *fakeTenantResolver) ResolveTenantContext(_ context.Context, input workspacesvc.ResolveTenantContextInput) (*workspacesvc.ResolveTenantContextResult, error) {
	r.input = input
	if r.err != nil {
		return nil, r.err
	}
	return r.result, nil
}

func TestCreateProject(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	memberID := uuid.Must(uuid.NewV7())
	service := &fakeProjectService{
		createResult: &projectsvc.CreateProjectResult{
			Project: project.Project{
				ID:          projectID,
				TenantID:    tenantContext.TenantID,
				WorkspaceID: tenantContext.WorkspaceID,
				Code:        "PRJ-2026-0001",
				Name:        "Project A",
				Type:        project.ProjectTypeClient,
				Status:      project.ProjectStatusDraft,
				Priority:    project.ProjectPriorityMedium,
			},
			Member: project.Member{
				ID:        memberID,
				ProjectID: projectID,
				Role:      project.ProjectRoleOwner,
				Status:    project.ProjectMemberStatusActive,
			},
		},
	}
	session := &fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: accountID, Status: auth.UserAccountStatusActive}}}
	tenant := &fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: tenantContext}}
	app := newProjectTestApp(newTestHandler(service, session, tenant))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspace/projects", bytes.NewBufferString(`{"name":"Project A","type":"client"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Workspace-Slug", "team-one")
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
	if tenant.input.WorkspaceSlug != "team-one" {
		t.Fatalf("workspace slug = %s, want team-one", tenant.input.WorkspaceSlug)
	}
	if service.createInput.Account.ID != accountID {
		t.Fatalf("account id = %s, want %s", service.createInput.Account.ID, accountID)
	}
	if service.createInput.TenantContext.TenantID != tenantContext.TenantID {
		t.Fatalf("tenant id = %s, want %s", service.createInput.TenantContext.TenantID, tenantContext.TenantID)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data := body["data"].(map[string]any)
	projectData := data["project"].(map[string]any)
	if _, ok := projectData["tenant_id"]; ok {
		t.Fatalf("response must not expose tenant_id: %#v", projectData)
	}
	if projectData["code"] != "PRJ-2026-0001" {
		t.Fatalf("project.code = %v, want PRJ-2026-0001", projectData["code"])
	}
}

func TestCreateProjectRequiresManagePermission(t *testing.T) {
	app := newProjectTestApp(newTestHandler(
		&fakeProjectService{},
		&fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}}},
		&fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: testTenantContext(workspace.WorkspaceRoleUser)}},
	))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspace/projects", bytes.NewBufferString(`{"name":"Project A"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertProjectError(t, resp, http.StatusForbidden, "PERMISSION_DENIED")
}

func TestListProjects(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleUser)
	projectID := uuid.Must(uuid.NewV7())
	service := &fakeProjectService{
		listResult: &projectsvc.ListProjectsResult{
			Total: 1,
			Items: []project.ProjectWithMember{
				{
					Project: project.Project{
						ID:          projectID,
						TenantID:    tenantContext.TenantID,
						WorkspaceID: tenantContext.WorkspaceID,
						Code:        "PRJ-2026-0001",
						Name:        "Project A",
						Type:        project.ProjectTypeInternal,
						Status:      project.ProjectStatusDraft,
						Priority:    project.ProjectPriorityMedium,
					},
				},
			},
		},
	}
	session := &fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: accountID, Status: auth.UserAccountStatusActive}}}
	tenant := &fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: tenantContext}}
	app := newProjectTestApp(newTestHandler(service, session, tenant))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/projects?page=1&limit=10", nil)
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if service.listInput.Limit != 10 || service.listInput.Offset != 0 {
		t.Fatalf("pagination = limit %d offset %d, want 10/0", service.listInput.Limit, service.listInput.Offset)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data := body["data"].(map[string]any)
	items := data["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items len = %d, want 1", len(items))
	}
}

func TestGetProject(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleUser)
	projectID := uuid.Must(uuid.NewV7())
	service := &fakeProjectService{
		getResult: &projectsvc.GetProjectResult{
			Item: project.ProjectWithMember{
				Project: project.Project{
					ID:          projectID,
					TenantID:    tenantContext.TenantID,
					WorkspaceID: tenantContext.WorkspaceID,
					Code:        "PRJ-2026-0001",
					Name:        "Project A",
					Type:        project.ProjectTypeInternal,
					Status:      project.ProjectStatusDraft,
					Priority:    project.ProjectPriorityMedium,
				},
				Member: project.Member{
					ID:        uuid.Must(uuid.NewV7()),
					ProjectID: projectID,
					Role:      project.ProjectRoleOwner,
					Status:    project.ProjectMemberStatusActive,
				},
			},
		},
	}
	session := &fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: accountID, Status: auth.UserAccountStatusActive}}}
	tenant := &fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: tenantContext}}
	app := newProjectTestApp(newTestHandler(service, session, tenant))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/projects/"+projectID.String(), nil)
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if service.getInput.ProjectID != projectID {
		t.Fatalf("project ID = %s, want %s", service.getInput.ProjectID, projectID)
	}
	if service.getInput.TenantContext.TenantID != tenantContext.TenantID {
		t.Fatalf("tenant ID = %s, want %s", service.getInput.TenantContext.TenantID, tenantContext.TenantID)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data := body["data"].(map[string]any)
	projectData := data["project"].(map[string]any)
	if _, ok := projectData["tenant_id"]; ok {
		t.Fatalf("response must not expose tenant_id: %#v", projectData)
	}
	if projectData["id"] != projectID.String() {
		t.Fatalf("project.id = %v, want %s", projectData["id"], projectID)
	}
}

func TestGetProjectRejectsInvalidID(t *testing.T) {
	app := newProjectTestApp(newTestHandler(
		&fakeProjectService{},
		&fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}}},
		&fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: testTenantContext(workspace.WorkspaceRoleUser)}},
	))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/projects/not-a-uuid", nil)
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertProjectError(t, resp, http.StatusBadRequest, "PROJECT_ID_INVALID")
}

func TestGetProjectMapsNotFound(t *testing.T) {
	app := newProjectTestApp(newTestHandler(
		&fakeProjectService{getErr: projectsvc.ErrProjectNotFound},
		&fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}}},
		&fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: testTenantContext(workspace.WorkspaceRoleUser)}},
	))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/projects/"+uuid.Must(uuid.NewV7()).String(), nil)
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertProjectError(t, resp, http.StatusNotFound, "PROJECT_NOT_FOUND")
}

func TestUpdateProject(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	service := &fakeProjectService{
		updateResult: &projectsvc.UpdateProjectResult{
			Item: project.ProjectWithMember{
				Project: project.Project{
					ID:          projectID,
					TenantID:    tenantContext.TenantID,
					WorkspaceID: tenantContext.WorkspaceID,
					Code:        "PRJ-2026-0001",
					Name:        "Project B",
					Type:        project.ProjectTypeClient,
					Status:      project.ProjectStatusDraft,
					Priority:    project.ProjectPriorityHigh,
					Description: "",
				},
				Member: project.Member{
					ID:        uuid.Must(uuid.NewV7()),
					ProjectID: projectID,
					Role:      project.ProjectRoleOwner,
					Status:    project.ProjectMemberStatusActive,
				},
			},
		},
	}
	session := &fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: accountID, Status: auth.UserAccountStatusActive}}}
	tenant := &fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: tenantContext}}
	app := newProjectTestApp(newTestHandler(service, session, tenant))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspace/projects/"+projectID.String(), bytes.NewBufferString(`{"name":"Project B","type":"client","priority":"high","description":""}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if service.updateInput.ProjectID != projectID {
		t.Fatalf("project ID = %s, want %s", service.updateInput.ProjectID, projectID)
	}
	if service.updateInput.TenantContext.TenantID != tenantContext.TenantID {
		t.Fatalf("tenant ID = %s, want %s", service.updateInput.TenantContext.TenantID, tenantContext.TenantID)
	}
	if service.updateInput.Name == nil || *service.updateInput.Name != "Project B" {
		t.Fatalf("name = %#v, want Project B", service.updateInput.Name)
	}
	if service.updateInput.Description == nil || *service.updateInput.Description != "" {
		t.Fatalf("description = %#v, want empty string", service.updateInput.Description)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data := body["data"].(map[string]any)
	projectData := data["project"].(map[string]any)
	if _, ok := projectData["tenant_id"]; ok {
		t.Fatalf("response must not expose tenant_id: %#v", projectData)
	}
	if projectData["name"] != "Project B" {
		t.Fatalf("project.name = %v, want Project B", projectData["name"])
	}
}

func TestUpdateProjectRequiresManagePermission(t *testing.T) {
	app := newProjectTestApp(newTestHandler(
		&fakeProjectService{},
		&fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}}},
		&fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: testTenantContext(workspace.WorkspaceRoleUser)}},
	))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspace/projects/"+uuid.Must(uuid.NewV7()).String(), bytes.NewBufferString(`{"name":"Project B"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertProjectError(t, resp, http.StatusForbidden, "PERMISSION_DENIED")
}

func TestUpdateProjectRejectsInvalidID(t *testing.T) {
	app := newProjectTestApp(newTestHandler(
		&fakeProjectService{},
		&fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}}},
		&fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: testTenantContext(workspace.WorkspaceRoleOwner)}},
	))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspace/projects/not-a-uuid", bytes.NewBufferString(`{"name":"Project B"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertProjectError(t, resp, http.StatusBadRequest, "PROJECT_ID_INVALID")
}

func TestUpdateProjectMapsValidationError(t *testing.T) {
	app := newProjectTestApp(newTestHandler(
		&fakeProjectService{updateErr: projectsvc.ErrProjectPriorityInvalid},
		&fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}}},
		&fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: testTenantContext(workspace.WorkspaceRoleOwner)}},
	))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspace/projects/"+uuid.Must(uuid.NewV7()).String(), bytes.NewBufferString(`{"priority":"urgent"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertProjectError(t, resp, http.StatusBadRequest, "PROJECT_PRIORITY_INVALID")
}

func newTestHandler(projectService ProjectService, sessionService SessionService, tenantResolver TenantResolver) Handler {
	return NewHandler(projectService, sessionService, tenantResolver, CookieConfig{
		Name:     "prasankit_session",
		TTL:      24 * time.Hour,
		SameSite: "Lax",
	})
}

func newProjectTestApp(handler Handler) *fiber.App {
	app := fiber.New()
	api := app.Group("/api/v1")
	handler.RegisterRoutes(api)
	return app
}

func testTenantContext(role workspace.WorkspaceRole) workspace.TenantContext {
	return workspace.TenantContext{
		TenantID:      uuid.Must(uuid.NewV7()),
		WorkspaceID:   uuid.Must(uuid.NewV7()),
		WorkspaceSlug: "team-one",
		MembershipID:  uuid.Must(uuid.NewV7()),
		Role:          role,
	}
}

func assertProjectError(t *testing.T, resp *http.Response, wantStatus int, wantCode string) {
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
		t.Fatalf("error envelope missing: %#v", body)
	}
	if gotError["code"] != wantCode {
		t.Fatalf("error.code = %v, want %s", gotError["code"], wantCode)
	}
}
