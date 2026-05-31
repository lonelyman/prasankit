package projectmemberposition

import (
	"context"
	"fmt"
	"strings"
	"time"

	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/projectmember"
	"prasankit-api/internal/modules/projectposition"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
)

// Service implements project_member_position (junction) use cases.
type Service struct {
	positions ProjectMemberPositionRepository
	members   projectmember.ProjectMemberRepository    // member existence + removed_at read via FindByID
	masters   projectposition.PositionMasterRepository // D28 active-only pre-check
}

// NewService constructs a projectmemberposition Service (3-param).
// Import edges: projectmemberposition -> projectmember (one-way, D46) AND
// projectmemberposition -> projectposition (one-way, behavioral port only). Acyclic.
func NewService(
	positions ProjectMemberPositionRepository,
	members projectmember.ProjectMemberRepository,
	masters projectposition.PositionMasterRepository,
) *Service {
	return &Service{positions: positions, members: members, masters: masters}
}

// ── Inputs ─────────────────────────────────────────────────────────────────────

// AssignInput carries validated parameters for assigning a position to a project member.
type AssignInput struct {
	TenantCtx           workspace.TenantContext
	ProjectID           uuid.UUID
	ProjectMemberID     uuid.UUID
	ProjectPositionCode string
	IP                  string
	UserAgent           string
	RequestID           string
}

// UnassignInput carries validated parameters for unassigning a position from a project member.
type UnassignInput struct {
	TenantCtx           workspace.TenantContext
	ProjectID           uuid.UUID
	ProjectMemberID     uuid.UUID
	ProjectPositionCode string
	IP                  string
	UserAgent           string
	RequestID           string
}

// ListInput carries the lookup parameters for listing a member's positions.
type ListInput struct {
	TenantCtx       workspace.TenantContext
	ProjectID       uuid.UUID
	ProjectMemberID uuid.UUID
}

// ── Methods ────────────────────────────────────────────────────────────────────

// Assign validates input and atomically assigns a position to a project member + audit.
func (s *Service) Assign(ctx context.Context, in AssignInput) (*ProjectMemberPosition, error) {
	code := strings.ToLower(strings.TrimSpace(in.ProjectPositionCode))
	if code == "" {
		return nil, &ValidationError{Fields: []FieldError{{Field: "project_position_code", Message: "must not be empty"}}}
	}

	wsID := in.TenantCtx.WorkspaceID

	// Member guard (resolves the 404-vs-422 split, §M2.5(c)): read the target member
	// INCLUDING removed rows. nil -> 404; removed_at IS NOT NULL -> 422.
	if err := s.guardMember(ctx, wsID, in.ProjectID, in.ProjectMemberID); err != nil {
		return nil, err
	}

	// Active-only master pre-check (D28) — cannot assign a deprecated position to new rows.
	ok, err := s.masters.IsActiveProjectPositionCode(ctx, wsID, code)
	if err != nil {
		return nil, fmt.Errorf("check project_position_code: %w", err)
	}
	if !ok {
		return nil, ErrInvalidPositionCode
	}

	id, err := ids.New()
	if err != nil {
		return nil, fmt.Errorf("generate project_member_position id: %w", err)
	}
	now := time.Now().UTC()
	mp := ProjectMemberPosition{
		ID:                  id,
		WorkspaceID:         wsID,
		ProjectMemberID:     in.ProjectMemberID,
		ProjectPositionCode: code,
		CreatedAt:           now,
		CreatedBy:           in.TenantCtx.AccountID,
	}

	wsIDLocal := wsID
	projectID := in.ProjectID
	actorID := in.TenantCtx.AccountID
	resourceID := in.ProjectMemberID // stable addressable subject (junction row id is ephemeral)
	entry := audit.Entry{
		WorkspaceID:    &wsIDLocal,
		ProjectID:      &projectID, // project-scoped (D43): BOTH non-nil
		ActorAccountID: &actorID,
		Action:         AuditActionPositionAssign,
		ResourceType:   AuditResourceTypeProjectMemberPosition,
		ResourceID:     &resourceID,
		OldValue:       nil,
		NewValue: map[string]any{
			"project_member_id":     in.ProjectMemberID,
			"project_position_code": code,
		},
		Result:    AuditResultSuccess,
		IP:        in.IP,
		UserAgent: in.UserAgent,
		RequestID: in.RequestID,
	}

	if err := s.positions.AssignWithAudit(ctx, mp, entry); err != nil {
		return nil, err
	}
	return &mp, nil
}

// Unassign validates input and atomically unassigns a position from a project member + audit.
func (s *Service) Unassign(ctx context.Context, in UnassignInput) error {
	code := strings.ToLower(strings.TrimSpace(in.ProjectPositionCode))
	if code == "" {
		return &ValidationError{Fields: []FieldError{{Field: "project_position_code", Message: "must not be empty"}}}
	}

	wsID := in.TenantCtx.WorkspaceID

	// Member guard — MANDATORY (not just symmetry): the DELETE keys on
	// (workspace_id, project_member_id, project_position_code) and the table has NO project_id
	// column, so this gate is the ONLY thing proving the member belongs to the :id project,
	// which keeps the project-scoped audit row's project_id truthful (D43 integrity).
	if err := s.guardMember(ctx, wsID, in.ProjectID, in.ProjectMemberID); err != nil {
		return err
	}

	wsIDLocal := wsID
	projectID := in.ProjectID
	actorID := in.TenantCtx.AccountID
	resourceID := in.ProjectMemberID
	entry := audit.Entry{
		WorkspaceID:    &wsIDLocal,
		ProjectID:      &projectID, // project-scoped (D43): BOTH non-nil
		ActorAccountID: &actorID,
		Action:         AuditActionPositionUnassign,
		ResourceType:   AuditResourceTypeProjectMemberPosition,
		ResourceID:     &resourceID,
		OldValue: map[string]any{
			"project_member_id":     in.ProjectMemberID,
			"project_position_code": code,
		},
		NewValue:  nil,
		Result:    AuditResultSuccess,
		IP:        in.IP,
		UserAgent: in.UserAgent,
		RequestID: in.RequestID,
	}

	return s.positions.UnassignWithAudit(ctx, wsID, in.ProjectMemberID, code, entry)
}

// List returns the member's positions (with labels), after the member guard.
func (s *Service) List(ctx context.Context, in ListInput) ([]PositionWithLabel, error) {
	wsID := in.TenantCtx.WorkspaceID
	if err := s.guardMember(ctx, wsID, in.ProjectID, in.ProjectMemberID); err != nil {
		return nil, err
	}
	return s.positions.ListByMember(ctx, wsID, in.ProjectMemberID)
}

// guardMember reads the target project_member INCLUDING removed rows and branches:
// nil -> ErrProjectMemberNotFound (404); RemovedAt != nil -> ErrMemberRemoved (422); else nil.
func (s *Service) guardMember(ctx context.Context, workspaceID, projectID, memberID uuid.UUID) error {
	m, err := s.members.FindByID(ctx, workspaceID, projectID, memberID)
	if err != nil {
		return fmt.Errorf("find project_member: %w", err)
	}
	if m == nil {
		return ErrProjectMemberNotFound
	}
	if m.RemovedAt != nil {
		return ErrMemberRemoved
	}
	return nil
}
