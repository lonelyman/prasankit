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
	comment           *task.Comment
	commentItems      []task.Comment
	commentTotal      int
	commentID         uuid.UUID
	commentErr        error
	checklistItem     *task.ChecklistItem
	checklistItems    []task.ChecklistItem
	checklistItemID   uuid.UUID
	checklistPatch    task.ChecklistItemPatch
	checklistErr      error
	attachment        *task.Attachment
	attachmentItems   []task.Attachment
	attachmentTotal   int
	attachmentID      uuid.UUID
	attachmentErr     error
	tag               *task.Tag
	tagItems          []task.Tag
	tagID             uuid.UUID
	tagAssignment     *task.TagAssignment
	tagErr            error
	relation          *task.Relation
	relationItems     []task.Relation
	relationID        uuid.UUID
	relationErr       error
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

func (r *fakeRepository) CreateTaskComment(_ context.Context, comment *task.Comment) error {
	if r.commentErr != nil {
		return r.commentErr
	}
	comment.ID = uuid.Must(uuid.NewV7())
	r.comment = comment
	return nil
}

func (r *fakeRepository) ListTaskComments(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, _ int, _ int) ([]task.Comment, int, error) {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	r.statusTaskID = taskID
	if r.commentErr != nil {
		return nil, 0, r.commentErr
	}
	return r.commentItems, r.commentTotal, nil
}

func (r *fakeRepository) SoftDeleteTaskComment(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, commentID uuid.UUID, _ uuid.UUID, _ time.Time) error {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	r.statusTaskID = taskID
	r.commentID = commentID
	return r.commentErr
}

func (r *fakeRepository) CreateTaskChecklistItem(_ context.Context, item *task.ChecklistItem) error {
	if r.checklistErr != nil {
		return r.checklistErr
	}
	item.ID = uuid.Must(uuid.NewV7())
	r.checklistItem = item
	return nil
}

func (r *fakeRepository) UpdateTaskChecklistItem(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, itemID uuid.UUID, patch task.ChecklistItemPatch) error {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	r.statusTaskID = taskID
	r.checklistItemID = itemID
	r.checklistPatch = patch
	return r.checklistErr
}

func (r *fakeRepository) SoftDeleteTaskChecklistItem(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, itemID uuid.UUID, _ uuid.UUID, _ time.Time) error {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	r.statusTaskID = taskID
	r.checklistItemID = itemID
	return r.checklistErr
}

func (r *fakeRepository) ListTaskChecklistItems(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID) ([]task.ChecklistItem, error) {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	r.statusTaskID = taskID
	if r.checklistErr != nil {
		return nil, r.checklistErr
	}
	return r.checklistItems, nil
}

func (r *fakeRepository) FindTaskChecklistItemByID(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, itemID uuid.UUID) (*task.ChecklistItem, error) {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	r.statusTaskID = taskID
	r.checklistItemID = itemID
	if r.checklistErr != nil {
		return nil, r.checklistErr
	}
	if r.checklistItem != nil {
		return r.checklistItem, nil
	}
	return nil, task.ErrChecklistItemNotFound
}

func (r *fakeRepository) CreateTaskAttachment(_ context.Context, attachment *task.Attachment) error {
	if r.attachmentErr != nil {
		return r.attachmentErr
	}
	r.attachment = attachment
	return nil
}

func (r *fakeRepository) MarkTaskAttachmentUploaded(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, attachmentID uuid.UUID, _ time.Time) error {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	r.statusTaskID = taskID
	r.attachmentID = attachmentID
	return r.attachmentErr
}

func (r *fakeRepository) ListTaskAttachments(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, _ int, _ int) ([]task.Attachment, int, error) {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	r.statusTaskID = taskID
	if r.attachmentErr != nil {
		return nil, 0, r.attachmentErr
	}
	return r.attachmentItems, r.attachmentTotal, nil
}

func (r *fakeRepository) FindTaskAttachmentByID(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, attachmentID uuid.UUID) (*task.Attachment, error) {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	r.statusTaskID = taskID
	r.attachmentID = attachmentID
	if r.attachmentErr != nil {
		return nil, r.attachmentErr
	}
	if r.attachment != nil {
		return r.attachment, nil
	}
	return nil, task.ErrAttachmentNotFound
}

func (r *fakeRepository) SoftDeleteTaskAttachment(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, attachmentID uuid.UUID, _ uuid.UUID, _ time.Time) error {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	r.statusTaskID = taskID
	r.attachmentID = attachmentID
	return r.attachmentErr
}

func (r *fakeRepository) FindOrCreateTaskTag(_ context.Context, tag *task.Tag) (*task.Tag, error) {
	if r.tagErr != nil {
		return nil, r.tagErr
	}
	if r.tag != nil {
		return r.tag, nil
	}
	tag.ID = uuid.Must(uuid.NewV7())
	r.tag = tag
	return tag, nil
}

func (r *fakeRepository) AssignTaskTag(_ context.Context, assignment *task.TagAssignment) error {
	if r.tagErr != nil {
		return r.tagErr
	}
	assignment.ID = uuid.Must(uuid.NewV7())
	r.tagAssignment = assignment
	r.tagID = assignment.TagID
	return nil
}

func (r *fakeRepository) ListTaskTags(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID) ([]task.Tag, error) {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	r.statusTaskID = taskID
	if r.tagErr != nil {
		return nil, r.tagErr
	}
	return r.tagItems, nil
}

func (r *fakeRepository) RemoveTaskTag(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, tagID uuid.UUID, _ uuid.UUID, _ time.Time) error {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	r.statusTaskID = taskID
	r.tagID = tagID
	return r.tagErr
}

func (r *fakeRepository) CreateTaskRelation(_ context.Context, relation *task.Relation) error {
	if r.relationErr != nil {
		return r.relationErr
	}
	relation.ID = uuid.Must(uuid.NewV7())
	r.relation = relation
	r.relationID = relation.ID
	return nil
}

func (r *fakeRepository) ListTaskRelations(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID) ([]task.Relation, error) {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	r.statusTaskID = taskID
	if r.relationErr != nil {
		return nil, r.relationErr
	}
	return r.relationItems, nil
}

func (r *fakeRepository) SoftDeleteTaskRelation(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, relationID uuid.UUID, _ uuid.UUID, _ time.Time) error {
	r.tenantID = tenantID
	r.workspaceID = workspaceID
	r.projectID = projectID
	r.statusTaskID = taskID
	r.relationID = relationID
	return r.relationErr
}

type fakeAttachmentStorage struct {
	bucket      string
	err         error
	key         string
	downloadKey string
}

func (s *fakeAttachmentStorage) Bucket() string {
	if s.bucket == "" {
		return "prasankit"
	}
	return s.bucket
}

func (s *fakeAttachmentStorage) PresignUpload(_ context.Context, objectKey string, _ string) (*task.AttachmentUploadURL, error) {
	s.key = objectKey
	if s.err != nil {
		return nil, s.err
	}
	return &task.AttachmentUploadURL{URL: "http://minio/upload", ExpiresAt: time.Now().UTC().Add(15 * time.Minute)}, nil
}

func (s *fakeAttachmentStorage) PresignDownload(_ context.Context, objectKey string, _ string, _ string) (*task.AttachmentDownloadURL, error) {
	s.downloadKey = objectKey
	if s.err != nil {
		return nil, s.err
	}
	return &task.AttachmentDownloadURL{URL: "http://minio/download", ExpiresAt: time.Now().UTC().Add(15 * time.Minute)}, nil
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
	tagID := uuid.Must(uuid.NewV7())
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
		TagID:            &tagID,
		Search:           "  proposal  ",
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
	if repo.listFilter.TagID == nil || *repo.listFilter.TagID != tagID {
		t.Fatalf("tag filter = %#v, want %s", repo.listFilter.TagID, tagID)
	}
	if repo.listFilter.Search != "proposal" {
		t.Fatalf("search filter = %q, want proposal", repo.listFilter.Search)
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

func TestCreateTaskComment(t *testing.T) {
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

	result, err := NewService(repo).CreateTaskComment(context.Background(), CreateTaskCommentInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		Body:          "  please review  ",
	})
	if err != nil {
		t.Fatalf("CreateTaskComment: %v", err)
	}
	if result.Comment.ID == uuid.Nil {
		t.Fatal("comment ID was not set")
	}
	if result.Comment.Body != "please review" {
		t.Fatalf("comment body = %q, want trimmed", result.Comment.Body)
	}
	if repo.activity == nil || repo.activity.MetadataJSON["action"] != "comment_created" {
		t.Fatalf("activity metadata = %#v, want comment_created", repo.activity)
	}
}

func TestCreateTaskCommentRejectsBlankBody(t *testing.T) {
	_, err := NewService(&fakeRepository{}).CreateTaskComment(context.Background(), CreateTaskCommentInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: testTenantContext(),
		ProjectID:     uuid.Must(uuid.NewV7()),
		TaskID:        uuid.Must(uuid.NewV7()),
		Body:          "   ",
	})
	if !errors.Is(err, ErrTaskCommentBodyRequired) {
		t.Fatalf("err = %v, want ErrTaskCommentBodyRequired", err)
	}
}

func TestListTaskComments(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{
		findTask:     &task.Task{ID: taskID, ProjectID: projectID, No: "TASK-0001", Title: "Task A"},
		commentTotal: 1,
		commentItems: []task.Comment{
			{ID: uuid.Must(uuid.NewV7()), ProjectID: projectID, TaskID: taskID, Body: "hello"},
		},
	}

	result, err := NewService(repo).ListTaskComments(context.Background(), ListTaskCommentsInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("ListTaskComments: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("result = total %d len %d, want 1/1", result.Total, len(result.Items))
	}
}

func TestDeleteTaskComment(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	commentID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{findTask: &task.Task{
		ID:          taskID,
		TenantID:    tenantContext.TenantID,
		WorkspaceID: tenantContext.WorkspaceID,
		ProjectID:   projectID,
		Status:      task.StatusTodo,
	}}

	err := NewService(repo).DeleteTaskComment(context.Background(), DeleteTaskCommentInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		CommentID:     commentID,
	})
	if err != nil {
		t.Fatalf("DeleteTaskComment: %v", err)
	}
	if repo.commentID != commentID {
		t.Fatalf("comment ID = %s, want %s", repo.commentID, commentID)
	}
	if repo.activity == nil || repo.activity.MetadataJSON["action"] != "comment_deleted" {
		t.Fatalf("activity metadata = %#v, want comment_deleted", repo.activity)
	}
}

func TestCreateTaskChecklistItem(t *testing.T) {
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

	result, err := NewService(repo).CreateTaskChecklistItem(context.Background(), CreateTaskChecklistItemInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		Text:          "  prepare document  ",
		SortOrder:     2,
	})
	if err != nil {
		t.Fatalf("CreateTaskChecklistItem: %v", err)
	}
	if result.Item.ID == uuid.Nil {
		t.Fatal("checklist item ID was not set")
	}
	if result.Item.Text != "prepare document" || result.Item.SortOrder != 2 {
		t.Fatalf("item = %#v, want trimmed text and sort order 2", result.Item)
	}
	if repo.activity == nil || repo.activity.MetadataJSON["action"] != "checklist_item_created" {
		t.Fatalf("activity metadata = %#v, want checklist_item_created", repo.activity)
	}
}

func TestListTaskChecklistItems(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{
		findTask: &task.Task{ID: taskID, ProjectID: projectID, No: "TASK-0001", Title: "Task A"},
		checklistItems: []task.ChecklistItem{
			{ID: uuid.Must(uuid.NewV7()), ProjectID: projectID, TaskID: taskID, Text: "prepare document"},
		},
	}

	result, err := NewService(repo).ListTaskChecklistItems(context.Background(), ListTaskChecklistItemsInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
	})
	if err != nil {
		t.Fatalf("ListTaskChecklistItems: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("len = %d, want 1", len(result.Items))
	}
}

func TestUpdateTaskChecklistItem(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	itemID := uuid.Must(uuid.NewV7())
	text := "prepare final document"
	completed := true
	sortOrder := 1
	repo := &fakeRepository{
		findTask:      &task.Task{ID: taskID, ProjectID: projectID, No: "TASK-0001", Title: "Task A"},
		checklistItem: &task.ChecklistItem{ID: itemID, ProjectID: projectID, TaskID: taskID, Text: "prepare document"},
	}

	result, err := NewService(repo).UpdateTaskChecklistItem(context.Background(), UpdateTaskChecklistItemInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		ItemID:        itemID,
		Text:          &text,
		IsCompleted:   &completed,
		SortOrder:     &sortOrder,
	})
	if err != nil {
		t.Fatalf("UpdateTaskChecklistItem: %v", err)
	}
	if repo.checklistPatch.Text == nil || *repo.checklistPatch.Text != text {
		t.Fatalf("text patch = %#v, want %q", repo.checklistPatch.Text, text)
	}
	if repo.checklistPatch.IsCompleted == nil || !*repo.checklistPatch.IsCompleted {
		t.Fatalf("completed patch = %#v, want true", repo.checklistPatch.IsCompleted)
	}
	if result.Item.ID != itemID {
		t.Fatalf("item ID = %s, want %s", result.Item.ID, itemID)
	}
}

func TestDeleteTaskChecklistItem(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	itemID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{findTask: &task.Task{
		ID:          taskID,
		TenantID:    tenantContext.TenantID,
		WorkspaceID: tenantContext.WorkspaceID,
		ProjectID:   projectID,
		Status:      task.StatusTodo,
	}}

	err := NewService(repo).DeleteTaskChecklistItem(context.Background(), DeleteTaskChecklistItemInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		ItemID:        itemID,
	})
	if err != nil {
		t.Fatalf("DeleteTaskChecklistItem: %v", err)
	}
	if repo.checklistItemID != itemID {
		t.Fatalf("item ID = %s, want %s", repo.checklistItemID, itemID)
	}
	if repo.activity == nil || repo.activity.MetadataJSON["action"] != "checklist_item_deleted" {
		t.Fatalf("activity metadata = %#v, want checklist_item_deleted", repo.activity)
	}
}

func TestCreateTaskAttachmentUpload(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	storage := &fakeAttachmentStorage{bucket: "prasankit"}
	repo := &fakeRepository{findTask: &task.Task{
		ID:          taskID,
		TenantID:    tenantContext.TenantID,
		WorkspaceID: tenantContext.WorkspaceID,
		ProjectID:   projectID,
		Status:      task.StatusTodo,
	}}

	result, err := NewService(repo, WithAttachmentStorage(storage)).CreateTaskAttachmentUpload(context.Background(), CreateTaskAttachmentUploadInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		FileName:      "spec.pdf",
		ContentType:   "application/pdf",
		SizeBytes:     1024,
	})
	if err != nil {
		t.Fatalf("CreateTaskAttachmentUpload: %v", err)
	}
	if result.Attachment.ID == uuid.Nil {
		t.Fatal("attachment ID was not set")
	}
	if result.Attachment.UploadStatus != task.AttachmentPending {
		t.Fatalf("status = %s, want pending", result.Attachment.UploadStatus)
	}
	if storage.key == "" || result.UploadURL.URL == "" {
		t.Fatalf("storage key/url = %q/%q, want non-empty", storage.key, result.UploadURL.URL)
	}
}

func TestCompleteTaskAttachmentUpload(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	attachmentID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{findTask: &task.Task{
		ID:          taskID,
		TenantID:    tenantContext.TenantID,
		WorkspaceID: tenantContext.WorkspaceID,
		ProjectID:   projectID,
		Status:      task.StatusTodo,
	}}

	err := NewService(repo).CompleteTaskAttachmentUpload(context.Background(), CompleteTaskAttachmentUploadInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		AttachmentID:  attachmentID,
	})
	if err != nil {
		t.Fatalf("CompleteTaskAttachmentUpload: %v", err)
	}
	if repo.attachmentID != attachmentID {
		t.Fatalf("attachment ID = %s, want %s", repo.attachmentID, attachmentID)
	}
	if repo.activity == nil || repo.activity.MetadataJSON["action"] != "attachment_uploaded" {
		t.Fatalf("activity metadata = %#v, want attachment_uploaded", repo.activity)
	}
}

func TestListTaskAttachments(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{
		findTask:        &task.Task{ID: taskID, ProjectID: projectID, No: "TASK-0001", Title: "Task A"},
		attachmentTotal: 1,
		attachmentItems: []task.Attachment{
			{ID: uuid.Must(uuid.NewV7()), ProjectID: projectID, TaskID: taskID, FileName: "spec.pdf"},
		},
	}

	result, err := NewService(repo).ListTaskAttachments(context.Background(), ListTaskAttachmentsInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("ListTaskAttachments: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("result = total %d len %d, want 1/1", result.Total, len(result.Items))
	}
}

func TestGetTaskAttachmentDownloadURL(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	attachmentID := uuid.Must(uuid.NewV7())
	storage := &fakeAttachmentStorage{bucket: "prasankit"}
	repo := &fakeRepository{
		findTask: &task.Task{ID: taskID, ProjectID: projectID, No: "TASK-0001", Title: "Task A"},
		attachment: &task.Attachment{
			ID:            attachmentID,
			ProjectID:     projectID,
			TaskID:        taskID,
			FileName:      "spec.pdf",
			ContentType:   "application/pdf",
			ObjectKey:     "tenant/workspace/project/task/attachments/file.pdf",
			UploadStatus:  task.AttachmentUploaded,
			StorageBucket: "prasankit",
		},
	}

	result, err := NewService(repo, WithAttachmentStorage(storage)).GetTaskAttachmentDownloadURL(context.Background(), GetTaskAttachmentDownloadURLInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		AttachmentID:  attachmentID,
	})
	if err != nil {
		t.Fatalf("GetTaskAttachmentDownloadURL: %v", err)
	}
	if result.DownloadURL.URL == "" || storage.downloadKey == "" {
		t.Fatalf("download url/key = %q/%q, want non-empty", result.DownloadURL.URL, storage.downloadKey)
	}
}

func TestGetTaskAttachmentDownloadURLRejectsPendingUpload(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	attachmentID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{
		findTask: &task.Task{ID: taskID, ProjectID: projectID, No: "TASK-0001", Title: "Task A"},
		attachment: &task.Attachment{
			ID:           attachmentID,
			ProjectID:    projectID,
			TaskID:       taskID,
			FileName:     "spec.pdf",
			UploadStatus: task.AttachmentPending,
		},
	}

	_, err := NewService(repo, WithAttachmentStorage(&fakeAttachmentStorage{})).GetTaskAttachmentDownloadURL(context.Background(), GetTaskAttachmentDownloadURLInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		AttachmentID:  attachmentID,
	})
	if !errors.Is(err, ErrAttachmentNotUploaded) {
		t.Fatalf("err = %v, want ErrAttachmentNotUploaded", err)
	}
}

func TestDeleteTaskAttachment(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	attachmentID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{findTask: &task.Task{
		ID:          taskID,
		TenantID:    tenantContext.TenantID,
		WorkspaceID: tenantContext.WorkspaceID,
		ProjectID:   projectID,
		Status:      task.StatusTodo,
	}}

	err := NewService(repo).DeleteTaskAttachment(context.Background(), DeleteTaskAttachmentInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		AttachmentID:  attachmentID,
	})
	if err != nil {
		t.Fatalf("DeleteTaskAttachment: %v", err)
	}
	if repo.attachmentID != attachmentID {
		t.Fatalf("attachment ID = %s, want %s", repo.attachmentID, attachmentID)
	}
	if repo.activity == nil || repo.activity.MetadataJSON["action"] != "attachment_deleted" {
		t.Fatalf("activity metadata = %#v, want attachment_deleted", repo.activity)
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

func TestAssignTaskTag(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	color := " #2f80ed "
	repo := &fakeRepository{findTask: &task.Task{ID: taskID, ProjectID: projectID, Status: task.StatusTodo}}

	result, err := NewService(repo).AssignTaskTag(context.Background(), AssignTaskTagInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		Name:          " Review Needed ",
		Color:         &color,
	})
	if err != nil {
		t.Fatalf("AssignTaskTag: %v", err)
	}
	if result.Tag.Name != "Review Needed" {
		t.Fatalf("tag name = %q, want trimmed", result.Tag.Name)
	}
	if result.Tag.NormalizedName != "review needed" {
		t.Fatalf("normalized name = %q, want review needed", result.Tag.NormalizedName)
	}
	if repo.tagAssignment == nil || repo.tagAssignment.TaskID != taskID {
		t.Fatalf("assignment = %#v, want task assignment", repo.tagAssignment)
	}
	if repo.activity == nil || repo.activity.MetadataJSON["action"] != "tag_assigned" {
		t.Fatalf("activity metadata = %#v, want tag_assigned", repo.activity)
	}
}

func TestAssignTaskTagRejectsBlankName(t *testing.T) {
	_, err := NewService(&fakeRepository{}).AssignTaskTag(context.Background(), AssignTaskTagInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: testTenantContext(),
		ProjectID:     uuid.Must(uuid.NewV7()),
		TaskID:        uuid.Must(uuid.NewV7()),
		Name:          " ",
	})
	if !errors.Is(err, ErrTaskTagNameRequired) {
		t.Fatalf("err = %v, want ErrTaskTagNameRequired", err)
	}
}

func TestListTaskTags(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	tagID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{
		findTask: &task.Task{ID: taskID, ProjectID: projectID, Status: task.StatusTodo},
		tagItems: []task.Tag{
			{ID: tagID, ProjectID: projectID, Name: "Review", CreatedBy: uuid.Must(uuid.NewV7()), CreatedAt: time.Now().UTC()},
		},
	}

	result, err := NewService(repo).ListTaskTags(context.Background(), ListTaskTagsInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
	})
	if err != nil {
		t.Fatalf("ListTaskTags: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].ID != tagID {
		t.Fatalf("items = %#v, want tag", result.Items)
	}
}

func TestRemoveTaskTag(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	tagID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{findTask: &task.Task{ID: taskID, ProjectID: projectID, Status: task.StatusTodo}}

	err := NewService(repo).RemoveTaskTag(context.Background(), RemoveTaskTagInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		TagID:         tagID,
	})
	if err != nil {
		t.Fatalf("RemoveTaskTag: %v", err)
	}
	if repo.tagID != tagID {
		t.Fatalf("tag ID = %s, want %s", repo.tagID, tagID)
	}
	if repo.activity == nil || repo.activity.MetadataJSON["action"] != "tag_removed" {
		t.Fatalf("activity metadata = %#v, want tag_removed", repo.activity)
	}
}

func TestCreateTaskRelation(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	targetTaskID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{findTask: &task.Task{ID: taskID, ProjectID: projectID, Status: task.StatusTodo}}

	result, err := NewService(repo).CreateTaskRelation(context.Background(), CreateTaskRelationInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		TargetTaskID:  targetTaskID,
		Type:          task.RelationBlocks,
	})
	if err != nil {
		t.Fatalf("CreateTaskRelation: %v", err)
	}
	if result.Relation.TargetTaskID != targetTaskID {
		t.Fatalf("target task ID = %s, want %s", result.Relation.TargetTaskID, targetTaskID)
	}
	if repo.activity == nil || repo.activity.MetadataJSON["action"] != "relation_created" {
		t.Fatalf("activity metadata = %#v, want relation_created", repo.activity)
	}
}

func TestCreateTaskRelationRejectsInvalidInput(t *testing.T) {
	service := NewService(&fakeRepository{})
	account := auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())

	_, err := service.CreateTaskRelation(context.Background(), CreateTaskRelationInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
	})
	if !errors.Is(err, ErrTaskRelationTargetRequired) {
		t.Fatalf("err = %v, want ErrTaskRelationTargetRequired", err)
	}

	_, err = service.CreateTaskRelation(context.Background(), CreateTaskRelationInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		TargetTaskID:  taskID,
		Type:          task.RelationBlocks,
	})
	if !errors.Is(err, ErrTaskRelationSelf) {
		t.Fatalf("err = %v, want ErrTaskRelationSelf", err)
	}

	_, err = service.CreateTaskRelation(context.Background(), CreateTaskRelationInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		TargetTaskID:  uuid.Must(uuid.NewV7()),
		Type:          task.RelationType("invalid"),
	})
	if !errors.Is(err, ErrTaskRelationTypeInvalid) {
		t.Fatalf("err = %v, want ErrTaskRelationTypeInvalid", err)
	}
}

func TestListTaskRelations(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	relationID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{
		findTask: &task.Task{ID: taskID, ProjectID: projectID, Status: task.StatusTodo},
		relationItems: []task.Relation{
			{ID: relationID, ProjectID: projectID, SourceTaskID: taskID, TargetTaskID: uuid.Must(uuid.NewV7()), Type: task.RelationRelatesTo},
		},
	}

	result, err := NewService(repo).ListTaskRelations(context.Background(), ListTaskRelationsInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
	})
	if err != nil {
		t.Fatalf("ListTaskRelations: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].ID != relationID {
		t.Fatalf("items = %#v, want relation", result.Items)
	}
}

func TestDeleteTaskRelation(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	taskID := uuid.Must(uuid.NewV7())
	relationID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{findTask: &task.Task{ID: taskID, ProjectID: projectID, Status: task.StatusTodo}}

	err := NewService(repo).DeleteTaskRelation(context.Background(), DeleteTaskRelationInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		TaskID:        taskID,
		RelationID:    relationID,
	})
	if err != nil {
		t.Fatalf("DeleteTaskRelation: %v", err)
	}
	if repo.relationID != relationID {
		t.Fatalf("relation ID = %s, want %s", repo.relationID, relationID)
	}
	if repo.activity == nil || repo.activity.MetadataJSON["action"] != "relation_deleted" {
		t.Fatalf("activity metadata = %#v, want relation_deleted", repo.activity)
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
