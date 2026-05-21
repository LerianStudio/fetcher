package http

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"

	libLog "github.com/LerianStudio/lib-observability/log"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type recoverMiddleware struct {
	Logger libLog.Logger
}

type RecoverMiddlewareOption func(r *recoverMiddleware)

func WithRecoverLogger(logger libLog.Logger) RecoverMiddlewareOption {
	return func(r *recoverMiddleware) {
		r.Logger = logger
	}
}

func buildRecoverOpts(opts ...RecoverMiddlewareOption) *recoverMiddleware {
	mid := &recoverMiddleware{
		Logger: &libLog.GoLogger{},
	}
	for _, opt := range opts {
		opt(mid)
	}

	return mid
}

func WithRecover(opts ...RecoverMiddlewareOption) fiber.Handler {
	mid := buildRecoverOpts(opts...)

	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				reqCtx := c.UserContext()

				// Prefer the request-scoped logger from context when explicitly injected,
				// but fall back to the middleware logger (configured at startup) to ensure
				// panic logs are never silently swallowed by a nop logger.
				logger := mid.Logger
				if ctxLogger := observability.NewLoggerFromContext(reqCtx); ctxLogger != nil && mid.Logger == nil {
					logger = ctxLogger
				}

				stack := debug.Stack()
				panicErr := fmt.Errorf("panic recovered: %v", r)

				logger.Log(reqCtx, libLog.LevelError, fmt.Sprintf("Panic recovered: %v\nStack trace:\n%s", r, string(stack)))

				span := trace.SpanFromContext(reqCtx)
				if span.IsRecording() {
					span.RecordError(panicErr)
					span.SetStatus(codes.Error, fmt.Sprintf("Panic: %v", r))
				}

				internalErr := pkg.InternalServerError{
					Code:    constant.ErrInternalServer.Error(),
					Title:   "Internal Server Error",
					Message: "The server encountered an unexpected error. Please try again later or contact support.",
				}

				_ = c.Status(http.StatusInternalServerError).JSON(internalErr)
			}
		}()

		return c.Next()
	}
}
