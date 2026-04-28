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
