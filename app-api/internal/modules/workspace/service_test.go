package workspace_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
)

// ── Fake implementations of ports ─────────────────────────────────────────────

type fakeWorkspaceRepo struct {
	mu         sync.Mutex
	workspaces map[uuid.UUID]*workspace.Workspace
	slugIndex  map[string]uuid.UUID // slug → id
	forceErr   error                // if set, CreateWorkspaceWithOwner returns this
}

func newFakeWorkspaceRepo() *fakeWorkspaceRepo {
	return &fakeWorkspaceRepo{
		workspaces: map[uuid.UUID]*workspace.Workspace{},
		slugIndex:  map[string]uuid.UUID{},
	}
}

// lastEntry is the last audit.Entry passed to CreateWorkspaceWithOwner.
var lastEntry audit.Entry

// lastMembership is the last Membership passed to CreateWorkspaceWithOwner.
var lastMembership workspace.Membership

func (r *fakeWorkspaceRepo) CreateWorkspaceWithOwner(ctx context.Context, ws workspace.Workspace, m workspace.Membership, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.forceErr != nil {
		return r.forceErr
	}
	if _, exists := r.slugIndex[ws.Slug]; exists {
		return workspace.ErrSlugTaken
	}
	cp := ws
	r.workspaces[ws.ID] = &cp
	r.slugIndex[ws.Slug] = ws.ID
	lastEntry = entry
	lastMembership = m
	return nil
}

func (r *fakeWorkspaceRepo) FindBySlugActive(ctx context.Context, slug string) (*workspace.Workspace, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.slugIndex[slug]
	if !ok {
		return nil, nil
	}
	cp := *r.workspaces[id]
	return &cp, nil
}

type fakeMembershipRepo struct {
	mu          sync.Mutex
	memberships []workspace.Membership
}

func newFakeMembershipRepo() *fakeMembershipRepo {
	return &fakeMembershipRepo{}
}

func (r *fakeMembershipRepo) FindActiveByWorkspaceAndAccount(ctx context.Context, workspaceID, accountID uuid.UUID) (*workspace.Membership, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, m := range r.memberships {
		if m.WorkspaceID == workspaceID && m.UserAccountID == accountID && m.MembershipStatusCode == workspace.MembershipStatusActive {
			cp := m
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *fakeMembershipRepo) ListActiveWorkspacesByAccount(ctx context.Context, accountID uuid.UUID) ([]workspace.WorkspaceWithRole, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []workspace.WorkspaceWithRole
	for _, m := range r.memberships {
		if m.UserAccountID == accountID && m.MembershipStatusCode == workspace.MembershipStatusActive {
			result = append(result, workspace.WorkspaceWithRole{
				Workspace:   workspace.Workspace{ID: m.WorkspaceID},
				OrgRoleCode: m.OrgRoleCode,
			})
		}
	}
	return result, nil
}

func (r *fakeMembershipRepo) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]workspace.Membership, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []workspace.Membership
	for _, m := range r.memberships {
		if m.WorkspaceID == workspaceID {
			result = append(result, m)
		}
	}
	return result, nil
}

// ── Helpers ────────────────────────────────────────────────────────────────────

func buildSvc() (*workspace.Service, *fakeWorkspaceRepo, *fakeMembershipRepo) {
	wsRepo := newFakeWorkspaceRepo()
	memberRepo := newFakeMembershipRepo()
	svc := workspace.NewService(wsRepo, memberRepo)
	return svc, wsRepo, memberRepo
}

func newAccountID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := ids.New()
	if err != nil {
		t.Fatalf("ids.New: %v", err)
	}
	return id
}

// ── Tests ──────────────────────────────────────────────────────────────────────

func TestCreateWorkspace_Success(t *testing.T) {
	svc, wsRepo, _ := buildSvc()
	accountID := newAccountID(t)

	ws, err := svc.CreateWorkspace(context.Background(), workspace.CreateWorkspaceInput{
		AccountID:    accountID,
		Name:         "Acme Corp",
		Slug:         "acme-corp",
		ContactEmail: "admin@acme.com",
		IP:           "127.0.0.1",
		UserAgent:    "test-agent",
		RequestID:    "req-001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.WorkspaceName != "Acme Corp" {
		t.Errorf("name = %q, want Acme Corp", ws.WorkspaceName)
	}
	if ws.Slug != "acme-corp" {
		t.Errorf("slug = %q, want acme-corp", ws.Slug)
	}
	if ws.WorkspaceStatusCode != workspace.WorkspaceStatusActive {
		t.Errorf("status = %q, want active", ws.WorkspaceStatusCode)
	}
	if ws.OwnerUserAccountID != accountID {
		t.Errorf("owner = %v, want %v", ws.OwnerUserAccountID, accountID)
	}

	// Stored in repo.
	wsRepo.mu.Lock()
	_, exists := wsRepo.workspaces[ws.ID]
	wsRepo.mu.Unlock()
	if !exists {
		t.Error("workspace not found in repo after create")
	}

	// Owner membership built with org_role_code='owner'.
	if lastMembership.OrgRoleCode != workspace.OrgRoleOwner {
		t.Errorf("membership.org_role_code = %q, want owner", lastMembership.OrgRoleCode)
	}
	if lastMembership.UserAccountID != accountID {
		t.Errorf("membership.user_account_id = %v, want %v", lastMembership.UserAccountID, accountID)
	}
	if lastMembership.MembershipStatusCode != workspace.MembershipStatusActive {
		t.Errorf("membership.status = %q, want active", lastMembership.MembershipStatusCode)
	}

	// Audit entry built with action='workspace.create' and workspace_id set.
	if lastEntry.Action != workspace.AuditActionWorkspaceCreate {
		t.Errorf("audit.action = %q, want workspace.create", lastEntry.Action)
	}
	if lastEntry.WorkspaceID == nil || *lastEntry.WorkspaceID != ws.ID {
		t.Errorf("audit.workspace_id = %v, want %v", lastEntry.WorkspaceID, ws.ID)
	}
	if lastEntry.ActorAccountID == nil || *lastEntry.ActorAccountID != accountID {
		t.Errorf("audit.actor = %v, want %v", lastEntry.ActorAccountID, accountID)
	}
	if lastEntry.Result != workspace.AuditResultSuccess {
		t.Errorf("audit.result = %q, want success", lastEntry.Result)
	}
}

func TestCreateWorkspace_SlugReserved(t *testing.T) {
	svc, _, _ := buildSvc()
	accountID := newAccountID(t)

	for _, reserved := range []string{"admin", "api", "www", "auth", "health", "dashboard", "w", "workspace"} {
		_, err := svc.CreateWorkspace(context.Background(), workspace.CreateWorkspaceInput{
			AccountID:    accountID,
			Name:         "Test",
			Slug:         reserved,
			ContactEmail: "test@example.com",
		})
		if !errors.Is(err, workspace.ErrSlugReserved) {
			t.Errorf("slug %q: err = %v, want ErrSlugReserved", reserved, err)
		}
	}
}

func TestCreateWorkspace_SlugInvalid(t *testing.T) {
	svc, _, _ := buildSvc()
	accountID := newAccountID(t)

	// Note: the service normalises the slug with strings.ToLower before validation,
	// so uppercase input is accepted and normalised — not invalid. Single character
	// slugs are also valid per the regex. The cases below are genuinely invalid.
	cases := []struct{ slug string }{
		{"-leading"},
		{"trailing-"},
		{"has space"},
		{"has_underscore"},
		{""},
	}
	for _, tc := range cases {
		_, err := svc.CreateWorkspace(context.Background(), workspace.CreateWorkspaceInput{
			AccountID:    accountID,
			Name:         "Test",
			Slug:         tc.slug,
			ContactEmail: "test@example.com",
		})
		var valErr *workspace.ValidationError
		if !errors.As(err, &valErr) {
			t.Errorf("slug %q: err = %v, want ValidationError", tc.slug, err)
			continue
		}
		found := false
		for _, fe := range valErr.Fields {
			if fe.Field == "slug" {
				found = true
			}
		}
		if !found {
			t.Errorf("slug %q: no 'slug' field error in: %v", tc.slug, valErr.Fields)
		}
	}
}

func TestCreateWorkspace_NameValidation(t *testing.T) {
	svc, _, _ := buildSvc()
	accountID := newAccountID(t)

	// Empty name.
	_, err := svc.CreateWorkspace(context.Background(), workspace.CreateWorkspaceInput{
		AccountID:    accountID,
		Name:         "   ",
		Slug:         "valid-slug",
		ContactEmail: "test@example.com",
	})
	var valErr *workspace.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("empty name: err = %v, want ValidationError", err)
	}
	hasNameField := false
	for _, fe := range valErr.Fields {
		if fe.Field == "name" {
			hasNameField = true
		}
	}
	if !hasNameField {
		t.Errorf("expected 'name' field error, got: %v", valErr.Fields)
	}

	// Name too long (101 chars).
	longName := string(make([]byte, 101))
	for i := range []byte(longName) {
		longName = longName[:i] + "x" + longName[i+1:]
	}
	_, err = svc.CreateWorkspace(context.Background(), workspace.CreateWorkspaceInput{
		AccountID:    accountID,
		Name:         longName,
		Slug:         "valid-slug",
		ContactEmail: "test@example.com",
	})
	if !errors.As(err, &valErr) {
		t.Fatalf("long name: err = %v, want ValidationError", err)
	}
	hasNameField = false
	for _, fe := range valErr.Fields {
		if fe.Field == "name" {
			hasNameField = true
		}
	}
	if !hasNameField {
		t.Errorf("expected 'name' field error for long name, got: %v", valErr.Fields)
	}
}

func TestCreateWorkspace_EmailValidation(t *testing.T) {
	svc, _, _ := buildSvc()
	accountID := newAccountID(t)

	_, err := svc.CreateWorkspace(context.Background(), workspace.CreateWorkspaceInput{
		AccountID:    accountID,
		Name:         "Test",
		Slug:         "valid-slug",
		ContactEmail: "not-an-email",
	})
	var valErr *workspace.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
	found := false
	for _, fe := range valErr.Fields {
		if fe.Field == "contact_email" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'contact_email' field error: %v", valErr.Fields)
	}
}

func TestListMyWorkspaces(t *testing.T) {
	svc, _, memberRepo := buildSvc()
	accountID := newAccountID(t)
	wsID1, _ := ids.New()
	wsID2, _ := ids.New()

	memberRepo.mu.Lock()
	memberRepo.memberships = append(memberRepo.memberships,
		workspace.Membership{
			ID:                   uuid.Must(ids.New()),
			WorkspaceID:          wsID1,
			UserAccountID:        accountID,
			OrgRoleCode:          workspace.OrgRoleOwner,
			MembershipStatusCode: workspace.MembershipStatusActive,
		},
		workspace.Membership{
			ID:                   uuid.Must(ids.New()),
			WorkspaceID:          wsID2,
			UserAccountID:        accountID,
			OrgRoleCode:          workspace.OrgRoleUser,
			MembershipStatusCode: workspace.MembershipStatusActive,
		},
	)
	memberRepo.mu.Unlock()

	items, err := svc.ListMyWorkspaces(context.Background(), accountID)
	if err != nil {
		t.Fatalf("ListMyWorkspaces: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("got %d items, want 2", len(items))
	}
}
