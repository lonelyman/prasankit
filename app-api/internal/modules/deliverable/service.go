package deliverable

import (
	"context"
	"fmt"
	"strings"
	"time"

	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
)

// Service implements deliverable / submission / review use cases.
//
// Authz (D63) is enforced HERE (service-layer), not in middleware. The actor's project
// role is resolved per-request (D16, no cache) via ProjectAccessRepository; org Owner/Admin
// bypass the project-role gate for deliverable CRUD + read, but submit/review still require
// an active project_member row (FK), so a non-member org admin gets ErrNotProjectMember.
type Service struct {
	repo    DeliverableRepository
	masters MasterRepository
	access  ProjectAccessRepository
}

// NewService constructs a deliverable Service.
//
// The OQ-6 self-review guard (config DELIVERABLE_FORBID_SELF_REVIEW, default false) is NOT
// held here: it is enforced under the deliverable lock by the repository (which alone sees
// the locked submission's submitted_by), so the flag is wired into the repo at construction.
func NewService(repo DeliverableRepository, masters MasterRepository, access ProjectAccessRepository) *Service {
	return &Service{repo: repo, masters: masters, access: access}
}

// ── Inputs ─────────────────────────────────────────────────────────────────────

// CreateDeliverableInput carries validated parameters for deliverable creation.
type CreateDeliverableInput struct {
	TenantCtx   workspace.TenantContext
	ProjectID   uuid.UUID
	Title       string
	Description *string
	DueDate     *time.Time
	SortOrder   *int // nil → default 0
	IP          string
	UserAgent   string
	RequestID   string
}

// UpdateDeliverableInput carries validated parameters for deliverable update (PUT full replacement).
type UpdateDeliverableInput struct {
	TenantCtx     workspace.TenantContext
	ProjectID     uuid.UUID
	DeliverableID uuid.UUID
	Title         string
	Description   *string
	DueDate       *time.Time
	SortOrder     *int // nil → default 0
	IP            string
	UserAgent     string
	RequestID     string
}

// ListDeliverablesInput carries the project scope for listing.
type ListDeliverablesInput struct {
	TenantCtx workspace.TenantContext
	ProjectID uuid.UUID
}

// GetDeliverableInput carries the ID lookup parameters.
type GetDeliverableInput struct {
	TenantCtx     workspace.TenantContext
	ProjectID     uuid.UUID
	DeliverableID uuid.UUID
}

// DeleteDeliverableInput carries the ID + transport metadata for delete.
type DeleteDeliverableInput struct {
	TenantCtx     workspace.TenantContext
	ProjectID     uuid.UUID
	DeliverableID uuid.UUID
	IP            string
	UserAgent     string
	RequestID     string
}

// SubmitInput carries a new-round submission. SubmittedBy is NOT here — it is server-derived
// from the actor's project_member (D60/§5.4); any client-supplied value is ignored upstream.
type SubmitInput struct {
	TenantCtx     workspace.TenantContext
	ProjectID     uuid.UUID
	DeliverableID uuid.UUID
	Note          *string
	URL           *string
	IP            string
	UserAgent     string
	RequestID     string
}

// ReviewInput carries a ตรวจรับ decision. ReviewedBy is NOT here — server-derived (D60/§5.4).
type ReviewInput struct {
	TenantCtx     workspace.TenantContext
	ProjectID     uuid.UUID
	DeliverableID uuid.UUID
	SubmissionID  uuid.UUID
	DecisionCode  string
	Comment       *string
	IP            string
	UserAgent     string
	RequestID     string
}

// ── Detail read projection ───────────────────────────────────────────────────────

// DeliverableDetail bundles a deliverable+status with its full submissions history (round DESC).
type DeliverableDetail struct {
	DeliverableWithStatus
	Submissions []SubmissionWithReview
}

// ── authz helpers ────────────────────────────────────────────────────────────────

func isOrgBypass(orgRoleCode string) bool {
	return orgRoleCode == workspace.OrgRoleOwner || orgRoleCode == workspace.OrgRoleAdmin
}

// resolveProjectOrNotFound asserts the project exists in the ws (else ErrProjectNotFound),
// then resolves the actor's active project_member (nil = non-member). It is the single
// entry gate for every deliverable action.
func (s *Service) resolveProjectOrNotFound(ctx context.Context, tc workspace.TenantContext, projectID uuid.UUID) (*ActorMember, error) {
	exists, err := s.access.ProjectExistsForWorkspace(ctx, tc.WorkspaceID, projectID)
	if err != nil {
		return nil, fmt.Errorf("check project exists: %w", err)
	}
	if !exists {
		return nil, ErrProjectNotFound
	}
	member, err := s.access.FindActiveMemberByMembership(ctx, tc.WorkspaceID, projectID, tc.MembershipID)
	if err != nil {
		return nil, fmt.Errorf("resolve actor project_member: %w", err)
	}
	return member, nil
}

// canMutateDeliverable: org Owner/Admin OR project_owner/project_manager active member.
func canMutateDeliverable(orgRoleCode string, member *ActorMember) bool {
	if isOrgBypass(orgRoleCode) {
		return true
	}
	if member == nil {
		return false
	}
	return member.ProjectRoleCode == ProjectRoleOwner || member.ProjectRoleCode == ProjectRoleManager
}

// canReview: same set as deliverable mutate (org Owner/Admin OR project_owner/project_manager).
func canReview(orgRoleCode string, member *ActorMember) bool {
	return canMutateDeliverable(orgRoleCode, member)
}

// canRead: any active project member (incl. viewer), OR org Owner/Admin.
func canRead(orgRoleCode string, member *ActorMember) bool {
	if isOrgBypass(orgRoleCode) {
		return true
	}
	return member != nil
}

// canSubmit: org Owner/Admin OR active project member with role != viewer.
func canSubmit(orgRoleCode string, member *ActorMember) bool {
	if isOrgBypass(orgRoleCode) {
		return true
	}
	if member == nil {
		return false
	}
	return member.ProjectRoleCode != ProjectRoleViewer
}

// ── Deliverable CRUD ───────────────────────────────────────────────────────────

// CreateDeliverable validates input, gates authz, and creates a deliverable + audit row.
func (s *Service) CreateDeliverable(ctx context.Context, in CreateDeliverableInput) (*Deliverable, error) {
	member, err := s.resolveProjectOrNotFound(ctx, in.TenantCtx, in.ProjectID)
	if err != nil {
		return nil, err
	}
	if !canRead(in.TenantCtx.OrgRoleCode, member) {
		// non-member → 404 probe-collapse (D42): do not reveal the project's deliverables exist.
		return nil, ErrProjectNotFound
	}
	if !canMutateDeliverable(in.TenantCtx.OrgRoleCode, member) {
		return nil, ErrForbidden
	}

	title := strings.TrimSpace(in.Title)
	fields := validateDeliverableFields(title, in.Description, in.SortOrder)
	if len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}

	id, err := ids.New()
	if err != nil {
		return nil, fmt.Errorf("generate deliverable id: %w", err)
	}
	now := time.Now().UTC()
	sortOrder := 0
	if in.SortOrder != nil {
		sortOrder = *in.SortOrder
	}

	d := Deliverable{
		ID:          id,
		WorkspaceID: in.TenantCtx.WorkspaceID,
		ProjectID:   in.ProjectID,
		Title:       title,
		Description: normalizeDescription(in.Description),
		DueDate:     in.DueDate,
		SortOrder:   sortOrder,
		CreatedAt:   now,
		CreatedBy:   in.TenantCtx.AccountID,
		UpdatedAt:   now,
		UpdatedBy:   nil,
	}

	entry := s.auditEntry(in.TenantCtx, in.ProjectID, AuditActionDeliverableCreate, AuditResourceTypeDeliverable, d.ID, nil, snapshotDeliverable(d), in.IP, in.UserAgent, in.RequestID)
	if err := s.repo.CreateWithAudit(ctx, d, entry); err != nil {
		return nil, err
	}
	return &d, nil
}

// ListDeliverables returns the project's deliverables with derived status (no pagination).
func (s *Service) ListDeliverables(ctx context.Context, in ListDeliverablesInput) ([]DeliverableWithStatus, error) {
	member, err := s.resolveProjectOrNotFound(ctx, in.TenantCtx, in.ProjectID)
	if err != nil {
		return nil, err
	}
	if !canRead(in.TenantCtx.OrgRoleCode, member) {
		return nil, ErrProjectNotFound
	}
	return s.repo.ListByProjectWithStatus(ctx, in.TenantCtx.WorkspaceID, in.ProjectID)
}

// GetDeliverable returns one deliverable + status + full submissions history.
func (s *Service) GetDeliverable(ctx context.Context, in GetDeliverableInput) (*DeliverableDetail, error) {
	member, err := s.resolveProjectOrNotFound(ctx, in.TenantCtx, in.ProjectID)
	if err != nil {
		return nil, err
	}
	if !canRead(in.TenantCtx.OrgRoleCode, member) {
		return nil, ErrProjectNotFound
	}

	dws, err := s.repo.GetWithStatus(ctx, in.TenantCtx.WorkspaceID, in.DeliverableID)
	if err != nil {
		return nil, err
	}
	if dws == nil || dws.ProjectID != in.ProjectID {
		// not in this ws, soft-deleted, or belongs to a different project → 404 probe-collapse.
		return nil, ErrDeliverableNotFound
	}

	subs, err := s.repo.ListSubmissionsForDeliverable(ctx, in.TenantCtx.WorkspaceID, in.DeliverableID)
	if err != nil {
		return nil, err
	}
	return &DeliverableDetail{DeliverableWithStatus: *dws, Submissions: subs}, nil
}

// UpdateDeliverable performs a PUT-style full replacement of title/description/due_date/sort_order.
func (s *Service) UpdateDeliverable(ctx context.Context, in UpdateDeliverableInput) (*Deliverable, error) {
	member, err := s.resolveProjectOrNotFound(ctx, in.TenantCtx, in.ProjectID)
	if err != nil {
		return nil, err
	}
	if !canRead(in.TenantCtx.OrgRoleCode, member) {
		return nil, ErrProjectNotFound
	}
	if !canMutateDeliverable(in.TenantCtx.OrgRoleCode, member) {
		return nil, ErrForbidden
	}

	existing, err := s.repo.FindByIDForWorkspace(ctx, in.TenantCtx.WorkspaceID, in.DeliverableID)
	if err != nil {
		return nil, err
	}
	if existing == nil || existing.ProjectID != in.ProjectID {
		return nil, ErrDeliverableNotFound
	}

	title := strings.TrimSpace(in.Title)
	fields := validateDeliverableFields(title, in.Description, in.SortOrder)
	if len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}

	now := time.Now().UTC()
	actorID := in.TenantCtx.AccountID
	sortOrder := 0
	if in.SortOrder != nil {
		sortOrder = *in.SortOrder
	}

	updated := Deliverable{
		ID:          existing.ID,
		WorkspaceID: existing.WorkspaceID,
		ProjectID:   existing.ProjectID,
		Title:       title,
		Description: normalizeDescription(in.Description),
		DueDate:     in.DueDate,
		SortOrder:   sortOrder,
		CreatedAt:   existing.CreatedAt,
		CreatedBy:   existing.CreatedBy,
		UpdatedAt:   now,
		UpdatedBy:   &actorID,
	}

	entry := s.auditEntry(in.TenantCtx, in.ProjectID, AuditActionDeliverableUpdate, AuditResourceTypeDeliverable, updated.ID, snapshotDeliverable(*existing), snapshotDeliverable(updated), in.IP, in.UserAgent, in.RequestID)
	if err := s.repo.UpdateWithAudit(ctx, updated, entry); err != nil {
		return nil, err
	}
	return &updated, nil
}

// DeleteDeliverable soft-deletes the deliverable (allowed even with submissions).
func (s *Service) DeleteDeliverable(ctx context.Context, in DeleteDeliverableInput) error {
	member, err := s.resolveProjectOrNotFound(ctx, in.TenantCtx, in.ProjectID)
	if err != nil {
		return err
	}
	if !canRead(in.TenantCtx.OrgRoleCode, member) {
		return ErrProjectNotFound
	}
	if !canMutateDeliverable(in.TenantCtx.OrgRoleCode, member) {
		return ErrForbidden
	}

	existing, err := s.repo.FindByIDForWorkspace(ctx, in.TenantCtx.WorkspaceID, in.DeliverableID)
	if err != nil {
		return err
	}
	if existing == nil || existing.ProjectID != in.ProjectID {
		return ErrDeliverableNotFound
	}

	entry := s.auditEntry(in.TenantCtx, in.ProjectID, AuditActionDeliverableDelete, AuditResourceTypeDeliverable, existing.ID, snapshotDeliverable(*existing), nil, in.IP, in.UserAgent, in.RequestID)
	return s.repo.SoftDeleteWithAudit(ctx, in.TenantCtx.WorkspaceID, in.DeliverableID, in.TenantCtx.AccountID, entry)
}

// ── Submit ─────────────────────────────────────────────────────────────────────

// Submit appends a new submission round under the deliverable's serialization lock (D64).
// submitted_by is server-derived from the actor's project_member (D60); body-supplied values
// are never read by this method.
func (s *Service) Submit(ctx context.Context, in SubmitInput) (*Submission, error) {
	member, err := s.resolveProjectOrNotFound(ctx, in.TenantCtx, in.ProjectID)
	if err != nil {
		return nil, err
	}
	if !canRead(in.TenantCtx.OrgRoleCode, member) {
		return nil, ErrProjectNotFound
	}
	if !canSubmit(in.TenantCtx.OrgRoleCode, member) {
		return nil, ErrForbidden
	}
	// FK requires a project_member row for submitted_by — an org Owner/Admin who is not an
	// active project member cannot submit (decision this round): 422 not_project_member.
	if member == nil {
		return nil, ErrNotProjectMember
	}

	var fields []FieldError
	if in.Note != nil && len(*in.Note) > 10000 {
		fields = append(fields, FieldError{Field: "note", Message: "must be 10000 characters or fewer"})
	}
	if in.URL != nil && len(*in.URL) > 2000 {
		fields = append(fields, FieldError{Field: "url", Message: "must be 2000 characters or fewer"})
	}
	if len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}

	id, err := ids.New()
	if err != nil {
		return nil, fmt.Errorf("generate submission id: %w", err)
	}
	now := time.Now().UTC()
	sub := Submission{
		ID:            id,
		WorkspaceID:   in.TenantCtx.WorkspaceID,
		ProjectID:     in.ProjectID,
		DeliverableID: in.DeliverableID,
		// RoundNo assigned by the repo under FOR UPDATE (MAX+1); placeholder here.
		Note:        normalizeText(in.Note),
		URL:         normalizeText(in.URL),
		SubmittedBy: member.ID, // server-derived (D60)
		SubmittedAt: now,
		CreatedAt:   now,
		CreatedBy:   in.TenantCtx.AccountID,
	}

	// Audit ResourceID = submission id; bind project_id (invariant #6). NewValue snapshot is
	// finalized by the repo once round_no is known (the repo re-stamps round_no into the entry).
	entry := s.auditEntry(in.TenantCtx, in.ProjectID, AuditActionSubmissionCreate, AuditResourceTypeSubmission, sub.ID, nil, snapshotSubmission(sub), in.IP, in.UserAgent, in.RequestID)
	created, err := s.repo.CreateSubmissionLocked(ctx, sub, entry)
	if err != nil {
		return nil, err
	}
	return created, nil
}

// ── Review (ตรวจรับ) ──────────────────────────────────────────────────────────

// Review records a ตรวจรับ verdict under the deliverable's serialization lock (D64).
// reviewed_by is server-derived (D60). decision_code + self-review checks run before the
// locked repo call; latest-only + already-reviewed checks run under the lock in the repo.
func (s *Service) Review(ctx context.Context, in ReviewInput) (*SubmissionReview, error) {
	member, err := s.resolveProjectOrNotFound(ctx, in.TenantCtx, in.ProjectID)
	if err != nil {
		return nil, err
	}
	if !canRead(in.TenantCtx.OrgRoleCode, member) {
		return nil, ErrProjectNotFound
	}
	if !canReview(in.TenantCtx.OrgRoleCode, member) {
		return nil, ErrForbidden
	}
	if member == nil {
		// org Owner/Admin non-member cannot review (FK needs project_member row).
		return nil, ErrNotProjectMember
	}

	decision := strings.TrimSpace(in.DecisionCode)
	if decision == "" {
		return nil, &ValidationError{Fields: []FieldError{{Field: "decision_code", Message: "must not be empty"}}}
	}
	if in.Comment != nil && len(*in.Comment) > 10000 {
		return nil, &ValidationError{Fields: []FieldError{{Field: "comment", Message: "must be 10000 characters or fewer"}}}
	}

	ok, err := s.masters.IsActiveSubmissionDecisionCode(ctx, decision)
	if err != nil {
		return nil, fmt.Errorf("check decision_code: %w", err)
	}
	if !ok {
		return nil, ErrInvalidMasterCode
	}

	// Self-review guard (OQ-6) is enforced UNDER THE LOCK by the repo: the submission's
	// submitted_by is only known once the deliverable row is locked, so comparing it against
	// the reviewer (review.ReviewedBy) must happen there (race-free). The repo holds the
	// forbidSelfReview flag (wired from config at construction) and returns
	// ErrSelfReviewForbidden when flag=true and submitted_by == reviewed_by. Default flag=false
	// = self-review allowed, so the common path performs no extra comparison.

	id, err := ids.New()
	if err != nil {
		return nil, fmt.Errorf("generate review id: %w", err)
	}
	now := time.Now().UTC()
	review := SubmissionReview{
		ID:           id,
		WorkspaceID:  in.TenantCtx.WorkspaceID,
		ProjectID:    in.ProjectID,
		SubmissionID: in.SubmissionID,
		DecisionCode: decision,
		Comment:      normalizeText(in.Comment),
		ReviewedBy:   member.ID, // server-derived (D60)
		ReviewedAt:   now,
		CreatedAt:    now,
		CreatedBy:    in.TenantCtx.AccountID,
	}

	entry := s.auditEntry(in.TenantCtx, in.ProjectID, AuditActionReviewCreate, AuditResourceTypeReview, review.ID, nil, snapshotReview(review), in.IP, in.UserAgent, in.RequestID)
	created, err := s.repo.CreateReviewLocked(ctx, in.DeliverableID, review, entry)
	if err != nil {
		return nil, err
	}
	return created, nil
}

// ── validation + snapshot helpers ────────────────────────────────────────────────

func validateDeliverableFields(title string, description *string, sortOrder *int) []FieldError {
	var fields []FieldError
	if title == "" {
		fields = append(fields, FieldError{Field: "title", Message: "must not be empty"})
	} else if len(title) > 200 {
		fields = append(fields, FieldError{Field: "title", Message: "must be 200 characters or fewer"})
	}
	if description != nil && len(*description) > 10000 {
		fields = append(fields, FieldError{Field: "description", Message: "must be 10000 characters or fewer"})
	}
	if sortOrder != nil && *sortOrder < 0 {
		fields = append(fields, FieldError{Field: "sort_order", Message: "must be >= 0"})
	}
	return fields
}

// normalizeDescription trims to nil when the trimmed value is empty (clear-to-NULL on PUT).
func normalizeDescription(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// normalizeText trims a nullable text field; empty → nil. url is NOT URL-parsed (text-only D8).
func normalizeText(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func (s *Service) auditEntry(tc workspace.TenantContext, projectID uuid.UUID, action, resourceType string, resourceID uuid.UUID, oldVal, newVal map[string]any, ip, ua, reqID string) audit.Entry {
	wsID := tc.WorkspaceID
	pID := projectID
	actorID := tc.AccountID
	rID := resourceID
	return audit.Entry{
		WorkspaceID:    &wsID,
		ProjectID:      &pID,
		ActorAccountID: &actorID,
		Action:         action,
		ResourceType:   resourceType,
		ResourceID:     &rID,
		OldValue:       oldVal,
		NewValue:       newVal,
		Result:         AuditResultSuccess,
		IP:             ip,
		UserAgent:      ua,
		RequestID:      reqID,
	}
}

func snapshotDeliverable(d Deliverable) map[string]any {
	out := map[string]any{
		"title":      d.Title,
		"sort_order": d.SortOrder,
	}
	if d.Description != nil {
		out["description"] = *d.Description
	} else {
		out["description"] = nil
	}
	if d.DueDate != nil {
		out["due_date"] = d.DueDate.UTC().Format("2006-01-02")
	} else {
		out["due_date"] = nil
	}
	return out
}

func snapshotSubmission(s Submission) map[string]any {
	out := map[string]any{
		"deliverable_id": s.DeliverableID.String(),
		"round_no":       s.RoundNo,
		"submitted_by":   s.SubmittedBy.String(),
	}
	if s.Note != nil {
		out["note"] = *s.Note
	} else {
		out["note"] = nil
	}
	if s.URL != nil {
		out["url"] = *s.URL
	} else {
		out["url"] = nil
	}
	return out
}

func snapshotReview(r SubmissionReview) map[string]any {
	out := map[string]any{
		"submission_id": r.SubmissionID.String(),
		"decision_code": r.DecisionCode,
		"reviewed_by":   r.ReviewedBy.String(),
	}
	if r.Comment != nil {
		out["comment"] = *r.Comment
	} else {
		out["comment"] = nil
	}
	return out
}
