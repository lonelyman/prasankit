package workspacedbrepo

import (
	"prasankit-api/internal/modules/workspace"
)

func workspaceToModel(ws workspace.Workspace) workspaceModel {
	return workspaceModel{
		ID:                  ws.ID,
		WorkspaceName:       ws.WorkspaceName,
		Slug:                ws.Slug,
		WorkspaceStatusCode: ws.WorkspaceStatusCode,
		ContactEmail:        ws.ContactEmail,
		OwnerUserAccountID:  ws.OwnerUserAccountID,
		CreatedAt:           ws.CreatedAt,
		CreatedBy:           ws.CreatedBy,
		UpdatedAt:           ws.UpdatedAt,
		UpdatedBy:           ws.UpdatedBy,
		PendingDeletionAt:   ws.PendingDeletionAt,
		DeletedAt:           ws.DeletedAt,
		DeletedBy:           ws.DeletedBy,
	}
}

func modelToWorkspace(m workspaceModel) workspace.Workspace {
	return workspace.Workspace{
		ID:                  m.ID,
		WorkspaceName:       m.WorkspaceName,
		Slug:                m.Slug,
		WorkspaceStatusCode: m.WorkspaceStatusCode,
		ContactEmail:        m.ContactEmail,
		OwnerUserAccountID:  m.OwnerUserAccountID,
		CreatedAt:           m.CreatedAt,
		CreatedBy:           m.CreatedBy,
		UpdatedAt:           m.UpdatedAt,
		UpdatedBy:           m.UpdatedBy,
		PendingDeletionAt:   m.PendingDeletionAt,
		DeletedAt:           m.DeletedAt,
		DeletedBy:           m.DeletedBy,
	}
}

func membershipToModel(m workspace.Membership) membershipModel {
	return membershipModel{
		ID:                     m.ID,
		WorkspaceID:            m.WorkspaceID,
		UserAccountID:          m.UserAccountID,
		OrgRoleCode:            m.OrgRoleCode,
		MembershipStatusCode:   m.MembershipStatusCode,
		InvitedByUserAccountID: m.InvitedByUserAccountID,
		JoinedAt:               m.JoinedAt,
		CreatedAt:              m.CreatedAt,
		CreatedBy:              m.CreatedBy,
		UpdatedAt:              m.UpdatedAt,
		UpdatedBy:              m.UpdatedBy,
	}
}

func modelToMembership(m membershipModel) workspace.Membership {
	return workspace.Membership{
		ID:                     m.ID,
		WorkspaceID:            m.WorkspaceID,
		UserAccountID:          m.UserAccountID,
		OrgRoleCode:            m.OrgRoleCode,
		MembershipStatusCode:   m.MembershipStatusCode,
		InvitedByUserAccountID: m.InvitedByUserAccountID,
		JoinedAt:               m.JoinedAt,
		CreatedAt:              m.CreatedAt,
		CreatedBy:              m.CreatedBy,
		UpdatedAt:              m.UpdatedAt,
		UpdatedBy:              m.UpdatedBy,
	}
}
