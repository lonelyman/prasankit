package presenter

import (
	"github.com/gofiber/fiber/v3"
)

type SuccessResponse struct {
	Data any `json:"data"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func RenderItem(c fiber.Ctx, data any, statusCode ...int) error {
	status := fiber.StatusOK
	if len(statusCode) > 0 {
		status = statusCode[0]
	}

	return c.Status(status).JSON(SuccessResponse{
		Data: data,
	})
}

func RenderError(c fiber.Ctx, statusCode int, code string, message string, details ...any) error {
	var detail any
	if len(details) > 0 {
		detail = details[0]
	}

	return c.Status(statusCode).JSON(ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: detail,
		},
	})
}
