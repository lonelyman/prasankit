package project

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

// Service implements project use cases.
type Service struct {
	projects ProjectRepository
	masters  MasterRepository
}

// NewService constructs a project Service.
func NewService(projects ProjectRepository, masters MasterRepository) *Service {
	return &Service{projects: projects, masters: masters}
}

// ── Inputs ─────────────────────────────────────────────────────────────────────

// CreateProjectInput carries validated parameters for project creation.
type CreateProjectInput struct {
	TenantCtx         workspace.TenantContext
	ProjectName       string
	Slug              string // empty → NULL
	ProjectTypeCode   string
	ProjectStatusCode string
	RequestingUnit    *string
	Description       *string
	StartDate         *time.Time
	EndDate           *time.Time
	IP                string
	UserAgent         string
	RequestID         string
}

// UpdateProjectInput carries validated parameters for project update (PUT — full replacement).
// NOTE: NO ProjectStatusCode field — status changes use ChangeStatus.
type UpdateProjectInput struct {
	TenantCtx       workspace.TenantContext
	ProjectID       uuid.UUID
	ProjectName     string
	Slug            string // "" → clear to NULL
	ProjectTypeCode string
	RequestingUnit  *string
	Description     *string
	StartDate       *time.Time
	EndDate         *time.Time
	IP              string
	UserAgent       string
	RequestID       string
}

// ListProjectsInput carries pagination + filter parameters for ListProjects.
type ListProjectsInput struct {
	TenantCtx  workspace.TenantContext
	Page       int
	Limit      int
	StatusCode string
	TypeCode   string
}

// GetProjectInput carries the ID lookup parameters.
type GetProjectInput struct {
	TenantCtx workspace.TenantContext
	ProjectID uuid.UUID
}

// DeleteProjectInput carries the ID and transport metadata for delete.
type DeleteProjectInput struct {
	TenantCtx workspace.TenantContext
	ProjectID uuid.UUID
	IP        string
	UserAgent string
	RequestID string
}

// ChangeStatusInput carries the status change request.
type ChangeStatusInput struct {
	TenantCtx     workspace.TenantContext
	ProjectID     uuid.UUID
	NewStatusCode string
	IP            string
	UserAgent     string
	RequestID     string
}

// ── Methods ────────────────────────────────────────────────────────────────────

// CreateProject validates input and atomically creates a project + audit log entry.
func (s *Service) CreateProject(ctx context.Context, in CreateProjectInput) (*Project, error) {
	name := strings.TrimSpace(in.ProjectName)
	slugNorm := strings.ToLower(strings.TrimSpace(in.Slug))

	var fields []FieldError
	// project_name
	if name == "" {
		fields = append(fields, FieldError{Field: "project_name", Message: "must not be empty"})
	} else if len(name) > 200 {
		fields = append(fields, FieldError{Field: "project_name", Message: "must be 200 characters or fewer"})
	}
	// slug — optional; if non-empty, must match
	if slugNorm != "" && !slugPattern.MatchString(slugNorm) {
		fields = append(fields, FieldError{Field: "slug", Message: "must match ^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$"})
	}
	// project_type_code / project_status_code (non-empty)
	if strings.TrimSpace(in.ProjectTypeCode) == "" {
		fields = append(fields, FieldError{Field: "project_type_code", Message: "must not be empty"})
	}
	if strings.TrimSpace(in.ProjectStatusCode) == "" {
		fields = append(fields, FieldError{Field: "project_status_code", Message: "must not be empty"})
	}
	// requesting_unit nullable
	if in.RequestingUnit != nil {
		ru := strings.TrimSpace(*in.RequestingUnit)
		if len(ru) < 1 || len(ru) > 200 {
			fields = append(fields, FieldError{Field: "requesting_unit", Message: "must be between 1 and 200 characters"})
		}
	}
	// description nullable
	if in.Description != nil {
		if len(*in.Description) > 10000 {
			fields = append(fields, FieldError{Field: "description", Message: "must be 10000 characters or fewer"})
		}
	}
	// dates
	if in.StartDate != nil && in.EndDate != nil && in.StartDate.After(*in.EndDate) {
		fields = append(fields, FieldError{Field: "end_date", Message: "must be on or after start_date"})
	}

	if len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}

	// Master pre-checks (D28).
	ok, err := s.masters.IsActiveProjectStatusCode(ctx, in.ProjectStatusCode)
	if err != nil {
		return nil, fmt.Errorf("check project_status_code: %w", err)
	}
	if !ok {
		return nil, ErrInvalidStatusCode
	}
	ok, err = s.masters.IsActiveProjectTypeCode(ctx, in.ProjectTypeCode)
	if err != nil {
		return nil, fmt.Errorf("check project_type_code: %w", err)
	}
	if !ok {
		return nil, ErrInvalidTypeCode
	}

	id, err := ids.New()
	if err != nil {
		return nil, fmt.Errorf("generate project id: %w", err)
	}
	now := time.Now().UTC()

	var slugPtr *string
	if slugNorm != "" {
		s := slugNorm
		slugPtr = &s
	}

	p := Project{
		ID:                   id,
		WorkspaceID:          in.TenantCtx.WorkspaceID,
		ProjectName:          name,
		Slug:                 slugPtr,
		ProjectTypeCode:      in.ProjectTypeCode,
		ProjectStatusCode:    in.ProjectStatusCode,
		OwnerProjectMemberID: nil, // ALWAYS nil in 6a
		RequestingUnit:       in.RequestingUnit,
		Description:          in.Description,
		StartDate:            in.StartDate,
		EndDate:              in.EndDate,
		CreatedAt:            now,
		CreatedBy:            in.TenantCtx.AccountID,
		UpdatedAt:            now,
		UpdatedBy:            nil,
	}

	wsID := in.TenantCtx.WorkspaceID
	projectID := id
	actorID := in.TenantCtx.AccountID
	resourceID := id
	entry := audit.Entry{
		WorkspaceID:    &wsID,
		ProjectID:      &projectID,
		ActorAccountID: &actorID,
		Action:         AuditActionProjectCreate,
		ResourceType:   AuditResourceTypeProject,
		ResourceID:     &resourceID,
		OldValue:       nil,
		NewValue:       snapshotProject(p),
		Result:         AuditResultSuccess,
		IP:             in.IP,
		UserAgent:      in.UserAgent,
		RequestID:      in.RequestID,
	}

	if err := s.projects.CreateWithAudit(ctx, p, entry); err != nil {
		return nil, err
	}
	return &p, nil
}

// ListProjects validates pagination + filter input and returns the page.
func (s *Service) ListProjects(ctx context.Context, in ListProjectsInput) ([]Project, int64, error) {
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
	if len(fields) > 0 {
		return nil, 0, &ValidationError{Fields: fields}
	}

	return s.projects.ListByWorkspace(ctx, in.TenantCtx.WorkspaceID, ListOptions{
		Page:       page,
		Limit:      limit,
		StatusCode: in.StatusCode,
		TypeCode:   in.TypeCode,
	})
}

// GetProject returns the project bound to TenantContext's workspace.
func (s *Service) GetProject(ctx context.Context, in GetProjectInput) (*Project, error) {
	p, err := s.projects.FindByIDForWorkspace(ctx, in.TenantCtx.WorkspaceID, in.ProjectID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrProjectNotFound
	}
	return p, nil
}

// UpdateProject performs a PUT-style full replacement.
// project_status_code is NOT updated here — use ChangeStatus.
func (s *Service) UpdateProject(ctx context.Context, in UpdateProjectInput) (*Project, error) {
	existing, err := s.projects.FindByIDForWorkspace(ctx, in.TenantCtx.WorkspaceID, in.ProjectID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrProjectNotFound
	}

	name := strings.TrimSpace(in.ProjectName)
	slugNorm := strings.ToLower(strings.TrimSpace(in.Slug))

	var fields []FieldError
	if name == "" {
		fields = append(fields, FieldError{Field: "project_name", Message: "must not be empty"})
	} else if len(name) > 200 {
		fields = append(fields, FieldError{Field: "project_name", Message: "must be 200 characters or fewer"})
	}
	if slugNorm != "" && !slugPattern.MatchString(slugNorm) {
		fields = append(fields, FieldError{Field: "slug", Message: "must match ^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$"})
	}
	if strings.TrimSpace(in.ProjectTypeCode) == "" {
		fields = append(fields, FieldError{Field: "project_type_code", Message: "must not be empty"})
	}
	if in.RequestingUnit != nil {
		ru := strings.TrimSpace(*in.RequestingUnit)
		if len(ru) < 1 || len(ru) > 200 {
			fields = append(fields, FieldError{Field: "requesting_unit", Message: "must be between 1 and 200 characters"})
		}
	}
	if in.Description != nil && len(*in.Description) > 10000 {
		fields = append(fields, FieldError{Field: "description", Message: "must be 10000 characters or fewer"})
	}
	if in.StartDate != nil && in.EndDate != nil && in.StartDate.After(*in.EndDate) {
		fields = append(fields, FieldError{Field: "end_date", Message: "must be on or after start_date"})
	}
	if len(fields) > 0 {
		return nil, &ValidationError{Fields: fields}
	}

	ok, err := s.masters.IsActiveProjectTypeCode(ctx, in.ProjectTypeCode)
	if err != nil {
		return nil, fmt.Errorf("check project_type_code: %w", err)
	}
	if !ok {
		return nil, ErrInvalidTypeCode
	}

	var slugPtr *string
	if slugNorm != "" {
		s := slugNorm
		slugPtr = &s
	}

	now := time.Now().UTC()
	actorID := in.TenantCtx.AccountID

	updated := Project{
		ID:                   existing.ID,
		WorkspaceID:          existing.WorkspaceID,
		ProjectName:          name,
		Slug:                 slugPtr,
		ProjectTypeCode:      in.ProjectTypeCode,
		ProjectStatusCode:    existing.ProjectStatusCode, // unchanged on PUT
		OwnerProjectMemberID: existing.OwnerProjectMemberID,
		RequestingUnit:       in.RequestingUnit,
		Description:          in.Description,
		StartDate:            in.StartDate,
		EndDate:              in.EndDate,
		CreatedAt:            existing.CreatedAt,
		CreatedBy:            existing.CreatedBy,
		UpdatedAt:            now,
		UpdatedBy:            &actorID,
	}

	wsID := in.TenantCtx.WorkspaceID
	projectID := existing.ID
	resourceID := existing.ID
	entry := audit.Entry{
		WorkspaceID:    &wsID,
		ProjectID:      &projectID,
		ActorAccountID: &actorID,
		Action:         AuditActionProjectUpdate,
		ResourceType:   AuditResourceTypeProject,
		ResourceID:     &resourceID,
		OldValue:       snapshotProject(*existing),
		NewValue:       snapshotProject(updated),
		Result:         AuditResultSuccess,
		IP:             in.IP,
		UserAgent:      in.UserAgent,
		RequestID:      in.RequestID,
	}

	if err := s.projects.UpdateWithAudit(ctx, updated, entry); err != nil {
		return nil, err
	}
	return &updated, nil
}

// ChangeStatus changes the project's status code (D44 free transitions — no state machine).
func (s *Service) ChangeStatus(ctx context.Context, in ChangeStatusInput) (*Project, error) {
	newCode := strings.TrimSpace(in.NewStatusCode)
	if newCode == "" {
		return nil, &ValidationError{Fields: []FieldError{{Field: "new_status_code", Message: "must not be empty"}}}
	}

	ok, err := s.masters.IsActiveProjectStatusCode(ctx, newCode)
	if err != nil {
		return nil, fmt.Errorf("check project_status_code: %w", err)
	}
	if !ok {
		return nil, ErrInvalidStatusCode
	}

	existing, err := s.projects.FindByIDForWorkspace(ctx, in.TenantCtx.WorkspaceID, in.ProjectID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrProjectNotFound
	}

	// No-op when same — no audit row, no UPDATE.
	if newCode == existing.ProjectStatusCode {
		return existing, nil
	}

	wsID := in.TenantCtx.WorkspaceID
	projectID := existing.ID
	resourceID := existing.ID
	actorID := in.TenantCtx.AccountID
	entry := audit.Entry{
		WorkspaceID:    &wsID,
		ProjectID:      &projectID,
		ActorAccountID: &actorID,
		Action:         AuditActionProjectStatusChange,
		ResourceType:   AuditResourceTypeProject,
		ResourceID:     &resourceID,
		OldValue:       map[string]any{"project_status_code": existing.ProjectStatusCode},
		NewValue:       map[string]any{"project_status_code": newCode},
		Result:         AuditResultSuccess,
		IP:             in.IP,
		UserAgent:      in.UserAgent,
		RequestID:      in.RequestID,
	}

	if err := s.projects.ChangeStatusWithAudit(ctx, in.TenantCtx.WorkspaceID, in.ProjectID, newCode, in.TenantCtx.AccountID, entry); err != nil {
		return nil, err
	}

	refreshed, err := s.projects.FindByIDForWorkspace(ctx, in.TenantCtx.WorkspaceID, in.ProjectID)
	if err != nil {
		return nil, err
	}
	if refreshed == nil {
		return nil, ErrProjectNotFound
	}
	return refreshed, nil
}

// DeleteProject soft-deletes the project.
func (s *Service) DeleteProject(ctx context.Context, in DeleteProjectInput) error {
	existing, err := s.projects.FindByIDForWorkspace(ctx, in.TenantCtx.WorkspaceID, in.ProjectID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrProjectNotFound
	}

	wsID := in.TenantCtx.WorkspaceID
	projectID := existing.ID
	resourceID := existing.ID
	actorID := in.TenantCtx.AccountID
	entry := audit.Entry{
		WorkspaceID:    &wsID,
		ProjectID:      &projectID,
		ActorAccountID: &actorID,
		Action:         AuditActionProjectDelete,
		ResourceType:   AuditResourceTypeProject,
		ResourceID:     &resourceID,
		OldValue:       snapshotProject(*existing),
		NewValue:       nil,
		Result:         AuditResultSuccess,
		IP:             in.IP,
		UserAgent:      in.UserAgent,
		RequestID:      in.RequestID,
	}

	return s.projects.SoftDeleteWithAudit(ctx, in.TenantCtx.WorkspaceID, in.ProjectID, in.TenantCtx.AccountID, entry)
}

// snapshotProject builds a map representation of the project's mutable fields
// for OldValue / NewValue audit logging.
func snapshotProject(p Project) map[string]any {
	out := map[string]any{
		"project_name":        p.ProjectName,
		"project_type_code":   p.ProjectTypeCode,
		"project_status_code": p.ProjectStatusCode,
	}
	if p.Slug != nil {
		out["slug"] = *p.Slug
	} else {
		out["slug"] = nil
	}
	if p.RequestingUnit != nil {
		out["requesting_unit"] = *p.RequestingUnit
	} else {
		out["requesting_unit"] = nil
	}
	if p.Description != nil {
		out["description"] = *p.Description
	} else {
		out["description"] = nil
	}
	if p.StartDate != nil {
		out["start_date"] = p.StartDate.Format("2006-01-02")
	} else {
		out["start_date"] = nil
	}
	if p.EndDate != nil {
		out["end_date"] = p.EndDate.Format("2006-01-02")
	} else {
		out["end_date"] = nil
	}
	return out
}
