package projectsvc

import (
	"context"
	"errors"
	"strings"
	"time"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/project"
	"prasankit-api/internal/modules/workspace"

	"github.com/google/uuid"
)

const (
	defaultProjectCodePrefix = "PRJ"
	defaultProjectCodeLength = 4
)

var (
	ErrAccountRequired               = errors.New("account is required")
	ErrAccountInactive               = errors.New("account is inactive")
	ErrTenantContextRequired         = errors.New("tenant context is required")
	ErrProjectNameRequired           = errors.New("project name is required")
	ErrProjectTypeInvalid            = errors.New("project type is invalid")
	ErrProjectIDRequired             = errors.New("project id is required")
	ErrProjectUpdateNoFields         = errors.New("project update has no fields")
	ErrProjectRoleMissing            = errors.New("project role is missing")
	ErrProjectRoleInvalid            = errors.New("project role is invalid")
	ErrProjectPriorityMissing        = errors.New("project priority is missing")
	ErrProjectPriorityInvalid        = errors.New("project priority is invalid")
	ErrProjectNotFound               = errors.New("project not found")
	ErrWorkspaceMembershipIDRequired = errors.New("workspace membership id is required")
	ErrWorkspaceMembershipNotFound   = errors.New("workspace membership not found")
	ErrProjectMemberAlreadyExists    = errors.New("project member already exists")
	ErrProjectMemberCreateFail       = errors.New("project member create failed")
	ErrProjectMemberIDRequired       = errors.New("project member id is required")
	ErrProjectMemberUpdateNoFields   = errors.New("project member update has no fields")
	ErrProjectMemberNotFound         = errors.New("project member not found")
	ErrProjectMemberLastOwner        = errors.New("project member is the last project owner")
)

type CreateProjectInput struct {
	Account                auth.UserAccount
	TenantContext          workspace.TenantContext
	Name                   string
	Type                   project.ProjectType
	Description            string
	ClientOrRequestingUnit string
	ScopeOrObjective       string
}

type CreateProjectResult struct {
	Project project.Project
	Member  project.Member
}

type ListProjectsInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	Limit         int
	Offset        int
}

type ListProjectsResult struct {
	Items []project.ProjectWithMember
	Total int
}

type GetProjectInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
}

type GetProjectResult struct {
	Item project.ProjectWithMember
}

type UpdateProjectInput struct {
	Account                auth.UserAccount
	TenantContext          workspace.TenantContext
	ProjectID              uuid.UUID
	Name                   *string
	Type                   *project.ProjectType
	Priority               *project.ProjectPriority
	Description            *string
	ClientOrRequestingUnit *string
	ScopeOrObjective       *string
}

type UpdateProjectResult struct {
	Item project.ProjectWithMember
}

type ListProjectMembersInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
	Limit         int
	Offset        int
}

type ListProjectMembersResult struct {
	Items []project.Member
	Total int
}

type AddProjectMemberInput struct {
	Account               auth.UserAccount
	TenantContext         workspace.TenantContext
	ProjectID             uuid.UUID
	WorkspaceMembershipID uuid.UUID
	Role                  project.ProjectRole
}

type AddProjectMemberResult struct {
	Member project.Member
}

type UpdateProjectMemberInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
	MemberID      uuid.UUID
	Role          *project.ProjectRole
}

type UpdateProjectMemberResult struct {
	Member project.Member
}

type RemoveProjectMemberInput struct {
	Account       auth.UserAccount
	TenantContext workspace.TenantContext
	ProjectID     uuid.UUID
	MemberID      uuid.UUID
}

type Service struct {
	repository Repository
	clock      func() time.Time
}

type Repository interface {
	project.Repository
}

func NewService(repository Repository) *Service {
	return &Service{
		repository: repository,
		clock:      func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) CreateProject(ctx context.Context, input CreateProjectInput) (*CreateProjectResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrProjectNameRequired
	}

	projectType := input.Type
	if projectType == "" {
		projectType = project.ProjectTypeInternal
	}
	if projectType != project.ProjectTypeInternal && projectType != project.ProjectTypeClient {
		return nil, ErrProjectTypeInvalid
	}

	now := s.clock()
	year := now.Year()
	var createdProject project.Project
	var ownerMember project.Member

	err := s.repository.WithinTransaction(ctx, func(ctx context.Context, repo project.Repository) error {
		ownerRole, err := repo.FindProjectRoleByCode(ctx, project.ProjectRoleOwner)
		if errors.Is(err, project.ErrProjectRoleNotFound) {
			return ErrProjectRoleMissing
		}
		if err != nil {
			return err
		}

		priority, err := repo.FindProjectPriorityByCode(ctx, project.ProjectPriorityMedium)
		if errors.Is(err, project.ErrProjectPriorityNotFound) {
			return ErrProjectPriorityMissing
		}
		if err != nil {
			return err
		}

		code, err := repo.NextProjectCode(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, defaultProjectCodePrefix, year, defaultProjectCodeLength)
		if err != nil {
			return err
		}

		createdProject = project.Project{
			TenantID:               input.TenantContext.TenantID,
			WorkspaceID:            input.TenantContext.WorkspaceID,
			Code:                   code,
			Name:                   name,
			Type:                   projectType,
			Status:                 project.ProjectStatusDraft,
			PriorityID:             priority.ID,
			Priority:               priority.Code,
			Description:            strings.TrimSpace(input.Description),
			ClientOrRequestingUnit: strings.TrimSpace(input.ClientOrRequestingUnit),
			ScopeOrObjective:       strings.TrimSpace(input.ScopeOrObjective),
			CreatedBy:              input.Account.ID,
			CreatedAt:              now,
			UpdatedAt:              now,
		}
		if err := repo.CreateProject(ctx, &createdProject); err != nil {
			return err
		}

		userAccountID := input.Account.ID
		ownerMember = project.Member{
			TenantID:              input.TenantContext.TenantID,
			WorkspaceID:           input.TenantContext.WorkspaceID,
			ProjectID:             createdProject.ID,
			WorkspaceMembershipID: input.TenantContext.MembershipID,
			UserAccountID:         &userAccountID,
			RoleID:                ownerRole.ID,
			Role:                  ownerRole.Code,
			Status:                project.ProjectMemberStatusActive,
			JoinedAt:              &now,
			CreatedAt:             now,
			CreatedBy:             &input.Account.ID,
			UpdatedAt:             now,
			UpdatedBy:             &input.Account.ID,
		}
		if err := repo.CreateMember(ctx, &ownerMember); err != nil {
			return errors.Join(ErrProjectMemberCreateFail, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &CreateProjectResult{
		Project: createdProject,
		Member:  ownerMember,
	}, nil
}

func (s *Service) ListProjects(ctx context.Context, input ListProjectsInput) (*ListProjectsResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}
	limit := input.Limit
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}

	items, total, err := s.repository.ListProjects(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListProjectsResult{Items: items, Total: total}, nil
}

func (s *Service) GetProject(ctx context.Context, input GetProjectInput) (*GetProjectResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}
	if input.ProjectID == uuid.Nil {
		return nil, ErrProjectIDRequired
	}

	item, err := s.repository.FindProjectByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID)
	if errors.Is(err, project.ErrProjectNotFound) {
		return nil, ErrProjectNotFound
	}
	if err != nil {
		return nil, err
	}

	return &GetProjectResult{Item: *item}, nil
}

func (s *Service) UpdateProject(ctx context.Context, input UpdateProjectInput) (*UpdateProjectResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}
	if input.ProjectID == uuid.Nil {
		return nil, ErrProjectIDRequired
	}

	profilePatch := project.ProjectProfilePatch{
		UpdatedBy: input.Account.ID,
		UpdatedAt: s.clock(),
	}

	hasField := false
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrProjectNameRequired
		}
		profilePatch.Name = &name
		hasField = true
	}
	if input.Type != nil {
		projectType := project.ProjectType(strings.TrimSpace(string(*input.Type)))
		if projectType != project.ProjectTypeInternal && projectType != project.ProjectTypeClient {
			return nil, ErrProjectTypeInvalid
		}
		profilePatch.Type = &projectType
		hasField = true
	}
	if input.Priority != nil {
		priorityCode := project.ProjectPriority(strings.TrimSpace(string(*input.Priority)))
		if priorityCode == "" {
			return nil, ErrProjectPriorityInvalid
		}
		priority, err := s.repository.FindProjectPriorityByCode(ctx, priorityCode)
		if errors.Is(err, project.ErrProjectPriorityNotFound) {
			return nil, ErrProjectPriorityInvalid
		}
		if err != nil {
			return nil, err
		}
		profilePatch.PriorityID = &priority.ID
		hasField = true
	}
	if input.Description != nil {
		description := strings.TrimSpace(*input.Description)
		profilePatch.Description = &description
		hasField = true
	}
	if input.ClientOrRequestingUnit != nil {
		clientOrRequestingUnit := strings.TrimSpace(*input.ClientOrRequestingUnit)
		profilePatch.ClientOrRequestingUnit = &clientOrRequestingUnit
		hasField = true
	}
	if input.ScopeOrObjective != nil {
		scopeOrObjective := strings.TrimSpace(*input.ScopeOrObjective)
		profilePatch.ScopeOrObjective = &scopeOrObjective
		hasField = true
	}
	if !hasField {
		return nil, ErrProjectUpdateNoFields
	}

	if err := s.repository.UpdateProjectProfile(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, profilePatch); err != nil {
		if errors.Is(err, project.ErrProjectNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	item, err := s.repository.FindProjectByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID)
	if errors.Is(err, project.ErrProjectNotFound) {
		return nil, ErrProjectNotFound
	}
	if err != nil {
		return nil, err
	}
	return &UpdateProjectResult{Item: *item}, nil
}

func (s *Service) ListProjectMembers(ctx context.Context, input ListProjectMembersInput) (*ListProjectMembersResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}
	if input.ProjectID == uuid.Nil {
		return nil, ErrProjectIDRequired
	}
	limit := input.Limit
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}

	if _, err := s.repository.FindProjectByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID); err != nil {
		if errors.Is(err, project.ErrProjectNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}

	items, total, err := s.repository.ListProjectMembers(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListProjectMembersResult{Items: items, Total: total}, nil
}

func (s *Service) AddProjectMember(ctx context.Context, input AddProjectMemberInput) (*AddProjectMemberResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}
	if input.ProjectID == uuid.Nil {
		return nil, ErrProjectIDRequired
	}
	if input.WorkspaceMembershipID == uuid.Nil {
		return nil, ErrWorkspaceMembershipIDRequired
	}

	roleCode := input.Role
	if roleCode == "" {
		roleCode = project.ProjectRoleMember
	}

	now := s.clock()
	var createdMember project.Member
	err := s.repository.WithinTransaction(ctx, func(ctx context.Context, repo project.Repository) error {
		if _, err := repo.FindProjectByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID); err != nil {
			if errors.Is(err, project.ErrProjectNotFound) {
				return ErrProjectNotFound
			}
			return err
		}

		role, err := repo.FindProjectRoleByCode(ctx, roleCode)
		if errors.Is(err, project.ErrProjectRoleNotFound) {
			return ErrProjectRoleInvalid
		}
		if err != nil {
			return err
		}

		candidate, err := repo.FindActiveWorkspaceMembershipByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.WorkspaceMembershipID)
		if errors.Is(err, project.ErrWorkspaceMembershipNotFound) {
			return ErrWorkspaceMembershipNotFound
		}
		if err != nil {
			return err
		}

		createdMember = project.Member{
			TenantID:              input.TenantContext.TenantID,
			WorkspaceID:           input.TenantContext.WorkspaceID,
			ProjectID:             input.ProjectID,
			WorkspaceMembershipID: candidate.MembershipID,
			ProfileID:             candidate.ProfileID,
			UserAccountID:         candidate.UserAccountID,
			RoleID:                role.ID,
			Role:                  role.Code,
			Status:                project.ProjectMemberStatusActive,
			JoinedAt:              &now,
			CreatedBy:             &input.Account.ID,
			CreatedAt:             now,
			UpdatedBy:             &input.Account.ID,
			UpdatedAt:             now,
		}
		if err := repo.CreateMember(ctx, &createdMember); err != nil {
			if errors.Is(err, project.ErrProjectMemberAlreadyExists) {
				return ErrProjectMemberAlreadyExists
			}
			return errors.Join(ErrProjectMemberCreateFail, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &AddProjectMemberResult{Member: createdMember}, nil
}

func (s *Service) UpdateProjectMember(ctx context.Context, input UpdateProjectMemberInput) (*UpdateProjectMemberResult, error) {
	if err := validateAccount(input.Account); err != nil {
		return nil, err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return nil, err
	}
	if input.ProjectID == uuid.Nil {
		return nil, ErrProjectIDRequired
	}
	if input.MemberID == uuid.Nil {
		return nil, ErrProjectMemberIDRequired
	}
	if input.Role == nil {
		return nil, ErrProjectMemberUpdateNoFields
	}

	roleCode := project.ProjectRole(strings.TrimSpace(string(*input.Role)))
	if roleCode == "" {
		return nil, ErrProjectRoleInvalid
	}

	now := s.clock()
	var updatedMember project.Member
	err := s.repository.WithinTransaction(ctx, func(ctx context.Context, repo project.Repository) error {
		if _, err := repo.FindProjectByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID); err != nil {
			if errors.Is(err, project.ErrProjectNotFound) {
				return ErrProjectNotFound
			}
			return err
		}

		role, err := repo.FindProjectRoleByCode(ctx, roleCode)
		if errors.Is(err, project.ErrProjectRoleNotFound) {
			return ErrProjectRoleInvalid
		}
		if err != nil {
			return err
		}

		if err := repo.UpdateProjectMemberRole(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.MemberID, role.ID, input.Account.ID, now); err != nil {
			if errors.Is(err, project.ErrProjectMemberNotFound) {
				return ErrProjectMemberNotFound
			}
			return err
		}

		member, err := repo.FindProjectMemberByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.MemberID)
		if errors.Is(err, project.ErrProjectMemberNotFound) {
			return ErrProjectMemberNotFound
		}
		if err != nil {
			return err
		}
		updatedMember = *member
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &UpdateProjectMemberResult{Member: updatedMember}, nil
}

func (s *Service) RemoveProjectMember(ctx context.Context, input RemoveProjectMemberInput) error {
	if err := validateAccount(input.Account); err != nil {
		return err
	}
	if err := validateTenantContext(input.TenantContext); err != nil {
		return err
	}
	if input.ProjectID == uuid.Nil {
		return ErrProjectIDRequired
	}
	if input.MemberID == uuid.Nil {
		return ErrProjectMemberIDRequired
	}

	now := s.clock()
	return s.repository.WithinTransaction(ctx, func(ctx context.Context, repo project.Repository) error {
		if _, err := repo.FindProjectByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID); err != nil {
			if errors.Is(err, project.ErrProjectNotFound) {
				return ErrProjectNotFound
			}
			return err
		}

		member, err := repo.FindProjectMemberByID(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.MemberID)
		if errors.Is(err, project.ErrProjectMemberNotFound) {
			return ErrProjectMemberNotFound
		}
		if err != nil {
			return err
		}

		if member.Role == project.ProjectRoleOwner {
			ownerCount, err := repo.CountActiveProjectMembersByRole(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, project.ProjectRoleOwner)
			if err != nil {
				return err
			}
			if ownerCount <= 1 {
				return ErrProjectMemberLastOwner
			}
		}

		if err := repo.RemoveProjectMember(ctx, input.TenantContext.TenantID, input.TenantContext.WorkspaceID, input.ProjectID, input.MemberID, input.Account.ID, now); err != nil {
			if errors.Is(err, project.ErrProjectMemberNotFound) {
				return ErrProjectMemberNotFound
			}
			return err
		}
		return nil
	})
}

func validateAccount(account auth.UserAccount) error {
	if account.ID == uuid.Nil {
		return ErrAccountRequired
	}
	if account.Status != auth.UserAccountStatusActive {
		return ErrAccountInactive
	}
	return nil
}

func validateTenantContext(context workspace.TenantContext) error {
	if context.TenantID == uuid.Nil || context.WorkspaceID == uuid.Nil || context.MembershipID == uuid.Nil {
		return ErrTenantContextRequired
	}
	return nil
}
