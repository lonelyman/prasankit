package smtp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	auditdbrepo "prasankit-api/internal/adapters/database/audit"
	authdbrepo "prasankit-api/internal/adapters/database/auth"
	workspacedbrepo "prasankit-api/internal/adapters/database/workspace"
	smtpadapter "prasankit-api/internal/adapters/email/smtp"
	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/pkg/ids"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func invEnvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// openInviteTestDB opens Postgres for the invite email integration test.
func openInviteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		invEnvOr("POSTGRES_PRIMARY_HOST", "localhost"),
		invEnvOr("POSTGRES_PRIMARY_PORT", "15433"),
		invEnvOr("POSTGRES_PRIMARY_USER", "prasankit"),
		invEnvOr("POSTGRES_PRIMARY_PASSWORD", "change_me"),
		invEnvOr("POSTGRES_PRIMARY_NAME", "prasankit"),
		invEnvOr("POSTGRES_SSL_MODE", "disable"),
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Skipf("postgres not reachable (%v): skipping invite email integration test", err)
	}
	sqlDB, _ := db.DB()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		t.Skipf("postgres ping failed (%v): skipping invite email integration test", err)
	}
	return db
}

// truncateInviteTables cleans all data tables used by this test.
func truncateInviteTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, tbl := range []string{
		"audit_logs",
		"workspace_invitations",
		"workspace_memberships",
		"workspaces",
		"security_events",
		"auth_identities",
		"user_accounts",
	} {
		db.Exec("TRUNCATE TABLE " + tbl + " CASCADE")
	}
}

// createInviteTestAccount inserts a user_account + identity and returns the account ID.
func createInviteTestAccount(t *testing.T, db *gorm.DB, email string) auth.Account {
	t.Helper()
	accountRepo := authdbrepo.NewAccountRepo(db)
	id, _ := ids.New()
	iid, _ := ids.New()
	now := time.Now().UTC()
	account := auth.Account{
		ID:                id,
		PrimaryEmail:      email,
		DisplayName:       "Invite Test User",
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
		t.Fatalf("createInviteTestAccount: %v", err)
	}
	return account
}

// TestInviteMember_EmailArrivesInMailpit drives the real InviteMember service method
// with the real SMTP sender and asserts the invite email lands in Mailpit.
// Since the send is synchronous, the email is present immediately after InviteMember returns.
func TestInviteMember_EmailArrivesInMailpit(t *testing.T) {
	smtpPort := mailpitSMTPPort()
	uiPort := mailpitUIPort()

	ensureMailpit(t, smtpPort)
	clearMailpit(t, uiPort)

	db := openInviteTestDB(t)
	truncateInviteTables(t, db)
	t.Cleanup(func() {
		truncateInviteTables(t, db)
		clearMailpit(t, uiPort)
	})

	// Wire real repos and real SMTP sender.
	auditRepo := auditdbrepo.NewAuditRepo(db)
	wsRepo := workspacedbrepo.NewWorkspaceRepo(db, auditRepo)
	memberRepo := workspacedbrepo.NewMembershipRepo(db)
	invRepo := workspacedbrepo.NewInvitationRepo(db, auditRepo)

	inviteBaseURL := "http://localhost:13000/invitations/accept"
	sender := smtpadapter.New(smtpadapter.Config{
		Host:        "localhost",
		Port:        smtpPort,
		FromAddress: "noreply@prasankit.local",
		FromName:    "Prasankit",
	})
	svc := workspace.NewService(wsRepo, memberRepo, invRepo, sender, inviteBaseURL)

	// Create owner account + workspace.
	ownerEmail := fmt.Sprintf("invite-owner+%d@example.com", time.Now().UnixNano())
	ownerAccount := createInviteTestAccount(t, db, ownerEmail)

	wsSlug := fmt.Sprintf("email-test-%d", time.Now().UnixNano()%1_000_000)

	// Use CreateWorkspace service method to set up the workspace properly.
	createdWS, err := svc.CreateWorkspace(context.Background(), workspace.CreateWorkspaceInput{
		AccountID:    ownerAccount.ID,
		Name:         "Email Test Workspace",
		Slug:         wsSlug,
		ContactEmail: ownerEmail,
		IP:           "127.0.0.1",
		UserAgent:    "test",
		RequestID:    "req-ws-create",
	})
	if err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}

	inviteeEmail := fmt.Sprintf("invitee+%d@example.com", time.Now().UnixNano())
	tc := workspace.TenantContext{
		WorkspaceID: createdWS.ID,
		AccountID:   ownerAccount.ID,
		OrgRoleCode: workspace.OrgRoleOwner,
	}

	_, err = svc.InviteMember(context.Background(), workspace.InviteMemberInput{
		TenantCtx:   tc,
		Email:       inviteeEmail,
		OrgRoleCode: workspace.OrgRoleUser,
		IP:          "127.0.0.1",
		UserAgent:   "test-agent",
		RequestID:   "req-invite-email",
	})
	if err != nil {
		t.Fatalf("InviteMember: %v", err)
	}

	// Send is synchronous — email is present immediately. Small pause for Mailpit to index.
	time.Sleep(300 * time.Millisecond)

	// Assert via Mailpit API.
	apiURL := fmt.Sprintf("http://localhost:%s/api/v1/messages", uiPort)
	resp, err := http.Get(apiURL)
	if err != nil {
		t.Fatalf("Mailpit API GET failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Mailpit API returned status %d", resp.StatusCode)
	}

	var result struct {
		Messages []struct {
			Subject string `json:"Subject"`
			To      []struct {
				Address string `json:"Address"`
			} `json:"To"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode Mailpit response: %v", err)
	}
	if len(result.Messages) == 0 {
		t.Fatal("expected at least 1 message in Mailpit, got 0")
	}

	found := false
	for _, msg := range result.Messages {
		if len(msg.To) == 0 {
			continue
		}
		if strings.EqualFold(msg.To[0].Address, inviteeEmail) {
			found = true
			// Subject should contain "invited".
			if !strings.Contains(strings.ToLower(msg.Subject), "invited") {
				t.Errorf("subject %q does not contain 'invited'", msg.Subject)
			}
			break
		}
	}
	if !found {
		t.Fatalf("no message addressed to %q found in Mailpit (got %d messages)", inviteeEmail, len(result.Messages))
	}

	// Verify the message body contains the accept link by fetching message detail.
	// Get the message ID for the invitee message.
	resp2, err := http.Get(apiURL)
	if err == nil {
		defer resp2.Body.Close()
		var result2 struct {
			Messages []struct {
				ID string                     `json:"ID"`
				To []struct{ Address string } `json:"To"`
			} `json:"messages"`
		}
		json.NewDecoder(resp2.Body).Decode(&result2) //nolint
		for _, msg := range result2.Messages {
			if len(msg.To) > 0 && strings.EqualFold(msg.To[0].Address, inviteeEmail) {
				// Fetch message body.
				bodyURL := fmt.Sprintf("http://localhost:%s/api/v1/message/%s", uiPort, msg.ID)
				bodyResp, err := http.Get(bodyURL)
				if err == nil {
					defer bodyResp.Body.Close()
					var bodyResult struct {
						Text string `json:"Text"`
					}
					if err := json.NewDecoder(bodyResp.Body).Decode(&bodyResult); err == nil {
						if !strings.Contains(bodyResult.Text, "?token=") {
							t.Errorf("invite email body missing '?token=': %s", bodyResult.Text)
						}
						if !strings.Contains(bodyResult.Text, inviteBaseURL) {
							t.Errorf("invite email body missing base URL %q: %s", inviteBaseURL, bodyResult.Text)
						}
					}
				}
				break
			}
		}
	}

	t.Logf("invite email to %q arrived in Mailpit — test passed", inviteeEmail)
}
