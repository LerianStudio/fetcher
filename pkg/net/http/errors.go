package http

import (
	"errors"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/gofiber/fiber/v2"
)

// WithError returns an error with the given status code and message.
func WithError(c *fiber.Ctx, err error) error {
	var validationErr pkg.ValidationError
	if errors.As(err, &validationErr) {
		return BadRequest(c, pkg.ValidationKnownFieldsError{
			Code:    validationErr.Code,
			Title:   validationErr.Title,
			Message: validationErr.Message,
			Fields:  nil,
		})
	}

	var unprocessableErr pkg.UnprocessableOperationError
	if errors.As(err, &unprocessableErr) {
		return UnprocessableEntity(c, unprocessableErr.Code, unprocessableErr.Title, unprocessableErr.Message)
	}

	var unauthorizedErr pkg.UnauthorizedError
	if errors.As(err, &unauthorizedErr) {
		return Unauthorized(c, unauthorizedErr.Code, unauthorizedErr.Title, unauthorizedErr.Message)
	}

	var forbiddenErr pkg.ForbiddenError
	if errors.As(err, &forbiddenErr) {
		return Forbidden(c, forbiddenErr.Code, forbiddenErr.Title, forbiddenErr.Message)
	}

	var knownFieldsErr pkg.ValidationKnownFieldsError
	if errors.As(err, &knownFieldsErr) {
		return BadRequest(c, knownFieldsErr)
	}

	var unknownFieldsErr pkg.ValidationUnknownFieldsError
	if errors.As(err, &unknownFieldsErr) {
		return BadRequest(c, unknownFieldsErr)
	}

	var responseErr pkg.ResponseError
	if errors.As(err, &responseErr) {
		return JSONResponseError(c, responseErr)
	}

	var responseErrWithStatus pkg.ResponseErrorWithStatusCode
	if errors.As(err, &responseErrWithStatus) {
		return JSONResponseErrorWithStatusCode(c, responseErrWithStatus)
	}

	var iErr pkg.InternalServerError
	if errors.As(pkg.ValidateInternalError(err, ""), &iErr) {
		return InternalServerError(c, iErr.Code, iErr.Title, iErr.Message)
	}

	return InternalServerError(c, "INTERNAL_ERROR", "Internal Server Error", err.Error())
}
