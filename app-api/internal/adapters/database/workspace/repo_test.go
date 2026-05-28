package workspacedbrepo_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	auditdbrepo "prasankit-api/internal/adapters/database/audit"
	authdbrepo "prasankit-api/internal/adapters/database/auth"
	workspacedbrepo "prasankit-api/internal/adapters/database/workspace"
	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

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

// truncateTestTables cleans data tables; never truncates master/seed tables.
func truncateTestTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	tables := []string{
		"audit_logs",
		"workspace_memberships",
		"workspaces",
		"security_events",
		"auth_identities",
		"user_accounts",
	}
	for _, tbl := range tables {
		if err := db.Exec("TRUNCATE TABLE " + tbl + " CASCADE").Error; err != nil {
			t.Logf("truncate %s: %v", tbl, err)
		}
	}
}

// createTestAccount inserts a user_account (needed for FK in workspaces.owner_user_account_id).
func createTestAccount(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	accountRepo := authdbrepo.NewAccountRepo(db)
	identityRepo := authdbrepo.NewIdentityRepo(db)

	email := fmt.Sprintf("wstest+%d@example.com", time.Now().UnixNano())
	id, _ := ids.New()
	iid, _ := ids.New()
	now := time.Now().UTC()
	account := auth.Account{
		ID:                id,
		PrimaryEmail:      email,
		DisplayName:       "WS Test User",
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
	_ = identityRepo
	return id
}

func buildTestWorkspace(accountID uuid.UUID, slug string) workspace.Workspace {
	id, _ := ids.New()
	now := time.Now().UTC()
	return workspace.Workspace{
		ID:                  id,
		WorkspaceName:       "Test Workspace",
		Slug:                slug,
		WorkspaceStatusCode: workspace.WorkspaceStatusActive,
		ContactEmail:        "admin@test.com",
		OwnerUserAccountID:  accountID,
		CreatedAt:           now,
		CreatedBy:           &accountID,
		UpdatedAt:           now,
	}
}

func buildTestMembership(wsID, accountID uuid.UUID) workspace.Membership {
	id, _ := ids.New()
	now := time.Now().UTC()
	joinedAt := now
	return workspace.Membership{
		ID:                   id,
		WorkspaceID:          wsID,
		UserAccountID:        accountID,
		OrgRoleCode:          workspace.OrgRoleOwner,
		MembershipStatusCode: workspace.MembershipStatusActive,
		JoinedAt:             &joinedAt,
		CreatedAt:            now,
		CreatedBy:            &accountID,
		UpdatedAt:            now,
	}
}

func buildTestAuditEntry(wsID, accountID uuid.UUID, resourceID uuid.UUID) audit.Entry {
	return audit.Entry{
		WorkspaceID:    &wsID,
		ActorAccountID: &accountID,
		Action:         workspace.AuditActionWorkspaceCreate,
		ResourceType:   workspace.AuditResourceTypeWorkspace,
		ResourceID:     &resourceID,
		Result:         workspace.AuditResultSuccess,
		IP:             "127.0.0.1",
		UserAgent:      "test",
		RequestID:      "req-test",
	}
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestWorkspaceRepo_CreateWorkspaceWithOwner_Atomic(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	wsRepo := workspacedbrepo.NewWorkspaceRepo(db, auditRepo)
	memberRepo := workspacedbrepo.NewMembershipRepo(db)

	accountID := createTestAccount(t, db)
	slug := fmt.Sprintf("test-ws-%d", time.Now().UnixNano())
	ws := buildTestWorkspace(accountID, slug)
	m := buildTestMembership(ws.ID, accountID)
	entry := buildTestAuditEntry(ws.ID, accountID, ws.ID)

	if err := wsRepo.CreateWorkspaceWithOwner(context.Background(), ws, m, entry); err != nil {
		t.Fatalf("CreateWorkspaceWithOwner: %v", err)
	}

	// Workspace persisted.
	found, err := wsRepo.FindBySlugActive(context.Background(), slug)
	if err != nil {
		t.Fatalf("FindBySlugActive: %v", err)
	}
	if found == nil {
		t.Fatal("workspace not found after create")
	}
	if found.ID != ws.ID {
		t.Errorf("workspace id = %v, want %v", found.ID, ws.ID)
	}

	// Membership persisted.
	ms, err := memberRepo.FindActiveByWorkspaceAndAccount(context.Background(), ws.ID, accountID)
	if err != nil {
		t.Fatalf("FindActiveByWorkspaceAndAccount: %v", err)
	}
	if ms == nil {
		t.Fatal("membership not found after create")
	}
	if ms.OrgRoleCode != workspace.OrgRoleOwner {
		t.Errorf("org_role_code = %q, want owner", ms.OrgRoleCode)
	}

	// Audit log persisted.
	var auditCount int64
	db.Table("audit_logs").Where("workspace_id = ? AND action = ?", ws.ID, workspace.AuditActionWorkspaceCreate).Count(&auditCount)
	if auditCount != 1 {
		t.Errorf("audit_logs count = %d, want 1", auditCount)
	}
}

func TestWorkspaceRepo_CreateWorkspaceWithOwner_Atomicity_OnDuplicateSlug(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	wsRepo := workspacedbrepo.NewWorkspaceRepo(db, auditRepo)
	memberRepo := workspacedbrepo.NewMembershipRepo(db)

	accountID := createTestAccount(t, db)
	slug := fmt.Sprintf("dup-slug-%d", time.Now().UnixNano())

	// Create first workspace successfully.
	ws1 := buildTestWorkspace(accountID, slug)
	m1 := buildTestMembership(ws1.ID, accountID)
	entry1 := buildTestAuditEntry(ws1.ID, accountID, ws1.ID)
	if err := wsRepo.CreateWorkspaceWithOwner(context.Background(), ws1, m1, entry1); err != nil {
		t.Fatalf("first create: %v", err)
	}

	// Attempt duplicate slug — must fail atomically.
	account2ID := createTestAccount(t, db)
	ws2 := buildTestWorkspace(account2ID, slug) // same slug → conflict
	m2 := buildTestMembership(ws2.ID, account2ID)
	entry2 := buildTestAuditEntry(ws2.ID, account2ID, ws2.ID)

	err := wsRepo.CreateWorkspaceWithOwner(context.Background(), ws2, m2, entry2)
	if err == nil {
		t.Fatal("expected error on duplicate slug, got nil")
	}

	// ws2 must NOT be present.
	found, _ := wsRepo.FindBySlugActive(context.Background(), slug)
	if found != nil && found.ID == ws2.ID {
		t.Error("ws2 should not have been persisted (atomicity failed)")
	}

	// ws2's membership must NOT be present.
	ms, _ := memberRepo.FindActiveByWorkspaceAndAccount(context.Background(), ws2.ID, account2ID)
	if ms != nil {
		t.Error("ws2 membership should not have been persisted (atomicity failed)")
	}

	// ws2's audit entry must NOT be present.
	var auditCount int64
	db.Table("audit_logs").Where("workspace_id = ?", ws2.ID).Count(&auditCount)
	if auditCount != 0 {
		t.Errorf("ws2 audit_logs count = %d, want 0 (atomicity failed)", auditCount)
	}
}

func TestWorkspaceRepo_FindBySlugActive_NotFound(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	wsRepo := workspacedbrepo.NewWorkspaceRepo(db, auditRepo)

	found, err := wsRepo.FindBySlugActive(context.Background(), "no-such-workspace")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found != nil {
		t.Error("expected nil, got workspace")
	}
}

func TestMembershipRepo_FindActiveByWorkspaceAndAccount_NotFound(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	memberRepo := workspacedbrepo.NewMembershipRepo(db)

	wsID, _ := ids.New()
	accID, _ := ids.New()
	ms, err := memberRepo.FindActiveByWorkspaceAndAccount(context.Background(), wsID, accID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ms != nil {
		t.Error("expected nil, got membership")
	}
}

// ── Isolation tests ────────────────────────────────────────────────────────────

// TestIsolation_ListByWorkspace verifies that ListByWorkspace(wsA) does NOT
// return wsB's members (isolation invariant 02 §4.3).
func TestIsolation_ListByWorkspace(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	wsRepo := workspacedbrepo.NewWorkspaceRepo(db, auditRepo)
	memberRepo := workspacedbrepo.NewMembershipRepo(db)

	accountA := createTestAccount(t, db)
	accountB := createTestAccount(t, db)

	slugA := fmt.Sprintf("iso-ws-a-%d", time.Now().UnixNano())
	slugB := fmt.Sprintf("iso-ws-b-%d", time.Now().UnixNano())

	wsA := buildTestWorkspace(accountA, slugA)
	mA := buildTestMembership(wsA.ID, accountA)
	entryA := buildTestAuditEntry(wsA.ID, accountA, wsA.ID)
	if err := wsRepo.CreateWorkspaceWithOwner(context.Background(), wsA, mA, entryA); err != nil {
		t.Fatalf("create wsA: %v", err)
	}

	wsB := buildTestWorkspace(accountB, slugB)
	mB := buildTestMembership(wsB.ID, accountB)
	entryB := buildTestAuditEntry(wsB.ID, accountB, wsB.ID)
	if err := wsRepo.CreateWorkspaceWithOwner(context.Background(), wsB, mB, entryB); err != nil {
		t.Fatalf("create wsB: %v", err)
	}

	// ListByWorkspace(wsA) must only return wsA's members.
	membersA, err := memberRepo.ListByWorkspace(context.Background(), wsA.ID)
	if err != nil {
		t.Fatalf("ListByWorkspace(wsA): %v", err)
	}
	for _, m := range membersA {
		if m.WorkspaceID != wsA.ID {
			t.Errorf("cross-workspace leak: got membership workspace_id=%v in wsA list", m.WorkspaceID)
		}
	}
	// Ensure wsB's member is NOT in wsA's list.
	for _, m := range membersA {
		if m.UserAccountID == accountB {
			t.Errorf("isolation violated: accountB membership appeared in wsA's list")
		}
	}
}

// TestIsolation_FindActiveByWorkspaceAndAccount verifies that an account that is NOT a
// member of wsA gets nil (→ 403 path).
func TestIsolation_FindActiveByWorkspaceAndAccount_NonMember(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	wsRepo := workspacedbrepo.NewWorkspaceRepo(db, auditRepo)
	memberRepo := workspacedbrepo.NewMembershipRepo(db)

	accountA := createTestAccount(t, db)
	accountB := createTestAccount(t, db) // not a member of wsA

	slug := fmt.Sprintf("iso-nonmember-%d", time.Now().UnixNano())
	wsA := buildTestWorkspace(accountA, slug)
	mA := buildTestMembership(wsA.ID, accountA)
	entryA := buildTestAuditEntry(wsA.ID, accountA, wsA.ID)
	if err := wsRepo.CreateWorkspaceWithOwner(context.Background(), wsA, mA, entryA); err != nil {
		t.Fatalf("create wsA: %v", err)
	}

	// accountB should have no membership in wsA.
	ms, err := memberRepo.FindActiveByWorkspaceAndAccount(context.Background(), wsA.ID, accountB)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ms != nil {
		t.Error("isolation violated: non-member account found active membership in wsA")
	}
}
