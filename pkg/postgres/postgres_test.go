package postgres

import (
	"testing"

	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/fetcher/pkg/testutil"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
)

func TestIsFilterConditionEmpty(t *testing.T) {
	tests := []struct {
		name      string
		condition job.FilterCondition
		want      bool
	}{
		{
			name:      "empty condition",
			condition: job.FilterCondition{},
			want:      true,
		},
		{
			name: "condition with equals",
			condition: job.FilterCondition{
				Equals: []any{"value"},
			},
			want: false,
		},
		{
			name: "condition with greater than",
			condition: job.FilterCondition{
				GreaterThan: []any{100},
			},
			want: false,
		},
		{
			name: "condition with greater or equal",
			condition: job.FilterCondition{
				GreaterOrEqual: []any{50},
			},
			want: false,
		},
		{
			name: "condition with less than",
			condition: job.FilterCondition{
				LessThan: []any{200},
			},
			want: false,
		},
		{
			name: "condition with less or equal",
			condition: job.FilterCondition{
				LessOrEqual: []any{150},
			},
			want: false,
		},
		{
			name: "condition with between",
			condition: job.FilterCondition{
				Between: []any{10, 20},
			},
			want: false,
		},
		{
			name: "condition with in",
			condition: job.FilterCondition{
				In: []any{"a", "b", "c"},
			},
			want: false,
		},
		{
			name: "condition with not in",
			condition: job.FilterCondition{
				NotIn: []any{"x", "y"},
			},
			want: false,
		},
		{
			name: "condition with multiple operators",
			condition: job.FilterCondition{
				Equals:      []any{"value"},
				GreaterThan: []any{10},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isFilterConditionEmpty(tt.condition)
			if got != tt.want {
				t.Errorf("isFilterConditionEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateFilterCondition(t *testing.T) {
	validUUID := uuid.New().String()
	tests := []struct {
		name      string
		fieldName string
		condition job.FilterCondition
		wantErr   bool
	}{
		{
			name:      "valid between with 2 values",
			fieldName: "age",
			condition: job.FilterCondition{
				Between: []any{10, 20},
			},
			wantErr: false,
		},
		{
			name:      "invalid between with 1 value",
			fieldName: "age",
			condition: job.FilterCondition{
				Between: []any{10},
			},
			wantErr: true,
		},
		{
			name:      "invalid between with 3 values",
			fieldName: "age",
			condition: job.FilterCondition{
				Between: []any{10, 20, 30},
			},
			wantErr: true,
		},
		{
			name:      "valid greater than with 1 value",
			fieldName: "age",
			condition: job.FilterCondition{
				GreaterThan: []any{10},
			},
			wantErr: false,
		},
		{
			name:      "invalid greater than with 2 values",
			fieldName: "age",
			condition: job.FilterCondition{
				GreaterThan: []any{10, 20},
			},
			wantErr: true,
		},
		{
			name:      "valid greater or equal with 1 value",
			fieldName: "age",
			condition: job.FilterCondition{
				GreaterOrEqual: []any{10},
			},
			wantErr: false,
		},
		{
			name:      "invalid greater or equal with 2 values",
			fieldName: "age",
			condition: job.FilterCondition{
				GreaterOrEqual: []any{10, 20},
			},
			wantErr: true,
		},
		{
			name:      "valid less than with 1 value",
			fieldName: "age",
			condition: job.FilterCondition{
				LessThan: []any{100},
			},
			wantErr: false,
		},
		{
			name:      "invalid less than with 3 values",
			fieldName: "age",
			condition: job.FilterCondition{
				LessThan: []any{10, 20, 30},
			},
			wantErr: true,
		},
		{
			name:      "valid less or equal with 1 value",
			fieldName: "age",
			condition: job.FilterCondition{
				LessOrEqual: []any{100},
			},
			wantErr: false,
		},
		{
			name:      "invalid less or equal with 2 values",
			fieldName: "age",
			condition: job.FilterCondition{
				LessOrEqual: []any{10, 20},
			},
			wantErr: true,
		},
		{
			name:      "valid UUID field with UUID value",
			fieldName: "user_id",
			condition: job.FilterCondition{
				Equals: []any{validUUID},
			},
			wantErr: false,
		},
		{
			name:      "invalid UUID field with non-UUID value",
			fieldName: "user_id",
			condition: job.FilterCondition{
				Equals: []any{"not-a-uuid"},
			},
			wantErr: true,
		},
		{
			name:      "valid UUID field with multiple UUIDs in In operator",
			fieldName: "account_id",
			condition: job.FilterCondition{
				In: []any{validUUID, uuid.New().String()},
			},
			wantErr: false,
		},
		{
			name:      "invalid UUID field with non-UUID in NotIn operator",
			fieldName: "template_id",
			condition: job.FilterCondition{
				NotIn: []any{"invalid-uuid"},
			},
			wantErr: true,
		},
		{
			name:      "valid equals with multiple values",
			fieldName: "status",
			condition: job.FilterCondition{
				Equals: []any{"active", "pending", "completed"},
			},
			wantErr: false,
		},
		{
			name:      "valid in with multiple values",
			fieldName: "category",
			condition: job.FilterCondition{
				In: []any{"A", "B", "C"},
			},
			wantErr: false,
		},
		{
			name:      "valid not in with multiple values",
			fieldName: "type",
			condition: job.FilterCondition{
				NotIn: []any{"deleted", "archived"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilterCondition(tt.fieldName, tt.condition)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFilterCondition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsLikelyUUIDField(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		want      bool
	}{
		{
			name:      "field with id suffix",
			fieldName: "user_id",
			want:      true,
		},
		{
			name:      "field with uuid",
			fieldName: "uuid",
			want:      true,
		},
		{
			name:      "field with template_id",
			fieldName: "template_id",
			want:      true,
		},
		{
			name:      "field with organization_id",
			fieldName: "organization_id",
			want:      true,
		},
		{
			name:      "field with account_id",
			fieldName: "account_id",
			want:      true,
		},
		{
			name:      "plain id field",
			fieldName: "id",
			want:      true,
		},
		{
			name:      "field with _id suffix",
			fieldName: "product_id",
			want:      true,
		},
		{
			name:      "uppercase UUID field",
			fieldName: "USER_ID",
			want:      true,
		},
		{
			name:      "mixed case UUID field",
			fieldName: "UserId",
			want:      true,
		},
		{
			name:      "non-UUID field name",
			fieldName: "name",
			want:      false,
		},
		{
			name:      "non-UUID field email",
			fieldName: "email",
			want:      false,
		},
		{
			name:      "non-UUID field status",
			fieldName: "status",
			want:      false,
		},
		{
			name:      "non-UUID field created_at",
			fieldName: "created_at",
			want:      false,
		},
		{
			name:      "non-UUID field description",
			fieldName: "description",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLikelyUUIDField(tt.fieldName)
			if got != tt.want {
				t.Errorf("isLikelyUUIDField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidUUIDFormat(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{
			name: "valid UUID v4",
			s:    "550e8400-e29b-41d4-a716-446655440000",
			want: true,
		},
		{
			name: "valid UUID with uppercase",
			s:    "550E8400-E29B-41D4-A716-446655440000",
			want: true,
		},
		{
			name: "valid UUID generated",
			s:    uuid.New().String(),
			want: true,
		},
		{
			name: "invalid UUID - too short",
			s:    "550e8400-e29b-41d4-a716",
			want: false,
		},
		{
			name: "invalid UUID - wrong format",
			s:    "not-a-uuid",
			want: false,
		},
		{
			name: "invalid UUID - empty string",
			s:    "",
			want: false,
		},
		{
			name: "invalid UUID - random string",
			s:    "abcdefghijklmnopqrstuvwxyz",
			want: false,
		},
		{
			name: "valid UUID - no hyphens (accepted by uuid.Parse)",
			s:    "550e8400e29b41d4a716446655440000",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidUUIDFormat(tt.s)
			if got != tt.want {
				t.Errorf("isValidUUIDFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsDateField(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		want      bool
	}{
		{
			name:      "field with created_at",
			fieldName: "created_at",
			want:      true,
		},
		{
			name:      "field with updated_at",
			fieldName: "updated_at",
			want:      true,
		},
		{
			name:      "field with deleted_at",
			fieldName: "deleted_at",
			want:      true,
		},
		{
			name:      "field with completed_at",
			fieldName: "completed_at",
			want:      true,
		},
		{
			name:      "field with date",
			fieldName: "birth_date",
			want:      true,
		},
		{
			name:      "field with time",
			fieldName: "start_time",
			want:      true,
		},
		{
			name:      "field with _at suffix",
			fieldName: "published_at",
			want:      true,
		},
		{
			name:      "field with _date suffix",
			fieldName: "expiry_date",
			want:      true,
		},
		{
			name:      "field with _time suffix",
			fieldName: "access_time",
			want:      true,
		},
		{
			name:      "uppercase date field",
			fieldName: "CREATED_AT",
			want:      true,
		},
		{
			name:      "mixed case date field with _at",
			fieldName: "created_at_time",
			want:      true,
		},
		{
			name:      "non-date field name",
			fieldName: "name",
			want:      false,
		},
		{
			name:      "non-date field id",
			fieldName: "id",
			want:      false,
		},
		{
			name:      "non-date field status",
			fieldName: "status",
			want:      false,
		},
		{
			name:      "non-date field email",
			fieldName: "email",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDateField(tt.fieldName)
			if got != tt.want {
				t.Errorf("isDateField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsDateString(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  bool
	}{
		{
			name:  "valid date YYYY-MM-DD",
			value: "2023-12-25",
			want:  true,
		},
		{
			name:  "valid datetime with T separator",
			value: "2023-12-25T10:30:00",
			want:  true,
		},
		{
			name:  "valid datetime with timezone",
			value: "2023-12-25T10:30:00Z",
			want:  true,
		},
		{
			name:  "valid datetime ISO 8601",
			value: "2023-12-25T10:30:00.000Z",
			want:  true,
		},
		{
			name:  "valid datetime with offset",
			value: "2023-12-25T10:30:00+00:00",
			want:  true,
		},
		{
			name:  "invalid date string - too short",
			value: "2023-12",
			want:  false,
		},
		{
			name:  "invalid date string - no hyphens",
			value: "20231225",
			want:  false,
		},
		{
			name:  "invalid date string - wrong format",
			value: "25/12/2023",
			want:  false,
		},
		{
			name:  "not a string value - int",
			value: 12345,
			want:  false,
		},
		{
			name:  "not a string value - bool",
			value: true,
			want:  false,
		},
		{
			name:  "not a string value - nil",
			value: nil,
			want:  false,
		},
		{
			name:  "empty string",
			value: "",
			want:  false,
		},
		{
			name:  "random string",
			value: "not a date",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDateString(tt.value)
			if got != tt.want {
				t.Errorf("isDateString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateUUIDFieldValues(t *testing.T) {
	validUUID1 := uuid.New().String()
	validUUID2 := uuid.New().String()

	tests := []struct {
		name      string
		fieldName string
		condition job.FilterCondition
		wantErr   bool
	}{
		{
			name:      "valid UUID in Equals",
			fieldName: "user_id",
			condition: job.FilterCondition{
				Equals: []any{validUUID1},
			},
			wantErr: false,
		},
		{
			name:      "invalid UUID in Equals",
			fieldName: "user_id",
			condition: job.FilterCondition{
				Equals: []any{"not-a-uuid"},
			},
			wantErr: true,
		},
		{
			name:      "valid UUIDs in GreaterThan",
			fieldName: "user_id",
			condition: job.FilterCondition{
				GreaterThan: []any{validUUID1},
			},
			wantErr: false,
		},
		{
			name:      "invalid UUID in GreaterOrEqual",
			fieldName: "user_id",
			condition: job.FilterCondition{
				GreaterOrEqual: []any{"12345"},
			},
			wantErr: true,
		},
		{
			name:      "valid UUID in LessThan",
			fieldName: "user_id",
			condition: job.FilterCondition{
				LessThan: []any{validUUID1},
			},
			wantErr: false,
		},
		{
			name:      "invalid UUID in LessOrEqual",
			fieldName: "user_id",
			condition: job.FilterCondition{
				LessOrEqual: []any{"abc-def"},
			},
			wantErr: true,
		},
		{
			name:      "valid UUIDs in Between",
			fieldName: "user_id",
			condition: job.FilterCondition{
				Between: []any{validUUID1, validUUID2},
			},
			wantErr: false,
		},
		{
			name:      "invalid UUID in Between",
			fieldName: "user_id",
			condition: job.FilterCondition{
				Between: []any{validUUID1, "not-uuid"},
			},
			wantErr: true,
		},
		{
			name:      "valid UUIDs in In",
			fieldName: "user_id",
			condition: job.FilterCondition{
				In: []any{validUUID1, validUUID2},
			},
			wantErr: false,
		},
		{
			name:      "invalid UUID in In",
			fieldName: "user_id",
			condition: job.FilterCondition{
				In: []any{validUUID1, "bad-uuid"},
			},
			wantErr: true,
		},
		{
			name:      "valid UUIDs in NotIn",
			fieldName: "user_id",
			condition: job.FilterCondition{
				NotIn: []any{validUUID1, validUUID2},
			},
			wantErr: false,
		},
		{
			name:      "invalid UUID in NotIn",
			fieldName: "user_id",
			condition: job.FilterCondition{
				NotIn: []any{"invalid"},
			},
			wantErr: true,
		},
		{
			name:      "mixed valid and invalid UUIDs",
			fieldName: "user_id",
			condition: job.FilterCondition{
				Equals: []any{validUUID1, "invalid-uuid"},
			},
			wantErr: true,
		},
		{
			name:      "non-string value should pass",
			fieldName: "user_id",
			condition: job.FilterCondition{
				Equals: []any{12345},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUUIDFieldValues(tt.fieldName, tt.condition)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateUUIDFieldValues() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateRowMap(t *testing.T) {
	mockLogger := &testutil.MockLogger{}

	tests := []struct {
		name    string
		columns []string
		values  []any
		want    map[string]any
	}{
		{
			name:    "simple string values",
			columns: []string{"name", "email"},
			values:  []any{"John Doe", "john@example.com"},
			want: map[string]any{
				"name":  "John Doe",
				"email": "john@example.com",
			},
		},
		{
			name:    "mixed types",
			columns: []string{"id", "name", "age", "active"},
			values:  []any{123, "Alice", 30, true},
			want: map[string]any{
				"id":     123,
				"name":   "Alice",
				"age":    30,
				"active": true,
			},
		},
		{
			name:    "with nil values",
			columns: []string{"name", "email", "phone"},
			values:  []any{"Bob", nil, "555-1234"},
			want: map[string]any{
				"name":  "Bob",
				"email": nil,
				"phone": "555-1234",
			},
		},
		{
			name:    "with JSON byte data",
			columns: []string{"id", "metadata"},
			values:  []any{1, []uint8(`{"key":"value"}`)},
			want: map[string]any{
				"id":       1,
				"metadata": map[string]any{"key": "value"},
			},
		},
		{
			name:    "empty columns and values",
			columns: []string{},
			values:  []any{},
			want:    map[string]any{},
		},
		{
			name:    "single column",
			columns: []string{"count"},
			values:  []any{42},
			want: map[string]any{
				"count": 42,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createRowMap(testutil.TestContext(), tt.columns, tt.values, mockLogger)

			if len(got) != len(tt.want) {
				t.Errorf("createRowMap() returned map with %d keys, want %d", len(got), len(tt.want))
				return
			}

			for key, wantVal := range tt.want {
				gotVal, ok := got[key]
				if !ok {
					t.Errorf("createRowMap() missing key %q", key)
					continue
				}

				// Use valuesEqual for deep comparison
				if gotMap, ok := gotVal.(map[string]any); ok {
					if wantMap, ok := wantVal.(map[string]any); ok {
						if !mapsEqual(gotMap, wantMap) {
							t.Errorf("createRowMap()[%q] = %v, want %v", key, gotVal, wantVal)
						}
					}
				} else if gotVal != wantVal {
					t.Errorf("createRowMap()[%q] = %v, want %v", key, gotVal, wantVal)
				}
			}
		})
	}
}

func TestApplyFilter(t *testing.T) {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	tests := []struct {
		name       string
		fieldName  string
		values     []any
		wantClause string
	}{
		{
			name:       "single value",
			fieldName:  "status",
			values:     []any{"active"},
			wantClause: "status = ",
		},
		{
			name:       "multiple values",
			fieldName:  "id",
			values:     []any{1, 2, 3},
			wantClause: "id IN ",
		},
		{
			name:       "empty values returns unchanged",
			fieldName:  "name",
			values:     []any{},
			wantClause: "",
		},
		{
			name:       "single numeric value",
			fieldName:  "age",
			values:     []any{25},
			wantClause: "age = ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseQuery := psql.Select("*").From("users")
			resultQuery := applyFilter(baseQuery, tt.fieldName, tt.values)

			sql, args, err := resultQuery.ToSql()
			if err != nil {
				t.Fatalf("applyFilter() error generating SQL: %v", err)
			}

			if tt.wantClause == "" {
				// Empty values should return unchanged query
				baseSql, _, _ := baseQuery.ToSql()
				if sql != baseSql {
					t.Errorf("applyFilter() with empty values should return unchanged query")
				}
			} else {
				// Check if the WHERE clause contains our field
				if tt.wantClause != "" && !contains(sql, tt.wantClause) {
					t.Errorf("applyFilter() SQL = %q, want to contain %q", sql, tt.wantClause)
				}

				// Verify the number of arguments matches the number of values
				if len(tt.values) > 0 && len(args) != len(tt.values) {
					t.Errorf("applyFilter() args = %v (len %d), want length %d", args, len(args), len(tt.values))
				}
			}
		})
	}
}

func TestConnect_ErrorHandling(t *testing.T) {
	mockLogger := &testutil.MockLogger{}

	tests := []struct {
		name             string
		connectionString string
		wantErr          bool
	}{
		{
			name:             "invalid connection string",
			connectionString: "invalid://connection",
			wantErr:          true,
		},
		{
			name:             "empty connection string",
			connectionString: "",
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &Connection{
				ConnectionString:   tt.connectionString,
				Logger:             mockLogger,
				MaxOpenConnections: 10,
				MaxIdleConnections: 5,
			}

			err := conn.Connect()
			if (err != nil) != tt.wantErr {
				t.Errorf("Connect() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Connection should not be marked as connected on error
			if err != nil && conn.Connected {
				t.Errorf("Connect() failed but Connected = true, want false")
			}
		})
	}
}

func TestGetDB_WithNilConnection(t *testing.T) {
	mockLogger := &testutil.MockLogger{}

	conn := &Connection{
		ConnectionString:   "invalid://connection",
		Logger:             mockLogger,
		ConnectionDB:       nil,
		MaxOpenConnections: 10,
		MaxIdleConnections: 5,
	}

	db, err := conn.GetDB()
	if err == nil {
		t.Error("GetDB() with nil ConnectionDB should return error")
	}
	if db != nil {
		t.Error("GetDB() with error should return nil db")
	}
}

func TestBuildDynamicFilters(t *testing.T) {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	schema := []TableSchema{
		{
			TableName: "users",
			Columns: []ColumnInformation{
				{Name: "id"},
				{Name: "name"},
				{Name: "email"},
				{Name: "age"},
			},
		},
	}

	tests := []struct {
		name       string
		table      string
		filter     map[string][]any
		wantFields []string
	}{
		{
			name:  "filter by single field",
			table: "users",
			filter: map[string][]any{
				"name": {"John"},
			},
			wantFields: []string{"name"},
		},
		{
			name:  "filter by multiple fields",
			table: "users",
			filter: map[string][]any{
				"name": {"John"},
				"age":  {25},
			},
			wantFields: []string{"name", "age"},
		},
		{
			name:  "filter with multiple values for one field",
			table: "users",
			filter: map[string][]any{
				"id": {1, 2, 3},
			},
			wantFields: []string{"id"},
		},
		{
			name:       "empty filter",
			table:      "users",
			filter:     map[string][]any{},
			wantFields: []string{},
		},
		{
			name:  "filter with invalid field name (should be ignored)",
			table: "users",
			filter: map[string][]any{
				"invalid_field": {"value"},
				"name":          {"John"},
			},
			wantFields: []string{"name"},
		},
		{
			name:  "filter with empty values (should be ignored)",
			table: "users",
			filter: map[string][]any{
				"name": {},
			},
			wantFields: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseQuery := psql.Select("*").From(tt.table)
			resultQuery := buildDynamicFilters(baseQuery, schema, tt.table, tt.filter)

			sql, _, err := resultQuery.ToSql()
			if err != nil {
				t.Fatalf("buildDynamicFilters() error generating SQL: %v", err)
			}

			// Check that each expected field appears in the WHERE clause
			for _, field := range tt.wantFields {
				if !contains(sql, field) {
					t.Errorf("buildDynamicFilters() SQL = %q, want to contain field %q", sql, field)
				}
			}
		})
	}
}

func TestApplyAdvancedFilter(t *testing.T) {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	ds := &ExternalDataSource{}

	tests := []struct {
		name      string
		field     string
		condition job.FilterCondition
		wantErr   bool
	}{
		{
			name:  "equals with single value",
			field: "status",
			condition: job.FilterCondition{
				Equals: []any{"active"},
			},
			wantErr: false,
		},
		{
			name:  "equals with multiple values (IN clause)",
			field: "status",
			condition: job.FilterCondition{
				Equals: []any{"active", "pending", "completed"},
			},
			wantErr: false,
		},
		{
			name:  "greater than",
			field: "age",
			condition: job.FilterCondition{
				GreaterThan: []any{18},
			},
			wantErr: false,
		},
		{
			name:  "greater or equal",
			field: "age",
			condition: job.FilterCondition{
				GreaterOrEqual: []any{21},
			},
			wantErr: false,
		},
		{
			name:  "less than",
			field: "age",
			condition: job.FilterCondition{
				LessThan: []any{65},
			},
			wantErr: false,
		},
		{
			name:  "less or equal",
			field: "age",
			condition: job.FilterCondition{
				LessOrEqual: []any{100},
			},
			wantErr: false,
		},
		{
			name:  "between two values",
			field: "age",
			condition: job.FilterCondition{
				Between: []any{18, 65},
			},
			wantErr: false,
		},
		{
			name:  "in clause",
			field: "category",
			condition: job.FilterCondition{
				In: []any{"A", "B", "C"},
			},
			wantErr: false,
		},
		{
			name:  "not in clause",
			field: "status",
			condition: job.FilterCondition{
				NotIn: []any{"deleted", "archived"},
			},
			wantErr: false,
		},
		{
			name:  "multiple conditions combined",
			field: "age",
			condition: job.FilterCondition{
				GreaterOrEqual: []any{18},
				LessOrEqual:    []any{65},
			},
			wantErr: false,
		},
		{
			name:  "between for date field with string dates",
			field: "created_at",
			condition: job.FilterCondition{
				Between: []any{"2023-01-01", "2023-12-31"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseQuery := psql.Select("*").From("test_table")
			resultQuery := ds.applyAdvancedFilter(baseQuery, tt.field, tt.condition)

			sql, args, err := resultQuery.ToSql()
			if err != nil {
				t.Fatalf("applyAdvancedFilter() error generating SQL: %v", err)
			}

			// Verify the field name appears in the SQL
			if !contains(sql, tt.field) {
				t.Errorf("applyAdvancedFilter() SQL = %q, want to contain field %q", sql, tt.field)
			}

			// Verify we have arguments if we have filter conditions
			hasConditions := len(tt.condition.Equals) > 0 ||
				len(tt.condition.GreaterThan) > 0 ||
				len(tt.condition.GreaterOrEqual) > 0 ||
				len(tt.condition.LessThan) > 0 ||
				len(tt.condition.LessOrEqual) > 0 ||
				len(tt.condition.Between) > 0 ||
				len(tt.condition.In) > 0 ||
				len(tt.condition.NotIn) > 0

			if hasConditions && len(args) == 0 {
				t.Errorf("applyAdvancedFilter() with conditions should have args, got none")
			}
		})
	}
}

func TestBuildAdvancedFilters(t *testing.T) {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	ds := &ExternalDataSource{}

	schema := []TableSchema{
		{
			TableName: "users",
			Columns: []ColumnInformation{
				{Name: "id"},
				{Name: "name"},
				{Name: "age"},
				{Name: "status"},
			},
		},
	}

	tests := []struct {
		name    string
		table   string
		filter  map[string]job.FilterCondition
		wantErr bool
	}{
		{
			name:  "single field filter",
			table: "users",
			filter: map[string]job.FilterCondition{
				"status": {
					Equals: []any{"active"},
				},
			},
			wantErr: false,
		},
		{
			name:  "multiple field filters",
			table: "users",
			filter: map[string]job.FilterCondition{
				"status": {
					Equals: []any{"active"},
				},
				"age": {
					GreaterThan: []any{18},
				},
			},
			wantErr: false,
		},
		{
			name:    "empty filter",
			table:   "users",
			filter:  map[string]job.FilterCondition{},
			wantErr: false,
		},
		{
			name:  "filter with invalid field (should be ignored)",
			table: "users",
			filter: map[string]job.FilterCondition{
				"invalid_field": {
					Equals: []any{"value"},
				},
			},
			wantErr: false,
		},
		{
			name:  "filter with empty condition (should be ignored)",
			table: "users",
			filter: map[string]job.FilterCondition{
				"name": {},
			},
			wantErr: false,
		},
		{
			name:  "filter with invalid between (wrong count)",
			table: "users",
			filter: map[string]job.FilterCondition{
				"age": {
					Between: []any{18}, // Should have 2 values
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseQuery := psql.Select("*").From(tt.table)
			resultQuery, err := ds.buildAdvancedFilters(baseQuery, schema, tt.table, tt.filter)

			if (err != nil) != tt.wantErr {
				t.Errorf("buildAdvancedFilters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				sql, _, sqlErr := resultQuery.ToSql()
				if sqlErr != nil {
					t.Fatalf("buildAdvancedFilters() error generating SQL: %v", sqlErr)
				}

				// Basic sanity check that SQL was generated
				if len(sql) == 0 {
					t.Error("buildAdvancedFilters() generated empty SQL")
				}
			}
		})
	}
}

func TestBuildDynamicFilters_WithSchemaPrefix(t *testing.T) {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	schema := []TableSchema{
		{
			TableName: "users",
			Columns: []ColumnInformation{
				{Name: "id"},
				{Name: "name"},
			},
		},
	}

	tests := []struct {
		name   string
		table  string
		filter map[string][]any
	}{
		{
			name:  "table with schema prefix",
			table: "public.users",
			filter: map[string][]any{
				"name": {"John"},
			},
		},
		{
			name:  "table without schema prefix",
			table: "users",
			filter: map[string][]any{
				"name": {"Jane"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseQuery := psql.Select("*").From(tt.table)
			resultQuery := buildDynamicFilters(baseQuery, schema, tt.table, tt.filter)

			sql, _, err := resultQuery.ToSql()
			if err != nil {
				t.Fatalf("buildDynamicFilters() error generating SQL: %v", err)
			}

			// Should generate valid SQL
			if len(sql) == 0 {
				t.Error("buildDynamicFilters() generated empty SQL")
			}
		})
	}
}

func TestApplyAdvancedFilter_DateFieldHandling(t *testing.T) {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	ds := &ExternalDataSource{}

	tests := []struct {
		name      string
		field     string
		condition job.FilterCondition
	}{
		{
			name:  "date field between with date strings",
			field: "created_at",
			condition: job.FilterCondition{
				Between: []any{"2023-01-01", "2023-12-31"},
			},
		},
		{
			name:  "date field between adjusts end date",
			field: "updated_at",
			condition: job.FilterCondition{
				Between: []any{"2023-01-01", "2023-01-31"},
			},
		},
		{
			name:  "non-date field between",
			field: "age",
			condition: job.FilterCondition{
				Between: []any{18, 65},
			},
		},
		{
			name:  "date field with datetime strings",
			field: "created_at",
			condition: job.FilterCondition{
				Between: []any{"2023-01-01T00:00:00Z", "2023-12-31T23:59:59Z"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseQuery := psql.Select("*").From("test_table")
			resultQuery := ds.applyAdvancedFilter(baseQuery, tt.field, tt.condition)

			sql, args, err := resultQuery.ToSql()
			if err != nil {
				t.Fatalf("applyAdvancedFilter() error generating SQL: %v", err)
			}

			// Verify field appears in SQL
			if !contains(sql, tt.field) {
				t.Errorf("applyAdvancedFilter() SQL = %q, want to contain field %q", sql, tt.field)
			}

			// Verify we have arguments
			if len(args) == 0 {
				t.Error("applyAdvancedFilter() should have args for between condition")
			}
		})
	}
}

func TestApplyAdvancedFilter_AllOperators(t *testing.T) {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	ds := &ExternalDataSource{}

	// Test a condition with all operators
	condition := job.FilterCondition{
		Equals:         []any{"value1"},
		GreaterThan:    []any{10},
		GreaterOrEqual: []any{5},
		LessThan:       []any{100},
		LessOrEqual:    []any{200},
		Between:        []any{1, 50},
		In:             []any{"a", "b", "c"},
		NotIn:          []any{"x", "y"},
	}

	baseQuery := psql.Select("*").From("test_table")
	resultQuery := ds.applyAdvancedFilter(baseQuery, "test_field", condition)

	sql, args, err := resultQuery.ToSql()
	if err != nil {
		t.Fatalf("applyAdvancedFilter() error generating SQL: %v", err)
	}

	// Should contain the field name
	if !contains(sql, "test_field") {
		t.Errorf("SQL should contain field name, got: %s", sql)
	}

	// Should have many arguments (all the values from conditions)
	if len(args) < 5 {
		t.Errorf("Expected multiple args, got %d", len(args))
	}
}

func TestBuildAdvancedFilters_WithSchemaPrefix(t *testing.T) {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	ds := &ExternalDataSource{}

	schema := []TableSchema{
		{
			TableName: "orders",
			Columns: []ColumnInformation{
				{Name: "id"},
				{Name: "amount"},
				{Name: "status"},
			},
		},
	}

	tests := []struct {
		name    string
		table   string
		filter  map[string]job.FilterCondition
		wantErr bool
	}{
		{
			name:  "table with schema prefix public.orders",
			table: "public.orders",
			filter: map[string]job.FilterCondition{
				"status": {
					Equals: []any{"completed"},
				},
			},
			wantErr: false,
		},
		{
			name:  "table with schema prefix custom.orders",
			table: "custom.orders",
			filter: map[string]job.FilterCondition{
				"amount": {
					GreaterThan: []any{100},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseQuery := psql.Select("*").From(tt.table)
			resultQuery, err := ds.buildAdvancedFilters(baseQuery, schema, tt.table, tt.filter)

			if (err != nil) != tt.wantErr {
				t.Errorf("buildAdvancedFilters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				sql, _, sqlErr := resultQuery.ToSql()
				if sqlErr != nil {
					t.Fatalf("buildAdvancedFilters() error generating SQL: %v", sqlErr)
				}

				if len(sql) == 0 {
					t.Error("buildAdvancedFilters() generated empty SQL")
				}
			}
		})
	}
}

func TestTableSchema_Structure(t *testing.T) {
	// Test that TableSchema struct works correctly
	schema := TableSchema{
		TableName: "test_table",
		Columns: []ColumnInformation{
			{
				Name:         "id",
				DataType:     "integer",
				IsNullable:   false,
				IsPrimaryKey: true,
			},
			{
				Name:         "name",
				DataType:     "varchar",
				IsNullable:   true,
				IsPrimaryKey: false,
			},
		},
	}

	if schema.TableName != "test_table" {
		t.Errorf("Expected table name 'test_table', got %s", schema.TableName)
	}

	if len(schema.Columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(schema.Columns))
	}

	// Check first column
	if schema.Columns[0].Name != "id" {
		t.Errorf("Expected first column name 'id', got %s", schema.Columns[0].Name)
	}
	if !schema.Columns[0].IsPrimaryKey {
		t.Error("Expected first column to be primary key")
	}

	// Check second column
	if schema.Columns[1].Name != "name" {
		t.Errorf("Expected second column name 'name', got %s", schema.Columns[1].Name)
	}
	if schema.Columns[1].IsPrimaryKey {
		t.Error("Expected second column not to be primary key")
	}
}

func TestConnection_InitialState(t *testing.T) {
	mockLogger := &testutil.MockLogger{}

	tests := []struct {
		name string
		conn *Connection
	}{
		{
			name: "with connection string",
			conn: &Connection{
				ConnectionString:   "postgres://localhost/test",
				Logger:             mockLogger,
				MaxOpenConnections: 25,
				MaxIdleConnections: 10,
			},
		},
		{
			name: "with minimal configuration",
			conn: &Connection{
				ConnectionString:   "",
				Logger:             mockLogger,
				MaxOpenConnections: 5,
				MaxIdleConnections: 2,
			},
		},
		{
			name: "with database name",
			conn: &Connection{
				ConnectionString:   "postgres://localhost/mydb",
				DBName:             "mydb",
				Logger:             mockLogger,
				MaxOpenConnections: 10,
				MaxIdleConnections: 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify initial state
			if tt.conn.Connected {
				t.Error("Connection should not be connected initially")
			}
			if tt.conn.ConnectionDB != nil {
				t.Error("ConnectionDB should be nil initially")
			}
			if tt.conn.Logger == nil {
				t.Error("Logger should not be nil")
			}
		})
	}
}

func TestGetDB_MultipleCallsWithError(t *testing.T) {
	mockLogger := &testutil.MockLogger{}

	conn := &Connection{
		ConnectionString:   "invalid",
		Logger:             mockLogger,
		ConnectionDB:       nil,
		MaxOpenConnections: 10,
		MaxIdleConnections: 5,
	}

	// First call should attempt to connect and fail
	_, err1 := conn.GetDB()
	if err1 == nil {
		t.Error("GetDB() with invalid connection should return error")
	}

	// Second call should also attempt to connect (since ConnectionDB is still nil)
	_, err2 := conn.GetDB()
	if err2 == nil {
		t.Error("GetDB() second call should still return error")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func testMockLogger() *testutil.MockLogger {
	return &testutil.MockLogger{}
}

func TestParseJSONBField(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  any
	}{
		{
			name:  "nil value",
			value: nil,
			want:  nil,
		},
		{
			name:  "non-byte slice value - string",
			value: "plain string",
			want:  "plain string",
		},
		{
			name:  "non-byte slice value - int",
			value: 12345,
			want:  12345,
		},
		{
			name:  "valid JSON object as byte slice",
			value: []uint8(`{"key":"value","number":42}`),
			want:  map[string]any{"key": "value", "number": float64(42)},
		},
		{
			name:  "valid JSON array as byte slice",
			value: []uint8(`["item1","item2","item3"]`),
			want:  []any{"item1", "item2", "item3"},
		},
		{
			name:  "valid JSON string as byte slice",
			value: []uint8(`"simple string"`),
			want:  "simple string",
		},
		{
			name:  "invalid JSON as byte slice",
			value: []uint8(`{invalid json`),
			want:  []uint8(`{invalid json`),
		},
		{
			name:  "empty byte slice",
			value: []uint8{},
			want:  []uint8{},
		},
		{
			name:  "complex nested JSON object",
			value: []uint8(`{"user":{"name":"John","age":30},"active":true}`),
			want:  map[string]any{"user": map[string]any{"name": "John", "age": float64(30)}, "active": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We need a mock logger for this test
			mockLogger := testMockLogger()
			got := parseJSONBField(testutil.TestContext(), tt.value, mockLogger)

			// For map comparisons, we need to do a deep comparison
			if gotMap, ok := got.(map[string]any); ok {
				wantMap := tt.want.(map[string]any)
				if !mapsEqual(gotMap, wantMap) {
					t.Errorf("parseJSONBField() = %v, want %v", got, tt.want)
				}
			} else if gotSlice, ok := got.([]any); ok {
				wantSlice := tt.want.([]any)
				if !slicesEqual(gotSlice, wantSlice) {
					t.Errorf("parseJSONBField() = %v, want %v", got, tt.want)
				}
			} else if gotBytes, ok := got.([]uint8); ok {
				wantBytes := tt.want.([]uint8)
				if !bytesEqual(gotBytes, wantBytes) {
					t.Errorf("parseJSONBField() = %v, want %v", got, tt.want)
				}
			} else {
				if got != tt.want {
					t.Errorf("parseJSONBField() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// Helper functions for test comparisons
func mapsEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || !valuesEqual(v, bv) {
			return false
		}
	}
	return true
}

func slicesEqual(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !valuesEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func bytesEqual(a, b []uint8) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func valuesEqual(a, b any) bool {
	if aMap, ok := a.(map[string]any); ok {
		if bMap, ok := b.(map[string]any); ok {
			return mapsEqual(aMap, bMap)
		}
		return false
	}
	return a == b
}
