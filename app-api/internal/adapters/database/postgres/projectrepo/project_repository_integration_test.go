package projectrepo

import (
	"context"
	"os"
	"testing"
	"time"

	"prasankit-api/internal/modules/project"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestRepositoryIntegration(t *testing.T) {
	dsn := os.Getenv("PRASANKIT_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("PRASANKIT_TEST_DB_DSN is not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}

	ctx := context.Background()
	repo := NewRepository(db)

	accountID := mustUUID(t)
	tenantID := mustUUID(t)
	workspaceID := mustUUID(t)
	membershipID := mustUUID(t)
	email := "project-owner-" + uuid.NewString() + "@example.test"
	slug := "project-integration-" + uuid.NewString()[:8]

	var workspaceRoleIDRaw string
	if err := db.Raw(`SELECT id::text FROM workspace_roles WHERE code = 'owner'`).Scan(&workspaceRoleIDRaw).Error; err != nil {
		t.Fatalf("find workspace role: %v", err)
	}
	workspaceRoleID, err := uuid.Parse(workspaceRoleIDRaw)
	if err != nil {
		t.Fatalf("parse workspace role ID: %v", err)
	}
	if workspaceRoleID == uuid.Nil {
		t.Fatal("workspace owner role ID was not found")
	}

	if err := db.Exec(`INSERT INTO user_accounts (id, primary_email, status) VALUES (?, ?, 'active')`, accountID, email).Error; err != nil {
		t.Fatalf("insert user account: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO workspaces (
			id, tenant_id, workspace_name, slug, mode, status, contact_email,
			owner_user_account_id, email_verified_required, created_by
		)
		VALUES (?, ?, 'Project Integration Workspace', ?, 'demo', 'active', ?, ?, true, ?)
	`, workspaceID, tenantID, slug, email, accountID, accountID).Error; err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO workspace_memberships (
			id, tenant_id, workspace_id, user_account_id, workspace_role_id, status, joined_at, created_by, updated_by
		)
		VALUES (?, ?, ?, ?, ?, 'active', now(), ?, ?)
	`, membershipID, tenantID, workspaceID, accountID, workspaceRoleID, accountID, accountID).Error; err != nil {
		t.Fatalf("insert workspace membership: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Exec(`DELETE FROM project_members WHERE tenant_id = ?`, tenantID).Error
		_ = db.Exec(`DELETE FROM projects WHERE tenant_id = ?`, tenantID).Error
		_ = db.Exec(`DELETE FROM project_code_counters WHERE tenant_id = ?`, tenantID).Error
		_ = db.Exec(`DELETE FROM workspace_memberships WHERE tenant_id = ?`, tenantID).Error
		_ = db.Exec(`DELETE FROM workspaces WHERE tenant_id = ?`, tenantID).Error
		_ = db.Exec(`DELETE FROM user_accounts WHERE id = ?`, accountID).Error
		_ = sqlDB.Close()
	})

	ownerRole, err := repo.FindProjectRoleByCode(ctx, project.ProjectRoleOwner)
	if err != nil {
		t.Fatalf("find project owner role: %v", err)
	}
	mediumPriority, err := repo.FindProjectPriorityByCode(ctx, project.ProjectPriorityMedium)
	if err != nil {
		t.Fatalf("find project medium priority: %v", err)
	}

	code, err := repo.NextProjectCode(ctx, tenantID, workspaceID, "PRJ", time.Now().UTC().Year(), 4)
	if err != nil {
		t.Fatalf("next project code: %v", err)
	}
	if code == "" {
		t.Fatal("project code was empty")
	}

	projectRecord := &project.Project{
		TenantID:    tenantID,
		WorkspaceID: workspaceID,
		Code:        code,
		Name:        "Project A",
		Type:        project.ProjectTypeInternal,
		Status:      project.ProjectStatusDraft,
		PriorityID:  mediumPriority.ID,
		Priority:    mediumPriority.Code,
		CreatedBy:   accountID,
	}
	if err := repo.CreateProject(ctx, projectRecord); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if projectRecord.ID == uuid.Nil {
		t.Fatal("project ID was not set")
	}
	if projectRecord.ID.Version() != 7 {
		t.Fatalf("project ID version = %d, want 7", projectRecord.ID.Version())
	}

	member := &project.Member{
		TenantID:              tenantID,
		WorkspaceID:           workspaceID,
		ProjectID:             projectRecord.ID,
		WorkspaceMembershipID: membershipID,
		UserAccountID:         &accountID,
		RoleID:                ownerRole.ID,
		Role:                  ownerRole.Code,
		Status:                project.ProjectMemberStatusActive,
		CreatedBy:             &accountID,
		UpdatedBy:             &accountID,
	}
	if err := repo.CreateMember(ctx, member); err != nil {
		t.Fatalf("create member: %v", err)
	}
	if member.ID == uuid.Nil {
		t.Fatal("member ID was not set")
	}
	if member.ID.Version() != 7 {
		t.Fatalf("member ID version = %d, want 7", member.ID.Version())
	}

	items, total, err := repo.ListProjects(ctx, tenantID, workspaceID, 10, 0)
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if total < 1 {
		t.Fatalf("total = %d, want at least 1", total)
	}
	found := false
	for _, item := range items {
		if item.Project.ID == projectRecord.ID {
			found = true
			if item.Project.TenantID != tenantID {
				t.Fatalf("tenant ID = %s, want %s", item.Project.TenantID, tenantID)
			}
			if item.Member.Role != project.ProjectRoleOwner {
				t.Fatalf("member role = %s, want project_owner", item.Member.Role)
			}
		}
	}
	if !found {
		t.Fatalf("created project not found in list: %#v", items)
	}
}

func mustUUID(t *testing.T) uuid.UUID {
	t.Helper()

	id, err := ids.NewUUID()
	if err != nil {
		t.Fatalf("generate uuid: %v", err)
	}
	return id
}
