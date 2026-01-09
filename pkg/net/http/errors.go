package http

import (
	"github.com/LerianStudio/fetcher/pkg"
	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"
)

// WithError returns an error with the given status code and message.
func WithError(c *fiber.Ctx, err error) error {
	switch e := err.(type) {
	case pkg.ValidationError:
		return BadRequest(c, pkg.ValidationKnownFieldsError{
			Code:    e.Code,
			Title:   e.Title,
			Message: e.Message,
			Fields:  nil,
		})
	case pkg.UnprocessableOperationError:
		return UnprocessableEntity(c, e.Code, e.Title, e.Message)
	case pkg.UnauthorizedError:
		return Unauthorized(c, e.Code, e.Title, e.Message)
	case pkg.ForbiddenError:
		return Forbidden(c, e.Code, e.Title, e.Message)
	case pkg.ValidationKnownFieldsError, pkg.ValidationUnknownFieldsError:
		return BadRequest(c, e)
	case pkg.ResponseError:
		var rErr pkg.ResponseError

		_ = errors.As(err, &rErr)

		return JSONResponseError(c, rErr)
	case pkg.ResponseErrorWithStatusCode:
		var rErr pkg.ResponseErrorWithStatusCode

		_ = errors.As(err, &rErr)

		return JSONResponseErrorWithStatusCode(c, rErr)
	default:
		var iErr pkg.InternalServerError

		_ = errors.As(pkg.ValidateInternalError(err, ""), &iErr)

		return InternalServerError(c, iErr.Code, iErr.Title, iErr.Message)
	}
}
