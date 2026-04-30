package taskrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"prasankit-api/internal/modules/task"
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
	Count  int    `gorm:"column:count"`
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
		return task.ErrTaskNotFound
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
		counts = append(counts, task.StatusCount{Status: task.Status(row.Status), Count: row.Count})
	}
	return counts, nil
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
