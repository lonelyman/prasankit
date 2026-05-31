package companypositiondbrepo_test

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
	companypositiondbrepo "prasankit-api/internal/adapters/database/companyposition"
	workspacedbrepo "prasankit-api/internal/adapters/database/workspace"
	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/companyposition"
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
	if !db.Migrator().HasTable("company_positions") {
		t.Skipf("company_positions table absent (migrations 000014+ not applied): skipping integration test")
	}
	return db
}

func truncateTestTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	tables := []string{
		"audit_logs",
		"company_positions",
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
	email := fmt.Sprintf("cptest+%d@example.com", time.Now().UnixNano())
	id, _ := ids.New()
	iid, _ := ids.New()
	now := time.Now().UTC()
	account := auth.Account{
		ID:                id,
		PrimaryEmail:      email,
		DisplayName:       "CP Test User",
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
	ID                uuid.UUID
	OwnerMembershipID uuid.UUID
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
	return workspaceLite{ID: wsID, OwnerMembershipID: mID}
}

func buildPosition(wsID, accountID uuid.UUID, code string) companyposition.CompanyPosition {
	id, _ := ids.New()
	now := time.Now().UTC()
	return companyposition.CompanyPosition{
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
		ProjectID:      nil,
		ActorAccountID: &accountID,
		Action:         action,
		ResourceType:   companyposition.AuditResourceTypeCompanyPosition,
		ResourceID:     &resourceID,
		Result:         companyposition.AuditResultSuccess,
		IP:             "127.0.0.1",
		UserAgent:      "test",
		RequestID:      "req-test",
	}
}

func membershipAuditEntry(wsID, accountID, resourceID uuid.UUID) audit.Entry {
	return audit.Entry{
		WorkspaceID:    &wsID,
		ProjectID:      nil,
		ActorAccountID: &accountID,
		Action:         companyposition.AuditActionMembershipCompanyPositionChange,
		ResourceType:   companyposition.AuditResourceTypeWorkspaceMembership,
		ResourceID:     &resourceID,
		Result:         companyposition.AuditResultSuccess,
		IP:             "127.0.0.1",
		UserAgent:      "test",
		RequestID:      "req-test",
	}
}

func strPtr(s string) *string { return &s }

func mustCreatePosition(t *testing.T, repo *companypositiondbrepo.CompanyPositionRepo, wsID, accountID uuid.UUID, code string) {
	t.Helper()
	p := buildPosition(wsID, accountID, code)
	if err := repo.CreateWithAudit(context.Background(), p, auditEntry(wsID, accountID, p.ID, companyposition.AuditActionCompanyPositionCreate)); err != nil {
		t.Fatalf("create position %q: %v", code, err)
	}
}

// ── Master tests ─────────────────────────────────────────────────────────────

func TestRepo_CreateCompanyPosition_WorkspaceAandB_SameCode_BothSucceed(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := companypositiondbrepo.NewCompanyPositionRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)
	wsB := createTestWorkspace(t, db, accountID)

	mustCreatePosition(t, repo, wsA.ID, accountID, "engineer")
	mustCreatePosition(t, repo, wsB.ID, accountID, "engineer")
}

func TestRepo_ListByWorkspace_DoesNotSeeOtherWorkspace(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := companypositiondbrepo.NewCompanyPositionRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)
	wsB := createTestWorkspace(t, db, accountID)
	mustCreatePosition(t, repo, wsA.ID, accountID, "engineer")

	rows, total, err := repo.ListByWorkspace(context.Background(), wsB.ID, companyposition.ListOptions{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("list B: %v", err)
	}
	if total != 0 || len(rows) != 0 {
		t.Errorf("cross-WS leak: total = %d, rows = %d, want 0", total, len(rows))
	}
}

func TestRepo_CreateCompanyPosition_DuplicateCodeSameWorkspace_UniqueViolation(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := companypositiondbrepo.NewCompanyPositionRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)
	mustCreatePosition(t, repo, wsA.ID, accountID, "engineer")

	p2 := buildPosition(wsA.ID, accountID, "engineer")
	err := repo.CreateWithAudit(context.Background(), p2, auditEntry(wsA.ID, accountID, p2.ID, companyposition.AuditActionCompanyPositionCreate))
	if err != companyposition.ErrCodeTaken {
		t.Errorf("err = %v, want ErrCodeTaken (23505)", err)
	}
}

func TestRepo_DeprecateCompanyPosition_KeepsRowAsFKTarget(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := companypositiondbrepo.NewCompanyPositionRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)
	mustCreatePosition(t, repo, wsA.ID, accountID, "engineer")

	if err := repo.DeprecateWithAudit(context.Background(), wsA.ID, "engineer", accountID, auditEntry(wsA.ID, accountID, accountID, companyposition.AuditActionCompanyPositionDeprecate)); err != nil {
		t.Fatalf("deprecate: %v", err)
	}
	got, err := repo.FindByCodeForWorkspace(context.Background(), wsA.ID, "engineer")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got == nil || got.Status != "deprecated" {
		t.Errorf("got = %+v, want deprecated row kept", got)
	}
}

func TestRepo_UpdateCompanyPosition_DoesNotMutateCode(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := companypositiondbrepo.NewCompanyPositionRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)
	p := buildPosition(wsA.ID, accountID, "engineer")
	if err := repo.CreateWithAudit(context.Background(), p, auditEntry(wsA.ID, accountID, p.ID, companyposition.AuditActionCompanyPositionCreate)); err != nil {
		t.Fatalf("create: %v", err)
	}

	updated := p
	updated.LabelEN = "Senior Engineer"
	updatedBy := accountID
	updated.UpdatedBy = &updatedBy
	updated.UpdatedAt = time.Now().UTC()
	if err := repo.UpdateWithAudit(context.Background(), updated, auditEntry(wsA.ID, accountID, p.ID, companyposition.AuditActionCompanyPositionUpdate)); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, err := repo.FindByCodeForWorkspace(context.Background(), wsA.ID, "engineer")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got == nil || got.Code != "engineer" || got.LabelEN != "Senior Engineer" {
		t.Errorf("got = %+v, want code unchanged + label updated", got)
	}
}

// ── ws-attach tests ──────────────────────────────────────────────────────────

func TestRepo_SetCompanyPosition_CrossWorkspaceCode_FKReject(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := companypositiondbrepo.NewCompanyPositionRepo(db, auditRepo)
	memRepo := companypositiondbrepo.NewMembershipPositionRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)
	wsB := createTestWorkspace(t, db, accountID)
	// position "engineer" exists ONLY in ws B.
	mustCreatePosition(t, repo, wsB.ID, accountID, "engineer")

	// Attempt to set ws-A membership's company_position to ws-B's code under wsID=A → composite FK reject.
	err := memRepo.SetCompanyPositionWithAudit(context.Background(), wsA.ID, wsA.OwnerMembershipID, strPtr("engineer"), accountID, membershipAuditEntry(wsA.ID, accountID, wsA.OwnerMembershipID))
	if err == nil {
		t.Fatal("expected FK violation, got nil")
	}
	if !errors.Is(err, companyposition.ErrInvalidPositionCode) && !strings.Contains(err.Error(), "23503") {
		t.Errorf("err = %v, want 23503/ErrInvalidPositionCode", err)
	}

	// Post-condition: membership row unchanged + zero audit row for the attempted ws/action → tx rolled back.
	var code *string
	if err := db.Table("workspace_memberships").Select("company_position_code").
		Where("workspace_id = ? AND id = ?", wsA.ID, wsA.OwnerMembershipID).Row().Scan(&code); err != nil {
		t.Fatalf("read membership: %v", err)
	}
	if code != nil {
		t.Errorf("company_position_code = %v, want nil (unchanged)", *code)
	}
	var n int64
	if err := db.Table("audit_logs").
		Where("workspace_id = ? AND project_id IS NULL AND action = ?", wsA.ID, companyposition.AuditActionMembershipCompanyPositionChange).
		Count(&n).Error; err != nil {
		t.Fatalf("count audit: %v", err)
	}
	if n != 0 {
		t.Errorf("audit count = %d, want 0 (rolled back)", n)
	}
}

func TestRepo_SetCompanyPosition_CrossWorkspaceMembership_NotFound(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := companypositiondbrepo.NewCompanyPositionRepo(db, auditRepo)
	memRepo := companypositiondbrepo.NewMembershipPositionRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)
	wsB := createTestWorkspace(t, db, accountID)
	// valid code in ws B.
	mustCreatePosition(t, repo, wsB.ID, accountID, "engineer")

	// membership belongs to ws A; call with wsID=B → WHERE binds workspace_id → RowsAffected==0 → 404.
	err := memRepo.SetCompanyPositionWithAudit(context.Background(), wsB.ID, wsA.OwnerMembershipID, strPtr("engineer"), accountID, membershipAuditEntry(wsB.ID, accountID, wsA.OwnerMembershipID))
	if !errors.Is(err, companyposition.ErrMembershipNotFound) {
		t.Errorf("err = %v, want ErrMembershipNotFound (cross-ws isolation)", err)
	}
}

func TestRepo_SetCompanyPosition_ClearSetsNull(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := companypositiondbrepo.NewCompanyPositionRepo(db, auditRepo)
	memRepo := companypositiondbrepo.NewMembershipPositionRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)
	mustCreatePosition(t, repo, wsA.ID, accountID, "engineer")

	// set then clear.
	if err := memRepo.SetCompanyPositionWithAudit(context.Background(), wsA.ID, wsA.OwnerMembershipID, strPtr("engineer"), accountID, membershipAuditEntry(wsA.ID, accountID, wsA.OwnerMembershipID)); err != nil {
		t.Fatalf("set: %v", err)
	}
	if err := memRepo.SetCompanyPositionWithAudit(context.Background(), wsA.ID, wsA.OwnerMembershipID, nil, accountID, membershipAuditEntry(wsA.ID, accountID, wsA.OwnerMembershipID)); err != nil {
		t.Fatalf("clear: %v", err)
	}

	var code *string
	if err := db.Table("workspace_memberships").Select("company_position_code").
		Where("workspace_id = ? AND id = ?", wsA.ID, wsA.OwnerMembershipID).Row().Scan(&code); err != nil {
		t.Fatalf("read membership: %v", err)
	}
	if code != nil {
		t.Errorf("company_position_code = %v, want nil after clear", *code)
	}
}

func TestRepo_SetCompanyPosition_PersistsAndAudits(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := companypositiondbrepo.NewCompanyPositionRepo(db, auditRepo)
	memRepo := companypositiondbrepo.NewMembershipPositionRepo(db, auditRepo)

	accountID := createTestAccount(t, db)
	wsA := createTestWorkspace(t, db, accountID)
	mustCreatePosition(t, repo, wsA.ID, accountID, "engineer")

	if err := memRepo.SetCompanyPositionWithAudit(context.Background(), wsA.ID, wsA.OwnerMembershipID, strPtr("engineer"), accountID, membershipAuditEntry(wsA.ID, accountID, wsA.OwnerMembershipID)); err != nil {
		t.Fatalf("set: %v", err)
	}

	var code *string
	if err := db.Table("workspace_memberships").Select("company_position_code").
		Where("workspace_id = ? AND id = ?", wsA.ID, wsA.OwnerMembershipID).Row().Scan(&code); err != nil {
		t.Fatalf("read membership: %v", err)
	}
	if code == nil || *code != "engineer" {
		t.Errorf("company_position_code = %v, want engineer", code)
	}

	var n int64
	if err := db.Table("audit_logs").
		Where("workspace_id = ? AND project_id IS NULL AND action = ?", wsA.ID, companyposition.AuditActionMembershipCompanyPositionChange).
		Count(&n).Error; err != nil {
		t.Fatalf("count audit: %v", err)
	}
	if n != 1 {
		t.Errorf("audit count = %d, want 1", n)
	}
}

func TestRepo_AlterWorkspaceMemberships_ExistingRowsRemainNullAndUsable(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	accountID := createTestAccount(t, db)
	// createTestWorkspace inserts a membership via the M1 path against the migrated schema.
	wsA := createTestWorkspace(t, db, accountID)

	// company_position_code is NULL (no backfill, MATCH SIMPLE skips NULL FK component).
	var code *string
	if err := db.Table("workspace_memberships").Select("company_position_code").
		Where("workspace_id = ? AND id = ?", wsA.ID, wsA.OwnerMembershipID).Row().Scan(&code); err != nil {
		t.Fatalf("read membership: %v", err)
	}
	if code != nil {
		t.Errorf("company_position_code = %v, want nil (existing row, no backfill)", *code)
	}

	// The row is still updatable (e.g. updated_at), proving the ALTER did not break it.
	now := time.Now().UTC()
	if err := db.Exec("UPDATE workspace_memberships SET updated_at = ? WHERE workspace_id = ? AND id = ?", now, wsA.ID, wsA.OwnerMembershipID).Error; err != nil {
		t.Errorf("row not updatable after ALTER: %v", err)
	}
}
