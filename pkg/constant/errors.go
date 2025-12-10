package constant

import (
	"errors"
)

// List of errors that can be returned.
// You can standardize errors
// Standardized error
var (
	ErrInvalidHeaderParameter       = errors.New("FET-0001")
	ErrInvalidPathParameter         = errors.New("FET-0002")
	ErrEntityNotFound               = errors.New("FET-0003")
	ErrUnexpectedFieldsInTheRequest = errors.New("FET-0004")
	ErrMissingFieldsInRequest       = errors.New("FET-0005")
	ErrBadRequest                   = errors.New("FET-0006")
	ErrInternalServer               = errors.New("FET-0007")
	ErrInvalidQueryParameter        = errors.New("FET-0008")
	ErrInvalidDateFormat            = errors.New("FET-0009")
	ErrInvalidFinalDate             = errors.New("FET-0010")
	ErrPaginationLimitExceeded      = errors.New("FET-0011")
	ErrInvalidSortOrder             = errors.New("FET-0012")
	ErrMetadataKeyLengthExceeded    = errors.New("FET-0013")
	ErrMetadataValueLengthExceeded  = errors.New("FET-0014")
	ErrInvalidMetadataNesting       = errors.New("FET-0015")
	ErrMissingDataSource            = errors.New("FET-0016")
	ErrCommunicateSeaweedFS         = errors.New("FET-0017")
	ErrEntityConflict               = errors.New("FET-0018")
	ErrJobInProgress                = errors.New("FET-0019")
	ErrConnectionDown               = errors.New("FET-0020")
	ErrInvalidDataRequest           = errors.New("FET-0021")
)
