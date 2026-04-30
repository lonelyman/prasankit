package tasksvc

import (
	"context"
	"errors"
	"testing"

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
	updatePatch       task.Patch
	updateErr         error
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

func (r *fakeRepository) ListTasks(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, _ int, _ int) ([]task.Task, int, error) {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	if r.taskErr != nil {
		return nil, 0, r.taskErr
	}
	return r.taskItems, r.taskTotal, nil
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
	repo := &fakeRepository{
		projectExists: true,
		taskTotal:     1,
		taskItems: []task.Task{
			{ID: uuid.Must(uuid.NewV7()), ProjectID: projectID, No: "TASK-0001", Title: "Task A"},
		},
	}

	result, err := NewService(repo).ListTasks(context.Background(), ListTasksInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		Limit:         10,
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

func TestDeleteTask(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{}

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
