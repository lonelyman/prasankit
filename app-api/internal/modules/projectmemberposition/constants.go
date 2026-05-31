package projectmemberposition

import "prasankit-api/internal/modules/workspace" // for OrgRole* codes in the permission map

// Audit action codes (D29 — Go consts, no FK, no DB CHECK on vocabulary).
const (
	AuditActionPositionAssign   = "position.assign"
	AuditActionPositionUnassign = "position.unassign"

	AuditResourceTypeProjectMemberPosition = "project_member_position"

	// AuditResultSuccess is a local copy of the success literal. MUST equal "success".
	AuditResultSuccess = "success"
)

// Permission codes (module-scoped `project_member_position:` prefix).
const (
	PermissionAssignPosition   = "project_member_position:assign"
	PermissionUnassignPosition = "project_member_position:unassign"
	PermissionReadPosition     = "project_member_position:read"
)

// permissionAllowedRoles maps perm -> set of workspace.OrgRole* codes that pass the gate.
// Write perms -> {Owner, Admin}; read -> all four org roles.
var permissionAllowedRoles = map[string]map[string]struct{}{
	PermissionAssignPosition:   {workspace.OrgRoleOwner: {}, workspace.OrgRoleAdmin: {}},
	PermissionUnassignPosition: {workspace.OrgRoleOwner: {}, workspace.OrgRoleAdmin: {}},
	PermissionReadPosition: {
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
