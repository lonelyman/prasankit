package middlewares

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gofiber/fiber/v3"
)

const RequestIDHeader = "X-Request-Id"

func RequestID(c fiber.Ctx) error {
	requestID := c.Get(RequestIDHeader)
	if requestID == "" {
		requestID = newRequestID()
	}

	c.Set(RequestIDHeader, requestID)
	return c.Next()
}

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "request-id-unavailable"
	}
	return hex.EncodeToString(bytes[:])
}
