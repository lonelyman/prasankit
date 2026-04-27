package middlewares

import (
	"errors"
	"net/http"
	"strings"

	"prasankit-api/internal/transport/http/presenter"

	"github.com/gofiber/fiber/v3"
)

func ErrorHandler(c fiber.Ctx, err error) error {
	statusCode := fiber.StatusInternalServerError
	message := "An unexpected error occurred"

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		statusCode = fiberErr.Code
		message = fiberErr.Message
	}

	return presenter.RenderError(c, statusCode, errorCode(statusCode), message)
}

func errorCode(statusCode int) string {
	statusText := http.StatusText(statusCode)
	if statusText == "" {
		return "UNKNOWN_ERROR"
	}

	code := strings.ToUpper(statusText)
	code = strings.ReplaceAll(code, " ", "_")
	code = strings.ReplaceAll(code, "-", "_")

	return code
}
