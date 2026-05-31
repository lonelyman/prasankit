package projectmemberposition_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/projectmember"
	"prasankit-api/internal/modules/projectmemberposition"
	"prasankit-api/internal/modules/projectposition"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

type fakeJunctionRepo struct {
	mu sync.Mutex

	assignCalls   int
	unassignCalls int

	lastAssignEntry   audit.Entry
	lastUnassignEntry audit.Entry

	assignErr   error
	unassignErr error
	listRows    []projectmemberposition.PositionWithLabel
}

func (r *fakeJunctionRepo) AssignWithAudit(ctx context.Context, mp projectmemberposition.ProjectMemberPosition, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.assignCalls++
	r.lastAssignEntry = entry
	return r.assignErr
}

func (r *fakeJunctionRepo) UnassignWithAudit(ctx context.Context, workspaceID, projectMemberID uuid.UUID, positionCode string, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.unassignCalls++
	r.lastUnassignEntry = entry
	return r.unassignErr
}

func (r *fakeJunctionRepo) ListByMember(ctx context.Context, workspaceID, projectMemberID uuid.UUID) ([]projectmemberposition.PositionWithLabel, error) {
	return r.listRows, nil
}

// fakeMemberRepo stubs ALL 8 projectmember.ProjectMemberRepository methods (the 6b-1 gap):
// the 8th, FindByID, is the read-only addition used by the junction member guard.
type fakeMemberRepo struct {
	findByID    *projectmember.ProjectMember
	findByIDErr error
}

func (r *fakeMemberRepo) AddWithAudit(ctx context.Context, m projectmember.ProjectMember, entry audit.Entry) error {
	return nil
}
func (r *fakeMemberRepo) RemoveWithAudit(ctx context.Context, workspaceID, projectID, memberID, removedBy uuid.UUID, entry audit.Entry) error {
	return nil
}
func (r *fakeMemberRepo) ChangeRoleWithAudit(ctx context.Context, workspaceID, projectID, memberID uuid.UUID, newRoleCode string, updatedBy uuid.UUID, entry audit.Entry) error {
	return nil
}
func (r *fakeMemberRepo) ListByProject(ctx context.Context, workspaceID, projectID uuid.UUID) ([]projectmember.MemberWithDisplayName, error) {
	return nil, nil
}
func (r *fakeMemberRepo) FindActiveByID(ctx context.Context, workspaceID, projectID, memberID uuid.UUID) (*projectmember.ProjectMember, error) {
	return nil, nil
}
func (r *fakeMemberRepo) FindByID(ctx context.Context, workspaceID, projectID, memberID uuid.UUID) (*projectmember.ProjectMember, error) {
	if r.findByIDErr != nil {
		return nil, r.findByIDErr
	}
	if r.findByID != nil {
		cp := *r.findByID
		return &cp, nil
	}
	return nil, nil
}
func (r *fakeMemberRepo) IsActiveWorkspaceMember(ctx context.Context, workspaceID, membershipID uuid.UUID) (bool, error) {
	return true, nil
}
func (r *fakeMemberRepo) ProjectExistsForWorkspace(ctx context.Context, workspaceID, projectID uuid.UUID) (bool, error) {
	return true, nil
}

type fakePositionMasterRepo struct {
	active bool
}

func (r *fakePositionMasterRepo) IsActiveProjectPositionCode(ctx context.Context, workspaceID uuid.UUID, code string) (bool, error) {
	return r.active, nil
}

// Three explicit compile-time interface assertions.
var _ projectmemberposition.ProjectMemberPositionRepository = (*fakeJunctionRepo)(nil)
var _ projectmember.ProjectMemberRepository = (*fakeMemberRepo)(nil)
var _ projectposition.PositionMasterRepository = (*fakePositionMasterRepo)(nil)

// ── helpers ─────────────────────────────────────────────────────────────────────

func newTC() workspace.TenantContext {
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

func activeMember(tc workspace.TenantContext, projectID, memberID uuid.UUID) *projectmember.ProjectMember {
	return &projectmember.ProjectMember{
		ID:          memberID,
		WorkspaceID: tc.WorkspaceID,
		ProjectID:   projectID,
		RemovedAt:   nil,
	}
}

func removedMember(tc workspace.TenantContext, projectID, memberID uuid.UUID) *projectmember.ProjectMember {
	now := time.Now().UTC()
	m := activeMember(tc, projectID, memberID)
	m.RemovedAt = &now
	return m
}

func ids2(t *testing.T) (uuid.UUID, uuid.UUID) {
	t.Helper()
	pid, _ := ids.New()
	mid, _ := ids.New()
	return pid, mid
}

// ── Assign ──────────────────────────────────────────────────────────────────────

func TestService_AssignPosition_Success(t *testing.T) {
	tc := newTC()
	pid, mid := ids2(t)
	junction := &fakeJunctionRepo{}
	members := &fakeMemberRepo{findByID: activeMember(tc, pid, mid)}
	masters := &fakePositionMasterRepo{active: true}
	svc := projectmemberposition.NewService(junction, members, masters)

	mp, err := svc.Assign(context.Background(), projectmemberposition.AssignInput{
		TenantCtx: tc, ProjectID: pid, ProjectMemberID: mid, ProjectPositionCode: "tech_lead",
	})
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if mp == nil || mp.ProjectPositionCode != "tech_lead" {
		t.Fatalf("bad result: %+v", mp)
	}
	junction.mu.Lock()
	defer junction.mu.Unlock()
	if junction.assignCalls != 1 {
		t.Errorf("assignCalls = %d, want 1", junction.assignCalls)
	}
}

func TestService_AssignPosition_MemberNotFound(t *testing.T) {
	tc := newTC()
	pid, mid := ids2(t)
	junction := &fakeJunctionRepo{}
	members := &fakeMemberRepo{findByID: nil} // absent
	masters := &fakePositionMasterRepo{active: true}
	svc := projectmemberposition.NewService(junction, members, masters)

	_, err := svc.Assign(context.Background(), projectmemberposition.AssignInput{
		TenantCtx: tc, ProjectID: pid, ProjectMemberID: mid, ProjectPositionCode: "tech_lead",
	})
	if !errors.Is(err, projectmemberposition.ErrProjectMemberNotFound) {
		t.Errorf("err = %v, want ErrProjectMemberNotFound", err)
	}
	junction.mu.Lock()
	defer junction.mu.Unlock()
	if junction.assignCalls != 0 {
		t.Errorf("assignCalls = %d, want 0 (member guard)", junction.assignCalls)
	}
}

func TestService_AssignPosition_MemberRemoved_Rejected(t *testing.T) {
	tc := newTC()
	pid, mid := ids2(t)
	junction := &fakeJunctionRepo{}
	members := &fakeMemberRepo{findByID: removedMember(tc, pid, mid)} // removed_at IS NOT NULL
	masters := &fakePositionMasterRepo{active: true}
	svc := projectmemberposition.NewService(junction, members, masters)

	_, err := svc.Assign(context.Background(), projectmemberposition.AssignInput{
		TenantCtx: tc, ProjectID: pid, ProjectMemberID: mid, ProjectPositionCode: "tech_lead",
	})
	if !errors.Is(err, projectmemberposition.ErrMemberRemoved) {
		t.Errorf("err = %v, want ErrMemberRemoved (422, §M2.5(c))", err)
	}
	junction.mu.Lock()
	defer junction.mu.Unlock()
	if junction.assignCalls != 0 {
		t.Errorf("assignCalls = %d, want 0 (removed member)", junction.assignCalls)
	}
}

func TestService_AssignPosition_InvalidPositionCode(t *testing.T) {
	tc := newTC()
	pid, mid := ids2(t)
	junction := &fakeJunctionRepo{}
	members := &fakeMemberRepo{findByID: activeMember(tc, pid, mid)}
	masters := &fakePositionMasterRepo{active: false} // deprecated/nonexistent
	svc := projectmemberposition.NewService(junction, members, masters)

	_, err := svc.Assign(context.Background(), projectmemberposition.AssignInput{
		TenantCtx: tc, ProjectID: pid, ProjectMemberID: mid, ProjectPositionCode: "deprecated_pos",
	})
	if !errors.Is(err, projectmemberposition.ErrInvalidPositionCode) {
		t.Errorf("err = %v, want ErrInvalidPositionCode", err)
	}
}

func TestService_AssignPosition_AlreadyAssigned(t *testing.T) {
	tc := newTC()
	pid, mid := ids2(t)
	junction := &fakeJunctionRepo{assignErr: projectmemberposition.ErrAlreadyAssigned}
	members := &fakeMemberRepo{findByID: activeMember(tc, pid, mid)}
	masters := &fakePositionMasterRepo{active: true}
	svc := projectmemberposition.NewService(junction, members, masters)

	_, err := svc.Assign(context.Background(), projectmemberposition.AssignInput{
		TenantCtx: tc, ProjectID: pid, ProjectMemberID: mid, ProjectPositionCode: "tech_lead",
	})
	if !errors.Is(err, projectmemberposition.ErrAlreadyAssigned) {
		t.Errorf("err = %v, want ErrAlreadyAssigned", err)
	}
}

func TestService_AssignPosition_AuditEntryIsProjectScoped(t *testing.T) {
	tc := newTC()
	pid, mid := ids2(t)
	junction := &fakeJunctionRepo{}
	members := &fakeMemberRepo{findByID: activeMember(tc, pid, mid)}
	masters := &fakePositionMasterRepo{active: true}
	svc := projectmemberposition.NewService(junction, members, masters)

	if _, err := svc.Assign(context.Background(), projectmemberposition.AssignInput{
		TenantCtx: tc, ProjectID: pid, ProjectMemberID: mid, ProjectPositionCode: "tech_lead",
	}); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	junction.mu.Lock()
	defer junction.mu.Unlock()
	e := junction.lastAssignEntry
	if e.WorkspaceID == nil || *e.WorkspaceID != tc.WorkspaceID {
		t.Errorf("audit.WorkspaceID = %v, want %v", e.WorkspaceID, tc.WorkspaceID)
	}
	if e.ProjectID == nil || *e.ProjectID != pid {
		t.Errorf("audit.ProjectID = %v, want %v (project-scoped)", e.ProjectID, pid)
	}
}

// ── Unassign ──────────────────────────────────────────────────────────────────

func TestService_UnassignPosition_Success(t *testing.T) {
	tc := newTC()
	pid, mid := ids2(t)
	junction := &fakeJunctionRepo{}
	members := &fakeMemberRepo{findByID: activeMember(tc, pid, mid)}
	masters := &fakePositionMasterRepo{active: true}
	svc := projectmemberposition.NewService(junction, members, masters)

	if err := svc.Unassign(context.Background(), projectmemberposition.UnassignInput{
		TenantCtx: tc, ProjectID: pid, ProjectMemberID: mid, ProjectPositionCode: "tech_lead",
	}); err != nil {
		t.Fatalf("Unassign: %v", err)
	}
	junction.mu.Lock()
	defer junction.mu.Unlock()
	if junction.unassignCalls != 1 {
		t.Errorf("unassignCalls = %d, want 1", junction.unassignCalls)
	}
}

func TestService_UnassignPosition_NotFound(t *testing.T) {
	tc := newTC()
	pid, mid := ids2(t)
	junction := &fakeJunctionRepo{unassignErr: projectmemberposition.ErrAssignmentNotFound}
	members := &fakeMemberRepo{findByID: activeMember(tc, pid, mid)}
	masters := &fakePositionMasterRepo{active: true}
	svc := projectmemberposition.NewService(junction, members, masters)

	err := svc.Unassign(context.Background(), projectmemberposition.UnassignInput{
		TenantCtx: tc, ProjectID: pid, ProjectMemberID: mid, ProjectPositionCode: "tech_lead",
	})
	if !errors.Is(err, projectmemberposition.ErrAssignmentNotFound) {
		t.Errorf("err = %v, want ErrAssignmentNotFound", err)
	}
}

func TestService_UnassignPosition_MemberNotFound_404(t *testing.T) {
	tc := newTC()
	pid, mid := ids2(t)
	junction := &fakeJunctionRepo{}
	members := &fakeMemberRepo{findByID: nil} // member absent -> 404 BEFORE the DELETE
	masters := &fakePositionMasterRepo{active: true}
	svc := projectmemberposition.NewService(junction, members, masters)

	err := svc.Unassign(context.Background(), projectmemberposition.UnassignInput{
		TenantCtx: tc, ProjectID: pid, ProjectMemberID: mid, ProjectPositionCode: "tech_lead",
	})
	if !errors.Is(err, projectmemberposition.ErrProjectMemberNotFound) {
		t.Errorf("err = %v, want ErrProjectMemberNotFound", err)
	}
	junction.mu.Lock()
	defer junction.mu.Unlock()
	if junction.unassignCalls != 0 {
		t.Errorf("unassignCalls = %d, want 0 (member gate before DELETE)", junction.unassignCalls)
	}
}

func TestService_UnassignPosition_MemberRemoved_422(t *testing.T) {
	tc := newTC()
	pid, mid := ids2(t)
	junction := &fakeJunctionRepo{}
	members := &fakeMemberRepo{findByID: removedMember(tc, pid, mid)} // removed -> 422
	masters := &fakePositionMasterRepo{active: true}
	svc := projectmemberposition.NewService(junction, members, masters)

	err := svc.Unassign(context.Background(), projectmemberposition.UnassignInput{
		TenantCtx: tc, ProjectID: pid, ProjectMemberID: mid, ProjectPositionCode: "tech_lead",
	})
	if !errors.Is(err, projectmemberposition.ErrMemberRemoved) {
		t.Errorf("err = %v, want ErrMemberRemoved (symmetric with Assign)", err)
	}
	junction.mu.Lock()
	defer junction.mu.Unlock()
	if junction.unassignCalls != 0 {
		t.Errorf("unassignCalls = %d, want 0 (removed member gate)", junction.unassignCalls)
	}
}
