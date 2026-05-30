package project

import (
	"regexp"

	"prasankit-api/internal/modules/workspace"
)

// Audit action codes (D29 — Go consts, no FK, no DB CHECK on vocabulary).
const (
	AuditActionProjectCreate       = "project.create"
	AuditActionProjectUpdate       = "project.update"
	AuditActionProjectDelete       = "project.delete"
	AuditActionProjectStatusChange = "project.status_change"

	AuditResourceTypeProject = "project"

	// AuditResultSuccess is a local copy of the success literal. Each module owns its
	// own audit vocabulary (do NOT import workspace just for this constant). MUST equal
	// "success" to match the existing audit_logs rows from M1.
	AuditResultSuccess = "success"
)

// Permission codes (module-scoped `project:` prefix — explicit choice to avoid
// namespace collision with the workspace permission map, which uses `workspace:` for
// workspace-level actions).
const (
	PermissionCreateProject       = "project:create"
	PermissionUpdateProject       = "project:update"
	PermissionDeleteProject       = "project:delete"
	PermissionChangeProjectStatus = "project:change_status"
	PermissionReadProject         = "project:read"
)

// permissionAllowedRoles maps perm -> set of workspace.OrgRole* codes that pass the gate.
// Org-role only in 6a (no project_members exists yet).
var permissionAllowedRoles = map[string]map[string]struct{}{
	PermissionCreateProject:       {workspace.OrgRoleOwner: {}, workspace.OrgRoleAdmin: {}},
	PermissionUpdateProject:       {workspace.OrgRoleOwner: {}, workspace.OrgRoleAdmin: {}},
	PermissionDeleteProject:       {workspace.OrgRoleOwner: {}, workspace.OrgRoleAdmin: {}},
	PermissionChangeProjectStatus: {workspace.OrgRoleOwner: {}, workspace.OrgRoleAdmin: {}},
	PermissionReadProject: {
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

// slugPattern validates project slugs (same shape as workspace slugs):
// lowercase alphanumeric + hyphens, min 1 char, max 63 chars, no leading/trailing hyphens.
var slugPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$`)
