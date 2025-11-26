package constant

import (
	"errors"
)

// List of errors that can be returned.
// You can standardize errors
// Standardized error
var (
	ErrMissingRequiredFields           = errors.New("FET-0001")
	ErrInvalidFileFormat               = errors.New("FET-0002")
	ErrInvalidOutputFormat             = errors.New("FET-0003")
	ErrInvalidHeaderParameter          = errors.New("FET-0004")
	ErrInvalidFileUploaded             = errors.New("FET-0005")
	ErrEmptyFile                       = errors.New("FET-0006")
	ErrFileContentInvalid              = errors.New("FET-0007")
	ErrInvalidMapFields                = errors.New("FET-0008")
	ErrInvalidPathParameter            = errors.New("FET-0009")
	ErrOutputFormatWithoutTemplateFile = errors.New("FET-0010")
	ErrEntityNotFound                  = errors.New("FET-0011")
	ErrInvalidTemplateID               = errors.New("FET-0012")
	ErrInvalidLedgerIDList             = errors.New("FET-0013")
	ErrMissingTableFields              = errors.New("FET-0014")
	ErrUnexpectedFieldsInTheRequest    = errors.New("FET-0015")
	ErrMissingFieldsInRequest          = errors.New("FET-0016")
	ErrBadRequest                      = errors.New("FET-0017")
	ErrInternalServer                  = errors.New("FET-0018")
	ErrInvalidQueryParameter           = errors.New("FET-0019")
	ErrInvalidDateFormat               = errors.New("FET-0020")
	ErrInvalidFinalDate                = errors.New("FET-0021")
	ErrDateRangeExceedsLimit           = errors.New("FET-0022")
	ErrInvalidDateRange                = errors.New("FET-0023")
	ErrPaginationLimitExceeded         = errors.New("FET-0024")
	ErrInvalidSortOrder                = errors.New("FET-0025")
	ErrMetadataKeyLengthExceeded       = errors.New("FET-0026")
	ErrMetadataValueLengthExceeded     = errors.New("FET-0027")
	ErrInvalidMetadataNesting          = errors.New("FET-0028")
	ErrMissingSchemaTable              = errors.New("FET-0029")
	ErrMissingDataSource               = errors.New("FET-0030")
	ErrScriptTagDetected               = errors.New("FET-0031")
	ErrDecryptionData                  = errors.New("FET-0032")
	ErrCommunicateSeaweedFS            = errors.New("FET-0033")
	ErrEntityConflict                  = errors.New("FET-0034")
)
