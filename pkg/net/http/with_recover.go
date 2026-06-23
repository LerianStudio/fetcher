package http

import (
	"context"
	"net/http"

	observability "github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"

	libLog "github.com/LerianStudio/lib-observability/log"
	obsRuntime "github.com/LerianStudio/lib-observability/runtime"

	"github.com/gofiber/fiber/v2"
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

				recordHTTPPanic(reqCtx, logger, r)

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

func recordHTTPPanic(ctx context.Context, logger libLog.Logger, recovered any) {
	defer obsRuntime.RecoverWithPolicyAndContext(ctx, logger, "http", "fiber-recover-middleware", obsRuntime.KeepRunning)

	panic(recovered)
}
