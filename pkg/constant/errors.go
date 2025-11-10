package constant

import (
	"errors"
)

// List of errors that can be returned.
// You can standardize errors
var (
	ErrUnexpectedFieldsInTheRequest = errors.New("BLP-0001")
	ErrMissingFieldsInRequest       = errors.New("BLP-0002")
	ErrBadRequest                   = errors.New("BLP-0003")
	ErrInternalServer               = errors.New("BLP-0004")
	ErrCalculationFieldType         = errors.New("BLP-0005")
	ErrInvalidQueryParameter        = errors.New("BLP-0006")
	ErrInvalidDateFormat            = errors.New("BLP-0007")
	ErrInvalidFinalDate             = errors.New("BLP-0008")
	ErrDateRangeExceedsLimit        = errors.New("BLP-0009")
	ErrInvalidDateRange             = errors.New("BLP-0010")
	ErrPaginationLimitExceeded      = errors.New("BLP-0011")
	ErrInvalidSortOrder             = errors.New("BLP-0012")
	ErrEntityNotFound               = errors.New("BLP-0013")
	ErrActionNotPermitted           = errors.New("BLP-0014")
	ErrParentExampleIDNotFound      = errors.New("BLP-0015")
	ErrMetadataKeyLengthExceeded    = errors.New("BLP-0016")
	ErrMetadataValueLengthExceeded  = errors.New("BLP-0017")
	ErrInvalidMetadataNesting       = errors.New("BLP-0018")
	ErrInvalidPathParameter         = errors.New("BLP-0019")
)
