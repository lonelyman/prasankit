package httptransport

import (
	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/companyposition"
	"prasankit-api/internal/modules/project"
	"prasankit-api/internal/modules/projectmember"
	"prasankit-api/internal/modules/projectmemberposition"
	"prasankit-api/internal/modules/projectposition"
	"prasankit-api/internal/modules/workspace"
	authhandler "prasankit-api/internal/transport/http/auth"
	companypositionhandler "prasankit-api/internal/transport/http/companyposition"
	"prasankit-api/internal/transport/http/health"
	"prasankit-api/internal/transport/http/middlewares"
	projecthandler "prasankit-api/internal/transport/http/project"
	projectmemberhandler "prasankit-api/internal/transport/http/projectmember"
	projectmemberpositionhandler "prasankit-api/internal/transport/http/projectmemberposition"
	projectpositionhandler "prasankit-api/internal/transport/http/projectposition"
	workspacehandler "prasankit-api/internal/transport/http/workspace"

	"github.com/gofiber/fiber/v3"
)

func RegisterRoutes(
	app *fiber.App,
	healthHandler health.Handler,
	corsAllowedOrigins []string,
	authSvc *auth.Service,
	authH *authhandler.Handler,
	workspaceH *workspacehandler.Handler,
	wsRepo workspace.WorkspaceRepository,
	memberRepo workspace.MembershipRepository,
	projectH *projecthandler.Handler,
	projectMemberH *projectmemberhandler.Handler,
	projectPositionH *projectpositionhandler.Handler,
	companyPositionH *companypositionhandler.Handler,
	projectMemberPositionH *projectmemberpositionhandler.Handler,
	companyPositionAttachH *companypositionhandler.Handler, // same companyposition handler exposes set/clear
) {
	app.Use(middlewares.RequestID)
	app.Use(middlewares.CORS(corsAllowedOrigins))

	api := app.Group("/api/v1")

	// Health (no auth).
	api.Get("/health", healthHandler.Handle)
	api.Get("/health/live", healthHandler.HandleLive)
	api.Get("/health/ready", healthHandler.HandleReady)

	// Auth routes — skipped when handler is nil (health-only test setup).
	if authH != nil && authSvc != nil {
		authGroup := api.Group("/auth")
		authGroup.Post("/signup", authH.HandleSignup)
		authGroup.Post("/login", authH.HandleLogin)
		authGroup.Post("/logout", authH.HandleLogout)
		authGroup.Post("/verify-email", authH.HandleVerifyEmail)
		authGroup.Post("/verify-email/resend", authH.HandleResendVerification)
		authGroup.Post("/password-reset/request", authH.HandleRequestPasswordReset)
		authGroup.Post("/password-reset/confirm", authH.HandleConfirmPasswordReset)

		// Protected auth routes.
		authGroup.Get("/me", middlewares.RequireSession(authSvc), authH.HandleMe)
	}

	// Workspace routes — skipped when handler is nil.
	if workspaceH != nil && authSvc != nil && wsRepo != nil && memberRepo != nil {
		requireSession := middlewares.RequireSession(authSvc)
		requireTenant := middlewares.RequireTenantContext(wsRepo, memberRepo)
		requireInvite := middlewares.RequireWorkspacePermission(workspace.PermissionInviteMember)

		wsGroup := api.Group("/workspaces")
		// requireSession only.
		wsGroup.Post("/", requireSession, workspaceH.HandleCreate)
		wsGroup.Get("/", requireSession, workspaceH.HandleList)
		// requireSession + requireTenantContext.
		wsGroup.Get("/current", requireSession, requireTenant, workspaceH.HandleGetCurrent)
		// requireSession + requireTenantContext + requireWorkspacePermission(invite).
		wsGroup.Post("/invitations", requireSession, requireTenant, requireInvite, workspaceH.HandleInvite)
		// Add-member picker: list active workspace members with display_name.
		// Same gating as invite (owner/admin) — read-only.
		wsGroup.Get("/members", requireSession, requireTenant, requireInvite, workspaceH.HandleListMembers)

		// project_positions (ws-scoped master) — own guard (NOT under projectH).
		if projectPositionH != nil {
			requireCreateProjPos := middlewares.RequireProjectPositionPermission(projectposition.PermissionCreateProjectPosition)
			requireReadProjPos := middlewares.RequireProjectPositionPermission(projectposition.PermissionReadProjectPosition)
			requireUpdateProjPos := middlewares.RequireProjectPositionPermission(projectposition.PermissionUpdateProjectPosition)
			requireDeprecateProjPos := middlewares.RequireProjectPositionPermission(projectposition.PermissionDeprecateProjectPosition)

			wsGroup.Post("/project-positions", requireSession, requireTenant, requireCreateProjPos, projectPositionH.HandleCreate)
			wsGroup.Get("/project-positions", requireSession, requireTenant, requireReadProjPos, projectPositionH.HandleList)
			wsGroup.Put("/project-positions/:code", requireSession, requireTenant, requireUpdateProjPos, projectPositionH.HandleUpdate)
			wsGroup.Post("/project-positions/:code/deprecate", requireSession, requireTenant, requireDeprecateProjPos, projectPositionH.HandleDeprecate)
		}

		// company_positions (ws-scoped master) — own guard.
		if companyPositionH != nil {
			requireCreateCompPos := middlewares.RequireCompanyPositionPermission(companyposition.PermissionCreateCompanyPosition)
			requireReadCompPos := middlewares.RequireCompanyPositionPermission(companyposition.PermissionReadCompanyPosition)
			requireUpdateCompPos := middlewares.RequireCompanyPositionPermission(companyposition.PermissionUpdateCompanyPosition)
			requireDeprecateCompPos := middlewares.RequireCompanyPositionPermission(companyposition.PermissionDeprecateCompanyPosition)

			wsGroup.Post("/company-positions", requireSession, requireTenant, requireCreateCompPos, companyPositionH.HandleCreate)
			wsGroup.Get("/company-positions", requireSession, requireTenant, requireReadCompPos, companyPositionH.HandleList)
			wsGroup.Put("/company-positions/:code", requireSession, requireTenant, requireUpdateCompPos, companyPositionH.HandleUpdate)
			wsGroup.Post("/company-positions/:code/deprecate", requireSession, requireTenant, requireDeprecateCompPos, companyPositionH.HandleDeprecate)
		}

		// ws company_position attach (ws-scoped) — reuses RequireCompanyPositionPermission(assign).
		if companyPositionAttachH != nil {
			requireAssignCompPos := middlewares.RequireCompanyPositionPermission(companyposition.PermissionAssignCompanyPosition)
			wsGroup.Put("/memberships/:membershipId/company-position", requireSession, requireTenant, requireAssignCompPos, companyPositionAttachH.HandleSetCompanyPosition)
		}

		// Project routes — registered when projectH is wired.
		if projectH != nil {
			requireCreateProj := middlewares.RequireProjectPermission(project.PermissionCreateProject)
			requireUpdateProj := middlewares.RequireProjectPermission(project.PermissionUpdateProject)
			requireDeleteProj := middlewares.RequireProjectPermission(project.PermissionDeleteProject)
			requireStatusProj := middlewares.RequireProjectPermission(project.PermissionChangeProjectStatus)
			requireReadProj := middlewares.RequireProjectPermission(project.PermissionReadProject)

			wsGroup.Post("/projects", requireSession, requireTenant, requireCreateProj, projectH.HandleCreate)
			wsGroup.Get("/projects", requireSession, requireTenant, requireReadProj, projectH.HandleList)
			wsGroup.Get("/projects/:id", requireSession, requireTenant, requireReadProj, projectH.HandleGet)
			wsGroup.Put("/projects/:id", requireSession, requireTenant, requireUpdateProj, projectH.HandleUpdate)
			wsGroup.Delete("/projects/:id", requireSession, requireTenant, requireDeleteProj, projectH.HandleDelete)
			wsGroup.Post("/projects/:id/status", requireSession, requireTenant, requireStatusProj, projectH.HandleChangeStatus)

			// Project member routes — registered when projectMemberH is wired.
			if projectMemberH != nil {
				requireAddMember := middlewares.RequireProjectMemberPermission(projectmember.PermissionAddProjectMember)
				requireReadMember := middlewares.RequireProjectMemberPermission(projectmember.PermissionReadProjectMember)
				requireChangeMemberRole := middlewares.RequireProjectMemberPermission(projectmember.PermissionChangeProjectMemberRole)
				requireRemoveMember := middlewares.RequireProjectMemberPermission(projectmember.PermissionRemoveProjectMember)

				wsGroup.Post("/projects/:id/members", requireSession, requireTenant, requireAddMember, projectMemberH.HandleAdd)
				wsGroup.Get("/projects/:id/members", requireSession, requireTenant, requireReadMember, projectMemberH.HandleList)
				wsGroup.Put("/projects/:id/members/:memberId/role", requireSession, requireTenant, requireChangeMemberRole, projectMemberH.HandleChangeRole)
				wsGroup.Delete("/projects/:id/members/:memberId", requireSession, requireTenant, requireRemoveMember, projectMemberH.HandleRemove)
			}

			// Project member position (junction) routes — sibling to projectMemberH.
			if projectMemberPositionH != nil {
				requireAssignPos := middlewares.RequireProjectMemberPositionPermission(projectmemberposition.PermissionAssignPosition)
				requireReadPos := middlewares.RequireProjectMemberPositionPermission(projectmemberposition.PermissionReadPosition)
				requireUnassignPos := middlewares.RequireProjectMemberPositionPermission(projectmemberposition.PermissionUnassignPosition)

				wsGroup.Post("/projects/:id/members/:memberId/positions", requireSession, requireTenant, requireAssignPos, projectMemberPositionH.HandleAssign)
				wsGroup.Get("/projects/:id/members/:memberId/positions", requireSession, requireTenant, requireReadPos, projectMemberPositionH.HandleList)
				wsGroup.Delete("/projects/:id/members/:memberId/positions/:code", requireSession, requireTenant, requireUnassignPos, projectMemberPositionH.HandleUnassign)
			}
		}

		// Accept invitation — requireSession only (accepter is not yet a member).
		api.Post("/invitations/accept", requireSession, workspaceH.HandleAcceptInvite)
	}
}
