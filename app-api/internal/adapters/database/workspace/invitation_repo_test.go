package workspacedbrepo_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	auditdbrepo "prasankit-api/internal/adapters/database/audit"
	workspacedbrepo "prasankit-api/internal/adapters/database/workspace"
	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/pkg/ids"
	"prasankit-api/pkg/securetoken"

	"github.com/google/uuid"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func buildTestInvitation(wsID, inviterID uuid.UUID, email string) workspace.Invitation {
	id, _ := ids.New()
	_, tokenHash, _ := securetoken.New()
	now := time.Now().UTC()
	return workspace.Invitation{
		ID:                     id,
		WorkspaceID:            wsID,
		Email:                  email,
		OrgRoleCode:            workspace.OrgRoleUser,
		TokenHash:              tokenHash,
		InvitedByUserAccountID: inviterID,
		ExpiresAt:              now.Add(7 * 24 * time.Hour),
		CreatedAt:              now,
	}
}

func buildInviteAuditEntry(wsID, actorID, resourceID uuid.UUID) audit.Entry {
	return audit.Entry{
		WorkspaceID:    &wsID,
		ActorAccountID: &actorID,
		Action:         workspace.AuditActionMemberInvite,
		ResourceType:   workspace.AuditResourceTypeInvitation,
		ResourceID:     &resourceID,
		Result:         workspace.AuditResultSuccess,
		IP:             "127.0.0.1",
		UserAgent:      "test",
		RequestID:      "req-inv-test",
	}
}

// setupWorkspaceWithOwner creates a workspace + owner membership and returns the wsID + ownerAccountID.
func setupWorkspaceWithOwner(t *testing.T, db interface{ Error() error }, wsRepo *workspacedbrepo.WorkspaceRepo, invSlug string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	return uuid.UUID{}, uuid.UUID{}
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestInvitationRepo_CreateInvitationTx_Atomic(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	wsRepo := workspacedbrepo.NewWorkspaceRepo(db, auditRepo)
	invRepo := workspacedbrepo.NewInvitationRepo(db, auditRepo)

	ownerID := createTestAccount(t, db)
	slug := fmt.Sprintf("inv-atomic-%d", time.Now().UnixNano())
	ws := buildTestWorkspace(ownerID, slug)
	m := buildTestMembership(ws.ID, ownerID)
	entry := buildTestAuditEntry(ws.ID, ownerID, ws.ID)
	if err := wsRepo.CreateWorkspaceWithOwner(context.Background(), ws, m, entry); err != nil {
		t.Fatalf("CreateWorkspaceWithOwner: %v", err)
	}

	inv := buildTestInvitation(ws.ID, ownerID, "atomictest@example.com")
	invEntry := buildInviteAuditEntry(ws.ID, ownerID, inv.ID)

	if err := invRepo.CreateInvitationTx(context.Background(), inv, invEntry); err != nil {
		t.Fatalf("CreateInvitationTx: %v", err)
	}

	// Invitation row persisted.
	var invCount int64
	db.Table("workspace_invitations").Where("id = ?", inv.ID).Count(&invCount)
	if invCount != 1 {
		t.Errorf("workspace_invitations count = %d, want 1", invCount)
	}

	// Audit row persisted.
	var auditCount int64
	db.Table("audit_logs").Where("resource_id = ? AND action = ?", inv.ID, workspace.AuditActionMemberInvite).Count(&auditCount)
	if auditCount != 1 {
		t.Errorf("audit_logs count = %d, want 1", auditCount)
	}
}

func TestInvitationRepo_CreateInvitationTx_DuplicatePending(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	wsRepo := workspacedbrepo.NewWorkspaceRepo(db, auditRepo)
	invRepo := workspacedbrepo.NewInvitationRepo(db, auditRepo)

	ownerID := createTestAccount(t, db)
	slug := fmt.Sprintf("inv-dup-%d", time.Now().UnixNano())
	ws := buildTestWorkspace(ownerID, slug)
	m := buildTestMembership(ws.ID, ownerID)
	entry := buildTestAuditEntry(ws.ID, ownerID, ws.ID)
	if err := wsRepo.CreateWorkspaceWithOwner(context.Background(), ws, m, entry); err != nil {
		t.Fatalf("CreateWorkspaceWithOwner: %v", err)
	}

	inv1 := buildTestInvitation(ws.ID, ownerID, "dup@example.com")
	if err := invRepo.CreateInvitationTx(context.Background(), inv1, buildInviteAuditEntry(ws.ID, ownerID, inv1.ID)); err != nil {
		t.Fatalf("first invite: %v", err)
	}

	// Second invite to same email in same workspace → ErrAlreadyPending.
	inv2 := buildTestInvitation(ws.ID, ownerID, "dup@example.com")
	err := invRepo.CreateInvitationTx(context.Background(), inv2, buildInviteAuditEntry(ws.ID, ownerID, inv2.ID))
	if err == nil {
		t.Fatal("expected error on duplicate pending, got nil")
	}
	if err != workspace.ErrAlreadyPending {
		t.Errorf("err = %v, want ErrAlreadyPending", err)
	}

	// inv2 row must NOT exist.
	var count int64
	db.Table("workspace_invitations").Where("id = ?", inv2.ID).Count(&count)
	if count != 0 {
		t.Error("inv2 should not have been persisted (atomicity failed)")
	}
}

func TestInvitationRepo_FindActiveByTokenHash(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	wsRepo := workspacedbrepo.NewWorkspaceRepo(db, auditRepo)
	invRepo := workspacedbrepo.NewInvitationRepo(db, auditRepo)

	ownerID := createTestAccount(t, db)
	slug := fmt.Sprintf("inv-find-%d", time.Now().UnixNano())
	ws := buildTestWorkspace(ownerID, slug)
	m := buildTestMembership(ws.ID, ownerID)
	if err := wsRepo.CreateWorkspaceWithOwner(context.Background(), ws, m, buildTestAuditEntry(ws.ID, ownerID, ws.ID)); err != nil {
		t.Fatalf("CreateWorkspaceWithOwner: %v", err)
	}

	rawToken, tokenHash, _ := securetoken.New()
	now := time.Now().UTC()
	inv := workspace.Invitation{
		ID:                     uuid.Must(ids.New()),
		WorkspaceID:            ws.ID,
		Email:                  "findme@example.com",
		OrgRoleCode:            workspace.OrgRoleUser,
		TokenHash:              tokenHash,
		InvitedByUserAccountID: ownerID,
		ExpiresAt:              now.Add(7 * 24 * time.Hour),
		CreatedAt:              now,
	}
	if err := invRepo.CreateInvitationTx(context.Background(), inv, buildInviteAuditEntry(ws.ID, ownerID, inv.ID)); err != nil {
		t.Fatalf("CreateInvitationTx: %v", err)
	}

	// Should be found by raw token's hash.
	found, err := invRepo.FindActiveByTokenHash(context.Background(), securetoken.Hash(rawToken))
	if err != nil {
		t.Fatalf("FindActiveByTokenHash: %v", err)
	}
	if found == nil {
		t.Fatal("expected invitation, got nil")
	}
	if found.ID != inv.ID {
		t.Errorf("ID = %v, want %v", found.ID, inv.ID)
	}

	// Bogus token → nil.
	notFound, err := invRepo.FindActiveByTokenHash(context.Background(), securetoken.Hash("bogus-raw-token"))
	if err != nil {
		t.Fatalf("FindActiveByTokenHash bogus: %v", err)
	}
	if notFound != nil {
		t.Error("expected nil for bogus token")
	}
}

func TestInvitationRepo_FindActiveByTokenHash_ExpiredReturnsNil(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	wsRepo := workspacedbrepo.NewWorkspaceRepo(db, auditRepo)
	invRepo := workspacedbrepo.NewInvitationRepo(db, auditRepo)

	ownerID := createTestAccount(t, db)
	slug := fmt.Sprintf("inv-expired-%d", time.Now().UnixNano())
	ws := buildTestWorkspace(ownerID, slug)
	m := buildTestMembership(ws.ID, ownerID)
	if err := wsRepo.CreateWorkspaceWithOwner(context.Background(), ws, m, buildTestAuditEntry(ws.ID, ownerID, ws.ID)); err != nil {
		t.Fatalf("CreateWorkspaceWithOwner: %v", err)
	}

	rawToken, tokenHash, _ := securetoken.New()
	now := time.Now().UTC()
	inv := workspace.Invitation{
		ID:                     uuid.Must(ids.New()),
		WorkspaceID:            ws.ID,
		Email:                  "expired@example.com",
		OrgRoleCode:            workspace.OrgRoleUser,
		TokenHash:              tokenHash,
		InvitedByUserAccountID: ownerID,
		ExpiresAt:              now.Add(-1 * time.Hour), // already expired
		CreatedAt:              now,
	}
	// Insert directly via raw SQL to bypass the future expires_at constraint.
	if err := db.Exec(
		"INSERT INTO workspace_invitations (id, workspace_id, email, org_role_code, token_hash, invited_by_user_account_id, expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		inv.ID, inv.WorkspaceID, inv.Email, inv.OrgRoleCode, inv.TokenHash, inv.InvitedByUserAccountID, inv.ExpiresAt, inv.CreatedAt,
	).Error; err != nil {
		t.Fatalf("insert expired invitation: %v", err)
	}

	found, err := invRepo.FindActiveByTokenHash(context.Background(), securetoken.Hash(rawToken))
	if err != nil {
		t.Fatalf("FindActiveByTokenHash: %v", err)
	}
	if found != nil {
		t.Error("expired invitation should not be returned as active")
	}
}

func TestInvitationRepo_AcceptInvitationTx_Atomic(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	wsRepo := workspacedbrepo.NewWorkspaceRepo(db, auditRepo)
	invRepo := workspacedbrepo.NewInvitationRepo(db, auditRepo)

	ownerID := createTestAccount(t, db)
	inviteeID := createTestAccount(t, db)
	slug := fmt.Sprintf("inv-accept-%d", time.Now().UnixNano())
	ws := buildTestWorkspace(ownerID, slug)
	m := buildTestMembership(ws.ID, ownerID)
	if err := wsRepo.CreateWorkspaceWithOwner(context.Background(), ws, m, buildTestAuditEntry(ws.ID, ownerID, ws.ID)); err != nil {
		t.Fatalf("CreateWorkspaceWithOwner: %v", err)
	}

	inv := buildTestInvitation(ws.ID, ownerID, "acceptee@example.com")
	if err := invRepo.CreateInvitationTx(context.Background(), inv, buildInviteAuditEntry(ws.ID, ownerID, inv.ID)); err != nil {
		t.Fatalf("CreateInvitationTx: %v", err)
	}

	now := time.Now().UTC()
	membershipID := uuid.Must(ids.New())
	joinedAt := now
	newMembership := workspace.Membership{
		ID:                     membershipID,
		WorkspaceID:            ws.ID,
		UserAccountID:          inviteeID,
		OrgRoleCode:            inv.OrgRoleCode,
		MembershipStatusCode:   workspace.MembershipStatusActive,
		InvitedByUserAccountID: &ownerID,
		JoinedAt:               &joinedAt,
		CreatedAt:              now,
		CreatedBy:              &inviteeID,
		UpdatedAt:              now,
	}
	acceptEntry := audit.Entry{
		WorkspaceID:    &ws.ID,
		ActorAccountID: &inviteeID,
		Action:         workspace.AuditActionInvitationAccept,
		ResourceType:   workspace.AuditResourceTypeMembership,
		ResourceID:     &membershipID,
		Result:         workspace.AuditResultSuccess,
		IP:             "127.0.0.1",
		UserAgent:      "test",
		RequestID:      "req-accept",
	}

	if err := invRepo.AcceptInvitationTx(context.Background(), inv, newMembership, acceptEntry); err != nil {
		t.Fatalf("AcceptInvitationTx: %v", err)
	}

	// Membership row inserted.
	var msCount int64
	db.Table("workspace_memberships").Where("id = ?", membershipID).Count(&msCount)
	if msCount != 1 {
		t.Errorf("membership count = %d, want 1", msCount)
	}

	// invitation.accepted_at set.
	var acceptedAt *time.Time
	db.Table("workspace_invitations").
		Where("id = ?", inv.ID).
		Select("accepted_at").
		Scan(&acceptedAt)
	if acceptedAt == nil {
		t.Error("invitation.accepted_at should be set after AcceptInvitationTx")
	}

	// Audit row for accept.
	var auditCount int64
	db.Table("audit_logs").Where("resource_id = ? AND action = ?", membershipID, workspace.AuditActionInvitationAccept).Count(&auditCount)
	if auditCount != 1 {
		t.Errorf("accept audit_logs count = %d, want 1", auditCount)
	}
}

func TestInvitationRepo_IsActiveMemberByEmail(t *testing.T) {
	db := openTestDB(t)
	truncateTestTables(t, db)
	t.Cleanup(func() { truncateTestTables(t, db) })

	auditRepo := auditdbrepo.NewAuditRepo(db)
	wsRepo := workspacedbrepo.NewWorkspaceRepo(db, auditRepo)
	invRepo := workspacedbrepo.NewInvitationRepo(db, auditRepo)

	ownerID := createTestAccount(t, db)
	slug := fmt.Sprintf("inv-ismember-%d", time.Now().UnixNano())
	ws := buildTestWorkspace(ownerID, slug)
	m := buildTestMembership(ws.ID, ownerID)
	if err := wsRepo.CreateWorkspaceWithOwner(context.Background(), ws, m, buildTestAuditEntry(ws.ID, ownerID, ws.ID)); err != nil {
		t.Fatalf("CreateWorkspaceWithOwner: %v", err)
	}

	// Owner's primary_email is set by createTestAccount — get it.
	var ownerEmail string
	db.Table("user_accounts").Where("id = ?", ownerID).Select("primary_email").Scan(&ownerEmail)

	// Owner is an active member — should return true.
	isMember, err := invRepo.IsActiveMemberByEmail(context.Background(), ws.ID, ownerEmail)
	if err != nil {
		t.Fatalf("IsActiveMemberByEmail: %v", err)
	}
	if !isMember {
		t.Error("owner should be detected as active member by email")
	}

	// Non-member email → false.
	isMember2, err := invRepo.IsActiveMemberByEmail(context.Background(), ws.ID, "stranger@example.com")
	if err != nil {
		t.Fatalf("IsActiveMemberByEmail (stranger): %v", err)
	}
	if isMember2 {
		t.Error("stranger should NOT be detected as active member")
	}
}
