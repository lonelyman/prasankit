package deliverablehandler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/deliverable"
	"prasankit-api/internal/modules/workspace"
	deliverablehandler "prasankit-api/internal/transport/http/deliverable"
	workspacehandler "prasankit-api/internal/transport/http/workspace"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// ── fakes ─────────────────────────────────────────────────────────────────────

type fakeRepo struct {
	submitResult *deliverable.Submission
	submitErr    error
	reviewResult *deliverable.SubmissionReview
	reviewErr    error
	createErr    error
	getResult    *deliverable.DeliverableWithStatus
	listSubs     []deliverable.SubmissionWithReview
}

func (r *fakeRepo) CreateWithAudit(ctx context.Context, d deliverable.Deliverable, e audit.Entry) error {
	return r.createErr
}
func (r *fakeRepo) FindByIDForWorkspace(ctx context.Context, ws, id uuid.UUID) (*deliverable.Deliverable, error) {
	if r.getResult == nil {
		return nil, nil
	}
	d := r.getResult.Deliverable
	return &d, nil
}
func (r *fakeRepo) ListByProjectWithStatus(ctx context.Context, ws, pid uuid.UUID) ([]deliverable.DeliverableWithStatus, error) {
	if r.getResult != nil {
		return []deliverable.DeliverableWithStatus{*r.getResult}, nil
	}
	return []deliverable.DeliverableWithStatus{}, nil
}
func (r *fakeRepo) GetWithStatus(ctx context.Context, ws, id uuid.UUID) (*deliverable.DeliverableWithStatus, error) {
	return r.getResult, nil
}
func (r *fakeRepo) UpdateWithAudit(ctx context.Context, d deliverable.Deliverable, e audit.Entry) error {
	return nil
}
func (r *fakeRepo) SoftDeleteWithAudit(ctx context.Context, ws, id, by uuid.UUID, e audit.Entry) error {
	return nil
}
func (r *fakeRepo) ListSubmissionsForDeliverable(ctx context.Context, ws, id uuid.UUID) ([]deliverable.SubmissionWithReview, error) {
	return r.listSubs, nil
}
func (r *fakeRepo) CreateSubmissionLocked(ctx context.Context, s deliverable.Submission, e audit.Entry) (*deliverable.Submission, error) {
	if r.submitErr != nil {
		return nil, r.submitErr
	}
	if r.submitResult != nil {
		return r.submitResult, nil
	}
	s.RoundNo = 1
	return &s, nil
}
func (r *fakeRepo) CreateReviewLocked(ctx context.Context, did uuid.UUID, rv deliverable.SubmissionReview, e audit.Entry) (*deliverable.SubmissionReview, error) {
	if r.reviewErr != nil {
		return nil, r.reviewErr
	}
	if r.reviewResult != nil {
		return r.reviewResult, nil
	}
	return &rv, nil
}

type fakeMaster struct{ active bool }

func (m *fakeMaster) IsActiveSubmissionDecisionCode(ctx context.Context, code string) (bool, error) {
	return m.active, nil
}

type fakeAccess struct {
	exists bool
	member *deliverable.ActorMember
}

func (a *fakeAccess) ProjectExistsForWorkspace(ctx context.Context, ws, pid uuid.UUID) (bool, error) {
	return a.exists, nil
}
func (a *fakeAccess) FindActiveMemberByMembership(ctx context.Context, ws, pid, mid uuid.UUID) (*deliverable.ActorMember, error) {
	if a.member != nil {
		cp := *a.member
		return &cp, nil
	}
	return nil, nil
}

// ── test app builder ────────────────────────────────────────────────────────────

func buildApp(tc *workspace.TenantContext, repo *fakeRepo, master *fakeMaster, access *fakeAccess) *fiber.App {
	if repo == nil {
		repo = &fakeRepo{}
	}
	if master == nil {
		master = &fakeMaster{active: true}
	}
	if access == nil {
		access = &fakeAccess{exists: true}
	}
	svc := deliverable.NewService(repo, master, access)
	h := deliverablehandler.NewHandler(svc)

	app := fiber.New()
	withTenant := func(c fiber.Ctx) error {
		if tc != nil {
			c.Locals(workspacehandler.LocalsKeyTenant, tc)
		}
		return c.Next()
	}
	base := "/api/v1/workspaces/projects/:projectID/deliverables"
	app.Post(base, withTenant, h.HandleCreate)
	app.Get(base, withTenant, h.HandleList)
	app.Get(base+"/:deliverableID", withTenant, h.HandleGet)
	app.Put(base+"/:deliverableID", withTenant, h.HandleUpdate)
	app.Delete(base+"/:deliverableID", withTenant, h.HandleDelete)
	app.Post(base+"/:deliverableID/submissions", withTenant, h.HandleSubmit)
	app.Post(base+"/:deliverableID/submissions/:submissionID/review", withTenant, h.HandleReview)
	return app
}

func newTC(orgRole string) *workspace.TenantContext {
	return &workspace.TenantContext{
		WorkspaceID:  uuid.New(),
		MembershipID: uuid.New(),
		AccountID:    uuid.New(),
		OrgRoleCode:  orgRole,
	}
}

func memberOf(role string) *deliverable.ActorMember {
	return &deliverable.ActorMember{ID: uuid.New(), ProjectRoleCode: role}
}

func doJSON(t *testing.T, app *fiber.App, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var buf io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var parsed map[string]any
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &parsed)
	}
	return resp.StatusCode, parsed
}

func errCode(m map[string]any) string {
	if e, ok := m["error"].(map[string]any); ok {
		if c, ok := e["code"].(string); ok {
			return c
		}
	}
	return ""
}

const (
	pid  = "11111111-1111-1111-1111-111111111111"
	did  = "22222222-2222-2222-2222-222222222222"
	subi = "33333333-3333-3333-3333-333333333333"
)

func basePath() string { return "/api/v1/workspaces/projects/" + pid + "/deliverables" }

// ── happy paths ─────────────────────────────────────────────────────────────────

func TestHandleCreate_201(t *testing.T) {
	app := buildApp(newTC(workspace.OrgRoleUser), &fakeRepo{}, nil, &fakeAccess{exists: true, member: memberOf(deliverable.ProjectRoleManager)})
	status, body := doJSON(t, app, "POST", basePath(), map[string]any{"title": "Phase 1"})
	if status != fiber.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%v", status, body)
	}
	data, _ := body["data"].(map[string]any)
	if data["deliverable_status_code"] != deliverable.DeliverableStatusNotSubmitted {
		t.Errorf("status_code = %v, want not_submitted", data["deliverable_status_code"])
	}
}

func TestHandleSubmit_201(t *testing.T) {
	app := buildApp(newTC(workspace.OrgRoleUser), &fakeRepo{}, nil, &fakeAccess{exists: true, member: memberOf("member")})
	status, body := doJSON(t, app, "POST", basePath()+"/"+did+"/submissions", map[string]any{"note": "v1"})
	if status != fiber.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%v", status, body)
	}
	data, _ := body["data"].(map[string]any)
	if data["round_no"].(float64) != 1 {
		t.Errorf("round_no = %v, want 1", data["round_no"])
	}
}

func TestHandleReview_201(t *testing.T) {
	app := buildApp(newTC(workspace.OrgRoleUser), &fakeRepo{}, &fakeMaster{active: true},
		&fakeAccess{exists: true, member: memberOf(deliverable.ProjectRoleOwner)})
	status, body := doJSON(t, app, "POST", basePath()+"/"+did+"/submissions/"+subi+"/review",
		map[string]any{"decision_code": "accepted"})
	if status != fiber.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%v", status, body)
	}
	data, _ := body["data"].(map[string]any)
	if data["decision_code"] != "accepted" {
		t.Errorf("decision_code = %v, want accepted", data["decision_code"])
	}
}

func TestHandleGet_WithSubmissions(t *testing.T) {
	dws := &deliverable.DeliverableWithStatus{
		Deliverable: deliverable.Deliverable{ID: uuid.MustParse(did), ProjectID: uuid.MustParse(pid), Title: "P1"},
	}
	repo := &fakeRepo{
		getResult: dws,
		listSubs: []deliverable.SubmissionWithReview{
			{Submission: deliverable.Submission{ID: uuid.New(), RoundNo: 1}},
		},
	}
	app := buildApp(newTC(workspace.OrgRoleUser), repo, nil, &fakeAccess{exists: true, member: memberOf(deliverable.ProjectRoleViewer)})
	status, body := doJSON(t, app, "GET", basePath()+"/"+did, nil)
	if status != fiber.StatusOK {
		t.Fatalf("status = %d, want 200; body=%v", status, body)
	}
	data, _ := body["data"].(map[string]any)
	subs, _ := data["submissions"].([]any)
	if len(subs) != 1 {
		t.Errorf("submissions len = %d, want 1", len(subs))
	}
}

// ── error mapping ─────────────────────────────────────────────────────────────

func TestHandleCreate_403_PlainMember(t *testing.T) {
	app := buildApp(newTC(workspace.OrgRoleUser), &fakeRepo{}, nil, &fakeAccess{exists: true, member: memberOf("member")})
	status, body := doJSON(t, app, "POST", basePath(), map[string]any{"title": "Phase 1"})
	if status != fiber.StatusForbidden || errCode(body) != "deliverable.forbidden" {
		t.Fatalf("got status=%d code=%q, want 403 deliverable.forbidden", status, errCode(body))
	}
}

func TestHandleList_404_NonMember(t *testing.T) {
	app := buildApp(newTC(workspace.OrgRoleUser), &fakeRepo{}, nil, &fakeAccess{exists: true, member: nil})
	status, body := doJSON(t, app, "GET", basePath(), nil)
	if status != fiber.StatusNotFound || errCode(body) != "project.not_found" {
		t.Fatalf("got status=%d code=%q, want 404 project.not_found (probe-collapse)", status, errCode(body))
	}
}

func TestHandleSubmit_422_OrgAdminNonMember(t *testing.T) {
	app := buildApp(newTC(workspace.OrgRoleAdmin), &fakeRepo{}, nil, &fakeAccess{exists: true, member: nil})
	status, body := doJSON(t, app, "POST", basePath()+"/"+did+"/submissions", map[string]any{})
	if status != fiber.StatusUnprocessableEntity || errCode(body) != "deliverable.not_project_member" {
		t.Fatalf("got status=%d code=%q, want 422 deliverable.not_project_member", status, errCode(body))
	}
}

func TestHandleSubmit_409_AcceptTerminal(t *testing.T) {
	repo := &fakeRepo{submitErr: deliverable.ErrAcceptTerminal}
	app := buildApp(newTC(workspace.OrgRoleUser), repo, nil, &fakeAccess{exists: true, member: memberOf("member")})
	status, body := doJSON(t, app, "POST", basePath()+"/"+did+"/submissions", map[string]any{})
	if status != fiber.StatusConflict || errCode(body) != "deliverable.accept_terminal" {
		t.Fatalf("got status=%d code=%q, want 409 deliverable.accept_terminal", status, errCode(body))
	}
}

func TestHandleReview_409_Superseded(t *testing.T) {
	repo := &fakeRepo{reviewErr: deliverable.ErrSubmissionSuperseded}
	app := buildApp(newTC(workspace.OrgRoleUser), repo, &fakeMaster{active: true},
		&fakeAccess{exists: true, member: memberOf(deliverable.ProjectRoleOwner)})
	status, body := doJSON(t, app, "POST", basePath()+"/"+did+"/submissions/"+subi+"/review",
		map[string]any{"decision_code": "accepted"})
	if status != fiber.StatusConflict || errCode(body) != "submission.superseded" {
		t.Fatalf("got status=%d code=%q, want 409 submission.superseded", status, errCode(body))
	}
}

func TestHandleReview_409_AlreadyReviewed(t *testing.T) {
	repo := &fakeRepo{reviewErr: deliverable.ErrAlreadyReviewed}
	app := buildApp(newTC(workspace.OrgRoleUser), repo, &fakeMaster{active: true},
		&fakeAccess{exists: true, member: memberOf(deliverable.ProjectRoleOwner)})
	status, body := doJSON(t, app, "POST", basePath()+"/"+did+"/submissions/"+subi+"/review",
		map[string]any{"decision_code": "accepted"})
	if status != fiber.StatusConflict || errCode(body) != "submission.already_reviewed" {
		t.Fatalf("got status=%d code=%q, want 409 submission.already_reviewed", status, errCode(body))
	}
}

func TestHandleReview_422_InvalidDecision(t *testing.T) {
	app := buildApp(newTC(workspace.OrgRoleUser), &fakeRepo{}, &fakeMaster{active: false},
		&fakeAccess{exists: true, member: memberOf(deliverable.ProjectRoleOwner)})
	status, body := doJSON(t, app, "POST", basePath()+"/"+did+"/submissions/"+subi+"/review",
		map[string]any{"decision_code": "bogus"})
	if status != fiber.StatusUnprocessableEntity || errCode(body) != "deliverable.invalid_master_code" {
		t.Fatalf("got status=%d code=%q, want 422 deliverable.invalid_master_code", status, errCode(body))
	}
}

func TestHandleReview_422_SelfReviewForbidden(t *testing.T) {
	repo := &fakeRepo{reviewErr: deliverable.ErrSelfReviewForbidden}
	app := buildApp(newTC(workspace.OrgRoleUser), repo, &fakeMaster{active: true},
		&fakeAccess{exists: true, member: memberOf(deliverable.ProjectRoleOwner)})
	status, body := doJSON(t, app, "POST", basePath()+"/"+did+"/submissions/"+subi+"/review",
		map[string]any{"decision_code": "accepted"})
	if status != fiber.StatusUnprocessableEntity || errCode(body) != "submission.self_review_forbidden" {
		t.Fatalf("got status=%d code=%q, want 422 submission.self_review_forbidden", status, errCode(body))
	}
}

func TestHandleReview_404_SubmissionNotFound(t *testing.T) {
	repo := &fakeRepo{reviewErr: deliverable.ErrSubmissionNotFound}
	app := buildApp(newTC(workspace.OrgRoleUser), repo, &fakeMaster{active: true},
		&fakeAccess{exists: true, member: memberOf(deliverable.ProjectRoleOwner)})
	status, body := doJSON(t, app, "POST", basePath()+"/"+did+"/submissions/"+subi+"/review",
		map[string]any{"decision_code": "accepted"})
	if status != fiber.StatusNotFound || errCode(body) != "submission.not_found" {
		t.Fatalf("got status=%d code=%q, want 404 submission.not_found", status, errCode(body))
	}
}

func TestHandleCreate_400_EmptyTitle(t *testing.T) {
	app := buildApp(newTC(workspace.OrgRoleUser), &fakeRepo{}, nil, &fakeAccess{exists: true, member: memberOf(deliverable.ProjectRoleOwner)})
	status, body := doJSON(t, app, "POST", basePath(), map[string]any{"title": ""})
	if status != fiber.StatusBadRequest || errCode(body) != "validation.invalid_input" {
		t.Fatalf("got status=%d code=%q, want 400 validation.invalid_input", status, errCode(body))
	}
}

func TestHandleCreate_400_BadDueDate(t *testing.T) {
	app := buildApp(newTC(workspace.OrgRoleUser), &fakeRepo{}, nil, &fakeAccess{exists: true, member: memberOf(deliverable.ProjectRoleOwner)})
	status, body := doJSON(t, app, "POST", basePath(), map[string]any{"title": "P1", "due_date": "10-06-2026"})
	if status != fiber.StatusBadRequest || errCode(body) != "validation.invalid_input" {
		t.Fatalf("got status=%d code=%q, want 400 validation.invalid_input", status, errCode(body))
	}
}

func TestHandleDelete_404_DeliverableNotFound(t *testing.T) {
	// repo.getResult nil + FindByIDForWorkspace nil → service returns ErrDeliverableNotFound.
	app := buildApp(newTC(workspace.OrgRoleAdmin), &fakeRepo{getResult: nil}, nil, &fakeAccess{exists: true, member: nil})
	status, body := doJSON(t, app, "DELETE", basePath()+"/"+did, nil)
	if status != fiber.StatusNotFound || errCode(body) != "deliverable.not_found" {
		t.Fatalf("got status=%d code=%q, want 404 deliverable.not_found", status, errCode(body))
	}
}

// submitted_by / reviewed_by in the body must be ignored (server-derive). The handler request
// structs have no such field, so JSON with those keys is silently dropped — assert the request
// still succeeds and the response submitted_by is the server-resolved member id, not the body.
func TestHandleSubmit_IgnoresBodySubmittedBy(t *testing.T) {
	m := memberOf("member")
	app := buildApp(newTC(workspace.OrgRoleUser), &fakeRepo{}, nil, &fakeAccess{exists: true, member: m})
	bogus := uuid.New().String()
	status, body := doJSON(t, app, "POST", basePath()+"/"+did+"/submissions",
		map[string]any{"note": "v1", "submitted_by": bogus})
	if status != fiber.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%v", status, body)
	}
	data, _ := body["data"].(map[string]any)
	if data["submitted_by"] == bogus {
		t.Errorf("submitted_by echoed the body value %q — must be server-derived", bogus)
	}
	if data["submitted_by"] != m.ID.String() {
		t.Errorf("submitted_by = %v, want server-derived member id %v", data["submitted_by"], m.ID)
	}
}
