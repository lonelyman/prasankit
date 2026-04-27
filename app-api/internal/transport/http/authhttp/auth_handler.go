package authhttp

import (
	"prasankit-api/internal/transport/http/presenter"

	"github.com/gofiber/fiber/v3"
)

type Handler struct{}

func NewHandler() Handler {
	return Handler{}
}

func (h Handler) RegisterRoutes(router fiber.Router) {
	auth := router.Group("/auth")
	auth.Post("/register", h.NotImplemented)
	auth.Post("/verify-email", h.NotImplemented)
	auth.Post("/login", h.NotImplemented)
	auth.Post("/logout", h.NotImplemented)
	auth.Post("/logout-all", h.NotImplemented)
	auth.Post("/forgot-password", h.NotImplemented)
	auth.Post("/reset-password", h.NotImplemented)
	auth.Get("/me", h.NotImplemented)
}

func (h Handler) NotImplemented(c fiber.Ctx) error {
	return presenter.RenderError(c, fiber.StatusNotImplemented, "NOT_IMPLEMENTED", "Auth endpoint is not implemented yet")
}
