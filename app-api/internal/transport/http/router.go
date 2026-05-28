package httptransport

import (
	"prasankit-api/internal/modules/auth"
	authhandler "prasankit-api/internal/transport/http/auth"
	"prasankit-api/internal/transport/http/health"
	"prasankit-api/internal/transport/http/middlewares"

	"github.com/gofiber/fiber/v3"
)

func RegisterRoutes(
	app *fiber.App,
	healthHandler health.Handler,
	corsAllowedOrigins []string,
	authSvc *auth.Service,
	authH *authhandler.Handler,
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

		// Protected auth routes.
		authGroup.Get("/me", middlewares.RequireSession(authSvc), authH.HandleMe)
	}
}
