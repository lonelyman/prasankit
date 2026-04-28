package workspacerepo

import (
	"context"
	"errors"
	"os"
	"testing"

	"prasankit-api/internal/modules/workspace"
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
	defer sqlDB.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	accountID, err := ids.NewUUID()
	if err != nil {
		t.Fatalf("generate account id: %v", err)
	}
	email := "workspace-owner-" + uuid.NewString() + "@example.test"
	if err := db.Exec(
		`INSERT INTO user_accounts (id, primary_email, status) VALUES (?, ?, 'active')`,
		accountID,
		email,
	).Error; err != nil {
		t.Fatalf("insert user account: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Exec(`DELETE FROM workspace_memberships WHERE user_account_id = ?`, accountID).Error
		_ = db.Exec(`DELETE FROM workspaces WHERE owner_user_account_id = ?`, accountID).Error
		_ = db.Exec(`DELETE FROM user_accounts WHERE id = ?`, accountID).Error
	})

	workspaceRecord := &workspace.Workspace{
		Name:                  "Integration Workspace",
		Slug:                  "integration-" + uuid.NewString()[:8],
		Mode:                  workspace.WorkspaceModeDemo,
		Status:                workspace.WorkspaceStatusActive,
		ContactEmail:          email,
		OwnerUserAccountID:    accountID,
		EmailVerifiedRequired: true,
		CreatedBy:             accountID,
	}
	if err := repo.CreateWorkspace(ctx, workspaceRecord); err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if workspaceRecord.ID == uuid.Nil {
		t.Fatal("workspace ID was not set")
	}
	if workspaceRecord.ID.Version() != 7 {
		t.Fatalf("workspace ID version = %d, want 7", workspaceRecord.ID.Version())
	}
	if workspaceRecord.TenantID == uuid.Nil {
		t.Fatal("tenant ID was not set")
	}
	if workspaceRecord.TenantID.Version() != 7 {
		t.Fatalf("tenant ID version = %d, want 7", workspaceRecord.TenantID.Version())
	}

	foundWorkspace, err := repo.FindWorkspaceBySlug(ctx, workspaceRecord.Slug)
	if err != nil {
		t.Fatalf("find workspace by slug: %v", err)
	}
	if foundWorkspace.ID != workspaceRecord.ID {
		t.Fatalf("found workspace ID = %s, want %s", foundWorkspace.ID, workspaceRecord.ID)
	}

	membershipUserID := accountID
	membership := &workspace.Membership{
		TenantID:      workspaceRecord.TenantID,
		WorkspaceID:   workspaceRecord.ID,
		UserAccountID: &membershipUserID,
		Role:          workspace.WorkspaceRoleOwner,
		Status:        workspace.MembershipStatusActive,
		CreatedBy:     &accountID,
		UpdatedBy:     &accountID,
	}
	if err := repo.CreateMembership(ctx, membership); err != nil {
		t.Fatalf("create membership: %v", err)
	}
	if membership.ID == uuid.Nil {
		t.Fatal("membership ID was not set")
	}
	if membership.ID.Version() != 7 {
		t.Fatalf("membership ID version = %d, want 7", membership.ID.Version())
	}

	foundMembership, err := repo.FindActiveMembership(ctx, workspaceRecord.TenantID, accountID)
	if err != nil {
		t.Fatalf("find active membership: %v", err)
	}
	if foundMembership.ID != membership.ID {
		t.Fatalf("membership ID = %s, want %s", foundMembership.ID, membership.ID)
	}

	duplicate := *workspaceRecord
	duplicate.ID = uuid.Nil
	duplicate.TenantID = uuid.Nil
	err = repo.CreateWorkspace(ctx, &duplicate)
	if !errors.Is(err, workspace.ErrWorkspaceSlugAlreadyTaken) {
		t.Fatalf("duplicate err = %v, want ErrWorkspaceSlugAlreadyTaken", err)
	}
}
