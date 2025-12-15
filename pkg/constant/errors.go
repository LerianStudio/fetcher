package constant

import (
	"errors"
)

// List of errors that can be returned.
// You can standardize errors
// Standardized error
var (

	// #################################################### COMMON ERRORS ####################################################

	// General errors
	ErrBadRequest         = errors.New("FET-0001")
	ErrInternalServer     = errors.New("FET-0002")
	ErrServiceUnavailable = errors.New("FET-0003")
	ErrConflict           = errors.New("FET-0004")
	ErrNotFound           = errors.New("FET-0005")

	// Request related errors
	ErrUnexpectedFieldsInTheRequest = errors.New("FET-0400")
	ErrInvalidDataRequest           = errors.New("FET-0401")
	ErrMissingFieldsInRequest       = errors.New("FET-0402")
	ErrInvalidHeaderParameter       = errors.New("FET-0403")
	ErrInvalidPathParameter         = errors.New("FET-0404")
	ErrInvalidQueryParameter        = errors.New("FET-0405")
	ErrPaginationLimitExceeded      = errors.New("FET-0406")
	ErrInvalidSortOrder             = errors.New("FET-0407")
	ErrInvalidDateFormat            = errors.New("FET-0408")
	ErrInvalidFinalDate             = errors.New("FET-0409")
	ErrInvalidMetadataNesting       = errors.New("FET-0410")
	ErrMetadataValueLengthExceeded  = errors.New("FET-0411")
	ErrMetadataKeyLengthExceeded    = errors.New("FET-0412")

	// #################################################### BUSINESS LOGIC ERRORS ####################################################

	// Entity related errors
	ErrEntityNotFound = errors.New("FET-1001")
	ErrEntityConflict = errors.New("FET-1002")

	// Job related errors
	ErrMissingDataSource = errors.New("FET-1020")
	ErrJobInProgress     = errors.New("FET-1021")

	// Connection related errors
	ErrConnectionDown = errors.New("FET-1040")

	// Schema validation errors
	ErrSchemaValidationFailed   = errors.New("FET-1060")
	ErrSchemaValidationLimit    = errors.New("FET-1061")
	ErrSchemaValidationNotFound = errors.New("FET-1062")
)
