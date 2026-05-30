package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	httptransport "prasankit-api/internal/transport/http"
	"prasankit-api/internal/transport/http/health"
	"prasankit-api/internal/transport/http/middlewares"

	"github.com/gofiber/fiber/v3"
)

func TestHealthEndpoints(t *testing.T) {
	app := newTestHTTPApp(map[string]healthCheck{
		"minio":    {},
		"postgres": {},
		"redis":    {},
	})

	tests := []struct {
		name string
		path string
		want map[string]any
	}{
		{
			name: "legacy health",
			path: "/api/v1/health",
			want: map[string]any{
				"status": "ok",
			},
		},
		{
			name: "live health",
			path: "/api/v1/health/live",
			want: map[string]any{
				"status": "ok",
			},
		},
		{
			name: "ready health",
			path: "/api/v1/health/ready",
			want: map[string]any{
				"status": "ok",
				"checks": map[string]any{
					"api":      "ok",
					"minio":    "ok",
					"postgres": "ok",
					"redis":    "ok",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testRequest(t, app, http.MethodGet, tt.path)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status code = %d, want %d", resp.StatusCode, http.StatusOK)
			}

			var body map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			gotData, ok := body["data"].(map[string]any)
			if !ok {
				t.Fatalf("data envelope missing or invalid: %#v", body)
			}

			assertMapEqual(t, gotData, tt.want)
		})
	}
}

func TestReadyReportsPostgresFailure(t *testing.T) {
	app := newTestHTTPApp(map[string]healthCheck{
		"minio":    {},
		"postgres": {err: errors.New("postgres down")},
		"redis":    {},
	})

	resp := testRequest(t, app, http.MethodGet, "/api/v1/health/ready")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status code = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	gotError, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("error envelope missing or invalid: %#v", body)
	}

	if gotError["code"] != "SERVICE_UNAVAILABLE" {
		t.Fatalf("error.code = %v, want SERVICE_UNAVAILABLE", gotError["code"])
	}
}

func TestReadyReportsRedisFailure(t *testing.T) {
	app := newTestHTTPApp(map[string]healthCheck{
		"minio":    {},
		"postgres": {},
		"redis":    {err: errors.New("redis down")},
	})

	resp := testRequest(t, app, http.MethodGet, "/api/v1/health/ready")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status code = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	gotError, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("error envelope missing or invalid: %#v", body)
	}

	if gotError["code"] != "SERVICE_UNAVAILABLE" {
		t.Fatalf("error.code = %v, want SERVICE_UNAVAILABLE", gotError["code"])
	}
}

func TestReadyReportsMinIOFailure(t *testing.T) {
	app := newTestHTTPApp(map[string]healthCheck{
		"minio":    {err: errors.New("minio down")},
		"postgres": {},
		"redis":    {},
	})

	resp := testRequest(t, app, http.MethodGet, "/api/v1/health/ready")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status code = %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	gotError, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("error envelope missing or invalid: %#v", body)
	}

	if gotError["code"] != "SERVICE_UNAVAILABLE" {
		t.Fatalf("error.code = %v, want SERVICE_UNAVAILABLE", gotError["code"])
	}
}

func TestNotFoundUsesErrorEnvelope(t *testing.T) {
	app := newTestHTTPApp(map[string]healthCheck{
		"minio":    {},
		"postgres": {},
		"redis":    {},
	})

	resp := testRequest(t, app, http.MethodGet, "/api/v1/not-found")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if _, ok := body["data"]; ok {
		t.Fatalf("unexpected data envelope in error response: %#v", body)
	}

	gotError, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("error envelope missing or invalid: %#v", body)
	}

	if gotError["code"] != "NOT_FOUND" {
		t.Fatalf("error.code = %v, want NOT_FOUND", gotError["code"])
	}
}

type healthCheck struct {
	err error
}

func (c healthCheck) Run(ctx context.Context) error {
	return c.err
}

func newTestHTTPApp(checks map[string]healthCheck) *fiber.App {
	checkFuncs := make(map[string]health.CheckFunc, len(checks))
	for name, check := range checks {
		checkFuncs[name] = check.Run
	}

	app := fiber.New(fiber.Config{
		AppName:      "prasankit-api",
		ErrorHandler: middlewares.ErrorHandler,
	})

	httptransport.RegisterRoutes(
		app,
		health.NewHandler(checkFuncs),
		[]string{"http://localhost:3000"},
		nil, // authSvc — not needed for health tests
		nil, // authH — not needed for health tests
		nil, // workspaceH — not needed for health tests
		nil, // wsRepo — not needed for health tests
		nil, // memberRepo — not needed for health tests
		nil, // projectH — not needed for health tests
	)
	return app
}

func testRequest(t *testing.T, app *fiber.App, method string, path string) *http.Response {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, path, err)
	}

	return resp
}

func assertMapEqual(t *testing.T, got map[string]any, want map[string]any) {
	t.Helper()

	gotJSON, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal got: %v", err)
	}

	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal want: %v", err)
	}

	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("data = %s, want %s", gotJSON, wantJSON)
	}
}
