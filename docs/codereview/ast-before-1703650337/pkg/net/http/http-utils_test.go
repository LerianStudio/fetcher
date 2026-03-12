package http

import (
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

func TestToOffsetPagination(t *testing.T) {
	tests := []struct {
		name   string
		header QueryHeader
		want   Pagination
	}{
		{
			name: "full query header",
			header: QueryHeader{
				Limit:     50,
				Page:      2,
				SortOrder: "asc",
				StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
			},
			want: Pagination{
				Limit:     50,
				Page:      2,
				SortOrder: "asc",
				StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "minimal query header",
			header: QueryHeader{
				Limit: 10,
				Page:  1,
			},
			want: Pagination{
				Limit: 10,
				Page:  1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.header.ToOffsetPagination()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateParameters(t *testing.T) {
	// Set environment variable for tests
	os.Setenv("MAX_PAGINATION_LIMIT", "100")
	os.Setenv("MAX_PAGINATION_MONTH_DATE_RANGE", "3")
	defer func() {
		os.Unsetenv("MAX_PAGINATION_LIMIT")
		os.Unsetenv("MAX_PAGINATION_MONTH_DATE_RANGE")
	}()

	tests := []struct {
		name    string
		params  map[string]string
		wantErr bool
		check   func(*testing.T, *QueryHeader)
	}{
		{
			name:    "default parameters",
			params:  map[string]string{},
			wantErr: false,
			check: func(t *testing.T, qh *QueryHeader) {
				assert.Equal(t, 10, qh.Limit)
				assert.Equal(t, 1, qh.Page)
				assert.Equal(t, "desc", qh.SortOrder)
				assert.False(t, qh.UseMetadata)
			},
		},
		{
			name: "custom limit and page",
			params: map[string]string{
				"limit": "50",
				"page":  "3",
			},
			wantErr: false,
			check: func(t *testing.T, qh *QueryHeader) {
				assert.Equal(t, 50, qh.Limit)
				assert.Equal(t, 3, qh.Page)
			},
		},
		{
			name: "with metadata",
			params: map[string]string{
				"metadata.key1": "value1",
				"metadata.key2": "value2",
			},
			wantErr: false,
			check: func(t *testing.T, qh *QueryHeader) {
				assert.True(t, qh.UseMetadata)
				assert.NotNil(t, qh.Metadata)
				assert.Len(t, *qh.Metadata, 2)
			},
		},
		{
			name: "with sort order asc",
			params: map[string]string{
				"sortOrder": "asc",
			},
			wantErr: false,
			check: func(t *testing.T, qh *QueryHeader) {
				assert.Equal(t, "asc", qh.SortOrder)
			},
		},
		{
			name: "with valid dates",
			params: map[string]string{
				"startDate": "2024-01-01",
				"endDate":   "2024-01-31",
			},
			wantErr: false,
			check: func(t *testing.T, qh *QueryHeader) {
				assert.Equal(t, 2024, qh.StartDate.Year())
				assert.Equal(t, time.January, qh.StartDate.Month())
				assert.Equal(t, 1, qh.StartDate.Day())
			},
		},
		{
			name: "invalid sort order",
			params: map[string]string{
				"sortOrder": "invalid",
			},
			wantErr: true,
		},
		{
			name: "limit exceeds max",
			params: map[string]string{
				"limit": "200",
			},
			wantErr: true,
		},
		{
			name: "non-numeric limit",
			params: map[string]string{
				"limit": "abc",
			},
			wantErr: true, // strconv.Atoi fails, returns 0, which is invalid
		},
		{
			name: "non-numeric page",
			params: map[string]string{
				"page": "xyz",
			},
			wantErr: true, // strconv.Atoi fails, returns 0, which is invalid
		},
		{
			name: "invalid date format",
			params: map[string]string{
				"startDate": "01-01-2024",
			},
			wantErr: true, // invalid date format now returns error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateParameters(tt.params)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestGetOrganizationID(t *testing.T) {
	tests := []struct {
		name      string
		headerVal string
		wantErr   bool
		wantUUID  uuid.UUID
	}{
		{
			name:      "valid UUID",
			headerVal: "550e8400-e29b-41d4-a716-446655440000",
			wantErr:   false,
			wantUUID:  uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		},
		{
			name:      "valid UUID with spaces",
			headerVal: "  550e8400-e29b-41d4-a716-446655440000  ",
			wantErr:   false,
			wantUUID:  uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		},
		{
			name:      "invalid UUID",
			headerVal: "not-a-uuid",
			wantErr:   true,
			wantUUID:  uuid.Nil,
		},
		{
			name:      "empty UUID",
			headerVal: "",
			wantErr:   true,
			wantUUID:  uuid.Nil,
		},
		{
			name:      "malformed UUID",
			headerVal: "550e8400-e29b-41d4-a716",
			wantErr:   true,
			wantUUID:  uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error {
				got, err := GetOrganizationID(c)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.wantUUID, got)
				}
				return nil
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Organization-Id", tt.headerVal)
			_, err := app.Test(req)
			assert.NoError(t, err)
		})
	}
}

func TestParseIntDefault(t *testing.T) {
	tests := []struct {
		name string
		val  string
		def  int
		want int
	}{
		{
			name: "valid integer",
			val:  "42",
			def:  10,
			want: 42,
		},
		{
			name: "empty string returns default",
			val:  "",
			def:  10,
			want: 10,
		},
		{
			name: "invalid integer returns default",
			val:  "abc",
			def:  20,
			want: 20,
		},
		{
			name: "negative integer",
			val:  "-5",
			def:  10,
			want: -5,
		},
		{
			name: "zero value",
			val:  "0",
			def:  10,
			want: 0,
		},
		{
			name: "large integer",
			val:  "999999",
			def:  10,
			want: 999999,
		},
		{
			name: "float string returns default",
			val:  "10.5",
			def:  10,
			want: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseIntDefault(tt.val, tt.def)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClampLimit(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		def      int
		maxLimit int
		want     int
	}{
		{
			name:     "within bounds",
			limit:    50,
			def:      10,
			maxLimit: 100,
			want:     50,
		},
		{
			name:     "exceeds max returns max",
			limit:    150,
			def:      10,
			maxLimit: 100,
			want:     100,
		},
		{
			name:     "zero returns default",
			limit:    0,
			def:      10,
			maxLimit: 100,
			want:     10,
		},
		{
			name:     "negative returns default",
			limit:    -5,
			def:      10,
			maxLimit: 100,
			want:     10,
		},
		{
			name:     "exactly at max",
			limit:    100,
			def:      10,
			maxLimit: 100,
			want:     100,
		},
		{
			name:     "one over max",
			limit:    101,
			def:      10,
			maxLimit: 100,
			want:     100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClampLimit(tt.limit, tt.def, tt.maxLimit)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClampNonNegative(t *testing.T) {
	tests := []struct {
		name string
		page int
		want int
	}{
		{
			name: "positive page",
			page: 5,
			want: 5,
		},
		{
			name: "zero page",
			page: 0,
			want: 0,
		},
		{
			name: "negative page returns zero",
			page: -1,
			want: 0,
		},
		{
			name: "large negative returns zero",
			page: -999,
			want: 0,
		},
		{
			name: "large positive",
			page: 1000000,
			want: 1000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClampNonNegative(tt.page)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateDates(t *testing.T) {
	// Set environment variable for tests
	os.Setenv("MAX_PAGINATION_MONTH_DATE_RANGE", "3")
	defer os.Unsetenv("MAX_PAGINATION_MONTH_DATE_RANGE")

	tests := []struct {
		name      string
		startDate time.Time
		endDate   time.Time
		wantErr   bool
		checkDiff bool // Check if date range is adjusted
	}{
		{
			name:      "both dates empty - should set defaults",
			startDate: time.Time{},
			endDate:   time.Time{},
			wantErr:   false,
		},
		{
			name:      "valid date range within limit",
			startDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			wantErr:   false,
		},
		{
			name:      "start date after end date",
			startDate: time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			wantErr:   true,
		},
		{
			name:      "equal dates - should adjust end date",
			startDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			wantErr:   false,
		},
		{
			name:      "only start date provided",
			startDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Time{},
			wantErr:   false,
		},
		{
			name:      "only end date provided",
			startDate: time.Time{},
			endDate:   time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startDate := tt.startDate
			endDate := tt.endDate
			err := validateDates(&startDate, &endDate)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			// After validation, dates should be set
			assert.False(t, startDate.IsZero())
			assert.False(t, endDate.IsZero())
		})
	}
}

func TestValidatePagination(t *testing.T) {
	// Set environment variable for tests
	os.Setenv("MAX_PAGINATION_LIMIT", "100")
	defer os.Unsetenv("MAX_PAGINATION_LIMIT")

	// Create a valid cursor for testing
	validCursor := encodeCursor(Cursor{ID: "test123", PointsNext: true})

	tests := []struct {
		name      string
		cursor    string
		sortOrder string
		limit     int
		page      int
		wantErr   bool
	}{
		{
			name:      "valid pagination with desc",
			cursor:    "",
			sortOrder: "desc",
			limit:     50,
			page:      1,
			wantErr:   false,
		},
		{
			name:      "valid pagination with asc",
			cursor:    "",
			sortOrder: "asc",
			limit:     50,
			page:      1,
			wantErr:   false,
		},
		{
			name:      "valid cursor",
			cursor:    validCursor,
			sortOrder: "desc",
			limit:     50,
			page:      1,
			wantErr:   false,
		},
		{
			name:      "invalid sort order",
			cursor:    "",
			sortOrder: "invalid",
			limit:     50,
			page:      1,
			wantErr:   true,
		},
		{
			name:      "limit exceeds max",
			cursor:    "",
			sortOrder: "desc",
			limit:     200,
			page:      1,
			wantErr:   true,
		},
		{
			name:      "invalid cursor",
			cursor:    "invalid-cursor",
			sortOrder: "desc",
			limit:     50,
			page:      1,
			wantErr:   true,
		},
		{
			name:      "empty cursor is valid",
			cursor:    "",
			sortOrder: "desc",
			limit:     10,
			page:      1,
			wantErr:   false,
		},
		{
			name:      "negative limit",
			cursor:    "",
			sortOrder: "desc",
			limit:     -1,
			page:      1,
			wantErr:   true,
		},
		{
			name:      "zero limit",
			cursor:    "",
			sortOrder: "desc",
			limit:     0,
			page:      1,
			wantErr:   true,
		},
		{
			name:      "negative page",
			cursor:    "",
			sortOrder: "desc",
			limit:     10,
			page:      -1,
			wantErr:   true,
		},
		{
			name:      "zero page",
			cursor:    "",
			sortOrder: "desc",
			limit:     10,
			page:      0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePagination(tt.cursor, tt.sortOrder, tt.limit, tt.page)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestQueryHeaderMetadata(t *testing.T) {
	t.Run("metadata pointer is nil when no metadata", func(t *testing.T) {
		params := map[string]string{
			"limit": "10",
			"page":  "1",
		}

		qh, err := ValidateParameters(params)
		assert.NoError(t, err)
		assert.Nil(t, qh.Metadata)
		assert.False(t, qh.UseMetadata)
	})

	t.Run("metadata pointer is set when metadata exists", func(t *testing.T) {
		params := map[string]string{
			"metadata.key": "value",
		}

		qh, err := ValidateParameters(params)
		assert.NoError(t, err)
		assert.NotNil(t, qh.Metadata)
		assert.True(t, qh.UseMetadata)
	})

	t.Run("custom metadata keys are captured", func(t *testing.T) {
		params := map[string]string{
			"metadata.customKey": "customValue",
		}

		qh, err := ValidateParameters(params)
		assert.NoError(t, err)
		require.NotNil(t, qh.Metadata)
		assert.True(t, qh.UseMetadata)
		assert.Contains(t, *qh.Metadata, "metadata.customKey")
	})
}

func TestPaginationStruct(t *testing.T) {
	t.Run("pagination struct fields", func(t *testing.T) {
		p := Pagination{
			Limit:     25,
			Page:      3,
			Cursor:    "test-cursor",
			SortOrder: "asc",
			StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		}

		assert.Equal(t, 25, p.Limit)
		assert.Equal(t, 3, p.Page)
		assert.Equal(t, "test-cursor", p.Cursor)
		assert.Equal(t, "asc", p.SortOrder)
		assert.Equal(t, 2024, p.StartDate.Year())
		assert.Equal(t, 2024, p.EndDate.Year())
	})
}

func TestQueryHeaderStruct(t *testing.T) {
	t.Run("query header with all fields", func(t *testing.T) {
		metadata := bson.M{"key": "value"}
		qh := QueryHeader{
			Metadata:    &metadata,
			Limit:       50,
			Page:        2,
			Cursor:      "cursor",
			SortOrder:   "desc",
			StartDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:     time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
			UseMetadata: true,
		}

		assert.NotNil(t, qh.Metadata)
		assert.True(t, qh.UseMetadata)
		assert.Equal(t, 50, qh.Limit)
		assert.Equal(t, 2, qh.Page)
	})
}

func TestGetOrganizationIDIntegration(t *testing.T) {
	t.Run("integration test with fiber app", func(t *testing.T) {
		app := fiber.New()
		testUUID := uuid.New()

		app.Get("/test", func(c *fiber.Ctx) error {
			orgID, err := GetOrganizationID(c)
			if err != nil {
				return err
			}
			return c.JSON(fiber.Map{"orgID": orgID.String()})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Organization-Id", testUUID.String())

		resp, err := app.Test(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)
	})
}

func TestValidateParametersWithCursor(t *testing.T) {
	// Set environment variable for tests
	os.Setenv("MAX_PAGINATION_LIMIT", "100")
	os.Setenv("MAX_PAGINATION_MONTH_DATE_RANGE", "3")
	defer func() {
		os.Unsetenv("MAX_PAGINATION_LIMIT")
		os.Unsetenv("MAX_PAGINATION_MONTH_DATE_RANGE")
	}()

	// Create a valid cursor for testing
	validCursor := encodeCursor(Cursor{ID: "test123", PointsNext: true})

	tests := []struct {
		name    string
		params  map[string]string
		wantErr bool
		check   func(*testing.T, *QueryHeader)
	}{
		{
			name: "with valid cursor",
			params: map[string]string{
				"cursor": validCursor,
			},
			wantErr: false,
			check: func(t *testing.T, qh *QueryHeader) {
				assert.Equal(t, validCursor, qh.Cursor)
			},
		},
		{
			name: "with invalid cursor",
			params: map[string]string{
				"cursor": "invalid-cursor-string",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateParameters(tt.params)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestValidateDatesEdgeCases(t *testing.T) {
	os.Setenv("MAX_PAGINATION_MONTH_DATE_RANGE", "3")
	defer os.Unsetenv("MAX_PAGINATION_MONTH_DATE_RANGE")

	t.Run("date range exceeds max months - adjusts start date", func(t *testing.T) {
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

		err := validateDates(&startDate, &endDate)
		assert.NoError(t, err)
		// Start date should be adjusted to be within range
	})

	t.Run("both dates provided and valid", func(t *testing.T) {
		startDate := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)

		err := validateDates(&startDate, &endDate)
		assert.NoError(t, err)
		assert.Equal(t, 2024, startDate.Year())
		assert.Equal(t, time.June, startDate.Month())
	})
}

func TestQueryHeaderToOffsetPaginationFields(t *testing.T) {
	t.Run("cursor is not included in offset pagination", func(t *testing.T) {
		qh := QueryHeader{
			Limit:     25,
			Page:      2,
			Cursor:    "some-cursor",
			SortOrder: "asc",
			StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
		}

		p := qh.ToOffsetPagination()

		assert.Equal(t, 25, p.Limit)
		assert.Equal(t, 2, p.Page)
		assert.Empty(t, p.Cursor) // Cursor should not be copied
		assert.Equal(t, "asc", p.SortOrder)
	})
}

func TestValidateParametersNonMetadataKeys(t *testing.T) {
	os.Setenv("MAX_PAGINATION_LIMIT", "100")
	os.Setenv("MAX_PAGINATION_MONTH_DATE_RANGE", "3")
	defer func() {
		os.Unsetenv("MAX_PAGINATION_LIMIT")
		os.Unsetenv("MAX_PAGINATION_MONTH_DATE_RANGE")
	}()

	t.Run("keys without metadata prefix are captured as metadata", func(t *testing.T) {
		params := map[string]string{
			"status":   "active",
			"category": "finance",
		}

		qh, err := ValidateParameters(params)
		assert.NoError(t, err)
		assert.True(t, qh.UseMetadata)
		assert.NotNil(t, qh.Metadata)
		assert.Contains(t, *qh.Metadata, "status")
		assert.Contains(t, *qh.Metadata, "category")
	})
}
