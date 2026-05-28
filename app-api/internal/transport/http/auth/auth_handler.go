package authhandler

import (
	"errors"
	"time"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/transport/http/presenter"

	"github.com/gofiber/fiber/v3"
)

// LocalsKeyAccount is the c.Locals key set by requireSession middleware.
const LocalsKeyAccount = "auth_account"

// Handler handles /auth routes.
type Handler struct {
	svc    *auth.Service
	apiEnv string
}

// NewHandler constructs a Handler.
func NewHandler(svc *auth.Service, apiEnv string) *Handler {
	return &Handler{svc: svc, apiEnv: apiEnv}
}

// accountResponse is the JSON shape returned for account data.
type accountResponse struct {
	ID                string `json:"id"`
	PrimaryEmail      string `json:"primary_email"`
	DisplayName       string `json:"display_name"`
	AccountStatusCode string `json:"account_status_code"`
}

func toAccountResponse(a auth.Account) accountResponse {
	return accountResponse{
		ID:                a.ID.String(),
		PrimaryEmail:      a.PrimaryEmail,
		DisplayName:       a.DisplayName,
		AccountStatusCode: a.AccountStatusCode,
	}
}

// signupRequest is the JSON body for POST /auth/signup.
type signupRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// loginRequest is the JSON body for POST /auth/login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// verifyEmailRequest is the JSON body for POST /auth/verify-email.
type verifyEmailRequest struct {
	Token string `json:"token"`
}

// resendVerificationRequest is the JSON body for POST /auth/verify-email/resend.
type resendVerificationRequest struct {
	Email string `json:"email"`
}

// passwordResetRequestRequest is the JSON body for POST /auth/password-reset/request.
type passwordResetRequestRequest struct {
	Email string `json:"email"`
}

// passwordResetConfirmRequest is the JSON body for POST /auth/password-reset/confirm.
type passwordResetConfirmRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// HandleSignup handles POST /auth/signup.
func (h *Handler) HandleSignup(c fiber.Ctx) error {
	var req signupRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}

	account, err := h.svc.Signup(c.Context(), auth.SignupInput{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return presenter.RenderItem(c, toAccountResponse(*account), fiber.StatusCreated)
}

// HandleLogin handles POST /auth/login.
func (h *Handler) HandleLogin(c fiber.Ctx) error {
	var req loginRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}

	out, err := h.svc.Login(c.Context(), auth.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		IP:        c.IP(),
		UserAgent: string(c.Request().Header.UserAgent()),
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}

	h.setSessionCookie(c, out.RawToken)
	return presenter.RenderItem(c, toAccountResponse(out.Account))
}

// HandleLogout handles POST /auth/logout.
func (h *Handler) HandleLogout(c fiber.Ctx) error {
	cookie := c.Cookies(auth.SessionCookieName)
	if cookie != "" {
		// Logout is a public route (no requireSession), so we pass a nil accountID;
		// session deletion is keyed by the cookie token, not the account.
		_ = h.svc.Logout(c.Context(), cookie, nil)
	}

	h.clearSessionCookie(c)
	return c.SendStatus(fiber.StatusNoContent)
}

// HandleMe handles GET /auth/me (behind requireSession).
func (h *Handler) HandleMe(c fiber.Ctx) error {
	account, ok := c.Locals(LocalsKeyAccount).(*auth.Account)
	if !ok || account == nil {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}
	return presenter.RenderItem(c, toAccountResponse(*account))
}

// HandleVerifyEmail handles POST /auth/verify-email.
func (h *Handler) HandleVerifyEmail(c fiber.Ctx) error {
	var req verifyEmailRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}
	if req.Token == "" {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Token is required")
	}

	if err := h.svc.ConfirmEmailVerification(c.Context(), req.Token); err != nil {
		return h.handleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// HandleResendVerification handles POST /auth/verify-email/resend.
func (h *Handler) HandleResendVerification(c fiber.Ctx) error {
	var req resendVerificationRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}

	if err := h.svc.ResendVerification(c.Context(), req.Email); err != nil {
		return h.handleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// HandleRequestPasswordReset handles POST /auth/password-reset/request.
// Always returns 204 to avoid email enumeration.
func (h *Handler) HandleRequestPasswordReset(c fiber.Ctx) error {
	var req passwordResetRequestRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}

	if err := h.svc.RequestPasswordReset(c.Context(), req.Email); err != nil {
		return h.handleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// HandleConfirmPasswordReset handles POST /auth/password-reset/confirm.
func (h *Handler) HandleConfirmPasswordReset(c fiber.Ctx) error {
	var req passwordResetConfirmRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}
	if req.Token == "" {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Token is required")
	}

	if err := h.svc.ConfirmPasswordReset(c.Context(), req.Token, req.NewPassword); err != nil {
		return h.handleServiceError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// handleServiceError maps service errors to HTTP responses.
func (h *Handler) handleServiceError(c fiber.Ctx, err error) error {
	var valErr *auth.ValidationError
	if errors.As(err, &valErr) {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Validation failed", valErr.Fields)
	}
	if errors.Is(err, auth.ErrEmailTaken) {
		return presenter.RenderError(c, fiber.StatusConflict, "auth.email_taken", "Email is already in use")
	}
	if errors.Is(err, auth.ErrInvalidCredentials) {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.invalid_credentials", "Invalid email or password")
	}
	if errors.Is(err, auth.ErrAccountLocked) {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.account_locked", "Account is temporarily locked")
	}
	if errors.Is(err, auth.ErrUnauthenticated) {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}
	if errors.Is(err, auth.ErrEmailNotVerified) {
		return presenter.RenderError(c, fiber.StatusForbidden, "auth.email_not_verified", "Email not verified")
	}
	if errors.Is(err, auth.ErrTokenInvalid) {
		return presenter.RenderError(c, fiber.StatusNotFound, "auth.token_invalid", "Token not found or invalid")
	}
	if errors.Is(err, auth.ErrTokenExpired) {
		return presenter.RenderError(c, fiber.StatusGone, "auth.token_expired", "Token has expired or already been used")
	}
	// Unexpected error — let the central error handler render 500.
	return err
}

// setSessionCookie writes the session cookie.
func (h *Handler) setSessionCookie(c fiber.Ctx, rawToken string) {
	secure := h.apiEnv == "production"
	c.Cookie(&fiber.Cookie{
		Name:     auth.SessionCookieName,
		Value:    rawToken,
		Path:     "/",
		HTTPOnly: true,
		Secure:   secure,
		SameSite: "Lax",
		MaxAge:   auth.SessionTTL * 60 * 60, // seconds
	})
}

// clearSessionCookie removes the session cookie.
func (h *Handler) clearSessionCookie(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}
