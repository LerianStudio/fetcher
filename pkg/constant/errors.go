package constant

import (
	"errors"
)

// List of errors that can be returned.
// You can standardize errors
var (
	ErrUnexpectedFieldsInTheRequest = errors.New("FET-0001")
	ErrMissingFieldsInRequest       = errors.New("FET-0002")
	ErrBadRequest                   = errors.New("FET-0003")
	ErrInternalServer               = errors.New("FET-0004")
	ErrCalculationFieldType         = errors.New("FET-0005")
	ErrInvalidQueryParameter        = errors.New("FET-0006")
	ErrInvalidDateFormat            = errors.New("FET-0007")
	ErrInvalidFinalDate             = errors.New("FET-0008")
	ErrDateRangeExceedsLimit        = errors.New("FET-0009")
	ErrInvalidDateRange             = errors.New("FET-0010")
	ErrPaginationLimitExceeded      = errors.New("FET-0011")
	ErrInvalidSortOrder             = errors.New("FET-0012")
	ErrEntityNotFound               = errors.New("FET-0013")
	ErrActionNotPermitted           = errors.New("FET-0014")
	ErrParentExampleIDNotFound      = errors.New("FET-0015")
	ErrMetadataKeyLengthExceeded    = errors.New("FET-0016")
	ErrMetadataValueLengthExceeded  = errors.New("FET-0017")
	ErrInvalidMetadataNesting       = errors.New("FET-0018")
	ErrInvalidPathParameter         = errors.New("FET-0019")
	ErrInvalidHeaderParameter       = errors.New("FET-0020")
)
