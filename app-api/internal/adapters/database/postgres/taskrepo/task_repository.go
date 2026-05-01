package taskrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"prasankit-api/internal/modules/task"
	"prasankit-api/pkg/dbtypes"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

type taskRow struct {
	ID               uuid.UUID  `gorm:"column:id;type:uuid"`
	TenantID         uuid.UUID  `gorm:"column:tenant_id;type:uuid"`
	WorkspaceID      uuid.UUID  `gorm:"column:workspace_id;type:uuid"`
	ProjectID        uuid.UUID  `gorm:"column:project_id;type:uuid"`
	TaskNo           string     `gorm:"column:task_no"`
	TaskTitle        string     `gorm:"column:task_title"`
	Status           string     `gorm:"column:status"`
	PriorityID       uuid.UUID  `gorm:"column:priority_id;type:uuid"`
	PriorityCode     string     `gorm:"column:priority_code;->"`
	AssigneeMemberID *uuid.UUID `gorm:"column:assignee_member_id;type:uuid"`
	Description      string     `gorm:"column:description"`
	StartDate        *time.Time `gorm:"column:start_date"`
	DueDate          *time.Time `gorm:"column:due_date"`
	CompletedDate    *time.Time `gorm:"column:completed_date"`
	CreatedBy        uuid.UUID  `gorm:"column:created_by;type:uuid"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedBy        *uuid.UUID `gorm:"column:updated_by;type:uuid"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
	DeletedBy        *uuid.UUID `gorm:"column:deleted_by;type:uuid"`
	DeletedAt        *time.Time `gorm:"column:deleted_at"`
}

func (taskRow) TableName() string {
	return "tasks"
}

type priorityRow struct {
	ID   uuid.UUID `gorm:"column:id;type:uuid"`
	Code string    `gorm:"column:code"`
	Name string    `gorm:"column:name"`
}

type statusCountRow struct {
	Status string `gorm:"column:status"`
	Count  int64  `gorm:"column:count"`
}

type activityRow struct {
	ID             uuid.UUID     `gorm:"column:id;type:uuid"`
	TenantID       uuid.UUID     `gorm:"column:tenant_id;type:uuid"`
	WorkspaceID    uuid.UUID     `gorm:"column:workspace_id;type:uuid"`
	ProjectID      uuid.UUID     `gorm:"column:project_id;type:uuid"`
	TaskID         uuid.UUID     `gorm:"column:task_id;type:uuid"`
	ActorAccountID uuid.UUID     `gorm:"column:actor_account_id;type:uuid"`
	Action         string        `gorm:"column:action"`
	FromStatus     *string       `gorm:"column:from_status"`
	ToStatus       *string       `gorm:"column:to_status"`
	MetadataJSON   dbtypes.JSONB `gorm:"column:metadata_json;type:jsonb"`
	CreatedAt      time.Time     `gorm:"column:created_at"`
}

func (activityRow) TableName() string {
	return "task_activities"
}

type commentRow struct {
	ID          uuid.UUID  `gorm:"column:id;type:uuid"`
	TenantID    uuid.UUID  `gorm:"column:tenant_id;type:uuid"`
	WorkspaceID uuid.UUID  `gorm:"column:workspace_id;type:uuid"`
	ProjectID   uuid.UUID  `gorm:"column:project_id;type:uuid"`
	TaskID      uuid.UUID  `gorm:"column:task_id;type:uuid"`
	Body        string     `gorm:"column:body"`
	CreatedBy   uuid.UUID  `gorm:"column:created_by;type:uuid"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedBy   *uuid.UUID `gorm:"column:updated_by;type:uuid"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
	DeletedBy   *uuid.UUID `gorm:"column:deleted_by;type:uuid"`
	DeletedAt   *time.Time `gorm:"column:deleted_at"`
}

func (commentRow) TableName() string {
	return "task_comments"
}

type checklistItemRow struct {
	ID          uuid.UUID  `gorm:"column:id;type:uuid"`
	TenantID    uuid.UUID  `gorm:"column:tenant_id;type:uuid"`
	WorkspaceID uuid.UUID  `gorm:"column:workspace_id;type:uuid"`
	ProjectID   uuid.UUID  `gorm:"column:project_id;type:uuid"`
	TaskID      uuid.UUID  `gorm:"column:task_id;type:uuid"`
	ItemText    string     `gorm:"column:item_text"`
	IsCompleted bool       `gorm:"column:is_completed"`
	SortOrder   int        `gorm:"column:sort_order"`
	CreatedBy   uuid.UUID  `gorm:"column:created_by;type:uuid"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedBy   *uuid.UUID `gorm:"column:updated_by;type:uuid"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
	CompletedBy *uuid.UUID `gorm:"column:completed_by;type:uuid"`
	CompletedAt *time.Time `gorm:"column:completed_at"`
	DeletedBy   *uuid.UUID `gorm:"column:deleted_by;type:uuid"`
	DeletedAt   *time.Time `gorm:"column:deleted_at"`
}

func (checklistItemRow) TableName() string {
	return "task_checklist_items"
}

type attachmentRow struct {
	ID            uuid.UUID  `gorm:"column:id;type:uuid"`
	TenantID      uuid.UUID  `gorm:"column:tenant_id;type:uuid"`
	WorkspaceID   uuid.UUID  `gorm:"column:workspace_id;type:uuid"`
	ProjectID     uuid.UUID  `gorm:"column:project_id;type:uuid"`
	TaskID        uuid.UUID  `gorm:"column:task_id;type:uuid"`
	FileName      string     `gorm:"column:file_name"`
	ContentType   string     `gorm:"column:content_type"`
	SizeBytes     int64      `gorm:"column:size_bytes"`
	StorageBucket string     `gorm:"column:storage_bucket"`
	ObjectKey     string     `gorm:"column:object_key"`
	UploadStatus  string     `gorm:"column:upload_status"`
	UploadedBy    uuid.UUID  `gorm:"column:uploaded_by;type:uuid"`
	UploadedAt    *time.Time `gorm:"column:uploaded_at"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

func (attachmentRow) TableName() string {
	return "task_attachments"
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) WithinTransaction(ctx context.Context, fn func(context.Context, task.Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ctx, NewRepository(tx))
	})
}

func (r *Repository) ProjectExists(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("projects").
		Where("tenant_id = ?", tenantID).
		Where("workspace_id = ?", workspaceID).
		Where("id = ?", projectID).
		Where("deleted_at IS NULL").
		Count(&count).
		Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *Repository) FindPriorityByCode(ctx context.Context, code task.Priority) (*task.PriorityMaster, error) {
	var row priorityRow
	err := r.db.WithContext(ctx).
		Table("project_priorities").
		Select("id, code, name").
		Where("code = ?", string(code)).
		Where("status = ?", "active").
		Take(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, task.ErrPriorityNotFound
	}
	if err != nil {
		return nil, err
	}
	return &task.PriorityMaster{ID: row.ID, Code: task.Priority(row.Code), Name: row.Name}, nil
}

func (r *Repository) FindActiveProjectMemberByID(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, memberID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("project_members").
		Where("tenant_id = ?", tenantID).
		Where("workspace_id = ?", workspaceID).
		Where("project_id = ?", projectID).
		Where("id = ?", memberID).
		Where("status = ?", "active").
		Count(&count).
		Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *Repository) NextTaskNo(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, prefix string, numberLength int) (string, error) {
	counterID, err := ids.NewUUID()
	if err != nil {
		return "", err
	}

	var runningNo int
	err = r.db.WithContext(ctx).
		Raw(`
			INSERT INTO task_counters (
				id,
				tenant_id,
				workspace_id,
				project_id,
				prefix,
				running_no,
				number_length,
				last_generated_at,
				created_at,
				updated_at
			)
			VALUES (?, ?, ?, ?, ?, 1, ?, now(), now(), now())
			ON CONFLICT (tenant_id, workspace_id, project_id, prefix)
			DO UPDATE SET
				running_no = task_counters.running_no + 1,
				last_generated_at = now(),
				updated_at = now()
			RETURNING running_no
		`, counterID, tenantID, workspaceID, projectID, prefix, numberLength).
		Scan(&runningNo).
		Error
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%0*d", prefix, numberLength, runningNo), nil
}

func (r *Repository) CreateTask(ctx context.Context, item *task.Task) error {
	if err := ensureUUID(&item.ID); err != nil {
		return err
	}
	if item.Status == "" {
		item.Status = task.StatusTodo
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = item.CreatedAt
	}

	row := taskRow{
		ID:               item.ID,
		TenantID:         item.TenantID,
		WorkspaceID:      item.WorkspaceID,
		ProjectID:        item.ProjectID,
		TaskNo:           item.No,
		TaskTitle:        item.Title,
		Status:           string(item.Status),
		PriorityID:       item.PriorityID,
		AssigneeMemberID: item.AssigneeMemberID,
		Description:      item.Description,
		StartDate:        item.StartDate,
		DueDate:          item.DueDate,
		CompletedDate:    item.CompletedDate,
		CreatedBy:        item.CreatedBy,
		CreatedAt:        item.CreatedAt,
		UpdatedBy:        item.UpdatedBy,
		UpdatedAt:        item.UpdatedAt,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return mapCreateTaskError(err)
	}

	item.ID = row.ID
	item.CreatedAt = row.CreatedAt
	item.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *Repository) UpdateTask(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, patch task.Patch) error {
	updates := map[string]any{
		"updated_by": patch.UpdatedBy,
		"updated_at": patch.UpdatedAt,
	}
	if patch.Title != nil {
		updates["task_title"] = *patch.Title
	}
	if patch.PriorityID != nil {
		updates["priority_id"] = *patch.PriorityID
	}
	if patch.AssigneeMemberID != nil {
		updates["assignee_member_id"] = *patch.AssigneeMemberID
	}
	if patch.Description != nil {
		updates["description"] = *patch.Description
	}
	if patch.StartDate != nil {
		updates["start_date"] = *patch.StartDate
	}
	if patch.DueDate != nil {
		updates["due_date"] = *patch.DueDate
	}

	result := r.db.WithContext(ctx).
		Model(&taskRow{}).
		Where("tenant_id = ?", tenantID).
		Where("workspace_id = ?", workspaceID).
		Where("project_id = ?", projectID).
		Where("id = ?", taskID).
		Where("deleted_at IS NULL").
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return task.ErrTaskNotFound
	}
	return nil
}

func (r *Repository) UpdateTaskStatus(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, status task.Status, completedDate *time.Time, updatedBy uuid.UUID, updatedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&taskRow{}).
		Where("tenant_id = ?", tenantID).
		Where("workspace_id = ?", workspaceID).
		Where("project_id = ?", projectID).
		Where("id = ?", taskID).
		Where("deleted_at IS NULL").
		Updates(map[string]any{
			"status":         string(status),
			"completed_date": completedDate,
			"updated_by":     updatedBy,
			"updated_at":     updatedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return task.ErrAttachmentNotFound
	}
	return nil
}

func (r *Repository) SoftDeleteTask(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, deletedBy uuid.UUID) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&taskRow{}).
		Where("tenant_id = ?", tenantID).
		Where("workspace_id = ?", workspaceID).
		Where("project_id = ?", projectID).
		Where("id = ?", taskID).
		Where("deleted_at IS NULL").
		Updates(map[string]any{
			"deleted_by": deletedBy,
			"deleted_at": now,
			"updated_by": deletedBy,
			"updated_at": now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return task.ErrTaskNotFound
	}
	return nil
}

func (r *Repository) FindTaskByID(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID) (*task.Task, error) {
	var row taskRow
	err := r.db.WithContext(ctx).
		Table("tasks AS t").
		Select(taskSelect()).
		Joins("JOIN project_priorities AS pp ON pp.id = t.priority_id").
		Where("t.tenant_id = ?", tenantID).
		Where("t.workspace_id = ?", workspaceID).
		Where("t.project_id = ?", projectID).
		Where("t.id = ?", taskID).
		Where("t.deleted_at IS NULL").
		Take(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, task.ErrTaskNotFound
	}
	if err != nil {
		return nil, err
	}

	item := row.toDomain()
	return &item, nil
}

func (r *Repository) ListTasks(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, filter task.ListFilter, limit int, offset int) ([]task.Task, int, error) {
	var total int64
	countQuery := r.db.WithContext(ctx).
		Table("tasks AS t").
		Where("t.tenant_id = ?", tenantID).
		Where("t.workspace_id = ?", workspaceID).
		Where("t.project_id = ?", projectID).
		Where("t.deleted_at IS NULL")
	countQuery = applyListFilter(countQuery, filter)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []taskRow
	query := r.db.WithContext(ctx).
		Table("tasks AS t").
		Select(taskSelect()).
		Joins("JOIN project_priorities AS pp ON pp.id = t.priority_id").
		Where("t.tenant_id = ?", tenantID).
		Where("t.workspace_id = ?", workspaceID).
		Where("t.project_id = ?", projectID).
		Where("t.deleted_at IS NULL")
	err := applyListFilter(query, filter).
		Order("t.created_at DESC, t.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).
		Error
	if err != nil {
		return nil, 0, err
	}

	items := make([]task.Task, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toDomain())
	}
	return items, int(total), nil
}

func applyListFilter(query *gorm.DB, filter task.ListFilter) *gorm.DB {
	if filter.Status != nil {
		query = query.Where("t.status = ?", string(*filter.Status))
	}
	if filter.PriorityID != nil {
		query = query.Where("t.priority_id = ?", *filter.PriorityID)
	}
	if filter.AssigneeMemberID != nil {
		query = query.Where("t.assignee_member_id = ?", *filter.AssigneeMemberID)
	}
	return query
}

func (r *Repository) CountTasksByStatus(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID) ([]task.StatusCount, error) {
	var rows []statusCountRow
	err := r.db.WithContext(ctx).
		Table("tasks AS t").
		Select("t.status, COUNT(*) AS count").
		Where("t.tenant_id = ?", tenantID).
		Where("t.workspace_id = ?", workspaceID).
		Where("t.project_id = ?", projectID).
		Where("t.deleted_at IS NULL").
		Group("t.status").
		Scan(&rows).
		Error
	if err != nil {
		return nil, err
	}

	counts := make([]task.StatusCount, 0, len(rows))
	for _, row := range rows {
		counts = append(counts, task.StatusCount{Status: task.Status(row.Status), Count: int(row.Count)})
	}
	return counts, nil
}

func (r *Repository) CreateTaskActivity(ctx context.Context, activity *task.Activity) error {
	if err := ensureUUID(&activity.ID); err != nil {
		return err
	}
	if activity.CreatedAt.IsZero() {
		activity.CreatedAt = time.Now().UTC()
	}

	row := activityRow{
		ID:             activity.ID,
		TenantID:       activity.TenantID,
		WorkspaceID:    activity.WorkspaceID,
		ProjectID:      activity.ProjectID,
		TaskID:         activity.TaskID,
		ActorAccountID: activity.ActorAccountID,
		Action:         string(activity.Action),
		FromStatus:     statusStringPtr(activity.FromStatus),
		ToStatus:       statusStringPtr(activity.ToStatus),
		MetadataJSON:   dbtypes.NewJSONB(activity.MetadataJSON),
		CreatedAt:      activity.CreatedAt,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}

	activity.ID = row.ID
	activity.CreatedAt = row.CreatedAt
	return nil
}

func (r *Repository) ListTaskActivities(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, limit int, offset int) ([]task.Activity, int, error) {
	var total int64
	countQuery := r.db.WithContext(ctx).
		Table("task_activities AS ta").
		Where("ta.tenant_id = ?", tenantID).
		Where("ta.workspace_id = ?", workspaceID).
		Where("ta.project_id = ?", projectID).
		Where("ta.task_id = ?", taskID)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []activityRow
	err := r.db.WithContext(ctx).
		Table("task_activities AS ta").
		Where("ta.tenant_id = ?", tenantID).
		Where("ta.workspace_id = ?", workspaceID).
		Where("ta.project_id = ?", projectID).
		Where("ta.task_id = ?", taskID).
		Order("ta.created_at DESC, ta.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).
		Error
	if err != nil {
		return nil, 0, err
	}

	activities := make([]task.Activity, 0, len(rows))
	for _, row := range rows {
		activities = append(activities, row.toDomain())
	}
	return activities, int(total), nil
}

func (r *Repository) CreateTaskComment(ctx context.Context, comment *task.Comment) error {
	if err := ensureUUID(&comment.ID); err != nil {
		return err
	}
	if comment.CreatedAt.IsZero() {
		comment.CreatedAt = time.Now().UTC()
	}
	if comment.UpdatedAt.IsZero() {
		comment.UpdatedAt = comment.CreatedAt
	}

	row := commentRow{
		ID:          comment.ID,
		TenantID:    comment.TenantID,
		WorkspaceID: comment.WorkspaceID,
		ProjectID:   comment.ProjectID,
		TaskID:      comment.TaskID,
		Body:        comment.Body,
		CreatedBy:   comment.CreatedBy,
		CreatedAt:   comment.CreatedAt,
		UpdatedBy:   comment.UpdatedBy,
		UpdatedAt:   comment.UpdatedAt,
		DeletedBy:   comment.DeletedBy,
		DeletedAt:   comment.DeletedAt,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	comment.ID = row.ID
	comment.CreatedAt = row.CreatedAt
	comment.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *Repository) ListTaskComments(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, limit int, offset int) ([]task.Comment, int, error) {
	var total int64
	countQuery := r.db.WithContext(ctx).
		Table("task_comments AS tc").
		Where("tc.tenant_id = ?", tenantID).
		Where("tc.workspace_id = ?", workspaceID).
		Where("tc.project_id = ?", projectID).
		Where("tc.task_id = ?", taskID).
		Where("tc.deleted_at IS NULL")
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []commentRow
	err := r.db.WithContext(ctx).
		Table("task_comments AS tc").
		Where("tc.tenant_id = ?", tenantID).
		Where("tc.workspace_id = ?", workspaceID).
		Where("tc.project_id = ?", projectID).
		Where("tc.task_id = ?", taskID).
		Where("tc.deleted_at IS NULL").
		Order("tc.created_at DESC, tc.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).
		Error
	if err != nil {
		return nil, 0, err
	}

	comments := make([]task.Comment, 0, len(rows))
	for _, row := range rows {
		comments = append(comments, row.toDomain())
	}
	return comments, int(total), nil
}

func (r *Repository) SoftDeleteTaskComment(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, commentID uuid.UUID, deletedBy uuid.UUID, deletedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&commentRow{}).
		Where("tenant_id = ?", tenantID).
		Where("workspace_id = ?", workspaceID).
		Where("project_id = ?", projectID).
		Where("task_id = ?", taskID).
		Where("id = ?", commentID).
		Where("deleted_at IS NULL").
		Updates(map[string]any{
			"deleted_by": deletedBy,
			"deleted_at": deletedAt,
			"updated_by": deletedBy,
			"updated_at": deletedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return task.ErrCommentNotFound
	}
	return nil
}

func (r *Repository) CreateTaskChecklistItem(ctx context.Context, item *task.ChecklistItem) error {
	if err := ensureUUID(&item.ID); err != nil {
		return err
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = item.CreatedAt
	}

	row := checklistItemRow{
		ID:          item.ID,
		TenantID:    item.TenantID,
		WorkspaceID: item.WorkspaceID,
		ProjectID:   item.ProjectID,
		TaskID:      item.TaskID,
		ItemText:    item.Text,
		IsCompleted: item.IsCompleted,
		SortOrder:   item.SortOrder,
		CreatedBy:   item.CreatedBy,
		CreatedAt:   item.CreatedAt,
		UpdatedBy:   item.UpdatedBy,
		UpdatedAt:   item.UpdatedAt,
		CompletedBy: item.CompletedBy,
		CompletedAt: item.CompletedAt,
		DeletedBy:   item.DeletedBy,
		DeletedAt:   item.DeletedAt,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	item.ID = row.ID
	item.CreatedAt = row.CreatedAt
	item.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *Repository) UpdateTaskChecklistItem(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, itemID uuid.UUID, patch task.ChecklistItemPatch) error {
	updates := map[string]any{
		"updated_by": patch.UpdatedBy,
		"updated_at": patch.UpdatedAt,
	}
	if patch.Text != nil {
		updates["item_text"] = *patch.Text
	}
	if patch.SortOrder != nil {
		updates["sort_order"] = *patch.SortOrder
	}
	if patch.IsCompleted != nil {
		updates["is_completed"] = *patch.IsCompleted
		if *patch.IsCompleted {
			updates["completed_by"] = patch.UpdatedBy
			updates["completed_at"] = patch.UpdatedAt
		} else {
			updates["completed_by"] = nil
			updates["completed_at"] = nil
		}
	}

	result := r.db.WithContext(ctx).
		Model(&checklistItemRow{}).
		Where("tenant_id = ?", tenantID).
		Where("workspace_id = ?", workspaceID).
		Where("project_id = ?", projectID).
		Where("task_id = ?", taskID).
		Where("id = ?", itemID).
		Where("deleted_at IS NULL").
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return task.ErrChecklistItemNotFound
	}
	return nil
}

func (r *Repository) SoftDeleteTaskChecklistItem(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, itemID uuid.UUID, deletedBy uuid.UUID, deletedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&checklistItemRow{}).
		Where("tenant_id = ?", tenantID).
		Where("workspace_id = ?", workspaceID).
		Where("project_id = ?", projectID).
		Where("task_id = ?", taskID).
		Where("id = ?", itemID).
		Where("deleted_at IS NULL").
		Updates(map[string]any{
			"deleted_by": deletedBy,
			"deleted_at": deletedAt,
			"updated_by": deletedBy,
			"updated_at": deletedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return task.ErrChecklistItemNotFound
	}
	return nil
}

func (r *Repository) ListTaskChecklistItems(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID) ([]task.ChecklistItem, error) {
	var rows []checklistItemRow
	err := r.db.WithContext(ctx).
		Table("task_checklist_items AS tci").
		Where("tci.tenant_id = ?", tenantID).
		Where("tci.workspace_id = ?", workspaceID).
		Where("tci.project_id = ?", projectID).
		Where("tci.task_id = ?", taskID).
		Where("tci.deleted_at IS NULL").
		Order("tci.sort_order ASC, tci.created_at ASC, tci.id ASC").
		Scan(&rows).
		Error
	if err != nil {
		return nil, err
	}

	items := make([]task.ChecklistItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toDomain())
	}
	return items, nil
}

func (r *Repository) FindTaskChecklistItemByID(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, itemID uuid.UUID) (*task.ChecklistItem, error) {
	var row checklistItemRow
	err := r.db.WithContext(ctx).
		Table("task_checklist_items AS tci").
		Where("tci.tenant_id = ?", tenantID).
		Where("tci.workspace_id = ?", workspaceID).
		Where("tci.project_id = ?", projectID).
		Where("tci.task_id = ?", taskID).
		Where("tci.id = ?", itemID).
		Where("tci.deleted_at IS NULL").
		Take(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, task.ErrChecklistItemNotFound
	}
	if err != nil {
		return nil, err
	}
	item := row.toDomain()
	return &item, nil
}

func (r *Repository) CreateTaskAttachment(ctx context.Context, attachment *task.Attachment) error {
	if err := ensureUUID(&attachment.ID); err != nil {
		return err
	}
	if attachment.UploadStatus == "" {
		attachment.UploadStatus = task.AttachmentPending
	}
	if attachment.CreatedAt.IsZero() {
		attachment.CreatedAt = time.Now().UTC()
	}
	if attachment.UpdatedAt.IsZero() {
		attachment.UpdatedAt = attachment.CreatedAt
	}

	row := attachmentRow{
		ID:            attachment.ID,
		TenantID:      attachment.TenantID,
		WorkspaceID:   attachment.WorkspaceID,
		ProjectID:     attachment.ProjectID,
		TaskID:        attachment.TaskID,
		FileName:      attachment.FileName,
		ContentType:   attachment.ContentType,
		SizeBytes:     attachment.SizeBytes,
		StorageBucket: attachment.StorageBucket,
		ObjectKey:     attachment.ObjectKey,
		UploadStatus:  string(attachment.UploadStatus),
		UploadedBy:    attachment.UploadedBy,
		UploadedAt:    attachment.UploadedAt,
		CreatedAt:     attachment.CreatedAt,
		UpdatedAt:     attachment.UpdatedAt,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	attachment.ID = row.ID
	attachment.CreatedAt = row.CreatedAt
	attachment.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *Repository) MarkTaskAttachmentUploaded(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, attachmentID uuid.UUID, uploadedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&attachmentRow{}).
		Where("tenant_id = ?", tenantID).
		Where("workspace_id = ?", workspaceID).
		Where("project_id = ?", projectID).
		Where("task_id = ?", taskID).
		Where("id = ?", attachmentID).
		Updates(map[string]any{
			"upload_status": string(task.AttachmentUploaded),
			"uploaded_at":   uploadedAt,
			"updated_at":    uploadedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return task.ErrAttachmentNotFound
	}
	return nil
}

func (r *Repository) ListTaskAttachments(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, taskID uuid.UUID, limit int, offset int) ([]task.Attachment, int, error) {
	var total int64
	countQuery := r.db.WithContext(ctx).
		Table("task_attachments AS ta").
		Where("ta.tenant_id = ?", tenantID).
		Where("ta.workspace_id = ?", workspaceID).
		Where("ta.project_id = ?", projectID).
		Where("ta.task_id = ?", taskID)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []attachmentRow
	err := r.db.WithContext(ctx).
		Table("task_attachments AS ta").
		Where("ta.tenant_id = ?", tenantID).
		Where("ta.workspace_id = ?", workspaceID).
		Where("ta.project_id = ?", projectID).
		Where("ta.task_id = ?", taskID).
		Order("ta.created_at DESC, ta.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).
		Error
	if err != nil {
		return nil, 0, err
	}

	attachments := make([]task.Attachment, 0, len(rows))
	for _, row := range rows {
		attachments = append(attachments, row.toDomain())
	}
	return attachments, int(total), nil
}

func taskSelect() string {
	return `
		t.id,
		t.tenant_id,
		t.workspace_id,
		t.project_id,
		t.task_no,
		t.task_title,
		t.status,
		t.priority_id,
		pp.code AS priority_code,
		t.assignee_member_id,
		t.description,
		t.start_date,
		t.due_date,
		t.completed_date,
		t.created_by,
		t.created_at,
		t.updated_by,
		t.updated_at,
		t.deleted_by,
		t.deleted_at
	`
}

func statusStringPtr(value *task.Status) *string {
	if value == nil {
		return nil
	}
	status := string(*value)
	return &status
}

func parseStatusPtr(value *string) *task.Status {
	if value == nil {
		return nil
	}
	status := task.Status(*value)
	return &status
}

func ensureUUID(id *uuid.UUID) error {
	if *id != uuid.Nil {
		return nil
	}
	generated, err := ids.NewUUID()
	if err != nil {
		return err
	}
	*id = generated
	return nil
}

func mapCreateTaskError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if pgErr.ConstraintName == "uq_tasks_no_active" {
			return task.ErrTaskNoAlreadyTaken
		}
	}
	return err
}

func (r taskRow) toDomain() task.Task {
	return task.Task{
		ID:               r.ID,
		TenantID:         r.TenantID,
		WorkspaceID:      r.WorkspaceID,
		ProjectID:        r.ProjectID,
		No:               r.TaskNo,
		Title:            r.TaskTitle,
		Status:           task.Status(r.Status),
		PriorityID:       r.PriorityID,
		Priority:         task.Priority(r.PriorityCode),
		AssigneeMemberID: r.AssigneeMemberID,
		Description:      r.Description,
		StartDate:        r.StartDate,
		DueDate:          r.DueDate,
		CompletedDate:    r.CompletedDate,
		CreatedBy:        r.CreatedBy,
		CreatedAt:        r.CreatedAt,
		UpdatedBy:        r.UpdatedBy,
		UpdatedAt:        r.UpdatedAt,
		DeletedBy:        r.DeletedBy,
		DeletedAt:        r.DeletedAt,
	}
}

func (r activityRow) toDomain() task.Activity {
	return task.Activity{
		ID:             r.ID,
		TenantID:       r.TenantID,
		WorkspaceID:    r.WorkspaceID,
		ProjectID:      r.ProjectID,
		TaskID:         r.TaskID,
		ActorAccountID: r.ActorAccountID,
		Action:         task.ActivityAction(r.Action),
		FromStatus:     parseStatusPtr(r.FromStatus),
		ToStatus:       parseStatusPtr(r.ToStatus),
		MetadataJSON:   map[string]any(r.MetadataJSON),
		CreatedAt:      r.CreatedAt,
	}
}

func (r commentRow) toDomain() task.Comment {
	return task.Comment{
		ID:          r.ID,
		TenantID:    r.TenantID,
		WorkspaceID: r.WorkspaceID,
		ProjectID:   r.ProjectID,
		TaskID:      r.TaskID,
		Body:        r.Body,
		CreatedBy:   r.CreatedBy,
		CreatedAt:   r.CreatedAt,
		UpdatedBy:   r.UpdatedBy,
		UpdatedAt:   r.UpdatedAt,
		DeletedBy:   r.DeletedBy,
		DeletedAt:   r.DeletedAt,
	}
}

func (r checklistItemRow) toDomain() task.ChecklistItem {
	return task.ChecklistItem{
		ID:          r.ID,
		TenantID:    r.TenantID,
		WorkspaceID: r.WorkspaceID,
		ProjectID:   r.ProjectID,
		TaskID:      r.TaskID,
		Text:        r.ItemText,
		IsCompleted: r.IsCompleted,
		SortOrder:   r.SortOrder,
		CreatedBy:   r.CreatedBy,
		CreatedAt:   r.CreatedAt,
		UpdatedBy:   r.UpdatedBy,
		UpdatedAt:   r.UpdatedAt,
		CompletedBy: r.CompletedBy,
		CompletedAt: r.CompletedAt,
		DeletedBy:   r.DeletedBy,
		DeletedAt:   r.DeletedAt,
	}
}

func (r attachmentRow) toDomain() task.Attachment {
	return task.Attachment{
		ID:            r.ID,
		TenantID:      r.TenantID,
		WorkspaceID:   r.WorkspaceID,
		ProjectID:     r.ProjectID,
		TaskID:        r.TaskID,
		FileName:      r.FileName,
		ContentType:   r.ContentType,
		SizeBytes:     r.SizeBytes,
		StorageBucket: r.StorageBucket,
		ObjectKey:     r.ObjectKey,
		UploadStatus:  task.AttachmentStatus(r.UploadStatus),
		UploadedBy:    r.UploadedBy,
		UploadedAt:    r.UploadedAt,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}
