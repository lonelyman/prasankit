// Command seed restores the dev fixtures — owner@/bob@ accounts (active) + the
// "prasankit" workspace — after the integration suite (which shares the dev DB
// on :15433 and TRUNCATEs identity/workspace tables) wipes them. Run from app-api:
//
//	go run ./cmd/seed
//
// Expects an empty / freshly-truncated DB (e.g. right after `make test`). Uses the
// real repos so accounts/workspaces satisfy the same invariants as the app.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	auditdbrepo "prasankit-api/internal/adapters/database/audit"
	authdbrepo "prasankit-api/internal/adapters/database/auth"
	workspacedbrepo "prasankit-api/internal/adapters/database/workspace"
	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/pkg/ids"
	"prasankit-api/pkg/passwordhash"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const seedPassword = "demopass123"

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func main() {
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
		log.Fatalf("connect %s: %v", dsn, err)
	}

	ctx := context.Background()
	ownerID := createAccount(ctx, db, "owner@prasankit.local", "Owner")
	wsID := createWorkspace(ctx, db, ownerID, "Prasankit", "prasankit")
	bobID := createAccount(ctx, db, "bob@prasankit.local", "Bob")
	addMembership(db, wsID, bobID, workspace.OrgRoleUser)

	log.Printf("seeded ws 'prasankit' (%s): owner@prasankit.local (owner) + bob@prasankit.local (user) / %s",
		wsID, seedPassword)
}

func createAccount(ctx context.Context, db *gorm.DB, email, displayName string) uuid.UUID {
	hash, err := passwordhash.Hash(seedPassword)
	if err != nil {
		log.Fatalf("hash: %v", err)
	}
	id, _ := ids.New()
	iid, _ := ids.New()
	now := time.Now().UTC()
	account := auth.Account{
		ID:                id,
		PrimaryEmail:      email,
		DisplayName:       displayName,
		AccountStatusCode: auth.AccountStatusActive, // active → login gate passes
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	identity := auth.Identity{
		ID:               iid,
		UserAccountID:    id,
		IdentityTypeCode: auth.IdentityTypeEmailPassword,
		Email:            email,
		PasswordHash:     hash,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := authdbrepo.NewAccountRepo(db).CreateWithIdentity(ctx, account, identity); err != nil {
		log.Fatalf("create account %s: %v", email, err)
	}
	return id
}

func createWorkspace(ctx context.Context, db *gorm.DB, ownerID uuid.UUID, name, slug string) uuid.UUID {
	auditRepo := auditdbrepo.NewAuditRepo(db)
	wsRepo := workspacedbrepo.NewWorkspaceRepo(db, auditRepo)

	wsID, _ := ids.New()
	mID, _ := ids.New()
	now := time.Now().UTC()
	ws := workspace.Workspace{
		ID:                  wsID,
		WorkspaceName:       name,
		Slug:                slug,
		WorkspaceStatusCode: workspace.WorkspaceStatusActive,
		ContactEmail:        "admin@prasankit.local",
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
		UserAgent:      "seed",
		RequestID:      "seed",
	}
	if err := wsRepo.CreateWorkspaceWithOwner(ctx, ws, m, entry); err != nil {
		log.Fatalf("create workspace: %v", err)
	}
	return wsID
}

func addMembership(db *gorm.DB, wsID, accountID uuid.UUID, orgRole string) {
	mID, _ := ids.New()
	now := time.Now().UTC()
	if err := db.Exec(
		`INSERT INTO workspace_memberships
		 (id, workspace_id, user_account_id, org_role_code, membership_status_code, joined_at, created_at, created_by, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		mID, wsID, accountID, orgRole, workspace.MembershipStatusActive, now, now, accountID, now,
	).Error; err != nil {
		log.Fatalf("add membership: %v", err)
	}
}
