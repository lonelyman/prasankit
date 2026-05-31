package projectmember

import (
	"context"
	"fmt"
	"strings"
	"time"

	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/project"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
)

// Service implements project_member use cases.
type Service struct {
	members  ProjectMemberRepository
	masters  project.MasterRepository  // reuse 6a master port for IsActiveProjectRoleCode
	projects project.ProjectRepository // owner-read for role-drift/owner-removal guard (OPEN QUESTION option 1)
}

// NewService constructs a projectmember Service (3-param — OPEN QUESTION option 1).
func NewService(members ProjectMemberRepository, masters project.MasterRepository, projects project.ProjectRepository) *Service {
	return &Service{members: members, masters: masters, projects: projects}
}

// ── Inputs ─────────────────────────────────────────────────────────────────────

// AddMemberInput carries validated parameters for adding a project member.
type AddMemberInput struct {
	TenantCtx             workspace.TenantContext
	ProjectID             uuid.UUID
	WorkspaceMembershipID uuid.UUID
	ProjectRoleCode       string
	IP                    string
	UserAgent             string
	RequestID             string
}

// RemoveMemberInput carries validated parameters for removing a project member.
type RemoveMemberInput struct {
	TenantCtx workspace.TenantContext
	ProjectID uuid.UUID
	MemberID  uuid.UUID
	IP        string
	UserAgent string
	RequestID string
}

// ChangeRoleInput carries validated parameters for changing a project member role.
type ChangeRoleInput struct {
	TenantCtx   workspace.TenantContext
	ProjectID   uuid.UUID
	MemberID    uuid.UUID
	NewRoleCode string
	IP          string
	UserAgent   string
	RequestID   string
}

// ListMembersInput carries the lookup parameters for listing project members.
type ListMembersInput struct {
	TenantCtx workspace.TenantContext
	ProjectID uuid.UUID
}

// ── Methods ────────────────────────────────────────────────────────────────────

// AddMember validates input and atomically adds a project member + audit log entry.
func (s *Service) AddMember(ctx context.Context, in AddMemberInput) (*ProjectMember, error) {
	role := strings.TrimSpace(in.ProjectRoleCode)
	if role == "" {
		return nil, &ValidationError{Fields: []FieldError{{Field: "project_role_code", Message: "must not be empty"}}}
	}
	// No second owner — ownership is set only by the project-create flow; transfer is M3+.
	// Spec line 392/496: 422 ErrInvalidRoleCode (not a 400 ValidationError) — and consistent
	// with ChangeRole's promote-to-owner reject below.
	if role == ProjectRoleOwnerCode {
		return nil, ErrInvalidRoleCode
	}

	ok, err := s.masters.IsActiveProjectRoleCode(ctx, role)
	if err != nil {
		return nil, fmt.Errorf("check project_role_code: %w", err)
	}
	if !ok {
		return nil, ErrInvalidRoleCode
	}

	wsID := in.TenantCtx.WorkspaceID

	exists, err := s.members.ProjectExistsForWorkspace(ctx, wsID, in.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("check project exists: %w", err)
	}
	if !exists {
		return nil, ErrProjectNotFound
	}

	eligible, err := s.members.IsActiveWorkspaceMember(ctx, wsID, in.WorkspaceMembershipID)
	if err != nil {
		return nil, fmt.Errorf("check workspace membership: %w", err)
	}
	if !eligible {
		return nil, ErrMembershipNotEligible
	}

	id, err := ids.New()
	if err != nil {
		return nil, fmt.Errorf("generate project_member id: %w", err)
	}
	now := time.Now().UTC()
	m := ProjectMember{
		ID:                    id,
		WorkspaceID:           wsID,
		ProjectID:             in.ProjectID,
		WorkspaceMembershipID: in.WorkspaceMembershipID,
		ProjectRoleCode:       role,
		JoinedAt:              now,
		RemovedAt:             nil,
		CreatedAt:             now,
		CreatedBy:             in.TenantCtx.AccountID,
		UpdatedAt:             now,
		UpdatedBy:             nil,
	}

	projectID := in.ProjectID
	actorID := in.TenantCtx.AccountID
	resourceID := m.ID
	entry := audit.Entry{
		WorkspaceID:    &wsID,
		ProjectID:      &projectID,
		ActorAccountID: &actorID,
		Action:         AuditActionProjectMemberAdd,
		ResourceType:   AuditResourceTypeProjectMember,
		ResourceID:     &resourceID,
		OldValue:       nil,
		NewValue:       snapshotMember(m),
		Result:         AuditResultSuccess,
		IP:             in.IP,
		UserAgent:      in.UserAgent,
		RequestID:      in.RequestID,
	}

	if err := s.members.AddWithAudit(ctx, m, entry); err != nil {
		return nil, err
	}
	return &m, nil
}

// RemoveMember soft-removes a project member (removed_at=now) + audit, guarding the owner.
func (s *Service) RemoveMember(ctx context.Context, in RemoveMemberInput) error {
	wsID := in.TenantCtx.WorkspaceID

	existing, err := s.members.FindActiveByID(ctx, wsID, in.ProjectID, in.MemberID)
	if err != nil {
		return fmt.Errorf("find project_member: %w", err)
	}
	if existing == nil {
		return ErrProjectMemberNotFound
	}

	// Owner guard (D40): cannot remove the project owner outside ownership transfer (M3+).
	proj, err := s.projects.FindByIDForWorkspace(ctx, wsID, in.ProjectID)
	if err != nil {
		return fmt.Errorf("find project: %w", err)
	}
	if proj == nil {
		return ErrProjectNotFound
	}
	if proj.OwnerProjectMemberID != nil && *proj.OwnerProjectMemberID == in.MemberID {
		return ErrOwnerRoleImmutable
	}

	projectID := in.ProjectID
	actorID := in.TenantCtx.AccountID
	resourceID := in.MemberID
	entry := audit.Entry{
		WorkspaceID:    &wsID,
		ProjectID:      &projectID,
		ActorAccountID: &actorID,
		Action:         AuditActionProjectMemberRemove,
		ResourceType:   AuditResourceTypeProjectMember,
		ResourceID:     &resourceID,
		OldValue:       snapshotMember(*existing),
		NewValue:       nil,
		Result:         AuditResultSuccess,
		IP:             in.IP,
		UserAgent:      in.UserAgent,
		RequestID:      in.RequestID,
	}

	return s.members.RemoveWithAudit(ctx, wsID, in.ProjectID, in.MemberID, in.TenantCtx.AccountID, entry)
}

// ChangeRole changes a project member's project_role_code + audit, guarding the owner.
func (s *Service) ChangeRole(ctx context.Context, in ChangeRoleInput) (*ProjectMember, error) {
	newRole := strings.TrimSpace(in.NewRoleCode)
	if newRole == "" {
		return nil, &ValidationError{Fields: []FieldError{{Field: "new_role_code", Message: "must not be empty"}}}
	}
	// No member may be promoted to owner — ownership transfer is M3+.
	if newRole == ProjectRoleOwnerCode {
		return nil, ErrInvalidRoleCode
	}

	ok, err := s.masters.IsActiveProjectRoleCode(ctx, newRole)
	if err != nil {
		return nil, fmt.Errorf("check project_role_code: %w", err)
	}
	if !ok {
		return nil, ErrInvalidRoleCode
	}

	wsID := in.TenantCtx.WorkspaceID

	existing, err := s.members.FindActiveByID(ctx, wsID, in.ProjectID, in.MemberID)
	if err != nil {
		return nil, fmt.Errorf("find project_member: %w", err)
	}
	if existing == nil {
		return nil, ErrProjectMemberNotFound
	}

	// Role-drift guard (D40): cannot re-role the owner's member row outside ownership transfer.
	proj, err := s.projects.FindByIDForWorkspace(ctx, wsID, in.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("find project: %w", err)
	}
	if proj == nil {
		return nil, ErrProjectNotFound
	}
	if proj.OwnerProjectMemberID != nil && *proj.OwnerProjectMemberID == in.MemberID {
		return nil, ErrOwnerRoleImmutable
	}

	// No-op when same — no audit row, no UPDATE (mirror 6a ChangeStatus).
	if newRole == existing.ProjectRoleCode {
		return existing, nil
	}

	projectID := in.ProjectID
	actorID := in.TenantCtx.AccountID
	resourceID := in.MemberID
	entry := audit.Entry{
		WorkspaceID:    &wsID,
		ProjectID:      &projectID,
		ActorAccountID: &actorID,
		Action:         AuditActionProjectMemberRoleChange,
		ResourceType:   AuditResourceTypeProjectMember,
		ResourceID:     &resourceID,
		OldValue:       map[string]any{"project_role_code": existing.ProjectRoleCode},
		NewValue:       map[string]any{"project_role_code": newRole},
		Result:         AuditResultSuccess,
		IP:             in.IP,
		UserAgent:      in.UserAgent,
		RequestID:      in.RequestID,
	}

	if err := s.members.ChangeRoleWithAudit(ctx, wsID, in.ProjectID, in.MemberID, newRole, in.TenantCtx.AccountID, entry); err != nil {
		return nil, err
	}

	updated := *existing
	updated.ProjectRoleCode = newRole
	updated.UpdatedAt = time.Now().UTC()
	updatedBy := in.TenantCtx.AccountID
	updated.UpdatedBy = &updatedBy
	return &updated, nil
}

// ListMembers returns the active members of a project (ws-active only), with display_name.
func (s *Service) ListMembers(ctx context.Context, in ListMembersInput) ([]MemberWithDisplayName, error) {
	wsID := in.TenantCtx.WorkspaceID

	exists, err := s.members.ProjectExistsForWorkspace(ctx, wsID, in.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("check project exists: %w", err)
	}
	if !exists {
		return nil, ErrProjectNotFound
	}

	return s.members.ListByProject(ctx, wsID, in.ProjectID)
}

// snapshotMember builds a map representation of the member's mutable fields
// for OldValue / NewValue audit logging. Matches the OwnerMemberSeed snapshot
// shape used by project.CreateProject so the project_member.add trail is uniform.
func snapshotMember(m ProjectMember) map[string]any {
	return map[string]any{
		"workspace_membership_id": m.WorkspaceMembershipID,
		"project_role_code":       m.ProjectRoleCode,
		"joined_at":               m.JoinedAt,
	}
}
