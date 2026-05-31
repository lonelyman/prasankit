package companyposition

import (
	"regexp"

	"prasankit-api/internal/modules/workspace" // for OrgRole* codes in the permission map
)

// Audit action codes (D29 — Go consts, no FK, no DB CHECK on vocabulary).
const (
	AuditActionCompanyPositionCreate    = "company_position.create"
	AuditActionCompanyPositionUpdate    = "company_position.update"
	AuditActionCompanyPositionDeprecate = "company_position.deprecate"

	AuditResourceTypeCompanyPosition = "company_position"

	// ws company_position attach — single action, OldValue/NewValue carry old/new company_position_code
	// (null NewValue = clear); mirrors project_member.role_change (one action distinguishes set/clear/change).
	AuditActionMembershipCompanyPositionChange = "membership.company_position_change"
	AuditResourceTypeWorkspaceMembership       = "workspace_membership"

	// AuditResultSuccess is a local copy of the success literal. Each module owns its own
	// audit vocabulary (do NOT import workspace just for this constant). MUST equal "success".
	AuditResultSuccess = "success"
)

// Permission codes (module-scoped `company_position:` prefix).
const (
	PermissionCreateCompanyPosition    = "company_position:create"
	PermissionUpdateCompanyPosition    = "company_position:update"
	PermissionDeprecateCompanyPosition = "company_position:deprecate"
	PermissionReadCompanyPosition      = "company_position:read"

	// ws company_position attach (set/clear a membership's company position).
	PermissionAssignCompanyPosition = "company_position:assign"
)

// permissionAllowedRoles maps perm -> set of workspace.OrgRole* codes that pass the gate.
// Mirrors the 6a project map: write perms -> {Owner, Admin}; read -> all four org roles.
var permissionAllowedRoles = map[string]map[string]struct{}{
	PermissionCreateCompanyPosition:    {workspace.OrgRoleOwner: {}, workspace.OrgRoleAdmin: {}},
	PermissionUpdateCompanyPosition:    {workspace.OrgRoleOwner: {}, workspace.OrgRoleAdmin: {}},
	PermissionDeprecateCompanyPosition: {workspace.OrgRoleOwner: {}, workspace.OrgRoleAdmin: {}},
	PermissionAssignCompanyPosition:    {workspace.OrgRoleOwner: {}, workspace.OrgRoleAdmin: {}},
	PermissionReadCompanyPosition: {
		workspace.OrgRoleOwner:     {},
		workspace.OrgRoleAdmin:     {},
		workspace.OrgRoleExecutive: {},
		workspace.OrgRoleUser:      {},
	},
}

// IsRoleAllowedFor returns true when the given org_role_code is permitted to perform perm.
func IsRoleAllowedFor(perm, orgRoleCode string) bool {
	set, ok := permissionAllowedRoles[perm]
	if !ok {
		return false
	}
	_, ok = set[orgRoleCode]
	return ok
}

// codePattern validates position codes (§2.9): lowercase, starts with a letter, [a-z0-9_], 2..63 chars.
var codePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,62}$`)
