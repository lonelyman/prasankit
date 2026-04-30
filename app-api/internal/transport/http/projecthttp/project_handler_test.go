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
	createResult           *projectsvc.CreateProjectResult
	createErr              error
	createInput            projectsvc.CreateProjectInput
	listResult             *projectsvc.ListProjectsResult
	listErr                error
	listInput              projectsvc.ListProjectsInput
	getResult              *projectsvc.GetProjectResult
	getErr                 error
	getInput               projectsvc.GetProjectInput
	updateResult           *projectsvc.UpdateProjectResult
	updateErr              error
	updateInput            projectsvc.UpdateProjectInput
	membersResult          *projectsvc.ListProjectMembersResult
	membersErr             error
	membersInput           projectsvc.ListProjectMembersInput
	addMemberResult        *projectsvc.AddProjectMemberResult
	addMemberErr           error
	addMemberInput         projectsvc.AddProjectMemberInput
	updateMemberResult     *projectsvc.UpdateProjectMemberResult
	updateMemberErr        error
	updateMemberInput      projectsvc.UpdateProjectMemberInput
	removeMemberErr        error
	removeMemberInput      projectsvc.RemoveProjectMemberInput
	positionsResult        *projectsvc.ListProjectPositionsResult
	positionsErr           error
	positionsInput         projectsvc.ListProjectPositionsInput
	replacePositionsResult *projectsvc.ReplaceProjectMemberPositionsResult
	replacePositionsErr    error
	replacePositionsInput  projectsvc.ReplaceProjectMemberPositionsInput
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

func (s *fakeProjectService) ListProjectMembers(_ context.Context, input projectsvc.ListProjectMembersInput) (*projectsvc.ListProjectMembersResult, error) {
	s.membersInput = input
	if s.membersErr != nil {
		return nil, s.membersErr
	}
	return s.membersResult, nil
}

func (s *fakeProjectService) AddProjectMember(_ context.Context, input projectsvc.AddProjectMemberInput) (*projectsvc.AddProjectMemberResult, error) {
	s.addMemberInput = input
	if s.addMemberErr != nil {
		return nil, s.addMemberErr
	}
	return s.addMemberResult, nil
}

func (s *fakeProjectService) UpdateProjectMember(_ context.Context, input projectsvc.UpdateProjectMemberInput) (*projectsvc.UpdateProjectMemberResult, error) {
	s.updateMemberInput = input
	if s.updateMemberErr != nil {
		return nil, s.updateMemberErr
	}
	return s.updateMemberResult, nil
}

func (s *fakeProjectService) RemoveProjectMember(_ context.Context, input projectsvc.RemoveProjectMemberInput) error {
	s.removeMemberInput = input
	return s.removeMemberErr
}

func (s *fakeProjectService) ListProjectPositions(_ context.Context, input projectsvc.ListProjectPositionsInput) (*projectsvc.ListProjectPositionsResult, error) {
	s.positionsInput = input
	if s.positionsErr != nil {
		return nil, s.positionsErr
	}
	return s.positionsResult, nil
}

func (s *fakeProjectService) ReplaceProjectMemberPositions(_ context.Context, input projectsvc.ReplaceProjectMemberPositionsInput) (*projectsvc.ReplaceProjectMemberPositionsResult, error) {
	s.replacePositionsInput = input
	if s.replacePositionsErr != nil {
		return nil, s.replacePositionsErr
	}
	return s.replacePositionsResult, nil
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

func TestListProjectMembers(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleUser)
	projectID := uuid.Must(uuid.NewV7())
	memberID := uuid.Must(uuid.NewV7())
	userAccountID := uuid.Must(uuid.NewV7())
	service := &fakeProjectService{
		membersResult: &projectsvc.ListProjectMembersResult{
			Total: 1,
			Items: []project.Member{
				{
					ID:                    memberID,
					TenantID:              tenantContext.TenantID,
					WorkspaceID:           tenantContext.WorkspaceID,
					ProjectID:             projectID,
					WorkspaceMembershipID: tenantContext.MembershipID,
					UserAccountID:         &userAccountID,
					Role:                  project.ProjectRoleOwner,
					Status:                project.ProjectMemberStatusActive,
				},
			},
		},
	}
	session := &fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: accountID, Status: auth.UserAccountStatusActive}}}
	tenant := &fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: tenantContext}}
	app := newProjectTestApp(newTestHandler(service, session, tenant))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/projects/"+projectID.String()+"/members?page=1&limit=10", nil)
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
	if service.membersInput.ProjectID != projectID {
		t.Fatalf("project ID = %s, want %s", service.membersInput.ProjectID, projectID)
	}
	if service.membersInput.TenantContext.TenantID != tenantContext.TenantID {
		t.Fatalf("tenant ID = %s, want %s", service.membersInput.TenantContext.TenantID, tenantContext.TenantID)
	}
	if service.membersInput.Limit != 10 || service.membersInput.Offset != 0 {
		t.Fatalf("pagination = limit %d offset %d, want 10/0", service.membersInput.Limit, service.membersInput.Offset)
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
	memberData := items[0].(map[string]any)
	if _, ok := memberData["tenant_id"]; ok {
		t.Fatalf("response must not expose tenant_id: %#v", memberData)
	}
	if memberData["id"] != memberID.String() {
		t.Fatalf("member.id = %v, want %s", memberData["id"], memberID)
	}
	if memberData["workspace_membership_id"] != tenantContext.MembershipID.String() {
		t.Fatalf("workspace_membership_id = %v, want %s", memberData["workspace_membership_id"], tenantContext.MembershipID)
	}
}

func TestListProjectMembersRejectsInvalidID(t *testing.T) {
	app := newProjectTestApp(newTestHandler(
		&fakeProjectService{},
		&fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}}},
		&fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: testTenantContext(workspace.WorkspaceRoleUser)}},
	))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/projects/not-a-uuid/members", nil)
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertProjectError(t, resp, http.StatusBadRequest, "PROJECT_ID_INVALID")
}

func TestAddProjectMember(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	workspaceMembershipID := uuid.Must(uuid.NewV7())
	memberID := uuid.Must(uuid.NewV7())
	userAccountID := uuid.Must(uuid.NewV7())
	service := &fakeProjectService{
		addMemberResult: &projectsvc.AddProjectMemberResult{
			Member: project.Member{
				ID:                    memberID,
				TenantID:              tenantContext.TenantID,
				WorkspaceID:           tenantContext.WorkspaceID,
				ProjectID:             projectID,
				WorkspaceMembershipID: workspaceMembershipID,
				UserAccountID:         &userAccountID,
				Role:                  project.ProjectRoleManager,
				Status:                project.ProjectMemberStatusActive,
			},
		},
	}
	session := &fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: accountID, Status: auth.UserAccountStatusActive}}}
	tenant := &fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: tenantContext}}
	app := newProjectTestApp(newTestHandler(service, session, tenant))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspace/projects/"+projectID.String()+"/members", bytes.NewBufferString(`{"workspace_membership_id":"`+workspaceMembershipID.String()+`","role":"project_manager"}`))
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
	if service.addMemberInput.ProjectID != projectID {
		t.Fatalf("project ID = %s, want %s", service.addMemberInput.ProjectID, projectID)
	}
	if service.addMemberInput.WorkspaceMembershipID != workspaceMembershipID {
		t.Fatalf("workspace membership ID = %s, want %s", service.addMemberInput.WorkspaceMembershipID, workspaceMembershipID)
	}
	if service.addMemberInput.Role != project.ProjectRoleManager {
		t.Fatalf("role = %s, want project_manager", service.addMemberInput.Role)
	}
	if service.addMemberInput.TenantContext.TenantID != tenantContext.TenantID {
		t.Fatalf("tenant ID = %s, want %s", service.addMemberInput.TenantContext.TenantID, tenantContext.TenantID)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data := body["data"].(map[string]any)
	if _, ok := data["tenant_id"]; ok {
		t.Fatalf("response must not expose tenant_id: %#v", data)
	}
	if data["workspace_membership_id"] != workspaceMembershipID.String() {
		t.Fatalf("workspace_membership_id = %v, want %s", data["workspace_membership_id"], workspaceMembershipID)
	}
}

func TestAddProjectMemberRequiresManagePermission(t *testing.T) {
	app := newProjectTestApp(newTestHandler(
		&fakeProjectService{},
		&fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}}},
		&fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: testTenantContext(workspace.WorkspaceRoleUser)}},
	))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspace/projects/"+uuid.Must(uuid.NewV7()).String()+"/members", bytes.NewBufferString(`{"workspace_membership_id":"`+uuid.Must(uuid.NewV7()).String()+`"}`))
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

func TestAddProjectMemberRejectsInvalidIDs(t *testing.T) {
	app := newProjectTestApp(newTestHandler(
		&fakeProjectService{},
		&fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}}},
		&fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: testTenantContext(workspace.WorkspaceRoleOwner)}},
	))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspace/projects/not-a-uuid/members", bytes.NewBufferString(`{"workspace_membership_id":"`+uuid.Must(uuid.NewV7()).String()+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertProjectError(t, resp, http.StatusBadRequest, "PROJECT_ID_INVALID")

	req = httptest.NewRequest(http.MethodPost, "/api/v1/workspace/projects/"+uuid.Must(uuid.NewV7()).String()+"/members", bytes.NewBufferString(`{"workspace_membership_id":"not-a-uuid"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertProjectError(t, resp, http.StatusBadRequest, "WORKSPACE_MEMBERSHIP_ID_INVALID")
}

func TestAddProjectMemberMapsErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "workspace membership not found",
			err:        projectsvc.ErrWorkspaceMembershipNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "WORKSPACE_MEMBERSHIP_NOT_FOUND",
		},
		{
			name:       "already exists",
			err:        projectsvc.ErrProjectMemberAlreadyExists,
			wantStatus: http.StatusConflict,
			wantCode:   "PROJECT_MEMBER_ALREADY_EXISTS",
		},
		{
			name:       "role invalid",
			err:        projectsvc.ErrProjectRoleInvalid,
			wantStatus: http.StatusBadRequest,
			wantCode:   "PROJECT_ROLE_INVALID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newProjectTestApp(newTestHandler(
				&fakeProjectService{addMemberErr: tt.err},
				&fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}}},
				&fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: testTenantContext(workspace.WorkspaceRoleOwner)}},
			))

			req := httptest.NewRequest(http.MethodPost, "/api/v1/workspace/projects/"+uuid.Must(uuid.NewV7()).String()+"/members", bytes.NewBufferString(`{"workspace_membership_id":"`+uuid.Must(uuid.NewV7()).String()+`","role":"member"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Workspace-Slug", "team-one")
			req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer resp.Body.Close()

			assertProjectError(t, resp, tt.wantStatus, tt.wantCode)
		})
	}
}

func TestUpdateProjectMember(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	memberID := uuid.Must(uuid.NewV7())
	workspaceMembershipID := uuid.Must(uuid.NewV7())
	service := &fakeProjectService{
		updateMemberResult: &projectsvc.UpdateProjectMemberResult{
			Member: project.Member{
				ID:                    memberID,
				TenantID:              tenantContext.TenantID,
				WorkspaceID:           tenantContext.WorkspaceID,
				ProjectID:             projectID,
				WorkspaceMembershipID: workspaceMembershipID,
				Role:                  project.ProjectRoleManager,
				Status:                project.ProjectMemberStatusActive,
			},
		},
	}
	session := &fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: accountID, Status: auth.UserAccountStatusActive}}}
	tenant := &fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: tenantContext}}
	app := newProjectTestApp(newTestHandler(service, session, tenant))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspace/projects/"+projectID.String()+"/members/"+memberID.String(), bytes.NewBufferString(`{"role":"project_manager"}`))
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
	if service.updateMemberInput.ProjectID != projectID {
		t.Fatalf("project ID = %s, want %s", service.updateMemberInput.ProjectID, projectID)
	}
	if service.updateMemberInput.MemberID != memberID {
		t.Fatalf("member ID = %s, want %s", service.updateMemberInput.MemberID, memberID)
	}
	if service.updateMemberInput.Role == nil || *service.updateMemberInput.Role != project.ProjectRoleManager {
		t.Fatalf("role = %#v, want project_manager", service.updateMemberInput.Role)
	}
	if service.updateMemberInput.TenantContext.TenantID != tenantContext.TenantID {
		t.Fatalf("tenant ID = %s, want %s", service.updateMemberInput.TenantContext.TenantID, tenantContext.TenantID)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data := body["data"].(map[string]any)
	if _, ok := data["tenant_id"]; ok {
		t.Fatalf("response must not expose tenant_id: %#v", data)
	}
	if data["role"] != "project_manager" {
		t.Fatalf("role = %v, want project_manager", data["role"])
	}
}

func TestUpdateProjectMemberRequiresManagePermission(t *testing.T) {
	app := newProjectTestApp(newTestHandler(
		&fakeProjectService{},
		&fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}}},
		&fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: testTenantContext(workspace.WorkspaceRoleUser)}},
	))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspace/projects/"+uuid.Must(uuid.NewV7()).String()+"/members/"+uuid.Must(uuid.NewV7()).String(), bytes.NewBufferString(`{"role":"member"}`))
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

func TestUpdateProjectMemberRejectsInvalidIDs(t *testing.T) {
	app := newProjectTestApp(newTestHandler(
		&fakeProjectService{},
		&fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}}},
		&fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: testTenantContext(workspace.WorkspaceRoleOwner)}},
	))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspace/projects/not-a-uuid/members/"+uuid.Must(uuid.NewV7()).String(), bytes.NewBufferString(`{"role":"member"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertProjectError(t, resp, http.StatusBadRequest, "PROJECT_ID_INVALID")

	req = httptest.NewRequest(http.MethodPatch, "/api/v1/workspace/projects/"+uuid.Must(uuid.NewV7()).String()+"/members/not-a-uuid", bytes.NewBufferString(`{"role":"member"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertProjectError(t, resp, http.StatusBadRequest, "PROJECT_MEMBER_ID_INVALID")
}

func TestUpdateProjectMemberMapsErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "no fields",
			err:        projectsvc.ErrProjectMemberUpdateNoFields,
			wantStatus: http.StatusBadRequest,
			wantCode:   "PROJECT_MEMBER_UPDATE_NO_FIELDS",
		},
		{
			name:       "role invalid",
			err:        projectsvc.ErrProjectRoleInvalid,
			wantStatus: http.StatusBadRequest,
			wantCode:   "PROJECT_ROLE_INVALID",
		},
		{
			name:       "member not found",
			err:        projectsvc.ErrProjectMemberNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "PROJECT_MEMBER_NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newProjectTestApp(newTestHandler(
				&fakeProjectService{updateMemberErr: tt.err},
				&fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}}},
				&fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: testTenantContext(workspace.WorkspaceRoleOwner)}},
			))

			req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspace/projects/"+uuid.Must(uuid.NewV7()).String()+"/members/"+uuid.Must(uuid.NewV7()).String(), bytes.NewBufferString(`{"role":"member"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Workspace-Slug", "team-one")
			req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer resp.Body.Close()

			assertProjectError(t, resp, tt.wantStatus, tt.wantCode)
		})
	}
}

func TestRemoveProjectMember(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	memberID := uuid.Must(uuid.NewV7())
	service := &fakeProjectService{}
	session := &fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: accountID, Status: auth.UserAccountStatusActive}}}
	tenant := &fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: tenantContext}}
	app := newProjectTestApp(newTestHandler(service, session, tenant))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspace/projects/"+projectID.String()+"/members/"+memberID.String(), nil)
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status code = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
	if service.removeMemberInput.ProjectID != projectID {
		t.Fatalf("project ID = %s, want %s", service.removeMemberInput.ProjectID, projectID)
	}
	if service.removeMemberInput.MemberID != memberID {
		t.Fatalf("member ID = %s, want %s", service.removeMemberInput.MemberID, memberID)
	}
	if service.removeMemberInput.TenantContext.TenantID != tenantContext.TenantID {
		t.Fatalf("tenant ID = %s, want %s", service.removeMemberInput.TenantContext.TenantID, tenantContext.TenantID)
	}
	if service.removeMemberInput.Account.ID != accountID {
		t.Fatalf("account ID = %s, want %s", service.removeMemberInput.Account.ID, accountID)
	}
}

func TestRemoveProjectMemberRequiresManagePermission(t *testing.T) {
	app := newProjectTestApp(newTestHandler(
		&fakeProjectService{},
		&fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}}},
		&fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: testTenantContext(workspace.WorkspaceRoleUser)}},
	))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspace/projects/"+uuid.Must(uuid.NewV7()).String()+"/members/"+uuid.Must(uuid.NewV7()).String(), nil)
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertProjectError(t, resp, http.StatusForbidden, "PERMISSION_DENIED")
}

func TestRemoveProjectMemberRejectsInvalidIDs(t *testing.T) {
	app := newProjectTestApp(newTestHandler(
		&fakeProjectService{},
		&fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}}},
		&fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: testTenantContext(workspace.WorkspaceRoleOwner)}},
	))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspace/projects/not-a-uuid/members/"+uuid.Must(uuid.NewV7()).String(), nil)
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertProjectError(t, resp, http.StatusBadRequest, "PROJECT_ID_INVALID")

	req = httptest.NewRequest(http.MethodDelete, "/api/v1/workspace/projects/"+uuid.Must(uuid.NewV7()).String()+"/members/not-a-uuid", nil)
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertProjectError(t, resp, http.StatusBadRequest, "PROJECT_MEMBER_ID_INVALID")
}

func TestRemoveProjectMemberMapsErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "member not found",
			err:        projectsvc.ErrProjectMemberNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "PROJECT_MEMBER_NOT_FOUND",
		},
		{
			name:       "last owner",
			err:        projectsvc.ErrProjectMemberLastOwner,
			wantStatus: http.StatusConflict,
			wantCode:   "PROJECT_MEMBER_LAST_OWNER",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newProjectTestApp(newTestHandler(
				&fakeProjectService{removeMemberErr: tt.err},
				&fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}}},
				&fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: testTenantContext(workspace.WorkspaceRoleOwner)}},
			))

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspace/projects/"+uuid.Must(uuid.NewV7()).String()+"/members/"+uuid.Must(uuid.NewV7()).String(), nil)
			req.Header.Set("X-Workspace-Slug", "team-one")
			req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer resp.Body.Close()

			assertProjectError(t, resp, tt.wantStatus, tt.wantCode)
		})
	}
}

func TestListProjectPositions(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleUser)
	projectID := uuid.Must(uuid.NewV7())
	positionID := uuid.Must(uuid.NewV7())
	service := &fakeProjectService{
		positionsResult: &projectsvc.ListProjectPositionsResult{
			Items: []project.PositionMaster{
				{ID: positionID, Code: project.ProjectPositionDeveloper, Name: "Developer"},
			},
		},
	}
	session := &fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: accountID, Status: auth.UserAccountStatusActive}}}
	tenant := &fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: tenantContext}}
	app := newProjectTestApp(newTestHandler(service, session, tenant))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/projects/"+projectID.String()+"/positions", nil)
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
	if service.positionsInput.ProjectID != projectID {
		t.Fatalf("project ID = %s, want %s", service.positionsInput.ProjectID, projectID)
	}
	if service.positionsInput.TenantContext.TenantID != tenantContext.TenantID {
		t.Fatalf("tenant ID = %s, want %s", service.positionsInput.TenantContext.TenantID, tenantContext.TenantID)
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
	item := items[0].(map[string]any)
	if item["code"] != "developer" {
		t.Fatalf("code = %v, want developer", item["code"])
	}
}

func TestReplaceProjectMemberPositions(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	memberID := uuid.Must(uuid.NewV7())
	positionID := uuid.Must(uuid.NewV7())
	service := &fakeProjectService{
		replacePositionsResult: &projectsvc.ReplaceProjectMemberPositionsResult{
			Items: []project.MemberPosition{
				{
					ID:              uuid.Must(uuid.NewV7()),
					TenantID:        tenantContext.TenantID,
					ProjectID:       projectID,
					ProjectMemberID: memberID,
					PositionID:      positionID,
					Position:        project.ProjectPositionDeveloper,
					Name:            "Developer",
				},
			},
		},
	}
	session := &fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: accountID, Status: auth.UserAccountStatusActive}}}
	tenant := &fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: tenantContext}}
	app := newProjectTestApp(newTestHandler(service, session, tenant))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/workspace/projects/"+projectID.String()+"/members/"+memberID.String()+"/positions", bytes.NewBufferString(`{"position_codes":["developer"]}`))
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
	if service.replacePositionsInput.ProjectID != projectID {
		t.Fatalf("project ID = %s, want %s", service.replacePositionsInput.ProjectID, projectID)
	}
	if service.replacePositionsInput.MemberID != memberID {
		t.Fatalf("member ID = %s, want %s", service.replacePositionsInput.MemberID, memberID)
	}
	if len(service.replacePositionsInput.PositionCodes) != 1 || service.replacePositionsInput.PositionCodes[0] != project.ProjectPositionDeveloper {
		t.Fatalf("position codes = %#v, want developer", service.replacePositionsInput.PositionCodes)
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
	item := items[0].(map[string]any)
	if item["code"] != "developer" {
		t.Fatalf("code = %v, want developer", item["code"])
	}
	if _, ok := item["tenant_id"]; ok {
		t.Fatalf("response must not expose tenant_id: %#v", item)
	}
}

func TestReplaceProjectMemberPositionsRequiresManagePermission(t *testing.T) {
	app := newProjectTestApp(newTestHandler(
		&fakeProjectService{},
		&fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}}},
		&fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: testTenantContext(workspace.WorkspaceRoleUser)}},
	))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/workspace/projects/"+uuid.Must(uuid.NewV7()).String()+"/members/"+uuid.Must(uuid.NewV7()).String()+"/positions", bytes.NewBufferString(`{"position_codes":["developer"]}`))
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

func TestReplaceProjectMemberPositionsRejectsInvalidIDs(t *testing.T) {
	app := newProjectTestApp(newTestHandler(
		&fakeProjectService{},
		&fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}}},
		&fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: testTenantContext(workspace.WorkspaceRoleOwner)}},
	))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/workspace/projects/not-a-uuid/members/"+uuid.Must(uuid.NewV7()).String()+"/positions", bytes.NewBufferString(`{"position_codes":[]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertProjectError(t, resp, http.StatusBadRequest, "PROJECT_ID_INVALID")

	req = httptest.NewRequest(http.MethodPut, "/api/v1/workspace/projects/"+uuid.Must(uuid.NewV7()).String()+"/members/not-a-uuid/positions", bytes.NewBufferString(`{"position_codes":[]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertProjectError(t, resp, http.StatusBadRequest, "PROJECT_MEMBER_ID_INVALID")
}

func TestReplaceProjectMemberPositionsMapsErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "codes missing",
			err:        projectsvc.ErrProjectPositionCodesRequired,
			wantStatus: http.StatusBadRequest,
			wantCode:   "PROJECT_POSITION_CODES_REQUIRED",
		},
		{
			name:       "position invalid",
			err:        projectsvc.ErrProjectPositionInvalid,
			wantStatus: http.StatusBadRequest,
			wantCode:   "PROJECT_POSITION_INVALID",
		},
		{
			name:       "member not found",
			err:        projectsvc.ErrProjectMemberNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "PROJECT_MEMBER_NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newProjectTestApp(newTestHandler(
				&fakeProjectService{replacePositionsErr: tt.err},
				&fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}}},
				&fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: testTenantContext(workspace.WorkspaceRoleOwner)}},
			))

			req := httptest.NewRequest(http.MethodPut, "/api/v1/workspace/projects/"+uuid.Must(uuid.NewV7()).String()+"/members/"+uuid.Must(uuid.NewV7()).String()+"/positions", bytes.NewBufferString(`{"position_codes":["developer"]}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Workspace-Slug", "team-one")
			req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer resp.Body.Close()

			assertProjectError(t, resp, tt.wantStatus, tt.wantCode)
		})
	}
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
