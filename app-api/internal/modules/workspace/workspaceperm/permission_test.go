package workspaceperm

import (
	"testing"

	"prasankit-api/internal/modules/workspace"
)

func TestCan(t *testing.T) {
	tests := []struct {
		name       string
		role       workspace.WorkspaceRole
		permission Permission
		want       bool
	}{
		{name: "owner can view", role: workspace.WorkspaceRoleOwner, permission: PermissionWorkspaceView, want: true},
		{name: "owner can manage", role: workspace.WorkspaceRoleOwner, permission: PermissionWorkspaceManage, want: true},
		{name: "admin can manage", role: workspace.WorkspaceRoleAdmin, permission: PermissionWorkspaceManage, want: true},
		{name: "executive can view", role: workspace.WorkspaceRoleExecutive, permission: PermissionWorkspaceView, want: true},
		{name: "executive cannot manage", role: workspace.WorkspaceRoleExecutive, permission: PermissionWorkspaceManage, want: false},
		{name: "user can view", role: workspace.WorkspaceRoleUser, permission: PermissionWorkspaceView, want: true},
		{name: "user cannot manage", role: workspace.WorkspaceRoleUser, permission: PermissionWorkspaceManage, want: false},
		{name: "unknown role denied", role: workspace.WorkspaceRole("unknown"), permission: PermissionWorkspaceView, want: false},
		{name: "unknown permission denied", role: workspace.WorkspaceRoleOwner, permission: Permission("unknown"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Can(tt.role, tt.permission); got != tt.want {
				t.Fatalf("Can(%q, %q) = %v, want %v", tt.role, tt.permission, got, tt.want)
			}
		})
	}
}
