package projectrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"prasankit-api/internal/modules/project"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

type projectRow struct {
	ID                     uuid.UUID  `gorm:"column:id;type:uuid"`
	TenantID               uuid.UUID  `gorm:"column:tenant_id;type:uuid"`
	WorkspaceID            uuid.UUID  `gorm:"column:workspace_id;type:uuid"`
	ProjectCode            string     `gorm:"column:project_code"`
	ProjectName            string     `gorm:"column:project_name"`
	ProjectType            string     `gorm:"column:project_type"`
	ProjectStatus          string     `gorm:"column:project_status"`
	PriorityID             uuid.UUID  `gorm:"column:priority_id;type:uuid"`
	Description            string     `gorm:"column:description"`
	ClientOrRequestingUnit string     `gorm:"column:client_or_requesting_unit"`
	ScopeOrObjective       string     `gorm:"column:scope_or_objective"`
	CreatedBy              uuid.UUID  `gorm:"column:created_by;type:uuid"`
	CreatedAt              time.Time  `gorm:"column:created_at"`
	UpdatedBy              *uuid.UUID `gorm:"column:updated_by;type:uuid"`
	UpdatedAt              time.Time  `gorm:"column:updated_at"`
	DeletedBy              *uuid.UUID `gorm:"column:deleted_by;type:uuid"`
	DeletedAt              *time.Time `gorm:"column:deleted_at"`
	ArchivedAt             *time.Time `gorm:"column:archived_at"`
}

func (projectRow) TableName() string {
	return "projects"
}

type projectMemberRow struct {
	ID                    uuid.UUID  `gorm:"column:id;type:uuid"`
	TenantID              uuid.UUID  `gorm:"column:tenant_id;type:uuid"`
	WorkspaceID           uuid.UUID  `gorm:"column:workspace_id;type:uuid"`
	ProjectID             uuid.UUID  `gorm:"column:project_id;type:uuid"`
	WorkspaceMembershipID uuid.UUID  `gorm:"column:workspace_membership_id;type:uuid"`
	ProfileID             *uuid.UUID `gorm:"column:profile_id;type:uuid"`
	UserAccountID         *uuid.UUID `gorm:"column:user_account_id;type:uuid"`
	ProjectRoleID         uuid.UUID  `gorm:"column:project_role_id;type:uuid"`
	ProjectRoleCode       string     `gorm:"column:project_role_code;->"`
	Status                string     `gorm:"column:status"`
	JoinedAt              *time.Time `gorm:"column:joined_at"`
	RemovedAt             *time.Time `gorm:"column:removed_at"`
	CreatedBy             *uuid.UUID `gorm:"column:created_by;type:uuid"`
	CreatedAt             time.Time  `gorm:"column:created_at"`
	UpdatedBy             *uuid.UUID `gorm:"column:updated_by;type:uuid"`
	UpdatedAt             time.Time  `gorm:"column:updated_at"`
}

func (projectMemberRow) TableName() string {
	return "project_members"
}

type projectRoleRow struct {
	ID          uuid.UUID `gorm:"column:id;type:uuid"`
	Code        string    `gorm:"column:code"`
	Name        string    `gorm:"column:name"`
	Description string    `gorm:"column:description"`
	SortOrder   int       `gorm:"column:sort_order"`
	IsSystem    bool      `gorm:"column:is_system"`
	Status      string    `gorm:"column:status"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (projectRoleRow) TableName() string {
	return "project_roles"
}

type projectPriorityRow struct {
	ID        uuid.UUID `gorm:"column:id;type:uuid"`
	Code      string    `gorm:"column:code"`
	Name      string    `gorm:"column:name"`
	SortOrder int       `gorm:"column:sort_order"`
	IsSystem  bool      `gorm:"column:is_system"`
	Status    string    `gorm:"column:status"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (projectPriorityRow) TableName() string {
	return "project_priorities"
}

type projectPositionRow struct {
	ID          uuid.UUID `gorm:"column:id;type:uuid"`
	Code        string    `gorm:"column:code"`
	Name        string    `gorm:"column:name"`
	Description string    `gorm:"column:description"`
	SortOrder   int       `gorm:"column:sort_order"`
	IsSystem    bool      `gorm:"column:is_system"`
	Status      string    `gorm:"column:status"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (projectPositionRow) TableName() string {
	return "project_positions"
}

type projectMemberPositionRow struct {
	ID              uuid.UUID  `gorm:"column:id;type:uuid"`
	TenantID        uuid.UUID  `gorm:"column:tenant_id;type:uuid"`
	WorkspaceID     uuid.UUID  `gorm:"column:workspace_id;type:uuid"`
	ProjectID       uuid.UUID  `gorm:"column:project_id;type:uuid"`
	ProjectMemberID uuid.UUID  `gorm:"column:project_member_id;type:uuid"`
	PositionID      uuid.UUID  `gorm:"column:position_id;type:uuid"`
	PositionCode    string     `gorm:"column:position_code;->"`
	PositionName    string     `gorm:"column:position_name;->"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	CreatedBy       *uuid.UUID `gorm:"column:created_by;type:uuid"`
}

func (projectMemberPositionRow) TableName() string {
	return "project_member_positions"
}

type workspaceMemberCandidateRow struct {
	MembershipID  uuid.UUID  `gorm:"column:membership_id"`
	TenantID      uuid.UUID  `gorm:"column:tenant_id"`
	WorkspaceID   uuid.UUID  `gorm:"column:workspace_id"`
	ProfileID     *uuid.UUID `gorm:"column:profile_id"`
	UserAccountID *uuid.UUID `gorm:"column:user_account_id"`
}

type projectListRow struct {
	ProjectID                     uuid.UUID  `gorm:"column:project_id"`
	ProjectTenantID               uuid.UUID  `gorm:"column:project_tenant_id"`
	ProjectWorkspaceID            uuid.UUID  `gorm:"column:project_workspace_id"`
	ProjectCode                   string     `gorm:"column:project_code"`
	ProjectName                   string     `gorm:"column:project_name"`
	ProjectType                   string     `gorm:"column:project_type"`
	ProjectStatus                 string     `gorm:"column:project_status"`
	ProjectPriorityID             uuid.UUID  `gorm:"column:project_priority_id"`
	ProjectPriorityCode           string     `gorm:"column:project_priority_code"`
	ProjectDescription            string     `gorm:"column:project_description"`
	ProjectClientOrRequestingUnit string     `gorm:"column:project_client_or_requesting_unit"`
	ProjectScopeOrObjective       string     `gorm:"column:project_scope_or_objective"`
	ProjectCreatedBy              uuid.UUID  `gorm:"column:project_created_by"`
	ProjectCreatedAt              time.Time  `gorm:"column:project_created_at"`
	ProjectUpdatedBy              *uuid.UUID `gorm:"column:project_updated_by"`
	ProjectUpdatedAt              time.Time  `gorm:"column:project_updated_at"`
	ProjectDeletedBy              *uuid.UUID `gorm:"column:project_deleted_by"`
	ProjectDeletedAt              *time.Time `gorm:"column:project_deleted_at"`
	ProjectArchivedAt             *time.Time `gorm:"column:project_archived_at"`
	MemberID                      uuid.UUID  `gorm:"column:member_id"`
	MemberTenantID                uuid.UUID  `gorm:"column:member_tenant_id"`
	MemberWorkspaceID             uuid.UUID  `gorm:"column:member_workspace_id"`
	MemberProjectID               uuid.UUID  `gorm:"column:member_project_id"`
	MemberWorkspaceMembershipID   uuid.UUID  `gorm:"column:member_workspace_membership_id"`
	MemberProfileID               *uuid.UUID `gorm:"column:member_profile_id"`
	MemberUserAccountID           *uuid.UUID `gorm:"column:member_user_account_id"`
	MemberProjectRoleID           uuid.UUID  `gorm:"column:member_project_role_id"`
	MemberProjectRoleCode         string     `gorm:"column:member_project_role_code"`
	MemberStatus                  string     `gorm:"column:member_status"`
	MemberJoinedAt                *time.Time `gorm:"column:member_joined_at"`
	MemberRemovedAt               *time.Time `gorm:"column:member_removed_at"`
	MemberCreatedBy               *uuid.UUID `gorm:"column:member_created_by"`
	MemberCreatedAt               time.Time  `gorm:"column:member_created_at"`
	MemberUpdatedBy               *uuid.UUID `gorm:"column:member_updated_by"`
	MemberUpdatedAt               time.Time  `gorm:"column:member_updated_at"`
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) WithinTransaction(ctx context.Context, fn func(context.Context, project.Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ctx, NewRepository(tx))
	})
}

func (r *Repository) FindProjectRoleByCode(ctx context.Context, code project.ProjectRole) (*project.RoleMaster, error) {
	var row projectRoleRow
	err := r.db.WithContext(ctx).
		Where("code = ?", string(code)).
		Where("status = ?", "active").
		First(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, project.ErrProjectRoleNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *Repository) FindProjectPositionByCode(ctx context.Context, code project.ProjectPosition) (*project.PositionMaster, error) {
	var row projectPositionRow
	err := r.db.WithContext(ctx).
		Where("code = ?", string(code)).
		Where("status = ?", "active").
		First(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, project.ErrProjectPositionNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *Repository) FindProjectPriorityByCode(ctx context.Context, code project.ProjectPriority) (*project.PriorityMaster, error) {
	var row projectPriorityRow
	err := r.db.WithContext(ctx).
		Where("code = ?", string(code)).
		Where("status = ?", "active").
		First(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, project.ErrProjectPriorityNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *Repository) ListProjectPositions(ctx context.Context) ([]project.PositionMaster, error) {
	var rows []projectPositionRow
	err := r.db.WithContext(ctx).
		Where("status = ?", "active").
		Order("sort_order ASC, code ASC").
		Find(&rows).
		Error
	if err != nil {
		return nil, err
	}

	items := make([]project.PositionMaster, 0, len(rows))
	for _, row := range rows {
		items = append(items, *row.toDomain())
	}
	return items, nil
}

func (r *Repository) FindActiveWorkspaceMembershipByID(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, membershipID uuid.UUID) (*project.WorkspaceMemberCandidate, error) {
	var row workspaceMemberCandidateRow
	err := r.db.WithContext(ctx).
		Table("workspace_memberships AS wm").
		Select(`
			wm.id AS membership_id,
			wm.tenant_id,
			wm.workspace_id,
			wm.profile_id,
			wm.user_account_id
		`).
		Where("wm.tenant_id = ?", tenantID).
		Where("wm.workspace_id = ?", workspaceID).
		Where("wm.id = ?", membershipID).
		Where("wm.status = ?", "active").
		Take(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, project.ErrWorkspaceMembershipNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *Repository) NextProjectCode(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, prefix string, year int, numberLength int) (string, error) {
	counterID, err := ids.NewUUID()
	if err != nil {
		return "", err
	}

	var runningNo int
	err = r.db.WithContext(ctx).
		Raw(`
			INSERT INTO project_code_counters (
				id,
				tenant_id,
				workspace_id,
				prefix,
				year,
				running_no,
				number_length,
				last_generated_at,
				created_at,
				updated_at
			)
			VALUES (?, ?, ?, ?, ?, 1, ?, now(), now(), now())
			ON CONFLICT (tenant_id, workspace_id, prefix, year)
			DO UPDATE SET
				running_no = project_code_counters.running_no + 1,
				last_generated_at = now(),
				updated_at = now()
			RETURNING running_no
		`, counterID, tenantID, workspaceID, prefix, year, numberLength).
		Scan(&runningNo).
		Error
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%d-%0*d", prefix, year, numberLength, runningNo), nil
}

func (r *Repository) CreateProject(ctx context.Context, value *project.Project) error {
	if err := ensureUUID(&value.ID); err != nil {
		return err
	}
	if value.Type == "" {
		value.Type = project.ProjectTypeInternal
	}
	if value.Status == "" {
		value.Status = project.ProjectStatusDraft
	}
	if value.CreatedAt.IsZero() {
		value.CreatedAt = time.Now().UTC()
	}
	if value.UpdatedAt.IsZero() {
		value.UpdatedAt = value.CreatedAt
	}

	row := projectRow{
		ID:                     value.ID,
		TenantID:               value.TenantID,
		WorkspaceID:            value.WorkspaceID,
		ProjectCode:            value.Code,
		ProjectName:            value.Name,
		ProjectType:            string(value.Type),
		ProjectStatus:          string(value.Status),
		PriorityID:             value.PriorityID,
		Description:            value.Description,
		ClientOrRequestingUnit: value.ClientOrRequestingUnit,
		ScopeOrObjective:       value.ScopeOrObjective,
		CreatedBy:              value.CreatedBy,
		CreatedAt:              value.CreatedAt,
		UpdatedBy:              value.UpdatedBy,
		UpdatedAt:              value.UpdatedAt,
		DeletedBy:              value.DeletedBy,
		DeletedAt:              value.DeletedAt,
		ArchivedAt:             value.ArchivedAt,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return mapCreateProjectError(err)
	}

	value.ID = row.ID
	value.CreatedAt = row.CreatedAt
	value.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *Repository) CreateMember(ctx context.Context, value *project.Member) error {
	if err := ensureUUID(&value.ID); err != nil {
		return err
	}
	if value.Status == "" {
		value.Status = project.ProjectMemberStatusActive
	}
	if value.CreatedAt.IsZero() {
		value.CreatedAt = time.Now().UTC()
	}
	if value.UpdatedAt.IsZero() {
		value.UpdatedAt = value.CreatedAt
	}

	row := projectMemberRow{
		ID:                    value.ID,
		TenantID:              value.TenantID,
		WorkspaceID:           value.WorkspaceID,
		ProjectID:             value.ProjectID,
		WorkspaceMembershipID: value.WorkspaceMembershipID,
		ProfileID:             value.ProfileID,
		UserAccountID:         value.UserAccountID,
		ProjectRoleID:         value.RoleID,
		Status:                string(value.Status),
		JoinedAt:              value.JoinedAt,
		RemovedAt:             value.RemovedAt,
		CreatedBy:             value.CreatedBy,
		CreatedAt:             value.CreatedAt,
		UpdatedBy:             value.UpdatedBy,
		UpdatedAt:             value.UpdatedAt,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return mapCreateProjectMemberError(err)
	}

	value.ID = row.ID
	value.CreatedAt = row.CreatedAt
	value.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *Repository) UpdateProjectProfile(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, patch project.ProjectProfilePatch) error {
	updates := map[string]any{
		"updated_by": patch.UpdatedBy,
		"updated_at": patch.UpdatedAt,
	}
	if patch.Name != nil {
		updates["project_name"] = *patch.Name
	}
	if patch.Type != nil {
		updates["project_type"] = string(*patch.Type)
	}
	if patch.PriorityID != nil {
		updates["priority_id"] = *patch.PriorityID
	}
	if patch.Description != nil {
		updates["description"] = *patch.Description
	}
	if patch.ClientOrRequestingUnit != nil {
		updates["client_or_requesting_unit"] = *patch.ClientOrRequestingUnit
	}
	if patch.ScopeOrObjective != nil {
		updates["scope_or_objective"] = *patch.ScopeOrObjective
	}

	result := r.db.WithContext(ctx).
		Model(&projectRow{}).
		Where("tenant_id = ?", tenantID).
		Where("workspace_id = ?", workspaceID).
		Where("id = ?", projectID).
		Where("deleted_at IS NULL").
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return project.ErrProjectNotFound
	}
	return nil
}

func (r *Repository) FindProjectByID(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID) (*project.ProjectWithMember, error) {
	var row projectListRow
	err := r.db.WithContext(ctx).
		Table("projects AS p").
		Select(projectListSelect()).
		Joins("JOIN project_priorities AS pp ON pp.id = p.priority_id").
		Joins(projectOwnerJoin(), string(project.ProjectMemberStatusActive), string(project.ProjectRoleOwner)).
		Joins("LEFT JOIN project_roles AS pr ON pr.id = pm.project_role_id").
		Where("p.tenant_id = ?", tenantID).
		Where("p.workspace_id = ?", workspaceID).
		Where("p.id = ?", projectID).
		Where("p.deleted_at IS NULL").
		Take(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, project.ErrProjectNotFound
	}
	if err != nil {
		return nil, err
	}

	item := row.toDomain()
	return &item, nil
}

func (r *Repository) ListProjects(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, limit int, offset int) ([]project.ProjectWithMember, int, error) {
	var total int64
	countQuery := r.db.WithContext(ctx).
		Table("projects AS p").
		Where("p.tenant_id = ?", tenantID).
		Where("p.workspace_id = ?", workspaceID).
		Where("p.deleted_at IS NULL")
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []projectListRow
	err := r.db.WithContext(ctx).
		Table("projects AS p").
		Select(projectListSelect()).
		Joins("JOIN project_priorities AS pp ON pp.id = p.priority_id").
		Joins(projectOwnerJoin(), string(project.ProjectMemberStatusActive), string(project.ProjectRoleOwner)).
		Joins("LEFT JOIN project_roles AS pr ON pr.id = pm.project_role_id").
		Where("p.tenant_id = ?", tenantID).
		Where("p.workspace_id = ?", workspaceID).
		Where("p.deleted_at IS NULL").
		Order("p.created_at DESC, p.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).
		Error
	if err != nil {
		return nil, 0, err
	}

	items := make([]project.ProjectWithMember, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toDomain())
	}
	return items, int(total), nil
}

func (r *Repository) ListProjectMembers(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, limit int, offset int) ([]project.Member, int, error) {
	var total int64
	countQuery := r.db.WithContext(ctx).
		Table("project_members AS pm").
		Where("pm.tenant_id = ?", tenantID).
		Where("pm.workspace_id = ?", workspaceID).
		Where("pm.project_id = ?", projectID).
		Where("pm.status = ?", string(project.ProjectMemberStatusActive))
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []projectMemberRow
	err := r.db.WithContext(ctx).
		Table("project_members AS pm").
		Select(`
			pm.id,
			pm.tenant_id,
			pm.workspace_id,
			pm.project_id,
			pm.workspace_membership_id,
			pm.profile_id,
			pm.user_account_id,
			pm.project_role_id,
			pr.code AS project_role_code,
			pm.status,
			pm.joined_at,
			pm.removed_at,
			pm.created_by,
			pm.created_at,
			pm.updated_by,
			pm.updated_at
		`).
		Joins("JOIN project_roles AS pr ON pr.id = pm.project_role_id").
		Where("pm.tenant_id = ?", tenantID).
		Where("pm.workspace_id = ?", workspaceID).
		Where("pm.project_id = ?", projectID).
		Where("pm.status = ?", string(project.ProjectMemberStatusActive)).
		Order("pm.joined_at ASC NULLS LAST, pm.created_at ASC, pm.id ASC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).
		Error
	if err != nil {
		return nil, 0, err
	}

	items := make([]project.Member, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toDomain())
	}
	return items, int(total), nil
}

func (r *Repository) FindProjectMemberByID(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, memberID uuid.UUID) (*project.Member, error) {
	var row projectMemberRow
	err := r.db.WithContext(ctx).
		Table("project_members AS pm").
		Select(projectMemberSelect()).
		Joins("JOIN project_roles AS pr ON pr.id = pm.project_role_id").
		Where("pm.tenant_id = ?", tenantID).
		Where("pm.workspace_id = ?", workspaceID).
		Where("pm.project_id = ?", projectID).
		Where("pm.id = ?", memberID).
		Where("pm.status = ?", string(project.ProjectMemberStatusActive)).
		Take(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, project.ErrProjectMemberNotFound
	}
	if err != nil {
		return nil, err
	}

	member := row.toDomain()
	return &member, nil
}

func (r *Repository) CountActiveProjectMembersByRole(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, role project.ProjectRole) (int, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Table("project_members AS pm").
		Joins("JOIN project_roles AS pr ON pr.id = pm.project_role_id").
		Where("pm.tenant_id = ?", tenantID).
		Where("pm.workspace_id = ?", workspaceID).
		Where("pm.project_id = ?", projectID).
		Where("pm.status = ?", string(project.ProjectMemberStatusActive)).
		Where("pr.code = ?", string(role)).
		Count(&total).
		Error
	if err != nil {
		return 0, err
	}
	return int(total), nil
}

func (r *Repository) UpdateProjectMemberRole(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, memberID uuid.UUID, roleID uuid.UUID, updatedBy uuid.UUID, updatedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&projectMemberRow{}).
		Where("tenant_id = ?", tenantID).
		Where("workspace_id = ?", workspaceID).
		Where("project_id = ?", projectID).
		Where("id = ?", memberID).
		Where("status = ?", string(project.ProjectMemberStatusActive)).
		Updates(map[string]any{
			"project_role_id": roleID,
			"updated_by":      updatedBy,
			"updated_at":      updatedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return project.ErrProjectMemberNotFound
	}
	return nil
}

func (r *Repository) RemoveProjectMember(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, memberID uuid.UUID, removedBy uuid.UUID, removedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&projectMemberRow{}).
		Where("tenant_id = ?", tenantID).
		Where("workspace_id = ?", workspaceID).
		Where("project_id = ?", projectID).
		Where("id = ?", memberID).
		Where("status = ?", string(project.ProjectMemberStatusActive)).
		Updates(map[string]any{
			"status":     string(project.ProjectMemberStatusRemoved),
			"removed_at": removedAt,
			"updated_by": removedBy,
			"updated_at": removedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return project.ErrProjectMemberNotFound
	}
	return nil
}

func (r *Repository) ListProjectMemberPositions(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, memberID uuid.UUID) ([]project.MemberPosition, error) {
	var rows []projectMemberPositionRow
	err := r.db.WithContext(ctx).
		Table("project_member_positions AS pmp").
		Select(`
			pmp.id,
			pmp.tenant_id,
			pmp.workspace_id,
			pmp.project_id,
			pmp.project_member_id,
			pmp.position_id,
			pp.code AS position_code,
			pp.name AS position_name,
			pmp.created_at,
			pmp.created_by
		`).
		Joins("JOIN project_positions AS pp ON pp.id = pmp.position_id").
		Where("pmp.tenant_id = ?", tenantID).
		Where("pmp.workspace_id = ?", workspaceID).
		Where("pmp.project_id = ?", projectID).
		Where("pmp.project_member_id = ?", memberID).
		Order("pp.sort_order ASC, pp.code ASC").
		Scan(&rows).
		Error
	if err != nil {
		return nil, err
	}

	items := make([]project.MemberPosition, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toDomain())
	}
	return items, nil
}

func (r *Repository) ReplaceProjectMemberPositions(ctx context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, memberID uuid.UUID, positionIDs []uuid.UUID, createdBy uuid.UUID, createdAt time.Time) error {
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Where("workspace_id = ?", workspaceID).
		Where("project_id = ?", projectID).
		Where("project_member_id = ?", memberID).
		Delete(&projectMemberPositionRow{}).
		Error; err != nil {
		return err
	}

	if len(positionIDs) == 0 {
		return nil
	}

	rows := make([]projectMemberPositionRow, 0, len(positionIDs))
	for _, positionID := range positionIDs {
		id, err := ids.NewUUID()
		if err != nil {
			return err
		}
		rows = append(rows, projectMemberPositionRow{
			ID:              id,
			TenantID:        tenantID,
			WorkspaceID:     workspaceID,
			ProjectID:       projectID,
			ProjectMemberID: memberID,
			PositionID:      positionID,
			CreatedBy:       &createdBy,
			CreatedAt:       createdAt,
		})
	}
	return r.db.WithContext(ctx).Create(&rows).Error
}

func projectOwnerJoin() string {
	return `
		LEFT JOIN project_members AS pm
			ON pm.project_id = p.id
			AND pm.tenant_id = p.tenant_id
			AND pm.workspace_id = p.workspace_id
			AND pm.status = ?
			AND pm.project_role_id = (
				SELECT id FROM project_roles WHERE code = ? LIMIT 1
			)
	`
}

func projectListSelect() string {
	return `
		p.id AS project_id,
		p.tenant_id AS project_tenant_id,
		p.workspace_id AS project_workspace_id,
		p.project_code AS project_code,
		p.project_name AS project_name,
		p.project_type AS project_type,
		p.project_status AS project_status,
		p.priority_id AS project_priority_id,
		pp.code AS project_priority_code,
		p.description AS project_description,
		p.client_or_requesting_unit AS project_client_or_requesting_unit,
		p.scope_or_objective AS project_scope_or_objective,
		p.created_by AS project_created_by,
		p.created_at AS project_created_at,
		p.updated_by AS project_updated_by,
		p.updated_at AS project_updated_at,
		p.deleted_by AS project_deleted_by,
		p.deleted_at AS project_deleted_at,
		p.archived_at AS project_archived_at,
		pm.id AS member_id,
		pm.tenant_id AS member_tenant_id,
		pm.workspace_id AS member_workspace_id,
		pm.project_id AS member_project_id,
		pm.workspace_membership_id AS member_workspace_membership_id,
		pm.profile_id AS member_profile_id,
		pm.user_account_id AS member_user_account_id,
		pm.project_role_id AS member_project_role_id,
		pr.code AS member_project_role_code,
		pm.status AS member_status,
		pm.joined_at AS member_joined_at,
		pm.removed_at AS member_removed_at,
		pm.created_by AS member_created_by,
		pm.created_at AS member_created_at,
		pm.updated_by AS member_updated_by,
		pm.updated_at AS member_updated_at
	`
}

func projectMemberSelect() string {
	return `
		pm.id,
		pm.tenant_id,
		pm.workspace_id,
		pm.project_id,
		pm.workspace_membership_id,
		pm.profile_id,
		pm.user_account_id,
		pm.project_role_id,
		pr.code AS project_role_code,
		pm.status,
		pm.joined_at,
		pm.removed_at,
		pm.created_by,
		pm.created_at,
		pm.updated_by,
		pm.updated_at
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

func mapCreateProjectError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if pgErr.ConstraintName == "uq_projects_code_active" {
			return project.ErrProjectCodeAlreadyTaken
		}
	}
	return err
}

func mapCreateProjectMemberError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if pgErr.ConstraintName == "uq_project_members_active_workspace_membership" {
			return project.ErrProjectMemberAlreadyExists
		}
	}
	return err
}

func (r workspaceMemberCandidateRow) toDomain() *project.WorkspaceMemberCandidate {
	return &project.WorkspaceMemberCandidate{
		MembershipID:  r.MembershipID,
		TenantID:      r.TenantID,
		WorkspaceID:   r.WorkspaceID,
		ProfileID:     r.ProfileID,
		UserAccountID: r.UserAccountID,
	}
}

func (r projectRow) toDomain() project.Project {
	return project.Project{
		ID:                     r.ID,
		TenantID:               r.TenantID,
		WorkspaceID:            r.WorkspaceID,
		Code:                   r.ProjectCode,
		Name:                   r.ProjectName,
		Type:                   project.ProjectType(r.ProjectType),
		Status:                 project.ProjectStatus(r.ProjectStatus),
		PriorityID:             r.PriorityID,
		Description:            r.Description,
		ClientOrRequestingUnit: r.ClientOrRequestingUnit,
		ScopeOrObjective:       r.ScopeOrObjective,
		CreatedBy:              r.CreatedBy,
		CreatedAt:              r.CreatedAt,
		UpdatedBy:              r.UpdatedBy,
		UpdatedAt:              r.UpdatedAt,
		DeletedBy:              r.DeletedBy,
		DeletedAt:              r.DeletedAt,
		ArchivedAt:             r.ArchivedAt,
	}
}

func (r projectMemberRow) toDomain() project.Member {
	return project.Member{
		ID:                    r.ID,
		TenantID:              r.TenantID,
		WorkspaceID:           r.WorkspaceID,
		ProjectID:             r.ProjectID,
		WorkspaceMembershipID: r.WorkspaceMembershipID,
		ProfileID:             r.ProfileID,
		UserAccountID:         r.UserAccountID,
		RoleID:                r.ProjectRoleID,
		Role:                  project.ProjectRole(r.ProjectRoleCode),
		Status:                project.ProjectMemberStatus(r.Status),
		JoinedAt:              r.JoinedAt,
		RemovedAt:             r.RemovedAt,
		CreatedBy:             r.CreatedBy,
		CreatedAt:             r.CreatedAt,
		UpdatedBy:             r.UpdatedBy,
		UpdatedAt:             r.UpdatedAt,
	}
}

func (r projectRoleRow) toDomain() *project.RoleMaster {
	return &project.RoleMaster{
		ID:          r.ID,
		Code:        project.ProjectRole(r.Code),
		Name:        r.Name,
		Description: r.Description,
		SortOrder:   r.SortOrder,
		IsSystem:    r.IsSystem,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func (r projectPositionRow) toDomain() *project.PositionMaster {
	return &project.PositionMaster{
		ID:          r.ID,
		Code:        project.ProjectPosition(r.Code),
		Name:        r.Name,
		Description: r.Description,
		SortOrder:   r.SortOrder,
		IsSystem:    r.IsSystem,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func (r projectPriorityRow) toDomain() *project.PriorityMaster {
	return &project.PriorityMaster{
		ID:        r.ID,
		Code:      project.ProjectPriority(r.Code),
		Name:      r.Name,
		SortOrder: r.SortOrder,
		IsSystem:  r.IsSystem,
		Status:    r.Status,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func (r projectMemberPositionRow) toDomain() project.MemberPosition {
	return project.MemberPosition{
		ID:              r.ID,
		TenantID:        r.TenantID,
		WorkspaceID:     r.WorkspaceID,
		ProjectID:       r.ProjectID,
		ProjectMemberID: r.ProjectMemberID,
		PositionID:      r.PositionID,
		Position:        project.ProjectPosition(r.PositionCode),
		Name:            r.PositionName,
		CreatedBy:       r.CreatedBy,
		CreatedAt:       r.CreatedAt,
	}
}

func (r projectListRow) toDomain() project.ProjectWithMember {
	return project.ProjectWithMember{
		Project: project.Project{
			ID:                     r.ProjectID,
			TenantID:               r.ProjectTenantID,
			WorkspaceID:            r.ProjectWorkspaceID,
			Code:                   r.ProjectCode,
			Name:                   r.ProjectName,
			Type:                   project.ProjectType(r.ProjectType),
			Status:                 project.ProjectStatus(r.ProjectStatus),
			PriorityID:             r.ProjectPriorityID,
			Priority:               project.ProjectPriority(r.ProjectPriorityCode),
			Description:            r.ProjectDescription,
			ClientOrRequestingUnit: r.ProjectClientOrRequestingUnit,
			ScopeOrObjective:       r.ProjectScopeOrObjective,
			CreatedBy:              r.ProjectCreatedBy,
			CreatedAt:              r.ProjectCreatedAt,
			UpdatedBy:              r.ProjectUpdatedBy,
			UpdatedAt:              r.ProjectUpdatedAt,
			DeletedBy:              r.ProjectDeletedBy,
			DeletedAt:              r.ProjectDeletedAt,
			ArchivedAt:             r.ProjectArchivedAt,
		},
		Member: project.Member{
			ID:                    r.MemberID,
			TenantID:              r.MemberTenantID,
			WorkspaceID:           r.MemberWorkspaceID,
			ProjectID:             r.MemberProjectID,
			WorkspaceMembershipID: r.MemberWorkspaceMembershipID,
			ProfileID:             r.MemberProfileID,
			UserAccountID:         r.MemberUserAccountID,
			RoleID:                r.MemberProjectRoleID,
			Role:                  project.ProjectRole(r.MemberProjectRoleCode),
			Status:                project.ProjectMemberStatus(r.MemberStatus),
			JoinedAt:              r.MemberJoinedAt,
			RemovedAt:             r.MemberRemovedAt,
			CreatedBy:             r.MemberCreatedBy,
			CreatedAt:             r.MemberCreatedAt,
			UpdatedBy:             r.MemberUpdatedBy,
			UpdatedAt:             r.MemberUpdatedAt,
		},
	}
}
