package httptransport

import (
	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/workspace"
	authhandler "prasankit-api/internal/transport/http/auth"
	"prasankit-api/internal/transport/http/health"
	"prasankit-api/internal/transport/http/middlewares"
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

		// Accept invitation — requireSession only (accepter is not yet a member).
		api.Post("/invitations/accept", requireSession, workspaceH.HandleAcceptInvite)
	}
}
