package workspacesvc

import (
	"context"
	"errors"
	"testing"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/workspace"

	"github.com/google/uuid"
)

type fakeRepository struct {
	existingWorkspace *workspace.Workspace
	workspace         *workspace.Workspace
	membership        *workspace.Membership
	listItems         []workspace.WorkspaceWithMembership
	listTotal         int
	resolveItem       *workspace.WorkspaceWithMembership
	resolveErr        error
	resolveSlug       string
	resolveAccountID  uuid.UUID
	role              *workspace.WorkspaceRoleMaster
	transactionCalled bool
}

func (r *fakeRepository) WithinTransaction(ctx context.Context, fn func(context.Context, workspace.Repository) error) error {
	r.transactionCalled = true
	return fn(ctx, r)
}

func (r *fakeRepository) FindWorkspaceBySlug(context.Context, string) (*workspace.Workspace, error) {
	if r.existingWorkspace != nil {
		return r.existingWorkspace, nil
	}
	return nil, workspace.ErrWorkspaceNotFound
}

func (r *fakeRepository) FindWorkspaceRoleByCode(_ context.Context, code workspace.WorkspaceRole) (*workspace.WorkspaceRoleMaster, error) {
	if r.role != nil {
		return r.role, nil
	}
	if code == workspace.WorkspaceRoleOwner {
		return &workspace.WorkspaceRoleMaster{
			ID:   uuid.Must(uuid.NewV7()),
			Code: workspace.WorkspaceRoleOwner,
			Name: "Owner",
		}, nil
	}
	return nil, workspace.ErrWorkspaceRoleNotFound
}

func (r *fakeRepository) CreateWorkspace(_ context.Context, value *workspace.Workspace) error {
	value.ID = uuid.Must(uuid.NewV7())
	value.TenantID = uuid.Must(uuid.NewV7())
	r.workspace = value
	return nil
}

func (r *fakeRepository) CreateMembership(_ context.Context, value *workspace.Membership) error {
	value.ID = uuid.Must(uuid.NewV7())
	r.membership = value
	return nil
}

func (r *fakeRepository) FindActiveMembership(context.Context, uuid.UUID, uuid.UUID) (*workspace.Membership, error) {
	return nil, workspace.ErrMembershipNotFound
}

func (r *fakeRepository) FindActiveWorkspaceMembershipBySlug(_ context.Context, slug string, userAccountID uuid.UUID) (*workspace.WorkspaceWithMembership, error) {
	r.resolveSlug = slug
	r.resolveAccountID = userAccountID
	if r.resolveErr != nil {
		return nil, r.resolveErr
	}
	if r.resolveItem != nil {
		return r.resolveItem, nil
	}
	return nil, workspace.ErrMembershipNotFound
}

func (r *fakeRepository) ListWorkspacesByUserAccountID(context.Context, uuid.UUID, int, int) ([]workspace.WorkspaceWithMembership, int, error) {
	return r.listItems, r.listTotal, nil
}

func TestCheckSlug(t *testing.T) {
	service := NewService(&fakeRepository{})

	result, err := service.CheckSlug(context.Background(), CheckSlugInput{Slug: " Team-One "})
	if err != nil {
		t.Fatalf("CheckSlug: %v", err)
	}
	if result.Slug != "team-one" {
		t.Fatalf("slug = %s, want team-one", result.Slug)
	}
	if !result.Available {
		t.Fatal("available = false, want true")
	}
}

func TestCheckSlugReportsReservedAndTaken(t *testing.T) {
	service := NewService(&fakeRepository{})

	result, err := service.CheckSlug(context.Background(), CheckSlugInput{Slug: "admin"})
	if err != nil {
		t.Fatalf("CheckSlug reserved: %v", err)
	}
	if result.Available || result.Reason != "reserved" {
		t.Fatalf("reserved result = %#v, want unavailable reserved", result)
	}

	service = NewService(&fakeRepository{existingWorkspace: &workspace.Workspace{ID: uuid.Must(uuid.NewV7())}})
	result, err = service.CheckSlug(context.Background(), CheckSlugInput{Slug: "team-one"})
	if err != nil {
		t.Fatalf("CheckSlug taken: %v", err)
	}
	if result.Available || result.Reason != "taken" {
		t.Fatalf("taken result = %#v, want unavailable taken", result)
	}
}

func TestRegisterWorkspace(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{}
	service := NewService(repo)

	result, err := service.RegisterWorkspace(context.Background(), RegisterWorkspaceInput{
		Account: auth.UserAccount{
			ID:           accountID,
			PrimaryEmail: "owner@example.test",
			Status:       auth.UserAccountStatusActive,
		},
		Name:         "Team One",
		Slug:         "team-one",
		ContactEmail: "Owner@Example.Test",
	})
	if err != nil {
		t.Fatalf("RegisterWorkspace: %v", err)
	}
	if !repo.transactionCalled {
		t.Fatal("transaction was not used")
	}
	if result.Workspace.ID == uuid.Nil {
		t.Fatal("workspace ID was not set")
	}
	if result.Workspace.TenantID == uuid.Nil {
		t.Fatal("tenant ID was not set")
	}
	if result.Workspace.Slug != "team-one" {
		t.Fatalf("slug = %s, want team-one", result.Workspace.Slug)
	}
	if result.Workspace.ContactEmail != "owner@example.test" {
		t.Fatalf("contact email = %s, want owner@example.test", result.Workspace.ContactEmail)
	}
	if result.Membership.Role != workspace.WorkspaceRoleOwner {
		t.Fatalf("role = %s, want owner", result.Membership.Role)
	}
	if result.Membership.RoleID == uuid.Nil {
		t.Fatal("membership role ID was not set")
	}
	if result.Membership.UserAccountID == nil || *result.Membership.UserAccountID != accountID {
		t.Fatalf("membership user = %v, want %s", result.Membership.UserAccountID, accountID)
	}
}

func TestRegisterWorkspaceRejectsInvalidInput(t *testing.T) {
	account := auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive}
	service := NewService(&fakeRepository{})

	_, err := service.RegisterWorkspace(context.Background(), RegisterWorkspaceInput{
		Account:      account,
		Name:         "",
		Slug:         "team-one",
		ContactEmail: "owner@example.test",
	})
	if !errors.Is(err, ErrWorkspaceNameRequired) {
		t.Fatalf("err = %v, want ErrWorkspaceNameRequired", err)
	}

	_, err = service.RegisterWorkspace(context.Background(), RegisterWorkspaceInput{
		Account:      account,
		Name:         "Team One",
		Slug:         "admin",
		ContactEmail: "owner@example.test",
	})
	if !errors.Is(err, ErrWorkspaceSlugReserved) {
		t.Fatalf("err = %v, want ErrWorkspaceSlugReserved", err)
	}

	_, err = service.RegisterWorkspace(context.Background(), RegisterWorkspaceInput{
		Account:      account,
		Name:         "Team One",
		Slug:         "team-one",
		ContactEmail: "not-an-email",
	})
	if !errors.Is(err, ErrContactEmailInvalid) {
		t.Fatalf("err = %v, want ErrContactEmailInvalid", err)
	}
}

func TestRegisterWorkspaceRejectsTakenSlug(t *testing.T) {
	service := NewService(&fakeRepository{
		existingWorkspace: &workspace.Workspace{ID: uuid.Must(uuid.NewV7())},
	})

	_, err := service.RegisterWorkspace(context.Background(), RegisterWorkspaceInput{
		Account:      auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		Name:         "Team One",
		Slug:         "team-one",
		ContactEmail: "owner@example.test",
	})
	if !errors.Is(err, ErrWorkspaceSlugTaken) {
		t.Fatalf("err = %v, want ErrWorkspaceSlugTaken", err)
	}
}

func TestListMyWorkspaces(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	workspaceID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{
		listItems: []workspace.WorkspaceWithMembership{
			{
				Workspace: workspace.Workspace{
					ID:     workspaceID,
					Name:   "Team One",
					Slug:   "team-one",
					Status: workspace.WorkspaceStatusActive,
				},
				Membership: workspace.Membership{
					ID:          uuid.Must(uuid.NewV7()),
					WorkspaceID: workspaceID,
					Role:        workspace.WorkspaceRoleOwner,
					Status:      workspace.MembershipStatusActive,
				},
			},
		},
		listTotal: 1,
	}
	service := NewService(repo)

	result, err := service.ListMyWorkspaces(context.Background(), ListMyWorkspacesInput{
		Account: auth.UserAccount{
			ID:     accountID,
			Status: auth.UserAccountStatusActive,
		},
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		t.Fatalf("ListMyWorkspaces: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("total = %d, want 1", result.Total)
	}
	if len(result.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(result.Items))
	}
	if result.Items[0].Workspace.Slug != "team-one" {
		t.Fatalf("workspace slug = %s, want team-one", result.Items[0].Workspace.Slug)
	}
}

func TestResolveTenantContext(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	tenantID := uuid.Must(uuid.NewV7())
	workspaceID := uuid.Must(uuid.NewV7())
	membershipID := uuid.Must(uuid.NewV7())
	repo := &fakeRepository{
		resolveItem: &workspace.WorkspaceWithMembership{
			Workspace: workspace.Workspace{
				ID:       workspaceID,
				TenantID: tenantID,
				Name:     "Team One",
				Slug:     "team-one",
				Status:   workspace.WorkspaceStatusActive,
			},
			Membership: workspace.Membership{
				ID:          membershipID,
				TenantID:    tenantID,
				WorkspaceID: workspaceID,
				Role:        workspace.WorkspaceRoleOwner,
				Status:      workspace.MembershipStatusActive,
			},
		},
	}
	service := NewService(repo)

	result, err := service.ResolveTenantContext(context.Background(), ResolveTenantContextInput{
		Account: auth.UserAccount{
			ID:     accountID,
			Status: auth.UserAccountStatusActive,
		},
		WorkspaceSlug: " Team-One ",
	})
	if err != nil {
		t.Fatalf("ResolveTenantContext: %v", err)
	}
	if repo.resolveSlug != "team-one" {
		t.Fatalf("resolved slug = %s, want team-one", repo.resolveSlug)
	}
	if repo.resolveAccountID != accountID {
		t.Fatalf("resolved account id = %s, want %s", repo.resolveAccountID, accountID)
	}
	if result.Context.TenantID != tenantID {
		t.Fatalf("tenant id = %s, want %s", result.Context.TenantID, tenantID)
	}
	if result.Context.WorkspaceID != workspaceID {
		t.Fatalf("workspace id = %s, want %s", result.Context.WorkspaceID, workspaceID)
	}
	if result.Context.MembershipID != membershipID {
		t.Fatalf("membership id = %s, want %s", result.Context.MembershipID, membershipID)
	}
	if result.Context.Role != workspace.WorkspaceRoleOwner {
		t.Fatalf("role = %s, want owner", result.Context.Role)
	}
}

func TestResolveTenantContextRejectsMissingMembership(t *testing.T) {
	service := NewService(&fakeRepository{resolveErr: workspace.ErrMembershipNotFound})

	_, err := service.ResolveTenantContext(context.Background(), ResolveTenantContextInput{
		Account:       auth.UserAccount{ID: uuid.Must(uuid.NewV7()), Status: auth.UserAccountStatusActive},
		WorkspaceSlug: "team-one",
	})
	if !errors.Is(err, ErrWorkspaceAccessDenied) {
		t.Fatalf("err = %v, want ErrWorkspaceAccessDenied", err)
	}
}
