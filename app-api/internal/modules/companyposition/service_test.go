package companyposition_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/companyposition"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

type fakeRepo struct {
	mu sync.Mutex

	createCalls    int
	updateCalls    int
	deprecateCalls int

	lastCreateEntry audit.Entry

	createErr  error
	findResult *companyposition.CompanyPosition
}

func (r *fakeRepo) CreateWithAudit(ctx context.Context, p companyposition.CompanyPosition, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.createCalls++
	r.lastCreateEntry = entry
	return r.createErr
}

func (r *fakeRepo) FindByCodeForWorkspace(ctx context.Context, workspaceID uuid.UUID, code string) (*companyposition.CompanyPosition, error) {
	if r.findResult != nil {
		cp := *r.findResult
		return &cp, nil
	}
	return nil, nil
}

func (r *fakeRepo) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, opts companyposition.ListOptions) ([]companyposition.CompanyPosition, int64, error) {
	return nil, 0, nil
}

func (r *fakeRepo) UpdateWithAudit(ctx context.Context, p companyposition.CompanyPosition, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updateCalls++
	return nil
}

func (r *fakeRepo) DeprecateWithAudit(ctx context.Context, workspaceID uuid.UUID, code string, updatedBy uuid.UUID, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deprecateCalls++
	if r.findResult != nil {
		r.findResult.Status = "deprecated"
	}
	return nil
}

type fakeMaster struct {
	active bool
}

func (r *fakeMaster) IsActiveCompanyPositionCode(ctx context.Context, workspaceID uuid.UUID, code string) (bool, error) {
	return r.active, nil
}

type fakeMembership struct {
	mu sync.Mutex

	calls     int
	lastEntry audit.Entry
	lastCode  *string
	setErr    error
}

func (r *fakeMembership) SetCompanyPositionWithAudit(ctx context.Context, workspaceID, membershipID uuid.UUID, code *string, updatedBy uuid.UUID, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	r.lastEntry = entry
	r.lastCode = code
	return r.setErr
}

var _ companyposition.CompanyPositionRepository = (*fakeRepo)(nil)
var _ companyposition.CompanyPositionMasterRepository = (*fakeMaster)(nil)
var _ companyposition.MembershipPositionRepository = (*fakeMembership)(nil)

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

func existing(tc workspace.TenantContext, code, status string, isSystem bool) *companyposition.CompanyPosition {
	id, _ := ids.New()
	return &companyposition.CompanyPosition{
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

func strPtr(s string) *string { return &s }

// ── Master CRUD ───────────────────────────────────────────────────────────────

func TestService_CreateCompanyPosition_Success(t *testing.T) {
	tc := newTC()
	repo := &fakeRepo{}
	svc := companyposition.NewService(repo, &fakeMaster{active: true}, &fakeMembership{})

	p, err := svc.Create(context.Background(), companyposition.CreateInput{
		TenantCtx: tc, Code: "engineer", LabelTH: "วิศวกร", LabelEN: "Engineer", SortOrder: 10,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p == nil || p.Code != "engineer" || p.Status != "active" {
		t.Fatalf("bad result: %+v", p)
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if repo.createCalls != 1 {
		t.Errorf("createCalls = %d, want 1", repo.createCalls)
	}
}

func TestService_CreateCompanyPosition_ValidationErrors(t *testing.T) {
	tc := newTC()
	svc := companyposition.NewService(&fakeRepo{}, &fakeMaster{}, &fakeMembership{})
	cases := []companyposition.CreateInput{
		{TenantCtx: tc, Code: "", LabelTH: "a", LabelEN: "b"},
		{TenantCtx: tc, Code: "bad code", LabelTH: "a", LabelEN: "b"}, // space -> invalid even lowercased
		{TenantCtx: tc, Code: "ok_code", LabelTH: "", LabelEN: "b"},
		{TenantCtx: tc, Code: "ok_code", LabelTH: "a", LabelEN: "b", SortOrder: -1},
	}
	for i, in := range cases {
		_, err := svc.Create(context.Background(), in)
		var valErr *companyposition.ValidationError
		if !errors.As(err, &valErr) {
			t.Errorf("case %d: err = %v, want *ValidationError", i, err)
		}
	}
}

func TestService_CreateCompanyPosition_CodeTaken_Active(t *testing.T) {
	tc := newTC()
	repo := &fakeRepo{findResult: existing(tc, "engineer", "active", false)}
	svc := companyposition.NewService(repo, &fakeMaster{}, &fakeMembership{})
	_, err := svc.Create(context.Background(), companyposition.CreateInput{TenantCtx: tc, Code: "engineer", LabelTH: "a", LabelEN: "b"})
	if !errors.Is(err, companyposition.ErrCodeTaken) {
		t.Errorf("err = %v, want ErrCodeTaken", err)
	}
}

func TestService_CreateCompanyPosition_CodeTaken_Deprecated(t *testing.T) {
	tc := newTC()
	repo := &fakeRepo{findResult: existing(tc, "engineer", "deprecated", false)}
	svc := companyposition.NewService(repo, &fakeMaster{}, &fakeMembership{})
	_, err := svc.Create(context.Background(), companyposition.CreateInput{TenantCtx: tc, Code: "engineer", LabelTH: "a", LabelEN: "b"})
	if !errors.Is(err, companyposition.ErrCodeTaken) {
		t.Errorf("err = %v, want ErrCodeTaken (no-reuse §2.9)", err)
	}
}

func TestService_UpdateCompanyPosition_Success(t *testing.T) {
	tc := newTC()
	repo := &fakeRepo{findResult: existing(tc, "engineer", "active", false)}
	svc := companyposition.NewService(repo, &fakeMaster{}, &fakeMembership{})
	p, err := svc.Update(context.Background(), companyposition.UpdateInput{TenantCtx: tc, Code: "engineer", LabelTH: "ใหม่", LabelEN: "New", SortOrder: 5})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if p.LabelEN != "New" {
		t.Errorf("label = %q, want New", p.LabelEN)
	}
}

func TestService_UpdateCompanyPosition_CodeImmutable(t *testing.T) {
	tc := newTC()
	repo := &fakeRepo{findResult: existing(tc, "engineer", "active", false)}
	svc := companyposition.NewService(repo, &fakeMaster{}, &fakeMembership{})
	_, err := svc.Update(context.Background(), companyposition.UpdateInput{TenantCtx: tc, Code: "engineer", BodyCode: "other", LabelTH: "a", LabelEN: "b"})
	if !errors.Is(err, companyposition.ErrCodeImmutable) {
		t.Errorf("err = %v, want ErrCodeImmutable", err)
	}
}

func TestService_UpdateCompanyPosition_SystemImmutable(t *testing.T) {
	tc := newTC()
	repo := &fakeRepo{findResult: existing(tc, "engineer", "active", true)}
	svc := companyposition.NewService(repo, &fakeMaster{}, &fakeMembership{})
	_, err := svc.Update(context.Background(), companyposition.UpdateInput{TenantCtx: tc, Code: "engineer", LabelTH: "a", LabelEN: "b"})
	if !errors.Is(err, companyposition.ErrSystemImmutable) {
		t.Errorf("err = %v, want ErrSystemImmutable", err)
	}
}

func TestService_UpdateCompanyPosition_NotFound(t *testing.T) {
	tc := newTC()
	svc := companyposition.NewService(&fakeRepo{findResult: nil}, &fakeMaster{}, &fakeMembership{})
	_, err := svc.Update(context.Background(), companyposition.UpdateInput{TenantCtx: tc, Code: "ghost", LabelTH: "a", LabelEN: "b"})
	if !errors.Is(err, companyposition.ErrPositionNotFound) {
		t.Errorf("err = %v, want ErrPositionNotFound", err)
	}
}

func TestService_DeprecateCompanyPosition_Success(t *testing.T) {
	tc := newTC()
	repo := &fakeRepo{findResult: existing(tc, "engineer", "active", false)}
	svc := companyposition.NewService(repo, &fakeMaster{}, &fakeMembership{})
	p, err := svc.Deprecate(context.Background(), companyposition.DeprecateInput{TenantCtx: tc, Code: "engineer"})
	if err != nil {
		t.Fatalf("Deprecate: %v", err)
	}
	if p.Status != "deprecated" {
		t.Errorf("status = %q, want deprecated", p.Status)
	}
}

func TestService_DeprecateCompanyPosition_NoopWhenAlreadyDeprecated(t *testing.T) {
	tc := newTC()
	repo := &fakeRepo{findResult: existing(tc, "engineer", "deprecated", false)}
	svc := companyposition.NewService(repo, &fakeMaster{}, &fakeMembership{})
	if _, err := svc.Deprecate(context.Background(), companyposition.DeprecateInput{TenantCtx: tc, Code: "engineer"}); err != nil {
		t.Fatalf("Deprecate: %v", err)
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if repo.deprecateCalls != 0 {
		t.Errorf("deprecateCalls = %d, want 0 (no-op)", repo.deprecateCalls)
	}
}

func TestService_DeprecateCompanyPosition_SystemImmutable(t *testing.T) {
	tc := newTC()
	repo := &fakeRepo{findResult: existing(tc, "engineer", "active", true)}
	svc := companyposition.NewService(repo, &fakeMaster{}, &fakeMembership{})
	_, err := svc.Deprecate(context.Background(), companyposition.DeprecateInput{TenantCtx: tc, Code: "engineer"})
	if !errors.Is(err, companyposition.ErrSystemImmutable) {
		t.Errorf("err = %v, want ErrSystemImmutable", err)
	}
}

func TestService_CreateCompanyPosition_AuditEntryIsWorkspaceScoped(t *testing.T) {
	tc := newTC()
	repo := &fakeRepo{}
	svc := companyposition.NewService(repo, &fakeMaster{active: true}, &fakeMembership{})
	if _, err := svc.Create(context.Background(), companyposition.CreateInput{TenantCtx: tc, Code: "engineer", LabelTH: "a", LabelEN: "b"}); err != nil {
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

// ── ws company_position attach ──────────────────────────────────────────────────

func TestService_SetCompanyPosition_Success(t *testing.T) {
	tc := newTC()
	mem := &fakeMembership{}
	svc := companyposition.NewService(&fakeRepo{}, &fakeMaster{active: true}, mem)
	mID, _ := ids.New()

	if err := svc.SetCompanyPosition(context.Background(), companyposition.SetCompanyPositionInput{
		TenantCtx: tc, MembershipID: mID, CompanyPositionCode: strPtr("engineer"),
	}); err != nil {
		t.Fatalf("SetCompanyPosition: %v", err)
	}
	mem.mu.Lock()
	defer mem.mu.Unlock()
	if mem.calls != 1 {
		t.Errorf("calls = %d, want 1", mem.calls)
	}
	if mem.lastCode == nil || *mem.lastCode != "engineer" {
		t.Errorf("lastCode = %v, want engineer", mem.lastCode)
	}
}

func TestService_SetCompanyPosition_ClearWhenNil(t *testing.T) {
	tc := newTC()
	mem := &fakeMembership{}
	svc := companyposition.NewService(&fakeRepo{}, &fakeMaster{active: true}, mem)
	mID, _ := ids.New()

	if err := svc.SetCompanyPosition(context.Background(), companyposition.SetCompanyPositionInput{
		TenantCtx: tc, MembershipID: mID, CompanyPositionCode: nil,
	}); err != nil {
		t.Fatalf("SetCompanyPosition (clear): %v", err)
	}
	mem.mu.Lock()
	defer mem.mu.Unlock()
	if mem.lastCode != nil {
		t.Errorf("lastCode = %v, want nil (clear)", mem.lastCode)
	}
	if mem.lastEntry.NewValue != nil {
		t.Errorf("NewValue = %v, want nil (clear)", mem.lastEntry.NewValue)
	}
}

func TestService_SetCompanyPosition_InvalidCode(t *testing.T) {
	tc := newTC()
	mem := &fakeMembership{}
	svc := companyposition.NewService(&fakeRepo{}, &fakeMaster{active: false}, mem)
	mID, _ := ids.New()

	err := svc.SetCompanyPosition(context.Background(), companyposition.SetCompanyPositionInput{
		TenantCtx: tc, MembershipID: mID, CompanyPositionCode: strPtr("ghost"),
	})
	if !errors.Is(err, companyposition.ErrInvalidPositionCode) {
		t.Errorf("err = %v, want ErrInvalidPositionCode", err)
	}
	mem.mu.Lock()
	defer mem.mu.Unlock()
	if mem.calls != 0 {
		t.Errorf("calls = %d, want 0 (pre-check fails before write)", mem.calls)
	}
}

func TestService_SetCompanyPosition_MembershipNotFound(t *testing.T) {
	tc := newTC()
	mem := &fakeMembership{setErr: companyposition.ErrMembershipNotFound}
	svc := companyposition.NewService(&fakeRepo{}, &fakeMaster{active: true}, mem)
	mID, _ := ids.New()

	err := svc.SetCompanyPosition(context.Background(), companyposition.SetCompanyPositionInput{
		TenantCtx: tc, MembershipID: mID, CompanyPositionCode: strPtr("engineer"),
	})
	if !errors.Is(err, companyposition.ErrMembershipNotFound) {
		t.Errorf("err = %v, want ErrMembershipNotFound", err)
	}
}

func TestService_SetCompanyPosition_AuditEntryIsWorkspaceScoped(t *testing.T) {
	tc := newTC()
	mem := &fakeMembership{}
	svc := companyposition.NewService(&fakeRepo{}, &fakeMaster{active: true}, mem)
	mID, _ := ids.New()

	if err := svc.SetCompanyPosition(context.Background(), companyposition.SetCompanyPositionInput{
		TenantCtx: tc, MembershipID: mID, CompanyPositionCode: strPtr("engineer"),
	}); err != nil {
		t.Fatalf("SetCompanyPosition: %v", err)
	}
	mem.mu.Lock()
	defer mem.mu.Unlock()
	e := mem.lastEntry
	if e.ProjectID != nil {
		t.Errorf("audit.ProjectID = %v, want nil (workspace-scoped)", e.ProjectID)
	}
	if e.WorkspaceID == nil || *e.WorkspaceID != tc.WorkspaceID {
		t.Errorf("audit.WorkspaceID = %v, want %v", e.WorkspaceID, tc.WorkspaceID)
	}
	if e.Action != companyposition.AuditActionMembershipCompanyPositionChange {
		t.Errorf("audit.Action = %q, want membership.company_position_change", e.Action)
	}
}
