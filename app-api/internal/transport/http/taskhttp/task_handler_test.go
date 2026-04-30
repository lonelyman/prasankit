package taskhttp

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
	"prasankit-api/internal/modules/task"
	"prasankit-api/internal/modules/task/tasksvc"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/internal/modules/workspace/workspacesvc"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type fakeTaskService struct {
	createResult *tasksvc.CreateTaskResult
	createErr    error
	createInput  tasksvc.CreateTaskInput
	listResult   *tasksvc.ListTasksResult
	listErr      error
	listInput    tasksvc.ListTasksInput
	getResult    *tasksvc.GetTaskResult
	getErr       error
	getInput     tasksvc.GetTaskInput
	updateResult *tasksvc.UpdateTaskResult
	updateErr    error
	updateInput  tasksvc.UpdateTaskInput
	statusResult *tasksvc.UpdateTaskStatusResult
	statusErr    error
	statusInput  tasksvc.UpdateTaskStatusInput
	deleteErr    error
	deleteInput  tasksvc.DeleteTaskInput
}

func (s *fakeTaskService) CreateTask(_ context.Context, input tasksvc.CreateTaskInput) (*tasksvc.CreateTaskResult, error) {
	s.createInput = input
	if s.createErr != nil {
		return nil, s.createErr
	}
	return s.createResult, nil
}

func (s *fakeTaskService) ListTasks(_ context.Context, input tasksvc.ListTasksInput) (*tasksvc.ListTasksResult, error) {
	s.listInput = input
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.listResult, nil
}

func (s *fakeTaskService) GetTask(_ context.Context, input tasksvc.GetTaskInput) (*tasksvc.GetTaskResult, error) {
	s.getInput = input
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.getResult, nil
}

func (s *fakeTaskService) UpdateTask(_ context.Context, input tasksvc.UpdateTaskInput) (*tasksvc.UpdateTaskResult, error) {
	s.updateInput = input
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	return s.updateResult, nil
}

func (s *fakeTaskService) UpdateTaskStatus(_ context.Context, input tasksvc.UpdateTaskStatusInput) (*tasksvc.UpdateTaskStatusResult, error) {
	s.statusInput = input
	if s.statusErr != nil {
		return nil, s.statusErr
	}
	return s.statusResult, nil
}

func (s *fakeTaskService) DeleteTask(_ context.Context, input tasksvc.DeleteTaskInput) error {
	s.deleteInput = input
	return s.deleteErr
}

type fakeSessionService struct {
	result *authsvc.CurrentAccountResult
	err    error
}

func (s *fakeSessionService) CurrentAccount(context.Context, authsvc.CurrentAccountInput) (*authsvc.CurrentAccountResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.result, nil
}

type fakeTenantResolver struct {
	result *workspacesvc.ResolveTenantContextResult
	err    error
}

func (r *fakeTenantResolver) ResolveTenantContext(context.Context, workspacesvc.ResolveTenantContextInput) (*workspacesvc.ResolveTenantContextResult, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.result, nil
}

func TestCreateTask(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	assigneeID := uuid.Must(uuid.NewV7())
	service := &fakeTaskService{
		createResult: &tasksvc.CreateTaskResult{
			Task: task.Task{
				ID:               taskID,
				ProjectID:        projectID,
				No:               "TASK-0001",
				Title:            "Task A",
				Status:           task.StatusTodo,
				Priority:         task.PriorityHigh,
				AssigneeMemberID: &assigneeID,
			},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspace/projects/"+projectID.String()+"/tasks", bytes.NewBufferString(`{"title":"Task A","priority":"high","assignee_member_id":"`+assigneeID.String()+`"}`))
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
	if service.createInput.ProjectID != projectID {
		t.Fatalf("project ID = %s, want %s", service.createInput.ProjectID, projectID)
	}
	if service.createInput.AssigneeMemberID == nil || *service.createInput.AssigneeMemberID != assigneeID {
		t.Fatalf("assignee member ID = %#v, want %s", service.createInput.AssigneeMemberID, assigneeID)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data := body["data"].(map[string]any)
	if _, ok := data["tenant_id"]; ok {
		t.Fatalf("response must not expose tenant_id: %#v", data)
	}
	if data["no"] != "TASK-0001" {
		t.Fatalf("task.no = %v, want TASK-0001", data["no"])
	}
}

func TestListTasks(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleUser)
	projectID := uuid.Must(uuid.NewV7())
	service := &fakeTaskService{
		listResult: &tasksvc.ListTasksResult{
			Total: 1,
			Items: []task.Task{
				{ID: uuid.Must(uuid.NewV7()), ProjectID: projectID, No: "TASK-0001", Title: "Task A", Status: task.StatusTodo, Priority: task.PriorityMedium},
			},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/projects/"+projectID.String()+"/tasks?page=1&limit=10", nil)
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
	if service.listInput.ProjectID != projectID {
		t.Fatalf("project ID = %s, want %s", service.listInput.ProjectID, projectID)
	}
	if service.listInput.Limit != 10 || service.listInput.Offset != 0 {
		t.Fatalf("pagination = limit %d offset %d, want 10/0", service.listInput.Limit, service.listInput.Offset)
	}
}

func TestGetTask(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleUser)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	service := &fakeTaskService{
		getResult: &tasksvc.GetTaskResult{
			Task: task.Task{ID: taskID, ProjectID: projectID, No: "TASK-0001", Title: "Task A", Status: task.StatusTodo, Priority: task.PriorityMedium},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String(), nil)
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
	if service.getInput.TaskID != taskID {
		t.Fatalf("task ID = %s, want %s", service.getInput.TaskID, taskID)
	}
}

func TestCreateTaskRequiresManagePermission(t *testing.T) {
	app := newTaskTestApp(newTestHandler(&fakeTaskService{}, uuid.Must(uuid.NewV7()), testTenantContext(workspace.WorkspaceRoleUser)))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspace/projects/"+uuid.Must(uuid.NewV7()).String()+"/tasks", bytes.NewBufferString(`{"title":"Task A"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertTaskError(t, resp, http.StatusForbidden, "PERMISSION_DENIED")
}

func TestUpdateTask(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	assigneeID := uuid.Must(uuid.NewV7())
	service := &fakeTaskService{
		updateResult: &tasksvc.UpdateTaskResult{
			Task: task.Task{
				ID:               taskID,
				ProjectID:        projectID,
				No:               "TASK-0001",
				Title:            "Task B",
				Status:           task.StatusTodo,
				Priority:         task.PriorityHigh,
				AssigneeMemberID: &assigneeID,
			},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String(), bytes.NewBufferString(`{"title":"Task B","priority":"high","assignee_member_id":"`+assigneeID.String()+`"}`))
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
	if service.updateInput.TaskID != taskID {
		t.Fatalf("task ID = %s, want %s", service.updateInput.TaskID, taskID)
	}
	if service.updateInput.AssigneeMemberID == nil || *service.updateInput.AssigneeMemberID == nil || **service.updateInput.AssigneeMemberID != assigneeID {
		t.Fatalf("assignee member ID = %#v, want %s", service.updateInput.AssigneeMemberID, assigneeID)
	}
}

func TestUpdateTaskStatus(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	service := &fakeTaskService{
		statusResult: &tasksvc.UpdateTaskStatusResult{
			Task: task.Task{
				ID:        taskID,
				ProjectID: projectID,
				No:        "TASK-0001",
				Title:     "Task A",
				Status:    task.StatusInProgress,
				Priority:  task.PriorityMedium,
			},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/status", bytes.NewBufferString(`{"status":"in_progress"}`))
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
	if service.statusInput.TaskID != taskID {
		t.Fatalf("task ID = %s, want %s", service.statusInput.TaskID, taskID)
	}
	if service.statusInput.Status != task.StatusInProgress {
		t.Fatalf("status = %s, want in_progress", service.statusInput.Status)
	}
}

func TestUpdateTaskStatusRejectsInvalidStatus(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	service := &fakeTaskService{statusErr: tasksvc.ErrTaskStatusInvalid}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/status", bytes.NewBufferString(`{"status":"invalid"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertTaskError(t, resp, http.StatusBadRequest, "TASK_STATUS_INVALID")
}

func TestDeleteTask(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	service := &fakeTaskService{}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String(), nil)
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
	if service.deleteInput.TaskID != taskID {
		t.Fatalf("task ID = %s, want %s", service.deleteInput.TaskID, taskID)
	}
}

func newTestHandler(taskService TaskService, accountID uuid.UUID, tenantContext workspace.TenantContext) Handler {
	return NewHandler(
		taskService,
		&fakeSessionService{result: &authsvc.CurrentAccountResult{Account: auth.UserAccount{ID: accountID, Status: auth.UserAccountStatusActive}}},
		&fakeTenantResolver{result: &workspacesvc.ResolveTenantContextResult{Context: tenantContext}},
		CookieConfig{Name: "prasankit_session", TTL: 24 * time.Hour, SameSite: "Lax"},
	)
}

func newTaskTestApp(handler Handler) *fiber.App {
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

func assertTaskError(t *testing.T, resp *http.Response, wantStatus int, wantCode string) {
	t.Helper()

	if resp.StatusCode != wantStatus {
		t.Fatalf("status code = %d, want %d", resp.StatusCode, wantStatus)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	gotError := body["error"].(map[string]any)
	if gotError["code"] != wantCode {
		t.Fatalf("error.code = %v, want %s", gotError["code"], wantCode)
	}
}
