package middlewares

import (
	"prasankit-api/internal/modules/auth"
	authhandler "prasankit-api/internal/transport/http/auth"
	"prasankit-api/internal/transport/http/presenter"

	"github.com/gofiber/fiber/v3"
)

// RequireSession middleware validates the session cookie, looks up the session in
// Redis, and loads the account fresh from Postgres (D16 — no stale payload).
// On success it stores *auth.Account in c.Locals(LocalsKeyAccount).
// TTL is absolute — no sliding (no LastSeenAt update on every request).
func RequireSession(svc *auth.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		rawToken := c.Cookies(auth.SessionCookieName)
		if rawToken == "" {
			return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
		}

		// Resolve session → fresh account (D16), with lazy session revocation
		// (a session created before password_changed_at is invalidated).
		account, err := svc.ResolveSessionAccount(c.Context(), rawToken)
		if err != nil || account == nil {
			return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
		}

		c.Locals(authhandler.LocalsKeyAccount, account)
		return c.Next()
	}
}
