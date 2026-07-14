package in

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/LerianStudio/fetcher/v2/pkg"
	httpUtils "github.com/LerianStudio/fetcher/v2/pkg/net/http"

	"github.com/gofiber/fiber/v2"
)

// problemMiddlewareChain wraps a Fiber middleware that signals failure by
// short-circuiting with a written response. A marker distinguishes that from a
// downstream Huma problem returning through c.Next, preventing double writes.
// The non-zero marker key keeps nested middleware chains isolated.
func problemMiddlewareChain(middleware fiber.Handler) []fiber.Handler {
	markerKey := new(byte)

	wrapped := func(c *fiber.Ctx) error {
		err := middleware(c)
		if passed, _ := c.Locals(markerKey).(bool); passed {
			return err
		}

		status := c.Response().StatusCode()
		message := strings.TrimSpace(string(c.Response().Body()))
		code := ""

		if err != nil {
			message = err.Error()

			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) {
				status = fiberErr.Code
			}
		}

		if status < http.StatusBadRequest {
			return err
		}

		var payload struct {
			Code    json.RawMessage `json:"code"`
			Message string          `json:"message"`
			Detail  string          `json:"detail"`
		}
		if jsonErr := json.Unmarshal(c.Response().Body(), &payload); jsonErr == nil {
			if payload.Message != "" {
				message = payload.Message
			} else if payload.Detail != "" {
				message = payload.Detail
			}

			if len(payload.Code) > 0 {
				if stringErr := json.Unmarshal(payload.Code, &code); stringErr != nil {
					code = strings.Trim(string(payload.Code), `"`)
				}
			}
		}

		if message == "" {
			message = http.StatusText(status)
		}

		c.Response().ResetBody()

		return httpUtils.WithError(c, pkg.ResponseErrorWithStatusCode{
			StatusCode: status,
			Code:       code,
			Message:    message,
		})
	}

	marker := func(c *fiber.Ctx) error {
		c.Locals(markerKey, true)
		return c.Next()
	}

	return []fiber.Handler{wrapped, marker}
}
