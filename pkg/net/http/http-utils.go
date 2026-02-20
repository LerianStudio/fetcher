package http

import (
	"strconv"
	"strings"
	"time"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	libCommons "github.com/LerianStudio/lib-commons/commons"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

// QueryHeader entity from query parameter from get apis
type QueryHeader struct {
	Metadata    *bson.M
	Limit       int
	Page        int
	Cursor      string
	SortOrder   string
	StartDate   time.Time
	EndDate     time.Time
	UseMetadata bool
	ProductID   *uuid.UUID
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
		metadata    = bson.M{}
		startDate   time.Time
		endDate     time.Time
		cursor      string
		limit       = 10
		page        = 1
		sortOrder   = "desc"
		useMetadata = false
	)

	if err := parseParameters(params, metadata, &startDate, &endDate, &cursor, &limit, &page, &sortOrder); err != nil {
		return nil, err
	}

	var metadataPtr *bson.M
	if len(metadata) > 0 {
		metadataPtr = &metadata
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
		Metadata:    metadataPtr,
		Limit:       limit,
		Page:        page,
		Cursor:      cursor,
		SortOrder:   sortOrder,
		StartDate:   startDate,
		EndDate:     endDate,
		UseMetadata: useMetadata,
	}

	return query, nil
}

func parseParameters(
	params map[string]string,
	metadata bson.M,
	startDate, endDate *time.Time,
	cursor *string,
	limit, page *int,
	sortOrder *string,
) error {
	for key, value := range params {
		if value == "" {
			continue
		}

		switch {
		case strings.HasPrefix(key, "metadata."):
			metadata[key] = value
		case key == "limit":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return pkg.ValidateBusinessError(constant.ErrInvalidQueryParameter, "limit")
			}

			*limit = parsed
		case key == "page":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return pkg.ValidateBusinessError(constant.ErrInvalidQueryParameter, "page")
			}

			*page = parsed
		case key == "cursor":
			*cursor = value
		case key == "sortOrder":
			*sortOrder = strings.ToLower(value)
		case key == "startDate":
			parsed, err := time.Parse("2006-01-02", value)
			if err != nil {
				return pkg.ValidateBusinessError(constant.ErrInvalidDateFormat, "startDate")
			}

			*startDate = parsed
		case key == "endDate":
			parsed, err := time.Parse("2006-01-02", value)
			if err != nil {
				return pkg.ValidateBusinessError(constant.ErrInvalidDateFormat, "endDate")
			}

			*endDate = parsed
		default:
			// Capture unrecognized keys as metadata filters so callers
			// can pass domain-level filters (e.g. status, category)
			// without requiring the "metadata." prefix.
			//
			// Security: reject keys that start with "$" to prevent MongoDB
			// operator injection (e.g. $where, $ne, $regex). Also reject
			// keys starting with underscore (internal fields like _id).
			if strings.HasPrefix(key, "$") || strings.HasPrefix(key, "_") {
				return pkg.ValidateBusinessError(constant.ErrInvalidQueryParameter, key)
			}

			// Cap key/value length to prevent abuse via oversized filter payloads.
			const (
				maxFilterKeyLen   = 64
				maxFilterValueLen = 256
			)

			if len(key) > maxFilterKeyLen || len(value) > maxFilterValueLen {
				return pkg.ValidateBusinessError(constant.ErrInvalidQueryParameter, key)
			}

			metadata[key] = value
		}
	}

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

// GetOrganizationID extracts and validates X-Organization-Id header as UUID.
func GetOrganizationID(c *fiber.Ctx) (uuid.UUID, error) {
	orgHeader := strings.TrimSpace(c.Get("X-Organization-Id"))

	orgID, err := uuid.Parse(orgHeader)
	if err != nil {
		return uuid.Nil, pkg.ValidationError{
			EntityType: "request",
			Code:       constant.ErrInvalidHeaderParameter.Error(),
			Title:      "Invalid header",
			Message:    "X-Organization-Id header is required and must be a valid UUID",
			Err:        err,
		}
	}

	return orgID, nil
}

// GetProductID extracts and validates X-Product-Id header as UUID.
// Returns nil and nil error if the header is not provided (optional header).
func GetProductID(c *fiber.Ctx) (*uuid.UUID, error) {
	productHeader := strings.TrimSpace(c.Get("X-Product-Id"))
	if productHeader == "" {
		return nil, nil
	}

	productID, err := uuid.Parse(productHeader)
	if err != nil {
		return nil, pkg.ValidationError{
			EntityType: "request",
			Code:       constant.ErrInvalidHeaderParameter.Error(),
			Title:      "Invalid header",
			Message:    "X-Product-Id header must be a valid UUID",
			Err:        err,
		}
	}

	return &productID, nil
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
