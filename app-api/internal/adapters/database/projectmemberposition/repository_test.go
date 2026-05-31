package projectmemberpositiondbrepo_test

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
	projectmemberpositiondbrepo "prasankit-api/internal/adapters/database/projectmemberposition"
	workspacedbrepo "prasankit-api/internal/adapters/database/workspace"
	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/project"
	"prasankit-api/internal/modules/projectmemberposition"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ── Scaffolding (local copy, package projectmemberpositiondbrepo_test) ──────────
// Per established per-package convention these helpers are duplicated (NOT imported)
// from the sibling projectmember / projectposition adapter tests.

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
	if !db.Migrator().HasTable("project_member_positions") {
		t.Skipf("project_member_positions table absent (migrations 000015+ not applied): skipping integration test")
	}
	return db
}

// truncateTestTables wipes the 9 DATA tables (children before parents) via a single
// TRUNCATE ... RESTART IDENTITY CASCADE. Seeded system masters (project_statuses /
// project_types / project_roles) are NEVER truncated.
func truncateTestTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	tables := strings.Join([]string{
		"audit_logs",
		"project_member_positions",
		"project_members",
		"project_positions",
		"projects",
		"workspace_memberships",
		"workspaces",
		"auth_identities",
		"user_accounts",
	}, ", ")
	if err := db.Exec("TRUNCATE " + tables + " RESTART IDENTITY CASCADE").Error; err != nil {
		t.Logf("truncate: %v", err)
	}
}

func createTestAccount(t *testing.T, db *gorm.DB, displayName string) uuid.UUID {
	t.Helper()
	accountRepo := authdbrepo.NewAccountRepo(db)

	email := fmt.Sprintf("pmptest+%d@example.com", time.Now().UnixNano())
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

// createTestWorkspace mints a workspace + its owner membership and returns BOTH ids.
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

// createTestMembership inserts a workspace_membership for (wsID, accountID) and returns its id.
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

// createTestProjectMember raw-INSERTs an ACTIVE project_members row (removed_at NULL,
// project_role_code='member') and returns its id. The composite FK requires (wsID,
// membershipID) to already exist in workspace_memberships and (wsID, projectID) in projects.
func createTestProjectMember(t *testing.T, db *gorm.DB, wsID, projectID, membershipID, accountID uuid.UUID) uuid.UUID {
	t.Helper()
	id, _ := ids.New()
	now := time.Now().UTC()
	err := db.Exec(
		`INSERT INTO project_members
		 (id, workspace_id, project_id, workspace_membership_id, project_role_code, joined_at, created_at, created_by, updated_at)
		 VALUES (?, ?, ?, ?, 'member', ?, ?, ?, ?)`,
		id, wsID, projectID, membershipID, now, now, accountID, now,
	).Error
	if err != nil {
		t.Fatalf("createTestProjectMember: %v", err)
	}
	return id
}

// createTestProjectPosition raw-INSERTs an ACTIVE project_positions row in (wsID, code).
func createTestProjectPosition(t *testing.T, db *gorm.DB, wsID, accountID uuid.UUID, code, labelTH, labelEN string) {
	t.Helper()
	id, _ := ids.New()
	now := time.Now().UTC()
	err := db.Exec(
		`INSERT INTO project_positions
		 (id, workspace_id, code, label_th, label_en, sort_order, is_system, status, created_at, created_by, updated_at)
		 VALUES (?, ?, ?, ?, ?, 0, false, 'active', ?, ?, ?)`,
		id, wsID, code, labelTH, labelEN, now, accountID, now,
	).Error
	if err != nil {
		t.Fatalf("createTestProjectPosition: %v", err)
	}
}

// buildAssignment constructs a ProjectMemberPosition domain struct for AssignWithAudit.
func buildAssignment(wsID, projectMemberID uuid.UUID, code string, accountID uuid.UUID) projectmemberposition.ProjectMemberPosition {
	id, _ := ids.New()
	return projectmemberposition.ProjectMemberPosition{
		ID:                  id,
		WorkspaceID:         wsID,
		ProjectMemberID:     projectMemberID,
		ProjectPositionCode: code,
		CreatedAt:           time.Now().UTC(),
		CreatedBy:           accountID,
	}
}

// assignEntry builds a PROJECT-SCOPED audit entry (WorkspaceID AND ProjectID both non-nil, D43).
func assignEntry(wsID, projectID, accountID, resourceID uuid.UUID, action string) audit.Entry {
	return audit.Entry{
		WorkspaceID:    &wsID,
		ProjectID:      &projectID,
		ActorAccountID: &accountID,
		Action:         action,
		ResourceType:   projectmemberposition.AuditResourceTypeProjectMemberPosition,
		ResourceID:     &resourceID,
		Result:         projectmemberposition.AuditResultSuccess,
		IP:             "127.0.0.1",
		UserAgent:      "test",
		RequestID:      "req-test",
	}
}

// ── 1. cross-ws position composite FK (fk_pmp_project_position) ─────────────────

func TestRepo_AssignPosition_CrossWorkspacePosition_FKReject(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectmemberpositiondbrepo.NewProjectMemberPositionRepo(db, auditRepo)

	accountA := createTestAccount(t, db, "Owner A")
	wsA := createTestWorkspace(t, db, accountA)
	wsB := createTestWorkspace(t, db, accountA)
	projectA := createTestProject(t, db, wsA.ID, accountA)
	memberA := createTestProjectMember(t, db, wsA.ID, projectA, wsA.OwnerMembershipID, accountA)

	// position code 'lead_b' exists ONLY in ws B.
	createTestProjectPosition(t, db, wsB.ID, accountA, "lead_b", "ผู้นำบี", "Lead B")

	// (i) DB-backstop: a RAW insert (bypasses adapter mapping) must fire the composite FK.
	rawID, _ := ids.New()
	rawErr := db.Exec(
		"INSERT INTO project_member_positions (id, workspace_id, project_member_id, project_position_code, created_at, created_by) VALUES (?,?,?,?,now(),?)",
		rawID, wsA.ID, memberA, "lead_b", accountA,
	).Error
	if rawErr == nil {
		t.Fatal("raw insert: expected composite FK violation, got nil")
	}
	if !strings.Contains(rawErr.Error(), "23503") {
		t.Errorf("raw insert err = %v, want SQLSTATE 23503", rawErr)
	}

	// (ii) adapter contract: AssignWithAudit maps the FK to ErrInvalidPositionCode + full rollback.
	mp := buildAssignment(wsA.ID, memberA, "lead_b", accountA)
	err := repo.AssignWithAudit(context.Background(), mp, assignEntry(wsA.ID, projectA, accountA, memberA, projectmemberposition.AuditActionPositionAssign))
	if !errors.Is(err, projectmemberposition.ErrInvalidPositionCode) {
		t.Fatalf("AssignWithAudit err = %v, want ErrInvalidPositionCode", err)
	}

	var dataRows, auditRows int64
	if err := db.Table("project_member_positions").
		Where("workspace_id = ? AND project_member_id = ?", wsA.ID, memberA).
		Count(&dataRows).Error; err != nil {
		t.Fatalf("count data rows: %v", err)
	}
	if dataRows != 0 {
		t.Errorf("tx rollback failed: data rows = %d, want 0", dataRows)
	}
	if err := db.Table("audit_logs").
		Where("workspace_id = ? AND action = ?", wsA.ID, projectmemberposition.AuditActionPositionAssign).
		Count(&auditRows).Error; err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	if auditRows != 0 {
		t.Errorf("tx rollback failed: audit rows = %d, want 0", auditRows)
	}
}

// ── 2. cross-ws member composite FK (fk_pmp_project_member) ─────────────────────

func TestRepo_AssignPosition_CrossWorkspaceMember_FKReject(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectmemberpositiondbrepo.NewProjectMemberPositionRepo(db, auditRepo)

	accountA := createTestAccount(t, db, "Owner A")
	wsA := createTestWorkspace(t, db, accountA)
	wsB := createTestWorkspace(t, db, accountA)
	projectA := createTestProject(t, db, wsA.ID, accountA)
	projectB := createTestProject(t, db, wsB.ID, accountA)

	// position 'lead_a' exists in ws A; member M_B belongs to ws B.
	createTestProjectPosition(t, db, wsA.ID, accountA, "lead_a", "ผู้นำเอ", "Lead A")
	memberB := createTestProjectMember(t, db, wsB.ID, projectB, wsB.OwnerMembershipID, accountA)

	// Attach ws-A position to ws-B's member UNDER workspace_id=A → member composite FK must fire.
	// (i) DB-backstop: raw insert → 23503.
	rawID, _ := ids.New()
	rawErr := db.Exec(
		"INSERT INTO project_member_positions (id, workspace_id, project_member_id, project_position_code, created_at, created_by) VALUES (?,?,?,?,now(),?)",
		rawID, wsA.ID, memberB, "lead_a", accountA,
	).Error
	if rawErr == nil {
		t.Fatal("raw insert: expected composite FK violation, got nil")
	}
	if !strings.Contains(rawErr.Error(), "23503") {
		t.Errorf("raw insert err = %v, want SQLSTATE 23503", rawErr)
	}

	// (ii) adapter contract: AssignWithAudit maps to ErrInvalidPositionCode + full rollback.
	mp := buildAssignment(wsA.ID, memberB, "lead_a", accountA)
	err := repo.AssignWithAudit(context.Background(), mp, assignEntry(wsA.ID, projectA, accountA, memberB, projectmemberposition.AuditActionPositionAssign))
	if !errors.Is(err, projectmemberposition.ErrInvalidPositionCode) {
		t.Fatalf("AssignWithAudit err = %v, want ErrInvalidPositionCode", err)
	}

	var dataRows, auditRows int64
	if err := db.Table("project_member_positions").
		Where("workspace_id = ? AND project_member_id = ?", wsA.ID, memberB).
		Count(&dataRows).Error; err != nil {
		t.Fatalf("count data rows: %v", err)
	}
	if dataRows != 0 {
		t.Errorf("tx rollback failed: data rows = %d, want 0", dataRows)
	}
	if err := db.Table("audit_logs").
		Where("workspace_id = ? AND action = ?", wsA.ID, projectmemberposition.AuditActionPositionAssign).
		Count(&auditRows).Error; err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	if auditRows != 0 {
		t.Errorf("tx rollback failed: audit rows = %d, want 0", auditRows)
	}
}

// ── 3. ListByMember workspace isolation (the WHERE pmp.workspace_id bind) ────────

func TestRepo_ListByMember_DoesNotSeeOtherWorkspace(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectmemberpositiondbrepo.NewProjectMemberPositionRepo(db, auditRepo)

	accountA := createTestAccount(t, db, "Owner A")
	wsA := createTestWorkspace(t, db, accountA)
	wsB := createTestWorkspace(t, db, accountA)
	projectA := createTestProject(t, db, wsA.ID, accountA)
	projectB := createTestProject(t, db, wsB.ID, accountA)
	memberA := createTestProjectMember(t, db, wsA.ID, projectA, wsA.OwnerMembershipID, accountA)
	memberB := createTestProjectMember(t, db, wsB.ID, projectB, wsB.OwnerMembershipID, accountA)
	createTestProjectPosition(t, db, wsA.ID, accountA, "lead_a", "ผู้นำเอ", "Lead A")

	mp := buildAssignment(wsA.ID, memberA, "lead_a", accountA)
	if err := repo.AssignWithAudit(context.Background(), mp, assignEntry(wsA.ID, projectA, accountA, memberA, projectmemberposition.AuditActionPositionAssign)); err != nil {
		t.Fatalf("assign: %v", err)
	}

	// ws B's ListByMember (for ws-B's own member) must not see ws-A's assignment.
	rowsB, err := repo.ListByMember(context.Background(), wsB.ID, memberB)
	if err != nil {
		t.Fatalf("ListByMember B: %v", err)
	}
	if len(rowsB) != 0 {
		t.Errorf("cross-WS leak: ws-B rows = %d, want 0", len(rowsB))
	}

	// ws A returns the assignment.
	rowsA, err := repo.ListByMember(context.Background(), wsA.ID, memberA)
	if err != nil {
		t.Fatalf("ListByMember A: %v", err)
	}
	if len(rowsA) != 1 {
		t.Fatalf("ws-A rows = %d, want 1", len(rowsA))
	}
	if rowsA[0].ProjectPositionCode != "lead_a" {
		t.Errorf("code = %q, want lead_a", rowsA[0].ProjectPositionCode)
	}
}

// ── 4. same code in two ws → label resolved from the member's OWN workspace ─────

func TestRepo_ListByMember_SameCodeAcrossWorkspaces_LabelFromOwnWorkspace(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectmemberpositiondbrepo.NewProjectMemberPositionRepo(db, auditRepo)

	accountA := createTestAccount(t, db, "Owner A")
	wsA := createTestWorkspace(t, db, accountA)
	wsB := createTestWorkspace(t, db, accountA)
	projectA := createTestProject(t, db, wsA.ID, accountA)
	memberA := createTestProjectMember(t, db, wsA.ID, projectA, wsA.OwnerMembershipID, accountA)

	// SAME code 'tech_lead' in BOTH workspaces, with DIFFERENT labels.
	createTestProjectPosition(t, db, wsA.ID, accountA, "tech_lead", "หัวหน้าเทคนิคเอ", "A-Lead")
	createTestProjectPosition(t, db, wsB.ID, accountA, "tech_lead", "หัวหน้าเทคนิคบี", "B-Lead")

	mp := buildAssignment(wsA.ID, memberA, "tech_lead", accountA)
	if err := repo.AssignWithAudit(context.Background(), mp, assignEntry(wsA.ID, projectA, accountA, memberA, projectmemberposition.AuditActionPositionAssign)); err != nil {
		t.Fatalf("assign: %v", err)
	}

	rows, err := repo.ListByMember(context.Background(), wsA.ID, memberA)
	if err != nil {
		t.Fatalf("ListByMember: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	// The JOIN's pp.workspace_id = pmp.workspace_id half must pick ws-A's label, never ws-B's.
	if rows[0].LabelEN != "A-Lead" {
		t.Errorf("label_en = %q, want %q (no cross-ws label leak)", rows[0].LabelEN, "A-Lead")
	}
}

// ── 5. duplicate assignment → unique violation (uq_pmp_member_position) ─────────

func TestRepo_AssignPosition_DuplicateAssignment_UniqueViolation(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectmemberpositiondbrepo.NewProjectMemberPositionRepo(db, auditRepo)

	accountA := createTestAccount(t, db, "Owner A")
	wsA := createTestWorkspace(t, db, accountA)
	projectA := createTestProject(t, db, wsA.ID, accountA)
	memberA := createTestProjectMember(t, db, wsA.ID, projectA, wsA.OwnerMembershipID, accountA)
	createTestProjectPosition(t, db, wsA.ID, accountA, "lead_a", "ผู้นำเอ", "Lead A")

	mp1 := buildAssignment(wsA.ID, memberA, "lead_a", accountA)
	if err := repo.AssignWithAudit(context.Background(), mp1, assignEntry(wsA.ID, projectA, accountA, memberA, projectmemberposition.AuditActionPositionAssign)); err != nil {
		t.Fatalf("assign 1: %v", err)
	}

	mp2 := buildAssignment(wsA.ID, memberA, "lead_a", accountA) // same (member, position)
	err := repo.AssignWithAudit(context.Background(), mp2, assignEntry(wsA.ID, projectA, accountA, memberA, projectmemberposition.AuditActionPositionAssign))
	if !errors.Is(err, projectmemberposition.ErrAlreadyAssigned) {
		t.Errorf("assign 2 err = %v, want ErrAlreadyAssigned (23505)", err)
	}
}

// ── 6. unassign deletes row + writes audit; second unassign → not found ─────────

func TestRepo_UnassignPosition_DeletesRowAndAudits(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectmemberpositiondbrepo.NewProjectMemberPositionRepo(db, auditRepo)

	accountA := createTestAccount(t, db, "Owner A")
	wsA := createTestWorkspace(t, db, accountA)
	projectA := createTestProject(t, db, wsA.ID, accountA)
	memberA := createTestProjectMember(t, db, wsA.ID, projectA, wsA.OwnerMembershipID, accountA)
	createTestProjectPosition(t, db, wsA.ID, accountA, "lead_a", "ผู้นำเอ", "Lead A")

	mp := buildAssignment(wsA.ID, memberA, "lead_a", accountA)
	if err := repo.AssignWithAudit(context.Background(), mp, assignEntry(wsA.ID, projectA, accountA, memberA, projectmemberposition.AuditActionPositionAssign)); err != nil {
		t.Fatalf("assign: %v", err)
	}

	if err := repo.UnassignWithAudit(context.Background(), wsA.ID, memberA, "lead_a", assignEntry(wsA.ID, projectA, accountA, memberA, projectmemberposition.AuditActionPositionUnassign)); err != nil {
		t.Fatalf("unassign: %v", err)
	}

	// Row is gone.
	var dataRows int64
	if err := db.Table("project_member_positions").
		Where("workspace_id = ? AND project_member_id = ? AND project_position_code = ?", wsA.ID, memberA, "lead_a").
		Count(&dataRows).Error; err != nil {
		t.Fatalf("count data rows: %v", err)
	}
	if dataRows != 0 {
		t.Errorf("row not deleted: data rows = %d, want 0", dataRows)
	}

	// An audit row exists for the unassign (project-scoped: workspace_id AND project_id).
	var auditRows int64
	if err := db.Table("audit_logs").
		Where("workspace_id = ? AND project_id = ? AND action = ?", wsA.ID, projectA, projectmemberposition.AuditActionPositionUnassign).
		Count(&auditRows).Error; err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	if auditRows != 1 {
		t.Errorf("unassign audit count = %d, want 1", auditRows)
	}

	// Unassigning again → not found.
	err := repo.UnassignWithAudit(context.Background(), wsA.ID, memberA, "lead_a", assignEntry(wsA.ID, projectA, accountA, memberA, projectmemberposition.AuditActionPositionUnassign))
	if !errors.Is(err, projectmemberposition.ErrAssignmentNotFound) {
		t.Errorf("second unassign err = %v, want ErrAssignmentNotFound", err)
	}
}

// ── 7. assign writes a PROJECT-SCOPED audit row (binds BOTH workspace_id + project_id) ──

func TestRepo_AssignPosition_ProjectScopedAuditRowCount(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	repo := projectmemberpositiondbrepo.NewProjectMemberPositionRepo(db, auditRepo)

	accountA := createTestAccount(t, db, "Owner A")
	wsA := createTestWorkspace(t, db, accountA)
	projectA := createTestProject(t, db, wsA.ID, accountA)
	memberA := createTestProjectMember(t, db, wsA.ID, projectA, wsA.OwnerMembershipID, accountA)
	createTestProjectPosition(t, db, wsA.ID, accountA, "lead_a", "ผู้นำเอ", "Lead A")

	mp := buildAssignment(wsA.ID, memberA, "lead_a", accountA)
	if err := repo.AssignWithAudit(context.Background(), mp, assignEntry(wsA.ID, projectA, accountA, memberA, projectmemberposition.AuditActionPositionAssign)); err != nil {
		t.Fatalf("assign: %v", err)
	}

	// The audit verification MUST bind BOTH workspace_id AND project_id (D43; project_id-alone
	// is forbidden).
	var n int64
	if err := db.Table("audit_logs").
		Where("workspace_id = ? AND project_id = ? AND action = ?", wsA.ID, projectA, projectmemberposition.AuditActionPositionAssign).
		Count(&n).Error; err != nil {
		t.Fatalf("count audit rows: %v", err)
	}
	if n != 1 {
		t.Errorf("assign audit count = %d, want 1 (workspace_id AND project_id)", n)
	}

	// Additionally assert the persisted row carries the expected non-null workspace_id + project_id.
	var got struct {
		WorkspaceID *uuid.UUID `gorm:"column:workspace_id"`
		ProjectID   *uuid.UUID `gorm:"column:project_id"`
	}
	if err := db.Table("audit_logs").
		Select("workspace_id, project_id").
		Where("action = ?", projectmemberposition.AuditActionPositionAssign).
		Take(&got).Error; err != nil {
		t.Fatalf("read audit row: %v", err)
	}
	if got.WorkspaceID == nil || *got.WorkspaceID != wsA.ID {
		t.Errorf("audit workspace_id = %v, want %v", got.WorkspaceID, wsA.ID)
	}
	if got.ProjectID == nil || *got.ProjectID != projectA {
		t.Errorf("audit project_id = %v, want %v", got.ProjectID, projectA)
	}
}
