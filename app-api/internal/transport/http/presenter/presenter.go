package presenter

import (
	"math"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

const (
	defaultLimit = 10
	maxLimit     = 100
)

type OffsetQuery struct {
	Page   int
	Limit  int
	Offset int
}

func ParseOffsetQuery(c fiber.Ctx) OffsetQuery {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.Query("limit", strconv.Itoa(defaultLimit)))
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	return OffsetQuery{
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
}

type SuccessResponse struct {
	Data any `json:"data"`
}

type ListPayload struct {
	Items      any `json:"items"`
	Pagination any `json:"pagination,omitempty"`
}

type OffsetPagination struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasMore    bool `json:"has_more"`
}

func NewOffsetPagination(total int, limit int, offset int) OffsetPagination {
	if limit <= 0 {
		limit = defaultLimit
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages < 1 {
		totalPages = 1
	}

	page := (offset / limit) + 1

	return OffsetPagination{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasMore:    offset+limit < total,
	}
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

func RenderList(c fiber.Ctx, items any, pagination ...any) error {
	var pg any
	if len(pagination) > 0 {
		pg = pagination[0]
	}

	return c.Status(fiber.StatusOK).JSON(SuccessResponse{
		Data: ListPayload{
			Items:      items,
			Pagination: pg,
		},
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
