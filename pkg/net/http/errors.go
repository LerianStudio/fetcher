package http

import (
	"errors"
	stdhttp "net/http"
	"sort"
	"strconv"

	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/constant"
	"github.com/LerianStudio/lib-commons/v6/commons/net/http/problem"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v3"
)

const (
	problemContentType           = "application/problem+json"
	databaseConnectionErrorTitle = "Database Connection Error"
	schemaRetrievalErrorTitle    = "Schema Retrieval Error"
)

// WithError renders the service's existing error taxonomy as RFC 9457. Errors
// that were historically unmapped still collapse to the FET-0002 fallback.
func WithError(c fiber.Ctx, err error) error {
	mapped := MapError(err)

	detail, ok := mapped.(*problem.Detail)
	if !ok {
		detail = &problem.Detail{}
		detail.Status = stdhttp.StatusInternalServerError
		detail.Title = stdhttp.StatusText(stdhttp.StatusInternalServerError)
		detail.Detail = "internal error"
		detail.Code = constant.ErrInternalServer.Error()
	}

	err = c.Status(detail.Status).JSON(detail)
	c.Set(fiber.HeaderContentType, problemContentType)

	return err
}

// MapError maps the Fetcher error rail into the shared RFC 9457 problem model.
// Huma handlers return this value directly; Fiber handlers render it via
// WithError.
func MapError(err error) error {
	mappedInput := err
	if _, _, ok := problemCodeOf(mappedInput); !ok {
		mappedInput = pkg.ValidateInternalError(err, "")
	}

	status := problemStatus(mappedInput)

	mapped := problem.MapError(
		mappedInput,
		problemCodeOf,
		func(string) int { return status },
		constant.ErrInternalServer.Error(),
	)
	if detail, ok := mapped.(*problem.Detail); ok {
		if detail.Status < stdhttp.StatusInternalServerError {
			detail.Errors = validationFieldDetails(mappedInput)
		} else if title, trusted := trustedResponseErrorTitle(mappedInput); trusted {
			detail.Title = title
		}
	}

	return mapped
}

func trustedResponseErrorTitle(err error) (string, bool) {
	var responseErr pkg.ResponseError
	if !errors.As(err, &responseErr) || responseErr.Code < 500 || responseErr.Code > 599 {
		return "", false
	}

	switch responseErr.Title {
	case databaseConnectionErrorTitle, schemaRetrievalErrorTitle:
		return responseErr.Title, true
	default:
		return "", false
	}
}

func validationFieldDetails(err error) []*huma.ErrorDetail {
	var knownFieldsErr pkg.ValidationKnownFieldsError
	if errors.As(err, &knownFieldsErr) {
		keys := make([]string, 0, len(knownFieldsErr.Fields))
		for field := range knownFieldsErr.Fields {
			keys = append(keys, field)
		}

		sort.Strings(keys)

		details := make([]*huma.ErrorDetail, 0, len(keys))
		for _, field := range keys {
			details = append(details, &huma.ErrorDetail{
				Location: "body." + field,
				Message:  knownFieldsErr.Fields[field],
			})
		}

		return details
	}

	var unknownFieldsErr pkg.ValidationUnknownFieldsError
	if errors.As(err, &unknownFieldsErr) {
		keys := make([]string, 0, len(unknownFieldsErr.Fields))
		for field := range unknownFieldsErr.Fields {
			keys = append(keys, field)
		}

		sort.Strings(keys)

		details := make([]*huma.ErrorDetail, 0, len(keys))
		for _, field := range keys {
			details = append(details, &huma.ErrorDetail{
				Location: "body." + field,
				Message:  "unexpected field",
				Value:    unknownFieldsErr.Fields[field],
			})
		}

		return details
	}

	return nil
}

func problemCodeOf(err error) (code, message string, ok bool) {
	var validationErr pkg.ValidationError
	if errors.As(err, &validationErr) {
		return validationErr.Code, validationErr.Message, true
	}

	var unprocessableErr pkg.UnprocessableOperationError
	if errors.As(err, &unprocessableErr) {
		return unprocessableErr.Code, unprocessableErr.Message, true
	}

	var unauthorizedErr pkg.UnauthorizedError
	if errors.As(err, &unauthorizedErr) {
		return unauthorizedErr.Code, unauthorizedErr.Message, true
	}

	var forbiddenErr pkg.ForbiddenError
	if errors.As(err, &forbiddenErr) {
		return forbiddenErr.Code, forbiddenErr.Message, true
	}

	var knownFieldsErr pkg.ValidationKnownFieldsError
	if errors.As(err, &knownFieldsErr) {
		return knownFieldsErr.Code, knownFieldsErr.Message, true
	}

	var unknownFieldsErr pkg.ValidationUnknownFieldsError
	if errors.As(err, &unknownFieldsErr) {
		return unknownFieldsErr.Code, unknownFieldsErr.Message, true
	}

	var responseErr pkg.ResponseError
	if errors.As(err, &responseErr) {
		return strconv.Itoa(responseErr.Code), responseErr.Message, true
	}

	var responseErrWithStatus pkg.ResponseErrorWithStatusCode
	if errors.As(err, &responseErrWithStatus) {
		return responseErrWithStatus.Code, responseErrWithStatus.Message, true
	}

	var internalErr pkg.InternalServerError
	if errors.As(err, &internalErr) {
		return internalErr.Code, internalErr.Message, true
	}

	return "", "", false
}

func problemStatus(err error) int {
	var validationErr pkg.ValidationError
	if errors.As(err, &validationErr) {
		return stdhttp.StatusBadRequest
	}

	var unprocessableErr pkg.UnprocessableOperationError
	if errors.As(err, &unprocessableErr) {
		return stdhttp.StatusUnprocessableEntity
	}

	var unauthorizedErr pkg.UnauthorizedError
	if errors.As(err, &unauthorizedErr) {
		return stdhttp.StatusUnauthorized
	}

	var forbiddenErr pkg.ForbiddenError
	if errors.As(err, &forbiddenErr) {
		return stdhttp.StatusForbidden
	}

	var knownFieldsErr pkg.ValidationKnownFieldsError
	if errors.As(err, &knownFieldsErr) {
		return stdhttp.StatusBadRequest
	}

	var unknownFieldsErr pkg.ValidationUnknownFieldsError
	if errors.As(err, &unknownFieldsErr) {
		return stdhttp.StatusBadRequest
	}

	var responseErr pkg.ResponseError
	if errors.As(err, &responseErr) {
		return responseErr.Code
	}

	var responseErrWithStatus pkg.ResponseErrorWithStatusCode
	if errors.As(err, &responseErrWithStatus) {
		return responseErrWithStatus.StatusCode
	}

	return stdhttp.StatusInternalServerError
}
