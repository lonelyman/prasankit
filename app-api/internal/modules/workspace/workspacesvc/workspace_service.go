package workspacesvc

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/workspace"

	"github.com/google/uuid"
)

const (
	minSlugLength = 3
	maxSlugLength = 63
)

var (
	ErrWorkspaceNameRequired  = errors.New("workspace name is required")
	ErrWorkspaceSlugRequired  = errors.New("workspace slug is required")
	ErrWorkspaceSlugInvalid   = errors.New("workspace slug is invalid")
	ErrWorkspaceSlugReserved  = errors.New("workspace slug is reserved")
	ErrWorkspaceSlugTaken     = errors.New("workspace slug is already taken")
	ErrContactEmailInvalid    = errors.New("contact email is invalid")
	ErrAccountRequired        = errors.New("account is required")
	ErrAccountInactive        = errors.New("account is inactive")
	ErrMembershipCreateFailed = errors.New("workspace membership create failed")
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$`)

var reservedSlugs = map[string]struct{}{
	"admin":      {},
	"api":        {},
	"app":        {},
	"auth":       {},
	"dashboard":  {},
	"demo":       {},
	"help":       {},
	"login":      {},
	"logout":     {},
	"owner":      {},
	"platform":   {},
	"register":   {},
	"root":       {},
	"settings":   {},
	"support":    {},
	"system":     {},
	"workspace":  {},
	"workspaces": {},
	"www":        {},
}

type CheckSlugInput struct {
	Slug string
}

type CheckSlugResult struct {
	Slug      string
	Available bool
	Reason    string
}

type RegisterWorkspaceInput struct {
	Account      auth.UserAccount
	Name         string
	Slug         string
	ContactEmail string
	Mode         workspace.WorkspaceMode
}

type RegisterWorkspaceResult struct {
	Workspace  workspace.Workspace
	Membership workspace.Membership
}

type Service struct {
	repository workspace.Repository
}

func NewService(repository workspace.Repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) CheckSlug(ctx context.Context, input CheckSlugInput) (*CheckSlugResult, error) {
	slug, err := NormalizeSlug(input.Slug)
	if err != nil {
		return &CheckSlugResult{Slug: slug, Available: false, Reason: "invalid"}, nil
	}
	if IsReservedSlug(slug) {
		return &CheckSlugResult{Slug: slug, Available: false, Reason: "reserved"}, nil
	}

	_, err = s.repository.FindWorkspaceBySlug(ctx, slug)
	if errors.Is(err, workspace.ErrWorkspaceNotFound) {
		return &CheckSlugResult{Slug: slug, Available: true}, nil
	}
	if err != nil {
		return nil, err
	}

	return &CheckSlugResult{Slug: slug, Available: false, Reason: "taken"}, nil
}

func (s *Service) RegisterWorkspace(ctx context.Context, input RegisterWorkspaceInput) (*RegisterWorkspaceResult, error) {
	if input.Account.ID == uuid.Nil {
		return nil, ErrAccountRequired
	}
	if input.Account.Status != auth.UserAccountStatusActive {
		return nil, ErrAccountInactive
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrWorkspaceNameRequired
	}

	slug, err := NormalizeSlug(input.Slug)
	if err != nil {
		return nil, err
	}
	if IsReservedSlug(slug) {
		return nil, ErrWorkspaceSlugReserved
	}

	contactEmail, err := normalizeEmail(input.ContactEmail)
	if err != nil {
		return nil, ErrContactEmailInvalid
	}

	mode := input.Mode
	if mode == "" {
		mode = workspace.WorkspaceModeDemo
	}

	_, err = s.repository.FindWorkspaceBySlug(ctx, slug)
	if err == nil {
		return nil, ErrWorkspaceSlugTaken
	}
	if err != nil && !errors.Is(err, workspace.ErrWorkspaceNotFound) {
		return nil, err
	}

	now := time.Now().UTC()
	workspaceRecord := workspace.Workspace{
		Name:                  name,
		Slug:                  slug,
		Mode:                  mode,
		Status:                workspace.WorkspaceStatusActive,
		ContactEmail:          contactEmail,
		OwnerUserAccountID:    input.Account.ID,
		EmailVerifiedRequired: true,
		CreatedAt:             now,
		CreatedBy:             input.Account.ID,
		UpdatedAt:             now,
	}

	var membershipRecord workspace.Membership
	err = s.repository.WithinTransaction(ctx, func(ctx context.Context, repo workspace.Repository) error {
		if err := repo.CreateWorkspace(ctx, &workspaceRecord); err != nil {
			if errors.Is(err, workspace.ErrWorkspaceSlugAlreadyTaken) {
				return ErrWorkspaceSlugTaken
			}
			return err
		}

		membershipUserID := input.Account.ID
		membershipRecord = workspace.Membership{
			TenantID:      workspaceRecord.TenantID,
			WorkspaceID:   workspaceRecord.ID,
			UserAccountID: &membershipUserID,
			Role:          workspace.WorkspaceRoleOwner,
			Status:        workspace.MembershipStatusActive,
			JoinedAt:      &now,
			CreatedAt:     now,
			CreatedBy:     &input.Account.ID,
			UpdatedAt:     now,
			UpdatedBy:     &input.Account.ID,
		}
		if err := repo.CreateMembership(ctx, &membershipRecord); err != nil {
			return errors.Join(ErrMembershipCreateFailed, err)
		}
		return nil
	})
	if errors.Is(err, ErrWorkspaceSlugTaken) {
		return nil, ErrWorkspaceSlugTaken
	}
	if err != nil {
		return nil, err
	}

	return &RegisterWorkspaceResult{
		Workspace:  workspaceRecord,
		Membership: membershipRecord,
	}, nil
}

func NormalizeSlug(value string) (string, error) {
	slug := strings.ToLower(strings.TrimSpace(value))
	if slug == "" {
		return "", ErrWorkspaceSlugRequired
	}
	if len(slug) < minSlugLength || len(slug) > maxSlugLength {
		return slug, ErrWorkspaceSlugInvalid
	}
	if strings.Contains(slug, "--") || !slugPattern.MatchString(slug) {
		return slug, ErrWorkspaceSlugInvalid
	}
	return slug, nil
}

func IsReservedSlug(slug string) bool {
	_, ok := reservedSlugs[strings.ToLower(strings.TrimSpace(slug))]
	return ok
}

func normalizeEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	if email == "" || !strings.Contains(email, "@") || strings.Contains(email, " ") {
		return "", ErrContactEmailInvalid
	}
	return email, nil
}
