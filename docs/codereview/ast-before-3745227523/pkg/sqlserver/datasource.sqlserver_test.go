package sqlserver

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/fetcher/pkg/testutil"
	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	libLog "github.com/LerianStudio/lib-commons/v2/commons/log"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

// testContext creates a context with logger and tracer for testing
func testContext(t *testing.T) context.Context {
	t.Helper()
	logger := &libLog.GoLogger{Level: libLog.DebugLevel}
	values := &libCommons.CustomContextKeyValue{
		HeaderID: "test-request-id",
		Logger:   logger,
		Tracer:   otel.Tracer("test"),
	}
	return context.WithValue(context.Background(), libCommons.CustomContextKey, values)
}

// mockLogger is an alias for testutil.MockLogger for backward compatibility in this test file
type mockLogger = testutil.MockLogger

// TestSQLServerPlaceholder_ReplacePlaceholders tests the SQL Server placeholder replacement
func TestSQLServerPlaceholder_ReplacePlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single placeholder",
			input:    "SELECT * FROM users WHERE id = ?",
			expected: "SELECT * FROM users WHERE id = @p1",
		},
		{
			name:     "multiple placeholders",
			input:    "SELECT * FROM users WHERE id = ? AND name = ? AND age = ?",
			expected: "SELECT * FROM users WHERE id = @p1 AND name = @p2 AND age = @p3",
		},
		{
			name:     "no placeholders",
			input:    "SELECT * FROM users",
			expected: "SELECT * FROM users",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "placeholder at end",
			input:    "INSERT INTO users VALUES (?)",
			expected: "INSERT INTO users VALUES (@p1)",
		},
		{
			name:     "consecutive placeholders",
			input:    "INSERT INTO users VALUES (?, ?, ?)",
			expected: "INSERT INTO users VALUES (@p1, @p2, @p3)",
		},
		{
			name:     "placeholder with IN clause",
			input:    "SELECT * FROM users WHERE id IN (?, ?, ?)",
			expected: "SELECT * FROM users WHERE id IN (@p1, @p2, @p3)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			placeholder := sqlServerPlaceholder{}
			result, err := placeholder.ReplacePlaceholders(tt.input)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestNewDataSourceRepository tests the creation of a new ExternalDataSource
func TestNewDataSourceRepository(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func() (*sql.DB, sqlmock.Sqlmock)
		expectError bool
	}{
		{
			name: "successful connection",
			setupMock: func() (*sql.DB, sqlmock.Sqlmock) {
				db, mock, _ := sqlmock.New()
				mock.ExpectPing()
				return db, mock
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := tt.setupMock()
			defer db.Close()

			conn := &Connection{
				ConnectionString:   "test",
				Logger:             &mockLogger{},
				ConnectionDB:       db,
				Connected:          true,
				MaxOpenConnections: 10,
				MaxIdleConnections: 5,
			}

			ds, err := NewDataSourceRepository(conn)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, ds)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ds)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestNewDataSourceRepository_ConnectionFailure tests repository creation when connection fails
func TestNewDataSourceRepository_ConnectionFailure(t *testing.T) {
	mockLog := &mockLogger{}
	conn := &Connection{
		ConnectionString:   "invalid",
		Logger:             mockLog,
		ConnectionDB:       nil,
		Connected:          false,
		MaxOpenConnections: 10,
		MaxIdleConnections: 5,
	}

	ds, err := NewDataSourceRepository(conn)

	assert.Error(t, err)
	assert.Nil(t, ds)
}

// TestExternalDataSource_CloseConnection tests closing the database connection
func TestExternalDataSource_CloseConnection(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func() (*sql.DB, sqlmock.Sqlmock, *Connection)
		expectError bool
	}{
		{
			name: "successful close",
			setupMock: func() (*sql.DB, sqlmock.Sqlmock, *Connection) {
				db, mock, _ := sqlmock.New()
				mock.ExpectClose()
				conn := &Connection{
					ConnectionString:   "test",
					Logger:             &mockLogger{},
					ConnectionDB:       db,
					Connected:          true,
					MaxOpenConnections: 10,
					MaxIdleConnections: 5,
				}
				return db, mock, conn
			},
			expectError: false,
		},
		{
			name: "nil connection",
			setupMock: func() (*sql.DB, sqlmock.Sqlmock, *Connection) {
				conn := &Connection{
					ConnectionString:   "test",
					Logger:             &mockLogger{},
					ConnectionDB:       nil,
					Connected:          false,
					MaxOpenConnections: 10,
					MaxIdleConnections: 5,
				}
				return nil, nil, conn
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, conn := tt.setupMock()
			ds := &ExternalDataSource{connection: conn}

			err := ds.CloseConnection()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if mock != nil {
				assert.NoError(t, mock.ExpectationsWereMet())
			}

			// Verify connection state after close
			if conn.ConnectionDB == nil {
				assert.False(t, conn.Connected)
			}
		})
	}
}

// TestValidateFieldsInSchemaSQLServer tests field validation in schema
func TestValidateFieldsInSchemaSQLServer(t *testing.T) {
	tests := []struct {
		name           string
		expectedFields []string
		schema         TableSchema
		wantMissing    []string
		wantCount      int32
	}{
		{
			name:           "all fields exist",
			expectedFields: []string{"id", "name", "email"},
			schema: TableSchema{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id"},
					{Name: "name"},
					{Name: "email"},
				},
			},
			wantMissing: nil,
			wantCount:   3,
		},
		{
			name:           "some fields missing",
			expectedFields: []string{"id", "name", "phone"},
			schema: TableSchema{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id"},
					{Name: "name"},
				},
			},
			wantMissing: []string{"phone"},
			wantCount:   3,
		},
		{
			name:           "all fields missing",
			expectedFields: []string{"id", "name"},
			schema: TableSchema{
				TableName: "users",
				Columns:   []ColumnInformation{},
			},
			wantMissing: []string{"id", "name"},
			wantCount:   2,
		},
		{
			name:           "empty expected fields",
			expectedFields: []string{},
			schema: TableSchema{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id"},
				},
			},
			wantMissing: nil,
			wantCount:   0,
		},
		{
			name:           "case insensitive match - uppercase expected",
			expectedFields: []string{"ID", "Name", "EMAIL"},
			schema: TableSchema{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id"},
					{Name: "name"},
					{Name: "email"},
				},
			},
			wantMissing: nil,
			wantCount:   3,
		},
		{
			name:           "case insensitive match - mixed case schema",
			expectedFields: []string{"id", "name"},
			schema: TableSchema{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "ID"},
					{Name: "NAME"},
				},
			},
			wantMissing: nil,
			wantCount:   2,
		},
		{
			name:           "nil expected fields",
			expectedFields: nil,
			schema: TableSchema{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id"},
				},
			},
			wantMissing: nil,
			wantCount:   0,
		},
		{
			name:           "partial match with multiple missing",
			expectedFields: []string{"id", "name", "email", "phone", "address"},
			schema: TableSchema{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id"},
					{Name: "email"},
				},
			},
			wantMissing: []string{"name", "phone", "address"},
			wantCount:   5,
		},
		{
			name:           "empty schema columns",
			expectedFields: []string{"id"},
			schema: TableSchema{
				TableName: "empty_table",
				Columns:   nil,
			},
			wantMissing: []string{"id"},
			wantCount:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var count int32
			got := ValidateFieldsInSchemaSQLServer(tt.expectedFields, tt.schema, &count)

			assert.Equal(t, tt.wantCount, count)
			assert.Equal(t, tt.wantMissing, got)
		})
	}
}

// TestIsFilterConditionEmpty tests checking if a filter condition is empty
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
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestValidateFilterCondition tests filter condition validation
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
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestIsLikelyUUIDField tests UUID field name detection
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
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestIsValidUUIDFormat tests UUID format validation
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
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestIsDateField tests date field name detection
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
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestIsDateString tests date string detection
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
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestValidateUUIDFieldValues tests UUID value validation
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
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCreateRowMap tests row map creation from columns and values
func TestCreateRowMap(t *testing.T) {
	mockLog := &mockLogger{}

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
			got := createRowMap(tt.columns, tt.values, mockLog)

			assert.Equal(t, len(tt.want), len(got))
			for key, wantVal := range tt.want {
				gotVal, ok := got[key]
				assert.True(t, ok, "missing key %q", key)
				if gotMap, isMap := gotVal.(map[string]any); isMap {
					wantMap := wantVal.(map[string]any)
					assert.Equal(t, wantMap, gotMap)
				} else {
					assert.Equal(t, wantVal, gotVal)
				}
			}
		})
	}
}

// TestParseJSONField tests JSON field parsing
func TestParseJSONField(t *testing.T) {
	mockLog := &mockLogger{}

	tests := []struct {
		name     string
		value    any
		wantType string
	}{
		{
			name:     "nil value",
			value:    nil,
			wantType: "nil",
		},
		{
			name:     "non-byte slice value - string",
			value:    "plain string",
			wantType: "string",
		},
		{
			name:     "non-byte slice value - int",
			value:    12345,
			wantType: "int",
		},
		{
			name:     "valid JSON object as byte slice",
			value:    []uint8(`{"key":"value","number":42}`),
			wantType: "map",
		},
		{
			name:     "valid JSON array as byte slice",
			value:    []uint8(`["item1","item2","item3"]`),
			wantType: "array",
		},
		{
			name:     "valid JSON string as byte slice",
			value:    []uint8(`"simple string"`),
			wantType: "string",
		},
		{
			name:     "invalid JSON as byte slice",
			value:    []uint8(`{invalid json`),
			wantType: "bytes",
		},
		{
			name:     "empty byte slice",
			value:    []uint8{},
			wantType: "bytes",
		},
		{
			name:     "complex nested JSON object",
			value:    []uint8(`{"user":{"name":"John","age":30},"active":true}`),
			wantType: "map",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseJSONField(tt.value, mockLog)

			switch tt.wantType {
			case "nil":
				assert.Nil(t, got)
			case "string":
				_, ok := got.(string)
				assert.True(t, ok, "expected string type")
			case "int":
				_, ok := got.(int)
				assert.True(t, ok, "expected int type")
			case "map":
				_, ok := got.(map[string]any)
				assert.True(t, ok, "expected map type")
			case "array":
				_, ok := got.([]any)
				assert.True(t, ok, "expected array type")
			case "bytes":
				_, ok := got.([]uint8)
				assert.True(t, ok, "expected bytes type")
			}
		})
	}
}

// TestApplyFilter tests filter application
func TestApplyFilter(t *testing.T) {
	sqlServer := squirrel.StatementBuilder.PlaceholderFormat(sqlServerPlaceholderFormat)

	tests := []struct {
		name       string
		fieldName  string
		values     []any
		wantClause string
		wantArgs   int
	}{
		{
			name:       "single value",
			fieldName:  "status",
			values:     []any{"active"},
			wantClause: "status = ",
			wantArgs:   1,
		},
		{
			name:       "multiple values",
			fieldName:  "id",
			values:     []any{1, 2, 3},
			wantClause: "id IN ",
			wantArgs:   3,
		},
		{
			name:       "empty values returns unchanged",
			fieldName:  "name",
			values:     []any{},
			wantClause: "",
			wantArgs:   0,
		},
		{
			name:       "single numeric value",
			fieldName:  "age",
			values:     []any{25},
			wantClause: "age = ",
			wantArgs:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseQuery := sqlServer.Select("*").From("users")
			resultQuery := applyFilter(baseQuery, tt.fieldName, tt.values)

			sql, args, err := resultQuery.ToSql()
			require.NoError(t, err)

			if tt.wantClause == "" {
				baseSql, _, _ := baseQuery.ToSql()
				assert.Equal(t, baseSql, sql)
			} else {
				assert.Contains(t, sql, tt.wantClause)
				assert.Len(t, args, tt.wantArgs)
			}
		})
	}
}

// TestBuildDynamicFilters tests dynamic filter building
func TestBuildDynamicFilters(t *testing.T) {
	sqlServer := squirrel.StatementBuilder.PlaceholderFormat(sqlServerPlaceholderFormat)

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
			baseQuery := sqlServer.Select("*").From(tt.table)
			resultQuery := buildDynamicFilters(baseQuery, schema, tt.table, tt.filter)

			sql, _, err := resultQuery.ToSql()
			require.NoError(t, err)

			for _, field := range tt.wantFields {
				assert.Contains(t, sql, field)
			}
		})
	}
}

// TestBuildDynamicFilters_WithSchemaPrefix tests filter building with schema prefix
func TestBuildDynamicFilters_WithSchemaPrefix(t *testing.T) {
	sqlServer := squirrel.StatementBuilder.PlaceholderFormat(sqlServerPlaceholderFormat)

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
			table: "dbo.users",
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
			baseQuery := sqlServer.Select("*").From(tt.table)
			resultQuery := buildDynamicFilters(baseQuery, schema, tt.table, tt.filter)

			sql, _, err := resultQuery.ToSql()
			require.NoError(t, err)
			assert.NotEmpty(t, sql)
		})
	}
}

// TestApplyAdvancedFilter tests advanced filter application
func TestApplyAdvancedFilter(t *testing.T) {
	sqlServer := squirrel.StatementBuilder.PlaceholderFormat(sqlServerPlaceholderFormat)
	ds := &ExternalDataSource{}

	tests := []struct {
		name      string
		field     string
		condition job.FilterCondition
	}{
		{
			name:  "equals with single value",
			field: "status",
			condition: job.FilterCondition{
				Equals: []any{"active"},
			},
		},
		{
			name:  "equals with multiple values (IN clause)",
			field: "status",
			condition: job.FilterCondition{
				Equals: []any{"active", "pending", "completed"},
			},
		},
		{
			name:  "greater than",
			field: "age",
			condition: job.FilterCondition{
				GreaterThan: []any{18},
			},
		},
		{
			name:  "greater or equal",
			field: "age",
			condition: job.FilterCondition{
				GreaterOrEqual: []any{21},
			},
		},
		{
			name:  "less than",
			field: "age",
			condition: job.FilterCondition{
				LessThan: []any{65},
			},
		},
		{
			name:  "less or equal",
			field: "age",
			condition: job.FilterCondition{
				LessOrEqual: []any{100},
			},
		},
		{
			name:  "between two values",
			field: "age",
			condition: job.FilterCondition{
				Between: []any{18, 65},
			},
		},
		{
			name:  "in clause",
			field: "category",
			condition: job.FilterCondition{
				In: []any{"A", "B", "C"},
			},
		},
		{
			name:  "not in clause",
			field: "status",
			condition: job.FilterCondition{
				NotIn: []any{"deleted", "archived"},
			},
		},
		{
			name:  "multiple conditions combined",
			field: "age",
			condition: job.FilterCondition{
				GreaterOrEqual: []any{18},
				LessOrEqual:    []any{65},
			},
		},
		{
			name:  "between for date field with string dates",
			field: "created_at",
			condition: job.FilterCondition{
				Between: []any{"2023-01-01", "2023-12-31"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseQuery := sqlServer.Select("*").From("test_table")
			resultQuery := ds.applyAdvancedFilter(baseQuery, tt.field, tt.condition)

			sql, args, err := resultQuery.ToSql()
			require.NoError(t, err)
			assert.Contains(t, sql, tt.field)

			hasConditions := len(tt.condition.Equals) > 0 ||
				len(tt.condition.GreaterThan) > 0 ||
				len(tt.condition.GreaterOrEqual) > 0 ||
				len(tt.condition.LessThan) > 0 ||
				len(tt.condition.LessOrEqual) > 0 ||
				len(tt.condition.Between) > 0 ||
				len(tt.condition.In) > 0 ||
				len(tt.condition.NotIn) > 0

			if hasConditions {
				assert.NotEmpty(t, args)
			}
		})
	}
}

// TestApplyAdvancedFilter_DateFieldHandling tests date field handling in advanced filters
func TestApplyAdvancedFilter_DateFieldHandling(t *testing.T) {
	sqlServer := squirrel.StatementBuilder.PlaceholderFormat(sqlServerPlaceholderFormat)
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
			baseQuery := sqlServer.Select("*").From("test_table")
			resultQuery := ds.applyAdvancedFilter(baseQuery, tt.field, tt.condition)

			sql, args, err := resultQuery.ToSql()
			require.NoError(t, err)
			assert.Contains(t, sql, tt.field)
			assert.NotEmpty(t, args)
		})
	}
}

// TestApplyAdvancedFilter_AllOperators tests all operators combined
func TestApplyAdvancedFilter_AllOperators(t *testing.T) {
	sqlServer := squirrel.StatementBuilder.PlaceholderFormat(sqlServerPlaceholderFormat)
	ds := &ExternalDataSource{}

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

	baseQuery := sqlServer.Select("*").From("test_table")
	resultQuery := ds.applyAdvancedFilter(baseQuery, "test_field", condition)

	sql, args, err := resultQuery.ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "test_field")
	assert.GreaterOrEqual(t, len(args), 5)
}

// TestBuildAdvancedFilters tests advanced filter building
func TestBuildAdvancedFilters(t *testing.T) {
	sqlServer := squirrel.StatementBuilder.PlaceholderFormat(sqlServerPlaceholderFormat)
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
					Between: []any{18},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseQuery := sqlServer.Select("*").From(tt.table)
			resultQuery, err := ds.buildAdvancedFilters(baseQuery, schema, tt.table, tt.filter)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			sql, _, sqlErr := resultQuery.ToSql()
			require.NoError(t, sqlErr)
			assert.NotEmpty(t, sql)
		})
	}
}

// TestBuildAdvancedFilters_WithSchemaPrefix tests advanced filter building with schema prefix
func TestBuildAdvancedFilters_WithSchemaPrefix(t *testing.T) {
	sqlServer := squirrel.StatementBuilder.PlaceholderFormat(sqlServerPlaceholderFormat)
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
			name:  "table with schema prefix dbo.orders",
			table: "dbo.orders",
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
			baseQuery := sqlServer.Select("*").From(tt.table)
			resultQuery, err := ds.buildAdvancedFilters(baseQuery, schema, tt.table, tt.filter)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			sql, _, sqlErr := resultQuery.ToSql()
			require.NoError(t, sqlErr)
			assert.NotEmpty(t, sql)
		})
	}
}

// TestTableSchema_Structure tests TableSchema struct
func TestTableSchema_Structure(t *testing.T) {
	schema := TableSchema{
		TableName: "test_table",
		Columns: []ColumnInformation{
			{
				Name:         "id",
				DataType:     "int",
				IsNullable:   false,
				IsPrimaryKey: true,
			},
			{
				Name:         "name",
				DataType:     "nvarchar",
				IsNullable:   true,
				IsPrimaryKey: false,
			},
		},
	}

	assert.Equal(t, "test_table", schema.TableName)
	assert.Len(t, schema.Columns, 2)
	assert.Equal(t, "id", schema.Columns[0].Name)
	assert.True(t, schema.Columns[0].IsPrimaryKey)
	assert.Equal(t, "name", schema.Columns[1].Name)
	assert.False(t, schema.Columns[1].IsPrimaryKey)
}

// TestConnection_InitialState tests initial connection state
func TestConnection_InitialState(t *testing.T) {
	mockLog := &mockLogger{}

	tests := []struct {
		name string
		conn *Connection
	}{
		{
			name: "with connection string",
			conn: &Connection{
				ConnectionString:   "sqlserver://localhost/test",
				Logger:             mockLog,
				MaxOpenConnections: 25,
				MaxIdleConnections: 10,
			},
		},
		{
			name: "with minimal configuration",
			conn: &Connection{
				ConnectionString:   "",
				Logger:             mockLog,
				MaxOpenConnections: 5,
				MaxIdleConnections: 2,
			},
		},
		{
			name: "with database name",
			conn: &Connection{
				ConnectionString:   "sqlserver://localhost/mydb",
				DBName:             "mydb",
				Logger:             mockLog,
				MaxOpenConnections: 10,
				MaxIdleConnections: 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.False(t, tt.conn.Connected)
			assert.Nil(t, tt.conn.ConnectionDB)
			assert.NotNil(t, tt.conn.Logger)
		})
	}
}

// TestConnect_ErrorHandling tests connection error handling
func TestConnect_ErrorHandling(t *testing.T) {
	mockLog := &mockLogger{}

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
				Logger:             mockLog,
				MaxOpenConnections: 10,
				MaxIdleConnections: 5,
			}

			err := conn.Connect()
			if tt.wantErr {
				assert.Error(t, err)
				assert.False(t, conn.Connected)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGetDB_WithNilConnection tests GetDB with nil connection
func TestGetDB_WithNilConnection(t *testing.T) {
	mockLog := &mockLogger{}

	conn := &Connection{
		ConnectionString:   "invalid://connection",
		Logger:             mockLog,
		ConnectionDB:       nil,
		MaxOpenConnections: 10,
		MaxIdleConnections: 5,
	}

	db, err := conn.GetDB()
	assert.Error(t, err)
	assert.Nil(t, db)
}

// TestGetDB_MultipleCallsWithError tests multiple GetDB calls with error
func TestGetDB_MultipleCallsWithError(t *testing.T) {
	mockLog := &mockLogger{}

	conn := &Connection{
		ConnectionString:   "invalid",
		Logger:             mockLog,
		ConnectionDB:       nil,
		MaxOpenConnections: 10,
		MaxIdleConnections: 5,
	}

	_, err1 := conn.GetDB()
	assert.Error(t, err1)

	_, err2 := conn.GetDB()
	assert.Error(t, err2)
}

// TestExternalDataSource_Query tests Query method with sqlmock
func TestExternalDataSource_Query(t *testing.T) {
	ctx := testContext(t)

	schema := []TableSchema{
		{
			TableName: "users",
			Columns: []ColumnInformation{
				{Name: "id"},
				{Name: "name"},
				{Name: "email"},
			},
		},
	}

	tests := []struct {
		name        string
		table       string
		fields      []string
		filter      map[string][]any
		setupMock   func(sqlmock.Sqlmock)
		expectError bool
		expectRows  int
	}{
		{
			name:   "successful query with all fields",
			table:  "users",
			fields: []string{"id", "name", "email"},
			filter: map[string][]any{},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name", "email"}).
					AddRow(1, "John", "john@example.com").
					AddRow(2, "Jane", "jane@example.com")
				mock.ExpectQuery("SELECT").WillReturnRows(rows)
			},
			expectError: false,
			expectRows:  2,
		},
		{
			name:   "successful query with filter",
			table:  "users",
			fields: []string{"id", "name"},
			filter: map[string][]any{
				"name": {"John"},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name"}).
					AddRow(1, "John")
				mock.ExpectQuery("SELECT").WillReturnRows(rows)
			},
			expectError: false,
			expectRows:  1,
		},
		{
			name:   "query with database error",
			table:  "users",
			fields: []string{"id", "name"},
			filter: map[string][]any{},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnError(errors.New("database error"))
			},
			expectError: true,
			expectRows:  0,
		},
		{
			name:   "query with invalid table",
			table:  "invalid_table",
			fields: []string{"id"},
			filter: map[string][]any{},
			setupMock: func(mock sqlmock.Sqlmock) {
				// No mock needed as validation fails before query
			},
			expectError: true,
			expectRows:  0,
		},
		{
			name:   "query with invalid fields",
			table:  "users",
			fields: []string{"invalid_field"},
			filter: map[string][]any{},
			setupMock: func(mock sqlmock.Sqlmock) {
				// No mock needed as validation fails before query
			},
			expectError: true,
			expectRows:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			conn := &Connection{
				ConnectionString:   "test",
				Logger:             &mockLogger{},
				ConnectionDB:       db,
				Connected:          true,
				MaxOpenConnections: 10,
				MaxIdleConnections: 5,
			}

			ds := &ExternalDataSource{connection: conn}

			tt.setupMock(mock)

			results, err := ds.Query(ctx, schema, tt.table, tt.fields, tt.filter)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, results, tt.expectRows)
			}

			// Verify mock expectations were met only for non-validation failure cases
			if tt.table != "invalid_table" && tt.fields[0] != "invalid_field" {
				assert.NoError(t, mock.ExpectationsWereMet())
			}
		})
	}
}

// TestExternalDataSource_QueryWithAdvancedFilters tests QueryWithAdvancedFilters method
func TestExternalDataSource_QueryWithAdvancedFilters(t *testing.T) {
	ctx := testContext(t)

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
		name        string
		table       string
		fields      []string
		filter      map[string]job.FilterCondition
		setupMock   func(sqlmock.Sqlmock)
		expectError bool
		expectRows  int
	}{
		{
			name:   "successful query with equals filter",
			table:  "users",
			fields: []string{"id", "name"},
			filter: map[string]job.FilterCondition{
				"status": {Equals: []any{"active"}},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name"}).
					AddRow(1, "John")
				mock.ExpectQuery("SELECT").WillReturnRows(rows)
			},
			expectError: false,
			expectRows:  1,
		},
		{
			name:   "successful query with range filter",
			table:  "users",
			fields: []string{"id", "name", "age"},
			filter: map[string]job.FilterCondition{
				"age": {GreaterThan: []any{18}},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name", "age"}).
					AddRow(1, "John", 25)
				mock.ExpectQuery("SELECT").WillReturnRows(rows)
			},
			expectError: false,
			expectRows:  1,
		},
		{
			name:   "query with invalid between filter",
			table:  "users",
			fields: []string{"id"},
			filter: map[string]job.FilterCondition{
				"age": {Between: []any{18}}, // Invalid: needs 2 values
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				// No query expected due to validation failure
			},
			expectError: true,
			expectRows:  0,
		},
		{
			name:   "query with database error",
			table:  "users",
			fields: []string{"id", "name"},
			filter: map[string]job.FilterCondition{
				"status": {Equals: []any{"active"}},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnError(errors.New("database error"))
			},
			expectError: true,
			expectRows:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			conn := &Connection{
				ConnectionString:   "test",
				Logger:             &mockLogger{},
				ConnectionDB:       db,
				Connected:          true,
				MaxOpenConnections: 10,
				MaxIdleConnections: 5,
			}

			ds := &ExternalDataSource{connection: conn}

			tt.setupMock(mock)

			results, err := ds.QueryWithAdvancedFilters(ctx, schema, tt.table, tt.fields, tt.filter)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, results, tt.expectRows)
			}
		})
	}
}

// TestExternalDataSource_ValidateTableAndFields tests ValidateTableAndFields method
func TestExternalDataSource_ValidateTableAndFields(t *testing.T) {
	ctx := testContext(t)

	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	conn := &Connection{
		ConnectionString:   "test",
		Logger:             &mockLogger{},
		ConnectionDB:       db,
		Connected:          true,
		MaxOpenConnections: 10,
		MaxIdleConnections: 5,
	}

	ds := &ExternalDataSource{connection: conn}

	schema := []TableSchema{
		{
			TableName: "users",
			Columns: []ColumnInformation{
				{Name: "id"},
				{Name: "name"},
				{Name: "email"},
			},
		},
	}

	tests := []struct {
		name        string
		tableName   string
		fields      []string
		expectError bool
		expectCount int
	}{
		{
			name:        "valid table and fields",
			tableName:   "users",
			fields:      []string{"id", "name"},
			expectError: false,
			expectCount: 2,
		},
		{
			name:        "all fields with wildcard",
			tableName:   "users",
			fields:      []string{"*"},
			expectError: false,
			expectCount: 3,
		},
		{
			name:        "table with schema prefix",
			tableName:   "dbo.users",
			fields:      []string{"id"},
			expectError: false,
			expectCount: 1,
		},
		{
			name:        "invalid table",
			tableName:   "invalid_table",
			fields:      []string{"id"},
			expectError: true,
			expectCount: 0,
		},
		{
			name:        "invalid fields",
			tableName:   "users",
			fields:      []string{"invalid_field"},
			expectError: true,
			expectCount: 0,
		},
		{
			name:        "empty fields",
			tableName:   "users",
			fields:      []string{},
			expectError: true,
			expectCount: 0,
		},
		{
			name:        "mixed valid and invalid fields",
			tableName:   "users",
			fields:      []string{"id", "invalid_field"},
			expectError: true,
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ds.ValidateTableAndFields(ctx, tt.tableName, tt.fields, schema)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Len(t, result, tt.expectCount)
			}
		})
	}
}

// TestExternalDataSource_GetDatabaseSchema tests GetDatabaseSchema method
func TestExternalDataSource_GetDatabaseSchema(t *testing.T) {
	ctx := testContext(t)

	tests := []struct {
		name        string
		schemas     []string
		setupMock   func(sqlmock.Sqlmock)
		expectError bool
		expectCount int
	}{
		{
			name:    "successful schema retrieval with default schema",
			schemas: []string{},
			setupMock: func(mock sqlmock.Sqlmock) {
				// Tables query - now returns table_schema and table_name
				tableRows := sqlmock.NewRows([]string{"table_schema", "table_name"}).
					AddRow("dbo", "users").
					AddRow("dbo", "orders")
				mock.ExpectQuery("SELECT table_schema, table_name").WillReturnRows(tableRows)

				// Primary keys query
				pkRows := sqlmock.NewRows([]string{"table_schema", "table_name", "column_name"}).
					AddRow("dbo", "users", "id").
					AddRow("dbo", "orders", "id")
				mock.ExpectQuery("SELECT tc.table_schema").WillReturnRows(pkRows)

				// Column queries for each table
				userColRows := sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable"}).
					AddRow("id", "int", 0).
					AddRow("name", "nvarchar", 1)
				mock.ExpectQuery("SELECT column_name").WillReturnRows(userColRows)

				orderColRows := sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable"}).
					AddRow("id", "int", 0).
					AddRow("amount", "decimal", 0)
				mock.ExpectQuery("SELECT column_name").WillReturnRows(orderColRows)
			},
			expectError: false,
			expectCount: 2,
		},
		{
			name:    "successful schema retrieval with custom schemas",
			schemas: []string{"custom", "public"},
			setupMock: func(mock sqlmock.Sqlmock) {
				// Tables query - custom schema returns qualified table names
				tableRows := sqlmock.NewRows([]string{"table_schema", "table_name"}).
					AddRow("custom", "products")
				mock.ExpectQuery("SELECT table_schema, table_name").WillReturnRows(tableRows)

				pkRows := sqlmock.NewRows([]string{"table_schema", "table_name", "column_name"}).
					AddRow("custom", "products", "id")
				mock.ExpectQuery("SELECT tc.table_schema").WillReturnRows(pkRows)

				colRows := sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable"}).
					AddRow("id", "int", 0).
					AddRow("name", "nvarchar", 1)
				mock.ExpectQuery("SELECT column_name").WillReturnRows(colRows)
			},
			expectError: false,
			expectCount: 1,
		},
		{
			name:    "tables query error",
			schemas: []string{},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT table_schema, table_name").WillReturnError(errors.New("tables query error"))
			},
			expectError: true,
			expectCount: 0,
		},
		{
			name:    "primary keys query error",
			schemas: []string{},
			setupMock: func(mock sqlmock.Sqlmock) {
				tableRows := sqlmock.NewRows([]string{"table_schema", "table_name"}).
					AddRow("dbo", "users")
				mock.ExpectQuery("SELECT table_schema, table_name").WillReturnRows(tableRows)

				mock.ExpectQuery("SELECT tc.table_schema").WillReturnError(errors.New("pk query error"))
			},
			expectError: true,
			expectCount: 0,
		},
		{
			name:    "column query error",
			schemas: []string{},
			setupMock: func(mock sqlmock.Sqlmock) {
				tableRows := sqlmock.NewRows([]string{"table_schema", "table_name"}).
					AddRow("dbo", "users")
				mock.ExpectQuery("SELECT table_schema, table_name").WillReturnRows(tableRows)

				pkRows := sqlmock.NewRows([]string{"table_schema", "table_name", "column_name"}).
					AddRow("dbo", "users", "id")
				mock.ExpectQuery("SELECT tc.table_schema").WillReturnRows(pkRows)

				mock.ExpectQuery("SELECT column_name").WillReturnError(errors.New("column query error"))
			},
			expectError: true,
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			conn := &Connection{
				ConnectionString:   "test",
				Logger:             &mockLogger{},
				ConnectionDB:       db,
				Connected:          true,
				MaxOpenConnections: 10,
				MaxIdleConnections: 5,
			}

			ds := &ExternalDataSource{connection: conn}

			tt.setupMock(mock)

			result, err := ds.GetDatabaseSchema(ctx, tt.schemas)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, result, tt.expectCount)
			}
		})
	}
}

// TestQueryTables tests the queryTables internal method
func TestQueryTables(t *testing.T) {
	tests := []struct {
		name           string
		schemas        []string
		setupMock      func(sqlmock.Sqlmock)
		expectError    bool
		expectCount    int
		expectedTables []string
	}{
		{
			name:    "default schema",
			schemas: []string{},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"table_schema", "table_name"}).
					AddRow("dbo", "users").
					AddRow("dbo", "orders")
				mock.ExpectQuery("SELECT table_schema, table_name").WillReturnRows(rows)
			},
			expectError:    false,
			expectCount:    2,
			expectedTables: []string{"users", "orders"}, // default schema tables don't get prefix
		},
		{
			name:    "empty schemas (whitespace only)",
			schemas: []string{"  ", ""},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"table_schema", "table_name"}).
					AddRow("dbo", "users")
				mock.ExpectQuery("SELECT table_schema, table_name").WillReturnRows(rows)
			},
			expectError:    false,
			expectCount:    1,
			expectedTables: []string{"users"},
		},
		{
			name:    "multiple schemas",
			schemas: []string{"dbo", "custom"},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"table_schema", "table_name"}).
					AddRow("dbo", "users").
					AddRow("custom", "products").
					AddRow("dbo", "orders")
				mock.ExpectQuery("SELECT table_schema, table_name").WillReturnRows(rows)
			},
			expectError:    false,
			expectCount:    3,
			expectedTables: []string{"users", "custom.products", "orders"}, // non-default schema gets prefix
		},
		{
			name:    "query error",
			schemas: []string{},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT table_schema, table_name").WillReturnError(errors.New("query error"))
			},
			expectError: true,
			expectCount: 0,
		},
		{
			name:    "scan error",
			schemas: []string{},
			setupMock: func(mock sqlmock.Sqlmock) {
				// Create rows with wrong column count to cause scan error
				rows := sqlmock.NewRows([]string{"table_schema", "table_name", "extra"}).
					AddRow("dbo", "users", nil)
				mock.ExpectQuery("SELECT table_schema, table_name").WillReturnRows(rows)
			},
			expectError: true,
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			conn := &Connection{
				ConnectionString:   "test",
				Logger:             &mockLogger{},
				ConnectionDB:       db,
				Connected:          true,
				MaxOpenConnections: 10,
				MaxIdleConnections: 5,
			}

			ds := &ExternalDataSource{connection: conn}

			tt.setupMock(mock)

			result, err := ds.queryTables(context.Background(), tt.schemas)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, result, tt.expectCount)
				if tt.expectedTables != nil {
					assert.Equal(t, tt.expectedTables, result)
				}
			}
		})
	}
}

// TestQueryPrimaryKeys tests the queryPrimaryKeys internal method
func TestQueryPrimaryKeys(t *testing.T) {
	tests := []struct {
		name         string
		schemas      []string
		setupMock    func(sqlmock.Sqlmock)
		expectError  bool
		expectTables int
	}{
		{
			name:    "default schema with primary keys",
			schemas: []string{},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"table_schema", "table_name", "column_name"}).
					AddRow("dbo", "users", "id").
					AddRow("dbo", "orders", "id")
				mock.ExpectQuery("SELECT tc.table_schema").WillReturnRows(rows)
			},
			expectError:  false,
			expectTables: 2,
		},
		{
			name:    "multiple primary key columns",
			schemas: []string{},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"table_schema", "table_name", "column_name"}).
					AddRow("dbo", "order_items", "order_id").
					AddRow("dbo", "order_items", "product_id")
				mock.ExpectQuery("SELECT tc.table_schema").WillReturnRows(rows)
			},
			expectError:  false,
			expectTables: 1,
		},
		{
			name:    "query error",
			schemas: []string{},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT tc.table_schema").WillReturnError(errors.New("query error"))
			},
			expectError:  true,
			expectTables: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			conn := &Connection{
				ConnectionString:   "test",
				Logger:             &mockLogger{},
				ConnectionDB:       db,
				Connected:          true,
				MaxOpenConnections: 10,
				MaxIdleConnections: 5,
			}

			ds := &ExternalDataSource{connection: conn}

			tt.setupMock(mock)

			result, err := ds.queryPrimaryKeys(context.Background(), tt.schemas)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, result, tt.expectTables)
			}
		})
	}
}

// TestScanRows tests the scanRows function
func TestScanRows(t *testing.T) {
	mockLog := &mockLogger{}

	tests := []struct {
		name        string
		setupMock   func() (*sql.Rows, sqlmock.Sqlmock, func())
		expectError bool
		expectCount int
	}{
		{
			name: "successful scan with multiple rows",
			setupMock: func() (*sql.Rows, sqlmock.Sqlmock, func()) {
				db, mock, _ := sqlmock.New()
				rows := sqlmock.NewRows([]string{"id", "name", "email"}).
					AddRow(1, "John", "john@example.com").
					AddRow(2, "Jane", "jane@example.com")
				mock.ExpectQuery("SELECT").WillReturnRows(rows)
				sqlRows, _ := db.Query("SELECT")
				return sqlRows, mock, func() { db.Close() }
			},
			expectError: false,
			expectCount: 2,
		},
		{
			name: "empty result set",
			setupMock: func() (*sql.Rows, sqlmock.Sqlmock, func()) {
				db, mock, _ := sqlmock.New()
				rows := sqlmock.NewRows([]string{"id", "name"})
				mock.ExpectQuery("SELECT").WillReturnRows(rows)
				sqlRows, _ := db.Query("SELECT")
				return sqlRows, mock, func() { db.Close() }
			},
			expectError: false,
			expectCount: 0,
		},
		{
			name: "with JSON data",
			setupMock: func() (*sql.Rows, sqlmock.Sqlmock, func()) {
				db, mock, _ := sqlmock.New()
				rows := sqlmock.NewRows([]string{"id", "metadata"}).
					AddRow(1, []uint8(`{"key":"value"}`))
				mock.ExpectQuery("SELECT").WillReturnRows(rows)
				sqlRows, _ := db.Query("SELECT")
				return sqlRows, mock, func() { db.Close() }
			},
			expectError: false,
			expectCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, _, cleanup := tt.setupMock()
			defer cleanup()

			result, err := scanRows(rows, mockLog)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, result, tt.expectCount)
			}
		})
	}
}

// TestDefaultSchema tests the DefaultSchema constant
func TestDefaultSchema(t *testing.T) {
	assert.Equal(t, "dbo", DefaultSchema)
}

// TestSQLServerPlaceholderFormat tests the global placeholder format
func TestSQLServerPlaceholderFormat(t *testing.T) {
	query := "SELECT * FROM users WHERE id = ? AND name = ?"
	result, err := sqlServerPlaceholderFormat.ReplacePlaceholders(query)

	require.NoError(t, err)
	assert.Contains(t, result, "@p1")
	assert.Contains(t, result, "@p2")
	assert.NotContains(t, result, "?")
}

// TestColumnInformation_Structure tests ColumnInformation struct
func TestColumnInformation_Structure(t *testing.T) {
	col := ColumnInformation{
		Name:         "id",
		DataType:     "int",
		IsNullable:   false,
		IsPrimaryKey: true,
	}

	assert.Equal(t, "id", col.Name)
	assert.Equal(t, "int", col.DataType)
	assert.False(t, col.IsNullable)
	assert.True(t, col.IsPrimaryKey)
}

// TestScanColumns tests the scanColumns method
func TestScanColumns(t *testing.T) {
	mockLog := &mockLogger{}

	tests := []struct {
		name        string
		tableName   string
		primaryKeys map[string]map[string]bool
		setupMock   func() (*sql.Rows, func())
		expectError bool
		expectCount int
	}{
		{
			name:      "successful scan with primary key",
			tableName: "users",
			primaryKeys: map[string]map[string]bool{
				"users": {"id": true},
			},
			setupMock: func() (*sql.Rows, func()) {
				db, mock, _ := sqlmock.New()
				rows := sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable"}).
					AddRow("id", "int", 0).
					AddRow("name", "nvarchar", 1)
				mock.ExpectQuery("SELECT").WillReturnRows(rows)
				sqlRows, _ := db.Query("SELECT")
				return sqlRows, func() { db.Close() }
			},
			expectError: false,
			expectCount: 2,
		},
		{
			name:        "no primary keys for table",
			tableName:   "users",
			primaryKeys: map[string]map[string]bool{},
			setupMock: func() (*sql.Rows, func()) {
				db, mock, _ := sqlmock.New()
				rows := sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable"}).
					AddRow("id", "int", 0)
				mock.ExpectQuery("SELECT").WillReturnRows(rows)
				sqlRows, _ := db.Query("SELECT")
				return sqlRows, func() { db.Close() }
			},
			expectError: false,
			expectCount: 1,
		},
		{
			name:        "empty columns",
			tableName:   "empty_table",
			primaryKeys: map[string]map[string]bool{},
			setupMock: func() (*sql.Rows, func()) {
				db, mock, _ := sqlmock.New()
				rows := sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable"})
				mock.ExpectQuery("SELECT").WillReturnRows(rows)
				sqlRows, _ := db.Query("SELECT")
				return sqlRows, func() { db.Close() }
			},
			expectError: false,
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, _, _ := sqlmock.New()
			defer db.Close()

			conn := &Connection{
				ConnectionString:   "test",
				Logger:             mockLog,
				ConnectionDB:       db,
				Connected:          true,
				MaxOpenConnections: 10,
				MaxIdleConnections: 5,
			}

			ds := &ExternalDataSource{connection: conn}

			rows, cleanup := tt.setupMock()
			defer cleanup()

			result, err := ds.scanColumns(rows, tt.tableName, tt.primaryKeys, mockLog)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, result, tt.expectCount)
			}
		})
	}
}

// TestQueryPrimaryKeys_WithMultipleSchemas tests queryPrimaryKeys with multiple schemas
func TestQueryPrimaryKeys_WithMultipleSchemas(t *testing.T) {
	tests := []struct {
		name         string
		schemas      []string
		setupMock    func(sqlmock.Sqlmock)
		expectError  bool
		expectTables int
	}{
		{
			name:    "multiple schemas with primary keys",
			schemas: []string{"dbo", "custom"},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"table_schema", "table_name", "column_name"}).
					AddRow("dbo", "users", "id").
					AddRow("custom", "products", "id")
				mock.ExpectQuery("SELECT tc.table_schema").WillReturnRows(rows)
			},
			expectError:  false,
			expectTables: 2,
		},
		{
			name:    "schemas with whitespace (cleaned to default)",
			schemas: []string{"  ", ""},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"table_schema", "table_name", "column_name"}).
					AddRow("dbo", "users", "id")
				mock.ExpectQuery("SELECT tc.table_schema").WillReturnRows(rows)
			},
			expectError:  false,
			expectTables: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			conn := &Connection{
				ConnectionString:   "test",
				Logger:             &mockLogger{},
				ConnectionDB:       db,
				Connected:          true,
				MaxOpenConnections: 10,
				MaxIdleConnections: 5,
			}

			ds := &ExternalDataSource{connection: conn}

			tt.setupMock(mock)

			result, err := ds.queryPrimaryKeys(context.Background(), tt.schemas)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, result, tt.expectTables)
			}
		})
	}
}

// TestBuildTableSchema tests the buildTableSchema method
func TestBuildTableSchema(t *testing.T) {
	ctx := context.Background()
	mockLog := &mockLogger{}

	tests := []struct {
		name        string
		tableName   string
		schemas     []string
		primaryKeys map[string]map[string]bool
		setupMock   func(sqlmock.Sqlmock)
		expectError bool
		expectCols  int
	}{
		{
			name:      "table with columns and primary key",
			tableName: "users",
			schemas:   []string{},
			primaryKeys: map[string]map[string]bool{
				"users": {"id": true},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable"}).
					AddRow("id", "int", 0).
					AddRow("name", "nvarchar", 1).
					AddRow("email", "nvarchar", 1)
				mock.ExpectQuery("SELECT column_name").WillReturnRows(rows)
			},
			expectError: false,
			expectCols:  3,
		},
		{
			name:        "table with multiple schemas",
			tableName:   "users",
			schemas:     []string{"dbo", "custom"},
			primaryKeys: map[string]map[string]bool{},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable"}).
					AddRow("id", "int", 0)
				mock.ExpectQuery("SELECT column_name").WillReturnRows(rows)
			},
			expectError: false,
			expectCols:  1,
		},
		{
			name:        "table with whitespace schemas (cleaned to default)",
			tableName:   "users",
			schemas:     []string{"  ", ""},
			primaryKeys: map[string]map[string]bool{},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable"}).
					AddRow("id", "int", 0)
				mock.ExpectQuery("SELECT column_name").WillReturnRows(rows)
			},
			expectError: false,
			expectCols:  1,
		},
		{
			name:        "query error",
			tableName:   "users",
			schemas:     []string{},
			primaryKeys: map[string]map[string]bool{},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT column_name").WillReturnError(errors.New("query error"))
			},
			expectError: true,
			expectCols:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			conn := &Connection{
				ConnectionString:   "test",
				Logger:             mockLog,
				ConnectionDB:       db,
				Connected:          true,
				MaxOpenConnections: 10,
				MaxIdleConnections: 5,
			}

			ds := &ExternalDataSource{connection: conn}

			tt.setupMock(mock)

			result, err := ds.buildTableSchema(ctx, tt.tableName, tt.primaryKeys, mockLog, tt.schemas)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.tableName, result.TableName)
				assert.Len(t, result.Columns, tt.expectCols)
			}
		})
	}
}

// TestExternalDataSource_CloseConnection_WithError tests CloseConnection error case
func TestExternalDataSource_CloseConnection_WithError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	// Don't defer close here as we're testing the close error scenario

	mock.ExpectClose().WillReturnError(errors.New("close error"))

	conn := &Connection{
		ConnectionString:   "test",
		Logger:             &mockLogger{},
		ConnectionDB:       db,
		Connected:          true,
		MaxOpenConnections: 10,
		MaxIdleConnections: 5,
	}

	ds := &ExternalDataSource{connection: conn}

	err = ds.CloseConnection()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "close error")
}

// TestGetDB_WithExistingConnection tests GetDB when connection already exists
func TestGetDB_WithExistingConnection(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	conn := &Connection{
		ConnectionString:   "test",
		Logger:             &mockLogger{},
		ConnectionDB:       db,
		Connected:          true,
		MaxOpenConnections: 10,
		MaxIdleConnections: 5,
	}

	result, err := conn.GetDB()
	require.NoError(t, err)
	assert.Equal(t, db, result)
}

// TestQueryPrimaryKeys_RowsError tests queryPrimaryKeys with rows iteration error
func TestQueryPrimaryKeys_RowsError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	conn := &Connection{
		ConnectionString:   "test",
		Logger:             &mockLogger{},
		ConnectionDB:       db,
		Connected:          true,
		MaxOpenConnections: 10,
		MaxIdleConnections: 5,
	}

	ds := &ExternalDataSource{connection: conn}

	// Create rows that will cause a scan error (wrong number of columns)
	rows := sqlmock.NewRows([]string{"table_schema", "table_name"}).
		AddRow("dbo", "users")
	mock.ExpectQuery("SELECT tc.table_schema").WillReturnRows(rows)

	result, err := ds.queryPrimaryKeys(context.Background(), []string{})
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestQueryTables_RowsIterationError tests queryTables with rows.Err() scenario
func TestQueryTables_RowsIterationError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	conn := &Connection{
		ConnectionString:   "test",
		Logger:             &mockLogger{},
		ConnectionDB:       db,
		Connected:          true,
		MaxOpenConnections: 10,
		MaxIdleConnections: 5,
	}

	ds := &ExternalDataSource{connection: conn}

	// Create rows with an error during iteration
	rows := sqlmock.NewRows([]string{"table_schema", "table_name"}).
		AddRow("dbo", "users").
		RowError(0, errors.New("row iteration error"))
	mock.ExpectQuery("SELECT table_schema, table_name").WillReturnRows(rows)

	result, err := ds.queryTables(context.Background(), []string{})
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestSQLInjectionPrevention verifies that malicious input cannot be used for SQL injection.
// SQL Server uses parameterized queries with @p1, @p2 placeholders via squirrel query builder.
func TestSQLInjectionPrevention(t *testing.T) {
	ctx := testContext(t)

	// Malicious inputs that could be used for SQL injection attacks
	maliciousInputs := []string{
		"users; DROP TABLE users;--",
		"users' OR '1'='1",
		`users"; DROP TABLE users;--`,
		"users UNION SELECT * FROM passwords--",
		"users; DELETE FROM users WHERE 1=1;--",
	}

	t.Run("malicious table names are rejected by validation", func(t *testing.T) {
		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       &testutil.MockLogger{},
		}

		ds := &ExternalDataSource{connection: conn}

		// Schema with only valid table names
		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "int"},
					{Name: "name", DataType: "nvarchar"},
				},
			},
		}

		for _, maliciousTable := range maliciousInputs {
			t.Run(maliciousTable, func(t *testing.T) {
				// Attempting to query with a malicious table name should fail validation
				_, err := ds.Query(ctx, schema, maliciousTable, []string{"id"}, nil)

				// Should fail because the malicious table doesn't exist in schema
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "does not exist")
			})
		}
	})

	t.Run("malicious field names are rejected by validation", func(t *testing.T) {
		db, _, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       &testutil.MockLogger{},
		}

		ds := &ExternalDataSource{connection: conn}

		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "int"},
					{Name: "name", DataType: "nvarchar"},
				},
			},
		}

		for _, maliciousField := range maliciousInputs {
			t.Run(maliciousField, func(t *testing.T) {
				// Attempting to query with a malicious field name should fail validation
				_, err := ds.Query(ctx, schema, "users", []string{maliciousField}, nil)

				// Should fail because the malicious field doesn't exist in schema
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid fields")
			})
		}
	})

	t.Run("filter values are parameterized with SQL Server placeholders", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       &testutil.MockLogger{},
		}

		ds := &ExternalDataSource{connection: conn}

		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "int"},
					{Name: "name", DataType: "nvarchar"},
				},
			},
		}

		// Even with malicious filter values, they should be passed as parameters
		maliciousValue := "'; DROP TABLE users;--"

		// Mock expects the query with SQL Server's @p1 parameterized placeholder
		rows := sqlmock.NewRows([]string{"id", "name"})
		mock.ExpectQuery(`SELECT id, name FROM users WHERE name = @p1`).
			WithArgs(maliciousValue).
			WillReturnRows(rows)

		filter := map[string][]any{
			"name": {maliciousValue},
		}

		// Query should succeed - the malicious value is safely parameterized
		result, err := ds.Query(ctx, schema, "users", []string{"id", "name"}, filter)

		require.NoError(t, err)
		assert.Empty(t, result) // No rows returned, but no error
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("advanced filter values are parameterized", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       &testutil.MockLogger{},
		}

		ds := &ExternalDataSource{connection: conn}

		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "int"},
					{Name: "name", DataType: "nvarchar"},
				},
			},
		}

		maliciousValue := "admin' OR '1'='1"

		// Mock expects parameterized query with SQL Server @p1 placeholder
		rows := sqlmock.NewRows([]string{"id", "name"})
		mock.ExpectQuery(`SELECT id, name FROM users WHERE name = @p1`).
			WithArgs(maliciousValue).
			WillReturnRows(rows)

		filter := map[string]job.FilterCondition{
			"name": {
				Equals: []any{maliciousValue},
			},
		}

		result, err := ds.QueryWithAdvancedFilters(ctx, schema, "users", []string{"id", "name"}, filter)

		require.NoError(t, err)
		assert.Empty(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
