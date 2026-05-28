package workspacedbrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	auditdbrepo "prasankit-api/internal/adapters/database/audit"
	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/workspace"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// invitationModel is the GORM model for workspace_invitations.
type invitationModel struct {
	ID                     uuid.UUID  `gorm:"type:uuid;primaryKey"`
	WorkspaceID            uuid.UUID  `gorm:"type:uuid;column:workspace_id;not null"`
	Email                  string     `gorm:"column:email;not null"`
	OrgRoleCode            string     `gorm:"column:org_role_code;not null"`
	TokenHash              string     `gorm:"column:token_hash;not null"`
	InvitedByUserAccountID uuid.UUID  `gorm:"type:uuid;column:invited_by_user_account_id;not null"`
	ExpiresAt              time.Time  `gorm:"column:expires_at;not null"`
	AcceptedAt             *time.Time `gorm:"column:accepted_at"`
	RevokedAt              *time.Time `gorm:"column:revoked_at"`
	AcceptedUserAccountID  *uuid.UUID `gorm:"type:uuid;column:accepted_user_account_id"`
	CreatedAt              time.Time  `gorm:"column:created_at;not null"`
}

func (invitationModel) TableName() string { return "workspace_invitations" }

// invitationToModel converts a domain Invitation to the GORM model.
func invitationToModel(inv workspace.Invitation) invitationModel {
	return invitationModel{
		ID:                     inv.ID,
		WorkspaceID:            inv.WorkspaceID,
		Email:                  inv.Email,
		OrgRoleCode:            inv.OrgRoleCode,
		TokenHash:              inv.TokenHash,
		InvitedByUserAccountID: inv.InvitedByUserAccountID,
		ExpiresAt:              inv.ExpiresAt,
		AcceptedAt:             inv.AcceptedAt,
		RevokedAt:              inv.RevokedAt,
		AcceptedUserAccountID:  inv.AcceptedUserAccountID,
		CreatedAt:              inv.CreatedAt,
	}
}

// modelToInvitation converts a GORM model to the domain Invitation.
func modelToInvitation(m invitationModel) workspace.Invitation {
	return workspace.Invitation{
		ID:                     m.ID,
		WorkspaceID:            m.WorkspaceID,
		Email:                  m.Email,
		OrgRoleCode:            m.OrgRoleCode,
		TokenHash:              m.TokenHash,
		InvitedByUserAccountID: m.InvitedByUserAccountID,
		ExpiresAt:              m.ExpiresAt,
		AcceptedAt:             m.AcceptedAt,
		RevokedAt:              m.RevokedAt,
		AcceptedUserAccountID:  m.AcceptedUserAccountID,
		CreatedAt:              m.CreatedAt,
	}
}

// InvitationRepo implements workspace.InvitationRepository against Postgres via GORM.
type InvitationRepo struct {
	db        *gorm.DB
	auditRepo *auditdbrepo.AuditRepo
}

// NewInvitationRepo constructs an InvitationRepo.
func NewInvitationRepo(db *gorm.DB, auditRepo *auditdbrepo.AuditRepo) *InvitationRepo {
	return &InvitationRepo{db: db, auditRepo: auditRepo}
}

// CreateInvitationTx atomically inserts an invitation + audit entry in one DB transaction.
// If the uq_invitation_pending constraint fires, returns workspace.ErrAlreadyPending.
func (r *InvitationRepo) CreateInvitationTx(ctx context.Context, inv workspace.Invitation, entry audit.Entry) error {
	m := invitationToModel(inv)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&m).Error; err != nil {
			return err
		}
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		if isUniqueViolation(err) {
			return workspace.ErrAlreadyPending
		}
		return fmt.Errorf("create invitation: %w", err)
	}
	return nil
}

// FindActiveByTokenHash returns an invitation whose token_hash matches and which is still
// active (accepted_at IS NULL AND revoked_at IS NULL AND expires_at > now).
// Returns nil, nil when not found.
func (r *InvitationRepo) FindActiveByTokenHash(ctx context.Context, tokenHash string) (*workspace.Invitation, error) {
	var m invitationModel
	err := r.db.WithContext(ctx).
		Where(
			"token_hash = ? AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at > ?",
			tokenHash, time.Now().UTC(),
		).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find invitation by token hash: %w", err)
	}
	inv := modelToInvitation(m)
	return &inv, nil
}

// AcceptInvitationTx atomically:
//  1. Inserts the membership row.
//  2. Updates the invitation: accepted_at = now, accepted_user_account_id = accountID.
//  3. Writes an audit entry.
//
// All-or-nothing: any failure rolls back all three.
func (r *InvitationRepo) AcceptInvitationTx(
	ctx context.Context,
	inv workspace.Invitation,
	m workspace.Membership,
	entry audit.Entry,
) error {
	memberModel := membershipToModel(m)
	now := time.Now().UTC()

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Insert membership.
		if err := tx.Create(&memberModel).Error; err != nil {
			return err
		}
		// Mark invitation accepted.
		if err := tx.Model(&invitationModel{}).
			Where("id = ?", inv.ID).
			Updates(map[string]any{
				"accepted_at":              now,
				"accepted_user_account_id": m.UserAccountID,
			}).Error; err != nil {
			return err
		}
		// Audit.
		return r.auditRepo.LogTx(tx, entry)
	})
	if err != nil {
		if isUniqueViolation(err) {
			return workspace.ErrAlreadyMember
		}
		return fmt.Errorf("accept invitation: %w", err)
	}
	return nil
}

// IsActiveMemberByEmail returns true when there is already an active membership for
// the given workspace + email (joined via user_accounts.primary_email).
func (r *InvitationRepo) IsActiveMemberByEmail(ctx context.Context, workspaceID uuid.UUID, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("workspace_memberships wm").
		Joins("INNER JOIN user_accounts ua ON ua.id = wm.user_account_id").
		Where(
			"wm.workspace_id = ? AND wm.membership_status_code = ? AND lower(ua.primary_email) = lower(?)",
			workspaceID, workspace.MembershipStatusActive, email,
		).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("is active member by email: %w", err)
	}
	return count > 0, nil
}
