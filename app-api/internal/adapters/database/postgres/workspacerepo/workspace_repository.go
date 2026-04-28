package workspacerepo

import (
	"context"
	"errors"
	"time"

	"prasankit-api/internal/modules/workspace"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

type workspaceRow struct {
	ID                    uuid.UUID  `gorm:"column:id;type:uuid"`
	TenantID              uuid.UUID  `gorm:"column:tenant_id;type:uuid"`
	WorkspaceName         string     `gorm:"column:workspace_name"`
	Slug                  string     `gorm:"column:slug"`
	Mode                  string     `gorm:"column:mode"`
	Status                string     `gorm:"column:status"`
	ContactEmail          string     `gorm:"column:contact_email"`
	OwnerUserAccountID    uuid.UUID  `gorm:"column:owner_user_account_id;type:uuid"`
	EmailVerifiedRequired bool       `gorm:"column:email_verified_required"`
	CreatedAt             time.Time  `gorm:"column:created_at"`
	CreatedBy             uuid.UUID  `gorm:"column:created_by;type:uuid"`
	UpdatedAt             time.Time  `gorm:"column:updated_at"`
	UpdatedBy             *uuid.UUID `gorm:"column:updated_by;type:uuid"`
	PendingDeletionAt     *time.Time `gorm:"column:pending_deletion_at"`
	DeletedAt             *time.Time `gorm:"column:deleted_at"`
	DeletedBy             *uuid.UUID `gorm:"column:deleted_by;type:uuid"`
	HardDeletedAt         *time.Time `gorm:"column:hard_deleted_at"`
}

func (workspaceRow) TableName() string {
	return "workspaces"
}

type membershipRow struct {
	ID            uuid.UUID  `gorm:"column:id;type:uuid"`
	TenantID      uuid.UUID  `gorm:"column:tenant_id;type:uuid"`
	WorkspaceID   uuid.UUID  `gorm:"column:workspace_id;type:uuid"`
	ProfileID     *uuid.UUID `gorm:"column:profile_id;type:uuid"`
	UserAccountID *uuid.UUID `gorm:"column:user_account_id;type:uuid"`
	WorkspaceRole string     `gorm:"column:workspace_role"`
	Status        string     `gorm:"column:status"`
	StatusReason  string     `gorm:"column:status_reason"`
	JoinedAt      *time.Time `gorm:"column:joined_at"`
	RemovedAt     *time.Time `gorm:"column:removed_at"`
	SuspendedAt   *time.Time `gorm:"column:suspended_at"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	CreatedBy     *uuid.UUID `gorm:"column:created_by;type:uuid"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
	UpdatedBy     *uuid.UUID `gorm:"column:updated_by;type:uuid"`
}

func (membershipRow) TableName() string {
	return "workspace_memberships"
}

type workspaceMembershipListRow struct {
	WorkspaceID                    uuid.UUID  `gorm:"column:workspace_id"`
	WorkspaceTenantID              uuid.UUID  `gorm:"column:workspace_tenant_id"`
	WorkspaceName                  string     `gorm:"column:workspace_name"`
	WorkspaceSlug                  string     `gorm:"column:workspace_slug"`
	WorkspaceMode                  string     `gorm:"column:workspace_mode"`
	WorkspaceStatus                string     `gorm:"column:workspace_status"`
	WorkspaceContactEmail          string     `gorm:"column:workspace_contact_email"`
	WorkspaceOwnerUserAccountID    uuid.UUID  `gorm:"column:workspace_owner_user_account_id"`
	WorkspaceEmailVerifiedRequired bool       `gorm:"column:workspace_email_verified_required"`
	WorkspaceCreatedAt             time.Time  `gorm:"column:workspace_created_at"`
	WorkspaceCreatedBy             uuid.UUID  `gorm:"column:workspace_created_by"`
	WorkspaceUpdatedAt             time.Time  `gorm:"column:workspace_updated_at"`
	WorkspaceUpdatedBy             *uuid.UUID `gorm:"column:workspace_updated_by"`
	WorkspacePendingDeletionAt     *time.Time `gorm:"column:workspace_pending_deletion_at"`
	WorkspaceDeletedAt             *time.Time `gorm:"column:workspace_deleted_at"`
	WorkspaceDeletedBy             *uuid.UUID `gorm:"column:workspace_deleted_by"`
	WorkspaceHardDeletedAt         *time.Time `gorm:"column:workspace_hard_deleted_at"`
	MembershipID                   uuid.UUID  `gorm:"column:membership_id"`
	MembershipTenantID             uuid.UUID  `gorm:"column:membership_tenant_id"`
	MembershipWorkspaceID          uuid.UUID  `gorm:"column:membership_workspace_id"`
	MembershipProfileID            *uuid.UUID `gorm:"column:membership_profile_id"`
	MembershipUserAccountID        *uuid.UUID `gorm:"column:membership_user_account_id"`
	MembershipWorkspaceRole        string     `gorm:"column:membership_workspace_role"`
	MembershipStatus               string     `gorm:"column:membership_status"`
	MembershipStatusReason         string     `gorm:"column:membership_status_reason"`
	MembershipJoinedAt             *time.Time `gorm:"column:membership_joined_at"`
	MembershipRemovedAt            *time.Time `gorm:"column:membership_removed_at"`
	MembershipSuspendedAt          *time.Time `gorm:"column:membership_suspended_at"`
	MembershipCreatedAt            time.Time  `gorm:"column:membership_created_at"`
	MembershipCreatedBy            *uuid.UUID `gorm:"column:membership_created_by"`
	MembershipUpdatedAt            time.Time  `gorm:"column:membership_updated_at"`
	MembershipUpdatedBy            *uuid.UUID `gorm:"column:membership_updated_by"`
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) WithinTransaction(ctx context.Context, fn func(context.Context, workspace.Repository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(ctx, NewRepository(tx))
	})
}

func (r *Repository) FindWorkspaceBySlug(ctx context.Context, slug string) (*workspace.Workspace, error) {
	var row workspaceRow
	err := r.db.WithContext(ctx).
		Where("slug = ?", slug).
		Where("deleted_at IS NULL").
		First(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, workspace.ErrWorkspaceNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *Repository) CreateWorkspace(ctx context.Context, workspaceRecord *workspace.Workspace) error {
	if err := ensureUUID(&workspaceRecord.ID); err != nil {
		return err
	}
	if err := ensureUUID(&workspaceRecord.TenantID); err != nil {
		return err
	}
	if workspaceRecord.Mode == "" {
		workspaceRecord.Mode = workspace.WorkspaceModeDemo
	}
	if workspaceRecord.Status == "" {
		workspaceRecord.Status = workspace.WorkspaceStatusActive
	}
	if workspaceRecord.CreatedAt.IsZero() {
		workspaceRecord.CreatedAt = time.Now().UTC()
	}
	if workspaceRecord.UpdatedAt.IsZero() {
		workspaceRecord.UpdatedAt = workspaceRecord.CreatedAt
	}

	row := workspaceRow{
		ID:                    workspaceRecord.ID,
		TenantID:              workspaceRecord.TenantID,
		WorkspaceName:         workspaceRecord.Name,
		Slug:                  workspaceRecord.Slug,
		Mode:                  string(workspaceRecord.Mode),
		Status:                string(workspaceRecord.Status),
		ContactEmail:          workspaceRecord.ContactEmail,
		OwnerUserAccountID:    workspaceRecord.OwnerUserAccountID,
		EmailVerifiedRequired: workspaceRecord.EmailVerifiedRequired,
		CreatedAt:             workspaceRecord.CreatedAt,
		CreatedBy:             workspaceRecord.CreatedBy,
		UpdatedAt:             workspaceRecord.UpdatedAt,
		UpdatedBy:             workspaceRecord.UpdatedBy,
		PendingDeletionAt:     workspaceRecord.PendingDeletionAt,
		DeletedAt:             workspaceRecord.DeletedAt,
		DeletedBy:             workspaceRecord.DeletedBy,
		HardDeletedAt:         workspaceRecord.HardDeletedAt,
	}

	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return mapCreateWorkspaceError(err)
	}

	workspaceRecord.ID = row.ID
	workspaceRecord.TenantID = row.TenantID
	workspaceRecord.CreatedAt = row.CreatedAt
	workspaceRecord.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *Repository) CreateMembership(ctx context.Context, membership *workspace.Membership) error {
	if err := ensureUUID(&membership.ID); err != nil {
		return err
	}
	if membership.Status == "" {
		membership.Status = workspace.MembershipStatusActive
	}
	if membership.Role == "" {
		membership.Role = workspace.WorkspaceRoleUser
	}
	if membership.CreatedAt.IsZero() {
		membership.CreatedAt = time.Now().UTC()
	}
	if membership.UpdatedAt.IsZero() {
		membership.UpdatedAt = membership.CreatedAt
	}

	row := membershipRow{
		ID:            membership.ID,
		TenantID:      membership.TenantID,
		WorkspaceID:   membership.WorkspaceID,
		ProfileID:     membership.ProfileID,
		UserAccountID: membership.UserAccountID,
		WorkspaceRole: string(membership.Role),
		Status:        string(membership.Status),
		StatusReason:  membership.StatusReason,
		JoinedAt:      membership.JoinedAt,
		RemovedAt:     membership.RemovedAt,
		SuspendedAt:   membership.SuspendedAt,
		CreatedAt:     membership.CreatedAt,
		CreatedBy:     membership.CreatedBy,
		UpdatedAt:     membership.UpdatedAt,
		UpdatedBy:     membership.UpdatedBy,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}

	membership.ID = row.ID
	membership.CreatedAt = row.CreatedAt
	membership.UpdatedAt = row.UpdatedAt
	return nil
}

func (r *Repository) FindActiveMembership(ctx context.Context, tenantID uuid.UUID, userAccountID uuid.UUID) (*workspace.Membership, error) {
	var row membershipRow
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Where("user_account_id = ?", userAccountID).
		Where("status = ?", string(workspace.MembershipStatusActive)).
		First(&row).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, workspace.ErrMembershipNotFound
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *Repository) ListWorkspacesByUserAccountID(ctx context.Context, userAccountID uuid.UUID, limit int, offset int) ([]workspace.WorkspaceWithMembership, int, error) {
	var total int64
	countQuery := r.db.WithContext(ctx).
		Table("workspace_memberships AS wm").
		Joins("JOIN workspaces AS w ON w.id = wm.workspace_id AND w.tenant_id = wm.tenant_id").
		Where("wm.user_account_id = ?", userAccountID).
		Where("wm.status = ?", string(workspace.MembershipStatusActive)).
		Where("w.deleted_at IS NULL")
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []workspaceMembershipListRow
	err := r.db.WithContext(ctx).
		Table("workspace_memberships AS wm").
		Select(`
			w.id AS workspace_id,
			w.tenant_id AS workspace_tenant_id,
			w.workspace_name AS workspace_name,
			w.slug AS workspace_slug,
			w.mode AS workspace_mode,
			w.status AS workspace_status,
			w.contact_email AS workspace_contact_email,
			w.owner_user_account_id AS workspace_owner_user_account_id,
			w.email_verified_required AS workspace_email_verified_required,
			w.created_at AS workspace_created_at,
			w.created_by AS workspace_created_by,
			w.updated_at AS workspace_updated_at,
			w.updated_by AS workspace_updated_by,
			w.pending_deletion_at AS workspace_pending_deletion_at,
			w.deleted_at AS workspace_deleted_at,
			w.deleted_by AS workspace_deleted_by,
			w.hard_deleted_at AS workspace_hard_deleted_at,
			wm.id AS membership_id,
			wm.tenant_id AS membership_tenant_id,
			wm.workspace_id AS membership_workspace_id,
			wm.profile_id AS membership_profile_id,
			wm.user_account_id AS membership_user_account_id,
			wm.workspace_role AS membership_workspace_role,
			wm.status AS membership_status,
			wm.status_reason AS membership_status_reason,
			wm.joined_at AS membership_joined_at,
			wm.removed_at AS membership_removed_at,
			wm.suspended_at AS membership_suspended_at,
			wm.created_at AS membership_created_at,
			wm.created_by AS membership_created_by,
			wm.updated_at AS membership_updated_at,
			wm.updated_by AS membership_updated_by
		`).
		Joins("JOIN workspaces AS w ON w.id = wm.workspace_id AND w.tenant_id = wm.tenant_id").
		Where("wm.user_account_id = ?", userAccountID).
		Where("wm.status = ?", string(workspace.MembershipStatusActive)).
		Where("w.deleted_at IS NULL").
		Order("w.created_at ASC, w.id ASC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).
		Error
	if err != nil {
		return nil, 0, err
	}

	items := make([]workspace.WorkspaceWithMembership, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toDomain())
	}
	return items, int(total), nil
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

func mapCreateWorkspaceError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if pgErr.ConstraintName == "uq_workspaces_slug_active" {
			return workspace.ErrWorkspaceSlugAlreadyTaken
		}
	}
	return err
}

func (r workspaceRow) toDomain() *workspace.Workspace {
	return &workspace.Workspace{
		ID:                    r.ID,
		TenantID:              r.TenantID,
		Name:                  r.WorkspaceName,
		Slug:                  r.Slug,
		Mode:                  workspace.WorkspaceMode(r.Mode),
		Status:                workspace.WorkspaceStatus(r.Status),
		ContactEmail:          r.ContactEmail,
		OwnerUserAccountID:    r.OwnerUserAccountID,
		EmailVerifiedRequired: r.EmailVerifiedRequired,
		CreatedAt:             r.CreatedAt,
		CreatedBy:             r.CreatedBy,
		UpdatedAt:             r.UpdatedAt,
		UpdatedBy:             r.UpdatedBy,
		PendingDeletionAt:     r.PendingDeletionAt,
		DeletedAt:             r.DeletedAt,
		DeletedBy:             r.DeletedBy,
		HardDeletedAt:         r.HardDeletedAt,
	}
}

func (r membershipRow) toDomain() *workspace.Membership {
	return &workspace.Membership{
		ID:            r.ID,
		TenantID:      r.TenantID,
		WorkspaceID:   r.WorkspaceID,
		ProfileID:     r.ProfileID,
		UserAccountID: r.UserAccountID,
		Role:          workspace.WorkspaceRole(r.WorkspaceRole),
		Status:        workspace.MembershipStatus(r.Status),
		StatusReason:  r.StatusReason,
		JoinedAt:      r.JoinedAt,
		RemovedAt:     r.RemovedAt,
		SuspendedAt:   r.SuspendedAt,
		CreatedAt:     r.CreatedAt,
		CreatedBy:     r.CreatedBy,
		UpdatedAt:     r.UpdatedAt,
		UpdatedBy:     r.UpdatedBy,
	}
}

func (r workspaceMembershipListRow) toDomain() workspace.WorkspaceWithMembership {
	return workspace.WorkspaceWithMembership{
		Workspace: workspace.Workspace{
			ID:                    r.WorkspaceID,
			TenantID:              r.WorkspaceTenantID,
			Name:                  r.WorkspaceName,
			Slug:                  r.WorkspaceSlug,
			Mode:                  workspace.WorkspaceMode(r.WorkspaceMode),
			Status:                workspace.WorkspaceStatus(r.WorkspaceStatus),
			ContactEmail:          r.WorkspaceContactEmail,
			OwnerUserAccountID:    r.WorkspaceOwnerUserAccountID,
			EmailVerifiedRequired: r.WorkspaceEmailVerifiedRequired,
			CreatedAt:             r.WorkspaceCreatedAt,
			CreatedBy:             r.WorkspaceCreatedBy,
			UpdatedAt:             r.WorkspaceUpdatedAt,
			UpdatedBy:             r.WorkspaceUpdatedBy,
			PendingDeletionAt:     r.WorkspacePendingDeletionAt,
			DeletedAt:             r.WorkspaceDeletedAt,
			DeletedBy:             r.WorkspaceDeletedBy,
			HardDeletedAt:         r.WorkspaceHardDeletedAt,
		},
		Membership: workspace.Membership{
			ID:            r.MembershipID,
			TenantID:      r.MembershipTenantID,
			WorkspaceID:   r.MembershipWorkspaceID,
			ProfileID:     r.MembershipProfileID,
			UserAccountID: r.MembershipUserAccountID,
			Role:          workspace.WorkspaceRole(r.MembershipWorkspaceRole),
			Status:        workspace.MembershipStatus(r.MembershipStatus),
			StatusReason:  r.MembershipStatusReason,
			JoinedAt:      r.MembershipJoinedAt,
			RemovedAt:     r.MembershipRemovedAt,
			SuspendedAt:   r.MembershipSuspendedAt,
			CreatedAt:     r.MembershipCreatedAt,
			CreatedBy:     r.MembershipCreatedBy,
			UpdatedAt:     r.MembershipUpdatedAt,
			UpdatedBy:     r.MembershipUpdatedBy,
		},
	}
}
