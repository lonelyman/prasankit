package workspace_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"prasankit-api/internal/modules/audit"
	"prasankit-api/internal/modules/email"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/pkg/ids"

	"github.com/google/uuid"
)

// ── Fake implementations of ports ─────────────────────────────────────────────

type fakeWorkspaceRepo struct {
	mu         sync.Mutex
	workspaces map[uuid.UUID]*workspace.Workspace
	slugIndex  map[string]uuid.UUID // slug → id
	forceErr   error                // if set, CreateWorkspaceWithOwner returns this
}

func newFakeWorkspaceRepo() *fakeWorkspaceRepo {
	return &fakeWorkspaceRepo{
		workspaces: map[uuid.UUID]*workspace.Workspace{},
		slugIndex:  map[string]uuid.UUID{},
	}
}

// lastEntry is the last audit.Entry passed to CreateWorkspaceWithOwner.
var lastEntry audit.Entry

// lastMembership is the last Membership passed to CreateWorkspaceWithOwner.
var lastMembership workspace.Membership

func (r *fakeWorkspaceRepo) CreateWorkspaceWithOwner(ctx context.Context, ws workspace.Workspace, m workspace.Membership, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.forceErr != nil {
		return r.forceErr
	}
	if _, exists := r.slugIndex[ws.Slug]; exists {
		return workspace.ErrSlugTaken
	}
	cp := ws
	r.workspaces[ws.ID] = &cp
	r.slugIndex[ws.Slug] = ws.ID
	lastEntry = entry
	lastMembership = m
	return nil
}

func (r *fakeWorkspaceRepo) FindBySlugActive(ctx context.Context, slug string) (*workspace.Workspace, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.slugIndex[slug]
	if !ok {
		return nil, nil
	}
	cp := *r.workspaces[id]
	return &cp, nil
}

type fakeMembershipRepo struct {
	mu          sync.Mutex
	memberships []workspace.Membership
}

// ── fakeInvitationRepo ────────────────────────────────────────────────────────

type fakeInvitationRepo struct {
	mu              sync.Mutex
	invitations     []workspace.Invitation
	activeMemberMap map[string]bool // "workspaceID:email" → bool
	forceCreateErr  error           // if set, CreateInvitationTx returns this
}

func newFakeInvitationRepo() *fakeInvitationRepo {
	return &fakeInvitationRepo{
		activeMemberMap: map[string]bool{},
	}
}

func (r *fakeInvitationRepo) CreateInvitationTx(ctx context.Context, inv workspace.Invitation, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.forceCreateErr != nil {
		return r.forceCreateErr
	}
	r.invitations = append(r.invitations, inv)
	return nil
}

func (r *fakeInvitationRepo) FindActiveByTokenHash(ctx context.Context, tokenHash string) (*workspace.Invitation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, inv := range r.invitations {
		if inv.TokenHash == tokenHash && inv.IsActive(timeNow()) {
			cp := inv
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *fakeInvitationRepo) AcceptInvitationTx(ctx context.Context, inv workspace.Invitation, m workspace.Membership, entry audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, stored := range r.invitations {
		if stored.ID == inv.ID {
			now := timeNow()
			r.invitations[i].AcceptedAt = &now
			r.invitations[i].AcceptedUserAccountID = &m.UserAccountID
		}
	}
	return nil
}

func (r *fakeInvitationRepo) IsActiveMemberByEmail(ctx context.Context, workspaceID uuid.UUID, email string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := workspaceID.String() + ":" + email
	return r.activeMemberMap[key], nil
}

// timeNow is used by fakes to get current time; substituted in tests that manipulate time.
var timeNow = func() time.Time { return time.Now().UTC() }

// ── fakeEmailSender ───────────────────────────────────────────────────────────

type fakeEmailSender struct {
	mu       sync.Mutex
	sent     []email.Message
	forceErr error
}

func (s *fakeEmailSender) Send(ctx context.Context, msg email.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.forceErr != nil {
		return s.forceErr
	}
	s.sent = append(s.sent, msg)
	return nil
}

func (s *fakeEmailSender) lastSent() *email.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.sent) == 0 {
		return nil
	}
	cp := s.sent[len(s.sent)-1]
	return &cp
}

func (s *fakeEmailSender) sendCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sent)
}

func newFakeMembershipRepo() *fakeMembershipRepo {
	return &fakeMembershipRepo{}
}

func (r *fakeMembershipRepo) FindActiveByWorkspaceAndAccount(ctx context.Context, workspaceID, accountID uuid.UUID) (*workspace.Membership, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, m := range r.memberships {
		if m.WorkspaceID == workspaceID && m.UserAccountID == accountID && m.MembershipStatusCode == workspace.MembershipStatusActive {
			cp := m
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *fakeMembershipRepo) ListActiveWorkspacesByAccount(ctx context.Context, accountID uuid.UUID) ([]workspace.WorkspaceWithRole, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []workspace.WorkspaceWithRole
	for _, m := range r.memberships {
		if m.UserAccountID == accountID && m.MembershipStatusCode == workspace.MembershipStatusActive {
			result = append(result, workspace.WorkspaceWithRole{
				Workspace:   workspace.Workspace{ID: m.WorkspaceID},
				OrgRoleCode: m.OrgRoleCode,
			})
		}
	}
	return result, nil
}

func (r *fakeMembershipRepo) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]workspace.Membership, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []workspace.Membership
	for _, m := range r.memberships {
		if m.WorkspaceID == workspaceID {
			result = append(result, m)
		}
	}
	return result, nil
}

// ── Helpers ────────────────────────────────────────────────────────────────────

func buildSvc() (*workspace.Service, *fakeWorkspaceRepo, *fakeMembershipRepo) {
	wsRepo := newFakeWorkspaceRepo()
	memberRepo := newFakeMembershipRepo()
	invRepo := newFakeInvitationRepo()
	emailer := &fakeEmailSender{}
	svc := workspace.NewService(wsRepo, memberRepo, invRepo, emailer, "http://localhost:13000/invitations/accept")
	return svc, wsRepo, memberRepo
}

func buildSvcFull() (*workspace.Service, *fakeWorkspaceRepo, *fakeMembershipRepo, *fakeInvitationRepo, *fakeEmailSender) {
	wsRepo := newFakeWorkspaceRepo()
	memberRepo := newFakeMembershipRepo()
	invRepo := newFakeInvitationRepo()
	emailer := &fakeEmailSender{}
	svc := workspace.NewService(wsRepo, memberRepo, invRepo, emailer, "http://localhost:13000/invitations/accept")
	return svc, wsRepo, memberRepo, invRepo, emailer
}

func newAccountID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := ids.New()
	if err != nil {
		t.Fatalf("ids.New: %v", err)
	}
	return id
}

// ── Tests ──────────────────────────────────────────────────────────────────────

func TestCreateWorkspace_Success(t *testing.T) {
	svc, wsRepo, _ := buildSvc()
	accountID := newAccountID(t)

	ws, err := svc.CreateWorkspace(context.Background(), workspace.CreateWorkspaceInput{
		AccountID:    accountID,
		Name:         "Acme Corp",
		Slug:         "acme-corp",
		ContactEmail: "admin@acme.com",
		IP:           "127.0.0.1",
		UserAgent:    "test-agent",
		RequestID:    "req-001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ws.WorkspaceName != "Acme Corp" {
		t.Errorf("name = %q, want Acme Corp", ws.WorkspaceName)
	}
	if ws.Slug != "acme-corp" {
		t.Errorf("slug = %q, want acme-corp", ws.Slug)
	}
	if ws.WorkspaceStatusCode != workspace.WorkspaceStatusActive {
		t.Errorf("status = %q, want active", ws.WorkspaceStatusCode)
	}
	if ws.OwnerUserAccountID != accountID {
		t.Errorf("owner = %v, want %v", ws.OwnerUserAccountID, accountID)
	}

	// Stored in repo.
	wsRepo.mu.Lock()
	_, exists := wsRepo.workspaces[ws.ID]
	wsRepo.mu.Unlock()
	if !exists {
		t.Error("workspace not found in repo after create")
	}

	// Owner membership built with org_role_code='owner'.
	if lastMembership.OrgRoleCode != workspace.OrgRoleOwner {
		t.Errorf("membership.org_role_code = %q, want owner", lastMembership.OrgRoleCode)
	}
	if lastMembership.UserAccountID != accountID {
		t.Errorf("membership.user_account_id = %v, want %v", lastMembership.UserAccountID, accountID)
	}
	if lastMembership.MembershipStatusCode != workspace.MembershipStatusActive {
		t.Errorf("membership.status = %q, want active", lastMembership.MembershipStatusCode)
	}

	// Audit entry built with action='workspace.create' and workspace_id set.
	if lastEntry.Action != workspace.AuditActionWorkspaceCreate {
		t.Errorf("audit.action = %q, want workspace.create", lastEntry.Action)
	}
	if lastEntry.WorkspaceID == nil || *lastEntry.WorkspaceID != ws.ID {
		t.Errorf("audit.workspace_id = %v, want %v", lastEntry.WorkspaceID, ws.ID)
	}
	if lastEntry.ActorAccountID == nil || *lastEntry.ActorAccountID != accountID {
		t.Errorf("audit.actor = %v, want %v", lastEntry.ActorAccountID, accountID)
	}
	if lastEntry.Result != workspace.AuditResultSuccess {
		t.Errorf("audit.result = %q, want success", lastEntry.Result)
	}
}

func TestCreateWorkspace_SlugReserved(t *testing.T) {
	svc, _, _ := buildSvc()
	accountID := newAccountID(t)

	for _, reserved := range []string{"admin", "api", "www", "auth", "health", "dashboard", "w", "workspace"} {
		_, err := svc.CreateWorkspace(context.Background(), workspace.CreateWorkspaceInput{
			AccountID:    accountID,
			Name:         "Test",
			Slug:         reserved,
			ContactEmail: "test@example.com",
		})
		if !errors.Is(err, workspace.ErrSlugReserved) {
			t.Errorf("slug %q: err = %v, want ErrSlugReserved", reserved, err)
		}
	}
}

func TestCreateWorkspace_SlugInvalid(t *testing.T) {
	svc, _, _ := buildSvc()
	accountID := newAccountID(t)

	// Note: the service normalises the slug with strings.ToLower before validation,
	// so uppercase input is accepted and normalised — not invalid. Single character
	// slugs are also valid per the regex. The cases below are genuinely invalid.
	cases := []struct{ slug string }{
		{"-leading"},
		{"trailing-"},
		{"has space"},
		{"has_underscore"},
		{""},
	}
	for _, tc := range cases {
		_, err := svc.CreateWorkspace(context.Background(), workspace.CreateWorkspaceInput{
			AccountID:    accountID,
			Name:         "Test",
			Slug:         tc.slug,
			ContactEmail: "test@example.com",
		})
		var valErr *workspace.ValidationError
		if !errors.As(err, &valErr) {
			t.Errorf("slug %q: err = %v, want ValidationError", tc.slug, err)
			continue
		}
		found := false
		for _, fe := range valErr.Fields {
			if fe.Field == "slug" {
				found = true
			}
		}
		if !found {
			t.Errorf("slug %q: no 'slug' field error in: %v", tc.slug, valErr.Fields)
		}
	}
}

func TestCreateWorkspace_NameValidation(t *testing.T) {
	svc, _, _ := buildSvc()
	accountID := newAccountID(t)

	// Empty name.
	_, err := svc.CreateWorkspace(context.Background(), workspace.CreateWorkspaceInput{
		AccountID:    accountID,
		Name:         "   ",
		Slug:         "valid-slug",
		ContactEmail: "test@example.com",
	})
	var valErr *workspace.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("empty name: err = %v, want ValidationError", err)
	}
	hasNameField := false
	for _, fe := range valErr.Fields {
		if fe.Field == "name" {
			hasNameField = true
		}
	}
	if !hasNameField {
		t.Errorf("expected 'name' field error, got: %v", valErr.Fields)
	}

	// Name too long (101 chars).
	longName := string(make([]byte, 101))
	for i := range []byte(longName) {
		longName = longName[:i] + "x" + longName[i+1:]
	}
	_, err = svc.CreateWorkspace(context.Background(), workspace.CreateWorkspaceInput{
		AccountID:    accountID,
		Name:         longName,
		Slug:         "valid-slug",
		ContactEmail: "test@example.com",
	})
	if !errors.As(err, &valErr) {
		t.Fatalf("long name: err = %v, want ValidationError", err)
	}
	hasNameField = false
	for _, fe := range valErr.Fields {
		if fe.Field == "name" {
			hasNameField = true
		}
	}
	if !hasNameField {
		t.Errorf("expected 'name' field error for long name, got: %v", valErr.Fields)
	}
}

func TestCreateWorkspace_EmailValidation(t *testing.T) {
	svc, _, _ := buildSvc()
	accountID := newAccountID(t)

	_, err := svc.CreateWorkspace(context.Background(), workspace.CreateWorkspaceInput{
		AccountID:    accountID,
		Name:         "Test",
		Slug:         "valid-slug",
		ContactEmail: "not-an-email",
	})
	var valErr *workspace.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
	found := false
	for _, fe := range valErr.Fields {
		if fe.Field == "contact_email" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'contact_email' field error: %v", valErr.Fields)
	}
}

func TestListMyWorkspaces(t *testing.T) {
	svc, _, memberRepo := buildSvc()
	accountID := newAccountID(t)
	wsID1, _ := ids.New()
	wsID2, _ := ids.New()

	memberRepo.mu.Lock()
	memberRepo.memberships = append(memberRepo.memberships,
		workspace.Membership{
			ID:                   uuid.Must(ids.New()),
			WorkspaceID:          wsID1,
			UserAccountID:        accountID,
			OrgRoleCode:          workspace.OrgRoleOwner,
			MembershipStatusCode: workspace.MembershipStatusActive,
		},
		workspace.Membership{
			ID:                   uuid.Must(ids.New()),
			WorkspaceID:          wsID2,
			UserAccountID:        accountID,
			OrgRoleCode:          workspace.OrgRoleUser,
			MembershipStatusCode: workspace.MembershipStatusActive,
		},
	)
	memberRepo.mu.Unlock()

	items, err := svc.ListMyWorkspaces(context.Background(), accountID)
	if err != nil {
		t.Fatalf("ListMyWorkspaces: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("got %d items, want 2", len(items))
	}
}

// ── 4b unit tests: InviteMember ───────────────────────────────────────────────

func newTenantContext(t *testing.T) workspace.TenantContext {
	t.Helper()
	wsID, err := ids.New()
	if err != nil {
		t.Fatalf("ids.New: %v", err)
	}
	accID, err := ids.New()
	if err != nil {
		t.Fatalf("ids.New: %v", err)
	}
	return workspace.TenantContext{
		WorkspaceID: wsID,
		AccountID:   accID,
		OrgRoleCode: workspace.OrgRoleOwner,
	}
}

func TestInviteMember_Success(t *testing.T) {
	_, _, _, invRepo, emailer := buildSvcFull()
	wsRepo := newFakeWorkspaceRepo()
	memberRepo := newFakeMembershipRepo()
	svc := workspace.NewService(wsRepo, memberRepo, invRepo, emailer, "http://localhost:13000/invitations/accept")

	tc := newTenantContext(t)

	result, err := svc.InviteMember(context.Background(), workspace.InviteMemberInput{
		TenantCtx:   tc,
		Email:       "invitee@example.com",
		OrgRoleCode: workspace.OrgRoleUser,
		IP:          "127.0.0.1",
		UserAgent:   "test",
		RequestID:   "req-invite-1",
	})
	if err != nil {
		t.Fatalf("InviteMember: %v", err)
	}
	if result.Invitation == nil {
		t.Fatal("result.Invitation is nil")
	}
	if result.Invitation.Email != "invitee@example.com" {
		t.Errorf("Email = %q, want invitee@example.com", result.Invitation.Email)
	}
	if result.Invitation.OrgRoleCode != workspace.OrgRoleUser {
		t.Errorf("OrgRoleCode = %q, want user", result.Invitation.OrgRoleCode)
	}
	if result.Invitation.WorkspaceID != tc.WorkspaceID {
		t.Errorf("WorkspaceID mismatch")
	}
	if result.Invitation.TokenHash == "" {
		t.Error("TokenHash must not be empty")
	}

	// Invitation stored in repo.
	invRepo.mu.Lock()
	storedCount := len(invRepo.invitations)
	invRepo.mu.Unlock()
	if storedCount != 1 {
		t.Errorf("invitations stored = %d, want 1", storedCount)
	}

	// Email sent once, to the invitee, body contains the accept link.
	if emailer.sendCount() != 1 {
		t.Errorf("email send count = %d, want 1", emailer.sendCount())
	}
	sent := emailer.lastSent()
	if sent.To != "invitee@example.com" {
		t.Errorf("email To = %q, want invitee@example.com", sent.To)
	}
	if !strings.Contains(sent.TextBody, "http://localhost:13000/invitations/accept?token=") {
		t.Errorf("email body missing accept link: %s", sent.TextBody)
	}
}

func TestInviteMember_OwnerRoleNotAllowed(t *testing.T) {
	_, _, _, invRepo, emailer := buildSvcFull()
	svc := workspace.NewService(newFakeWorkspaceRepo(), newFakeMembershipRepo(), invRepo, emailer, "http://localhost:13000/invitations/accept")

	tc := newTenantContext(t)
	_, err := svc.InviteMember(context.Background(), workspace.InviteMemberInput{
		TenantCtx:   tc,
		Email:       "owner@example.com",
		OrgRoleCode: workspace.OrgRoleOwner,
	})
	if !errors.Is(err, workspace.ErrInviteRoleNotAllowed) {
		t.Errorf("err = %v, want ErrInviteRoleNotAllowed", err)
	}
	if emailer.sendCount() != 0 {
		t.Error("no email should be sent on role error")
	}
}

func TestInviteMember_AlreadyActiveMember(t *testing.T) {
	_, _, _, invRepo, emailer := buildSvcFull()
	svc := workspace.NewService(newFakeWorkspaceRepo(), newFakeMembershipRepo(), invRepo, emailer, "http://localhost:13000/invitations/accept")

	tc := newTenantContext(t)
	// Pre-seed the fake as already-member.
	invRepo.mu.Lock()
	invRepo.activeMemberMap[tc.WorkspaceID.String()+":already@example.com"] = true
	invRepo.mu.Unlock()

	_, err := svc.InviteMember(context.Background(), workspace.InviteMemberInput{
		TenantCtx:   tc,
		Email:       "already@example.com",
		OrgRoleCode: workspace.OrgRoleUser,
	})
	if !errors.Is(err, workspace.ErrAlreadyMember) {
		t.Errorf("err = %v, want ErrAlreadyMember", err)
	}
}

func TestInviteMember_DuplicatePending(t *testing.T) {
	_, _, _, invRepo, emailer := buildSvcFull()
	svc := workspace.NewService(newFakeWorkspaceRepo(), newFakeMembershipRepo(), invRepo, emailer, "http://localhost:13000/invitations/accept")

	// Force repo to return ErrAlreadyPending on create.
	invRepo.forceCreateErr = workspace.ErrAlreadyPending

	tc := newTenantContext(t)
	_, err := svc.InviteMember(context.Background(), workspace.InviteMemberInput{
		TenantCtx:   tc,
		Email:       "pending@example.com",
		OrgRoleCode: workspace.OrgRoleAdmin,
	})
	if !errors.Is(err, workspace.ErrAlreadyPending) {
		t.Errorf("err = %v, want ErrAlreadyPending", err)
	}
}

// ── 4b unit tests: AcceptInvitation ──────────────────────────────────────────

func TestAcceptInvitation_Success(t *testing.T) {
	wsRepo := newFakeWorkspaceRepo()
	memberRepo := newFakeMembershipRepo()
	invRepo := newFakeInvitationRepo()
	emailer := &fakeEmailSender{}
	svc := workspace.NewService(wsRepo, memberRepo, invRepo, emailer, "http://localhost:13000/invitations/accept")

	tc := newTenantContext(t)
	inviteeID := uuid.Must(ids.New())
	inviteeEmail := "accept-me@example.com"

	// Invite first to get a stored invitation with a known hash.
	result, err := svc.InviteMember(context.Background(), workspace.InviteMemberInput{
		TenantCtx:   tc,
		Email:       inviteeEmail,
		OrgRoleCode: workspace.OrgRoleUser,
	})
	if err != nil {
		t.Fatalf("InviteMember: %v", err)
	}

	// Extract raw token from the email body.
	sent := emailer.lastSent()
	rawToken := extractToken(t, sent.TextBody)

	_, err = svc.AcceptInvitation(context.Background(), inviteeID, inviteeEmail, rawToken, "127.0.0.1", "test", "req-accept-1")
	if err != nil {
		t.Fatalf("AcceptInvitation: %v", err)
	}

	// Invitation should be marked accepted in the fake repo.
	invRepo.mu.Lock()
	inv := invRepo.invitations[0]
	invRepo.mu.Unlock()
	if inv.AcceptedAt == nil {
		t.Error("invitation.AcceptedAt should be set after accept")
	}
	if inv.AcceptedUserAccountID == nil || *inv.AcceptedUserAccountID != inviteeID {
		t.Errorf("AcceptedUserAccountID = %v, want %v", inv.AcceptedUserAccountID, inviteeID)
	}
	_ = result
}

func TestAcceptInvitation_EmailMismatch(t *testing.T) {
	wsRepo := newFakeWorkspaceRepo()
	memberRepo := newFakeMembershipRepo()
	invRepo := newFakeInvitationRepo()
	emailer := &fakeEmailSender{}
	svc := workspace.NewService(wsRepo, memberRepo, invRepo, emailer, "http://localhost:13000/invitations/accept")

	tc := newTenantContext(t)
	_, err := svc.InviteMember(context.Background(), workspace.InviteMemberInput{
		TenantCtx:   tc,
		Email:       "right@example.com",
		OrgRoleCode: workspace.OrgRoleUser,
	})
	if err != nil {
		t.Fatalf("InviteMember: %v", err)
	}

	sent := emailer.lastSent()
	rawToken := extractToken(t, sent.TextBody)

	wrongID := uuid.Must(ids.New())
	_, err = svc.AcceptInvitation(context.Background(), wrongID, "wrong@example.com", rawToken, "", "", "")
	if !errors.Is(err, workspace.ErrEmailMismatch) {
		t.Errorf("err = %v, want ErrEmailMismatch", err)
	}
}

func TestAcceptInvitation_InvalidToken(t *testing.T) {
	svc, _, _ := buildSvc()
	accID := uuid.Must(ids.New())
	_, err := svc.AcceptInvitation(context.Background(), accID, "any@example.com", "bogus-token", "", "", "")
	if !errors.Is(err, workspace.ErrInvitationInvalid) {
		t.Errorf("err = %v, want ErrInvitationInvalid", err)
	}
}

func TestAcceptInvitation_AlreadyMemberInWorkspace(t *testing.T) {
	wsRepo := newFakeWorkspaceRepo()
	memberRepo := newFakeMembershipRepo()
	invRepo := newFakeInvitationRepo()
	emailer := &fakeEmailSender{}
	svc := workspace.NewService(wsRepo, memberRepo, invRepo, emailer, "http://localhost:13000/invitations/accept")

	tc := newTenantContext(t)
	inviteeID := uuid.Must(ids.New())
	inviteeEmail := "member-twice@example.com"

	_, err := svc.InviteMember(context.Background(), workspace.InviteMemberInput{
		TenantCtx:   tc,
		Email:       inviteeEmail,
		OrgRoleCode: workspace.OrgRoleUser,
	})
	if err != nil {
		t.Fatalf("InviteMember: %v", err)
	}

	sent := emailer.lastSent()
	rawToken := extractToken(t, sent.TextBody)

	// Pre-seed membership so FindActiveByWorkspaceAndAccount returns a hit.
	memberRepo.mu.Lock()
	memberRepo.memberships = append(memberRepo.memberships, workspace.Membership{
		ID:                   uuid.Must(ids.New()),
		WorkspaceID:          tc.WorkspaceID,
		UserAccountID:        inviteeID,
		OrgRoleCode:          workspace.OrgRoleUser,
		MembershipStatusCode: workspace.MembershipStatusActive,
	})
	memberRepo.mu.Unlock()

	_, err = svc.AcceptInvitation(context.Background(), inviteeID, inviteeEmail, rawToken, "", "", "")
	if !errors.Is(err, workspace.ErrAlreadyMember) {
		t.Errorf("err = %v, want ErrAlreadyMember", err)
	}
}

// extractToken parses the raw token from the email text body.
// The body contains a line like: "http://...?token=<rawToken>"
func extractToken(t *testing.T, body string) string {
	t.Helper()
	idx := strings.Index(body, "?token=")
	if idx < 0 {
		t.Fatalf("no ?token= found in email body: %s", body)
	}
	// Token runs to end of line (or end of string).
	rest := body[idx+len("?token="):]
	end := strings.IndexAny(rest, "\r\n \t")
	if end >= 0 {
		return rest[:end]
	}
	return rest
}
