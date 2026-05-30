package projectdbrepo_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	auditdbrepo "prasankit-api/internal/adapters/database/audit"
	authdbrepo "prasankit-api/internal/adapters/database/auth"
	projectdbrepo "prasankit-api/internal/adapters/database/project"
	workspacedbrepo "prasankit-api/internal/adapters/database/workspace"
	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/project"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ── Scaffolding ───────────────────────────────────────────────────────────────

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

// truncateTestTables wipes data tables (NEVER project master tables).
func truncateTestTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	tables := []string{
		"audit_logs",
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

	email := fmt.Sprintf("projtest+%d@example.com", time.Now().UnixNano())
	id, _ := ids.New()
	iid, _ := ids.New()
	now := time.Now().UTC()
	account := auth.Account{
		ID:                id,
		PrimaryEmail:      email,
		DisplayName:       "Project Test User",
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

type workspaceLite struct {
	ID   uuid.UUID
	Slug string
}

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
	return workspaceLite{ID: wsID, Slug: slug}
}

func buildTestProject(t *testing.T, wsID, ownerID uuid.UUID, slug *string, statusCode string) project.Project {
	t.Helper()
	id, err := ids.New()
	if err != nil {
		t.Fatalf("ids.New: %v", err)
	}
	now := time.Now().UTC()
	return project.Project{
		ID:                id,
		WorkspaceID:       wsID,
		ProjectName:       "Test Project",
		Slug:              slug,
		ProjectTypeCode:   "client",
		ProjectStatusCode: statusCode,
		CreatedAt:         now,
		CreatedBy:         ownerID,
		UpdatedAt:         now,
	}
}

func buildProjectAuditEntry(wsID, accountID, projectID uuid.UUID, action string) audit.Entry {
	return audit.Entry{
		WorkspaceID:    &wsID,
		ProjectID:      &projectID,
		ActorAccountID: &accountID,
		Action:         action,
		ResourceType:   project.AuditResourceTypeProject,
		ResourceID:     &projectID,
		Result:         project.AuditResultSuccess,
		IP:             "127.0.0.1",
		UserAgent:      "test",
		RequestID:      "req-test",
	}
}

func strPtr(s string) *string { return &s }

// ── Isolation ─────────────────────────────────────────────────────────────────

func TestProjectRepo_FindByIDForWorkspace_IsolationCrossWS(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectdbrepo.NewProjectRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)
	wsB := createTestWorkspace(t, db, accountID)

	p := buildTestProject(t, wsA.ID, accountID, strPtr("alpha"), "planning")
	if err := repo.CreateWithAudit(context.Background(), p, buildProjectAuditEntry(wsA.ID, accountID, p.ID, project.AuditActionProjectCreate)); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.FindByIDForWorkspace(context.Background(), wsB.ID, p.ID)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got != nil {
		t.Errorf("cross-WS leak: expected nil, got %+v", got)
	}
}

func TestProjectRepo_ListByWorkspace_IsolationCrossWS(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectdbrepo.NewProjectRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)
	wsB := createTestWorkspace(t, db, accountID)

	pA := buildTestProject(t, wsA.ID, accountID, strPtr("a-one"), "planning")
	pB := buildTestProject(t, wsB.ID, accountID, strPtr("b-one"), "planning")
	if err := repo.CreateWithAudit(context.Background(), pA, buildProjectAuditEntry(wsA.ID, accountID, pA.ID, project.AuditActionProjectCreate)); err != nil {
		t.Fatalf("create A: %v", err)
	}
	if err := repo.CreateWithAudit(context.Background(), pB, buildProjectAuditEntry(wsB.ID, accountID, pB.ID, project.AuditActionProjectCreate)); err != nil {
		t.Fatalf("create B: %v", err)
	}

	rows, total, err := repo.ListByWorkspace(context.Background(), wsB.ID, project.ListOptions{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	for _, r := range rows {
		if r.WorkspaceID != wsB.ID {
			t.Errorf("cross-WS leak: row WorkspaceID = %v, want %v", r.WorkspaceID, wsB.ID)
		}
	}
}

func TestProjectRepo_Update_BoundsToWorkspace(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectdbrepo.NewProjectRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)
	wsB := createTestWorkspace(t, db, accountID)

	p := buildTestProject(t, wsA.ID, accountID, strPtr("u-bound"), "planning")
	if err := repo.CreateWithAudit(context.Background(), p, buildProjectAuditEntry(wsA.ID, accountID, p.ID, project.AuditActionProjectCreate)); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Update with WRONG workspace_id.
	bogus := p
	bogus.WorkspaceID = wsB.ID
	bogus.ProjectName = "Renamed"
	bogus.UpdatedAt = time.Now().UTC()
	err := repo.UpdateWithAudit(context.Background(), bogus, buildProjectAuditEntry(wsB.ID, accountID, p.ID, project.AuditActionProjectUpdate))
	if !errors.Is(err, project.ErrProjectNotFound) {
		t.Errorf("err = %v, want ErrProjectNotFound", err)
	}
}

func TestProjectRepo_SoftDelete_BoundsToWorkspace(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectdbrepo.NewProjectRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)
	wsB := createTestWorkspace(t, db, accountID)

	p := buildTestProject(t, wsA.ID, accountID, strPtr("d-bound"), "planning")
	if err := repo.CreateWithAudit(context.Background(), p, buildProjectAuditEntry(wsA.ID, accountID, p.ID, project.AuditActionProjectCreate)); err != nil {
		t.Fatalf("create: %v", err)
	}

	err := repo.SoftDeleteWithAudit(context.Background(), wsB.ID, p.ID, accountID, buildProjectAuditEntry(wsB.ID, accountID, p.ID, project.AuditActionProjectDelete))
	if !errors.Is(err, project.ErrProjectNotFound) {
		t.Errorf("err = %v, want ErrProjectNotFound", err)
	}
}

func TestProjectRepo_ChangeStatus_BoundsToWorkspace(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectdbrepo.NewProjectRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)
	wsB := createTestWorkspace(t, db, accountID)

	p := buildTestProject(t, wsA.ID, accountID, strPtr("s-bound"), "planning")
	if err := repo.CreateWithAudit(context.Background(), p, buildProjectAuditEntry(wsA.ID, accountID, p.ID, project.AuditActionProjectCreate)); err != nil {
		t.Fatalf("create: %v", err)
	}

	err := repo.ChangeStatusWithAudit(context.Background(), wsB.ID, p.ID, "active", accountID, buildProjectAuditEntry(wsB.ID, accountID, p.ID, project.AuditActionProjectStatusChange))
	if !errors.Is(err, project.ErrProjectNotFound) {
		t.Errorf("err = %v, want ErrProjectNotFound", err)
	}
}

// ── Slug uniqueness ───────────────────────────────────────────────────────────

func TestProjectRepo_Create_DuplicateSlug_PerWorkspace(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectdbrepo.NewProjectRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	ws := createTestWorkspace(t, db, accountID)

	slug := strPtr("dup-slug")
	p1 := buildTestProject(t, ws.ID, accountID, slug, "planning")
	if err := repo.CreateWithAudit(context.Background(), p1, buildProjectAuditEntry(ws.ID, accountID, p1.ID, project.AuditActionProjectCreate)); err != nil {
		t.Fatalf("first create: %v", err)
	}

	p2 := buildTestProject(t, ws.ID, accountID, slug, "planning")
	err := repo.CreateWithAudit(context.Background(), p2, buildProjectAuditEntry(ws.ID, accountID, p2.ID, project.AuditActionProjectCreate))
	if !errors.Is(err, project.ErrSlugTaken) {
		t.Errorf("err = %v, want ErrSlugTaken", err)
	}
}

func TestProjectRepo_Create_SameSlug_DifferentWorkspaces_OK(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectdbrepo.NewProjectRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)
	wsB := createTestWorkspace(t, db, accountID)

	slug := strPtr("shared-slug")
	pA := buildTestProject(t, wsA.ID, accountID, slug, "planning")
	if err := repo.CreateWithAudit(context.Background(), pA, buildProjectAuditEntry(wsA.ID, accountID, pA.ID, project.AuditActionProjectCreate)); err != nil {
		t.Fatalf("wsA create: %v", err)
	}
	pB := buildTestProject(t, wsB.ID, accountID, slug, "planning")
	if err := repo.CreateWithAudit(context.Background(), pB, buildProjectAuditEntry(wsB.ID, accountID, pB.ID, project.AuditActionProjectCreate)); err != nil {
		t.Errorf("wsB create: %v", err)
	}
}

func TestProjectRepo_Create_NullSlug_NoConflict(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectdbrepo.NewProjectRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	ws := createTestWorkspace(t, db, accountID)

	p1 := buildTestProject(t, ws.ID, accountID, nil, "planning")
	if err := repo.CreateWithAudit(context.Background(), p1, buildProjectAuditEntry(ws.ID, accountID, p1.ID, project.AuditActionProjectCreate)); err != nil {
		t.Fatalf("first nil-slug create: %v", err)
	}
	p2 := buildTestProject(t, ws.ID, accountID, nil, "planning")
	if err := repo.CreateWithAudit(context.Background(), p2, buildProjectAuditEntry(ws.ID, accountID, p2.ID, project.AuditActionProjectCreate)); err != nil {
		t.Errorf("second nil-slug create: %v", err)
	}
}

func TestProjectRepo_Create_SoftDeletedSlugRecreate_OK(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectdbrepo.NewProjectRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	ws := createTestWorkspace(t, db, accountID)

	slug := strPtr("reuse-slug")
	p1 := buildTestProject(t, ws.ID, accountID, slug, "planning")
	if err := repo.CreateWithAudit(context.Background(), p1, buildProjectAuditEntry(ws.ID, accountID, p1.ID, project.AuditActionProjectCreate)); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if err := repo.SoftDeleteWithAudit(context.Background(), ws.ID, p1.ID, accountID, buildProjectAuditEntry(ws.ID, accountID, p1.ID, project.AuditActionProjectDelete)); err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	p2 := buildTestProject(t, ws.ID, accountID, slug, "planning")
	if err := repo.CreateWithAudit(context.Background(), p2, buildProjectAuditEntry(ws.ID, accountID, p2.ID, project.AuditActionProjectCreate)); err != nil {
		t.Errorf("recreate after delete: %v", err)
	}
}

// ── Audit D31 + D43 ──────────────────────────────────────────────────────────

func TestProjectRepo_Audit_WrittenInTxAtomically(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectdbrepo.NewProjectRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	ws := createTestWorkspace(t, db, accountID)

	p := buildTestProject(t, ws.ID, accountID, strPtr("audit-test"), "planning")
	if err := repo.CreateWithAudit(context.Background(), p, buildProjectAuditEntry(ws.ID, accountID, p.ID, project.AuditActionProjectCreate)); err != nil {
		t.Fatalf("create: %v", err)
	}

	var count int64
	if err := db.Table("audit_logs").
		Where("workspace_id = ? AND project_id = ? AND action = ?", ws.ID, p.ID, project.AuditActionProjectCreate).
		Count(&count).Error; err != nil {
		t.Fatalf("count audit_logs: %v", err)
	}
	if count != 1 {
		t.Errorf("audit_logs count = %d, want 1", count)
	}
}

func TestAuditRepo_ListByProject_ReadInvariantD43(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectdbrepo.NewProjectRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)
	wsB := createTestWorkspace(t, db, accountID)

	// Project P_B in ws B.
	pB := buildTestProject(t, wsB.ID, accountID, strPtr("p-b"), "planning")
	if err := repo.CreateWithAudit(context.Background(), pB, buildProjectAuditEntry(wsB.ID, accountID, pB.ID, project.AuditActionProjectCreate)); err != nil {
		t.Fatalf("create P_B: %v", err)
	}

	// Stray raw INSERT: workspace_id = wsA.ID, project_id = P_B.ID.
	strayID, _ := ids.New()
	now := time.Now().UTC()
	if err := db.Exec(
		`INSERT INTO audit_logs (id, workspace_id, project_id, actor_user_account_id, action, resource_type, result, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		strayID, wsA.ID, pB.ID, accountID, project.AuditActionProjectCreate, project.AuditResourceTypeProject, project.AuditResultSuccess, now,
	).Error; err != nil {
		t.Fatalf("stray insert: %v", err)
	}

	// Raw count confirms the stray row exists.
	var n int64
	if err := db.Raw(`SELECT COUNT(*) FROM audit_logs WHERE workspace_id = ? AND project_id = ?`, wsA.ID, pB.ID).Scan(&n).Error; err != nil {
		t.Fatalf("raw count: %v", err)
	}
	if n != 1 {
		t.Fatalf("raw count = %d, want 1 (stray row missing)", n)
	}

	// ListByProject(wsA.ID, pB.ID) → returns the stray row (binds to caller's TenantContext).
	rowsA, err := auditRepo.ListByProject(context.Background(), wsA.ID, pB.ID)
	if err != nil {
		t.Fatalf("ListByProject wsA: %v", err)
	}
	if len(rowsA) != 1 {
		t.Errorf("wsA: rows = %d, want 1", len(rowsA))
	}

	// ListByProject(wsB.ID, pB.ID) → INVISIBLE (proves D43 invariant): only the
	// genuine create row from project create is in ws B's view; the stray row
	// belongs to ws A's workspace_id binding.
	rowsB, err := auditRepo.ListByProject(context.Background(), wsB.ID, pB.ID)
	if err != nil {
		t.Fatalf("ListByProject wsB: %v", err)
	}
	for _, r := range rowsB {
		if r.WorkspaceID == nil || *r.WorkspaceID != wsB.ID {
			t.Errorf("D43 violation: ws B row has WorkspaceID = %v, want %v", r.WorkspaceID, wsB.ID)
		}
		if r.ID == strayID {
			t.Errorf("D43 violation: stray row leaked into ws B view")
		}
	}
}

func TestAuditRepo_LogTx_Rejects_ProjectIDWithoutWorkspaceID(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)

	pid, _ := ids.New()
	err := db.Transaction(func(tx *gorm.DB) error {
		return auditRepo.LogTx(tx, audit.Entry{
			ProjectID:    &pid,
			WorkspaceID:  nil,
			Action:       project.AuditActionProjectCreate,
			ResourceType: project.AuditResourceTypeProject,
			Result:       project.AuditResultSuccess,
		})
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "D43") {
		t.Errorf("err = %v, want message containing 'D43'", err)
	}
}

func TestAuditRepo_CHECK_Rejects_RawProjectIDWithoutWorkspaceID(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	accountID := createTestAccount(t, db)
	pid, _ := ids.New()
	auditID, _ := ids.New()
	now := time.Now().UTC()
	err := db.Exec(
		`INSERT INTO audit_logs (id, workspace_id, project_id, actor_user_account_id, action, resource_type, result, created_at)
		 VALUES (?, NULL, ?, ?, ?, ?, ?, ?)`,
		auditID, pid, accountID, project.AuditActionProjectCreate, project.AuditResourceTypeProject, project.AuditResultSuccess, now,
	).Error
	if err == nil {
		t.Fatal("expected CHECK violation, got nil")
	}
	if !strings.Contains(err.Error(), "chk_audit_logs_project_implies_workspace") && !strings.Contains(err.Error(), "23514") {
		t.Errorf("err = %v, want CHECK violation marker", err)
	}
}

// ── FK violations ─────────────────────────────────────────────────────────────

func TestProjectRepo_Create_InvalidStatusCode_FKReject(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectdbrepo.NewProjectRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	ws := createTestWorkspace(t, db, accountID)

	p := buildTestProject(t, ws.ID, accountID, strPtr("fk-status"), "not_real")
	err := repo.CreateWithAudit(context.Background(), p, buildProjectAuditEntry(ws.ID, accountID, p.ID, project.AuditActionProjectCreate))
	if err == nil {
		t.Fatal("expected FK violation, got nil")
	}
	if !strings.Contains(err.Error(), "23503") {
		t.Errorf("err = %v, want SQLSTATE 23503", err)
	}
}

func TestProjectRepo_Create_InvalidTypeCode_FKReject(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectdbrepo.NewProjectRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	ws := createTestWorkspace(t, db, accountID)

	p := buildTestProject(t, ws.ID, accountID, strPtr("fk-type"), "planning")
	p.ProjectTypeCode = "not_real"
	err := repo.CreateWithAudit(context.Background(), p, buildProjectAuditEntry(ws.ID, accountID, p.ID, project.AuditActionProjectCreate))
	if err == nil {
		t.Fatal("expected FK violation, got nil")
	}
	if !strings.Contains(err.Error(), "23503") {
		t.Errorf("err = %v, want SQLSTATE 23503", err)
	}
}

func TestProjectRepo_ChangeStatus_InvalidStatusCode_FKReject(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectdbrepo.NewProjectRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	ws := createTestWorkspace(t, db, accountID)

	p := buildTestProject(t, ws.ID, accountID, strPtr("fk-cs"), "planning")
	if err := repo.CreateWithAudit(context.Background(), p, buildProjectAuditEntry(ws.ID, accountID, p.ID, project.AuditActionProjectCreate)); err != nil {
		t.Fatalf("create: %v", err)
	}

	err := repo.ChangeStatusWithAudit(context.Background(), ws.ID, p.ID, "not_real", accountID, buildProjectAuditEntry(ws.ID, accountID, p.ID, project.AuditActionProjectStatusChange))
	if !errors.Is(err, project.ErrInvalidStatusCode) {
		t.Errorf("err = %v, want ErrInvalidStatusCode", err)
	}
}

func TestProjectRepo_Create_DateRangeCheck_Reject(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectdbrepo.NewProjectRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	ws := createTestWorkspace(t, db, accountID)

	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	p := buildTestProject(t, ws.ID, accountID, strPtr("date-bad"), "planning")
	p.StartDate = &start
	p.EndDate = &end

	err := repo.CreateWithAudit(context.Background(), p, buildProjectAuditEntry(ws.ID, accountID, p.ID, project.AuditActionProjectCreate))
	if err == nil {
		t.Fatal("expected CHECK violation, got nil")
	}
	if !strings.Contains(err.Error(), "ck_projects_date_range") && !strings.Contains(err.Error(), "23514") {
		t.Errorf("err = %v, want CHECK constraint violation", err)
	}
}

// ── Lifecycle (D44 free transitions) ──────────────────────────────────────────

func TestProjectRepo_ChangeStatus_FreeTransition_AllPairs(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectdbrepo.NewProjectRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	ws := createTestWorkspace(t, db, accountID)

	p := buildTestProject(t, ws.ID, accountID, strPtr("free-trans"), "planning")
	if err := repo.CreateWithAudit(context.Background(), p, buildProjectAuditEntry(ws.ID, accountID, p.ID, project.AuditActionProjectCreate)); err != nil {
		t.Fatalf("create: %v", err)
	}

	statuses := []string{"draft", "planning", "proposal", "active", "closing", "maintenance", "closed", "archived"}
	transitionCount := 0
	prev := "planning"
	for _, s := range statuses {
		if s == prev {
			continue
		}
		if err := repo.ChangeStatusWithAudit(context.Background(), ws.ID, p.ID, s, accountID, buildProjectAuditEntry(ws.ID, accountID, p.ID, project.AuditActionProjectStatusChange)); err != nil {
			t.Errorf("transition %s -> %s: %v", prev, s, err)
		}
		transitionCount++
		prev = s
	}

	var count int64
	if err := db.Table("audit_logs").
		Where("workspace_id = ? AND project_id = ? AND action = ?", ws.ID, p.ID, project.AuditActionProjectStatusChange).
		Count(&count).Error; err != nil {
		t.Fatalf("count audit: %v", err)
	}
	if int(count) != transitionCount {
		t.Errorf("status-change audit rows = %d, want %d", count, transitionCount)
	}
}

func TestProjectRepo_SoftDelete_ExcludedFromList(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectdbrepo.NewProjectRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	ws := createTestWorkspace(t, db, accountID)

	p := buildTestProject(t, ws.ID, accountID, strPtr("excluded"), "planning")
	if err := repo.CreateWithAudit(context.Background(), p, buildProjectAuditEntry(ws.ID, accountID, p.ID, project.AuditActionProjectCreate)); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.SoftDeleteWithAudit(context.Background(), ws.ID, p.ID, accountID, buildProjectAuditEntry(ws.ID, accountID, p.ID, project.AuditActionProjectDelete)); err != nil {
		t.Fatalf("delete: %v", err)
	}

	rows, total, err := repo.ListByWorkspace(context.Background(), ws.ID, project.ListOptions{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(rows) != 0 {
		t.Errorf("rows = %d, want 0", len(rows))
	}
}
