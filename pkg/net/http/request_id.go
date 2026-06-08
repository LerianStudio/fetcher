package http

import (
	"github.com/LerianStudio/lib-observability"
	"github.com/gofiber/fiber/v2"
)

// RequestIDHeader add requestId to response header
//
// - Read requestId from context using: observability.NewTrackingFromContext(ctx)
//
// - If exists, write to response header (ex: X-Request-Id)
func RequestIDHeader(headerName string) fiber.Handler {
	if headerName == "" {
		headerName = "X-Request-Id"
	}

	return func(c *fiber.Ctx) error {
		err := c.Next()

		ctx := c.UserContext()

		_, _, requestId, _ := observability.NewTrackingFromContext(ctx)

		if requestId != "" {
			c.Set(headerName, requestId)
		}

		return err
	}
}
