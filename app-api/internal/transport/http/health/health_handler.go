package health

import (
	"context"
	"time"

	"prasankit-api/internal/transport/http/presenter"

	"github.com/gofiber/fiber/v3"
)

const readinessCheckTimeout = 2 * time.Second

type CheckFunc func(context.Context) error

type Handler struct {
	checks map[string]CheckFunc
}

func NewHandler(checks map[string]CheckFunc) Handler {
	copiedChecks := make(map[string]CheckFunc, len(checks))
	for name, check := range checks {
		copiedChecks[name] = check
	}

	return Handler{
		checks: copiedChecks,
	}
}

func (h Handler) Handle(c fiber.Ctx) error {
	return h.HandleLive(c)
}

func (h Handler) HandleLive(c fiber.Ctx) error {
	return presenter.RenderItem(c, map[string]any{
		"status": "ok",
	}, fiber.StatusOK)
}

func (h Handler) HandleReady(c fiber.Ctx) error {
	checks := map[string]string{
		"api": "ok",
	}

	ready := true
	for name, check := range h.checks {
		if check == nil {
			checks[name] = "not_configured"
			ready = false
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), readinessCheckTimeout)
		err := check(ctx)
		cancel()

		if err != nil {
			checks[name] = "error"
			ready = false
			continue
		}

		checks[name] = "ok"
	}

	if !ready {
		return presenter.RenderError(c, fiber.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service is not ready", map[string]any{
			"status": "not_ready",
			"checks": checks,
		})
	}

	return presenter.RenderItem(c, map[string]any{
		"status": "ok",
		"checks": checks,
	}, fiber.StatusOK)
}
