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
