package project_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/project"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

type fakeProjectRepo struct {
	mu               sync.Mutex
	projects         map[uuid.UUID]project.Project
	lastEntry        audit.Entry
	lastMemberEntry  audit.Entry
	lastOwnerSeed    project.OwnerMemberSeed
	createCalls      int
	updateCalls      int
	deleteCalls      int
	statusCalls      int
	createErr        error
	updateErr        error
	deleteErr        error
	statusErr        error
	findErr          error
	findResult       *project.Project // override for FindByIDForWorkspace
	suppressFindOnce bool
}

func newFakeProjectRepo() *fakeProjectRepo {
	return &fakeProjectRepo{projects: map[uuid.UUID]project.Project{}}
}

func (r *fakeProjectRepo) CreateWithAudit(ctx context.Context, p project.Project, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.createCalls++
	if r.createErr != nil {
		return r.createErr
	}
	r.projects[p.ID] = p
	r.lastEntry = entry
	return nil
}

func (r *fakeProjectRepo) CreateWithOwner(ctx context.Context, p project.Project, owner project.OwnerMemberSeed, projectEntry, memberEntry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.createCalls++
	r.lastOwnerSeed = owner
	r.lastMemberEntry = memberEntry
	if r.createErr != nil {
		return r.createErr
	}
	p.OwnerProjectMemberID = &owner.ID
	r.projects[p.ID] = p
	r.lastEntry = projectEntry
	return nil
}

func (r *fakeProjectRepo) FindByIDForWorkspace(ctx context.Context, workspaceID, projectID uuid.UUID) (*project.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.findErr != nil {
		return nil, r.findErr
	}
	if r.findResult != nil {
		cp := *r.findResult
		return &cp, nil
	}
	p, ok := r.projects[projectID]
	if !ok || p.WorkspaceID != workspaceID {
		return nil, nil
	}
	cp := p
	return &cp, nil
}

func (r *fakeProjectRepo) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, opts project.ListOptions) ([]project.Project, int64, error) {
	return nil, 0, nil
}

func (r *fakeProjectRepo) UpdateWithAudit(ctx context.Context, p project.Project, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updateCalls++
	if r.updateErr != nil {
		return r.updateErr
	}
	r.projects[p.ID] = p
	r.lastEntry = entry
	return nil
}

func (r *fakeProjectRepo) SoftDeleteWithAudit(ctx context.Context, workspaceID, projectID, deletedBy uuid.UUID, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deleteCalls++
	r.lastEntry = entry
	return r.deleteErr
}

func (r *fakeProjectRepo) ChangeStatusWithAudit(ctx context.Context, workspaceID, projectID uuid.UUID, newStatusCode string, updatedBy uuid.UUID, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.statusCalls++
	if r.statusErr != nil {
		return r.statusErr
	}
	p := r.projects[projectID]
	p.ProjectStatusCode = newStatusCode
	r.projects[projectID] = p
	r.lastEntry = entry
	return nil
}

type fakeMasterRepo struct {
	statusActive bool
	typeActive   bool
	roleActive   bool
	statusErr    error
	typeErr      error
	roleErr      error
}

func (r *fakeMasterRepo) IsActiveProjectStatusCode(ctx context.Context, code string) (bool, error) {
	if r.statusErr != nil {
		return false, r.statusErr
	}
	return r.statusActive, nil
}

func (r *fakeMasterRepo) IsActiveProjectTypeCode(ctx context.Context, code string) (bool, error) {
	if r.typeErr != nil {
		return false, r.typeErr
	}
	return r.typeActive, nil
}

func (r *fakeMasterRepo) IsActiveProjectRoleCode(ctx context.Context, code string) (bool, error) {
	if r.roleErr != nil {
		return false, r.roleErr
	}
	return r.roleActive, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newTC(t *testing.T) workspace.TenantContext {
	t.Helper()
	wsID, err := ids.New()
	if err != nil {
		t.Fatalf("ids.New: %v", err)
	}
	accID, err := ids.New()
	if err != nil {
		t.Fatalf("ids.New: %v", err)
	}
	return workspace.TenantContext{
		WorkspaceID: wsID,
		AccountID:   accID,
		OrgRoleCode: workspace.OrgRoleOwner,
	}
}

func buildSvc() (*project.Service, *fakeProjectRepo, *fakeMasterRepo) {
	pr := newFakeProjectRepo()
	mr := &fakeMasterRepo{statusActive: true, typeActive: true, roleActive: true}
	return project.NewService(pr, mr), pr, mr
}

func validCreateInput(tc workspace.TenantContext) project.CreateProjectInput {
	return project.CreateProjectInput{
		TenantCtx:         tc,
		ProjectName:       "Acme Site Redesign",
		Slug:              "acme-redesign",
		ProjectTypeCode:   "client",
		ProjectStatusCode: "planning",
	}
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestService_CreateProject_ValidationAggregates(t *testing.T) {
	svc, _, _ := buildSvc()
	tc := newTC(t)

	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC) // before start
	_, err := svc.CreateProject(context.Background(), project.CreateProjectInput{
		TenantCtx:         tc,
		ProjectName:       "   ", // empty after trim
		Slug:              "Has Space",
		ProjectTypeCode:   "client",
		ProjectStatusCode: "planning",
		StartDate:         &start,
		EndDate:           &end,
	})
	var valErr *project.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
	got := map[string]bool{}
	for _, fe := range valErr.Fields {
		got[fe.Field] = true
	}
	if !got["project_name"] {
		t.Errorf("missing project_name field error: %v", valErr.Fields)
	}
	if !got["slug"] {
		t.Errorf("missing slug field error: %v", valErr.Fields)
	}
	if !got["end_date"] {
		t.Errorf("missing end_date field error: %v", valErr.Fields)
	}
}

func TestService_CreateProject_DescriptionTooLong(t *testing.T) {
	svc, _, _ := buildSvc()
	tc := newTC(t)

	long := strings.Repeat("x", 10001)
	in := validCreateInput(tc)
	in.Description = &long
	_, err := svc.CreateProject(context.Background(), in)
	var valErr *project.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
	found := false
	for _, fe := range valErr.Fields {
		if fe.Field == "description" {
			found = true
		}
	}
	if !found {
		t.Errorf("missing description field error: %v", valErr.Fields)
	}
}

func TestService_CreateProject_RequestingUnitLengthCap(t *testing.T) {
	svc, _, _ := buildSvc()
	tc := newTC(t)

	long := strings.Repeat("u", 201)
	in := validCreateInput(tc)
	in.RequestingUnit = &long
	_, err := svc.CreateProject(context.Background(), in)
	var valErr *project.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
	found := false
	for _, fe := range valErr.Fields {
		if fe.Field == "requesting_unit" {
			found = true
		}
	}
	if !found {
		t.Errorf("missing requesting_unit field error: %v", valErr.Fields)
	}
}

func TestService_CreateProject_RequestingUnit_NullOK(t *testing.T) {
	svc, _, _ := buildSvc()
	tc := newTC(t)

	in := validCreateInput(tc)
	in.RequestingUnit = nil
	p, err := svc.CreateProject(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("nil project")
	}
}

func TestService_CreateProject_StatusCodeNotActive_Returns_ErrInvalidStatusCode(t *testing.T) {
	pr := newFakeProjectRepo()
	mr := &fakeMasterRepo{statusActive: false, typeActive: true, roleActive: true}
	svc := project.NewService(pr, mr)
	tc := newTC(t)

	_, err := svc.CreateProject(context.Background(), validCreateInput(tc))
	if !errors.Is(err, project.ErrInvalidStatusCode) {
		t.Errorf("err = %v, want ErrInvalidStatusCode", err)
	}
}

func TestService_CreateProject_TypeCodeNotActive_Returns_ErrInvalidTypeCode(t *testing.T) {
	pr := newFakeProjectRepo()
	mr := &fakeMasterRepo{statusActive: true, typeActive: false, roleActive: true}
	svc := project.NewService(pr, mr)
	tc := newTC(t)

	_, err := svc.CreateProject(context.Background(), validCreateInput(tc))
	if !errors.Is(err, project.ErrInvalidTypeCode) {
		t.Errorf("err = %v, want ErrInvalidTypeCode", err)
	}
}

func TestService_CreateProject_BuildsAuditEntry_WithBothWorkspaceAndProjectID(t *testing.T) {
	svc, pr, _ := buildSvc()
	tc := newTC(t)

	p, err := svc.CreateProject(context.Background(), validCreateInput(tc))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	pr.mu.Lock()
	entry := pr.lastEntry
	pr.mu.Unlock()

	if entry.WorkspaceID == nil || *entry.WorkspaceID != tc.WorkspaceID {
		t.Errorf("audit.WorkspaceID = %v, want %v", entry.WorkspaceID, tc.WorkspaceID)
	}
	if entry.ProjectID == nil || *entry.ProjectID != p.ID {
		t.Errorf("audit.ProjectID = %v, want %v", entry.ProjectID, p.ID)
	}
	if entry.Action != project.AuditActionProjectCreate {
		t.Errorf("audit.Action = %q, want project.create", entry.Action)
	}
}

func TestService_ChangeStatus_NoOp_WhenSame(t *testing.T) {
	svc, pr, _ := buildSvc()
	tc := newTC(t)

	// Seed an existing project.
	pid, _ := ids.New()
	pr.mu.Lock()
	pr.projects[pid] = project.Project{
		ID:                pid,
		WorkspaceID:       tc.WorkspaceID,
		ProjectStatusCode: "planning",
	}
	pr.mu.Unlock()

	out, err := svc.ChangeStatus(context.Background(), project.ChangeStatusInput{
		TenantCtx:     tc,
		ProjectID:     pid,
		NewStatusCode: "planning", // same
	})
	if err != nil {
		t.Fatalf("ChangeStatus: %v", err)
	}
	pr.mu.Lock()
	calls := pr.statusCalls
	pr.mu.Unlock()
	if calls != 0 {
		t.Errorf("statusCalls = %d, want 0 (no-op)", calls)
	}
	if out == nil || out.ProjectStatusCode != "planning" {
		t.Errorf("unexpected return: %+v", out)
	}
}

func TestService_ListProjects_ValidatesPageAndLimit(t *testing.T) {
	svc, _, _ := buildSvc()
	tc := newTC(t)

	_, _, err := svc.ListProjects(context.Background(), project.ListProjectsInput{
		TenantCtx: tc,
		Page:      -1,
		Limit:     10,
	})
	var valErr *project.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("Page=-1 → err = %v, want ValidationError", err)
	}
	if len(valErr.Fields) == 0 || valErr.Fields[0].Field != "page" {
		t.Errorf("expected field=page error, got %v", valErr.Fields)
	}

	_, _, err = svc.ListProjects(context.Background(), project.ListProjectsInput{
		TenantCtx: tc,
		Page:      1,
		Limit:     101,
	})
	if !errors.As(err, &valErr) {
		t.Fatalf("Limit=101 → err = %v, want ValidationError", err)
	}
	if len(valErr.Fields) == 0 || valErr.Fields[0].Field != "limit" {
		t.Errorf("expected field=limit error, got %v", valErr.Fields)
	}
}
