// Package workspace contains the domain and service layer for workspace/tenancy.
// Domain structs and interfaces are pure — no gorm tags, no fiber imports.
package workspace

// Org role codes — FK by code, must match org_roles.code seed.
const (
	OrgRoleOwner     = "owner"
	OrgRoleAdmin     = "admin"
	OrgRoleExecutive = "executive"
	OrgRoleUser      = "user"
)

// Membership status codes — FK by code, must match membership_statuses.code seed.
const (
	MembershipStatusActive    = "active"
	MembershipStatusSuspended = "suspended"
	MembershipStatusRemoved   = "removed"
)

// Workspace status codes — FK by code, must match workspace_statuses.code seed.
const (
	WorkspaceStatusActive          = "active"
	WorkspaceStatusSuspended       = "suspended"
	WorkspaceStatusPendingDeletion = "pending_deletion"
	WorkspaceStatusDeleted         = "deleted"
)

// Audit action codes (machine codes, §4.4 — no DB FK, controlled by domain constant).
const (
	AuditActionWorkspaceCreate = "workspace.create"
)

// Audit resource types.
const (
	AuditResourceTypeWorkspace = "workspace"
)

// Audit result codes.
const (
	AuditResultSuccess = "success"
	AuditResultFailure = "failure"
)

// reservedSlugs is the set of slugs that may never be used as workspace identifiers.
var reservedSlugs = map[string]struct{}{
	"api":       {},
	"admin":     {},
	"app":       {},
	"www":       {},
	"system":    {},
	"static":    {},
	"assets":    {},
	"status":    {},
	"health":    {},
	"auth":      {},
	"login":     {},
	"signup":    {},
	"settings":  {},
	"dashboard": {},
	"w":         {},
	"workspace": {},
}

// IsReservedSlug returns true when the given slug (already lower-cased) is reserved.
func IsReservedSlug(slug string) bool {
	_, ok := reservedSlugs[slug]
	return ok
}
