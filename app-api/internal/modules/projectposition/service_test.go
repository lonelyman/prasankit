package projectposition_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/projectposition"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
)

// ── Fake ──────────────────────────────────────────────────────────────────────

type fakeRepo struct {
	mu sync.Mutex

	createCalls    int
	updateCalls    int
	deprecateCalls int

	lastCreateEntry    audit.Entry
	lastUpdateEntry    audit.Entry
	lastDeprecateEntry audit.Entry

	createErr error

	// findResult is returned by FindByCodeForWorkspace (nil -> not found).
	findResult *projectposition.ProjectPosition
}

func (r *fakeRepo) CreateWithAudit(ctx context.Context, p projectposition.ProjectPosition, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.createCalls++
	r.lastCreateEntry = entry
	return r.createErr
}

func (r *fakeRepo) FindByCodeForWorkspace(ctx context.Context, workspaceID uuid.UUID, code string) (*projectposition.ProjectPosition, error) {
	if r.findResult != nil {
		cp := *r.findResult
		return &cp, nil
	}
	return nil, nil
}

func (r *fakeRepo) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, opts projectposition.ListOptions) ([]projectposition.ProjectPosition, int64, error) {
	return nil, 0, nil
}

func (r *fakeRepo) UpdateWithAudit(ctx context.Context, p projectposition.ProjectPosition, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updateCalls++
	r.lastUpdateEntry = entry
	return nil
}

func (r *fakeRepo) DeprecateWithAudit(ctx context.Context, workspaceID uuid.UUID, code string, updatedBy uuid.UUID, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deprecateCalls++
	r.lastDeprecateEntry = entry
	// reflect the deprecate so the re-find returns deprecated.
	if r.findResult != nil {
		r.findResult.Status = "deprecated"
	}
	return nil
}

var _ projectposition.ProjectPositionRepository = (*fakeRepo)(nil)

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

func existing(tc workspace.TenantContext, code, status string, isSystem bool) *projectposition.ProjectPosition {
	id, _ := ids.New()
	return &projectposition.ProjectPosition{
		ID:          id,
		WorkspaceID: tc.WorkspaceID,
		Code:        code,
		LabelTH:     "ตำแหน่ง",
		LabelEN:     "Position",
		SortOrder:   10,
		IsSystem:    isSystem,
		Status:      status,
	}
}

// ── Create ──────────────────────────────────────────────────────────────────────

func TestService_CreateProjectPosition_Success(t *testing.T) {
	tc := newTC()
	repo := &fakeRepo{}
	svc := projectposition.NewService(repo)

	p, err := svc.Create(context.Background(), projectposition.CreateInput{
		TenantCtx: tc, Code: "tech_lead", LabelTH: "หัวหน้าเทคนิค", LabelEN: "Technical Lead", SortOrder: 10,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p == nil || p.Code != "tech_lead" || p.Status != "active" || p.IsSystem {
		t.Fatalf("bad result: %+v", p)
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if repo.createCalls != 1 {
		t.Errorf("createCalls = %d, want 1", repo.createCalls)
	}
}

func TestService_CreateProjectPosition_ValidationErrors(t *testing.T) {
	tc := newTC()
	svc := projectposition.NewService(&fakeRepo{})

	cases := []projectposition.CreateInput{
		{TenantCtx: tc, Code: "", LabelTH: "a", LabelEN: "b"},                       // empty code
		{TenantCtx: tc, Code: "Bad Code", LabelTH: "a", LabelEN: "b"},               // bad code
		{TenantCtx: tc, Code: "ok_code", LabelTH: "", LabelEN: "b"},                 // empty label_th
		{TenantCtx: tc, Code: "ok_code", LabelTH: "a", LabelEN: "b", SortOrder: -1}, // sort<0
	}
	for i, in := range cases {
		_, err := svc.Create(context.Background(), in)
		var valErr *projectposition.ValidationError
		if !errors.As(err, &valErr) {
			t.Errorf("case %d: err = %v, want *ValidationError", i, err)
		}
	}
}

func TestService_CreateProjectPosition_CodeTaken_Active(t *testing.T) {
	tc := newTC()
	repo := &fakeRepo{findResult: existing(tc, "tech_lead", "active", false)}
	svc := projectposition.NewService(repo)

	_, err := svc.Create(context.Background(), projectposition.CreateInput{
		TenantCtx: tc, Code: "tech_lead", LabelTH: "a", LabelEN: "b",
	})
	if !errors.Is(err, projectposition.ErrCodeTaken) {
		t.Errorf("err = %v, want ErrCodeTaken", err)
	}
}

func TestService_CreateProjectPosition_CodeTaken_Deprecated(t *testing.T) {
	tc := newTC()
	repo := &fakeRepo{findResult: existing(tc, "tech_lead", "deprecated", false)}
	svc := projectposition.NewService(repo)

	_, err := svc.Create(context.Background(), projectposition.CreateInput{
		TenantCtx: tc, Code: "tech_lead", LabelTH: "a", LabelEN: "b",
	})
	if !errors.Is(err, projectposition.ErrCodeTaken) {
		t.Errorf("err = %v, want ErrCodeTaken (no-reuse §2.9)", err)
	}
}

// ── Update ──────────────────────────────────────────────────────────────────────

func TestService_UpdateProjectPosition_Success(t *testing.T) {
	tc := newTC()
	repo := &fakeRepo{findResult: existing(tc, "tech_lead", "active", false)}
	svc := projectposition.NewService(repo)

	p, err := svc.Update(context.Background(), projectposition.UpdateInput{
		TenantCtx: tc, Code: "tech_lead", LabelTH: "ใหม่", LabelEN: "New Lead", SortOrder: 20,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if p.LabelEN != "New Lead" || p.Code != "tech_lead" {
		t.Errorf("bad update result: %+v", p)
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if repo.updateCalls != 1 {
		t.Errorf("updateCalls = %d, want 1", repo.updateCalls)
	}
}

func TestService_UpdateProjectPosition_CodeImmutable(t *testing.T) {
	tc := newTC()
	repo := &fakeRepo{findResult: existing(tc, "tech_lead", "active", false)}
	svc := projectposition.NewService(repo)

	_, err := svc.Update(context.Background(), projectposition.UpdateInput{
		TenantCtx: tc, Code: "tech_lead", BodyCode: "different_code", LabelTH: "a", LabelEN: "b",
	})
	if !errors.Is(err, projectposition.ErrCodeImmutable) {
		t.Errorf("err = %v, want ErrCodeImmutable", err)
	}
}

func TestService_UpdateProjectPosition_SystemImmutable(t *testing.T) {
	tc := newTC()
	repo := &fakeRepo{findResult: existing(tc, "tech_lead", "active", true)} // is_system
	svc := projectposition.NewService(repo)

	_, err := svc.Update(context.Background(), projectposition.UpdateInput{
		TenantCtx: tc, Code: "tech_lead", LabelTH: "a", LabelEN: "b",
	})
	if !errors.Is(err, projectposition.ErrSystemImmutable) {
		t.Errorf("err = %v, want ErrSystemImmutable", err)
	}
}

func TestService_UpdateProjectPosition_NotFound(t *testing.T) {
	tc := newTC()
	repo := &fakeRepo{findResult: nil}
	svc := projectposition.NewService(repo)

	_, err := svc.Update(context.Background(), projectposition.UpdateInput{
		TenantCtx: tc, Code: "ghost", LabelTH: "a", LabelEN: "b",
	})
	if !errors.Is(err, projectposition.ErrPositionNotFound) {
		t.Errorf("err = %v, want ErrPositionNotFound", err)
	}
}

// ── Deprecate ─────────────────────────────────────────────────────────────────

func TestService_DeprecateProjectPosition_Success(t *testing.T) {
	tc := newTC()
	repo := &fakeRepo{findResult: existing(tc, "tech_lead", "active", false)}
	svc := projectposition.NewService(repo)

	p, err := svc.Deprecate(context.Background(), projectposition.DeprecateInput{TenantCtx: tc, Code: "tech_lead"})
	if err != nil {
		t.Fatalf("Deprecate: %v", err)
	}
	if p.Status != "deprecated" {
		t.Errorf("status = %q, want deprecated", p.Status)
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if repo.deprecateCalls != 1 {
		t.Errorf("deprecateCalls = %d, want 1", repo.deprecateCalls)
	}
}

func TestService_DeprecateProjectPosition_NoopWhenAlreadyDeprecated(t *testing.T) {
	tc := newTC()
	repo := &fakeRepo{findResult: existing(tc, "tech_lead", "deprecated", false)}
	svc := projectposition.NewService(repo)

	_, err := svc.Deprecate(context.Background(), projectposition.DeprecateInput{TenantCtx: tc, Code: "tech_lead"})
	if err != nil {
		t.Fatalf("Deprecate: %v", err)
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if repo.deprecateCalls != 0 {
		t.Errorf("deprecateCalls = %d, want 0 (no-op)", repo.deprecateCalls)
	}
}

func TestService_DeprecateProjectPosition_SystemImmutable(t *testing.T) {
	tc := newTC()
	repo := &fakeRepo{findResult: existing(tc, "tech_lead", "active", true)}
	svc := projectposition.NewService(repo)

	_, err := svc.Deprecate(context.Background(), projectposition.DeprecateInput{TenantCtx: tc, Code: "tech_lead"})
	if !errors.Is(err, projectposition.ErrSystemImmutable) {
		t.Errorf("err = %v, want ErrSystemImmutable", err)
	}
}

// ── Audit scoping ─────────────────────────────────────────────────────────────

func TestService_CreateProjectPosition_AuditEntryIsWorkspaceScoped(t *testing.T) {
	tc := newTC()
	repo := &fakeRepo{}
	svc := projectposition.NewService(repo)

	if _, err := svc.Create(context.Background(), projectposition.CreateInput{
		TenantCtx: tc, Code: "tech_lead", LabelTH: "a", LabelEN: "b",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	e := repo.lastCreateEntry
	if e.ProjectID != nil {
		t.Errorf("audit.ProjectID = %v, want nil (workspace-scoped)", e.ProjectID)
	}
	if e.WorkspaceID == nil || *e.WorkspaceID != tc.WorkspaceID {
		t.Errorf("audit.WorkspaceID = %v, want %v", e.WorkspaceID, tc.WorkspaceID)
	}
}
