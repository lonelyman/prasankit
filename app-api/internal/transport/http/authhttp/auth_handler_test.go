package authhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/auth/authsvc"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type fakeRegistrar struct {
	registerResult *authsvc.RegisterEmailPasswordResult
	registerErr    error
	registerInput  authsvc.RegisterEmailPasswordInput
	verifyResult   *authsvc.VerifyEmailResult
	verifyErr      error
	verifyInput    authsvc.VerifyEmailInput
	resendResult   *authsvc.ResendVerificationEmailResult
	resendErr      error
	resendInput    authsvc.ResendVerificationEmailInput
	loginResult    *authsvc.LoginEmailPasswordResult
	loginErr       error
	loginInput     authsvc.LoginEmailPasswordInput
}

func (r *fakeRegistrar) RegisterEmailPassword(_ context.Context, input authsvc.RegisterEmailPasswordInput) (*authsvc.RegisterEmailPasswordResult, error) {
	r.registerInput = input
	if r.registerErr != nil {
		return nil, r.registerErr
	}
	return r.registerResult, nil
}

func (r *fakeRegistrar) VerifyEmail(_ context.Context, input authsvc.VerifyEmailInput) (*authsvc.VerifyEmailResult, error) {
	r.verifyInput = input
	if r.verifyErr != nil {
		return nil, r.verifyErr
	}
	return r.verifyResult, nil
}

func (r *fakeRegistrar) ResendVerificationEmail(_ context.Context, input authsvc.ResendVerificationEmailInput) (*authsvc.ResendVerificationEmailResult, error) {
	r.resendInput = input
	if r.resendErr != nil {
		return nil, r.resendErr
	}
	return r.resendResult, nil
}

func (r *fakeRegistrar) LoginEmailPassword(_ context.Context, input authsvc.LoginEmailPasswordInput) (*authsvc.LoginEmailPasswordResult, error) {
	r.loginInput = input
	if r.loginErr != nil {
		return nil, r.loginErr
	}
	return r.loginResult, nil
}

func TestRegisterEmailPassword(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	registrar := &fakeRegistrar{
		registerResult: &authsvc.RegisterEmailPasswordResult{
			Account: auth.UserAccount{
				ID:           accountID,
				PrimaryEmail: "owner@example.test",
				Status:       auth.UserAccountStatusPendingVerification,
			},
			VerificationEmailSent: true,
		},
	}
	app := newAuthTestApp(newTestHandler(registrar))

	resp := authTestRequest(t, app, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "owner@example.test",
		"password": "correct-password",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status code = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
	if registrar.registerInput.Email != "owner@example.test" {
		t.Fatalf("input.Email = %s, want owner@example.test", registrar.registerInput.Email)
	}
	if registrar.registerInput.Password != "correct-password" {
		t.Fatal("password was not passed to service")
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	data := body["data"].(map[string]any)
	accountData := data["account"].(map[string]any)
	if accountData["id"] != accountID.String() {
		t.Fatalf("account.id = %v, want %s", accountData["id"], accountID)
	}
	if accountData["primary_email"] != "owner@example.test" {
		t.Fatalf("account.primary_email = %v, want owner@example.test", accountData["primary_email"])
	}
	if _, ok := accountData["password"]; ok {
		t.Fatalf("response must not expose password: %#v", accountData)
	}
	if data["verification_email_sent"] != true {
		t.Fatalf("verification_email_sent = %v, want true", data["verification_email_sent"])
	}
}

func TestRegisterEmailPasswordRejectsInvalidJSON(t *testing.T) {
	app := newAuthTestApp(newTestHandler(&fakeRegistrar{}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertAuthError(t, resp, http.StatusBadRequest, "INVALID_REQUEST")
}

func TestRegisterEmailPasswordMapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "invalid email",
			err:        authsvc.ErrInvalidEmail,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_EMAIL",
		},
		{
			name:       "password too short",
			err:        authsvc.ErrPasswordTooShort,
			wantStatus: http.StatusBadRequest,
			wantCode:   "PASSWORD_TOO_SHORT",
		},
		{
			name:       "duplicate email",
			err:        auth.ErrEmailAlreadyRegistered,
			wantStatus: http.StatusConflict,
			wantCode:   "EMAIL_ALREADY_REGISTERED",
		},
		{
			name:       "verification email rate limited",
			err:        authsvc.ErrVerificationEmailRateLimited,
			wantStatus: http.StatusTooManyRequests,
			wantCode:   "VERIFICATION_EMAIL_RATE_LIMITED",
		},
		{
			name:       "verification email send failed",
			err:        authsvc.ErrVerificationEmailSendFailed,
			wantStatus: http.StatusBadGateway,
			wantCode:   "VERIFICATION_EMAIL_SEND_FAILED",
		},
		{
			name:       "unexpected error",
			err:        errors.New("database down"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL_SERVER_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newAuthTestApp(newTestHandler(&fakeRegistrar{registerErr: tt.err}))

			resp := authTestRequest(t, app, http.MethodPost, "/api/v1/auth/register", map[string]string{
				"email":    "owner@example.test",
				"password": "correct-password",
			})
			defer resp.Body.Close()

			assertAuthError(t, resp, tt.wantStatus, tt.wantCode)
		})
	}
}

func TestVerifyEmail(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	registrar := &fakeRegistrar{
		verifyResult: &authsvc.VerifyEmailResult{
			Account: auth.UserAccount{
				ID:           accountID,
				PrimaryEmail: "owner@example.test",
				Status:       auth.UserAccountStatusActive,
			},
		},
	}
	app := newAuthTestApp(newTestHandler(registrar))

	resp := authTestRequest(t, app, http.MethodPost, "/api/v1/auth/verify-email", map[string]string{
		"token": "verify-token",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if registrar.verifyInput.Token != "verify-token" {
		t.Fatalf("input.Token = %s, want verify-token", registrar.verifyInput.Token)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	data := body["data"].(map[string]any)
	accountData := data["account"].(map[string]any)
	if accountData["id"] != accountID.String() {
		t.Fatalf("account.id = %v, want %s", accountData["id"], accountID)
	}
	if accountData["status"] != string(auth.UserAccountStatusActive) {
		t.Fatalf("account.status = %v, want %s", accountData["status"], auth.UserAccountStatusActive)
	}
}

func TestVerifyEmailRejectsInvalidJSON(t *testing.T) {
	app := newAuthTestApp(newTestHandler(&fakeRegistrar{}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/verify-email", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertAuthError(t, resp, http.StatusBadRequest, "INVALID_REQUEST")
}

func TestVerifyEmailMapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "token required",
			err:        authsvc.ErrVerificationTokenRequired,
			wantStatus: http.StatusBadRequest,
			wantCode:   "VERIFICATION_TOKEN_REQUIRED",
		},
		{
			name:       "token invalid",
			err:        authsvc.ErrVerificationTokenInvalid,
			wantStatus: http.StatusBadRequest,
			wantCode:   "VERIFICATION_TOKEN_INVALID",
		},
		{
			name:       "token expired",
			err:        authsvc.ErrVerificationTokenExpired,
			wantStatus: http.StatusBadRequest,
			wantCode:   "VERIFICATION_TOKEN_EXPIRED",
		},
		{
			name:       "token already used",
			err:        authsvc.ErrVerificationTokenAlreadyUsed,
			wantStatus: http.StatusConflict,
			wantCode:   "VERIFICATION_TOKEN_ALREADY_USED",
		},
		{
			name:       "unexpected error",
			err:        errors.New("database down"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL_SERVER_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newAuthTestApp(newTestHandler(&fakeRegistrar{verifyErr: tt.err}))

			resp := authTestRequest(t, app, http.MethodPost, "/api/v1/auth/verify-email", map[string]string{
				"token": "verify-token",
			})
			defer resp.Body.Close()

			assertAuthError(t, resp, tt.wantStatus, tt.wantCode)
		})
	}
}

func TestResendVerificationEmail(t *testing.T) {
	registrar := &fakeRegistrar{
		resendResult: &authsvc.ResendVerificationEmailResult{
			VerificationEmailSent: true,
		},
	}
	app := newAuthTestApp(newTestHandler(registrar))

	resp := authTestRequest(t, app, http.MethodPost, "/api/v1/auth/resend-verification-email", map[string]string{
		"email": "owner@example.test",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if registrar.resendInput.Email != "owner@example.test" {
		t.Fatalf("input.Email = %s, want owner@example.test", registrar.resendInput.Email)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data := body["data"].(map[string]any)
	if data["status"] != "ok" {
		t.Fatalf("data.status = %v, want ok", data["status"])
	}
}

func TestResendVerificationEmailRejectsInvalidJSON(t *testing.T) {
	app := newAuthTestApp(newTestHandler(&fakeRegistrar{}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/resend-verification-email", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertAuthError(t, resp, http.StatusBadRequest, "INVALID_REQUEST")
}

func TestResendVerificationEmailMapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "invalid email",
			err:        authsvc.ErrInvalidEmail,
			wantStatus: http.StatusBadRequest,
			wantCode:   "INVALID_EMAIL",
		},
		{
			name:       "rate limited",
			err:        authsvc.ErrVerificationEmailRateLimited,
			wantStatus: http.StatusTooManyRequests,
			wantCode:   "VERIFICATION_EMAIL_RATE_LIMITED",
		},
		{
			name:       "email send failed",
			err:        authsvc.ErrVerificationEmailSendFailed,
			wantStatus: http.StatusBadGateway,
			wantCode:   "VERIFICATION_EMAIL_SEND_FAILED",
		},
		{
			name:       "unexpected error",
			err:        errors.New("database down"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL_SERVER_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newAuthTestApp(newTestHandler(&fakeRegistrar{resendErr: tt.err}))

			resp := authTestRequest(t, app, http.MethodPost, "/api/v1/auth/resend-verification-email", map[string]string{
				"email": "owner@example.test",
			})
			defer resp.Body.Close()

			assertAuthError(t, resp, tt.wantStatus, tt.wantCode)
		})
	}
}

func TestLoginEmailPassword(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	registrar := &fakeRegistrar{
		loginResult: &authsvc.LoginEmailPasswordResult{
			Account: auth.UserAccount{
				ID:           accountID,
				PrimaryEmail: "owner@example.test",
				Status:       auth.UserAccountStatusActive,
			},
			SessionID:        uuid.Must(uuid.NewV7()),
			SessionToken:     "raw-session-token",
			SessionExpiresAt: expiresAt,
		},
	}
	app := newAuthTestApp(newTestHandler(registrar))

	resp := authTestRequest(t, app, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    "owner@example.test",
		"password": "correct-password",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status code = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if registrar.loginInput.Email != "owner@example.test" {
		t.Fatalf("input.Email = %s, want owner@example.test", registrar.loginInput.Email)
	}
	if registrar.loginInput.Password != "correct-password" {
		t.Fatal("password was not passed to service")
	}

	cookies := resp.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies count = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != "prasankit_session" {
		t.Fatalf("cookie.Name = %s, want prasankit_session", cookie.Name)
	}
	if cookie.Value != "raw-session-token" {
		t.Fatalf("cookie.Value = %s, want raw-session-token", cookie.Value)
	}
	if !cookie.HttpOnly {
		t.Fatal("cookie.HttpOnly = false, want true")
	}
	if cookie.Path != "/" {
		t.Fatalf("cookie.Path = %s, want /", cookie.Path)
	}
	if !strings.Contains(resp.Header.Get("Set-Cookie"), "SameSite=Lax") {
		t.Fatalf("Set-Cookie missing SameSite=Lax: %s", resp.Header.Get("Set-Cookie"))
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	data := body["data"].(map[string]any)
	accountData := data["account"].(map[string]any)
	if accountData["id"] != accountID.String() {
		t.Fatalf("account.id = %v, want %s", accountData["id"], accountID)
	}
	if _, ok := data["session_token"]; ok {
		t.Fatal("response must not expose raw session token")
	}
}

func TestLoginEmailPasswordRejectsInvalidJSON(t *testing.T) {
	app := newAuthTestApp(newTestHandler(&fakeRegistrar{}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	assertAuthError(t, resp, http.StatusBadRequest, "INVALID_REQUEST")
}

func TestLoginEmailPasswordMapsServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "invalid credentials",
			err:        authsvc.ErrInvalidCredentials,
			wantStatus: http.StatusUnauthorized,
			wantCode:   "INVALID_CREDENTIALS",
		},
		{
			name:       "email not verified",
			err:        authsvc.ErrEmailNotVerified,
			wantStatus: http.StatusForbidden,
			wantCode:   "EMAIL_NOT_VERIFIED",
		},
		{
			name:       "account inactive",
			err:        authsvc.ErrAccountInactive,
			wantStatus: http.StatusForbidden,
			wantCode:   "ACCOUNT_INACTIVE",
		},
		{
			name:       "unexpected error",
			err:        errors.New("redis down"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL_SERVER_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newAuthTestApp(newTestHandler(&fakeRegistrar{loginErr: tt.err}))

			resp := authTestRequest(t, app, http.MethodPost, "/api/v1/auth/login", map[string]string{
				"email":    "owner@example.test",
				"password": "correct-password",
			})
			defer resp.Body.Close()

			assertAuthError(t, resp, tt.wantStatus, tt.wantCode)
		})
	}
}

func newTestHandler(service AuthService) Handler {
	return NewHandler(service, CookieConfig{
		Name:     "prasankit_session",
		TTL:      24 * time.Hour,
		SameSite: "Lax",
	})
}

func newAuthTestApp(handler Handler) *fiber.App {
	app := fiber.New()
	api := app.Group("/api/v1")
	handler.RegisterRoutes(api)
	return app
}

func authTestRequest(t *testing.T, app *fiber.App, method string, path string, body any) *http.Response {
	t.Helper()

	var requestBody bytes.Buffer
	if err := json.NewEncoder(&requestBody).Encode(body); err != nil {
		t.Fatalf("encode request body: %v", err)
	}

	req := httptest.NewRequest(method, path, &requestBody)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, path, err)
	}
	return resp
}

func assertAuthError(t *testing.T, resp *http.Response, wantStatus int, wantCode string) {
	t.Helper()

	if resp.StatusCode != wantStatus {
		t.Fatalf("status code = %d, want %d", resp.StatusCode, wantStatus)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	gotError, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("error envelope missing or invalid: %#v", body)
	}
	if gotError["code"] != wantCode {
		t.Fatalf("error.code = %v, want %s", gotError["code"], wantCode)
	}
}
