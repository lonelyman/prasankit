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
	createResult          *tasksvc.CreateTaskResult
	createErr             error
	createInput           tasksvc.CreateTaskInput
	listResult            *tasksvc.ListTasksResult
	listErr               error
	listInput             tasksvc.ListTasksInput
	summaryResult         *tasksvc.GetTaskBoardSummaryResult
	summaryErr            error
	summaryInput          tasksvc.GetTaskBoardSummaryInput
	getResult             *tasksvc.GetTaskResult
	getErr                error
	getInput              tasksvc.GetTaskInput
	activityResult        *tasksvc.ListTaskActivitiesResult
	activityErr           error
	activityInput         tasksvc.ListTaskActivitiesInput
	commentResult         *tasksvc.ListTaskCommentsResult
	commentErr            error
	commentInput          tasksvc.ListTaskCommentsInput
	createCommentResult   *tasksvc.CreateTaskCommentResult
	createCommentErr      error
	createCommentInput    tasksvc.CreateTaskCommentInput
	deleteCommentErr      error
	deleteCommentInput    tasksvc.DeleteTaskCommentInput
	checklistResult       *tasksvc.ListTaskChecklistItemsResult
	checklistErr          error
	checklistInput        tasksvc.ListTaskChecklistItemsInput
	createChecklistResult *tasksvc.CreateTaskChecklistItemResult
	createChecklistErr    error
	createChecklistInput  tasksvc.CreateTaskChecklistItemInput
	updateChecklistResult *tasksvc.UpdateTaskChecklistItemResult
	updateChecklistErr    error
	updateChecklistInput  tasksvc.UpdateTaskChecklistItemInput
	deleteChecklistErr    error
	deleteChecklistInput  tasksvc.DeleteTaskChecklistItemInput
	uploadResult          *tasksvc.CreateTaskAttachmentUploadResult
	uploadErr             error
	uploadInput           tasksvc.CreateTaskAttachmentUploadInput
	completeErr           error
	completeInput         tasksvc.CompleteTaskAttachmentUploadInput
	downloadResult        *tasksvc.GetTaskAttachmentDownloadURLResult
	downloadErr           error
	downloadInput         tasksvc.GetTaskAttachmentDownloadURLInput
	attachmentResult      *tasksvc.ListTaskAttachmentsResult
	attachmentErr         error
	attachmentInput       tasksvc.ListTaskAttachmentsInput
	deleteAttachmentErr   error
	deleteAttachmentInput tasksvc.DeleteTaskAttachmentInput
	tagResult             *tasksvc.ListTaskTagsResult
	tagErr                error
	tagInput              tasksvc.ListTaskTagsInput
	assignTagResult       *tasksvc.AssignTaskTagResult
	assignTagErr          error
	assignTagInput        tasksvc.AssignTaskTagInput
	removeTagErr          error
	removeTagInput        tasksvc.RemoveTaskTagInput
	relationResult        *tasksvc.ListTaskRelationsResult
	relationErr           error
	relationInput         tasksvc.ListTaskRelationsInput
	createRelationResult  *tasksvc.CreateTaskRelationResult
	createRelationErr     error
	createRelationInput   tasksvc.CreateTaskRelationInput
	deleteRelationErr     error
	deleteRelationInput   tasksvc.DeleteTaskRelationInput
	viewResult            *tasksvc.ListTaskViewsResult
	viewErr               error
	viewInput             tasksvc.ListTaskViewsInput
	createViewResult      *tasksvc.CreateTaskViewResult
	createViewErr         error
	createViewInput       tasksvc.CreateTaskViewInput
	updateViewResult      *tasksvc.UpdateTaskViewResult
	updateViewErr         error
	updateViewInput       tasksvc.UpdateTaskViewInput
	deleteViewErr         error
	deleteViewInput       tasksvc.DeleteTaskViewInput
	updateResult          *tasksvc.UpdateTaskResult
	updateErr             error
	updateInput           tasksvc.UpdateTaskInput
	statusResult          *tasksvc.UpdateTaskStatusResult
	statusErr             error
	statusInput           tasksvc.UpdateTaskStatusInput
	deleteErr             error
	deleteInput           tasksvc.DeleteTaskInput
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

func (s *fakeTaskService) GetTaskBoardSummary(_ context.Context, input tasksvc.GetTaskBoardSummaryInput) (*tasksvc.GetTaskBoardSummaryResult, error) {
	s.summaryInput = input
	if s.summaryErr != nil {
		return nil, s.summaryErr
	}
	return s.summaryResult, nil
}

func (s *fakeTaskService) GetTask(_ context.Context, input tasksvc.GetTaskInput) (*tasksvc.GetTaskResult, error) {
	s.getInput = input
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.getResult, nil
}

func (s *fakeTaskService) ListTaskActivities(_ context.Context, input tasksvc.ListTaskActivitiesInput) (*tasksvc.ListTaskActivitiesResult, error) {
	s.activityInput = input
	if s.activityErr != nil {
		return nil, s.activityErr
	}
	return s.activityResult, nil
}

func (s *fakeTaskService) CreateTaskComment(_ context.Context, input tasksvc.CreateTaskCommentInput) (*tasksvc.CreateTaskCommentResult, error) {
	s.createCommentInput = input
	if s.createCommentErr != nil {
		return nil, s.createCommentErr
	}
	return s.createCommentResult, nil
}

func (s *fakeTaskService) ListTaskComments(_ context.Context, input tasksvc.ListTaskCommentsInput) (*tasksvc.ListTaskCommentsResult, error) {
	s.commentInput = input
	if s.commentErr != nil {
		return nil, s.commentErr
	}
	return s.commentResult, nil
}

func (s *fakeTaskService) DeleteTaskComment(_ context.Context, input tasksvc.DeleteTaskCommentInput) error {
	s.deleteCommentInput = input
	return s.deleteCommentErr
}

func (s *fakeTaskService) CreateTaskChecklistItem(_ context.Context, input tasksvc.CreateTaskChecklistItemInput) (*tasksvc.CreateTaskChecklistItemResult, error) {
	s.createChecklistInput = input
	if s.createChecklistErr != nil {
		return nil, s.createChecklistErr
	}
	return s.createChecklistResult, nil
}

func (s *fakeTaskService) ListTaskChecklistItems(_ context.Context, input tasksvc.ListTaskChecklistItemsInput) (*tasksvc.ListTaskChecklistItemsResult, error) {
	s.checklistInput = input
	if s.checklistErr != nil {
		return nil, s.checklistErr
	}
	return s.checklistResult, nil
}

func (s *fakeTaskService) UpdateTaskChecklistItem(_ context.Context, input tasksvc.UpdateTaskChecklistItemInput) (*tasksvc.UpdateTaskChecklistItemResult, error) {
	s.updateChecklistInput = input
	if s.updateChecklistErr != nil {
		return nil, s.updateChecklistErr
	}
	return s.updateChecklistResult, nil
}

func (s *fakeTaskService) DeleteTaskChecklistItem(_ context.Context, input tasksvc.DeleteTaskChecklistItemInput) error {
	s.deleteChecklistInput = input
	return s.deleteChecklistErr
}

func (s *fakeTaskService) CreateTaskAttachmentUpload(_ context.Context, input tasksvc.CreateTaskAttachmentUploadInput) (*tasksvc.CreateTaskAttachmentUploadResult, error) {
	s.uploadInput = input
	if s.uploadErr != nil {
		return nil, s.uploadErr
	}
	return s.uploadResult, nil
}

func (s *fakeTaskService) CompleteTaskAttachmentUpload(_ context.Context, input tasksvc.CompleteTaskAttachmentUploadInput) error {
	s.completeInput = input
	return s.completeErr
}

func (s *fakeTaskService) GetTaskAttachmentDownloadURL(_ context.Context, input tasksvc.GetTaskAttachmentDownloadURLInput) (*tasksvc.GetTaskAttachmentDownloadURLResult, error) {
	s.downloadInput = input
	if s.downloadErr != nil {
		return nil, s.downloadErr
	}
	return s.downloadResult, nil
}

func (s *fakeTaskService) ListTaskAttachments(_ context.Context, input tasksvc.ListTaskAttachmentsInput) (*tasksvc.ListTaskAttachmentsResult, error) {
	s.attachmentInput = input
	if s.attachmentErr != nil {
		return nil, s.attachmentErr
	}
	return s.attachmentResult, nil
}

func (s *fakeTaskService) DeleteTaskAttachment(_ context.Context, input tasksvc.DeleteTaskAttachmentInput) error {
	s.deleteAttachmentInput = input
	return s.deleteAttachmentErr
}

func (s *fakeTaskService) AssignTaskTag(_ context.Context, input tasksvc.AssignTaskTagInput) (*tasksvc.AssignTaskTagResult, error) {
	s.assignTagInput = input
	if s.assignTagErr != nil {
		return nil, s.assignTagErr
	}
	return s.assignTagResult, nil
}

func (s *fakeTaskService) ListTaskTags(_ context.Context, input tasksvc.ListTaskTagsInput) (*tasksvc.ListTaskTagsResult, error) {
	s.tagInput = input
	if s.tagErr != nil {
		return nil, s.tagErr
	}
	return s.tagResult, nil
}

func (s *fakeTaskService) RemoveTaskTag(_ context.Context, input tasksvc.RemoveTaskTagInput) error {
	s.removeTagInput = input
	return s.removeTagErr
}

func (s *fakeTaskService) CreateTaskRelation(_ context.Context, input tasksvc.CreateTaskRelationInput) (*tasksvc.CreateTaskRelationResult, error) {
	s.createRelationInput = input
	if s.createRelationErr != nil {
		return nil, s.createRelationErr
	}
	return s.createRelationResult, nil
}

func (s *fakeTaskService) ListTaskRelations(_ context.Context, input tasksvc.ListTaskRelationsInput) (*tasksvc.ListTaskRelationsResult, error) {
	s.relationInput = input
	if s.relationErr != nil {
		return nil, s.relationErr
	}
	return s.relationResult, nil
}

func (s *fakeTaskService) DeleteTaskRelation(_ context.Context, input tasksvc.DeleteTaskRelationInput) error {
	s.deleteRelationInput = input
	return s.deleteRelationErr
}

func (s *fakeTaskService) CreateTaskView(_ context.Context, input tasksvc.CreateTaskViewInput) (*tasksvc.CreateTaskViewResult, error) {
	s.createViewInput = input
	if s.createViewErr != nil {
		return nil, s.createViewErr
	}
	return s.createViewResult, nil
}

func (s *fakeTaskService) ListTaskViews(_ context.Context, input tasksvc.ListTaskViewsInput) (*tasksvc.ListTaskViewsResult, error) {
	s.viewInput = input
	if s.viewErr != nil {
		return nil, s.viewErr
	}
	return s.viewResult, nil
}

func (s *fakeTaskService) UpdateTaskView(_ context.Context, input tasksvc.UpdateTaskViewInput) (*tasksvc.UpdateTaskViewResult, error) {
	s.updateViewInput = input
	if s.updateViewErr != nil {
		return nil, s.updateViewErr
	}
	return s.updateViewResult, nil
}

func (s *fakeTaskService) DeleteTaskView(_ context.Context, input tasksvc.DeleteTaskViewInput) error {
	s.deleteViewInput = input
	return s.deleteViewErr
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
	assigneeID := uuid.Must(uuid.NewV7())
	tagID := uuid.Must(uuid.NewV7())
	service := &fakeTaskService{
		listResult: &tasksvc.ListTasksResult{
			Total: 1,
			Items: []task.Task{
				{ID: uuid.Must(uuid.NewV7()), ProjectID: projectID, No: "TASK-0001", Title: "Task A", Status: task.StatusTodo, Priority: task.PriorityMedium},
			},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/projects/"+projectID.String()+"/tasks?page=1&limit=10&status=todo&priority=medium&assignee_member_id="+assigneeID.String()+"&tag_id="+tagID.String()+"&q=proposal", nil)
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
	if service.listInput.Status == nil || *service.listInput.Status != task.StatusTodo {
		t.Fatalf("status filter = %#v, want todo", service.listInput.Status)
	}
	if service.listInput.Priority == nil || *service.listInput.Priority != task.PriorityMedium {
		t.Fatalf("priority filter = %#v, want medium", service.listInput.Priority)
	}
	if service.listInput.AssigneeMemberID == nil || *service.listInput.AssigneeMemberID != assigneeID {
		t.Fatalf("assignee filter = %#v, want %s", service.listInput.AssigneeMemberID, assigneeID)
	}
	if service.listInput.TagID == nil || *service.listInput.TagID != tagID {
		t.Fatalf("tag filter = %#v, want %s", service.listInput.TagID, tagID)
	}
	if service.listInput.Search != "proposal" {
		t.Fatalf("search filter = %q, want proposal", service.listInput.Search)
	}
}

func TestListTasksRejectsInvalidAssigneeFilter(t *testing.T) {
	app := newTaskTestApp(newTestHandler(&fakeTaskService{}, uuid.Must(uuid.NewV7()), testTenantContext(workspace.WorkspaceRoleUser)))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/projects/"+uuid.Must(uuid.NewV7()).String()+"/tasks?assignee_member_id=invalid", nil)
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertTaskError(t, resp, http.StatusBadRequest, "TASK_ASSIGNEE_MEMBER_ID_INVALID")
}

func TestListTasksRejectsInvalidTagFilter(t *testing.T) {
	app := newTaskTestApp(newTestHandler(&fakeTaskService{}, uuid.Must(uuid.NewV7()), testTenantContext(workspace.WorkspaceRoleUser)))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/projects/"+uuid.Must(uuid.NewV7()).String()+"/tasks?tag_id=invalid", nil)
	req.Header.Set("X-Workspace-Slug", "team-one")
	req.AddCookie(&http.Cookie{Name: "prasankit_session", Value: "raw-session-token"})
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertTaskError(t, resp, http.StatusBadRequest, "TASK_TAG_ID_INVALID")
}

func TestGetTaskBoardSummary(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleUser)
	projectID := uuid.Must(uuid.NewV7())
	service := &fakeTaskService{
		summaryResult: &tasksvc.GetTaskBoardSummaryResult{
			ProjectID: projectID,
			Total:     3,
			Counts: []task.StatusCount{
				{Status: task.StatusTodo, Count: 2},
				{Status: task.StatusInProgress, Count: 1},
				{Status: task.StatusBlocked, Count: 0},
				{Status: task.StatusDone, Count: 0},
				{Status: task.StatusCancelled, Count: 0},
			},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/summary", nil)
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
	if service.summaryInput.ProjectID != projectID {
		t.Fatalf("project ID = %s, want %s", service.summaryInput.ProjectID, projectID)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data := body["data"].(map[string]any)
	if data["total"].(float64) != 3 {
		t.Fatalf("total = %v, want 3", data["total"])
	}
	counts := data["counts"].(map[string]any)
	if counts["todo"].(float64) != 2 || counts["in_progress"].(float64) != 1 {
		t.Fatalf("counts = %#v, want todo=2 in_progress=1", counts)
	}
}

func TestCreateTaskView(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleUser)
	projectID := uuid.Must(uuid.NewV7())
	viewID := uuid.Must(uuid.NewV7())
	tagID := uuid.Must(uuid.NewV7())
	now := time.Date(2026, 5, 1, 10, 30, 0, 0, time.UTC)
	service := &fakeTaskService{
		createViewResult: &tasksvc.CreateTaskViewResult{
			View: task.View{ID: viewID, ProjectID: projectID, Name: "My Todo", FiltersJSON: map[string]any{"status": "todo", "tag_id": tagID.String()}, CreatedAt: now, UpdatedAt: now},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	body := `{"name":"My Todo","filters":{"status":"todo","tag_id":"` + tagID.String() + `","q":"proposal"},"sort_order":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/views", bytes.NewBufferString(body))
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
	if service.createViewInput.Name != "My Todo" || service.createViewInput.SortOrder != 1 {
		t.Fatalf("view input = %#v, want name and sort order", service.createViewInput)
	}
	if service.createViewInput.Filters["q"] != "proposal" {
		t.Fatalf("filters = %#v, want q proposal", service.createViewInput.Filters)
	}
}

func TestListTaskViews(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleUser)
	projectID := uuid.Must(uuid.NewV7())
	now := time.Date(2026, 5, 1, 10, 30, 0, 0, time.UTC)
	service := &fakeTaskService{
		viewResult: &tasksvc.ListTaskViewsResult{
			Items: []task.View{{ID: uuid.Must(uuid.NewV7()), ProjectID: projectID, Name: "My Todo", FiltersJSON: map[string]any{"status": "todo"}, CreatedAt: now, UpdatedAt: now}},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/views", nil)
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
	if service.viewInput.ProjectID != projectID {
		t.Fatalf("project ID = %s, want %s", service.viewInput.ProjectID, projectID)
	}
}

func TestUpdateTaskView(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleUser)
	projectID := uuid.Must(uuid.NewV7())
	viewID := uuid.Must(uuid.NewV7())
	now := time.Date(2026, 5, 1, 10, 30, 0, 0, time.UTC)
	service := &fakeTaskService{
		updateViewResult: &tasksvc.UpdateTaskViewResult{
			View: task.View{ID: viewID, ProjectID: projectID, Name: "Updated", FiltersJSON: map[string]any{"q": "proposal"}, CreatedAt: now, UpdatedAt: now},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/views/"+viewID.String(), bytes.NewBufferString(`{"name":"Updated","filters":{"q":"proposal"},"sort_order":2}`))
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
	if service.updateViewInput.ViewID != viewID {
		t.Fatalf("view ID = %s, want %s", service.updateViewInput.ViewID, viewID)
	}
}

func TestDeleteTaskView(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleUser)
	projectID := uuid.Must(uuid.NewV7())
	viewID := uuid.Must(uuid.NewV7())
	service := &fakeTaskService{}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/views/"+viewID.String(), nil)
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
	if service.deleteViewInput.ViewID != viewID {
		t.Fatalf("view ID = %s, want %s", service.deleteViewInput.ViewID, viewID)
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

func TestListTaskActivities(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleUser)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	actorID := uuid.Must(uuid.NewV7())
	now := time.Date(2026, 5, 1, 10, 30, 0, 0, time.UTC)
	service := &fakeTaskService{
		activityResult: &tasksvc.ListTaskActivitiesResult{
			Total: 1,
			Items: []task.Activity{
				{
					ID:             uuid.Must(uuid.NewV7()),
					ProjectID:      projectID,
					TaskID:         taskID,
					ActorAccountID: actorID,
					Action:         task.ActivityCreated,
					MetadataJSON:   map[string]any{"title": "Task A"},
					CreatedAt:      now,
				},
			},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/activities?page=1&limit=10", nil)
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
	if service.activityInput.TaskID != taskID {
		t.Fatalf("task ID = %s, want %s", service.activityInput.TaskID, taskID)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data := body["data"].(map[string]any)
	items := data["items"].([]any)
	first := items[0].(map[string]any)
	if first["action"] != string(task.ActivityCreated) {
		t.Fatalf("activity action = %v, want created", first["action"])
	}
}

func TestCreateTaskComment(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	commentID := uuid.Must(uuid.NewV7())
	now := time.Date(2026, 5, 1, 10, 30, 0, 0, time.UTC)
	service := &fakeTaskService{
		createCommentResult: &tasksvc.CreateTaskCommentResult{
			Comment: task.Comment{ID: commentID, ProjectID: projectID, TaskID: taskID, Body: "please review", CreatedBy: accountID, CreatedAt: now, UpdatedAt: now},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/comments", bytes.NewBufferString(`{"body":"please review"}`))
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
	if service.createCommentInput.Body != "please review" {
		t.Fatalf("body = %q, want please review", service.createCommentInput.Body)
	}
}

func TestListTaskComments(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleUser)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	now := time.Date(2026, 5, 1, 10, 30, 0, 0, time.UTC)
	service := &fakeTaskService{
		commentResult: &tasksvc.ListTaskCommentsResult{
			Total: 1,
			Items: []task.Comment{
				{ID: uuid.Must(uuid.NewV7()), ProjectID: projectID, TaskID: taskID, Body: "please review", CreatedBy: accountID, CreatedAt: now, UpdatedAt: now},
			},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/comments?page=1&limit=10", nil)
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
	if service.commentInput.TaskID != taskID {
		t.Fatalf("task ID = %s, want %s", service.commentInput.TaskID, taskID)
	}
}

func TestDeleteTaskComment(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	commentID := uuid.Must(uuid.NewV7())
	service := &fakeTaskService{}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/comments/"+commentID.String(), nil)
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
	if service.deleteCommentInput.CommentID != commentID {
		t.Fatalf("comment ID = %s, want %s", service.deleteCommentInput.CommentID, commentID)
	}
}

func TestCreateTaskChecklistItem(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	itemID := uuid.Must(uuid.NewV7())
	now := time.Date(2026, 5, 1, 10, 30, 0, 0, time.UTC)
	service := &fakeTaskService{
		createChecklistResult: &tasksvc.CreateTaskChecklistItemResult{
			Item: task.ChecklistItem{ID: itemID, ProjectID: projectID, TaskID: taskID, Text: "prepare document", SortOrder: 1, CreatedBy: accountID, CreatedAt: now, UpdatedAt: now},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/checklist", bytes.NewBufferString(`{"text":"prepare document","sort_order":1}`))
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
	if service.createChecklistInput.Text != "prepare document" || service.createChecklistInput.SortOrder != 1 {
		t.Fatalf("checklist input = %#v, want text and sort order", service.createChecklistInput)
	}
}

func TestListTaskChecklistItems(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleUser)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	now := time.Date(2026, 5, 1, 10, 30, 0, 0, time.UTC)
	service := &fakeTaskService{
		checklistResult: &tasksvc.ListTaskChecklistItemsResult{
			Items: []task.ChecklistItem{
				{ID: uuid.Must(uuid.NewV7()), ProjectID: projectID, TaskID: taskID, Text: "prepare document", CreatedBy: accountID, CreatedAt: now, UpdatedAt: now},
			},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/checklist", nil)
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
	if service.checklistInput.TaskID != taskID {
		t.Fatalf("task ID = %s, want %s", service.checklistInput.TaskID, taskID)
	}
}

func TestUpdateTaskChecklistItem(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	itemID := uuid.Must(uuid.NewV7())
	now := time.Date(2026, 5, 1, 10, 30, 0, 0, time.UTC)
	service := &fakeTaskService{
		updateChecklistResult: &tasksvc.UpdateTaskChecklistItemResult{
			Item: task.ChecklistItem{ID: itemID, ProjectID: projectID, TaskID: taskID, Text: "prepare final document", IsCompleted: true, CreatedBy: accountID, CreatedAt: now, UpdatedAt: now, CompletedBy: &accountID, CompletedAt: &now},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/checklist/"+itemID.String(), bytes.NewBufferString(`{"text":"prepare final document","is_completed":true}`))
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
	if service.updateChecklistInput.ItemID != itemID {
		t.Fatalf("item ID = %s, want %s", service.updateChecklistInput.ItemID, itemID)
	}
	if service.updateChecklistInput.IsCompleted == nil || !*service.updateChecklistInput.IsCompleted {
		t.Fatalf("is completed = %#v, want true", service.updateChecklistInput.IsCompleted)
	}
}

func TestDeleteTaskChecklistItem(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	itemID := uuid.Must(uuid.NewV7())
	service := &fakeTaskService{}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/checklist/"+itemID.String(), nil)
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
	if service.deleteChecklistInput.ItemID != itemID {
		t.Fatalf("item ID = %s, want %s", service.deleteChecklistInput.ItemID, itemID)
	}
}

func TestCreateTaskAttachmentUpload(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	attachmentID := uuid.Must(uuid.NewV7())
	now := time.Date(2026, 5, 1, 10, 30, 0, 0, time.UTC)
	service := &fakeTaskService{
		uploadResult: &tasksvc.CreateTaskAttachmentUploadResult{
			Attachment: task.Attachment{
				ID:           attachmentID,
				ProjectID:    projectID,
				TaskID:       taskID,
				FileName:     "spec.pdf",
				ContentType:  "application/pdf",
				SizeBytes:    1024,
				UploadStatus: task.AttachmentPending,
				UploadedBy:   accountID,
				CreatedAt:    now,
			},
			UploadURL: task.AttachmentUploadURL{URL: "http://minio/upload", ExpiresAt: now.Add(15 * time.Minute)},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/attachments/uploads", bytes.NewBufferString(`{"file_name":"spec.pdf","content_type":"application/pdf","size_bytes":1024}`))
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
	if service.uploadInput.FileName != "spec.pdf" || service.uploadInput.SizeBytes != 1024 {
		t.Fatalf("upload input = %#v, want file spec.pdf size 1024", service.uploadInput)
	}
}

func TestCompleteTaskAttachmentUpload(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	attachmentID := uuid.Must(uuid.NewV7())
	service := &fakeTaskService{}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/attachments/"+attachmentID.String()+"/complete", nil)
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
	if service.completeInput.AttachmentID != attachmentID {
		t.Fatalf("attachment ID = %s, want %s", service.completeInput.AttachmentID, attachmentID)
	}
}

func TestListTaskAttachments(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleUser)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	service := &fakeTaskService{
		attachmentResult: &tasksvc.ListTaskAttachmentsResult{
			Total: 1,
			Items: []task.Attachment{
				{ID: uuid.Must(uuid.NewV7()), ProjectID: projectID, TaskID: taskID, FileName: "spec.pdf", UploadStatus: task.AttachmentUploaded, UploadedBy: accountID, CreatedAt: time.Now().UTC()},
			},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/attachments?page=1&limit=10", nil)
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
	if service.attachmentInput.TaskID != taskID {
		t.Fatalf("task ID = %s, want %s", service.attachmentInput.TaskID, taskID)
	}
}

func TestGetTaskAttachmentDownloadURL(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleUser)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	attachmentID := uuid.Must(uuid.NewV7())
	now := time.Date(2026, 5, 1, 10, 30, 0, 0, time.UTC)
	service := &fakeTaskService{
		downloadResult: &tasksvc.GetTaskAttachmentDownloadURLResult{
			Attachment: task.Attachment{
				ID:           attachmentID,
				ProjectID:    projectID,
				TaskID:       taskID,
				FileName:     "spec.pdf",
				ContentType:  "application/pdf",
				SizeBytes:    1024,
				UploadStatus: task.AttachmentUploaded,
				UploadedBy:   accountID,
				CreatedAt:    now,
			},
			DownloadURL: task.AttachmentDownloadURL{URL: "http://minio/download", ExpiresAt: now.Add(15 * time.Minute)},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/attachments/"+attachmentID.String()+"/download", nil)
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
	if service.downloadInput.AttachmentID != attachmentID {
		t.Fatalf("attachment ID = %s, want %s", service.downloadInput.AttachmentID, attachmentID)
	}
}

func TestDeleteTaskAttachment(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	attachmentID := uuid.Must(uuid.NewV7())
	service := &fakeTaskService{}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/attachments/"+attachmentID.String(), nil)
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
	if service.deleteAttachmentInput.AttachmentID != attachmentID {
		t.Fatalf("attachment ID = %s, want %s", service.deleteAttachmentInput.AttachmentID, attachmentID)
	}
}

func TestAssignTaskTag(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	tagID := uuid.Must(uuid.NewV7())
	now := time.Date(2026, 5, 1, 10, 30, 0, 0, time.UTC)
	service := &fakeTaskService{
		assignTagResult: &tasksvc.AssignTaskTagResult{
			Tag: task.Tag{ID: tagID, ProjectID: projectID, Name: "Review", CreatedBy: accountID, CreatedAt: now},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/tags", bytes.NewBufferString(`{"name":"Review","color":"#2f80ed"}`))
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
	if service.assignTagInput.TaskID != taskID || service.assignTagInput.Name != "Review" {
		t.Fatalf("tag input = %#v, want task and name", service.assignTagInput)
	}
}

func TestListTaskTags(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleUser)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	service := &fakeTaskService{
		tagResult: &tasksvc.ListTaskTagsResult{
			Items: []task.Tag{{ID: uuid.Must(uuid.NewV7()), ProjectID: projectID, Name: "Review", CreatedBy: accountID, CreatedAt: time.Now().UTC()}},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/tags", nil)
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
	if service.tagInput.TaskID != taskID {
		t.Fatalf("task ID = %s, want %s", service.tagInput.TaskID, taskID)
	}
}

func TestRemoveTaskTag(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	tagID := uuid.Must(uuid.NewV7())
	service := &fakeTaskService{}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/tags/"+tagID.String(), nil)
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
	if service.removeTagInput.TagID != tagID {
		t.Fatalf("tag ID = %s, want %s", service.removeTagInput.TagID, tagID)
	}
}

func TestCreateTaskRelation(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	targetTaskID := uuid.Must(uuid.NewV7())
	relationID := uuid.Must(uuid.NewV7())
	now := time.Date(2026, 5, 1, 10, 30, 0, 0, time.UTC)
	service := &fakeTaskService{
		createRelationResult: &tasksvc.CreateTaskRelationResult{
			Relation: task.Relation{ID: relationID, ProjectID: projectID, SourceTaskID: taskID, TargetTaskID: targetTaskID, Type: task.RelationBlocks, CreatedBy: accountID, CreatedAt: now},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/relations", bytes.NewBufferString(`{"target_task_id":"`+targetTaskID.String()+`","type":"blocks"}`))
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
	if service.createRelationInput.TargetTaskID != targetTaskID || service.createRelationInput.Type != task.RelationBlocks {
		t.Fatalf("relation input = %#v, want target and type", service.createRelationInput)
	}
}

func TestListTaskRelations(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleUser)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	service := &fakeTaskService{
		relationResult: &tasksvc.ListTaskRelationsResult{
			Items: []task.Relation{{ID: uuid.Must(uuid.NewV7()), ProjectID: projectID, SourceTaskID: taskID, TargetTaskID: uuid.Must(uuid.NewV7()), Type: task.RelationRelatesTo, CreatedBy: accountID, CreatedAt: time.Now().UTC()}},
		},
	}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/relations", nil)
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
	if service.relationInput.TaskID != taskID {
		t.Fatalf("task ID = %s, want %s", service.relationInput.TaskID, taskID)
	}
}

func TestDeleteTaskRelation(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext(workspace.WorkspaceRoleOwner)
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	relationID := uuid.Must(uuid.NewV7())
	service := &fakeTaskService{}
	app := newTaskTestApp(newTestHandler(service, accountID, tenantContext))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspace/projects/"+projectID.String()+"/tasks/"+taskID.String()+"/relations/"+relationID.String(), nil)
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
	if service.deleteRelationInput.RelationID != relationID {
		t.Fatalf("relation ID = %s, want %s", service.deleteRelationInput.RelationID, relationID)
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
