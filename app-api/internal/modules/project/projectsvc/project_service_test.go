package projectsvc

import (
	"context"
	"errors"
	"testing"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/project"
	"prasankit-api/internal/modules/workspace"

	"github.com/google/uuid"
)

type fakeRepository struct {
	project           *project.Project
	member            *project.Member
	listItems         []project.ProjectWithMember
	listTotal         int
	transactionCalled bool
}

func (r *fakeRepository) WithinTransaction(ctx context.Context, fn func(context.Context, project.Repository) error) error {
	r.transactionCalled = true
	return fn(ctx, r)
}

func (r *fakeRepository) FindProjectRoleByCode(_ context.Context, code project.ProjectRole) (*project.RoleMaster, error) {
	if code == project.ProjectRoleOwner {
		return &project.RoleMaster{ID: uuid.Must(uuid.NewV7()), Code: project.ProjectRoleOwner, Name: "Project Owner"}, nil
	}
	return nil, project.ErrProjectRoleNotFound
}

func (r *fakeRepository) FindProjectPriorityByCode(_ context.Context, code project.ProjectPriority) (*project.PriorityMaster, error) {
	if code == project.ProjectPriorityMedium {
		return &project.PriorityMaster{ID: uuid.Must(uuid.NewV7()), Code: project.ProjectPriorityMedium, Name: "Medium"}, nil
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
	value.ID = uuid.Must(uuid.NewV7())
	r.member = value
	return nil
}

func (r *fakeRepository) ListProjects(context.Context, uuid.UUID, uuid.UUID, int, int) ([]project.ProjectWithMember, int, error) {
	return r.listItems, r.listTotal, nil
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

func testTenantContext() workspace.TenantContext {
	return workspace.TenantContext{
		TenantID:      uuid.Must(uuid.NewV7()),
		WorkspaceID:   uuid.Must(uuid.NewV7()),
		WorkspaceSlug: "team-one",
		MembershipID:  uuid.Must(uuid.NewV7()),
		Role:          workspace.WorkspaceRoleOwner,
	}
}
