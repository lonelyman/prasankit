package projectmemberdbrepo_test

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
	projectmemberdbrepo "prasankit-api/internal/adapters/database/projectmember"
	workspacedbrepo "prasankit-api/internal/adapters/database/workspace"
	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/project"
	"prasankit-api/internal/modules/projectmember"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ── Scaffolding (local copy, package projectmemberdbrepo_test) ──────────────────

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
// project_members is listed explicitly (do not rely on TRUNCATE projects CASCADE).
func truncateTestTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	tables := []string{
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

func createTestAccount(t *testing.T, db *gorm.DB, displayName string) uuid.UUID {
	t.Helper()
	accountRepo := authdbrepo.NewAccountRepo(db)

	email := fmt.Sprintf("pmtest+%d@example.com", time.Now().UnixNano())
	id, _ := ids.New()
	iid, _ := ids.New()
	now := time.Now().UTC()
	account := auth.Account{
		ID:                id,
		PrimaryEmail:      email,
		DisplayName:       displayName,
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

// createTestWorkspace mints a workspace + its owner membership and returns BOTH ids
// (6a discarded the membership id; here we keep it for the FK / display_name tests).
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

// createTestMembership inserts a workspace_membership for (wsID, accountID) with the given
// org role + status and returns its id. Use to mint extra/second-ws/suspended memberships.
func createTestMembership(t *testing.T, db *gorm.DB, wsID, accountID uuid.UUID, orgRole, statusCode string) uuid.UUID {
	t.Helper()
	mID, _ := ids.New()
	now := time.Now().UTC()
	err := db.Exec(
		`INSERT INTO workspace_memberships
		 (id, workspace_id, user_account_id, org_role_code, membership_status_code, joined_at, created_at, created_by, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mID, wsID, accountID, orgRole, statusCode, now, now, accountID, now,
	).Error
	if err != nil {
		t.Fatalf("createTestMembership: %v", err)
	}
	return mID
}

// createTestProject inserts a project (no owner) and returns its id.
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

func buildMember(wsID, projectID, membershipID, accountID uuid.UUID, role string) projectmember.ProjectMember {
	id, _ := ids.New()
	now := time.Now().UTC()
	return projectmember.ProjectMember{
		ID:                    id,
		WorkspaceID:           wsID,
		ProjectID:             projectID,
		WorkspaceMembershipID: membershipID,
		ProjectRoleCode:       role,
		JoinedAt:              now,
		CreatedAt:             now,
		CreatedBy:             accountID,
		UpdatedAt:             now,
	}
}

func buildMemberAuditEntry(wsID, projectID, accountID, resourceID uuid.UUID, action string) audit.Entry {
	return audit.Entry{
		WorkspaceID:    &wsID,
		ProjectID:      &projectID,
		ActorAccountID: &accountID,
		Action:         action,
		ResourceType:   projectmember.AuditResourceTypeProjectMember,
		ResourceID:     &resourceID,
		Result:         projectmember.AuditResultSuccess,
		IP:             "127.0.0.1",
		UserAgent:      "test",
		RequestID:      "req-test",
	}
}

// ── (a) cross-ws membership FK reject ───────────────────────────────────────────

func TestProjectMemberRepo_AddWithAudit_CrossWorkspaceMembership_FKReject(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectmemberdbrepo.NewProjectMemberRepo(db, auditRepo)

	accountID := createTestAccount(t, db, "User A")
	wsA := createTestWorkspace(t, db, accountID)
	wsB := createTestWorkspace(t, db, accountID)
	projectA := createTestProject(t, db, wsA.ID, accountID)

	// membership from ws B used under wsID = A → composite FK #2 (membership) must fire.
	m := buildMember(wsA.ID, projectA, wsB.OwnerMembershipID, accountID, "member")
	err := repo.AddWithAudit(context.Background(), m, buildMemberAuditEntry(wsA.ID, projectA, accountID, m.ID, projectmember.AuditActionProjectMemberAdd))
	if err == nil {
		t.Fatal("expected composite FK violation, got nil")
	}
	if !strings.Contains(err.Error(), "23503") {
		t.Errorf("err = %v, want SQLSTATE 23503", err)
	}
}

// ── (b) cross-ws project FK reject ──────────────────────────────────────────────

func TestProjectMemberRepo_AddWithAudit_CrossWorkspaceProject_FKReject(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectmemberdbrepo.NewProjectMemberRepo(db, auditRepo)

	accountID := createTestAccount(t, db, "User A")
	wsA := createTestWorkspace(t, db, accountID)
	wsB := createTestWorkspace(t, db, accountID)
	projectB := createTestProject(t, db, wsB.ID, accountID)

	// project from ws B used under wsID = A → composite FK #1 (project) must fire.
	m := buildMember(wsA.ID, projectB, wsA.OwnerMembershipID, accountID, "member")
	err := repo.AddWithAudit(context.Background(), m, buildMemberAuditEntry(wsA.ID, projectB, accountID, m.ID, projectmember.AuditActionProjectMemberAdd))
	if err == nil {
		t.Fatal("expected composite FK violation, got nil")
	}
	if !strings.Contains(err.Error(), "23503") {
		t.Errorf("err = %v, want SQLSTATE 23503", err)
	}
}

// ── (c) list workspace isolation ────────────────────────────────────────────────

func TestProjectMemberRepo_ListByProject_WorkspaceIsolation(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectmemberdbrepo.NewProjectMemberRepo(db, auditRepo)

	accountID := createTestAccount(t, db, "User A")
	wsA := createTestWorkspace(t, db, accountID)
	wsB := createTestWorkspace(t, db, accountID)
	projectA := createTestProject(t, db, wsA.ID, accountID)

	mA := buildMember(wsA.ID, projectA, wsA.OwnerMembershipID, accountID, "member")
	if err := repo.AddWithAudit(context.Background(), mA, buildMemberAuditEntry(wsA.ID, projectA, accountID, mA.ID, projectmember.AuditActionProjectMemberAdd)); err != nil {
		t.Fatalf("add A: %v", err)
	}

	// ws B's ListByProject for project A's id must not see ws A's members.
	rows, err := repo.ListByProject(context.Background(), wsB.ID, projectA)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("cross-WS leak: rows = %d, want 0", len(rows))
	}
}

// ── (d) re-add after remove (partial unique allows) ─────────────────────────────

func TestProjectMemberRepo_RemoveThenReAdd_PartialUniqueAllows(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectmemberdbrepo.NewProjectMemberRepo(db, auditRepo)

	accountID := createTestAccount(t, db, "User A")
	wsA := createTestWorkspace(t, db, accountID)
	projectA := createTestProject(t, db, wsA.ID, accountID)
	// a second member account to add to the project.
	memberAcct := createTestAccount(t, db, "Member")
	membershipID := createTestMembership(t, db, wsA.ID, memberAcct, workspace.OrgRoleUser, workspace.MembershipStatusActive)

	m1 := buildMember(wsA.ID, projectA, membershipID, accountID, "member")
	if err := repo.AddWithAudit(context.Background(), m1, buildMemberAuditEntry(wsA.ID, projectA, accountID, m1.ID, projectmember.AuditActionProjectMemberAdd)); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := repo.RemoveWithAudit(context.Background(), wsA.ID, projectA, m1.ID, accountID, buildMemberAuditEntry(wsA.ID, projectA, accountID, m1.ID, projectmember.AuditActionProjectMemberRemove)); err != nil {
		t.Fatalf("remove: %v", err)
	}

	m2 := buildMember(wsA.ID, projectA, membershipID, accountID, "member")
	if err := repo.AddWithAudit(context.Background(), m2, buildMemberAuditEntry(wsA.ID, projectA, accountID, m2.ID, projectmember.AuditActionProjectMemberAdd)); err != nil {
		t.Fatalf("re-add: %v", err)
	}

	// Two rows total, exactly one active.
	var total, active int64
	if err := db.Table("project_members").Where("workspace_id = ? AND project_id = ? AND workspace_membership_id = ?", wsA.ID, projectA, membershipID).Count(&total).Error; err != nil {
		t.Fatalf("count total: %v", err)
	}
	if total != 2 {
		t.Errorf("total rows = %d, want 2", total)
	}
	if err := db.Table("project_members").Where("workspace_id = ? AND project_id = ? AND workspace_membership_id = ? AND removed_at IS NULL", wsA.ID, projectA, membershipID).Count(&active).Error; err != nil {
		t.Fatalf("count active: %v", err)
	}
	if active != 1 {
		t.Errorf("active rows = %d, want 1", active)
	}
}

// ── (e) multi-ws user, wrong-ws membership FK reject ────────────────────────────

func TestProjectMemberRepo_MultiWorkspaceUser_WrongWsMembership_FKReject(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectmemberdbrepo.NewProjectMemberRepo(db, auditRepo)

	// One user U with a membership in ws A and a membership in ws B.
	u := createTestAccount(t, db, "Multi-WS User")
	owner := createTestAccount(t, db, "Owner")
	wsA := createTestWorkspace(t, db, owner)
	wsB := createTestWorkspace(t, db, owner)
	_ = createTestMembership(t, db, wsA.ID, u, workspace.OrgRoleUser, workspace.MembershipStatusActive)
	membershipB := createTestMembership(t, db, wsB.ID, u, workspace.OrgRoleUser, workspace.MembershipStatusActive)
	projectA := createTestProject(t, db, wsA.ID, owner)

	// Insert at ws-A project with U's ws-B membership id → composite FK must fire.
	m := buildMember(wsA.ID, projectA, membershipB, owner, "member")
	err := repo.AddWithAudit(context.Background(), m, buildMemberAuditEntry(wsA.ID, projectA, owner, m.ID, projectmember.AuditActionProjectMemberAdd))
	if err == nil {
		t.Fatal("expected composite FK violation, got nil")
	}
	if !strings.Contains(err.Error(), "23503") {
		t.Errorf("err = %v, want SQLSTATE 23503", err)
	}
}

// ── (f) display_name from the project-workspace membership ──────────────────────

func TestProjectMemberRepo_ListByProject_DisplayNameFromProjectWorkspaceMembership(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectmemberdbrepo.NewProjectMemberRepo(db, auditRepo)

	// U is a member of ws A and ws B. (Same account → same user_accounts.display_name,
	// but the JOIN must reach the account VIA the ws-A membership, not a sibling.)
	owner := createTestAccount(t, db, "Owner")
	u := createTestAccount(t, db, "Display User")
	wsA := createTestWorkspace(t, db, owner)
	wsB := createTestWorkspace(t, db, owner)
	membershipA := createTestMembership(t, db, wsA.ID, u, workspace.OrgRoleUser, workspace.MembershipStatusActive)
	_ = createTestMembership(t, db, wsB.ID, u, workspace.OrgRoleUser, workspace.MembershipStatusActive)
	projectA := createTestProject(t, db, wsA.ID, owner)

	m := buildMember(wsA.ID, projectA, membershipA, owner, "member")
	if err := repo.AddWithAudit(context.Background(), m, buildMemberAuditEntry(wsA.ID, projectA, owner, m.ID, projectmember.AuditActionProjectMemberAdd)); err != nil {
		t.Fatalf("add: %v", err)
	}

	rows, err := repo.ListByProject(context.Background(), wsA.ID, projectA)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if rows[0].DisplayName != "Display User" {
		t.Errorf("display_name = %q, want %q", rows[0].DisplayName, "Display User")
	}
	if rows[0].WorkspaceMembershipID != membershipA {
		t.Errorf("membership id = %v, want ws-A membership %v", rows[0].WorkspaceMembershipID, membershipA)
	}
}

// ── ws-suspended member excluded ────────────────────────────────────────────────

func TestProjectMemberRepo_ListByProject_ExcludesSuspendedWorkspaceMember(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectmemberdbrepo.NewProjectMemberRepo(db, auditRepo)

	owner := createTestAccount(t, db, "Owner")
	suspended := createTestAccount(t, db, "Suspended User")
	wsA := createTestWorkspace(t, db, owner)
	projectA := createTestProject(t, db, wsA.ID, owner)
	suspendedMembership := createTestMembership(t, db, wsA.ID, suspended, workspace.OrgRoleUser, workspace.MembershipStatusSuspended)

	// Active project_members row, but the ws-membership is suspended → must be excluded.
	m := buildMember(wsA.ID, projectA, suspendedMembership, owner, "member")
	if err := repo.AddWithAudit(context.Background(), m, buildMemberAuditEntry(wsA.ID, projectA, owner, m.ID, projectmember.AuditActionProjectMemberAdd)); err != nil {
		t.Fatalf("add: %v", err)
	}

	rows, err := repo.ListByProject(context.Background(), wsA.ID, projectA)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("suspended ws-member leaked: rows = %d, want 0", len(rows))
	}
}

// ── audit written in same tx ────────────────────────────────────────────────────

func TestProjectMemberRepo_AddWithAudit_WritesAuditInSameTx(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectmemberdbrepo.NewProjectMemberRepo(db, auditRepo)

	owner := createTestAccount(t, db, "Owner")
	member := createTestAccount(t, db, "Member")
	wsA := createTestWorkspace(t, db, owner)
	projectA := createTestProject(t, db, wsA.ID, owner)
	membershipID := createTestMembership(t, db, wsA.ID, member, workspace.OrgRoleUser, workspace.MembershipStatusActive)

	m := buildMember(wsA.ID, projectA, membershipID, owner, "member")
	if err := repo.AddWithAudit(context.Background(), m, buildMemberAuditEntry(wsA.ID, projectA, owner, m.ID, projectmember.AuditActionProjectMemberAdd)); err != nil {
		t.Fatalf("add: %v", err)
	}

	var count int64
	if err := db.Table("audit_logs").
		Where("workspace_id = ? AND project_id = ? AND action = ?", wsA.ID, projectA, projectmember.AuditActionProjectMemberAdd).
		Count(&count).Error; err != nil {
		t.Fatalf("count audit_logs: %v", err)
	}
	if count != 1 {
		t.Errorf("audit_logs count = %d, want 1", count)
	}
}

// ── remove not-found → RowsAffected==0 → ErrProjectMemberNotFound ───────────────

func TestProjectMemberRepo_RemoveWithAudit_NotFound_RowsAffectedZero(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectmemberdbrepo.NewProjectMemberRepo(db, auditRepo)

	owner := createTestAccount(t, db, "Owner")
	wsA := createTestWorkspace(t, db, owner)
	projectA := createTestProject(t, db, wsA.ID, owner)

	nonexistent, _ := ids.New()
	err := repo.RemoveWithAudit(context.Background(), wsA.ID, projectA, nonexistent, owner, buildMemberAuditEntry(wsA.ID, projectA, owner, nonexistent, projectmember.AuditActionProjectMemberRemove))
	if !errors.Is(err, projectmember.ErrProjectMemberNotFound) {
		t.Errorf("err = %v, want ErrProjectMemberNotFound", err)
	}
}

// ── FindByID returns a REMOVED member (6b-2 additive read) ──────────────────────

// TestRepo_FindByID_ReturnsRemovedMember proves FindByID = FindActiveByID minus the
// removed_at IS NULL predicate: a removed member is returned with RemovedAt != nil, while
// FindActiveByID returns nil for the same row. This is what lets the junction service
// distinguish 404 (absent) from 422 (removed) — the composite FK cannot.
func TestRepo_FindByID_ReturnsRemovedMember(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectmemberdbrepo.NewProjectMemberRepo(db, auditRepo)

	owner := createTestAccount(t, db, "Owner")
	member := createTestAccount(t, db, "Member")
	wsA := createTestWorkspace(t, db, owner)
	projectA := createTestProject(t, db, wsA.ID, owner)
	membershipID := createTestMembership(t, db, wsA.ID, member, workspace.OrgRoleUser, workspace.MembershipStatusActive)

	m := buildMember(wsA.ID, projectA, membershipID, owner, "member")
	if err := repo.AddWithAudit(context.Background(), m, buildMemberAuditEntry(wsA.ID, projectA, owner, m.ID, projectmember.AuditActionProjectMemberAdd)); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := repo.RemoveWithAudit(context.Background(), wsA.ID, projectA, m.ID, owner, buildMemberAuditEntry(wsA.ID, projectA, owner, m.ID, projectmember.AuditActionProjectMemberRemove)); err != nil {
		t.Fatalf("remove: %v", err)
	}

	// FindActiveByID collapses the removed row to nil.
	active, err := repo.FindActiveByID(context.Background(), wsA.ID, projectA, m.ID)
	if err != nil {
		t.Fatalf("FindActiveByID: %v", err)
	}
	if active != nil {
		t.Errorf("FindActiveByID = %+v, want nil for a removed member", active)
	}

	// FindByID returns the removed row with RemovedAt populated.
	got, err := repo.FindByID(context.Background(), wsA.ID, projectA, m.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got == nil {
		t.Fatal("FindByID = nil, want the removed member row")
	}
	if got.RemovedAt == nil {
		t.Errorf("FindByID.RemovedAt = nil, want non-nil (removed_at IS NOT NULL)")
	}
	if got.ID != m.ID {
		t.Errorf("FindByID.ID = %v, want %v", got.ID, m.ID)
	}
}
