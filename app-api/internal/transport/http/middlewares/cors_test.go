package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"prasankit-api/internal/transport/http/middlewares"

	"github.com/gofiber/fiber/v3"
)

// TestCORS_AllowsWorkspaceSlugHeader guards a regression where the CORS Allow-Headers
// listed only "Content-Type", omitting "X-Workspace-Slug". Because Allow-Headers is a
// fixed list (not reflected from the request), the browser blocked EVERY workspace-scoped
// call (getCurrentWorkspace + all project/member/position requests, which send the
// X-Workspace-Slug tenant header per D13) at CORS preflight.
func TestCORS_AllowsWorkspaceSlugHeader(t *testing.T) {
	app := fiber.New()
	app.Use(middlewares.CORS([]string{"http://localhost:13000"}))
	app.Get("/x", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	req := httptest.NewRequest(http.MethodOptions, "/x", nil)
	req.Header.Set("Origin", "http://localhost:13000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "x-workspace-slug")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	allow := resp.Header.Get("Access-Control-Allow-Headers")
	if !strings.Contains(allow, "X-Workspace-Slug") {
		t.Errorf("Access-Control-Allow-Headers = %q, want it to contain X-Workspace-Slug", allow)
	}
}

// TestCORS_EchoesAllowedOrigin confirms the configured origin is reflected with credentials,
// so cookie-authed cross-origin requests from the SPA are accepted.
func TestCORS_EchoesAllowedOrigin(t *testing.T) {
	app := fiber.New()
	app.Use(middlewares.CORS([]string{"http://localhost:13000"}))
	app.Get("/x", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "http://localhost:13000")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:13000" {
		t.Errorf("Access-Control-Allow-Origin = %q, want http://localhost:13000", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want true", got)
	}
}
