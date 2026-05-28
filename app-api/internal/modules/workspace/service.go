package workspace

import (
	"context"
	"fmt"
	"log"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/email"
	"prasankit-api/pkg/ids"
	"prasankit-api/pkg/securetoken"

	"github.com/google/uuid"
)

// slugPattern validates workspace slugs: lowercase alphanumeric + hyphens,
// min 2 chars, max 63 chars, no leading/trailing hyphens.
var slugPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$`)

// Service implements workspace use cases.
type Service struct {
	workspaces    WorkspaceRepository
	memberships   MembershipRepository
	invitations   InvitationRepository
	emailSender   email.Sender
	inviteBaseURL string // base URL for the accept-invitation link sent in the email
}

// NewService constructs a workspace Service.
func NewService(
	workspaces WorkspaceRepository,
	memberships MembershipRepository,
	invitations InvitationRepository,
	emailSender email.Sender,
	inviteBaseURL string,
) *Service {
	return &Service{
		workspaces:    workspaces,
		memberships:   memberships,
		invitations:   invitations,
		emailSender:   emailSender,
		inviteBaseURL: inviteBaseURL,
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

// InviteMemberInput carries validated parameters for member invitation.
type InviteMemberInput struct {
	TenantCtx   TenantContext
	Email       string
	OrgRoleCode string
	// Transport metadata for audit log.
	IP        string
	UserAgent string
	RequestID string
}

// InviteMemberResult is returned from InviteMember (token is NOT included — sent via email only).
type InviteMemberResult struct {
	Invitation *Invitation
}

// InviteMember validates input, creates an invitation atomically, and sends the invite email
// best-effort (email failure is logged, not fatal).
func (s *Service) InviteMember(ctx context.Context, in InviteMemberInput) (*InviteMemberResult, error) {
	// Validate email.
	if _, err := mail.ParseAddress(in.Email); err != nil {
		return nil, &ValidationError{Fields: []FieldError{{Field: "email", Message: "must be a valid email address"}}}
	}
	normalEmail := strings.ToLower(strings.TrimSpace(in.Email))

	// Validate org_role_code — owner is not allowed via invite.
	switch in.OrgRoleCode {
	case OrgRoleAdmin, OrgRoleExecutive, OrgRoleUser:
		// allowed
	case OrgRoleOwner:
		return nil, ErrInviteRoleNotAllowed
	default:
		return nil, &ValidationError{Fields: []FieldError{{Field: "org_role_code", Message: "must be one of: admin, executive, user"}}}
	}

	// Check if email is already an active member (via JOIN on user_accounts.primary_email).
	alreadyMember, err := s.invitations.IsActiveMemberByEmail(ctx, in.TenantCtx.WorkspaceID, normalEmail)
	if err != nil {
		return nil, fmt.Errorf("check existing member: %w", err)
	}
	if alreadyMember {
		return nil, ErrAlreadyMember
	}

	// Generate token.
	rawToken, tokenHash, err := securetoken.New()
	if err != nil {
		return nil, fmt.Errorf("generate invite token: %w", err)
	}

	now := time.Now().UTC()
	invID, err := ids.New()
	if err != nil {
		return nil, fmt.Errorf("generate invitation id: %w", err)
	}

	inv := Invitation{
		ID:                     invID,
		WorkspaceID:            in.TenantCtx.WorkspaceID,
		Email:                  normalEmail,
		OrgRoleCode:            in.OrgRoleCode,
		TokenHash:              tokenHash,
		InvitedByUserAccountID: in.TenantCtx.AccountID,
		ExpiresAt:              now.Add(7 * 24 * time.Hour),
		CreatedAt:              now,
	}

	actorID := in.TenantCtx.AccountID
	resourceID := invID
	entry := audit.Entry{
		WorkspaceID:    &in.TenantCtx.WorkspaceID,
		ActorAccountID: &actorID,
		Action:         AuditActionMemberInvite,
		ResourceType:   AuditResourceTypeInvitation,
		ResourceID:     &resourceID,
		Result:         AuditResultSuccess,
		IP:             in.IP,
		UserAgent:      in.UserAgent,
		RequestID:      in.RequestID,
	}

	if err := s.invitations.CreateInvitationTx(ctx, inv, entry); err != nil {
		return nil, err
	}

	// Send invite email best-effort — failure is logged, not fatal.
	link := fmt.Sprintf("%s?token=%s", s.inviteBaseURL, rawToken)
	msg := email.Message{
		To:       normalEmail,
		Subject:  "You've been invited to join a workspace on Prasankit",
		TextBody: fmt.Sprintf("You have been invited to join a workspace on Prasankit.\n\nAccept your invitation by visiting:\n%s\n\nThis link expires in 7 days.", link),
	}
	if err := s.emailSender.Send(ctx, msg); err != nil {
		log.Printf("invite email send failed (non-fatal): invitation_id=%s to=%s err=%v", invID, normalEmail, err)
	}

	return &InviteMemberResult{Invitation: &inv}, nil
}

// AcceptInvitationResult is returned from AcceptInvitation.
type AcceptInvitationResult struct {
	WorkspaceID uuid.UUID
	OrgRoleCode string
}

// AcceptInvitation validates a raw token, enforces email-match anti-hijack, and atomically
// creates a membership + marks the invitation as accepted.
func (s *Service) AcceptInvitation(ctx context.Context, accountID uuid.UUID, accountEmail string, rawToken string, ip, userAgent, requestID string) (*AcceptInvitationResult, error) {
	tokenHash := securetoken.Hash(rawToken)

	inv, err := s.invitations.FindActiveByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("find invitation: %w", err)
	}
	if inv == nil {
		return nil, ErrInvitationInvalid
	}

	// Derive state from timestamps.
	now := time.Now().UTC()
	if !inv.IsActive(now) {
		return nil, ErrInvitationExpired
	}

	// Anti-hijack: email must match (case-insensitive).
	if !strings.EqualFold(accountEmail, inv.Email) {
		return nil, ErrEmailMismatch
	}

	// Check if already a member.
	existing, err := s.memberships.FindActiveByWorkspaceAndAccount(ctx, inv.WorkspaceID, accountID)
	if err != nil {
		return nil, fmt.Errorf("check existing membership: %w", err)
	}
	if existing != nil {
		return nil, ErrAlreadyMember
	}

	membershipID, err := ids.New()
	if err != nil {
		return nil, fmt.Errorf("generate membership id: %w", err)
	}

	joinedAt := now
	m := Membership{
		ID:                     membershipID,
		WorkspaceID:            inv.WorkspaceID,
		UserAccountID:          accountID,
		OrgRoleCode:            inv.OrgRoleCode,
		MembershipStatusCode:   MembershipStatusActive,
		InvitedByUserAccountID: &inv.InvitedByUserAccountID,
		JoinedAt:               &joinedAt,
		CreatedAt:              now,
		CreatedBy:              &accountID,
		UpdatedAt:              now,
	}

	resourceID := membershipID
	entry := audit.Entry{
		WorkspaceID:    &inv.WorkspaceID,
		ActorAccountID: &accountID,
		Action:         AuditActionInvitationAccept,
		ResourceType:   AuditResourceTypeMembership,
		ResourceID:     &resourceID,
		Result:         AuditResultSuccess,
		IP:             ip,
		UserAgent:      userAgent,
		RequestID:      requestID,
	}

	if err := s.invitations.AcceptInvitationTx(ctx, *inv, m, entry); err != nil {
		return nil, err
	}

	return &AcceptInvitationResult{
		WorkspaceID: inv.WorkspaceID,
		OrgRoleCode: inv.OrgRoleCode,
	}, nil
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
