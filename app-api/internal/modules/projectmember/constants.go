package projectmember

import "prasankit-api/internal/modules/workspace"

// Audit action codes (D29 — Go consts, no FK, no DB CHECK on vocabulary).
const (
	AuditActionProjectMemberAdd        = "project_member.add"
	AuditActionProjectMemberRemove     = "project_member.remove"
	AuditActionProjectMemberRoleChange = "project_member.role_change"

	AuditResourceTypeProjectMember = "project_member"

	// AuditResultSuccess is a local copy of the success literal. Each module owns its
	// own audit vocabulary (do NOT import workspace just for this constant). MUST equal
	// "success" to match the existing audit_logs rows.
	AuditResultSuccess = "success"
)

// ProjectRoleOwnerCode is the canonical owner role literal, owned by THIS package.
// (`project` inlines the same literal to avoid importing `projectmember` — see the
//
//	project service's owner-seed code + the resolution note in the design call.)
const ProjectRoleOwnerCode = "project_owner"

// Permission codes (module-scoped `project_member:` prefix).
const (
	PermissionAddProjectMember        = "project_member:add"
	PermissionRemoveProjectMember     = "project_member:remove"
	PermissionChangeProjectMemberRole = "project_member:change_role"
	PermissionReadProjectMember       = "project_member:read"
)

// permissionAllowedRoles maps perm -> set of workspace.OrgRole* codes that pass the gate.
// Mirrors the 6a project map: write perms -> {Owner, Admin}; read -> all four org roles.
var permissionAllowedRoles = map[string]map[string]struct{}{
	PermissionAddProjectMember:        {workspace.OrgRoleOwner: {}, workspace.OrgRoleAdmin: {}},
	PermissionRemoveProjectMember:     {workspace.OrgRoleOwner: {}, workspace.OrgRoleAdmin: {}},
	PermissionChangeProjectMemberRole: {workspace.OrgRoleOwner: {}, workspace.OrgRoleAdmin: {}},
	PermissionReadProjectMember: {
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
