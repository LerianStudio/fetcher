package pkg

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/LerianStudio/fetcher/pkg/constant"
)

// ValidationError records an error indicating an entity was not found in any case that caused it.
// You can use it to representing a Database not found, cache not found or any other repository.
type ValidationError struct {
	EntityType string `json:"entityType,omitempty"`
	Title      string `json:"title,omitempty"`
	Message    string `json:"message,omitempty"`
	Code       string `json:"code,omitempty"`
	Err        error  `json:"err,omitempty"`
}

// Error implements the error interface.
func (e ValidationError) Error() string {
	if strings.TrimSpace(e.Code) != "" {
		return fmt.Sprintf("%s - %s", e.Code, e.Message)
	}

	return e.Message
}

// Unwrap implements the error interface introduced in Go 1.13 to unwrap the internal error.
func (e ValidationError) Unwrap() error {
	return e.Err
}

// UnauthorizedError indicates an operation that couldn't be performed because there's no user authenticated.
type UnauthorizedError struct {
	EntityType string `json:"entityType,omitempty"`
	Title      string `json:"title,omitempty"`
	Message    string `json:"message,omitempty"`
	Code       string `json:"code,omitempty"`
	Err        error  `json:"err,omitempty"`
}

func (e UnauthorizedError) Error() string {
	return e.Message
}

// Unwrap implements the error interface introduced in Go 1.13 to unwrap the internal error.
func (e UnauthorizedError) Unwrap() error {
	return e.Err
}

// ForbiddenError indicates an operation that couldn't be performed because the authenticated user has no sufficient privileges.
type ForbiddenError struct {
	EntityType string `json:"entityType,omitempty"`
	Title      string `json:"title,omitempty"`
	Message    string `json:"message,omitempty"`
	Code       string `json:"code,omitempty"`
	Err        error  `json:"err,omitempty"`
}

func (e ForbiddenError) Error() string {
	return e.Message
}

// Unwrap implements the error interface introduced in Go 1.13 to unwrap the internal error.
func (e ForbiddenError) Unwrap() error {
	return e.Err
}

// UnprocessableOperationError indicates an operation that couldn't be performed because it's invalid.
type UnprocessableOperationError struct {
	EntityType string `json:"entityType,omitempty"`
	Title      string `json:"title,omitempty"`
	Message    string `json:"message,omitempty"`
	Code       string `json:"code,omitempty"`
	Err        error  `json:"err,omitempty"`
}

func (e UnprocessableOperationError) Error() string {
	return e.Message
}

// Unwrap implements the error interface introduced in Go 1.13 to unwrap the internal error.
func (e UnprocessableOperationError) Unwrap() error {
	return e.Err
}

// HTTPError indicates a http error raised in a http client.
type HTTPError struct {
	EntityType string `json:"entityType,omitempty"`
	Title      string `json:"title,omitempty"`
	Message    string `json:"message,omitempty"`
	Code       string `json:"code,omitempty"`
	Err        error  `json:"err,omitempty"`
}

func (e HTTPError) Error() string {
	return e.Message
}

// Unwrap implements the error interface introduced in Go 1.13 to unwrap the internal error.
func (e HTTPError) Unwrap() error {
	return e.Err
}

// FailedPreconditionError indicates a precondition failed during an operation.
type FailedPreconditionError struct {
	EntityType string `json:"entityType,omitempty"`
	Title      string `json:"title,omitempty"`
	Message    string `json:"message,omitempty"`
	Code       string `json:"code,omitempty"`
	Err        error  `json:"err,omitempty"`
}

func (e FailedPreconditionError) Error() string {
	return e.Message
}

// Unwrap implements the error interface introduced in Go 1.13 to unwrap the internal error.
func (e FailedPreconditionError) Unwrap() error {
	return e.Err
}

// InternalServerError indicates an internal server error during an operation.
type InternalServerError struct {
	EntityType string `json:"entityType,omitempty"`
	Title      string `json:"title,omitempty"`
	Message    string `json:"message,omitempty"`
	Code       string `json:"code,omitempty"`
	Err        error  `json:"err,omitempty"`
}

func (e InternalServerError) Error() string {
	return e.Message
}

// Unwrap implements the error interface introduced in Go 1.13 to unwrap the internal error.
func (e InternalServerError) Unwrap() error {
	return e.Err
}

// ResponseError is a struct used to return errors to the client.
type ResponseError struct {
	Code    int    `json:"code,omitempty"`
	Title   string `json:"title,omitempty"`
	Message string `json:"message,omitempty"`
}

// Error returns the message of the ResponseError.
//
// No parameters.
// Returns a string.
func (r ResponseError) Error() string {
	return r.Message
}

// ResponseErrorWithStatusCode is a struct used to return errors to the client with status code.
type ResponseErrorWithStatusCode struct {
	StatusCode int
	Code       string
	Title      string
	Message    string
}

// Error returns the message of the ResponseError.
//
// No parameters.
// Returns a string.
func (r ResponseErrorWithStatusCode) Error() string {
	return r.Message
}

// ValidationKnownFieldsError records an error that occurred during a validation of known fields.
type ValidationKnownFieldsError struct {
	EntityType string           `json:"entityType,omitempty"`
	Title      string           `json:"title,omitempty"`
	Code       string           `json:"code,omitempty"`
	Message    string           `json:"message,omitempty"`
	Fields     FieldValidations `json:"fields,omitempty"`
}

// FieldValidations is a map of known fields and their validation errors.
type FieldValidations map[string]string

// Error returns the error message for a ValidationKnownFieldsError.
//
// No parameters.
// Returns a string.
func (r ValidationKnownFieldsError) Error() string {
	return r.Message
}

// ValidationUnknownFieldsError records an error that occurred during a validation of known fields.
type ValidationUnknownFieldsError struct {
	EntityType string        `json:"entityType,omitempty"`
	Title      string        `json:"title,omitempty"`
	Code       string        `json:"code,omitempty"`
	Message    string        `json:"message,omitempty"`
	Fields     UnknownFields `json:"fields,omitempty"`
}

// Error returns the error message for a ValidationUnknownFieldsError.
//
// No parameters.
// Returns a string.
func (r ValidationUnknownFieldsError) Error() string {
	return r.Message
}

// UnknownFields is a map of unknown fields and their error messages.
type UnknownFields map[string]any

// Methods to create errors for different scenarios:

// ValidateInternalError validates the error and returns an appropriate 4XX and 5XX.
//
// Parameters:
// - err: The error to be validated.
// - entityType: The type of the entity associated with the error.
//
// Returns:
// - An error indicating the appropriate 4XX or 5XX error.
func ValidateInternalError(err error, entityType string) error {
	switch err {
	case constant.ErrBadRequest:
		return ValidationError{
			EntityType: entityType,
			Code:       constant.ErrBadRequest.Error(),
			Title:      "Bad Request",
			Message:    "The server could not understand the request due to malformed syntax. Please check the request and try again.",
		}
	case constant.ErrServiceUnavailable:
		return ResponseErrorWithStatusCode{
			StatusCode: http.StatusServiceUnavailable,
			Code:       constant.ErrServiceUnavailable.Error(),
			Title:      "Service Unavailable",
			Message:    "The server is currently unable to handle the request due to temporary overloading or maintenance of the server. Please try again later.",
		}
	case constant.ErrConflict:
		return ResponseErrorWithStatusCode{
			StatusCode: http.StatusConflict,
			Code:       constant.ErrConflict.Error(),
			Title:      "Conflict",
			Message:    "The request could not be completed due to a conflict with the current state of the resource. Please resolve the conflict and try again.",
		}
	case constant.ErrNotFound:
		return ResponseErrorWithStatusCode{
			StatusCode: http.StatusNotFound,
			Code:       constant.ErrNotFound.Error(),
			Title:      "Not Found",
			Message:    "The requested resource could not be found. Please check the request and try again.",
		}
	default:
		return InternalServerError{
			EntityType: entityType,
			Code:       constant.ErrInternalServer.Error(),
			Title:      "Internal Server Error",
			Message:    "The server encountered an unexpected error. Please try again later or contact support.",
			Err:        err,
		}
	}
}

// ValidateBadRequestFieldsError validates the error and returns the appropriate bad request error code, title, message, and the invalid fields.
//
// Parameters:
// - requiredFields: A map of missing required fields and their error messages.
// - knownInvalidFields: A map of known invalid fields and their validation errors.
// - entityType: The type of the entity associated with the error.
// - unknownFields: A map of unknown fields and their error messages.
//
// Returns:
// - An error indicating the validation result, which could be a ValidationUnknownFieldsError or a ValidationKnownFieldsError.
func ValidateBadRequestFieldsError(requiredFields, knownInvalidFields map[string]string, entityType string, unknownFields map[string]any) error {
	if len(unknownFields) == 0 && len(knownInvalidFields) == 0 && len(requiredFields) == 0 {
		return errors.New("expected knownInvalidFields, unknownFields and requiredFields to be non-empty")
	}

	if len(unknownFields) > 0 {
		return ValidationUnknownFieldsError{
			EntityType: entityType,
			Code:       constant.ErrUnexpectedFieldsInTheRequest.Error(),
			Title:      "Unexpected Fields in the Request",
			Message:    "The request body contains more fields than expected. Please send only the allowed fields as per the documentation. The unexpected fields are listed in the fields object.",
			Fields:     unknownFields,
		}
	}

	if len(requiredFields) > 0 {
		return ValidationKnownFieldsError{
			EntityType: entityType,
			Code:       constant.ErrMissingFieldsInRequest.Error(),
			Title:      "Missing Fields in Request",
			Message:    "Your request is missing one or more required fields. Please refer to the documentation to ensure all necessary fields are included in your request.",
			Fields:     requiredFields,
		}
	}

	return ValidationKnownFieldsError{
		EntityType: entityType,
		Code:       constant.ErrBadRequest.Error(),
		Title:      "Bad Request",
		Message:    "The server could not understand the request due to malformed syntax. Please check the listed fields and try again.",
		Fields:     knownInvalidFields,
	}
}

// ValidateBusinessError validates the error and returns the appropriate business error code, title, and message.
// error: The appropriate business error with code, title, and message.
func ValidateBusinessError(err error, entityType string, args ...any) error {
	if result := validateCommonErrors(err, entityType, args...); result != nil {
		return result
	}

	if result := validateEntityErrors(err, entityType); result != nil {
		return result
	}

	if result := validateTenantErrors(err, entityType); result != nil {
		return result
	}

	if result := validateProductErrors(err, entityType, args...); result != nil {
		return result
	}

	if result := validateJobAndConnectionErrors(err, entityType, args...); result != nil {
		return result
	}

	return err
}

// validateCommonErrors handles common validation errors (query params, dates, pagination, metadata, etc.).
func validateCommonErrors(err error, entityType string, args ...any) error {
	switch err {
	case constant.ErrInvalidQueryParameter:
		return ValidationError{
			EntityType: entityType,
			Code:       constant.ErrInvalidQueryParameter.Error(),
			Title:      "Invalid Query Parameter",
			Message:    fmt.Sprintf("One or more query parameters are in an incorrect format. Please check the following parameters '%v' and ensure they meet the required format before trying again.", args...),
		}
	case constant.ErrInvalidDateFormat:
		return ValidationError{
			EntityType: entityType,
			Code:       constant.ErrInvalidDateFormat.Error(),
			Title:      "Invalid Date Format Error",
			Message:    "The 'initialDate', 'finalDate', or both are in the incorrect format. Please use the 'yyyy-mm-dd' format and try again.",
		}
	case constant.ErrInvalidFinalDate:
		return ValidationError{
			EntityType: entityType,
			Code:       constant.ErrInvalidFinalDate.Error(),
			Title:      "Invalid Final Date Error",
			Message:    "The 'finalDate' cannot be earlier than the 'initialDate'. Please verify the dates and try again.",
		}
	case constant.ErrPaginationLimitExceeded:
		return ValidationError{
			EntityType: entityType,
			Code:       constant.ErrPaginationLimitExceeded.Error(),
			Title:      "Pagination Limit Exceeded",
			Message:    fmt.Sprintf("The pagination limit exceeds the maximum allowed of %v items per page. Please verify the limit and try again.", args...),
		}
	case constant.ErrInvalidSortOrder:
		return ValidationError{
			EntityType: entityType,
			Code:       constant.ErrInvalidSortOrder.Error(),
			Title:      "Invalid Sort Order",
			Message:    "The 'sort_order' field must be 'asc' or 'desc'. Please provide a valid sort order and try again.",
		}
	case constant.ErrMetadataKeyLengthExceeded:
		return ValidationError{
			EntityType: entityType,
			Code:       constant.ErrMetadataKeyLengthExceeded.Error(),
			Title:      "Metadata Key Length Exceeded",
			Message:    fmt.Sprintf("The metadata key %v exceeds the maximum allowed length of %v characters. Please use a shorter key.", args...),
		}
	case constant.ErrMetadataValueLengthExceeded:
		return ValidationError{
			EntityType: entityType,
			Code:       constant.ErrMetadataValueLengthExceeded.Error(),
			Title:      "Metadata Value Length Exceeded",
			Message:    fmt.Sprintf("The metadata value %v exceeds the maximum allowed length of %v characters. Please use a shorter value.", args...),
		}
	case constant.ErrInvalidMetadataNesting:
		return ValidationError{
			EntityType: entityType,
			Code:       constant.ErrInvalidMetadataNesting.Error(),
			Title:      "Invalid Metadata Nesting",
			Message:    fmt.Sprintf("The metadata object cannot contain nested values. Please ensure that the value %v is not nested and try again.", args...),
		}
	case constant.ErrInvalidHeaderParameter:
		return ValidationError{
			EntityType: entityType,
			Code:       constant.ErrInvalidHeaderParameter.Error(),
			Title:      "Invalid header",
			Message:    fmt.Sprintf("One or more header values are missing or incorrectly formatted. Please verify required headers %v.", args...),
		}
	case constant.ErrInvalidPathParameter:
		return ValidationError{
			EntityType: entityType,
			Code:       constant.ErrInvalidPathParameter.Error(),
			Title:      "Invalid Path Parameter",
			Message:    fmt.Sprintf("Path parameters is in an incorrect format. Please check the following parameter %v and ensure they meet the required format before trying again.", args...),
		}
	case constant.ErrInvalidDataRequest:
		msg := "The request contains invalid data. Please check the request payload and try again."
		if len(args) > 0 {
			msg = fmt.Sprint(args...)
		}

		return ValidationError{
			EntityType: entityType,
			Code:       constant.ErrInvalidDataRequest.Error(),
			Title:      "Invalid Data Request",
			Message:    msg,
		}
	case constant.ErrInvalidSSLMode:
		return ValidationError{
			EntityType: entityType,
			Code:       constant.ErrInvalidSSLMode.Error(),
			Title:      "Invalid SSL Mode",
			Message:    fmt.Sprintf("Invalid SSL mode. The provided SSL mode '%s' is not supported. Please use a valid SSL mode for this database type.", args...),
		}
	case constant.ErrForbiddenHost:
		// Generic message — must NOT echo the host or reveal which CIDR / suffix matched,
		// otherwise the response itself becomes a reconnaissance oracle.
		return ValidationError{
			EntityType: entityType,
			Code:       constant.ErrForbiddenHost.Error(),
			Title:      "Forbidden Host",
			Message:    "Host is not a valid external database endpoint",
		}
	default:
		return nil
	}
}

// validateEntityErrors handles entity-related errors (not found, conflict).
func validateEntityErrors(err error, entityType string) error {
	switch err {
	case constant.ErrEntityNotFound:
		return ResponseErrorWithStatusCode{
			StatusCode: http.StatusNotFound,
			Code:       constant.ErrEntityNotFound.Error(),
			Title:      "Entity Not Found",
			Message:    fmt.Sprintf("It was not possible to find the %v entity during the requested flow. Please review the data provided in the request.", entityType),
		}
	case constant.ErrEntityConflict:
		return ResponseErrorWithStatusCode{
			StatusCode: http.StatusConflict,
			Code:       constant.ErrEntityConflict.Error(),
			Title:      "Conflict",
			Message:    fmt.Sprintf("An entity of type %v with the same unique attributes already exists. Please use different values to avoid conflicts and review the data provided in the request.", entityType),
		}
	default:
		return nil
	}
}

// validateTenantErrors handles multi-tenant errors (context required, not found, circuit breaker).
func validateTenantErrors(err error, entityType string) error {
	switch err {
	case constant.ErrTenantContextRequired:
		return ValidationError{
			EntityType: entityType,
			Code:       constant.ErrTenantContextRequired.Error(),
			Title:      "Tenant Context Required",
			Message:    "The request requires a tenant context. Ensure the X-Tenant-ID header is present and valid.",
		}
	case constant.ErrTenantNotFound:
		return ResponseErrorWithStatusCode{
			StatusCode: http.StatusNotFound,
			Code:       constant.ErrTenantNotFound.Error(),
			Title:      "Tenant Not Found",
			Message:    "The specified tenant was not found. Verify the tenant ID and try again.",
		}
	case constant.ErrTenantCircuitBreaker:
		return ResponseErrorWithStatusCode{
			StatusCode: http.StatusServiceUnavailable,
			Code:       constant.ErrTenantCircuitBreaker.Error(),
			Title:      "Tenant Service Unavailable",
			Message:    "The tenant service is temporarily unavailable due to circuit breaker protection. Please try again later.",
		}
	default:
		return nil
	}
}

// validateProductErrors handles product-related errors (has connections, not assigned, already assigned, mismatch).
func validateProductErrors(err error, entityType string, args ...any) error {
	switch err {
	case constant.ErrProductHasConnections:
		return ResponseErrorWithStatusCode{
			StatusCode: http.StatusConflict,
			Code:       constant.ErrProductHasConnections.Error(),
			Title:      "Product Has Connections",
			Message:    fmt.Sprintf("The %v cannot be deleted because it has associated connections. Remove or reassign all connections before deleting.", entityType),
		}
	case constant.ErrConnectionNotAssigned:
		msg := "The connection is not assigned to any product."
		if len(args) > 0 {
			msg = fmt.Sprint(args...)
		}

		return ValidationError{
			EntityType: entityType,
			Code:       constant.ErrConnectionNotAssigned.Error(),
			Title:      "Connection Not Assigned",
			Message:    msg,
		}
	case constant.ErrConnectionAlreadyAssigned:
		return ResponseErrorWithStatusCode{
			StatusCode: http.StatusConflict,
			Code:       constant.ErrConnectionAlreadyAssigned.Error(),
			Title:      "Connection Already Assigned",
			Message:    fmt.Sprintf("The %v is already assigned to a product and cannot be reassigned.", entityType),
		}
	case constant.ErrProductMismatch:
		msg := "One or more datasources do not belong to the specified product."
		if len(args) > 0 {
			msg = fmt.Sprint(args...)
		}

		return ValidationError{
			EntityType: entityType,
			Code:       constant.ErrProductMismatch.Error(),
			Title:      "Product Mismatch",
			Message:    msg,
		}
	default:
		return nil
	}
}

// validateJobAndConnectionErrors handles job, connection, and schema validation errors.
func validateJobAndConnectionErrors(err error, entityType string, args ...any) error {
	switch err {
	case constant.ErrMissingDataSource:
		return ValidationError{
			EntityType: entityType,
			Code:       constant.ErrMissingDataSource.Error(),
			Title:      "Missing Data Source Table",
			Message:    fmt.Sprintf("The data source %v is missing. Please check the value passed.", args...),
		}
	case constant.ErrJobInProgress:
		msg := "The operation cannot be completed because there are active jobs for this connection."
		if len(args) > 0 {
			msg = fmt.Sprint(args...)
		}

		return ResponseErrorWithStatusCode{
			StatusCode: http.StatusConflict,
			Code:       constant.ErrJobInProgress.Error(),
			Title:      "Job In Progress",
			Message:    msg,
		}
	case constant.ErrConnectionDown:
		msg := "The database connection is not available. Please check the connection configuration and try again."
		if len(args) > 0 {
			msg = fmt.Sprint(args...)
		}

		return ValidationError{
			EntityType: entityType,
			Code:       constant.ErrConnectionDown.Error(),
			Title:      "Connection Down",
			Message:    msg,
		}
	case constant.ErrSchemaValidationFailed:
		msg := "Schema validation found inconsistencies."
		if len(args) > 0 {
			msg = fmt.Sprint(args...)
		}

		return ValidationError{
			EntityType: entityType,
			Code:       constant.ErrSchemaValidationFailed.Error(),
			Title:      "Schema Validation Failed",
			Message:    msg,
		}
	case constant.ErrSchemaValidationLimit:
		msg := "Validation request exceeds allowed limits."
		if len(args) > 0 {
			msg = fmt.Sprint(args...)
		}

		return ValidationError{
			EntityType: entityType,
			Code:       constant.ErrSchemaValidationLimit.Error(),
			Title:      "Validation Limit Exceeded",
			Message:    msg,
		}
	default:
		return nil
	}
}
