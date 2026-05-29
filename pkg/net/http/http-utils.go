package http

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	libCommons "github.com/LerianStudio/lib-commons/v5/commons"
	"github.com/gofiber/fiber/v2"
)

// QueryHeader entity from query parameter from get apis
type QueryHeader struct {
	Metadata    map[string]string
	Limit       int
	Page        int
	Cursor      string
	SortOrder   string
	StartDate   time.Time
	EndDate     time.Time
	UseMetadata bool
	ProductName string
	Type        string
}

// Pagination entity from query parameter from get apis
type Pagination struct {
	Limit     int
	Page      int
	Cursor    string
	SortOrder string
	StartDate time.Time
	EndDate   time.Time
}

func (qh *QueryHeader) ToOffsetPagination() Pagination {
	return Pagination{
		Limit:     qh.Limit,
		Page:      qh.Page,
		SortOrder: qh.SortOrder,
		StartDate: qh.StartDate,
		EndDate:   qh.EndDate,
	}
}

// ValidateParameters validate and return struct of default parameters
func ValidateParameters(params map[string]string) (*QueryHeader, error) {
	var (
		metadata    = make(map[string]string)
		startDate   time.Time
		endDate     time.Time
		cursor      string
		connType    string
		limit       = 10
		page        = 1
		sortOrder   = "desc"
		useMetadata = false
	)

	if err := parseParameters(params, metadata, &startDate, &endDate, &cursor, &limit, &page, &sortOrder, &connType); err != nil {
		return nil, err
	}

	var metadataResult map[string]string
	if len(metadata) > 0 {
		metadataResult = metadata
		useMetadata = true
	}

	err := validateDates(&startDate, &endDate)
	if err != nil {
		return nil, err
	}

	err = validatePagination(cursor, sortOrder, limit, page)
	if err != nil {
		return nil, err
	}

	query := &QueryHeader{
		Metadata:    metadataResult,
		Limit:       limit,
		Page:        page,
		Cursor:      cursor,
		SortOrder:   sortOrder,
		StartDate:   startDate,
		EndDate:     endDate,
		UseMetadata: useMetadata,
		Type:        connType,
	}

	return query, nil
}

func parseParameters(
	params map[string]string,
	metadata map[string]string,
	startDate, endDate *time.Time,
	cursor *string,
	limit, page *int,
	sortOrder *string,
	connType *string,
) error {
	for key, value := range params {
		if value == "" {
			continue
		}

		switch {
		case strings.HasPrefix(key, "metadata."):
			metadata[key] = value
		case key == "limit":
			if err := parseIntParam(value, "limit", limit); err != nil {
				return err
			}
		case key == "page":
			if err := parseIntParam(value, "page", page); err != nil {
				return err
			}
		case key == "cursor":
			*cursor = value
		case key == "type":
			*connType = strings.ToUpper(value)
		case key == "sortOrder":
			*sortOrder = strings.ToLower(value)
		case key == "startDate":
			if err := parseDateParam(value, "startDate", startDate); err != nil {
				return err
			}
		case key == "endDate":
			if err := parseDateParam(value, "endDate", endDate); err != nil {
				return err
			}
		default:
			if err := validateUnknownParam(key, value); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateUnknownParam rejects unsafe query parameter keys (MongoDB operator injection, oversized payloads).
func validateUnknownParam(key, value string) error {
	// Reject keys that start with "$" to prevent MongoDB operator
	// injection (e.g. $where, $ne, $regex). Also reject keys starting
	// with underscore (internal fields like _id).
	if strings.HasPrefix(key, "$") || strings.HasPrefix(key, "_") {
		return pkg.ValidateBusinessError(constant.ErrInvalidQueryParameter, key)
	}

	// Cap key/value length to prevent abuse via oversized query payloads.
	const (
		maxFilterKeyLen   = 64
		maxFilterValueLen = 256
	)

	if len(key) > maxFilterKeyLen || len(value) > maxFilterValueLen {
		return pkg.ValidateBusinessError(constant.ErrInvalidQueryParameter, key)
	}

	// Unknown keys that pass safety checks are silently ignored.
	// Only keys with the "metadata." prefix are captured as filters.
	return nil
}

func parseIntParam(value, name string, dest *int) error {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return pkg.ValidateBusinessError(constant.ErrInvalidQueryParameter, name)
	}

	*dest = parsed

	return nil
}

func parseDateParam(value, name string, dest *time.Time) error {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return pkg.ValidateBusinessError(constant.ErrInvalidDateFormat, name)
	}

	*dest = parsed

	return nil
}

func validateDates(startDate, endDate *time.Time) error {
	maxDateRangeMonths := libCommons.SafeInt64ToInt(pkg.GetenvIntOrDefault("MAX_PAGINATION_MONTH_DATE_RANGE", 1))

	today := time.Date(
		time.Now().Year(),
		time.Now().Month(),
		time.Now().Day(),
		0, 0, 0, 0,
		time.Now().Location(),
	)

	bothDatesEmpty := startDate.IsZero() && endDate.IsZero()

	if bothDatesEmpty {
		*endDate = today.AddDate(0, 0, 1)
		*startDate = endDate.AddDate(0, -maxDateRangeMonths, 0)

		return nil
	}

	if startDate.IsZero() {
		*startDate = today.AddDate(0, -maxDateRangeMonths, 0)
	}

	if endDate.IsZero() {
		*endDate = startDate.AddDate(0, 0, 1)
	}

	if !pkg.IsValidDate(pkg.NormalizeDate(*startDate, nil)) || !pkg.IsValidDate(pkg.NormalizeDate(*endDate, nil)) {
		return pkg.ValidateBusinessError(constant.ErrInvalidDateFormat, "")
	}

	if startDate.Equal(*endDate) {
		*endDate = endDate.AddDate(0, 0, 1)
	}

	if !pkg.IsInitialDateBeforeFinalDate(*startDate, *endDate) {
		return pkg.ValidateBusinessError(constant.ErrInvalidFinalDate, "")
	}

	if !pkg.IsDateRangeWithinMonthLimit(*startDate, *endDate, maxDateRangeMonths) {
		*startDate = endDate.AddDate(0, -maxDateRangeMonths, 0)
	}

	return nil
}

// productNameRegex validates product name format: alphanumeric with underscores and hyphens.
var productNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// validateProductName checks character set and length constraints on a product name.
func validateProductName(productName string) error {
	if len(productName) > 100 {
		return pkg.ValidationError{
			EntityType: "request",
			Code:       constant.ErrInvalidHeaderParameter.Error(),
			Title:      "Invalid header",
			Message:    "X-Product-Name must not exceed 100 characters",
		}
	}

	if !productNameRegex.MatchString(productName) {
		return pkg.ValidationError{
			EntityType: "request",
			Code:       constant.ErrInvalidHeaderParameter.Error(),
			Title:      "Invalid header",
			Message:    "X-Product-Name can only contain alphanumeric characters, underscores, and hyphens",
		}
	}

	return nil
}

// GetProductName extracts X-Product-Name header (optional).
// Returns empty string if the header is not provided.
// Returns error if the header is provided but is empty, whitespace-only, or has invalid format.
func GetProductName(c *fiber.Ctx) (string, error) {
	raw := c.Get("X-Product-Name")
	if raw == "" {
		return "", nil // header not provided
	}

	productName := strings.TrimSpace(raw)
	if productName == "" {
		return "", pkg.ValidationError{
			EntityType: "request",
			Code:       constant.ErrInvalidHeaderParameter.Error(),
			Title:      "Invalid header",
			Message:    "X-Product-Name header must not be empty or whitespace-only",
		}
	}

	productName = strings.ToLower(productName)

	if err := validateProductName(productName); err != nil {
		return "", err
	}

	return productName, nil
}

// GetRequiredProductName extracts X-Product-Name header (required).
// Returns error if the header is missing, empty, whitespace-only, or has invalid format.
func GetRequiredProductName(c *fiber.Ctx) (string, error) {
	productName := strings.TrimSpace(c.Get("X-Product-Name"))
	if productName == "" {
		return "", pkg.ValidationError{
			EntityType: "request",
			Code:       constant.ErrInvalidHeaderParameter.Error(),
			Title:      "Invalid header",
			Message:    "X-Product-Name header is required and must not be empty",
		}
	}

	productName = strings.ToLower(productName)

	if err := validateProductName(productName); err != nil {
		return "", err
	}

	return productName, nil
}

// ParseIntDefault parses int with fallback.
func ParseIntDefault(val string, def int) int {
	if val == "" {
		return def
	}

	if parsed, err := strconv.Atoi(val); err == nil {
		return parsed
	}

	return def
}

// ClampLimit ensures limit is within bounds, applying default if <=0.
func ClampLimit(limit, def, maxLimit int) int {
	if limit <= 0 {
		return def
	}

	if limit > maxLimit {
		return maxLimit
	}

	return limit
}

// ClampNonNegative ensures page is not negative.
func ClampNonNegative(page int) int {
	if page < 0 {
		return 0
	}

	return page
}

func validatePagination(cursor, sortOrder string, limit, page int) error {
	maxPaginationLimit := libCommons.SafeInt64ToInt(pkg.GetenvIntOrDefault("MAX_PAGINATION_LIMIT", 100))

	if limit < 1 {
		return pkg.ValidateBusinessError(constant.ErrInvalidQueryParameter, "", "limit must be greater than 0")
	}

	if limit > maxPaginationLimit {
		return pkg.ValidateBusinessError(constant.ErrPaginationLimitExceeded, "", maxPaginationLimit)
	}

	if page < 1 {
		return pkg.ValidateBusinessError(constant.ErrInvalidQueryParameter, "", "page must be greater than 0")
	}

	if (sortOrder != string(constant.Asc)) && (sortOrder != string(constant.Desc)) {
		return pkg.ValidateBusinessError(constant.ErrInvalidSortOrder, "")
	}

	if !libCommons.IsNilOrEmpty(&cursor) {
		_, err := DecodeCursor(cursor)
		if err != nil {
			return pkg.ValidateBusinessError(constant.ErrInvalidQueryParameter, "", "cursor")
		}
	}

	return nil
}
