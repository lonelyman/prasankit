package workspaceperm

import "prasankit-api/internal/modules/workspace"

type Permission string

const (
	PermissionWorkspaceView   Permission = "workspace.view"
	PermissionWorkspaceManage Permission = "workspace.manage"
)

var rolePermissions = map[workspace.WorkspaceRole]map[Permission]struct{}{
	workspace.WorkspaceRoleOwner: {
		PermissionWorkspaceView:   {},
		PermissionWorkspaceManage: {},
	},
	workspace.WorkspaceRoleAdmin: {
		PermissionWorkspaceView:   {},
		PermissionWorkspaceManage: {},
	},
	workspace.WorkspaceRoleExecutive: {
		PermissionWorkspaceView: {},
	},
	workspace.WorkspaceRoleUser: {
		PermissionWorkspaceView: {},
	},
}

func Can(role workspace.WorkspaceRole, permission Permission) bool {
	permissions, ok := rolePermissions[role]
	if !ok {
		return false
	}
	_, ok = permissions[permission]
	return ok
}
