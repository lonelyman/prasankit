package deliverable_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/deliverable"
	"prasankit-api/internal/modules/workspace"

	"github.com/google/uuid"
)

// ── Fakes ─────────────────────────────────────────────────────────────────────

type fakeRepo struct {
	mu sync.Mutex

	createCalls int
	submitCalls int
	reviewCalls int

	lastCreated       deliverable.Deliverable
	lastSubmission    deliverable.Submission
	lastReview        deliverable.SubmissionReview
	lastDeliverableID uuid.UUID

	findResult *deliverable.Deliverable // FindByIDForWorkspace
	findNil    bool

	getResult *deliverable.DeliverableWithStatus

	submitErr error
	reviewErr error

	createErr error
}

func (r *fakeRepo) CreateWithAudit(ctx context.Context, d deliverable.Deliverable, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.createCalls++
	r.lastCreated = d
	return r.createErr
}
func (r *fakeRepo) FindByIDForWorkspace(ctx context.Context, workspaceID, deliverableID uuid.UUID) (*deliverable.Deliverable, error) {
	if r.findNil {
		return nil, nil
	}
	if r.findResult != nil {
		cp := *r.findResult
		return &cp, nil
	}
	return &deliverable.Deliverable{ID: deliverableID, WorkspaceID: workspaceID}, nil
}
func (r *fakeRepo) ListByProjectWithStatus(ctx context.Context, workspaceID, projectID uuid.UUID) ([]deliverable.DeliverableWithStatus, error) {
	return []deliverable.DeliverableWithStatus{}, nil
}
func (r *fakeRepo) GetWithStatus(ctx context.Context, workspaceID, deliverableID uuid.UUID) (*deliverable.DeliverableWithStatus, error) {
	if r.getResult != nil {
		cp := *r.getResult
		return &cp, nil
	}
	return nil, nil
}
func (r *fakeRepo) UpdateWithAudit(ctx context.Context, d deliverable.Deliverable, entry audit.Entry) error {
	return nil
}
func (r *fakeRepo) SoftDeleteWithAudit(ctx context.Context, workspaceID, deliverableID, deletedBy uuid.UUID, entry audit.Entry) error {
	return nil
}
func (r *fakeRepo) ListSubmissionsForDeliverable(ctx context.Context, workspaceID, deliverableID uuid.UUID) ([]deliverable.SubmissionWithReview, error) {
	return []deliverable.SubmissionWithReview{}, nil
}
func (r *fakeRepo) CreateSubmissionLocked(ctx context.Context, s deliverable.Submission, entry audit.Entry) (*deliverable.Submission, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.submitCalls++
	r.lastSubmission = s
	if r.submitErr != nil {
		return nil, r.submitErr
	}
	s.RoundNo = 1
	return &s, nil
}
func (r *fakeRepo) CreateReviewLocked(ctx context.Context, deliverableID uuid.UUID, rv deliverable.SubmissionReview, entry audit.Entry) (*deliverable.SubmissionReview, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reviewCalls++
	r.lastReview = rv
	r.lastDeliverableID = deliverableID
	if r.reviewErr != nil {
		return nil, r.reviewErr
	}
	return &rv, nil
}

type fakeMaster struct {
	decisionActive bool
	decisionErr    error
}

func (m *fakeMaster) IsActiveSubmissionDecisionCode(ctx context.Context, code string) (bool, error) {
	return m.decisionActive, m.decisionErr
}

type fakeAccess struct {
	projectExists bool
	member        *deliverable.ActorMember
	existsErr     error
	memberErr     error
}

func (a *fakeAccess) ProjectExistsForWorkspace(ctx context.Context, workspaceID, projectID uuid.UUID) (bool, error) {
	return a.projectExists, a.existsErr
}
func (a *fakeAccess) FindActiveMemberByMembership(ctx context.Context, workspaceID, projectID, membershipID uuid.UUID) (*deliverable.ActorMember, error) {
	if a.memberErr != nil {
		return nil, a.memberErr
	}
	if a.member != nil {
		cp := *a.member
		return &cp, nil
	}
	return nil, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newSvc(repo *fakeRepo, master *fakeMaster, access *fakeAccess) *deliverable.Service {
	if repo == nil {
		repo = &fakeRepo{}
	}
	if master == nil {
		master = &fakeMaster{decisionActive: true}
	}
	if access == nil {
		access = &fakeAccess{projectExists: true}
	}
	return deliverable.NewService(repo, master, access)
}

func tc(orgRole string) workspace.TenantContext {
	return workspace.TenantContext{
		WorkspaceID:  uuid.New(),
		MembershipID: uuid.New(),
		AccountID:    uuid.New(),
		OrgRoleCode:  orgRole,
	}
}

func memberOf(role string) *deliverable.ActorMember {
	return &deliverable.ActorMember{ID: uuid.New(), ProjectRoleCode: role}
}

// ── Deliverable CRUD authz ────────────────────────────────────────────────────

func TestCreateDeliverable_ManagerAllowed(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, nil, &fakeAccess{projectExists: true, member: memberOf(deliverable.ProjectRoleManager)})
	_, err := svc.CreateDeliverable(context.Background(), deliverable.CreateDeliverableInput{
		TenantCtx: tc(workspace.OrgRoleUser), ProjectID: uuid.New(), Title: "Phase 1",
	})
	if err != nil {
		t.Fatalf("manager create: unexpected err %v", err)
	}
	if repo.createCalls != 1 {
		t.Errorf("createCalls = %d, want 1", repo.createCalls)
	}
}

func TestCreateDeliverable_PlainMemberDenied(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, nil, &fakeAccess{projectExists: true, member: memberOf("member")})
	_, err := svc.CreateDeliverable(context.Background(), deliverable.CreateDeliverableInput{
		TenantCtx: tc(workspace.OrgRoleUser), ProjectID: uuid.New(), Title: "Phase 1",
	})
	if !errors.Is(err, deliverable.ErrForbidden) {
		t.Fatalf("plain member create: got %v, want ErrForbidden", err)
	}
	if repo.createCalls != 0 {
		t.Errorf("createCalls = %d, want 0 (should not reach repo)", repo.createCalls)
	}
}

func TestCreateDeliverable_OrgAdminBypassAllowed(t *testing.T) {
	repo := &fakeRepo{}
	// Org admin, NOT a project member → still allowed for CRUD (bypass).
	svc := newSvc(repo, nil, &fakeAccess{projectExists: true, member: nil})
	_, err := svc.CreateDeliverable(context.Background(), deliverable.CreateDeliverableInput{
		TenantCtx: tc(workspace.OrgRoleAdmin), ProjectID: uuid.New(), Title: "Phase 1",
	})
	if err != nil {
		t.Fatalf("org admin create: unexpected err %v", err)
	}
	if repo.createCalls != 1 {
		t.Errorf("createCalls = %d, want 1", repo.createCalls)
	}
}

func TestCreateDeliverable_ProjectNotFound(t *testing.T) {
	svc := newSvc(nil, nil, &fakeAccess{projectExists: false})
	_, err := svc.CreateDeliverable(context.Background(), deliverable.CreateDeliverableInput{
		TenantCtx: tc(workspace.OrgRoleAdmin), ProjectID: uuid.New(), Title: "Phase 1",
	})
	if !errors.Is(err, deliverable.ErrProjectNotFound) {
		t.Fatalf("missing project: got %v, want ErrProjectNotFound", err)
	}
}

func TestCreateDeliverable_TitleValidation(t *testing.T) {
	svc := newSvc(nil, nil, &fakeAccess{projectExists: true, member: memberOf(deliverable.ProjectRoleOwner)})
	_, err := svc.CreateDeliverable(context.Background(), deliverable.CreateDeliverableInput{
		TenantCtx: tc(workspace.OrgRoleUser), ProjectID: uuid.New(), Title: "   ",
	})
	var ve *deliverable.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("empty title: got %v, want ValidationError", err)
	}
}

// ── Read authz (non-member → 404 probe-collapse) ────────────────────────────────

func TestListDeliverables_NonMemberNotFound(t *testing.T) {
	// Org user (no bypass), not a project member → 404, not 403.
	svc := newSvc(nil, nil, &fakeAccess{projectExists: true, member: nil})
	_, err := svc.ListDeliverables(context.Background(), deliverable.ListDeliverablesInput{
		TenantCtx: tc(workspace.OrgRoleUser), ProjectID: uuid.New(),
	})
	if !errors.Is(err, deliverable.ErrProjectNotFound) {
		t.Fatalf("non-member list: got %v, want ErrProjectNotFound (probe-collapse)", err)
	}
}

func TestListDeliverables_ViewerAllowed(t *testing.T) {
	svc := newSvc(nil, nil, &fakeAccess{projectExists: true, member: memberOf(deliverable.ProjectRoleViewer)})
	_, err := svc.ListDeliverables(context.Background(), deliverable.ListDeliverablesInput{
		TenantCtx: tc(workspace.OrgRoleUser), ProjectID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("viewer list: unexpected err %v", err)
	}
}

// ── Submit authz + server-derive ────────────────────────────────────────────────

func TestSubmit_ViewerDenied(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, nil, &fakeAccess{projectExists: true, member: memberOf(deliverable.ProjectRoleViewer)})
	_, err := svc.Submit(context.Background(), deliverable.SubmitInput{
		TenantCtx: tc(workspace.OrgRoleUser), ProjectID: uuid.New(), DeliverableID: uuid.New(),
	})
	if !errors.Is(err, deliverable.ErrForbidden) {
		t.Fatalf("viewer submit: got %v, want ErrForbidden", err)
	}
	if repo.submitCalls != 0 {
		t.Errorf("submitCalls = %d, want 0", repo.submitCalls)
	}
}

func TestSubmit_PlainMemberAllowed_ServerDerivesSubmittedBy(t *testing.T) {
	repo := &fakeRepo{}
	m := memberOf("member")
	svc := newSvc(repo, nil, &fakeAccess{projectExists: true, member: m})
	_, err := svc.Submit(context.Background(), deliverable.SubmitInput{
		TenantCtx: tc(workspace.OrgRoleUser), ProjectID: uuid.New(), DeliverableID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("member submit: unexpected err %v", err)
	}
	// submitted_by MUST be the resolved member id (server-derived, D60) — not anything from body.
	if repo.lastSubmission.SubmittedBy != m.ID {
		t.Errorf("submitted_by = %v, want resolved member id %v (server-derive)", repo.lastSubmission.SubmittedBy, m.ID)
	}
}

func TestSubmit_OrgAdminNonMember_NotProjectMember(t *testing.T) {
	repo := &fakeRepo{}
	// Org admin but NOT a project member → 422 not_project_member (FK needs a member row).
	svc := newSvc(repo, nil, &fakeAccess{projectExists: true, member: nil})
	_, err := svc.Submit(context.Background(), deliverable.SubmitInput{
		TenantCtx: tc(workspace.OrgRoleAdmin), ProjectID: uuid.New(), DeliverableID: uuid.New(),
	})
	if !errors.Is(err, deliverable.ErrNotProjectMember) {
		t.Fatalf("org admin non-member submit: got %v, want ErrNotProjectMember", err)
	}
	if repo.submitCalls != 0 {
		t.Errorf("submitCalls = %d, want 0", repo.submitCalls)
	}
}

func TestSubmit_NoteTooLong(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, nil, &fakeAccess{projectExists: true, member: memberOf("member")})
	big := make([]byte, 10001)
	for i := range big {
		big[i] = 'x'
	}
	s := string(big)
	_, err := svc.Submit(context.Background(), deliverable.SubmitInput{
		TenantCtx: tc(workspace.OrgRoleUser), ProjectID: uuid.New(), DeliverableID: uuid.New(), Note: &s,
	})
	var ve *deliverable.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("note too long: got %v, want ValidationError", err)
	}
}

// ── Review authz + server-derive + master check ─────────────────────────────────

func TestReview_PlainMemberDenied(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, &fakeMaster{decisionActive: true}, &fakeAccess{projectExists: true, member: memberOf("member")})
	_, err := svc.Review(context.Background(), deliverable.ReviewInput{
		TenantCtx: tc(workspace.OrgRoleUser), ProjectID: uuid.New(), DeliverableID: uuid.New(),
		SubmissionID: uuid.New(), DecisionCode: deliverable.SubmissionDecisionAccepted,
	})
	if !errors.Is(err, deliverable.ErrForbidden) {
		t.Fatalf("member review: got %v, want ErrForbidden", err)
	}
	if repo.reviewCalls != 0 {
		t.Errorf("reviewCalls = %d, want 0", repo.reviewCalls)
	}
}

func TestReview_ManagerAllowed_ServerDerivesReviewedBy(t *testing.T) {
	repo := &fakeRepo{}
	m := memberOf(deliverable.ProjectRoleManager)
	svc := newSvc(repo, &fakeMaster{decisionActive: true}, &fakeAccess{projectExists: true, member: m})
	delivID := uuid.New()
	_, err := svc.Review(context.Background(), deliverable.ReviewInput{
		TenantCtx: tc(workspace.OrgRoleUser), ProjectID: uuid.New(), DeliverableID: delivID,
		SubmissionID: uuid.New(), DecisionCode: deliverable.SubmissionDecisionAccepted,
	})
	if err != nil {
		t.Fatalf("manager review: unexpected err %v", err)
	}
	if repo.lastReview.ReviewedBy != m.ID {
		t.Errorf("reviewed_by = %v, want resolved member id %v (server-derive)", repo.lastReview.ReviewedBy, m.ID)
	}
	if repo.lastDeliverableID != delivID {
		t.Errorf("deliverableID passed to repo = %v, want %v", repo.lastDeliverableID, delivID)
	}
}

func TestReview_InvalidDecisionCode(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, &fakeMaster{decisionActive: false}, &fakeAccess{projectExists: true, member: memberOf(deliverable.ProjectRoleOwner)})
	_, err := svc.Review(context.Background(), deliverable.ReviewInput{
		TenantCtx: tc(workspace.OrgRoleUser), ProjectID: uuid.New(), DeliverableID: uuid.New(),
		SubmissionID: uuid.New(), DecisionCode: "bogus",
	})
	if !errors.Is(err, deliverable.ErrInvalidMasterCode) {
		t.Fatalf("invalid decision: got %v, want ErrInvalidMasterCode", err)
	}
	if repo.reviewCalls != 0 {
		t.Errorf("reviewCalls = %d, want 0", repo.reviewCalls)
	}
}

func TestReview_OrgAdminNonMember_NotProjectMember(t *testing.T) {
	repo := &fakeRepo{}
	svc := newSvc(repo, &fakeMaster{decisionActive: true}, &fakeAccess{projectExists: true, member: nil})
	_, err := svc.Review(context.Background(), deliverable.ReviewInput{
		TenantCtx: tc(workspace.OrgRoleAdmin), ProjectID: uuid.New(), DeliverableID: uuid.New(),
		SubmissionID: uuid.New(), DecisionCode: deliverable.SubmissionDecisionAccepted,
	})
	if !errors.Is(err, deliverable.ErrNotProjectMember) {
		t.Fatalf("org admin non-member review: got %v, want ErrNotProjectMember", err)
	}
}

// ── error propagation from repo (terminal/superseded etc) ────────────────────────

func TestSubmit_AcceptTerminalPropagates(t *testing.T) {
	repo := &fakeRepo{submitErr: deliverable.ErrAcceptTerminal}
	svc := newSvc(repo, nil, &fakeAccess{projectExists: true, member: memberOf("member")})
	_, err := svc.Submit(context.Background(), deliverable.SubmitInput{
		TenantCtx: tc(workspace.OrgRoleUser), ProjectID: uuid.New(), DeliverableID: uuid.New(),
	})
	if !errors.Is(err, deliverable.ErrAcceptTerminal) {
		t.Fatalf("got %v, want ErrAcceptTerminal", err)
	}
}

func TestReview_SupersededPropagates(t *testing.T) {
	repo := &fakeRepo{reviewErr: deliverable.ErrSubmissionSuperseded}
	svc := newSvc(repo, &fakeMaster{decisionActive: true}, &fakeAccess{projectExists: true, member: memberOf(deliverable.ProjectRoleOwner)})
	_, err := svc.Review(context.Background(), deliverable.ReviewInput{
		TenantCtx: tc(workspace.OrgRoleUser), ProjectID: uuid.New(), DeliverableID: uuid.New(),
		SubmissionID: uuid.New(), DecisionCode: deliverable.SubmissionDecisionAccepted,
	})
	if !errors.Is(err, deliverable.ErrSubmissionSuperseded) {
		t.Fatalf("got %v, want ErrSubmissionSuperseded", err)
	}
}
