package projectmember_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/project"
	"prasankit-api/internal/modules/projectmember"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

type fakeProjectMemberRepo struct {
	mu sync.Mutex

	addCalls    int
	removeCalls int
	roleCalls   int
	listCalls   int

	lastAddEntry    audit.Entry
	lastRemoveEntry audit.Entry
	lastRoleEntry   audit.Entry

	addErr    error
	removeErr error
	roleErr   error
	listErr   error

	projectExists     bool
	projectExistsErr  error
	activeWsMember    bool
	activeWsMemberErr error
	findActive        *projectmember.ProjectMember
	findActiveErr     error
	listRows          []projectmember.MemberWithDisplayName
}

func (r *fakeProjectMemberRepo) AddWithAudit(ctx context.Context, m projectmember.ProjectMember, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.addCalls++
	r.lastAddEntry = entry
	return r.addErr
}

func (r *fakeProjectMemberRepo) RemoveWithAudit(ctx context.Context, workspaceID, projectID, memberID, removedBy uuid.UUID, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.removeCalls++
	r.lastRemoveEntry = entry
	return r.removeErr
}

func (r *fakeProjectMemberRepo) ChangeRoleWithAudit(ctx context.Context, workspaceID, projectID, memberID uuid.UUID, newRoleCode string, updatedBy uuid.UUID, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.roleCalls++
	r.lastRoleEntry = entry
	return r.roleErr
}

func (r *fakeProjectMemberRepo) ListByProject(ctx context.Context, workspaceID, projectID uuid.UUID) ([]projectmember.MemberWithDisplayName, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listCalls++
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.listRows, nil
}

func (r *fakeProjectMemberRepo) FindActiveByID(ctx context.Context, workspaceID, projectID, memberID uuid.UUID) (*projectmember.ProjectMember, error) {
	if r.findActiveErr != nil {
		return nil, r.findActiveErr
	}
	if r.findActive != nil {
		cp := *r.findActive
		return &cp, nil
	}
	return nil, nil
}

func (r *fakeProjectMemberRepo) FindByID(ctx context.Context, workspaceID, projectID, memberID uuid.UUID) (*projectmember.ProjectMember, error) {
	if r.findActiveErr != nil {
		return nil, r.findActiveErr
	}
	if r.findActive != nil {
		cp := *r.findActive
		return &cp, nil
	}
	return nil, nil
}

func (r *fakeProjectMemberRepo) IsActiveWorkspaceMember(ctx context.Context, workspaceID, membershipID uuid.UUID) (bool, error) {
	return r.activeWsMember, r.activeWsMemberErr
}

func (r *fakeProjectMemberRepo) ProjectExistsForWorkspace(ctx context.Context, workspaceID, projectID uuid.UUID) (bool, error) {
	return r.projectExists, r.projectExistsErr
}

type fakeMasterRepo struct {
	roleActive bool
	roleErr    error
}

func (r *fakeMasterRepo) IsActiveProjectStatusCode(ctx context.Context, code string) (bool, error) {
	return true, nil
}
func (r *fakeMasterRepo) IsActiveProjectTypeCode(ctx context.Context, code string) (bool, error) {
	return true, nil
}
func (r *fakeMasterRepo) IsActiveProjectRoleCode(ctx context.Context, code string) (bool, error) {
	return r.roleActive, r.roleErr
}

type fakeProjectRepo struct {
	owner   *uuid.UUID // OwnerProjectMemberID returned by FindByIDForWorkspace
	findNil bool
	findErr error
}

func (r *fakeProjectRepo) CreateWithAudit(ctx context.Context, p project.Project, entry audit.Entry) error {
	return nil
}
func (r *fakeProjectRepo) CreateWithOwner(ctx context.Context, p project.Project, owner project.OwnerMemberSeed, projectEntry, memberEntry audit.Entry) error {
	return nil
}
func (r *fakeProjectRepo) FindByIDForWorkspace(ctx context.Context, workspaceID, projectID uuid.UUID) (*project.Project, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	if r.findNil {
		return nil, nil
	}
	return &project.Project{
		ID:                   projectID,
		WorkspaceID:          workspaceID,
		OwnerProjectMemberID: r.owner,
	}, nil
}
func (r *fakeProjectRepo) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, opts project.ListOptions) ([]project.Project, int64, error) {
	return nil, 0, nil
}
func (r *fakeProjectRepo) UpdateWithAudit(ctx context.Context, p project.Project, entry audit.Entry) error {
	return nil
}
func (r *fakeProjectRepo) SoftDeleteWithAudit(ctx context.Context, workspaceID, projectID, deletedBy uuid.UUID, entry audit.Entry) error {
	return nil
}
func (r *fakeProjectRepo) ChangeStatusWithAudit(ctx context.Context, workspaceID, projectID uuid.UUID, newStatusCode string, updatedBy uuid.UUID, entry audit.Entry) error {
	return nil
}

// Compile-time interface assertions (6a-style).
var _ projectmember.ProjectMemberRepository = (*fakeProjectMemberRepo)(nil)
var _ project.MasterRepository = (*fakeMasterRepo)(nil)
var _ project.ProjectRepository = (*fakeProjectRepo)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

func newTC(t *testing.T) workspace.TenantContext {
	t.Helper()
	wsID, _ := ids.New()
	accID, _ := ids.New()
	mID, _ := ids.New()
	return workspace.TenantContext{
		WorkspaceID:  wsID,
		MembershipID: mID,
		AccountID:    accID,
		OrgRoleCode:  workspace.OrgRoleOwner,
	}
}

func newProjectID(t *testing.T) uuid.UUID {
	t.Helper()
	id, _ := ids.New()
	return id
}

func buildAddInput(t *testing.T, tc workspace.TenantContext, projectID uuid.UUID, role string) projectmember.AddMemberInput {
	t.Helper()
	memID, _ := ids.New()
	return projectmember.AddMemberInput{
		TenantCtx:             tc,
		ProjectID:             projectID,
		WorkspaceMembershipID: memID,
		ProjectRoleCode:       role,
	}
}

// ── AddMember ─────────────────────────────────────────────────────────────────

func TestService_AddMember_Success(t *testing.T) {
	tc := newTC(t)
	pid := newProjectID(t)
	members := &fakeProjectMemberRepo{projectExists: true, activeWsMember: true}
	masters := &fakeMasterRepo{roleActive: true}
	projects := &fakeProjectRepo{}
	svc := projectmember.NewService(members, masters, projects)

	m, err := svc.AddMember(context.Background(), buildAddInput(t, tc, pid, "member"))
	if err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	if m == nil {
		t.Fatal("nil member")
	}
	members.mu.Lock()
	defer members.mu.Unlock()
	if members.addCalls != 1 {
		t.Errorf("addCalls = %d, want 1", members.addCalls)
	}
	e := members.lastAddEntry
	if e.WorkspaceID == nil || *e.WorkspaceID != tc.WorkspaceID {
		t.Errorf("audit.WorkspaceID = %v, want %v", e.WorkspaceID, tc.WorkspaceID)
	}
	if e.ProjectID == nil || *e.ProjectID != pid {
		t.Errorf("audit.ProjectID = %v, want %v", e.ProjectID, pid)
	}
	if e.Action != projectmember.AuditActionProjectMemberAdd {
		t.Errorf("audit.Action = %q, want project_member.add", e.Action)
	}
}

func TestService_AddMember_InvalidRoleCode_422(t *testing.T) {
	tc := newTC(t)
	pid := newProjectID(t)
	members := &fakeProjectMemberRepo{projectExists: true, activeWsMember: true}
	masters := &fakeMasterRepo{roleActive: false}
	svc := projectmember.NewService(members, masters, &fakeProjectRepo{})

	_, err := svc.AddMember(context.Background(), buildAddInput(t, tc, pid, "bogus"))
	if !errors.Is(err, projectmember.ErrInvalidRoleCode) {
		t.Errorf("err = %v, want ErrInvalidRoleCode", err)
	}
}

func TestService_AddMember_OwnerRoleForbidden_422(t *testing.T) {
	tc := newTC(t)
	pid := newProjectID(t)
	members := &fakeProjectMemberRepo{projectExists: true, activeWsMember: true}
	masters := &fakeMasterRepo{roleActive: true}
	svc := projectmember.NewService(members, masters, &fakeProjectRepo{})

	_, err := svc.AddMember(context.Background(), buildAddInput(t, tc, pid, projectmember.ProjectRoleOwnerCode))
	if !errors.Is(err, projectmember.ErrInvalidRoleCode) {
		t.Errorf("err = %v, want ErrInvalidRoleCode (owner role forbidden -> 422)", err)
	}
}

func TestService_AddMember_MembershipNotEligible_422(t *testing.T) {
	tc := newTC(t)
	pid := newProjectID(t)
	members := &fakeProjectMemberRepo{projectExists: true, activeWsMember: false}
	masters := &fakeMasterRepo{roleActive: true}
	svc := projectmember.NewService(members, masters, &fakeProjectRepo{})

	_, err := svc.AddMember(context.Background(), buildAddInput(t, tc, pid, "member"))
	if !errors.Is(err, projectmember.ErrMembershipNotEligible) {
		t.Errorf("err = %v, want ErrMembershipNotEligible", err)
	}
}

func TestService_AddMember_ProjectNotFound_404(t *testing.T) {
	tc := newTC(t)
	pid := newProjectID(t)
	members := &fakeProjectMemberRepo{projectExists: false, activeWsMember: true}
	masters := &fakeMasterRepo{roleActive: true}
	svc := projectmember.NewService(members, masters, &fakeProjectRepo{})

	_, err := svc.AddMember(context.Background(), buildAddInput(t, tc, pid, "member"))
	if !errors.Is(err, projectmember.ErrProjectNotFound) {
		t.Errorf("err = %v, want ErrProjectNotFound", err)
	}
}

func TestService_AddMember_AlreadyMember_Conflict(t *testing.T) {
	tc := newTC(t)
	pid := newProjectID(t)
	members := &fakeProjectMemberRepo{projectExists: true, activeWsMember: true, addErr: projectmember.ErrAlreadyMember}
	masters := &fakeMasterRepo{roleActive: true}
	svc := projectmember.NewService(members, masters, &fakeProjectRepo{})

	_, err := svc.AddMember(context.Background(), buildAddInput(t, tc, pid, "member"))
	if !errors.Is(err, projectmember.ErrAlreadyMember) {
		t.Errorf("err = %v, want ErrAlreadyMember", err)
	}
}

// ── ChangeRole ────────────────────────────────────────────────────────────────

func existingMember(id, wsID, pid uuid.UUID, role string) *projectmember.ProjectMember {
	return &projectmember.ProjectMember{
		ID:              id,
		WorkspaceID:     wsID,
		ProjectID:       pid,
		ProjectRoleCode: role,
	}
}

func TestService_ChangeRole_Success(t *testing.T) {
	tc := newTC(t)
	pid := newProjectID(t)
	memID, _ := ids.New()
	ownerID, _ := ids.New() // different from memID → not owner
	members := &fakeProjectMemberRepo{findActive: existingMember(memID, tc.WorkspaceID, pid, "member")}
	masters := &fakeMasterRepo{roleActive: true}
	projects := &fakeProjectRepo{owner: &ownerID}
	svc := projectmember.NewService(members, masters, projects)

	m, err := svc.ChangeRole(context.Background(), projectmember.ChangeRoleInput{
		TenantCtx: tc, ProjectID: pid, MemberID: memID, NewRoleCode: "finance",
	})
	if err != nil {
		t.Fatalf("ChangeRole: %v", err)
	}
	if m.ProjectRoleCode != "finance" {
		t.Errorf("role = %q, want finance", m.ProjectRoleCode)
	}
	members.mu.Lock()
	defer members.mu.Unlock()
	if members.roleCalls != 1 {
		t.Errorf("roleCalls = %d, want 1", members.roleCalls)
	}
}

func TestService_ChangeRole_NoOpSameRole_NoAudit(t *testing.T) {
	tc := newTC(t)
	pid := newProjectID(t)
	memID, _ := ids.New()
	ownerID, _ := ids.New()
	members := &fakeProjectMemberRepo{findActive: existingMember(memID, tc.WorkspaceID, pid, "member")}
	masters := &fakeMasterRepo{roleActive: true}
	projects := &fakeProjectRepo{owner: &ownerID}
	svc := projectmember.NewService(members, masters, projects)

	_, err := svc.ChangeRole(context.Background(), projectmember.ChangeRoleInput{
		TenantCtx: tc, ProjectID: pid, MemberID: memID, NewRoleCode: "member", // same
	})
	if err != nil {
		t.Fatalf("ChangeRole: %v", err)
	}
	members.mu.Lock()
	defer members.mu.Unlock()
	if members.roleCalls != 0 {
		t.Errorf("roleCalls = %d, want 0 (no-op)", members.roleCalls)
	}
}

func TestService_ChangeRole_OwnerRoleDrift_Rejected_422(t *testing.T) {
	tc := newTC(t)
	pid := newProjectID(t)
	memID, _ := ids.New()
	members := &fakeProjectMemberRepo{findActive: existingMember(memID, tc.WorkspaceID, pid, "member")}
	masters := &fakeMasterRepo{roleActive: true}
	projects := &fakeProjectRepo{owner: &memID} // memID IS the owner
	svc := projectmember.NewService(members, masters, projects)

	_, err := svc.ChangeRole(context.Background(), projectmember.ChangeRoleInput{
		TenantCtx: tc, ProjectID: pid, MemberID: memID, NewRoleCode: "finance",
	})
	if !errors.Is(err, projectmember.ErrOwnerRoleImmutable) {
		t.Errorf("err = %v, want ErrOwnerRoleImmutable", err)
	}
}

func TestService_ChangeRole_PromoteToOwnerForbidden_422(t *testing.T) {
	tc := newTC(t)
	pid := newProjectID(t)
	memID, _ := ids.New()
	members := &fakeProjectMemberRepo{findActive: existingMember(memID, tc.WorkspaceID, pid, "member")}
	masters := &fakeMasterRepo{roleActive: true}
	svc := projectmember.NewService(members, masters, &fakeProjectRepo{})

	_, err := svc.ChangeRole(context.Background(), projectmember.ChangeRoleInput{
		TenantCtx: tc, ProjectID: pid, MemberID: memID, NewRoleCode: projectmember.ProjectRoleOwnerCode,
	})
	if !errors.Is(err, projectmember.ErrInvalidRoleCode) {
		t.Errorf("err = %v, want ErrInvalidRoleCode", err)
	}
}

// ── RemoveMember ──────────────────────────────────────────────────────────────

func TestService_RemoveMember_Success(t *testing.T) {
	tc := newTC(t)
	pid := newProjectID(t)
	memID, _ := ids.New()
	ownerID, _ := ids.New()
	members := &fakeProjectMemberRepo{findActive: existingMember(memID, tc.WorkspaceID, pid, "member")}
	masters := &fakeMasterRepo{roleActive: true}
	projects := &fakeProjectRepo{owner: &ownerID}
	svc := projectmember.NewService(members, masters, projects)

	if err := svc.RemoveMember(context.Background(), projectmember.RemoveMemberInput{
		TenantCtx: tc, ProjectID: pid, MemberID: memID,
	}); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
	members.mu.Lock()
	defer members.mu.Unlock()
	if members.removeCalls != 1 {
		t.Errorf("removeCalls = %d, want 1", members.removeCalls)
	}
}

func TestService_RemoveMember_OwnerRemovalRejected_422(t *testing.T) {
	tc := newTC(t)
	pid := newProjectID(t)
	memID, _ := ids.New()
	members := &fakeProjectMemberRepo{findActive: existingMember(memID, tc.WorkspaceID, pid, "project_owner")}
	masters := &fakeMasterRepo{roleActive: true}
	projects := &fakeProjectRepo{owner: &memID} // memID IS the owner
	svc := projectmember.NewService(members, masters, projects)

	err := svc.RemoveMember(context.Background(), projectmember.RemoveMemberInput{
		TenantCtx: tc, ProjectID: pid, MemberID: memID,
	})
	if !errors.Is(err, projectmember.ErrOwnerRoleImmutable) {
		t.Errorf("err = %v, want ErrOwnerRoleImmutable", err)
	}
	members.mu.Lock()
	defer members.mu.Unlock()
	if members.removeCalls != 0 {
		t.Errorf("removeCalls = %d, want 0 (owner guard)", members.removeCalls)
	}
}

func TestService_RemoveMember_NotFound_404(t *testing.T) {
	tc := newTC(t)
	pid := newProjectID(t)
	memID, _ := ids.New()
	members := &fakeProjectMemberRepo{findActive: nil} // not found
	masters := &fakeMasterRepo{roleActive: true}
	svc := projectmember.NewService(members, masters, &fakeProjectRepo{})

	err := svc.RemoveMember(context.Background(), projectmember.RemoveMemberInput{
		TenantCtx: tc, ProjectID: pid, MemberID: memID,
	})
	if !errors.Is(err, projectmember.ErrProjectMemberNotFound) {
		t.Errorf("err = %v, want ErrProjectMemberNotFound", err)
	}
}

// ── ListMembers ───────────────────────────────────────────────────────────────

func TestService_ListMembers_ProjectNotFound_404(t *testing.T) {
	tc := newTC(t)
	pid := newProjectID(t)
	members := &fakeProjectMemberRepo{projectExists: false}
	masters := &fakeMasterRepo{roleActive: true}
	svc := projectmember.NewService(members, masters, &fakeProjectRepo{})

	_, err := svc.ListMembers(context.Background(), projectmember.ListMembersInput{
		TenantCtx: tc, ProjectID: pid,
	})
	if !errors.Is(err, projectmember.ErrProjectNotFound) {
		t.Errorf("err = %v, want ErrProjectNotFound", err)
	}
}
