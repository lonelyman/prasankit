package projectsvc

import (
	"context"
	"errors"
	"testing"
	"time"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/project"
	"prasankit-api/internal/modules/workspace"

	"github.com/google/uuid"
)

type fakeRepository struct {
	project                    *project.Project
	member                     *project.Member
	findItem                   *project.ProjectWithMember
	findErr                    error
	findProjectID              uuid.UUID
	findTenantID               uuid.UUID
	findWorkspaceID            uuid.UUID
	updateProjectID            uuid.UUID
	updateTenantID             uuid.UUID
	updateWorkspaceID          uuid.UUID
	updatePatch                project.ProjectProfilePatch
	updateErr                  error
	memberItems                []project.Member
	memberTotal                int
	memberErr                  error
	findMember                 *project.Member
	findMemberErr              error
	findMemberID               uuid.UUID
	updateMemberRoleID         uuid.UUID
	updateMemberErr            error
	updateMemberProjectID      uuid.UUID
	updateMemberTenantID       uuid.UUID
	updateMemberWorkspaceID    uuid.UUID
	updateMemberID             uuid.UUID
	updateMemberUpdatedBy      uuid.UUID
	ownerCount                 int
	ownerCountErr              error
	removeMemberErr            error
	removeMemberProjectID      uuid.UUID
	removeMemberTenantID       uuid.UUID
	removeMemberWorkspaceID    uuid.UUID
	removeMemberID             uuid.UUID
	removeMemberRemovedBy      uuid.UUID
	memberCandidate            *project.WorkspaceMemberCandidate
	memberCandidateErr         error
	memberCandidateID          uuid.UUID
	memberCandidateTenantID    uuid.UUID
	memberCandidateWorkspaceID uuid.UUID
	createMemberErr            error
	memberProjectID            uuid.UUID
	memberTenantID             uuid.UUID
	memberWorkspaceID          uuid.UUID
	memberLimit                int
	memberOffset               int
	listItems                  []project.ProjectWithMember
	listTotal                  int
	transactionCalled          bool
}

func (r *fakeRepository) WithinTransaction(ctx context.Context, fn func(context.Context, project.Repository) error) error {
	r.transactionCalled = true
	return fn(ctx, r)
}

func (r *fakeRepository) FindProjectRoleByCode(_ context.Context, code project.ProjectRole) (*project.RoleMaster, error) {
	switch code {
	case project.ProjectRoleOwner, project.ProjectRoleManager, project.ProjectRoleMember, project.ProjectRoleFinance, project.ProjectRoleViewer:
		return &project.RoleMaster{ID: uuid.Must(uuid.NewV7()), Code: code, Name: string(code)}, nil
	}
	return nil, project.ErrProjectRoleNotFound
}

func (r *fakeRepository) FindProjectPriorityByCode(_ context.Context, code project.ProjectPriority) (*project.PriorityMaster, error) {
	switch code {
	case project.ProjectPriorityLow, project.ProjectPriorityMedium, project.ProjectPriorityHigh:
		return &project.PriorityMaster{ID: uuid.Must(uuid.NewV7()), Code: code, Name: string(code)}, nil
	}
	return nil, project.ErrProjectPriorityNotFound
}

func (r *fakeRepository) NextProjectCode(context.Context, uuid.UUID, uuid.UUID, string, int, int) (string, error) {
	return "PRJ-2026-0001", nil
}

func (r *fakeRepository) CreateProject(_ context.Context, value *project.Project) error {
	value.ID = uuid.Must(uuid.NewV7())
	r.project = value
	return nil
}

func (r *fakeRepository) CreateMember(_ context.Context, value *project.Member) error {
	if r.createMemberErr != nil {
		return r.createMemberErr
	}
	value.ID = uuid.Must(uuid.NewV7())
	r.member = value
	return nil
}

func (r *fakeRepository) FindProjectByID(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID) (*project.ProjectWithMember, error) {
	r.findTenantID = tenantID
	r.findWorkspaceID = workspaceID
	r.findProjectID = projectID
	if r.findErr != nil {
		return nil, r.findErr
	}
	if r.findItem != nil {
		return r.findItem, nil
	}
	return nil, project.ErrProjectNotFound
}

func (r *fakeRepository) UpdateProjectProfile(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, patch project.ProjectProfilePatch) error {
	r.updateTenantID = tenantID
	r.updateWorkspaceID = workspaceID
	r.updateProjectID = projectID
	r.updatePatch = patch
	if r.updateErr != nil {
		return r.updateErr
	}
	return nil
}

func (r *fakeRepository) ListProjects(context.Context, uuid.UUID, uuid.UUID, int, int) ([]project.ProjectWithMember, int, error) {
	return r.listItems, r.listTotal, nil
}

func (r *fakeRepository) FindActiveWorkspaceMembershipByID(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, membershipID uuid.UUID) (*project.WorkspaceMemberCandidate, error) {
	r.memberCandidateTenantID = tenantID
	r.memberCandidateWorkspaceID = workspaceID
	r.memberCandidateID = membershipID
	if r.memberCandidateErr != nil {
		return nil, r.memberCandidateErr
	}
	if r.memberCandidate != nil {
		return r.memberCandidate, nil
	}
	return nil, project.ErrWorkspaceMembershipNotFound
}

func (r *fakeRepository) ListProjectMembers(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, limit int, offset int) ([]project.Member, int, error) {
	r.memberTenantID = tenantID
	r.memberWorkspaceID = workspaceID
	r.memberProjectID = projectID
	r.memberLimit = limit
	r.memberOffset = offset
	if r.memberErr != nil {
		return nil, 0, r.memberErr
	}
	return r.memberItems, r.memberTotal, nil
}

func (r *fakeRepository) FindProjectMemberByID(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, memberID uuid.UUID) (*project.Member, error) {
	r.memberTenantID = tenantID
	r.memberWorkspaceID = workspaceID
	r.memberProjectID = projectID
	r.findMemberID = memberID
	if r.findMemberErr != nil {
		return nil, r.findMemberErr
	}
	if r.findMember != nil {
		return r.findMember, nil
	}
	return nil, project.ErrProjectMemberNotFound
}

func (r *fakeRepository) UpdateProjectMemberRole(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, memberID uuid.UUID, roleID uuid.UUID, updatedBy uuid.UUID, _ time.Time) error {
	r.updateMemberTenantID = tenantID
	r.updateMemberWorkspaceID = workspaceID
	r.updateMemberProjectID = projectID
	r.updateMemberID = memberID
	r.updateMemberRoleID = roleID
	r.updateMemberUpdatedBy = updatedBy
	if r.updateMemberErr != nil {
		return r.updateMemberErr
	}
	return nil
}

func (r *fakeRepository) CountActiveProjectMembersByRole(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, role project.ProjectRole) (int, error) {
	r.memberTenantID = tenantID
	r.memberWorkspaceID = workspaceID
	r.memberProjectID = projectID
	if r.ownerCountErr != nil {
		return 0, r.ownerCountErr
	}
	if role == project.ProjectRoleOwner {
		return r.ownerCount, nil
	}
	return 0, nil
}

func (r *fakeRepository) RemoveProjectMember(_ context.Context, tenantID uuid.UUID, workspaceID uuid.UUID, projectID uuid.UUID, memberID uuid.UUID, removedBy uuid.UUID, _ time.Time) error {
	r.removeMemberTenantID = tenantID
	r.removeMemberWorkspaceID = workspaceID
	r.removeMemberProjectID = projectID
	r.removeMemberID = memberID
	r.removeMemberRemovedBy = removedBy
	if r.removeMemberErr != nil {
		return r.removeMemberErr
	}
	return nil
}

func TestCreateProject(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantContext := testTenantContext()
	repo := &fakeRepository{}
	service := NewService(repo)

	result, err := service.CreateProject(context.Background(), CreateProjectInput{
		Account: auth.UserAccount{
			ID:     accountID,
			Status: auth.UserAccountStatusActive,
		},
		TenantContext:          tenantContext,
		Name:                   "Project A",
		Type:                   project.ProjectTypeClient,
		Description:            "Description",
		ClientOrRequestingUnit: "Client A",
		ScopeOrObjective:       "Scope",
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if !repo.transactionCalled {
		t.Fatal("transaction was not used")
	}
	if result.Project.ID == uuid.Nil {
		t.Fatal("project ID was not set")
	}
	if result.Project.TenantID != tenantContext.TenantID {
		t.Fatalf("tenant ID = %s, want %s", result.Project.TenantID, tenantContext.TenantID)
	}
	if result.Project.WorkspaceID != tenantContext.WorkspaceID {
		t.Fatalf("workspace ID = %s, want %s", result.Project.WorkspaceID, tenantContext.WorkspaceID)
	}
	if result.Project.Code != "PRJ-2026-0001" {
		t.Fatalf("project code = %s, want PRJ-2026-0001", result.Project.Code)
	}
	if result.Project.Status != project.ProjectStatusDraft {
		t.Fatalf("status = %s, want draft", result.Project.Status)
	}
	if result.Project.Priority != project.ProjectPriorityMedium {
		t.Fatalf("priority = %s, want medium", result.Project.Priority)
	}
	if result.Member.Role != project.ProjectRoleOwner {
		t.Fatalf("member role = %s, want project_owner", result.Member.Role)
	}
	if result.Member.WorkspaceMembershipID != tenantContext.MembershipID {
		t.Fatalf("workspace membership ID = %s, want %s", result.Member.WorkspaceMembershipID, tenantContext.MembershipID)
	}
}

func TestCreateProjectRejectsInvalidInput(t *testing.T) {
	service := NewService(&fakeRepository{})
	account := auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}

	_, err := service.CreateProject(context.Background(), CreateProjectInput{
		Account:       account,
		TenantContext: testTenantContext(),
		Name:          "",
	})
	if !errors.Is(err, ErrProjectNameRequired) {
		t.Fatalf("err = %v, want ErrProjectNameRequired", err)
	}

	_, err = service.CreateProject(context.Background(), CreateProjectInput{
		Account:       account,
		TenantContext: testTenantContext(),
		Name:          "Project A",
		Type:          project.ProjectType("invalid"),
	})
	if !errors.Is(err, ErrProjectTypeInvalid) {
		t.Fatalf("err = %v, want ErrProjectTypeInvalid", err)
	}
}

func TestListProjects(t *testing.T) {
	tenantContext := testTenantContext()
	repo := &fakeRepository{
		listTotal: 1,
		listItems: []project.ProjectWithMember{
			{
				Project: project.Project{
					ID:          uuid.Must(uuid.NewV7()),
					TenantID:    tenantContext.TenantID,
					WorkspaceID: tenantContext.WorkspaceID,
					Code:        "PRJ-2026-0001",
					Name:        "Project A",
					Status:      project.ProjectStatusDraft,
				},
			},
		},
	}
	service := NewService(repo)

	result, err := service.ListProjects(context.Background(), ListProjectsInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		Limit:         10,
		Offset:        0,
	})
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("total = %d, want 1", result.Total)
	}
	if len(result.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(result.Items))
	}
}

func TestGetProject(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{
		findItem: &project.ProjectWithMember{
			Project: project.Project{
				ID:          projectID,
				TenantID:    tenantContext.TenantID,
				WorkspaceID: tenantContext.WorkspaceID,
				Code:        "PRJ-2026-0001",
				Name:        "Project A",
				Status:      project.ProjectStatusDraft,
			},
		},
	}
	service := NewService(repo)

	result, err := service.GetProject(context.Background(), GetProjectInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
	})
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if repo.findTenantID != tenantContext.TenantID {
		t.Fatalf("tenant ID = %s, want %s", repo.findTenantID, tenantContext.TenantID)
	}
	if repo.findWorkspaceID != tenantContext.WorkspaceID {
		t.Fatalf("workspace ID = %s, want %s", repo.findWorkspaceID, tenantContext.WorkspaceID)
	}
	if repo.findProjectID != projectID {
		t.Fatalf("project ID = %s, want %s", repo.findProjectID, projectID)
	}
	if result.Item.Project.ID != projectID {
		t.Fatalf("project ID = %s, want %s", result.Item.Project.ID, projectID)
	}
}

func TestGetProjectRejectsMissingProjectID(t *testing.T) {
	service := NewService(&fakeRepository{})

	_, err := service.GetProject(context.Background(), GetProjectInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: testTenantContext(),
	})
	if !errors.Is(err, ErrProjectIDRequired) {
		t.Fatalf("err = %v, want ErrProjectIDRequired", err)
	}
}

func TestGetProjectMapsNotFound(t *testing.T) {
	service := NewService(&fakeRepository{findErr: project.ErrProjectNotFound})

	_, err := service.GetProject(context.Background(), GetProjectInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: testTenantContext(),
		ProjectID:     uuid.Must(uuid.NewV7()),
	})
	if !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("err = %v, want ErrProjectNotFound", err)
	}
}

func TestUpdateProject(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	name := " Project B "
	projectType := project.ProjectTypeClient
	priority := project.ProjectPriorityHigh
	description := " "
	repo := &fakeRepository{
		findItem: &project.ProjectWithMember{
			Project: project.Project{
				ID:          projectID,
				TenantID:    tenantContext.TenantID,
				WorkspaceID: tenantContext.WorkspaceID,
				Code:        "PRJ-2026-0001",
				Name:        "Project B",
				Type:        project.ProjectTypeClient,
				Status:      project.ProjectStatusDraft,
				Priority:    project.ProjectPriorityHigh,
			},
		},
	}
	service := NewService(repo)

	result, err := service.UpdateProject(context.Background(), UpdateProjectInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		Name:          &name,
		Type:          &projectType,
		Priority:      &priority,
		Description:   &description,
	})
	if err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}
	if repo.updateTenantID != tenantContext.TenantID {
		t.Fatalf("tenant ID = %s, want %s", repo.updateTenantID, tenantContext.TenantID)
	}
	if repo.updateWorkspaceID != tenantContext.WorkspaceID {
		t.Fatalf("workspace ID = %s, want %s", repo.updateWorkspaceID, tenantContext.WorkspaceID)
	}
	if repo.updateProjectID != projectID {
		t.Fatalf("project ID = %s, want %s", repo.updateProjectID, projectID)
	}
	if repo.updatePatch.Name == nil || *repo.updatePatch.Name != "Project B" {
		t.Fatalf("patch name = %#v, want Project B", repo.updatePatch.Name)
	}
	if repo.updatePatch.Description == nil || *repo.updatePatch.Description != "" {
		t.Fatalf("patch description = %#v, want empty string", repo.updatePatch.Description)
	}
	if repo.updatePatch.PriorityID == nil || *repo.updatePatch.PriorityID == uuid.Nil {
		t.Fatal("patch priority ID was not set")
	}
	if result.Item.Project.ID != projectID {
		t.Fatalf("project ID = %s, want %s", result.Item.Project.ID, projectID)
	}
}

func TestUpdateProjectRejectsInvalidInput(t *testing.T) {
	service := NewService(&fakeRepository{})
	account := auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())

	_, err := service.UpdateProject(context.Background(), UpdateProjectInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
	})
	if !errors.Is(err, ErrProjectUpdateNoFields) {
		t.Fatalf("err = %v, want ErrProjectUpdateNoFields", err)
	}

	name := " "
	_, err = service.UpdateProject(context.Background(), UpdateProjectInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		Name:          &name,
	})
	if !errors.Is(err, ErrProjectNameRequired) {
		t.Fatalf("err = %v, want ErrProjectNameRequired", err)
	}

	priority := project.ProjectPriority("urgent")
	_, err = service.UpdateProject(context.Background(), UpdateProjectInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		Priority:      &priority,
	})
	if !errors.Is(err, ErrProjectPriorityInvalid) {
		t.Fatalf("err = %v, want ErrProjectPriorityInvalid", err)
	}
}

func TestUpdateProjectMapsNotFound(t *testing.T) {
	name := "Project B"
	service := NewService(&fakeRepository{updateErr: project.ErrProjectNotFound})

	_, err := service.UpdateProject(context.Background(), UpdateProjectInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: testTenantContext(),
		ProjectID:     uuid.Must(uuid.NewV7()),
		Name:          &name,
	})
	if !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("err = %v, want ErrProjectNotFound", err)
	}
}

func TestListProjectMembers(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	memberID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{
		findItem: &project.ProjectWithMember{
			Project: project.Project{
				ID:          projectID,
				TenantID:    tenantContext.TenantID,
				WorkspaceID: tenantContext.WorkspaceID,
			},
		},
		memberTotal: 1,
		memberItems: []project.Member{
			{
				ID:                    memberID,
				TenantID:              tenantContext.TenantID,
				WorkspaceID:           tenantContext.WorkspaceID,
				ProjectID:             projectID,
				WorkspaceMembershipID: tenantContext.MembershipID,
				Role:                  project.ProjectRoleOwner,
				Status:                project.ProjectMemberStatusActive,
			},
		},
	}
	service := NewService(repo)

	result, err := service.ListProjectMembers(context.Background(), ListProjectMembersInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		Limit:         10,
		Offset:        0,
	})
	if err != nil {
		t.Fatalf("ListProjectMembers: %v", err)
	}
	if repo.memberTenantID != tenantContext.TenantID {
		t.Fatalf("tenant ID = %s, want %s", repo.memberTenantID, tenantContext.TenantID)
	}
	if repo.memberWorkspaceID != tenantContext.WorkspaceID {
		t.Fatalf("workspace ID = %s, want %s", repo.memberWorkspaceID, tenantContext.WorkspaceID)
	}
	if repo.memberProjectID != projectID {
		t.Fatalf("project ID = %s, want %s", repo.memberProjectID, projectID)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("result = total %d len %d, want 1/1", result.Total, len(result.Items))
	}
	if result.Items[0].ID != memberID {
		t.Fatalf("member ID = %s, want %s", result.Items[0].ID, memberID)
	}
}

func TestListProjectMembersRequiresExistingProject(t *testing.T) {
	service := NewService(&fakeRepository{findErr: project.ErrProjectNotFound})

	_, err := service.ListProjectMembers(context.Background(), ListProjectMembersInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: testTenantContext(),
		ProjectID:     uuid.Must(uuid.NewV7()),
	})
	if !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("err = %v, want ErrProjectNotFound", err)
	}
}

func TestAddProjectMember(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	workspaceMembershipID := uuid.Must(uuid.NewV7())
	userAccountID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{
		findItem: &project.ProjectWithMember{
			Project: project.Project{
				ID:          projectID,
				TenantID:    tenantContext.TenantID,
				WorkspaceID: tenantContext.WorkspaceID,
			},
		},
		memberCandidate: &project.WorkspaceMemberCandidate{
			MembershipID:  workspaceMembershipID,
			TenantID:      tenantContext.TenantID,
			WorkspaceID:   tenantContext.WorkspaceID,
			UserAccountID: &userAccountID,
		},
	}
	service := NewService(repo)

	result, err := service.AddProjectMember(context.Background(), AddProjectMemberInput{
		Account:               auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext:         tenantContext,
		ProjectID:             projectID,
		WorkspaceMembershipID: workspaceMembershipID,
		Role:                  project.ProjectRoleManager,
	})
	if err != nil {
		t.Fatalf("AddProjectMember: %v", err)
	}
	if !repo.transactionCalled {
		t.Fatal("transaction was not used")
	}
	if repo.memberCandidateTenantID != tenantContext.TenantID {
		t.Fatalf("tenant ID = %s, want %s", repo.memberCandidateTenantID, tenantContext.TenantID)
	}
	if repo.memberCandidateWorkspaceID != tenantContext.WorkspaceID {
		t.Fatalf("workspace ID = %s, want %s", repo.memberCandidateWorkspaceID, tenantContext.WorkspaceID)
	}
	if repo.memberCandidateID != workspaceMembershipID {
		t.Fatalf("membership ID = %s, want %s", repo.memberCandidateID, workspaceMembershipID)
	}
	if result.Member.ID == uuid.Nil {
		t.Fatal("member ID was not set")
	}
	if result.Member.ProjectID != projectID {
		t.Fatalf("project ID = %s, want %s", result.Member.ProjectID, projectID)
	}
	if result.Member.WorkspaceMembershipID != workspaceMembershipID {
		t.Fatalf("workspace membership ID = %s, want %s", result.Member.WorkspaceMembershipID, workspaceMembershipID)
	}
	if result.Member.Role != project.ProjectRoleManager {
		t.Fatalf("role = %s, want project_manager", result.Member.Role)
	}
	if result.Member.UserAccountID == nil || *result.Member.UserAccountID != userAccountID {
		t.Fatalf("user account ID = %#v, want %s", result.Member.UserAccountID, userAccountID)
	}
}

func TestAddProjectMemberDefaultsRole(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	workspaceMembershipID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{
		findItem: &project.ProjectWithMember{
			Project: project.Project{ID: projectID, TenantID: tenantContext.TenantID, WorkspaceID: tenantContext.WorkspaceID},
		},
		memberCandidate: &project.WorkspaceMemberCandidate{
			MembershipID: workspaceMembershipID,
			TenantID:     tenantContext.TenantID,
			WorkspaceID:  tenantContext.WorkspaceID,
		},
	}
	service := NewService(repo)

	result, err := service.AddProjectMember(context.Background(), AddProjectMemberInput{
		Account:               auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext:         tenantContext,
		ProjectID:             projectID,
		WorkspaceMembershipID: workspaceMembershipID,
	})
	if err != nil {
		t.Fatalf("AddProjectMember: %v", err)
	}
	if result.Member.Role != project.ProjectRoleMember {
		t.Fatalf("role = %s, want member", result.Member.Role)
	}
}

func TestAddProjectMemberRejectsInvalidInput(t *testing.T) {
	service := NewService(&fakeRepository{})
	account := auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())

	_, err := service.AddProjectMember(context.Background(), AddProjectMemberInput{
		Account:               account,
		TenantContext:         tenantContext,
		WorkspaceMembershipID: uuid.Must(uuid.NewV7()),
	})
	if !errors.Is(err, ErrProjectIDRequired) {
		t.Fatalf("err = %v, want ErrProjectIDRequired", err)
	}

	_, err = service.AddProjectMember(context.Background(), AddProjectMemberInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     uuid.Must(uuid.NewV7()),
	})
	if !errors.Is(err, ErrWorkspaceMembershipIDRequired) {
		t.Fatalf("err = %v, want ErrWorkspaceMembershipIDRequired", err)
	}

	repo := &fakeRepository{
		findItem: &project.ProjectWithMember{
			Project: project.Project{ID: projectID, TenantID: tenantContext.TenantID, WorkspaceID: tenantContext.WorkspaceID},
		},
	}
	_, err = NewService(repo).AddProjectMember(context.Background(), AddProjectMemberInput{
		Account:               account,
		TenantContext:         tenantContext,
		ProjectID:             projectID,
		WorkspaceMembershipID: uuid.Must(uuid.NewV7()),
		Role:                  project.ProjectRole("invalid"),
	})
	if !errors.Is(err, ErrProjectRoleInvalid) {
		t.Fatalf("err = %v, want ErrProjectRoleInvalid", err)
	}
}

func TestAddProjectMemberMapsErrors(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	workspaceMembershipID := uuid.Must(uuid.NewV7())
	account := auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}

	_, err := NewService(&fakeRepository{findErr: project.ErrProjectNotFound}).AddProjectMember(context.Background(), AddProjectMemberInput{
		Account:               account,
		TenantContext:         tenantContext,
		ProjectID:             projectID,
		WorkspaceMembershipID: workspaceMembershipID,
	})
	if !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("err = %v, want ErrProjectNotFound", err)
	}

	repo := &fakeRepository{
		findItem: &project.ProjectWithMember{
			Project: project.Project{ID: projectID, TenantID: tenantContext.TenantID, WorkspaceID: tenantContext.WorkspaceID},
		},
		memberCandidateErr: project.ErrWorkspaceMembershipNotFound,
	}
	_, err = NewService(repo).AddProjectMember(context.Background(), AddProjectMemberInput{
		Account:               account,
		TenantContext:         tenantContext,
		ProjectID:             projectID,
		WorkspaceMembershipID: workspaceMembershipID,
	})
	if !errors.Is(err, ErrWorkspaceMembershipNotFound) {
		t.Fatalf("err = %v, want ErrWorkspaceMembershipNotFound", err)
	}

	repo = &fakeRepository{
		findItem: &project.ProjectWithMember{
			Project: project.Project{ID: projectID, TenantID: tenantContext.TenantID, WorkspaceID: tenantContext.WorkspaceID},
		},
		memberCandidate: &project.WorkspaceMemberCandidate{
			MembershipID: workspaceMembershipID,
			TenantID:     tenantContext.TenantID,
			WorkspaceID:  tenantContext.WorkspaceID,
		},
		createMemberErr: project.ErrProjectMemberAlreadyExists,
	}
	_, err = NewService(repo).AddProjectMember(context.Background(), AddProjectMemberInput{
		Account:               account,
		TenantContext:         tenantContext,
		ProjectID:             projectID,
		WorkspaceMembershipID: workspaceMembershipID,
	})
	if !errors.Is(err, ErrProjectMemberAlreadyExists) {
		t.Fatalf("err = %v, want ErrProjectMemberAlreadyExists", err)
	}
}

func TestUpdateProjectMember(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	memberID := uuid.Must(uuid.NewV7())
	accountID := uuid.Must(uuid.NewV7())
	role := project.ProjectRoleManager
	repo := &fakeRepository{
		findItem: &project.ProjectWithMember{
			Project: project.Project{ID: projectID, TenantID: tenantContext.TenantID, WorkspaceID: tenantContext.WorkspaceID},
		},
		findMember: &project.Member{
			ID:                    memberID,
			TenantID:              tenantContext.TenantID,
			WorkspaceID:           tenantContext.WorkspaceID,
			ProjectID:             projectID,
			WorkspaceMembershipID: uuid.Must(uuid.NewV7()),
			Role:                  project.ProjectRoleManager,
			Status:                project.ProjectMemberStatusActive,
		},
	}
	service := NewService(repo)

	result, err := service.UpdateProjectMember(context.Background(), UpdateProjectMemberInput{
		Account:       auth.UserAccount{ID: accountID, Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		MemberID:      memberID,
		Role:          &role,
	})
	if err != nil {
		t.Fatalf("UpdateProjectMember: %v", err)
	}
	if !repo.transactionCalled {
		t.Fatal("transaction was not used")
	}
	if repo.updateMemberTenantID != tenantContext.TenantID {
		t.Fatalf("tenant ID = %s, want %s", repo.updateMemberTenantID, tenantContext.TenantID)
	}
	if repo.updateMemberWorkspaceID != tenantContext.WorkspaceID {
		t.Fatalf("workspace ID = %s, want %s", repo.updateMemberWorkspaceID, tenantContext.WorkspaceID)
	}
	if repo.updateMemberProjectID != projectID {
		t.Fatalf("project ID = %s, want %s", repo.updateMemberProjectID, projectID)
	}
	if repo.updateMemberID != memberID {
		t.Fatalf("member ID = %s, want %s", repo.updateMemberID, memberID)
	}
	if repo.updateMemberRoleID == uuid.Nil {
		t.Fatal("role ID was not set")
	}
	if repo.updateMemberUpdatedBy != accountID {
		t.Fatalf("updated by = %s, want %s", repo.updateMemberUpdatedBy, accountID)
	}
	if result.Member.ID != memberID {
		t.Fatalf("member ID = %s, want %s", result.Member.ID, memberID)
	}
	if result.Member.Role != project.ProjectRoleManager {
		t.Fatalf("role = %s, want project_manager", result.Member.Role)
	}
}

func TestUpdateProjectMemberRejectsInvalidInput(t *testing.T) {
	service := NewService(&fakeRepository{})
	account := auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	memberID := uuid.Must(uuid.NewV7())

	_, err := service.UpdateProjectMember(context.Background(), UpdateProjectMemberInput{
		Account:       account,
		TenantContext: tenantContext,
		MemberID:      memberID,
		Role:          ptrProjectRole(project.ProjectRoleMember),
	})
	if !errors.Is(err, ErrProjectIDRequired) {
		t.Fatalf("err = %v, want ErrProjectIDRequired", err)
	}

	_, err = service.UpdateProjectMember(context.Background(), UpdateProjectMemberInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		Role:          ptrProjectRole(project.ProjectRoleMember),
	})
	if !errors.Is(err, ErrProjectMemberIDRequired) {
		t.Fatalf("err = %v, want ErrProjectMemberIDRequired", err)
	}

	_, err = service.UpdateProjectMember(context.Background(), UpdateProjectMemberInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		MemberID:      memberID,
	})
	if !errors.Is(err, ErrProjectMemberUpdateNoFields) {
		t.Fatalf("err = %v, want ErrProjectMemberUpdateNoFields", err)
	}

	invalidRole := project.ProjectRole("invalid")
	repo := &fakeRepository{
		findItem: &project.ProjectWithMember{
			Project: project.Project{ID: projectID, TenantID: tenantContext.TenantID, WorkspaceID: tenantContext.WorkspaceID},
		},
	}
	_, err = NewService(repo).UpdateProjectMember(context.Background(), UpdateProjectMemberInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		MemberID:      memberID,
		Role:          &invalidRole,
	})
	if !errors.Is(err, ErrProjectRoleInvalid) {
		t.Fatalf("err = %v, want ErrProjectRoleInvalid", err)
	}
}

func TestUpdateProjectMemberMapsErrors(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	memberID := uuid.Must(uuid.NewV7())
	account := auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}
	role := project.ProjectRoleMember

	_, err := NewService(&fakeRepository{findErr: project.ErrProjectNotFound}).UpdateProjectMember(context.Background(), UpdateProjectMemberInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		MemberID:      memberID,
		Role:          &role,
	})
	if !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("err = %v, want ErrProjectNotFound", err)
	}

	repo := &fakeRepository{
		findItem: &project.ProjectWithMember{
			Project: project.Project{ID: projectID, TenantID: tenantContext.TenantID, WorkspaceID: tenantContext.WorkspaceID},
		},
		updateMemberErr: project.ErrProjectMemberNotFound,
	}
	_, err = NewService(repo).UpdateProjectMember(context.Background(), UpdateProjectMemberInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		MemberID:      memberID,
		Role:          &role,
	})
	if !errors.Is(err, ErrProjectMemberNotFound) {
		t.Fatalf("err = %v, want ErrProjectMemberNotFound", err)
	}
}

func TestRemoveProjectMember(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	memberID := uuid.Must(uuid.NewV7())
	accountID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{
		findItem: &project.ProjectWithMember{
			Project: project.Project{ID: projectID, TenantID: tenantContext.TenantID, WorkspaceID: tenantContext.WorkspaceID},
		},
		findMember: &project.Member{
			ID:        memberID,
			TenantID:  tenantContext.TenantID,
			ProjectID: projectID,
			Role:      project.ProjectRoleMember,
			Status:    project.ProjectMemberStatusActive,
		},
	}
	service := NewService(repo)

	err := service.RemoveProjectMember(context.Background(), RemoveProjectMemberInput{
		Account:       auth.UserAccount{ID: accountID, Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		MemberID:      memberID,
	})
	if err != nil {
		t.Fatalf("RemoveProjectMember: %v", err)
	}
	if !repo.transactionCalled {
		t.Fatal("transaction was not used")
	}
	if repo.removeMemberTenantID != tenantContext.TenantID {
		t.Fatalf("tenant ID = %s, want %s", repo.removeMemberTenantID, tenantContext.TenantID)
	}
	if repo.removeMemberWorkspaceID != tenantContext.WorkspaceID {
		t.Fatalf("workspace ID = %s, want %s", repo.removeMemberWorkspaceID, tenantContext.WorkspaceID)
	}
	if repo.removeMemberProjectID != projectID {
		t.Fatalf("project ID = %s, want %s", repo.removeMemberProjectID, projectID)
	}
	if repo.removeMemberID != memberID {
		t.Fatalf("member ID = %s, want %s", repo.removeMemberID, memberID)
	}
	if repo.removeMemberRemovedBy != accountID {
		t.Fatalf("removed by = %s, want %s", repo.removeMemberRemovedBy, accountID)
	}
}

func TestRemoveProjectMemberRejectsInvalidInput(t *testing.T) {
	service := NewService(&fakeRepository{})
	account := auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	memberID := uuid.Must(uuid.NewV7())

	err := service.RemoveProjectMember(context.Background(), RemoveProjectMemberInput{
		Account:       account,
		TenantContext: tenantContext,
		MemberID:      memberID,
	})
	if !errors.Is(err, ErrProjectIDRequired) {
		t.Fatalf("err = %v, want ErrProjectIDRequired", err)
	}

	err = service.RemoveProjectMember(context.Background(), RemoveProjectMemberInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
	})
	if !errors.Is(err, ErrProjectMemberIDRequired) {
		t.Fatalf("err = %v, want ErrProjectMemberIDRequired", err)
	}
}

func TestRemoveProjectMemberRejectsLastOwner(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	memberID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{
		findItem: &project.ProjectWithMember{
			Project: project.Project{ID: projectID, TenantID: tenantContext.TenantID, WorkspaceID: tenantContext.WorkspaceID},
		},
		findMember: &project.Member{
			ID:        memberID,
			TenantID:  tenantContext.TenantID,
			ProjectID: projectID,
			Role:      project.ProjectRoleOwner,
			Status:    project.ProjectMemberStatusActive,
		},
		ownerCount: 1,
	}

	err := NewService(repo).RemoveProjectMember(context.Background(), RemoveProjectMemberInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		TenantContext: tenantContext,
		ProjectID:     projectID,
		MemberID:      memberID,
	})
	if !errors.Is(err, ErrProjectMemberLastOwner) {
		t.Fatalf("err = %v, want ErrProjectMemberLastOwner", err)
	}
	if repo.removeMemberID != uuid.Nil {
		t.Fatalf("member should not be removed, got %s", repo.removeMemberID)
	}
}

func TestRemoveProjectMemberMapsErrors(t *testing.T) {
	tenantContext := testTenantContext()
	projectID := uuid.Must(uuid.NewV7())
	memberID := uuid.Must(uuid.NewV7())
	account := auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}

	err := NewService(&fakeRepository{findErr: project.ErrProjectNotFound}).RemoveProjectMember(context.Background(), RemoveProjectMemberInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		MemberID:      memberID,
	})
	if !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("err = %v, want ErrProjectNotFound", err)
	}

	repo := &fakeRepository{
		findItem: &project.ProjectWithMember{
			Project: project.Project{ID: projectID, TenantID: tenantContext.TenantID, WorkspaceID: tenantContext.WorkspaceID},
		},
		findMemberErr: project.ErrProjectMemberNotFound,
	}
	err = NewService(repo).RemoveProjectMember(context.Background(), RemoveProjectMemberInput{
		Account:       account,
		TenantContext: tenantContext,
		ProjectID:     projectID,
		MemberID:      memberID,
	})
	if !errors.Is(err, ErrProjectMemberNotFound) {
		t.Fatalf("err = %v, want ErrProjectMemberNotFound", err)
	}
}

func ptrProjectRole(value project.ProjectRole) *project.ProjectRole {
	return &value
}

func testTenantContext() workspace.TenantContext {
	return workspace.TenantContext{
		TenantID:      uuid.Must(uuid.NewV7()),
		WorkspaceID:   uuid.Must(uuid.NewV7()),
		WorkspaceSlug: "team-one",
		MembershipID:  uuid.Must(uuid.NewV7()),
		Role:          workspace.WorkspaceRoleOwner,
	}
}
