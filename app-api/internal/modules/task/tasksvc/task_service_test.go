package tasksvc

import (
	"context"
	"errors"
	"testing"
	"time"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/task"
	"prasankit-api/internal/modules/workspace"

	"github.com/google/uuid"
)

type fakeRepository struct {
	task              *task.Task
	taskItems         []task.Task
	taskTotal         int
	taskErr           error
	listFilter        task.ListFilter
	statusCounts      []task.StatusCount
	statusCountErr    error
	activity          *task.Activity
	activityItems     []task.Activity
	activityTotal     int
	activityErr       error
	updatePatch       task.Patch
	updateErr         error
	statusTaskID      uuid.UUID
	statusValue       task.Status
	completedDate     *time.Time
	statusErr         error
	deleteTaskID      uuid.UUID
	deleteErr         error
	findTask          *task.Task
	findTaskErr       error
	projectExists     bool
	projectErr        error
	projectID         uuid.UUID
	tenantID          uuid.UUID
	workspaceID       uuid.UUID
	priorityErr       error
	assigneeExists    bool
	assigneeErr       error
	assigneeID        uuid.UUID
	transactionCalled bool
	nextTaskNo        string
}

func (r *fakeRepository) WithinTransaction(ctx context.Context, fn func(context.Context, task.Repository) error) error {
	r.transactionCalled = true
	return fn(ctx, r)
}

func (r *fakeRepository) ProjectExists(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID) (bool, error) {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	if r.projectErr != nil {
		return false, r.projectErr
	}
	return r.projectExists, nil
}

func (r *fakeRepository) FindPriorityByCode(_ context.Context, code task.Priority) (*task.PriorityMaster, error) {
	if r.priorityErr != nil {
		return nil, r.priorityErr
	}
	switch code {
	case task.PriorityLow, task.PriorityMedium, task.PriorityHigh:
		return &task.PriorityMaster{ID: uuid.Must(uuid.NewV7()), Code: code, Name: string(code)}, nil
	}
	return nil, task.ErrPriorityNotFound
}

func (r *fakeRepository) FindActiveProjectMemberByID(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, memberID uuid.UUID) (bool, error) {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	r.assigneeID = memberID
	if r.assigneeErr != nil {
		return false, r.assigneeErr
	}
	return r.assigneeExists, nil
}

func (r *fakeRepository) NextTaskNo(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string, int) (string, error) {
	if r.nextTaskNo != "" {
		return r.nextTaskNo, nil
	}
	return "TASK-0001", nil
}

func (r *fakeRepository) CreateTask(_ context.Context, item *task.Task) error {
	if r.taskErr != nil {
		return r.taskErr
	}
	item.ID = uuid.Must(uuid.NewV7())
	r.task = item
	return nil
}

func (r *fakeRepository) UpdateTask(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, _ uuid.UUID, patch task.Patch) error {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	r.updatePatch = patch
	if r.updateErr != nil {
		return r.updateErr
	}
	return nil
}

func (r *fakeRepository) UpdateTaskStatus(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, status task.Status, completedDate *time.Time, _ uuid.UUID, _ time.Time) error {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	r.statusTaskID = taskID
	r.statusValue = status
	r.completedDate = completedDate
	if r.statusErr != nil {
		return r.statusErr
	}
	return nil
}

func (r *fakeRepository) SoftDeleteTask(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, _ uuid.UUID) error {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	r.deleteTaskID = taskID
	if r.deleteErr != nil {
		return r.deleteErr
	}
	return nil
}

func (r *fakeRepository) FindTaskByID(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID) (*task.Task, error) {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	if r.findTaskErr != nil {
		return nil, r.findTaskErr
	}
	if r.findTask != nil {
		return r.findTask, nil
	}
	return nil, task.ErrTaskNotFound
}

func (r *fakeRepository) ListTasks(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, filter task.ListFilter, _ int, _ int) ([]task.Task, int, error) {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	r.listFilter = filter
	if r.taskErr != nil {
		return nil, 0, r.taskErr
	}
	return r.taskItems, r.taskTotal, nil
}

func (r *fakeRepository) CountTasksByStatus(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID) ([]task.StatusCount, error) {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	if r.statusCountErr != nil {
		return nil, r.statusCountErr
	}
	return r.statusCounts, nil
}

func (r *fakeRepository) CreateTaskActivity(_ context.Context, activity *task.Activity) error {
	if r.activityErr != nil {
		return r.activityErr
	}
	activity.ID = uuid.Must(uuid.NewV7())
	r.activity = activity
	return nil
}

func (r *fakeRepository) ListTaskActivities(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, _ int, _ int) ([]task.Activity, int, error) {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	r.statusTaskID = taskID
	if r.activityErr != nil {
		return nil, 0, r.activityErr
	}
	return r.activityItems, r.activityTotal, nil
}

func TestCreateTask(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	assigneeID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{projectExists: true, assigneeExists: true}

	result, err := NewService(repo).CreateTask(context.Background(), CreateTaskInput{
		Account:          auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext:    tenantContext,
		ProjectID:        projectID,
		Title:            "Task A",
		Priority:         task.PriorityHigh,
		AssigneeMemberID: &assigneeID,
		Description:      "Description",
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if !repo.transactionCalled {
		t.Fatal("transaction was not used")
	}
	if result.Task.ID == uuid.Nil {
		t.Fatal("task ID was not set")
	}
	if result.Task.No != "TASK-0001" {
		t.Fatalf("task no = %s, want TASK-0001", result.Task.No)
	}
	if result.Task.Priority != task.PriorityHigh {
		t.Fatalf("priority = %s, want high", result.Task.Priority)
	}
	if result.Task.Status != task.StatusTodo {
		t.Fatalf("status = %s, want todo", result.Task.Status)
	}
	if repo.assigneeID != assigneeID {
		t.Fatalf("assignee ID = %s, want %s", repo.assigneeID, assigneeID)
	}
	if repo.activity == nil || repo.activity.Action != task.ActivityCreated {
		t.Fatalf("activity = %#v, want created", repo.activity)
	}
}

func TestCreateTaskRejectsInvalidInput(t *testing.T) {
	service := NewService(&fakeRepository{})
	account := auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())

	_, err := service.CreateTask(context.Background(), CreateTaskInput{
		Account:       account,
		TenantContext: tenantContext,
		Title:         "Task A",
	})
	if !errors.Is(err, ErrProjectIDRequired) {
		t.Fatalf("err = %v, want ErrProjectIDRequired", err)
	}

	_, err = service.CreateTask(context.Background(), CreateTaskInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		Title:         "",
	})
	if !errors.Is(err, ErrTaskTitleRequired) {
		t.Fatalf("err = %v, want ErrTaskTitleRequired", err)
	}

	_, err = NewService(&fakeRepository{projectExists: true}).CreateTask(context.Background(), CreateTaskInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		Title:         "Task A",
		Priority:      task.Priority("urgent"),
	})
	if !errors.Is(err, ErrTaskPriorityInvalid) {
		t.Fatalf("err = %v, want ErrTaskPriorityInvalid", err)
	}
}

func TestCreateTaskMapsErrors(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	account := auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}

	_, err := NewService(&fakeRepository{projectExists: false}).CreateTask(context.Background(), CreateTaskInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		Title:         "Task A",
	})
	if !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("err = %v, want ErrProjectNotFound", err)
	}

	assigneeID := uuid.Must(uuid.NewV7())
	_, err = NewService(&fakeRepository{projectExists: true, assigneeExists: false}).CreateTask(context.Background(), CreateTaskInput{
		Account:          account,
		TenantContext:    tenantContext,
		ProjectID:        projectID,
		Title:            "Task A",
		AssigneeMemberID: &assigneeID,
	})
	if !errors.Is(err, ErrTaskAssigneeNotFound) {
		t.Fatalf("err = %v, want ErrTaskAssigneeNotFound", err)
	}
}

func TestListTasks(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	assigneeID := uuid.Must(uuid.NewV7())
	status := task.StatusInProgress
	priority := task.PriorityHigh
	repo := &fakeRepository{
		projectExists: true,
		taskTotal:     1,
		taskItems: []task.Task{
			{ID: uuid.Must(uuid.NewV7()), ProjectID: projectID, No: "TASK-0001", Title: "Task A"},
		},
	}

	result, err := NewService(repo).ListTasks(context.Background(), ListTasksInput{
		Account:          auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext:    tenantContext,
		ProjectID:        projectID,
		Status:           &status,
		Priority:         &priority,
		AssigneeMemberID: &assigneeID,
		Limit:            10,
	})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("result = total %d len %d, want 1/1", result.Total, len(result.Items))
	}
	if repo.projectID != projectID {
		t.Fatalf("project ID = %s, want %s", repo.projectID, projectID)
	}
	if repo.listFilter.Status == nil || *repo.listFilter.Status != task.StatusInProgress {
		t.Fatalf("status filter = %#v, want in_progress", repo.listFilter.Status)
	}
	if repo.listFilter.PriorityID == nil {
		t.Fatal("priority ID filter was not set")
	}
	if repo.listFilter.AssigneeMemberID == nil || *repo.listFilter.AssigneeMemberID != assigneeID {
		t.Fatalf("assignee filter = %#v, want %s", repo.listFilter.AssigneeMemberID, assigneeID)
	}
}

func TestListTasksRejectsInvalidFilters(t *testing.T) {
	account := auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	invalidStatus := task.Status("invalid")
	invalidPriority := task.Priority("urgent")

	_, err := NewService(&fakeRepository{projectExists: true}).ListTasks(context.Background(), ListTasksInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		Status:        &invalidStatus,
	})
	if !errors.Is(err, ErrTaskStatusInvalid) {
		t.Fatalf("err = %v, want ErrTaskStatusInvalid", err)
	}

	_, err = NewService(&fakeRepository{projectExists: true}).ListTasks(context.Background(), ListTasksInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		Priority:      &invalidPriority,
	})
	if !errors.Is(err, ErrTaskPriorityInvalid) {
		t.Fatalf("err = %v, want ErrTaskPriorityInvalid", err)
	}
}

func TestGetTaskBoardSummary(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{
		projectExists: true,
		statusCounts: []task.StatusCount{
			{Status: task.StatusTodo, Count: 2},
			{Status: task.StatusDone, Count: 1},
		},
	}

	result, err := NewService(repo).GetTaskBoardSummary(context.Background(), GetTaskBoardSummaryInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
	})
	if err != nil {
		t.Fatalf("GetTaskBoardSummary: %v", err)
	}
	if result.ProjectID != projectID {
		t.Fatalf("project ID = %s, want %s", result.ProjectID, projectID)
	}
	if result.Total != 3 {
		t.Fatalf("total = %d, want 3", result.Total)
	}
	if len(result.Counts) != len(task.AllStatuses()) {
		t.Fatalf("counts len = %d, want %d", len(result.Counts), len(task.AllStatuses()))
	}
	got := make(map[task.Status]int, len(result.Counts))
	for _, item := range result.Counts {
		got[item.Status] = item.Count
	}
	if got[task.StatusTodo] != 2 || got[task.StatusDone] != 1 || got[task.StatusBlocked] != 0 {
		t.Fatalf("counts = %#v, want todo=2 done=1 blocked=0", got)
	}
}

func TestGetTaskBoardSummaryRejectsInvalidInput(t *testing.T) {
	service := NewService(&fakeRepository{})
	account := auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}

	_, err := service.GetTaskBoardSummary(context.Background(), GetTaskBoardSummaryInput{
		Account:       account,
		TenantContext: testTenantContext(),
	})
	if !errors.Is(err, ErrProjectIDRequired) {
		t.Fatalf("err = %v, want ErrProjectIDRequired", err)
	}
}

func TestListTaskActivities(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{
		findTask:      &task.Task{ID: taskID, ProjectID: projectID, No: "TASK-0001", Title: "Task A"},
		activityTotal: 1,
		activityItems: []task.Activity{
			{ID: uuid.Must(uuid.NewV7()), ProjectID: projectID, TaskID: taskID, Action: task.ActivityCreated},
		},
	}

	result, err := NewService(repo).ListTaskActivities(context.Background(), ListTaskActivitiesInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("ListTaskActivities: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("result = total %d len %d, want 1/1", result.Total, len(result.Items))
	}
	if repo.statusTaskID != taskID {
		t.Fatalf("task ID = %s, want %s", repo.statusTaskID, taskID)
	}
}

func TestGetTask(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{findTask: &task.Task{ID: taskID, ProjectID: projectID, No: "TASK-0001", Title: "Task A"}}

	result, err := NewService(repo).GetTask(context.Background(), GetTaskInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
	})
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if result.Task.ID != taskID {
		t.Fatalf("task ID = %s, want %s", result.Task.ID, taskID)
	}
}

func TestUpdateTask(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	assigneeID := uuid.Must(uuid.NewV7())
	title := "Task B"
	priority := task.PriorityHigh
	description := "Updated"
	repo := &fakeRepository{
		assigneeExists: true,
		findTask:       &task.Task{ID: taskID, ProjectID: projectID, No: "TASK-0001", Title: "Task B", Priority: task.PriorityHigh},
	}

	result, err := NewService(repo).UpdateTask(context.Background(), UpdateTaskInput{
		Account:          auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext:    tenantContext,
		ProjectID:        projectID,
		TaskID:           taskID,
		Title:            &title,
		Priority:         &priority,
		AssigneeMemberID: ptrUUIDPtr(&assigneeID),
		Description:      &description,
	})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	if repo.updatePatch.Title == nil || *repo.updatePatch.Title != "Task B" {
		t.Fatalf("title patch = %#v, want Task B", repo.updatePatch.Title)
	}
	if repo.updatePatch.PriorityID == nil {
		t.Fatal("priority ID patch was not set")
	}
	if repo.updatePatch.AssigneeMemberID == nil || *repo.updatePatch.AssigneeMemberID == nil || **repo.updatePatch.AssigneeMemberID != assigneeID {
		t.Fatalf("assignee patch = %#v, want %s", repo.updatePatch.AssigneeMemberID, assigneeID)
	}
	if result.Task.ID != taskID {
		t.Fatalf("task ID = %s, want %s", result.Task.ID, taskID)
	}
}

func TestUpdateTaskRejectsInvalidInput(t *testing.T) {
	service := NewService(&fakeRepository{})
	account := auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())

	_, err := service.UpdateTask(context.Background(), UpdateTaskInput{
		Account:       account,
		TenantContext: tenantContext,
		TaskID:        taskID,
		Title:         ptrString("Task B"),
	})
	if !errors.Is(err, ErrProjectIDRequired) {
		t.Fatalf("err = %v, want ErrProjectIDRequired", err)
	}

	_, err = service.UpdateTask(context.Background(), UpdateTaskInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		Title:         ptrString("Task B"),
	})
	if !errors.Is(err, ErrTaskIDRequired) {
		t.Fatalf("err = %v, want ErrTaskIDRequired", err)
	}

	_, err = service.UpdateTask(context.Background(), UpdateTaskInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
	})
	if !errors.Is(err, ErrTaskUpdateNoFields) {
		t.Fatalf("err = %v, want ErrTaskUpdateNoFields", err)
	}
}

func TestUpdateTaskStatus(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{
		findTask: &task.Task{ID: taskID, ProjectID: projectID, No: "TASK-0001", Title: "Task A", Status: task.StatusDone},
	}

	result, err := NewService(repo).UpdateTaskStatus(context.Background(), UpdateTaskStatusInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		Status:        task.StatusDone,
	})
	if err != nil {
		t.Fatalf("UpdateTaskStatus: %v", err)
	}
	if repo.statusTaskID != taskID {
		t.Fatalf("task ID = %s, want %s", repo.statusTaskID, taskID)
	}
	if repo.statusValue != task.StatusDone {
		t.Fatalf("status = %s, want done", repo.statusValue)
	}
	if repo.completedDate == nil {
		t.Fatal("completed date was not set for done status")
	}
	if result.Task.Status != task.StatusDone {
		t.Fatalf("result status = %s, want done", result.Task.Status)
	}
}

func TestUpdateTaskStatusRejectsInvalidInput(t *testing.T) {
	service := NewService(&fakeRepository{})
	account := auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())

	_, err := service.UpdateTaskStatus(context.Background(), UpdateTaskStatusInput{
		Account:       account,
		TenantContext: tenantContext,
		TaskID:        taskID,
		Status:        task.StatusDone,
	})
	if !errors.Is(err, ErrProjectIDRequired) {
		t.Fatalf("err = %v, want ErrProjectIDRequired", err)
	}

	_, err = service.UpdateTaskStatus(context.Background(), UpdateTaskStatusInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		Status:        task.StatusDone,
	})
	if !errors.Is(err, ErrTaskIDRequired) {
		t.Fatalf("err = %v, want ErrTaskIDRequired", err)
	}

	_, err = service.UpdateTaskStatus(context.Background(), UpdateTaskStatusInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		Status:        task.Status("invalid"),
	})
	if !errors.Is(err, ErrTaskStatusInvalid) {
		t.Fatalf("err = %v, want ErrTaskStatusInvalid", err)
	}
}

func TestDeleteTask(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{findTask: &task.Task{
		ID:          taskID,
		TenantID:    tenantContext.TenantID,
		WorkspaceID: tenantContext.WorkspaceID,
		ProjectID:   projectID,
		Status:      task.StatusTodo,
	}}

	err := NewService(repo).DeleteTask(context.Background(), DeleteTaskInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
	})
	if err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}
	if repo.deleteTaskID != taskID {
		t.Fatalf("delete task ID = %s, want %s", repo.deleteTaskID, taskID)
	}
	if repo.activity == nil || repo.activity.Action != task.ActivityDeleted {
		t.Fatalf("activity = %#v, want deleted", repo.activity)
	}
}

func ptrString(value string) *string {
	return &value
}

func ptrUUIDPtr(value *uuid.UUID) **uuid.UUID {
	return &value
}

func testTenantContext() workspace.TenantContext {
	return workspace.TenantContext{
		TenantID:      uuid.Must(uuid.NewV7()),
		WorkspaceID:   uuid.Must(uuid.NewV7()),
		WorkspaceSlug: "team-one",
		MembershipID:  uuid.Must(uuid.NewV7()),
		Role:          workspace.WorkspaceRoleOwner,
	}
}
