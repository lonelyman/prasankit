package authhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"prasankit-api/internal/modules/auth"
	"prasankit-api/internal/modules/auth/authsvc"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type fakeRegistrar struct {
	result *authsvc.RegisterEmailPasswordResult
	err    error
	input  authsvc.RegisterEmailPasswordInput
}

func (r *fakeRegistrar) RegisterEmailPassword(_ context.Context, input authsvc.RegisterEmailPasswordInput) (*authsvc.RegisterEmailPasswordResult, error) {
	r.input = input
	if r.err != nil {
		return nil, r.err
	}
	return r.result, nil
}

func TestRegisterEmailPassword(t *testing.T) {
	accountID := uuid.Must(uuid.NewV7())
	registrar := &fakeRegistrar{
		result: &authsvc.RegisterEmailPasswordResult{
			Account: auth.UserAccount{
				ID:           accountID,
				PrimaryEmail: "owner@example.test",
				Status:       auth.UserAccountStatusPendingVerification,
			},
			VerificationEmailSent: true,
		},
	}
	app := newAuthTestApp(NewHandler(registrar))

	resp := authTestRequest(t, app, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "owner@example.test",
		"password": "correct-password",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status code = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
	if registrar.input.Email != "owner@example.test" {
		t.Fatalf("input.Email = %s, want owner@example.test", registrar.input.Email)
	}
	if registrar.input.Password != "correct-password" {
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
	app := newAuthTestApp(NewHandler(&fakeRegistrar{}))

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
			app := newAuthTestApp(NewHandler(&fakeRegistrar{err: tt.err}))

			resp := authTestRequest(t, app, http.MethodPost, "/api/v1/auth/register", map[string]string{
				"email":    "owner@example.test",
				"password": "correct-password",
			})
			defer resp.Body.Close()

			assertAuthError(t, resp, tt.wantStatus, tt.wantCode)
		})
	}
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
