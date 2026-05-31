package projectpositiondbrepo_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	auditdbrepo "prasankit-api/internal/adapters/database/audit"
	authdbrepo "prasankit-api/internal/adapters/database/auth"
	projectpositiondbrepo "prasankit-api/internal/adapters/database/projectposition"
	workspacedbrepo "prasankit-api/internal/adapters/database/workspace"
	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/projectposition"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ── Scaffolding (own copy; NO shared helper) ────────────────────────────────────

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
	// Skip when the 6b-2 schema has not been migrated yet (the reviewer owns the goose run).
	if !db.Migrator().HasTable("project_positions") {
		t.Skipf("project_positions table absent (migrations 000014+ not applied): skipping integration test")
	}
	return db
}

// truncateTestTables wipes DATA tables (project_positions/project_member_positions ARE data
// tables in M2, not seeded). Seeded system masters are NEVER truncated.
func truncateTestTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	tables := []string{
		"audit_logs",
		"project_member_positions",
		"project_positions",
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
	email := fmt.Sprintf("pptest+%d@example.com", time.Now().UnixNano())
	id, _ := ids.New()
	iid, _ := ids.New()
	now := time.Now().UTC()
	account := auth.Account{
		ID:                id,
		PrimaryEmail:      email,
		DisplayName:       "PP Test User",
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
	ID uuid.UUID
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
	return workspaceLite{ID: wsID}
}

func buildPosition(wsID, accountID uuid.UUID, code string) projectposition.ProjectPosition {
	id, _ := ids.New()
	now := time.Now().UTC()
	return projectposition.ProjectPosition{
		ID:          id,
		WorkspaceID: wsID,
		Code:        code,
		LabelTH:     "ตำแหน่ง",
		LabelEN:     "Position",
		SortOrder:   10,
		IsSystem:    false,
		Status:      "active",
		CreatedAt:   now,
		CreatedBy:   accountID,
		UpdatedAt:   now,
	}
}

func auditEntry(wsID, accountID, resourceID uuid.UUID, action string) audit.Entry {
	return audit.Entry{
		WorkspaceID:    &wsID,
		ProjectID:      nil, // masters workspace-scoped
		ActorAccountID: &accountID,
		Action:         action,
		ResourceType:   projectposition.AuditResourceTypeProjectPosition,
		ResourceID:     &resourceID,
		Result:         projectposition.AuditResultSuccess,
		IP:             "127.0.0.1",
		UserAgent:      "test",
		RequestID:      "req-test",
	}
}

// ── tests ───────────────────────────────────────────────────────────────────────

func TestRepo_CreateProjectPosition_PersistsAndAudits(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectpositiondbrepo.NewProjectPositionRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)

	p := buildPosition(wsA.ID, accountID, "tech_lead")
	if err := repo.CreateWithAudit(context.Background(), p, auditEntry(wsA.ID, accountID, p.ID, projectposition.AuditActionProjectPositionCreate)); err != nil {
		t.Fatalf("create: %v", err)
	}

	var n int64
	if err := db.Table("audit_logs").
		Where("workspace_id = ? AND project_id IS NULL AND action = ?", wsA.ID, projectposition.AuditActionProjectPositionCreate).
		Count(&n).Error; err != nil {
		t.Fatalf("count audit: %v", err)
	}
	if n != 1 {
		t.Errorf("audit count = %d, want 1 (project_id IS NULL)", n)
	}
}

func TestRepo_CreateProjectPosition_WorkspaceAandB_SameCode_BothSucceed(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectpositiondbrepo.NewProjectPositionRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)
	wsB := createTestWorkspace(t, db, accountID)

	pA := buildPosition(wsA.ID, accountID, "tech_lead")
	pB := buildPosition(wsB.ID, accountID, "tech_lead")
	if err := repo.CreateWithAudit(context.Background(), pA, auditEntry(wsA.ID, accountID, pA.ID, projectposition.AuditActionProjectPositionCreate)); err != nil {
		t.Fatalf("create A: %v", err)
	}
	if err := repo.CreateWithAudit(context.Background(), pB, auditEntry(wsB.ID, accountID, pB.ID, projectposition.AuditActionProjectPositionCreate)); err != nil {
		t.Fatalf("create B (same code, different ws): %v", err)
	}
}

func TestRepo_ListByWorkspace_DoesNotSeeOtherWorkspace(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectpositiondbrepo.NewProjectPositionRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)
	wsB := createTestWorkspace(t, db, accountID)

	pA := buildPosition(wsA.ID, accountID, "tech_lead")
	if err := repo.CreateWithAudit(context.Background(), pA, auditEntry(wsA.ID, accountID, pA.ID, projectposition.AuditActionProjectPositionCreate)); err != nil {
		t.Fatalf("create A: %v", err)
	}

	rows, total, err := repo.ListByWorkspace(context.Background(), wsB.ID, projectposition.ListOptions{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("list B: %v", err)
	}
	if total != 0 || len(rows) != 0 {
		t.Errorf("cross-WS leak: total = %d, rows = %d, want 0", total, len(rows))
	}
}

func TestRepo_CreateProjectPosition_DuplicateCodeSameWorkspace_UniqueViolation(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectpositiondbrepo.NewProjectPositionRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)

	p1 := buildPosition(wsA.ID, accountID, "tech_lead")
	if err := repo.CreateWithAudit(context.Background(), p1, auditEntry(wsA.ID, accountID, p1.ID, projectposition.AuditActionProjectPositionCreate)); err != nil {
		t.Fatalf("create 1: %v", err)
	}
	p2 := buildPosition(wsA.ID, accountID, "tech_lead") // same code, same ws
	err := repo.CreateWithAudit(context.Background(), p2, auditEntry(wsA.ID, accountID, p2.ID, projectposition.AuditActionProjectPositionCreate))
	if err != projectposition.ErrCodeTaken {
		t.Errorf("err = %v, want ErrCodeTaken (23505)", err)
	}
}

func TestRepo_DeprecateProjectPosition_KeepsRowAsFKTarget(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectpositiondbrepo.NewProjectPositionRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)

	p := buildPosition(wsA.ID, accountID, "tech_lead")
	if err := repo.CreateWithAudit(context.Background(), p, auditEntry(wsA.ID, accountID, p.ID, projectposition.AuditActionProjectPositionCreate)); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.DeprecateWithAudit(context.Background(), wsA.ID, "tech_lead", accountID, auditEntry(wsA.ID, accountID, p.ID, projectposition.AuditActionProjectPositionDeprecate)); err != nil {
		t.Fatalf("deprecate: %v", err)
	}

	// The row still exists (non-partial UNIQUE keeps it as an FK target) and is deprecated.
	got, err := repo.FindByCodeForWorkspace(context.Background(), wsA.ID, "tech_lead")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got == nil {
		t.Fatal("deprecated row missing — should remain as FK target")
	}
	if got.Status != "deprecated" {
		t.Errorf("status = %q, want deprecated", got.Status)
	}
}

func TestRepo_UpdateProjectPosition_DoesNotMutateCode(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectpositiondbrepo.NewProjectPositionRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)

	p := buildPosition(wsA.ID, accountID, "tech_lead")
	if err := repo.CreateWithAudit(context.Background(), p, auditEntry(wsA.ID, accountID, p.ID, projectposition.AuditActionProjectPositionCreate)); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Update label only; the Updates map never carries code.
	updated := p
	updated.LabelEN = "Renamed Lead"
	updatedBy := accountID
	updated.UpdatedBy = &updatedBy
	updated.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateWithAudit(context.Background(), updated, auditEntry(wsA.ID, accountID, p.ID, projectposition.AuditActionProjectPositionUpdate)); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := repo.FindByCodeForWorkspace(context.Background(), wsA.ID, "tech_lead")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got == nil || got.Code != "tech_lead" || got.LabelEN != "Renamed Lead" {
		t.Errorf("got = %+v, want code unchanged + label updated", got)
	}
}
