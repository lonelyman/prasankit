package middlewares

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

// CORS returns a configured CORS middleware. Origins must be specific (never wildcard).
// AllowCredentials is true as foundation for M1 cookie auth.
func CORS(allowedOrigins []string) fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowCredentials: true,
		// X-Workspace-Slug is the tenant header (D13) sent on every workspace-scoped
		// request; it MUST be allowed or the browser blocks those calls at CORS preflight
		// (Allow-Headers is a fixed list, not reflected). Without it, getCurrentWorkspace
		// and all project/member/position calls fail cross-origin.
		AllowHeaders:     []string{"Content-Type", "X-Workspace-Slug"},
	})
}
