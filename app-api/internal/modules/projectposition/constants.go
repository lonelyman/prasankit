package projectposition

import (
	"regexp"

	"prasankit-api/internal/modules/workspace" // for OrgRole* codes in the permission map
)

// Audit action codes (D29 — Go consts, no FK, no DB CHECK on vocabulary).
const (
	AuditActionProjectPositionCreate    = "project_position.create"
	AuditActionProjectPositionUpdate    = "project_position.update"
	AuditActionProjectPositionDeprecate = "project_position.deprecate"

	AuditResourceTypeProjectPosition = "project_position"

	// AuditResultSuccess is a local copy of the success literal. Each module owns its own
	// audit vocabulary (do NOT import workspace just for this constant). MUST equal "success".
	AuditResultSuccess = "success"
)

// Permission codes (module-scoped `project_position:` prefix).
const (
	PermissionCreateProjectPosition    = "project_position:create"
	PermissionUpdateProjectPosition    = "project_position:update"
	PermissionDeprecateProjectPosition = "project_position:deprecate"
	PermissionReadProjectPosition      = "project_position:read"
)

// permissionAllowedRoles maps perm -> set of workspace.OrgRole* codes that pass the gate.
// Mirrors the 6a project map: write perms -> {Owner, Admin}; read -> all four org roles.
var permissionAllowedRoles = map[string]map[string]struct{}{
	PermissionCreateProjectPosition:    {workspace.OrgRoleOwner: {}, workspace.OrgRoleAdmin: {}},
	PermissionUpdateProjectPosition:    {workspace.OrgRoleOwner: {}, workspace.OrgRoleAdmin: {}},
	PermissionDeprecateProjectPosition: {workspace.OrgRoleOwner: {}, workspace.OrgRoleAdmin: {}},
	PermissionReadProjectPosition: {
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
