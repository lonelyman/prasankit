package httptransport

import (
	"prasankit-api/internal/transport/http/health"
	"prasankit-api/internal/transport/http/middlewares"

	"github.com/gofiber/fiber/v3"
)

func RegisterRoutes(app *fiber.App, healthHandler health.Handler, corsAllowedOrigins []string) {
	app.Use(middlewares.RequestID)
	app.Use(middlewares.CORS(corsAllowedOrigins))

	api := app.Group("/api/v1")
	api.Get("/health", healthHandler.Handle)
	api.Get("/health/live", healthHandler.HandleLive)
	api.Get("/health/ready", healthHandler.HandleReady)
}
