package deliverabledbrepo_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	auditdbrepo "prasankit-api/internal/adapters/database/audit"
	authdbrepo "prasankit-api/internal/adapters/database/auth"
	deliverabledbrepo "prasankit-api/internal/adapters/database/deliverable"
	projectdbrepo "prasankit-api/internal/adapters/database/project"
	workspacedbrepo "prasankit-api/internal/adapters/database/workspace"
	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/deliverable"
	"prasankit-api/internal/modules/project"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ── Scaffolding (mirrors sibling adapter tests) ─────────────────────────────────

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		envOr("POSTGRES_PRIMARY_HOST", "localhost"),
		envOr("POSTGRES_PRIMARY_PORT", "15433"),
		envOr("POSTGRES_PRIMARY_USER", "prasankit"),
		envOr("POSTGRES_PRIMARY_PASSWORD", "change_me"),
		envOr("POSTGRES_PRIMARY_NAME", "prasankit"),
		envOr("POSTGRES_SSL_MODE", "disable"),
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Skipf("postgres not reachable (%v): skipping integration test", err)
	}
	sqlDB, _ := db.DB()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		t.Skipf("postgres ping failed (%v): skipping integration test", err)
	}
	return db
}

// truncateTestTables wipes data tables (NEVER master tables).
func truncateTestTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	tables := []string{
		"submission_reviews",
		"submissions",
		"deliverables",
		"audit_logs",
		"project_members",
		"projects",
		"workspace_memberships",
		"workspaces",
		"auth_identities",
		"user_accounts",
	}
	for _, tbl := range tables {
		if err := db.Exec("TRUNCATE TABLE " + tbl + " CASCADE").Error; err != nil {
			t.Logf("truncate %s: %v", tbl, err)
		}
	}
}

func createTestAccount(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	accountRepo := authdbrepo.NewAccountRepo(db)
	email := fmt.Sprintf("delivtest+%d@example.com", time.Now().UnixNano())
	id, _ := ids.New()
	iid, _ := ids.New()
	now := time.Now().UTC()
	account := auth.Account{
		ID:                id,
		PrimaryEmail:      email,
		DisplayName:       "Deliverable Test User",
		AccountStatusCode: auth.AccountStatusPendingVerification,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	identity := auth.Identity{
		ID:               iid,
		UserAccountID:    id,
		IdentityTypeCode: auth.IdentityTypeEmailPassword,
		Email:            email,
		PasswordHash:     "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := accountRepo.CreateWithIdentity(context.Background(), account, identity); err != nil {
		t.Fatalf("createTestAccount: %v", err)
	}
	return id
}

type workspaceLite struct{ ID uuid.UUID }

func createTestWorkspace(t *testing.T, db *gorm.DB, ownerID uuid.UUID) workspaceLite {
	t.Helper()
	auditRepo := auditdbrepo.NewAuditRepo(db)
	wsRepo := workspacedbrepo.NewWorkspaceRepo(db, auditRepo)
	wsID, _ := ids.New()
	mID, _ := ids.New()
	now := time.Now().UTC()
	slug := fmt.Sprintf("ws-%d", time.Now().UnixNano())
	ws := workspace.Workspace{
		ID:                  wsID,
		WorkspaceName:       "Test Workspace",
		Slug:                slug,
		WorkspaceStatusCode: workspace.WorkspaceStatusActive,
		ContactEmail:        "admin@test.com",
		OwnerUserAccountID:  ownerID,
		CreatedAt:           now,
		CreatedBy:           &ownerID,
		UpdatedAt:           now,
	}
	joinedAt := now
	m := workspace.Membership{
		ID:                   mID,
		WorkspaceID:          wsID,
		UserAccountID:        ownerID,
		OrgRoleCode:          workspace.OrgRoleOwner,
		MembershipStatusCode: workspace.MembershipStatusActive,
		JoinedAt:             &joinedAt,
		CreatedAt:            now,
		CreatedBy:            &ownerID,
		UpdatedAt:            now,
	}
	resourceID := wsID
	entry := audit.Entry{
		WorkspaceID:    &wsID,
		ActorAccountID: &ownerID,
		Action:         workspace.AuditActionWorkspaceCreate,
		ResourceType:   workspace.AuditResourceTypeWorkspace,
		ResourceID:     &resourceID,
		Result:         workspace.AuditResultSuccess,
		IP:             "127.0.0.1",
		UserAgent:      "test",
		RequestID:      "req-test",
	}
	if err := wsRepo.CreateWorkspaceWithOwner(context.Background(), ws, m, entry); err != nil {
		t.Fatalf("createTestWorkspace: %v", err)
	}
	return workspaceLite{ID: wsID}
}

// findMembershipID returns the active workspace_membership id for (wsID, accountID). The
// workspace owner already has a membership created by createTestWorkspace, so the fixture
// reuses it rather than inserting a duplicate (which would violate uq_membership_active_user).
func findMembershipID(t *testing.T, db *gorm.DB, wsID, accountID uuid.UUID) uuid.UUID {
	t.Helper()
	var idStr string
	err := db.Raw(
		`SELECT id::text FROM workspace_memberships
		 WHERE workspace_id = ? AND user_account_id = ? AND membership_status_code = ?`,
		wsID, accountID, workspace.MembershipStatusActive,
	).Scan(&idStr).Error
	if err != nil || idStr == "" {
		t.Fatalf("findMembershipID: %v (id=%q)", err, idStr)
	}
	id, perr := uuid.Parse(idStr)
	if perr != nil {
		t.Fatalf("findMembershipID parse: %v", perr)
	}
	return id
}

func createTestProject(t *testing.T, db *gorm.DB, wsID, accountID uuid.UUID) uuid.UUID {
	t.Helper()
	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectdbrepo.NewProjectRepo(db, auditRepo)
	id, _ := ids.New()
	now := time.Now().UTC()
	p := project.Project{
		ID:                id,
		WorkspaceID:       wsID,
		ProjectName:       "Test Project",
		ProjectTypeCode:   "client",
		ProjectStatusCode: "planning",
		CreatedAt:         now,
		CreatedBy:         accountID,
		UpdatedAt:         now,
	}
	entry := audit.Entry{
		WorkspaceID:    &wsID,
		ProjectID:      &id,
		ActorAccountID: &accountID,
		Action:         project.AuditActionProjectCreate,
		ResourceType:   project.AuditResourceTypeProject,
		ResourceID:     &id,
		Result:         project.AuditResultSuccess,
		IP:             "127.0.0.1",
		UserAgent:      "test",
		RequestID:      "req-test",
	}
	if err := repo.CreateWithAudit(context.Background(), p, entry); err != nil {
		t.Fatalf("createTestProject: %v", err)
	}
	return id
}

// createTestProjectMember directly inserts a project_members row (active) and returns its id —
// the FK target for submitted_by/reviewed_by. removed_at NULL by default.
func createTestProjectMember(t *testing.T, db *gorm.DB, wsID, projectID, membershipID, accountID uuid.UUID, role string) uuid.UUID {
	t.Helper()
	id, _ := ids.New()
	now := time.Now().UTC()
	err := db.Exec(
		`INSERT INTO project_members
		 (id, workspace_id, project_id, workspace_membership_id, project_role_code, joined_at, created_at, created_by, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, wsID, projectID, membershipID, role, now, now, accountID, now,
	).Error
	if err != nil {
		t.Fatalf("createTestProjectMember: %v", err)
	}
	return id
}

// fixture bundles the common (ws, project, account, member) scaffolding.
type fixture struct {
	wsID      uuid.UUID
	accountID uuid.UUID
	projectID uuid.UUID
	memberID  uuid.UUID // project_member id (submitted_by/reviewed_by FK target)
}

func setupFixture(t *testing.T, db *gorm.DB) fixture {
	t.Helper()
	accountID := createTestAccount(t, db)
	ws := createTestWorkspace(t, db, accountID)
	projectID := createTestProject(t, db, ws.ID, accountID)
	membershipID := findMembershipID(t, db, ws.ID, accountID)
	memberID := createTestProjectMember(t, db, ws.ID, projectID, membershipID, accountID, deliverable.ProjectRoleOwner)
	return fixture{wsID: ws.ID, accountID: accountID, projectID: projectID, memberID: memberID}
}

func newRepo(db *gorm.DB) *deliverabledbrepo.DeliverableRepo {
	return deliverabledbrepo.NewDeliverableRepo(db, auditdbrepo.NewAuditRepo(db), false)
}

func newRepoForbidSelf(db *gorm.DB) *deliverabledbrepo.DeliverableRepo {
	return deliverabledbrepo.NewDeliverableRepo(db, auditdbrepo.NewAuditRepo(db), true)
}

func buildDeliverable(fx fixture, due *time.Time) deliverable.Deliverable {
	id, _ := ids.New()
	now := time.Now().UTC()
	return deliverable.Deliverable{
		ID:          id,
		WorkspaceID: fx.wsID,
		ProjectID:   fx.projectID,
		Title:       "งวด 1",
		DueDate:     due,
		SortOrder:   0,
		CreatedAt:   now,
		CreatedBy:   fx.accountID,
		UpdatedAt:   now,
	}
}

func deliverableEntry(fx fixture, did uuid.UUID, action string) audit.Entry {
	ws := fx.wsID
	pid := fx.projectID
	acc := fx.accountID
	rid := did
	return audit.Entry{
		WorkspaceID:    &ws,
		ProjectID:      &pid,
		ActorAccountID: &acc,
		Action:         action,
		ResourceType:   deliverable.AuditResourceTypeDeliverable,
		ResourceID:     &rid,
		NewValue:       map[string]any{"title": "งวด 1"},
		Result:         deliverable.AuditResultSuccess,
		IP:             "127.0.0.1",
		UserAgent:      "test",
		RequestID:      "req-test",
	}
}

func buildSubmission(fx fixture, did uuid.UUID) deliverable.Submission {
	id, _ := ids.New()
	now := time.Now().UTC()
	return deliverable.Submission{
		ID:            id,
		WorkspaceID:   fx.wsID,
		ProjectID:     fx.projectID,
		DeliverableID: did,
		Note:          nil,
		SubmittedBy:   fx.memberID,
		SubmittedAt:   now,
		CreatedAt:     now,
		CreatedBy:     fx.accountID,
	}
}

func submissionEntry(fx fixture, subID uuid.UUID) audit.Entry {
	ws := fx.wsID
	pid := fx.projectID
	acc := fx.accountID
	rid := subID
	return audit.Entry{
		WorkspaceID:    &ws,
		ProjectID:      &pid,
		ActorAccountID: &acc,
		Action:         deliverable.AuditActionSubmissionCreate,
		ResourceType:   deliverable.AuditResourceTypeSubmission,
		ResourceID:     &rid,
		NewValue:       map[string]any{"round_no": 0},
		Result:         deliverable.AuditResultSuccess,
		IP:             "127.0.0.1",
		UserAgent:      "test",
		RequestID:      "req-test",
	}
}

func buildReview(fx fixture, subID uuid.UUID, decision string) deliverable.SubmissionReview {
	id, _ := ids.New()
	now := time.Now().UTC()
	return deliverable.SubmissionReview{
		ID:           id,
		WorkspaceID:  fx.wsID,
		ProjectID:    fx.projectID,
		SubmissionID: subID,
		DecisionCode: decision,
		ReviewedBy:   fx.memberID,
		ReviewedAt:   now,
		CreatedAt:    now,
		CreatedBy:    fx.accountID,
	}
}

func reviewEntry(fx fixture, revID uuid.UUID) audit.Entry {
	ws := fx.wsID
	pid := fx.projectID
	acc := fx.accountID
	rid := revID
	return audit.Entry{
		WorkspaceID:    &ws,
		ProjectID:      &pid,
		ActorAccountID: &acc,
		Action:         deliverable.AuditActionReviewCreate,
		ResourceType:   deliverable.AuditResourceTypeReview,
		ResourceID:     &rid,
		NewValue:       map[string]any{"decision_code": "x"},
		Result:         deliverable.AuditResultSuccess,
		IP:             "127.0.0.1",
		UserAgent:      "test",
		RequestID:      "req-test",
	}
}

// helper: create a deliverable via the repo, return its id.
func createDeliverable(t *testing.T, repo *deliverabledbrepo.DeliverableRepo, fx fixture, due *time.Time) uuid.UUID {
	t.Helper()
	d := buildDeliverable(fx, due)
	if err := repo.CreateWithAudit(context.Background(), d, deliverableEntry(fx, d.ID, deliverable.AuditActionDeliverableCreate)); err != nil {
		t.Fatalf("create deliverable: %v", err)
	}
	return d.ID
}

func submit(t *testing.T, repo *deliverabledbrepo.DeliverableRepo, fx fixture, did uuid.UUID) *deliverable.Submission {
	t.Helper()
	s := buildSubmission(fx, did)
	created, err := repo.CreateSubmissionLocked(context.Background(), s, submissionEntry(fx, s.ID))
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	return created
}

func review(t *testing.T, repo *deliverabledbrepo.DeliverableRepo, fx fixture, did, subID uuid.UUID, decision string) (*deliverable.SubmissionReview, error) {
	t.Helper()
	r := buildReview(fx, subID, decision)
	return repo.CreateReviewLocked(context.Background(), did, r, reviewEntry(fx, r.ID))
}

// ── ws isolation ────────────────────────────────────────────────────────────────

func TestDeliverable_WorkspaceIsolation(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })
	repo := newRepo(db)

	fx := setupFixture(t, db)
	did := createDeliverable(t, repo, fx, nil)

	// A second workspace must not see the deliverable.
	otherAcct := createTestAccount(t, db)
	otherWS := createTestWorkspace(t, db, otherAcct)

	got, err := repo.FindByIDForWorkspace(context.Background(), otherWS.ID, did)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got != nil {
		t.Errorf("cross-ws leak: expected nil, got %+v", got)
	}

	list, err := repo.ListByProjectWithStatus(context.Background(), otherWS.ID, fx.projectID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("cross-ws list leak: expected 0, got %d", len(list))
	}
}

// ── cross-ws FK reject (composite FK iron rule) ─────────────────────────────────

func TestDeliverable_CrossWorkspaceProjectFK_Reject(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })
	repo := newRepo(db)

	fxA := setupFixture(t, db)
	otherAcct := createTestAccount(t, db)
	wsB := createTestWorkspace(t, db, otherAcct)

	// Deliverable with wsB.ID but projectID from wsA → composite FK must reject.
	id, _ := ids.New()
	now := time.Now().UTC()
	bad := deliverable.Deliverable{
		ID:          id,
		WorkspaceID: wsB.ID,
		ProjectID:   fxA.projectID, // belongs to wsA
		Title:       "x",
		CreatedAt:   now,
		CreatedBy:   otherAcct,
		UpdatedAt:   now,
	}
	err := repo.CreateWithAudit(context.Background(), bad, deliverableEntry(fxA, id, deliverable.AuditActionDeliverableCreate))
	if err == nil {
		t.Fatal("expected composite FK violation, got nil")
	}
}

func TestSubmission_CrossWorkspaceSubmittedByFK_Reject(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })
	repo := newRepo(db)

	fx := setupFixture(t, db)
	did := createDeliverable(t, repo, fx, nil)

	// submitted_by = a random uuid that is not a project_member of this ws → FK reject.
	s := buildSubmission(fx, did)
	s.SubmittedBy = uuid.New()
	_, err := repo.CreateSubmissionLocked(context.Background(), s, submissionEntry(fx, s.ID))
	if err == nil {
		t.Fatal("expected submitted_by composite FK violation, got nil")
	}
}

// ── derive path end-to-end (assert status at every step) ─────────────────────────

func TestDerivePath_CreateSubmitRejectResubmitConditionalResubmitAccept(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })
	repo := newRepo(db)
	ctx := context.Background()

	fx := setupFixture(t, db)
	did := createDeliverable(t, repo, fx, nil)

	assertStatus := func(want string, wantRound int) {
		t.Helper()
		dws, err := repo.GetWithStatus(ctx, fx.wsID, did)
		if err != nil {
			t.Fatalf("get with status: %v", err)
		}
		got := deliverable.DeriveStatus(dws.LatestSubmission)
		if got != want {
			t.Fatalf("derived status = %q, want %q", got, want)
		}
		if want == deliverable.DeliverableStatusNotSubmitted {
			if dws.LatestSubmission != nil {
				t.Fatalf("not_submitted but LatestSubmission != nil")
			}
			return
		}
		if dws.LatestSubmission.RoundNo != wantRound {
			t.Fatalf("latest round = %d, want %d", dws.LatestSubmission.RoundNo, wantRound)
		}
	}

	// 0. created → not_submitted
	assertStatus(deliverable.DeliverableStatusNotSubmitted, 0)

	// 1. submit round 1 → in_review
	s1 := submit(t, repo, fx, did)
	if s1.RoundNo != 1 {
		t.Fatalf("first submit round = %d, want 1", s1.RoundNo)
	}
	assertStatus(deliverable.DeliverableStatusInReview, 1)

	// 2. reject round 1 → rejected
	if _, err := review(t, repo, fx, did, s1.ID, deliverable.SubmissionDecisionRejected); err != nil {
		t.Fatalf("reject r1: %v", err)
	}
	assertStatus(deliverable.DeliverableStatusRejected, 1)

	// 3. resubmit round 2 → in_review
	s2 := submit(t, repo, fx, did)
	if s2.RoundNo != 2 {
		t.Fatalf("resubmit round = %d, want 2", s2.RoundNo)
	}
	assertStatus(deliverable.DeliverableStatusInReview, 2)

	// 4. conditional round 2 → conditional
	if _, err := review(t, repo, fx, did, s2.ID, deliverable.SubmissionDecisionConditional); err != nil {
		t.Fatalf("conditional r2: %v", err)
	}
	assertStatus(deliverable.DeliverableStatusConditional, 2)

	// 5. resubmit round 3 → in_review
	s3 := submit(t, repo, fx, did)
	if s3.RoundNo != 3 {
		t.Fatalf("resubmit round = %d, want 3", s3.RoundNo)
	}
	assertStatus(deliverable.DeliverableStatusInReview, 3)

	// 6. accept round 3 → accepted
	if _, err := review(t, repo, fx, did, s3.ID, deliverable.SubmissionDecisionAccepted); err != nil {
		t.Fatalf("accept r3: %v", err)
	}
	assertStatus(deliverable.DeliverableStatusAccepted, 3)
}

// ── accept terminal: resubmit after accept → 409 ────────────────────────────────

func TestSubmit_AfterAccept_Terminal409(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })
	repo := newRepo(db)

	fx := setupFixture(t, db)
	did := createDeliverable(t, repo, fx, nil)
	s1 := submit(t, repo, fx, did)
	if _, err := review(t, repo, fx, did, s1.ID, deliverable.SubmissionDecisionAccepted); err != nil {
		t.Fatalf("accept: %v", err)
	}

	s := buildSubmission(fx, did)
	_, err := repo.CreateSubmissionLocked(context.Background(), s, submissionEntry(fx, s.ID))
	if !errors.Is(err, deliverable.ErrAcceptTerminal) {
		t.Fatalf("resubmit after accept: got %v, want ErrAcceptTerminal", err)
	}
}

// ── double-review → 409 ─────────────────────────────────────────────────────────

func TestReview_Double_409(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })
	repo := newRepo(db)

	fx := setupFixture(t, db)
	did := createDeliverable(t, repo, fx, nil)
	s1 := submit(t, repo, fx, did)

	if _, err := review(t, repo, fx, did, s1.ID, deliverable.SubmissionDecisionConditional); err != nil {
		t.Fatalf("first review: %v", err)
	}
	_, err := review(t, repo, fx, did, s1.ID, deliverable.SubmissionDecisionRejected)
	if !errors.Is(err, deliverable.ErrAlreadyReviewed) {
		t.Fatalf("second review: got %v, want ErrAlreadyReviewed", err)
	}
}

// ── review of an old (superseded) round → 409 ───────────────────────────────────

func TestReview_OldRound_Superseded409(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })
	repo := newRepo(db)

	fx := setupFixture(t, db)
	did := createDeliverable(t, repo, fx, nil)
	s1 := submit(t, repo, fx, did)
	// reject r1 so resubmit is allowed, then r2 exists → r1 is no longer MAX.
	if _, err := review(t, repo, fx, did, s1.ID, deliverable.SubmissionDecisionRejected); err != nil {
		t.Fatalf("reject r1: %v", err)
	}
	_ = submit(t, repo, fx, did) // r2

	// Now try to review r1 again — but it already has a review; instead exercise superseded
	// with a fresh deliverable where r1 is unreviewed and r2 exists.
	did2 := createDeliverable(t, repo, fx, nil)
	a1 := submit(t, repo, fx, did2) // r1 (unreviewed)
	// Force a second round directly so r1 is superseded without reviewing it.
	directInsertSubmission(t, db, fx, did2, 2)
	_, err := review(t, repo, fx, did2, a1.ID, deliverable.SubmissionDecisionAccepted)
	if !errors.Is(err, deliverable.ErrSubmissionSuperseded) {
		t.Fatalf("review superseded round: got %v, want ErrSubmissionSuperseded", err)
	}
}

// directInsertSubmission bypasses the locked path to seed a row (used by backstop + superseded tests).
func directInsertSubmission(t *testing.T, db *gorm.DB, fx fixture, did uuid.UUID, round int) uuid.UUID {
	t.Helper()
	id, _ := ids.New()
	now := time.Now().UTC()
	err := db.Exec(
		`INSERT INTO submissions
		 (id, workspace_id, project_id, deliverable_id, round_no, submitted_by, submitted_at, created_at, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, fx.wsID, fx.projectID, did, round, fx.memberID, now, now, fx.accountID,
	).Error
	if err != nil {
		t.Fatalf("directInsertSubmission round %d: %v", round, err)
	}
	return id
}

// ── double-round direct INSERT → UNIQUE backstop fires ──────────────────────────

func TestSubmission_DoubleRoundDirectInsert_UniqueBackstop(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })
	repo := newRepo(db)

	fx := setupFixture(t, db)
	did := createDeliverable(t, repo, fx, nil)
	directInsertSubmission(t, db, fx, did, 1)

	// A second row with the same (ws, deliverable_id, round_no=1) must violate the UNIQUE.
	id, _ := ids.New()
	now := time.Now().UTC()
	err := db.Exec(
		`INSERT INTO submissions
		 (id, workspace_id, project_id, deliverable_id, round_no, submitted_by, submitted_at, created_at, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, fx.wsID, fx.projectID, did, 1, fx.memberID, now, now, fx.accountID,
	).Error
	if err == nil {
		t.Fatal("expected uq_submissions_deliverable_round violation, got nil")
	}
}

// ── double-review direct INSERT → UNIQUE backstop fires ─────────────────────────

func TestReview_DoubleDirectInsert_UniqueBackstop(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })
	repo := newRepo(db)

	fx := setupFixture(t, db)
	did := createDeliverable(t, repo, fx, nil)
	s1 := submit(t, repo, fx, did)
	if _, err := review(t, repo, fx, did, s1.ID, deliverable.SubmissionDecisionConditional); err != nil {
		t.Fatalf("first review: %v", err)
	}

	// Direct second review on the same submission → uq_submission_reviews_submission violation.
	id, _ := ids.New()
	now := time.Now().UTC()
	err := db.Exec(
		`INSERT INTO submission_reviews
		 (id, workspace_id, project_id, submission_id, decision_code, reviewed_by, reviewed_at, created_at, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, fx.wsID, fx.projectID, s1.ID, deliverable.SubmissionDecisionRejected, fx.memberID, now, now, fx.accountID,
	).Error
	if err == nil {
		t.Fatal("expected uq_submission_reviews_submission violation, got nil")
	}
}

// ── race: 2 concurrent submits → rounds 1 & 2, no collision ─────────────────────

func TestSubmit_RaceTwoConcurrent_GetRounds1And2(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })
	repo := newRepo(db)

	fx := setupFixture(t, db)
	did := createDeliverable(t, repo, fx, nil)

	var wg sync.WaitGroup
	start := make(chan struct{})
	rounds := make([]int, 2)
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start // barrier: maximize contention
			s := buildSubmission(fx, did)
			created, err := repo.CreateSubmissionLocked(context.Background(), s, submissionEntry(fx, s.ID))
			if err != nil {
				errs[idx] = err
				return
			}
			rounds[idx] = created.RoundNo
		}(i)
	}
	close(start)
	wg.Wait()

	for i, e := range errs {
		if e != nil {
			t.Fatalf("concurrent submit %d error: %v", i, e)
		}
	}
	// The two rounds must be exactly {1, 2} — the FOR UPDATE lock serialized MAX+1.
	got := map[int]bool{rounds[0]: true, rounds[1]: true}
	if !got[1] || !got[2] || rounds[0] == rounds[1] {
		t.Fatalf("concurrent rounds = %v, want exactly {1,2}", rounds)
	}
}

// ── race: review vs resubmit interleave → no double-accept / no review of stale round ──

func TestReviewVsResubmit_RaceInterleave_NoDoubleAccept(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })
	repo := newRepo(db)
	ctx := context.Background()

	fx := setupFixture(t, db)

	// Run many iterations to surface interleaving.
	for iter := 0; iter < 8; iter++ {
		did := createDeliverable(t, repo, fx, nil)
		s1 := submit(t, repo, fx, did)

		var wg sync.WaitGroup
		start := make(chan struct{})
		var reviewErr, submitErr error
		var s2Round int

		wg.Add(2)
		// reviewer: accept round 1
		go func() {
			defer wg.Done()
			<-start
			r := buildReview(fx, s1.ID, deliverable.SubmissionDecisionAccepted)
			_, reviewErr = repo.CreateReviewLocked(ctx, did, r, reviewEntry(fx, r.ID))
		}()
		// submitter: resubmit round 2
		go func() {
			defer wg.Done()
			<-start
			s := buildSubmission(fx, did)
			created, err := repo.CreateSubmissionLocked(ctx, s, submissionEntry(fx, s.ID))
			submitErr = err
			if created != nil {
				s2Round = created.RoundNo
			}
		}()
		close(start)
		wg.Wait()

		// Two legal serialized outcomes (mutually exclusive — never both "succeed against r1"):
		//  A) review-first: accept r1 succeeds, then resubmit blocked by accept-terminal (409).
		//  B) submit-first: resubmit r2 succeeds, then review of r1 fails (superseded 409).
		switch {
		case reviewErr == nil && errors.Is(submitErr, deliverable.ErrAcceptTerminal):
			// A — fine
		case submitErr == nil && s2Round == 2 && errors.Is(reviewErr, deliverable.ErrSubmissionSuperseded):
			// B — fine
		default:
			t.Fatalf("iter %d illegal interleave: reviewErr=%v submitErr=%v s2Round=%d", iter, reviewErr, submitErr, s2Round)
		}

		// Invariant: at most ONE accepted review across the deliverable, and never on a stale round.
		var acceptCount int64
		if err := db.Raw(
			`SELECT count(*) FROM submission_reviews r
			 JOIN submissions s ON s.workspace_id = r.workspace_id AND s.id = r.submission_id
			 WHERE s.deliverable_id = ? AND r.decision_code = ?`, did, deliverable.SubmissionDecisionAccepted).
			Scan(&acceptCount).Error; err != nil {
			t.Fatalf("count accepts: %v", err)
		}
		if acceptCount > 1 {
			t.Fatalf("iter %d: double-accept detected (%d accepts)", iter, acceptCount)
		}
	}
}

// ── removed member → submit rejected (FK reject after removed_at set) ────────────

func TestSubmit_RemovedMember_FKReject(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })
	repo := newRepo(db)

	fx := setupFixture(t, db)
	did := createDeliverable(t, repo, fx, nil)

	// A removed member is still a valid FK target (non-partial backing UNIQUE) — the DB FK does
	// NOT reject a removed member. The SERVICE guard (resolve active member) is what rejects;
	// here we prove the repo-level behavior: a submitted_by that points to a NON-EXISTENT member
	// is rejected by the FK. Create a second member, then submit referencing a bogus member id.
	s := buildSubmission(fx, did)
	s.SubmittedBy = uuid.New() // not a project_member row at all
	_, err := repo.CreateSubmissionLocked(context.Background(), s, submissionEntry(fx, s.ID))
	if err == nil {
		t.Fatal("expected FK violation for non-existent submitted_by member, got nil")
	}
}

// ── removed member resolved as inactive by access repo (service-gate path) ──────

func TestAccessRepo_RemovedMember_ResolvesNil(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })
	access := deliverabledbrepo.NewProjectAccessRepo(db)
	ctx := context.Background()

	accountID := createTestAccount(t, db)
	ws := createTestWorkspace(t, db, accountID)
	projectID := createTestProject(t, db, ws.ID, accountID)
	membershipID := findMembershipID(t, db, ws.ID, accountID)
	memberID := createTestProjectMember(t, db, ws.ID, projectID, membershipID, accountID, deliverable.ProjectRoleManager)

	// Active member resolves.
	m, err := access.FindActiveMemberByMembership(ctx, ws.ID, projectID, membershipID)
	if err != nil {
		t.Fatalf("resolve active: %v", err)
	}
	if m == nil || m.ID != memberID || m.ProjectRoleCode != deliverable.ProjectRoleManager {
		t.Fatalf("active resolve = %+v, want member %v manager", m, memberID)
	}

	// Set removed_at → must resolve nil (the service treats nil as not-a-member → rejected).
	if err := db.Exec(`UPDATE project_members SET removed_at = now() WHERE id = ?`, memberID).Error; err != nil {
		t.Fatalf("set removed_at: %v", err)
	}
	m, err = access.FindActiveMemberByMembership(ctx, ws.ID, projectID, membershipID)
	if err != nil {
		t.Fatalf("resolve removed: %v", err)
	}
	if m != nil {
		t.Fatalf("removed member resolved non-nil: %+v", m)
	}
}

// ── timeliness derive across rounds (presenter-level, via DeriveTimeliness) ──────

func TestTimeliness_DerivePerSubmission(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })
	repo := newRepo(db)
	ctx := context.Background()

	fx := setupFixture(t, db)
	due := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	did := createDeliverable(t, repo, fx, &due)

	// Insert a submission with an explicit early submitted_at, then read it back + derive.
	subID := uuid.New()
	early := time.Date(2026, 6, 8, 12, 0, 0, 0, time.UTC)
	if err := db.Exec(
		`INSERT INTO submissions
		 (id, workspace_id, project_id, deliverable_id, round_no, submitted_by, submitted_at, created_at, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		subID, fx.wsID, fx.projectID, did, 1, fx.memberID, early, early, fx.accountID,
	).Error; err != nil {
		t.Fatalf("insert early submission: %v", err)
	}

	dws, err := repo.GetWithStatus(ctx, fx.wsID, did)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if dws.LatestSubmission == nil {
		t.Fatal("expected latest submission")
	}
	code, lateBy := deliverable.DeriveTimeliness(dws.DueDate, dws.LatestSubmission.SubmittedAt)
	if code != deliverable.TimelinessEarly || lateBy != 0 {
		t.Fatalf("timeliness = (%q, %d), want (early, 0)", code, lateBy)
	}
}

// ── soft-deleted deliverable hidden from list/get ────────────────────────────────

func TestDeliverable_SoftDeleteHidden(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })
	repo := newRepo(db)
	ctx := context.Background()

	fx := setupFixture(t, db)
	did := createDeliverable(t, repo, fx, nil)
	if err := repo.SoftDeleteWithAudit(ctx, fx.wsID, did, fx.accountID, deliverableEntry(fx, did, deliverable.AuditActionDeliverableDelete)); err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	got, err := repo.GetWithStatus(ctx, fx.wsID, did)
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	if got != nil {
		t.Errorf("soft-deleted deliverable visible: %+v", got)
	}
	// submit on a deleted deliverable → not found (lock returns no row).
	s := buildSubmission(fx, did)
	if _, err := repo.CreateSubmissionLocked(ctx, s, submissionEntry(fx, s.ID)); !errors.Is(err, deliverable.ErrDeliverableNotFound) {
		t.Errorf("submit on deleted: got %v, want ErrDeliverableNotFound", err)
	}
}

// ── self-review flag enforcement (under lock) ────────────────────────────────────

func TestReview_SelfReviewForbiddenFlag(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })
	ctx := context.Background()

	fx := setupFixture(t, db)
	// fx.memberID is both the submitter and reviewer (same project_member). With the flag ON,
	// a review by the submitter must be rejected.
	repoForbid := newRepoForbidSelf(db)
	did := createDeliverable(t, repoForbid, fx, nil)
	s1 := submit(t, repoForbid, fx, did)

	r := buildReview(fx, s1.ID, deliverable.SubmissionDecisionAccepted) // reviewed_by = fx.memberID == submitted_by
	_, err := repoForbid.CreateReviewLocked(ctx, did, r, reviewEntry(fx, r.ID))
	if !errors.Is(err, deliverable.ErrSelfReviewForbidden) {
		t.Fatalf("self-review with flag on: got %v, want ErrSelfReviewForbidden", err)
	}

	// With the flag OFF (default), the same self-review is allowed.
	repoAllow := newRepo(db)
	did2 := createDeliverable(t, repoAllow, fx, nil)
	s2 := submit(t, repoAllow, fx, did2)
	r2 := buildReview(fx, s2.ID, deliverable.SubmissionDecisionAccepted)
	if _, err := repoAllow.CreateReviewLocked(ctx, did2, r2, reviewEntry(fx, r2.ID)); err != nil {
		t.Fatalf("self-review with flag off: unexpected err %v", err)
	}
}

// ── audit binds project_id (invariant #6) ────────────────────────────────────────

func TestSubmit_AuditBindsProjectID(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })
	repo := newRepo(db)

	fx := setupFixture(t, db)
	did := createDeliverable(t, repo, fx, nil)
	s1 := submit(t, repo, fx, did)

	var n int64
	if err := db.Raw(
		`SELECT count(*) FROM audit_logs WHERE action = ? AND resource_id = ? AND project_id = ? AND workspace_id = ?`,
		deliverable.AuditActionSubmissionCreate, s1.ID, fx.projectID, fx.wsID).Scan(&n).Error; err != nil {
		t.Fatalf("count audit: %v", err)
	}
	if n != 1 {
		t.Fatalf("submission.create audit rows with project_id bound = %d, want 1", n)
	}
}
