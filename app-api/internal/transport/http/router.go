package httptransport

import (
	"prasankit-api/internal/transport/http/authhttp"
	"prasankit-api/internal/transport/http/health"
	"prasankit-api/internal/transport/http/middlewares"
	"prasankit-api/internal/transport/http/workspacehttp"

	"github.com/gofiber/fiber/v3"
)

func RegisterRoutes(app *fiber.App, healthHandler health.Handler, authHandler authhttp.Handler, workspaceHandler workspacehttp.Handler) {
	app.Use(middlewares.RequestID)

	api := app.Group("/api/v1")
	api.Get("/health", healthHandler.Handle)
	api.Get("/health/live", healthHandler.HandleLive)
	api.Get("/health/ready", healthHandler.HandleReady)
	authHandler.RegisterRoutes(api)
	workspaceHandler.RegisterRoutes(api)
}
