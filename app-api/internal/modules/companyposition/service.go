package companyposition

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

// Service implements company_position use cases (master CRUD + ws-attach D41).
type Service struct {
	positions  CompanyPositionRepository
	masters    CompanyPositionMasterRepository
	membership MembershipPositionRepository
}

// NewService constructs a companyposition Service.
func NewService(positions CompanyPositionRepository, masters CompanyPositionMasterRepository, membership MembershipPositionRepository) *Service {
	return &Service{positions: positions, masters: masters, membership: membership}
}

// ── Inputs ─────────────────────────────────────────────────────────────────────

// CreateInput carries validated parameters for position creation.
type CreateInput struct {
	TenantCtx   workspace.TenantContext
	Code        string
	LabelTH     string
	LabelEN     string
	Description *string
	SortOrder   int
	IP          string
	UserAgent   string
	RequestID   string
}

// UpdateInput carries validated parameters for position update.
type UpdateInput struct {
	TenantCtx   workspace.TenantContext
	Code        string // path code — the immutable target key
	BodyCode    string // body code (if present); must equal Code or "" -> else ErrCodeImmutable
	LabelTH     string
	LabelEN     string
	Description *string
	SortOrder   int
	Status      string // "active" | "deprecated"
	IP          string
	UserAgent   string
	RequestID   string
}

// ListInput carries pagination + filter parameters.
type ListInput struct {
	TenantCtx workspace.TenantContext
	Page      int
	Limit     int
	Status    string
}

// GetInput carries the code lookup parameters.
type GetInput struct {
	TenantCtx workspace.TenantContext
	Code      string
}

// DeprecateInput carries the deprecate request.
type DeprecateInput struct {
	TenantCtx workspace.TenantContext
	Code      string
	IP        string
	UserAgent string
	RequestID string
}

// SetCompanyPositionInput carries the ws-attach set/clear request.
// CompanyPositionCode nil = clear.
type SetCompanyPositionInput struct {
	TenantCtx           workspace.TenantContext
	MembershipID        uuid.UUID
	CompanyPositionCode *string
	IP                  string
	UserAgent           string
	RequestID           string
}

// ── Master CRUD ────────────────────────────────────────────────────────────────

// Create validates input and atomically creates a company_position + audit log entry.
func (s *Service) Create(ctx context.Context, in CreateInput) (*CompanyPosition, error) {
	code := strings.ToLower(strings.TrimSpace(in.Code))
	labelTH := strings.TrimSpace(in.LabelTH)
	labelEN := strings.TrimSpace(in.LabelEN)

	fields := validateFields(code, labelTH, labelEN, in.Description, in.SortOrder)
	if len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}

	wsID := in.TenantCtx.WorkspaceID

	existing, err := s.positions.FindByCodeForWorkspace(ctx, wsID, code)
	if err != nil {
		return nil, fmt.Errorf("check existing company_position: %w", err)
	}
	if existing != nil {
		return nil, ErrCodeTaken
	}

	id, err := ids.New()
	if err != nil {
		return nil, fmt.Errorf("generate company_position id: %w", err)
	}
	now := time.Now().UTC()

	p := CompanyPosition{
		ID:          id,
		WorkspaceID: wsID,
		Code:        code,
		LabelTH:     labelTH,
		LabelEN:     labelEN,
		Description: trimDescription(in.Description),
		SortOrder:   in.SortOrder,
		IsSystem:    false,
		Status:      "active",
		CreatedAt:   now,
		CreatedBy:   in.TenantCtx.AccountID,
		UpdatedAt:   now,
		UpdatedBy:   nil,
	}

	wsIDLocal := wsID
	actorID := in.TenantCtx.AccountID
	resourceID := id
	entry := audit.Entry{
		WorkspaceID:    &wsIDLocal,
		ProjectID:      nil, // masters are workspace-scoped (D43)
		ActorAccountID: &actorID,
		Action:         AuditActionCompanyPositionCreate,
		ResourceType:   AuditResourceTypeCompanyPosition,
		ResourceID:     &resourceID,
		OldValue:       nil,
		NewValue:       snapshot(p),
		Result:         AuditResultSuccess,
		IP:             in.IP,
		UserAgent:      in.UserAgent,
		RequestID:      in.RequestID,
	}

	if err := s.positions.CreateWithAudit(ctx, p, entry); err != nil {
		return nil, err
	}
	return &p, nil
}

// List validates pagination + filter input and returns the page.
func (s *Service) List(ctx context.Context, in ListInput) ([]CompanyPosition, int64, error) {
	page := in.Page
	limit := in.Limit
	var fields []FieldError
	if page == 0 {
		page = 1
	} else if page < 1 {
		fields = append(fields, FieldError{Field: "page", Message: "must be >= 1"})
	}
	if limit == 0 {
		limit = 10
	} else if limit < 1 || limit > 100 {
		fields = append(fields, FieldError{Field: "limit", Message: "must be between 1 and 100"})
	}
	status := strings.TrimSpace(in.Status)
	if status != "" && status != "active" && status != "deprecated" {
		fields = append(fields, FieldError{Field: "status", Message: "must be active or deprecated"})
	}
	if len(fields) > 0 {
		return nil, 0, &ValidationError{Fields: fields}
	}

	return s.positions.ListByWorkspace(ctx, in.TenantCtx.WorkspaceID, ListOptions{
		Page:   page,
		Limit:  limit,
		Status: status,
	})
}

// Get returns the position keyed on (workspace_id, code).
func (s *Service) Get(ctx context.Context, in GetInput) (*CompanyPosition, error) {
	code := strings.ToLower(strings.TrimSpace(in.Code))
	p, err := s.positions.FindByCodeForWorkspace(ctx, in.TenantCtx.WorkspaceID, code)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrPositionNotFound
	}
	return p, nil
}

// Update changes label/sort/status/description only — code is immutable.
func (s *Service) Update(ctx context.Context, in UpdateInput) (*CompanyPosition, error) {
	code := strings.ToLower(strings.TrimSpace(in.Code))

	if bodyCode := strings.ToLower(strings.TrimSpace(in.BodyCode)); bodyCode != "" && bodyCode != code {
		return nil, ErrCodeImmutable
	}

	wsID := in.TenantCtx.WorkspaceID

	existing, err := s.positions.FindByCodeForWorkspace(ctx, wsID, code)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrPositionNotFound
	}
	if existing.IsSystem {
		return nil, ErrSystemImmutable
	}

	labelTH := strings.TrimSpace(in.LabelTH)
	labelEN := strings.TrimSpace(in.LabelEN)
	status := strings.TrimSpace(in.Status)

	var fields []FieldError
	if labelTH == "" || len(labelTH) > 200 {
		fields = append(fields, FieldError{Field: "label_th", Message: "must be between 1 and 200 characters"})
	}
	if labelEN == "" || len(labelEN) > 200 {
		fields = append(fields, FieldError{Field: "label_en", Message: "must be between 1 and 200 characters"})
	}
	if in.Description != nil && len(*in.Description) > 10000 {
		fields = append(fields, FieldError{Field: "description", Message: "must be 10000 characters or fewer"})
	}
	if in.SortOrder < 0 {
		fields = append(fields, FieldError{Field: "sort_order", Message: "must be >= 0"})
	}
	if len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}

	if status == "" {
		status = existing.Status
	} else if status != "active" && status != "deprecated" {
		return nil, ErrInvalidStatus
	}

	now := time.Now().UTC()
	actorID := in.TenantCtx.AccountID

	updated := CompanyPosition{
		ID:          existing.ID,
		WorkspaceID: existing.WorkspaceID,
		Code:        existing.Code, // immutable
		LabelTH:     labelTH,
		LabelEN:     labelEN,
		Description: trimDescription(in.Description),
		SortOrder:   in.SortOrder,
		IsSystem:    existing.IsSystem,
		Status:      status,
		CreatedAt:   existing.CreatedAt,
		CreatedBy:   existing.CreatedBy,
		UpdatedAt:   now,
		UpdatedBy:   &actorID,
	}

	wsIDLocal := wsID
	resourceID := existing.ID
	entry := audit.Entry{
		WorkspaceID:    &wsIDLocal,
		ProjectID:      nil,
		ActorAccountID: &actorID,
		Action:         AuditActionCompanyPositionUpdate,
		ResourceType:   AuditResourceTypeCompanyPosition,
		ResourceID:     &resourceID,
		OldValue:       snapshot(*existing),
		NewValue:       snapshot(updated),
		Result:         AuditResultSuccess,
		IP:             in.IP,
		UserAgent:      in.UserAgent,
		RequestID:      in.RequestID,
	}

	if err := s.positions.UpdateWithAudit(ctx, updated, entry); err != nil {
		return nil, err
	}
	return &updated, nil
}

// Deprecate is the "delete" verb — UPDATE status='deprecated' (§2.9, no hard delete).
func (s *Service) Deprecate(ctx context.Context, in DeprecateInput) (*CompanyPosition, error) {
	code := strings.ToLower(strings.TrimSpace(in.Code))
	wsID := in.TenantCtx.WorkspaceID

	existing, err := s.positions.FindByCodeForWorkspace(ctx, wsID, code)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrPositionNotFound
	}
	if existing.IsSystem {
		return nil, ErrSystemImmutable
	}

	if existing.Status == "deprecated" {
		return existing, nil
	}

	wsIDLocal := wsID
	actorID := in.TenantCtx.AccountID
	resourceID := existing.ID
	entry := audit.Entry{
		WorkspaceID:    &wsIDLocal,
		ProjectID:      nil,
		ActorAccountID: &actorID,
		Action:         AuditActionCompanyPositionDeprecate,
		ResourceType:   AuditResourceTypeCompanyPosition,
		ResourceID:     &resourceID,
		OldValue:       map[string]any{"status": "active"},
		NewValue:       map[string]any{"status": "deprecated"},
		Result:         AuditResultSuccess,
		IP:             in.IP,
		UserAgent:      in.UserAgent,
		RequestID:      in.RequestID,
	}

	if err := s.positions.DeprecateWithAudit(ctx, wsID, code, actorID, entry); err != nil {
		return nil, err
	}

	refreshed, err := s.positions.FindByCodeForWorkspace(ctx, wsID, code)
	if err != nil {
		return nil, err
	}
	if refreshed == nil {
		return nil, ErrPositionNotFound
	}
	return refreshed, nil
}

// ── ws company_position attach (D41) ─────────────────────────────────────────────

// SetCompanyPosition sets (or clears, when CompanyPositionCode is nil) a membership's
// company_position_code. Workspace-scoped audit (ProjectID=nil). The OldValue is read
// inside the adapter's tx (the affected row's prior company_position_code).
func (s *Service) SetCompanyPosition(ctx context.Context, in SetCompanyPositionInput) error {
	wsID := in.TenantCtx.WorkspaceID

	var codePtr *string
	if in.CompanyPositionCode != nil {
		code := strings.ToLower(strings.TrimSpace(*in.CompanyPositionCode))
		if code == "" {
			// empty string -> clear (treat as nil).
			codePtr = nil
		} else {
			ok, err := s.masters.IsActiveCompanyPositionCode(ctx, wsID, code)
			if err != nil {
				return fmt.Errorf("check company_position_code: %w", err)
			}
			if !ok {
				return ErrInvalidPositionCode
			}
			codePtr = &code
		}
	}

	wsIDLocal := wsID
	actorID := in.TenantCtx.AccountID
	resourceID := in.MembershipID
	var newValue map[string]any
	if codePtr != nil {
		newValue = map[string]any{"company_position_code": *codePtr}
	} else {
		// null NewValue = clear.
		newValue = nil
	}
	entry := audit.Entry{
		WorkspaceID:    &wsIDLocal,
		ProjectID:      nil, // membership row, no project (D43)
		ActorAccountID: &actorID,
		Action:         AuditActionMembershipCompanyPositionChange,
		ResourceType:   AuditResourceTypeWorkspaceMembership,
		ResourceID:     &resourceID,
		OldValue:       nil, // populated by the adapter from the affected row's prior value
		NewValue:       newValue,
		Result:         AuditResultSuccess,
		IP:             in.IP,
		UserAgent:      in.UserAgent,
		RequestID:      in.RequestID,
	}

	return s.membership.SetCompanyPositionWithAudit(ctx, wsID, in.MembershipID, codePtr, actorID, entry)
}

// ── helpers ────────────────────────────────────────────────────────────────────

// validateFields accumulates field errors for create.
func validateFields(code, labelTH, labelEN string, description *string, sortOrder int) []FieldError {
	var fields []FieldError
	if code == "" {
		fields = append(fields, FieldError{Field: "code", Message: "must not be empty"})
	} else if !codePattern.MatchString(code) {
		fields = append(fields, FieldError{Field: "code", Message: "must match ^[a-z][a-z0-9_]{1,62}$"})
	}
	if labelTH == "" || len(labelTH) > 200 {
		fields = append(fields, FieldError{Field: "label_th", Message: "must be between 1 and 200 characters"})
	}
	if labelEN == "" || len(labelEN) > 200 {
		fields = append(fields, FieldError{Field: "label_en", Message: "must be between 1 and 200 characters"})
	}
	if description != nil && len(*description) > 10000 {
		fields = append(fields, FieldError{Field: "description", Message: "must be 10000 characters or fewer"})
	}
	if sortOrder < 0 {
		fields = append(fields, FieldError{Field: "sort_order", Message: "must be >= 0"})
	}
	return fields
}

// trimDescription trims a non-nil description, preserving nil.
func trimDescription(d *string) *string {
	if d == nil {
		return nil
	}
	t := strings.TrimSpace(*d)
	return &t
}

// snapshot builds a map representation of the position's mutable fields for audit logging.
func snapshot(p CompanyPosition) map[string]any {
	out := map[string]any{
		"code":       p.Code,
		"label_th":   p.LabelTH,
		"label_en":   p.LabelEN,
		"sort_order": p.SortOrder,
		"status":     p.Status,
	}
	if p.Description != nil {
		out["description"] = *p.Description
	} else {
		out["description"] = nil
	}
	return out
}
