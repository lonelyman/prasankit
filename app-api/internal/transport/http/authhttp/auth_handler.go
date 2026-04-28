package authhttp

import (
	"context"
	"errors"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/auth/authsvc"
	"prasankit-api/internal/transport/http/presenter"

	"github.com/gofiber/fiber/v3"
)

type EmailPasswordRegistrar interface {
	RegisterEmailPassword(ctx context.Context, input authsvc.RegisterEmailPasswordInput) (*authsvc.RegisterEmailPasswordResult, error)
}

type Handler struct {
	registrar EmailPasswordRegistrar
}

type registerEmailPasswordRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerEmailPasswordResponse struct {
	Account               accountResponse `json:"account"`
	VerificationEmailSent bool            `json:"verification_email_sent"`
}

type accountResponse struct {
	ID           string `json:"id"`
	PrimaryEmail string `json:"primary_email"`
	Status       string `json:"status"`
}

func NewHandler(registrar EmailPasswordRegistrar) Handler {
	return Handler{
		registrar: registrar,
	}
}

func (h Handler) RegisterRoutes(router fiber.Router) {
	auth := router.Group("/auth")
	auth.Post("/register", h.RegisterEmailPassword)
	auth.Post("/verify-email", h.NotImplemented)
	auth.Post("/login", h.NotImplemented)
	auth.Post("/logout", h.NotImplemented)
	auth.Post("/logout-all", h.NotImplemented)
	auth.Post("/forgot-password", h.NotImplemented)
	auth.Post("/reset-password", h.NotImplemented)
	auth.Get("/me", h.NotImplemented)
}

func (h Handler) RegisterEmailPassword(c fiber.Ctx) error {
	if h.registrar == nil {
		return h.NotImplemented(c)
	}

	var req registerEmailPasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid")
	}

	result, err := h.registrar.RegisterEmailPassword(c.Context(), authsvc.RegisterEmailPasswordInput{
		Email:     req.Email,
		Password:  req.Password,
		IPAddress: c.IP(),
		UserAgent: c.UserAgent(),
	})
	if err != nil {
		return renderRegisterError(c, err)
	}

	return presenter.RenderItem(c, registerEmailPasswordResponse{
		Account: accountResponse{
			ID:           result.Account.ID.String(),
			PrimaryEmail: result.Account.PrimaryEmail,
			Status:       string(result.Account.Status),
		},
		VerificationEmailSent: result.VerificationEmailSent,
	}, fiber.StatusCreated)
}

func (h Handler) NotImplemented(c fiber.Ctx) error {
	return presenter.RenderError(c, fiber.StatusNotImplemented, "NOT_IMPLEMENTED", "Auth endpoint is not implemented yet")
}

func renderRegisterError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, authsvc.ErrInvalidEmail):
		return presenter.RenderError(c, fiber.StatusBadRequest, "INVALID_EMAIL", "Email is invalid")
	case errors.Is(err, authsvc.ErrPasswordTooShort):
		return presenter.RenderError(c, fiber.StatusBadRequest, "PASSWORD_TOO_SHORT", "Password is too short")
	case errors.Is(err, auth.ErrEmailAlreadyRegistered):
		return presenter.RenderError(c, fiber.StatusConflict, "EMAIL_ALREADY_REGISTERED", "Email is already registered")
	case errors.Is(err, authsvc.ErrVerificationEmailRateLimited):
		return presenter.RenderError(c, fiber.StatusTooManyRequests, "VERIFICATION_EMAIL_RATE_LIMITED", "Too many verification email requests")
	case errors.Is(err, authsvc.ErrVerificationEmailSendFailed):
		return presenter.RenderError(c, fiber.StatusBadGateway, "VERIFICATION_EMAIL_SEND_FAILED", "Verification email could not be sent")
	default:
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
}
