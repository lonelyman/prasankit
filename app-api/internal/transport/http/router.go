package httptransport

import (
	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/project"
	"prasankit-api/internal/modules/workspace"
	authhandler "prasankit-api/internal/transport/http/auth"
	"prasankit-api/internal/transport/http/health"
	"prasankit-api/internal/transport/http/middlewares"
	projecthandler "prasankit-api/internal/transport/http/project"
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
		}

		// Accept invitation — requireSession only (accepter is not yet a member).
		api.Post("/invitations/accept", requireSession, workspaceH.HandleAcceptInvite)
	}
}
