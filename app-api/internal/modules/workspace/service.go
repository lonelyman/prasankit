package workspace

import (
	"context"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"prasankit-api/internal/modules/audit"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
)

// slugPattern validates workspace slugs: lowercase alphanumeric + hyphens,
// min 2 chars, max 63 chars, no leading/trailing hyphens.
var slugPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$`)

// Service implements workspace use cases.
type Service struct {
	workspaces  WorkspaceRepository
	memberships MembershipRepository
}

// NewService constructs a workspace Service.
func NewService(workspaces WorkspaceRepository, memberships MembershipRepository) *Service {
	return &Service{
		workspaces:  workspaces,
		memberships: memberships,
	}
}

// CreateWorkspaceInput carries validated parameters for workspace creation.
type CreateWorkspaceInput struct {
	AccountID    uuid.UUID
	Name         string
	Slug         string
	ContactEmail string
	// Transport metadata for audit log.
	IP        string
	UserAgent string
	RequestID string
}

// CreateWorkspace validates input and atomically creates a workspace + owner membership +
// audit log entry in a single transaction.
func (s *Service) CreateWorkspace(ctx context.Context, in CreateWorkspaceInput) (*Workspace, error) {
	if err := validateCreateWorkspaceInput(in); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	wsID, err := ids.New()
	if err != nil {
		return nil, fmt.Errorf("generate workspace id: %w", err)
	}
	membershipID, err := ids.New()
	if err != nil {
		return nil, fmt.Errorf("generate membership id: %w", err)
	}

	createdBy := in.AccountID

	ws := Workspace{
		ID:                  wsID,
		WorkspaceName:       strings.TrimSpace(in.Name),
		Slug:                strings.ToLower(strings.TrimSpace(in.Slug)),
		WorkspaceStatusCode: WorkspaceStatusActive,
		ContactEmail:        strings.ToLower(strings.TrimSpace(in.ContactEmail)),
		OwnerUserAccountID:  in.AccountID,
		CreatedAt:           now,
		CreatedBy:           &createdBy,
		UpdatedAt:           now,
	}

	joinedAt := now
	m := Membership{
		ID:                   membershipID,
		WorkspaceID:          wsID,
		UserAccountID:        in.AccountID,
		OrgRoleCode:          OrgRoleOwner,
		MembershipStatusCode: MembershipStatusActive,
		JoinedAt:             &joinedAt,
		CreatedAt:            now,
		CreatedBy:            &createdBy,
		UpdatedAt:            now,
	}

	resourceID := wsID
	actorID := in.AccountID
	entry := audit.Entry{
		WorkspaceID:    &wsID,
		ActorAccountID: &actorID,
		Action:         AuditActionWorkspaceCreate,
		ResourceType:   AuditResourceTypeWorkspace,
		ResourceID:     &resourceID,
		Result:         AuditResultSuccess,
		IP:             in.IP,
		UserAgent:      in.UserAgent,
		RequestID:      in.RequestID,
	}

	if err := s.workspaces.CreateWorkspaceWithOwner(ctx, ws, m, entry); err != nil {
		return nil, err
	}

	return &ws, nil
}

// ListMyWorkspaces returns all workspaces (with the account's role) where the account
// has an active membership and the workspace is active.
func (s *Service) ListMyWorkspaces(ctx context.Context, accountID uuid.UUID) ([]WorkspaceWithRole, error) {
	return s.memberships.ListActiveWorkspacesByAccount(ctx, accountID)
}

// GetCurrentWorkspace returns the workspace + role identified by the TenantContext.
// The workspace and role are already resolved by the middleware; we look them up
// from the membership list to produce a canonical response.
func (s *Service) GetCurrentWorkspace(ctx context.Context, tc TenantContext) (*WorkspaceWithRole, error) {
	items, err := s.memberships.ListActiveWorkspacesByAccount(ctx, tc.AccountID)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].Workspace.ID == tc.WorkspaceID {
			return &items[i], nil
		}
	}
	return nil, ErrWorkspaceNotFound
}

// validateCreateWorkspaceInput validates all input fields. Returns *ValidationError
// with per-field details, ErrSlugReserved, or nil.
func validateCreateWorkspaceInput(in CreateWorkspaceInput) error {
	var fields []FieldError

	name := strings.TrimSpace(in.Name)
	if name == "" {
		fields = append(fields, FieldError{Field: "name", Message: "must not be empty"})
	} else if len(name) > 100 {
		fields = append(fields, FieldError{Field: "name", Message: "must be 100 characters or fewer"})
	}

	slug := strings.ToLower(strings.TrimSpace(in.Slug))
	if !slugPattern.MatchString(slug) {
		fields = append(fields, FieldError{Field: "slug", Message: "must match ^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$"})
	}

	if _, err := mail.ParseAddress(in.ContactEmail); err != nil {
		fields = append(fields, FieldError{Field: "contact_email", Message: "must be a valid email address"})
	}

	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}

	// Check reserved slug separately (after format validation passes).
	if IsReservedSlug(slug) {
		return ErrSlugReserved
	}

	return nil
}
