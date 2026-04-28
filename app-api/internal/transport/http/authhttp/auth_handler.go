package authhttp

import (
	"context"
	"errors"
	"time"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/auth/authsvc"
	"prasankit-api/internal/transport/http/presenter"

	"github.com/gofiber/fiber/v3"
)

type AuthService interface {
	RegisterEmailPassword(ctx context.Context, input authsvc.RegisterEmailPasswordInput) (*authsvc.RegisterEmailPasswordResult, error)
	VerifyEmail(ctx context.Context, input authsvc.VerifyEmailInput) (*authsvc.VerifyEmailResult, error)
	ResendVerificationEmail(ctx context.Context, input authsvc.ResendVerificationEmailInput) (*authsvc.ResendVerificationEmailResult, error)
	LoginEmailPassword(ctx context.Context, input authsvc.LoginEmailPasswordInput) (*authsvc.LoginEmailPasswordResult, error)
	CurrentAccount(ctx context.Context, input authsvc.CurrentAccountInput) (*authsvc.CurrentAccountResult, error)
	LogoutCurrentSession(ctx context.Context, input authsvc.LogoutCurrentSessionInput) (*authsvc.LogoutCurrentSessionResult, error)
}

const accountLocalKey = "auth.account"

type Handler struct {
	auth   AuthService
	cookie CookieConfig
}

type CookieConfig struct {
	Name     string
	TTL      time.Duration
	Secure   bool
	SameSite string
}

type registerEmailPasswordRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerEmailPasswordResponse struct {
	Account               accountResponse `json:"account"`
	VerificationEmailSent bool            `json:"verification_email_sent"`
}

type verifyEmailRequest struct {
	Token string `json:"token"`
}

type verifyEmailResponse struct {
	Account accountResponse `json:"account"`
}

type resendVerificationEmailRequest struct {
	Email string `json:"email"`
}

type resendVerificationEmailResponse struct {
	Status string `json:"status"`
}

type loginEmailPasswordRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginEmailPasswordResponse struct {
	Account          accountResponse `json:"account"`
	SessionExpiresAt string          `json:"session_expires_at"`
}

type currentAccountResponse struct {
	Account accountResponse `json:"account"`
}

type logoutResponse struct {
	Status string `json:"status"`
}

type accountResponse struct {
	ID           string `json:"id"`
	PrimaryEmail string `json:"primary_email"`
	Status       string `json:"status"`
}

func NewHandler(auth AuthService, cookie CookieConfig) Handler {
	return Handler{
		auth:   auth,
		cookie: cookie,
	}
}

func (h Handler) RegisterRoutes(router fiber.Router) {
	auth := router.Group("/auth")
	auth.Post("/register", h.RegisterEmailPassword)
	auth.Post("/verify-email", h.VerifyEmail)
	auth.Post("/resend-verification-email", h.ResendVerificationEmail)
	auth.Post("/login", h.LoginEmailPassword)
	auth.Post("/logout", h.Logout)
	auth.Post("/logout-all", h.NotImplemented)
	auth.Post("/forgot-password", h.NotImplemented)
	auth.Post("/reset-password", h.NotImplemented)
	auth.Get("/me", h.requireSession, h.Me)
}

func (h Handler) RegisterEmailPassword(c fiber.Ctx) error {
	if h.auth == nil {
		return h.NotImplemented(c)
	}

	var req registerEmailPasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid")
	}

	result, err := h.auth.RegisterEmailPassword(c.Context(), authsvc.RegisterEmailPasswordInput{
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

func (h Handler) VerifyEmail(c fiber.Ctx) error {
	if h.auth == nil {
		return h.NotImplemented(c)
	}

	var req verifyEmailRequest
	if err := c.Bind().Body(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid")
	}

	result, err := h.auth.VerifyEmail(c.Context(), authsvc.VerifyEmailInput{
		Token:     req.Token,
		IPAddress: c.IP(),
		UserAgent: c.UserAgent(),
	})
	if err != nil {
		return renderVerifyEmailError(c, err)
	}

	return presenter.RenderItem(c, verifyEmailResponse{
		Account: accountResponse{
			ID:           result.Account.ID.String(),
			PrimaryEmail: result.Account.PrimaryEmail,
			Status:       string(result.Account.Status),
		},
	})
}

func (h Handler) ResendVerificationEmail(c fiber.Ctx) error {
	if h.auth == nil {
		return h.NotImplemented(c)
	}

	var req resendVerificationEmailRequest
	if err := c.Bind().Body(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid")
	}

	_, err := h.auth.ResendVerificationEmail(c.Context(), authsvc.ResendVerificationEmailInput{
		Email:     req.Email,
		IPAddress: c.IP(),
		UserAgent: c.UserAgent(),
	})
	if err != nil {
		return renderResendVerificationEmailError(c, err)
	}

	return presenter.RenderItem(c, resendVerificationEmailResponse{
		Status: "ok",
	})
}

func (h Handler) LoginEmailPassword(c fiber.Ctx) error {
	if h.auth == nil {
		return h.NotImplemented(c)
	}

	var req loginEmailPasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Request body is invalid")
	}

	result, err := h.auth.LoginEmailPassword(c.Context(), authsvc.LoginEmailPasswordInput{
		Email:     req.Email,
		Password:  req.Password,
		IPAddress: c.IP(),
		UserAgent: c.UserAgent(),
	})
	if err != nil {
		return renderLoginError(c, err)
	}

	h.setSessionCookie(c, result.SessionToken, result.SessionExpiresAt)

	return presenter.RenderItem(c, loginEmailPasswordResponse{
		Account: accountResponse{
			ID:           result.Account.ID.String(),
			PrimaryEmail: result.Account.PrimaryEmail,
			Status:       string(result.Account.Status),
		},
		SessionExpiresAt: result.SessionExpiresAt.Format(time.RFC3339),
	})
}

func (h Handler) Me(c fiber.Ctx) error {
	account, ok := c.Locals(accountLocalKey).(auth.UserAccount)
	if !ok {
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}

	return presenter.RenderItem(c, currentAccountResponse{
		Account: accountResponse{
			ID:           account.ID.String(),
			PrimaryEmail: account.PrimaryEmail,
			Status:       string(account.Status),
		},
	})
}

func (h Handler) Logout(c fiber.Ctx) error {
	if h.auth == nil {
		return h.NotImplemented(c)
	}

	_, err := h.auth.LogoutCurrentSession(c.Context(), authsvc.LogoutCurrentSessionInput{
		SessionToken: c.Cookies(h.cookieName()),
		IPAddress:    c.IP(),
		UserAgent:    c.UserAgent(),
	})
	if err != nil {
		return renderSessionError(c, err)
	}

	h.clearSessionCookie(c)

	return presenter.RenderItem(c, logoutResponse{
		Status: "ok",
	})
}

func (h Handler) requireSession(c fiber.Ctx) error {
	if h.auth == nil {
		return h.NotImplemented(c)
	}

	result, err := h.auth.CurrentAccount(c.Context(), authsvc.CurrentAccountInput{
		SessionToken: c.Cookies(h.cookieName()),
		IPAddress:    c.IP(),
		UserAgent:    c.UserAgent(),
	})
	if err != nil {
		return renderSessionError(c, err)
	}

	c.Locals(accountLocalKey, result.Account)
	return c.Next()
}

func (h Handler) NotImplemented(c fiber.Ctx) error {
	return presenter.RenderError(c, fiber.StatusNotImplemented, "NOT_IMPLEMENTED", "Auth endpoint is not implemented yet")
}

func (h Handler) cookieName() string {
	if h.cookie.Name == "" {
		return "prasankit_session"
	}
	return h.cookie.Name
}

func (h Handler) setSessionCookie(c fiber.Ctx, value string, expiresAt time.Time) {
	maxAge := 0
	if h.cookie.TTL > 0 {
		maxAge = int(h.cookie.TTL.Seconds())
	}

	sameSite := h.cookie.SameSite
	if sameSite == "" {
		sameSite = "Lax"
	}

	c.Cookie(&fiber.Cookie{
		Name:     h.cookieName(),
		Value:    value,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   maxAge,
		Secure:   h.cookie.Secure,
		HTTPOnly: true,
		SameSite: sameSite,
	})
}

func (h Handler) clearSessionCookie(c fiber.Ctx) {
	sameSite := h.cookie.SameSite
	if sameSite == "" {
		sameSite = "Lax"
	}

	c.Cookie(&fiber.Cookie{
		Name:     h.cookieName(),
		Value:    "",
		Path:     "/",
		Expires:  time.Now().UTC().Add(-time.Hour),
		MaxAge:   -1,
		Secure:   h.cookie.Secure,
		HTTPOnly: true,
		SameSite: sameSite,
	})
}

func renderSessionError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, authsvc.ErrSessionTokenRequired):
		return presenter.RenderError(c, fiber.StatusUnauthorized, "AUTH_SESSION_REQUIRED", "Authentication session is required")
	case errors.Is(err, authsvc.ErrSessionInvalid):
		return presenter.RenderError(c, fiber.StatusUnauthorized, "AUTH_SESSION_INVALID", "Authentication session is invalid")
	case errors.Is(err, authsvc.ErrSessionExpired):
		return presenter.RenderError(c, fiber.StatusUnauthorized, "AUTH_SESSION_EXPIRED", "Authentication session is expired")
	case errors.Is(err, authsvc.ErrAccountInactive):
		return presenter.RenderError(c, fiber.StatusForbidden, "ACCOUNT_INACTIVE", "Account is inactive")
	default:
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
}

func renderLoginError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, authsvc.ErrInvalidCredentials):
		return presenter.RenderError(c, fiber.StatusUnauthorized, "INVALID_CREDENTIALS", "Email or password is invalid")
	case errors.Is(err, authsvc.ErrEmailNotVerified):
		return presenter.RenderError(c, fiber.StatusForbidden, "EMAIL_NOT_VERIFIED", "Email is not verified")
	case errors.Is(err, authsvc.ErrAccountInactive):
		return presenter.RenderError(c, fiber.StatusForbidden, "ACCOUNT_INACTIVE", "Account is inactive")
	default:
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
}

func renderResendVerificationEmailError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, authsvc.ErrInvalidEmail):
		return presenter.RenderError(c, fiber.StatusBadRequest, "INVALID_EMAIL", "Email is invalid")
	case errors.Is(err, authsvc.ErrVerificationEmailRateLimited):
		return presenter.RenderError(c, fiber.StatusTooManyRequests, "VERIFICATION_EMAIL_RATE_LIMITED", "Too many verification email requests")
	case errors.Is(err, authsvc.ErrVerificationEmailSendFailed):
		return presenter.RenderError(c, fiber.StatusBadGateway, "VERIFICATION_EMAIL_SEND_FAILED", "Verification email could not be sent")
	default:
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
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

func renderVerifyEmailError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, authsvc.ErrVerificationTokenRequired):
		return presenter.RenderError(c, fiber.StatusBadRequest, "VERIFICATION_TOKEN_REQUIRED", "Verification token is required")
	case errors.Is(err, authsvc.ErrVerificationTokenInvalid):
		return presenter.RenderError(c, fiber.StatusBadRequest, "VERIFICATION_TOKEN_INVALID", "Verification token is invalid")
	case errors.Is(err, authsvc.ErrVerificationTokenExpired):
		return presenter.RenderError(c, fiber.StatusBadRequest, "VERIFICATION_TOKEN_EXPIRED", "Verification token is expired")
	case errors.Is(err, authsvc.ErrVerificationTokenAlreadyUsed):
		return presenter.RenderError(c, fiber.StatusConflict, "VERIFICATION_TOKEN_ALREADY_USED", "Verification token is already used")
	default:
		return presenter.RenderError(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred")
	}
}
