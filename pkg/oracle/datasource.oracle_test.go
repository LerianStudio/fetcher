package oracle

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

// testContext creates a context with tracking information for testing.
func testContext() context.Context {
	logger := &mockLogger{}
	values := &libCommons.CustomContextKeyValue{
		HeaderID: "test-request-id",
		Logger:   logger,
		Tracer:   otel.Tracer("test"),
	}

	return context.WithValue(context.Background(), libCommons.CustomContextKey, values)
}

// testLogger returns a mock logger for testing.
func testLogger() *mockLogger {
	return &mockLogger{}
}

// setupMockDB creates a mock database connection for testing.
func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	return db, mock
}

func TestNewDataSourceRepository(t *testing.T) {
	t.Run("success with valid connection", func(t *testing.T) {
		db, _ := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds, err := NewDataSourceRepository(conn)

		require.NoError(t, err)
		assert.NotNil(t, ds)
	})

	t.Run("failure with nil connection db", func(t *testing.T) {
		conn := &Connection{
			ConnectionDB:     nil,
			Connected:        false,
			Logger:           testLogger(),
			ConnectionString: "invalid://connection",
		}

		ds, err := NewDataSourceRepository(conn)

		assert.Error(t, err)
		assert.Nil(t, ds)
		assert.Contains(t, err.Error(), "failed to establish Oracle connection")
	})
}

func TestExternalDataSource_CloseConnection(t *testing.T) {
	t.Run("closes connection successfully", func(t *testing.T) {
		db, mock := setupMockDB(t)

		// Expect Close to be called
		mock.ExpectClose()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		err := ds.CloseConnection()

		assert.NoError(t, err)
		assert.False(t, conn.Connected)
		assert.Nil(t, conn.ConnectionDB)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("handles nil connection gracefully", func(t *testing.T) {
		conn := &Connection{
			ConnectionDB: nil,
			Connected:    false,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		err := ds.CloseConnection()

		assert.NoError(t, err)
	})
}

func TestExternalDataSource_Query(t *testing.T) {
	ctx := testContext()

	t.Run("successful query with results", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		rows := sqlmock.NewRows([]string{"ID", "NAME"}).
			AddRow(1, "Alice").
			AddRow(2, "Bob")

		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "USERS",
				Columns: []ColumnInformation{
					{Name: "ID", DataType: "NUMBER"},
					{Name: "NAME", DataType: "VARCHAR2"},
				},
			},
		}

		result, err := ds.Query(ctx, schema, "USERS", []string{"ID", "NAME"}, nil)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query with filter", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		rows := sqlmock.NewRows([]string{"ID", "NAME"}).
			AddRow(1, "Alice")

		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "USERS",
				Columns: []ColumnInformation{
					{Name: "ID", DataType: "NUMBER"},
					{Name: "NAME", DataType: "VARCHAR2"},
				},
			},
		}

		filter := map[string][]any{
			"NAME": {"Alice"},
		}

		result, err := ds.Query(ctx, schema, "USERS", []string{"ID", "NAME"}, filter)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query with wildcard fields", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		rows := sqlmock.NewRows([]string{"ID", "NAME", "EMAIL"}).
			AddRow(1, "Alice", "alice@example.com")

		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "USERS",
				Columns: []ColumnInformation{
					{Name: "ID", DataType: "NUMBER"},
					{Name: "NAME", DataType: "VARCHAR2"},
					{Name: "EMAIL", DataType: "VARCHAR2"},
				},
			},
		}

		result, err := ds.Query(ctx, schema, "USERS", []string{"*"}, nil)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query with invalid table returns error", func(t *testing.T) {
		db, _ := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		schema := []TableSchema{
			{
				TableName: "USERS",
				Columns: []ColumnInformation{
					{Name: "ID", DataType: "NUMBER"},
				},
			},
		}

		_, err := ds.Query(ctx, schema, "NONEXISTENT", []string{"ID"}, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("query with invalid fields returns error", func(t *testing.T) {
		db, _ := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		schema := []TableSchema{
			{
				TableName: "USERS",
				Columns: []ColumnInformation{
					{Name: "ID", DataType: "NUMBER"},
				},
			},
		}

		_, err := ds.Query(ctx, schema, "USERS", []string{"INVALID_FIELD"}, nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid fields")
	})
}

func TestExternalDataSource_QueryWithAdvancedFilters(t *testing.T) {
	ctx := testContext()

	t.Run("query with equals filter", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		rows := sqlmock.NewRows([]string{"ID", "STATUS"}).
			AddRow(1, "active")

		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "ORDERS",
				Columns: []ColumnInformation{
					{Name: "ID", DataType: "NUMBER"},
					{Name: "STATUS", DataType: "VARCHAR2"},
				},
			},
		}

		filter := map[string]job.FilterCondition{
			"STATUS": {
				Equals: []any{"active"},
			},
		}

		result, err := ds.QueryWithAdvancedFilters(ctx, schema, "ORDERS", []string{"ID", "STATUS"}, filter)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query with between filter", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		rows := sqlmock.NewRows([]string{"ID", "AMOUNT"}).
			AddRow(1, 150.00)

		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "ORDERS",
				Columns: []ColumnInformation{
					{Name: "ID", DataType: "NUMBER"},
					{Name: "AMOUNT", DataType: "NUMBER"},
				},
			},
		}

		filter := map[string]job.FilterCondition{
			"AMOUNT": {
				Between: []any{100, 200},
			},
		}

		result, err := ds.QueryWithAdvancedFilters(ctx, schema, "ORDERS", []string{"ID", "AMOUNT"}, filter)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query with invalid between filter returns error", func(t *testing.T) {
		db, _ := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		schema := []TableSchema{
			{
				TableName: "ORDERS",
				Columns: []ColumnInformation{
					{Name: "ID", DataType: "NUMBER"},
					{Name: "AMOUNT", DataType: "NUMBER"},
				},
			},
		}

		filter := map[string]job.FilterCondition{
			"AMOUNT": {
				Between: []any{100}, // Invalid: needs exactly 2 values
			},
		}

		_, err := ds.QueryWithAdvancedFilters(ctx, schema, "ORDERS", []string{"ID", "AMOUNT"}, filter)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "between")
	})
}

func TestExternalDataSource_ValidateTableAndFields(t *testing.T) {
	ctx := testContext()

	tests := []struct {
		name        string
		schema      []TableSchema
		table       string
		fields      []string
		expectError bool
		errContains string
	}{
		{
			name: "valid table and fields",
			schema: []TableSchema{
				{
					TableName: "USERS",
					Columns: []ColumnInformation{
						{Name: "ID"},
						{Name: "NAME"},
					},
				},
			},
			table:       "USERS",
			fields:      []string{"ID", "NAME"},
			expectError: false,
		},
		{
			name: "valid table with case insensitive match",
			schema: []TableSchema{
				{
					TableName: "USERS",
					Columns: []ColumnInformation{
						{Name: "ID"},
						{Name: "NAME"},
					},
				},
			},
			table:       "users",
			fields:      []string{"id", "name"},
			expectError: false,
		},
		{
			name: "invalid table",
			schema: []TableSchema{
				{TableName: "USERS"},
			},
			table:       "NONEXISTENT",
			fields:      []string{"ID"},
			expectError: true,
			errContains: "does not exist",
		},
		{
			name: "invalid fields",
			schema: []TableSchema{
				{
					TableName: "USERS",
					Columns: []ColumnInformation{
						{Name: "ID"},
						{Name: "NAME"},
					},
				},
			},
			table:       "USERS",
			fields:      []string{"ID", "INVALID"},
			expectError: true,
			errContains: "invalid fields",
		},
		{
			name: "wildcard returns all fields",
			schema: []TableSchema{
				{
					TableName: "USERS",
					Columns: []ColumnInformation{
						{Name: "ID"},
						{Name: "NAME"},
						{Name: "EMAIL"},
					},
				},
			},
			table:       "USERS",
			fields:      []string{"*"},
			expectError: false,
		},
		{
			name: "empty fields returns error",
			schema: []TableSchema{
				{
					TableName: "USERS",
					Columns: []ColumnInformation{
						{Name: "ID"},
					},
				},
			},
			table:       "USERS",
			fields:      []string{},
			expectError: true,
			errContains: "no valid fields",
		},
		{
			name: "table with schema prefix",
			schema: []TableSchema{
				{
					TableName: "ORDERS",
					Columns: []ColumnInformation{
						{Name: "ID"},
						{Name: "TOTAL"},
					},
				},
			},
			table:       "SALES.ORDERS",
			fields:      []string{"ID", "TOTAL"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, _ := setupMockDB(t)
			defer db.Close()

			conn := &Connection{
				ConnectionDB: db,
				Connected:    true,
				Logger:       testLogger(),
			}

			ds := &ExternalDataSource{connection: conn}

			result, err := ds.ValidateTableAndFields(ctx, tt.table, tt.fields, tt.schema)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestExternalDataSource_GetDatabaseSchema(t *testing.T) {
	ctx := testContext()

	t.Run("retrieves schema without filter", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		// Mock table query
		tableRows := sqlmock.NewRows([]string{"table_name"}).
			AddRow("USERS").
			AddRow("ORDERS")
		mock.ExpectQuery("SELECT table_name").WillReturnRows(tableRows)

		// Mock primary key query
		pkRows := sqlmock.NewRows([]string{"table_name", "column_name"}).
			AddRow("USERS", "ID").
			AddRow("ORDERS", "ORDER_ID")
		mock.ExpectQuery("SELECT table_name, column_name").WillReturnRows(pkRows)

		// Mock column query for USERS
		usersColRows := sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable"}).
			AddRow("ID", "NUMBER", 0).
			AddRow("NAME", "VARCHAR2", 1)
		mock.ExpectQuery("SELECT column_name").WillReturnRows(usersColRows)

		// Mock column query for ORDERS
		ordersColRows := sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable"}).
			AddRow("ORDER_ID", "NUMBER", 0).
			AddRow("TOTAL", "NUMBER", 1)
		mock.ExpectQuery("SELECT column_name").WillReturnRows(ordersColRows)

		result, err := ds.GetDatabaseSchema(ctx, nil)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("retrieves schema with schema filter", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		// Mock table query with schema filter
		tableRows := sqlmock.NewRows([]string{"table_name"}).
			AddRow("USERS")
		mock.ExpectQuery("SELECT table_name").WillReturnRows(tableRows)

		// Mock primary key query
		pkRows := sqlmock.NewRows([]string{"table_name", "column_name"}).
			AddRow("USERS", "ID")
		mock.ExpectQuery("SELECT").WillReturnRows(pkRows)

		// Mock column query
		colRows := sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable"}).
			AddRow("ID", "NUMBER", 0)
		mock.ExpectQuery("SELECT column_name").WillReturnRows(colRows)

		result, err := ds.GetDatabaseSchema(ctx, []string{"MYSCHEMA"})

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestValidateFieldsInSchemaOracle(t *testing.T) {
	tests := []struct {
		name           string
		expectedFields []string
		schema         TableSchema
		wantMissing    []string
		wantCount      int32
	}{
		{
			name:           "all fields exist",
			expectedFields: []string{"ID", "NAME", "EMAIL"},
			schema: TableSchema{
				TableName: "USERS",
				Columns: []ColumnInformation{
					{Name: "ID"},
					{Name: "NAME"},
					{Name: "EMAIL"},
				},
			},
			wantMissing: nil,
			wantCount:   3,
		},
		{
			name:           "some fields missing",
			expectedFields: []string{"ID", "NAME", "PHONE"},
			schema: TableSchema{
				TableName: "USERS",
				Columns: []ColumnInformation{
					{Name: "ID"},
					{Name: "NAME"},
				},
			},
			wantMissing: []string{"PHONE"},
			wantCount:   3,
		},
		{
			name:           "all fields missing",
			expectedFields: []string{"ID", "NAME"},
			schema: TableSchema{
				TableName: "USERS",
				Columns:   []ColumnInformation{},
			},
			wantMissing: []string{"ID", "NAME"},
			wantCount:   2,
		},
		{
			name:           "empty expected fields",
			expectedFields: []string{},
			schema: TableSchema{
				TableName: "USERS",
				Columns: []ColumnInformation{
					{Name: "ID"},
				},
			},
			wantMissing: nil,
			wantCount:   0,
		},
		{
			name:           "case insensitive match",
			expectedFields: []string{"id", "name", "email"},
			schema: TableSchema{
				TableName: "USERS",
				Columns: []ColumnInformation{
					{Name: "ID"},
					{Name: "NAME"},
					{Name: "EMAIL"},
				},
			},
			wantMissing: nil,
			wantCount:   3,
		},
		{
			name:           "nil expected fields",
			expectedFields: nil,
			schema: TableSchema{
				TableName: "USERS",
				Columns: []ColumnInformation{
					{Name: "ID"},
				},
			},
			wantMissing: nil,
			wantCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var count int32
			got := ValidateFieldsInSchemaOracle(tt.expectedFields, tt.schema, &count)

			assert.Equal(t, tt.wantCount, count)
			assert.Equal(t, tt.wantMissing, got)
		})
	}
}

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
			fieldName: "AGE",
			condition: job.FilterCondition{
				Between: []any{10, 20},
			},
			wantErr: false,
		},
		{
			name:      "invalid between with 1 value",
			fieldName: "AGE",
			condition: job.FilterCondition{
				Between: []any{10},
			},
			wantErr: true,
		},
		{
			name:      "invalid between with 3 values",
			fieldName: "AGE",
			condition: job.FilterCondition{
				Between: []any{10, 20, 30},
			},
			wantErr: true,
		},
		{
			name:      "valid greater than with 1 value",
			fieldName: "AGE",
			condition: job.FilterCondition{
				GreaterThan: []any{10},
			},
			wantErr: false,
		},
		{
			name:      "invalid greater than with 2 values",
			fieldName: "AGE",
			condition: job.FilterCondition{
				GreaterThan: []any{10, 20},
			},
			wantErr: true,
		},
		{
			name:      "valid UUID field with UUID value",
			fieldName: "USER_ID",
			condition: job.FilterCondition{
				Equals: []any{validUUID},
			},
			wantErr: false,
		},
		{
			name:      "invalid UUID field with non-UUID value",
			fieldName: "USER_ID",
			condition: job.FilterCondition{
				Equals: []any{"not-a-uuid"},
			},
			wantErr: true,
		},
		{
			name:      "valid equals with multiple values",
			fieldName: "STATUS",
			condition: job.FilterCondition{
				Equals: []any{"active", "pending", "completed"},
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

func TestIsLikelyUUIDField(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		want      bool
	}{
		{
			name:      "field with id suffix",
			fieldName: "USER_ID",
			want:      true,
		},
		{
			name:      "field with uuid",
			fieldName: "UUID",
			want:      true,
		},
		{
			name:      "field with template_id",
			fieldName: "TEMPLATE_ID",
			want:      true,
		},
		{
			name:      "field with organization_id",
			fieldName: "ORGANIZATION_ID",
			want:      true,
		},
		{
			name:      "plain id field",
			fieldName: "ID",
			want:      true,
		},
		{
			name:      "non-UUID field name",
			fieldName: "NAME",
			want:      false,
		},
		{
			name:      "non-UUID field email",
			fieldName: "EMAIL",
			want:      false,
		},
		{
			name:      "non-UUID field status",
			fieldName: "STATUS",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidUUIDFormat(tt.s)
			assert.Equal(t, tt.want, got)
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
			fieldName: "CREATED_AT",
			want:      true,
		},
		{
			name:      "field with updated_at",
			fieldName: "UPDATED_AT",
			want:      true,
		},
		{
			name:      "field with deleted_at",
			fieldName: "DELETED_AT",
			want:      true,
		},
		{
			name:      "field with date",
			fieldName: "BIRTH_DATE",
			want:      true,
		},
		{
			name:      "field with time",
			fieldName: "START_TIME",
			want:      true,
		},
		{
			name:      "non-date field name",
			fieldName: "NAME",
			want:      false,
		},
		{
			name:      "non-date field id",
			fieldName: "ID",
			want:      false,
		},
		{
			name:      "non-date field status",
			fieldName: "STATUS",
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
			name:  "not a string value - int",
			value: 12345,
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDateString(tt.value)
			assert.Equal(t, tt.want, got)
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
			fieldName: "USER_ID",
			condition: job.FilterCondition{
				Equals: []any{validUUID1},
			},
			wantErr: false,
		},
		{
			name:      "invalid UUID in Equals",
			fieldName: "USER_ID",
			condition: job.FilterCondition{
				Equals: []any{"not-a-uuid"},
			},
			wantErr: true,
		},
		{
			name:      "valid UUIDs in Between",
			fieldName: "USER_ID",
			condition: job.FilterCondition{
				Between: []any{validUUID1, validUUID2},
			},
			wantErr: false,
		},
		{
			name:      "invalid UUID in Between",
			fieldName: "USER_ID",
			condition: job.FilterCondition{
				Between: []any{validUUID1, "not-uuid"},
			},
			wantErr: true,
		},
		{
			name:      "valid UUIDs in In",
			fieldName: "USER_ID",
			condition: job.FilterCondition{
				In: []any{validUUID1, validUUID2},
			},
			wantErr: false,
		},
		{
			name:      "non-string value should pass",
			fieldName: "USER_ID",
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

func TestCreateRowMap(t *testing.T) {
	mockLog := testLogger()

	tests := []struct {
		name    string
		columns []string
		values  []any
		want    map[string]any
	}{
		{
			name:    "simple string values",
			columns: []string{"NAME", "EMAIL"},
			values:  []any{"John Doe", "john@example.com"},
			want: map[string]any{
				"NAME":  "John Doe",
				"EMAIL": "john@example.com",
			},
		},
		{
			name:    "mixed types",
			columns: []string{"ID", "NAME", "AGE", "ACTIVE"},
			values:  []any{123, "Alice", 30, true},
			want: map[string]any{
				"ID":     123,
				"NAME":   "Alice",
				"AGE":    30,
				"ACTIVE": true,
			},
		},
		{
			name:    "with nil values",
			columns: []string{"NAME", "EMAIL", "PHONE"},
			values:  []any{"Bob", nil, "555-1234"},
			want: map[string]any{
				"NAME":  "Bob",
				"EMAIL": nil,
				"PHONE": "555-1234",
			},
		},
		{
			name:    "with JSON byte data",
			columns: []string{"ID", "METADATA"},
			values:  []any{1, []uint8(`{"key":"value"}`)},
			want: map[string]any{
				"ID":       1,
				"METADATA": map[string]any{"key": "value"},
			},
		},
		{
			name:    "empty columns and values",
			columns: []string{},
			values:  []any{},
			want:    map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createRowMap(tt.columns, tt.values, mockLog)

			assert.Equal(t, len(tt.want), len(got))
			for key, wantVal := range tt.want {
				gotVal, ok := got[key]
				assert.True(t, ok, "expected key %q", key)

				// Handle map comparison
				if gotMap, ok := gotVal.(map[string]any); ok {
					wantMap, ok := wantVal.(map[string]any)
					assert.True(t, ok)
					assert.Equal(t, wantMap, gotMap)
				} else {
					assert.Equal(t, wantVal, gotVal)
				}
			}
		})
	}
}

func TestParseJSONField(t *testing.T) {
	mockLog := testLogger()

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseJSONField(tt.value, mockLog)

			// Handle different types for comparison
			switch expected := tt.want.(type) {
			case map[string]any:
				gotMap, ok := got.(map[string]any)
				assert.True(t, ok)
				assert.Equal(t, expected, gotMap)
			case []any:
				gotSlice, ok := got.([]any)
				assert.True(t, ok)
				assert.Equal(t, expected, gotSlice)
			default:
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestApplyFilter(t *testing.T) {
	ora := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Colon)

	tests := []struct {
		name       string
		fieldName  string
		values     []any
		wantClause string
	}{
		{
			name:       "single value",
			fieldName:  "STATUS",
			values:     []any{"active"},
			wantClause: "STATUS",
		},
		{
			name:       "multiple values",
			fieldName:  "ID",
			values:     []any{1, 2, 3},
			wantClause: "ID IN",
		},
		{
			name:       "empty values returns unchanged",
			fieldName:  "NAME",
			values:     []any{},
			wantClause: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseQuery := ora.Select("*").From("USERS")
			resultQuery := applyFilter(baseQuery, tt.fieldName, tt.values)

			sql, args, err := resultQuery.ToSql()
			require.NoError(t, err)

			if tt.wantClause == "" {
				baseSql, _, _ := baseQuery.ToSql()
				assert.Equal(t, baseSql, sql)
			} else {
				assert.Contains(t, sql, tt.wantClause)
				if len(tt.values) > 0 {
					assert.Equal(t, len(tt.values), len(args))
				}
			}
		})
	}
}

func TestBuildDynamicFilters(t *testing.T) {
	ora := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Colon)

	schema := []TableSchema{
		{
			TableName: "USERS",
			Columns: []ColumnInformation{
				{Name: "ID"},
				{Name: "NAME"},
				{Name: "EMAIL"},
				{Name: "AGE"},
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
			table: "USERS",
			filter: map[string][]any{
				"NAME": {"John"},
			},
			wantFields: []string{"NAME"},
		},
		{
			name:  "filter by multiple fields",
			table: "USERS",
			filter: map[string][]any{
				"NAME": {"John"},
				"AGE":  {25},
			},
			wantFields: []string{"NAME", "AGE"},
		},
		{
			name:       "empty filter",
			table:      "USERS",
			filter:     map[string][]any{},
			wantFields: []string{},
		},
		{
			name:  "filter with invalid field name (should be ignored)",
			table: "USERS",
			filter: map[string][]any{
				"INVALID_FIELD": {"value"},
				"NAME":          {"John"},
			},
			wantFields: []string{"NAME"},
		},
		{
			name:  "filter with empty values (should be ignored)",
			table: "USERS",
			filter: map[string][]any{
				"NAME": {},
			},
			wantFields: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseQuery := ora.Select("*").From(tt.table)
			resultQuery := buildDynamicFilters(baseQuery, schema, tt.table, tt.filter)

			sql, _, err := resultQuery.ToSql()
			require.NoError(t, err)

			for _, field := range tt.wantFields {
				assert.Contains(t, sql, field)
			}
		})
	}
}

func TestApplyAdvancedFilter(t *testing.T) {
	ora := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Colon)
	ds := &ExternalDataSource{}

	tests := []struct {
		name      string
		field     string
		condition job.FilterCondition
	}{
		{
			name:  "equals with single value",
			field: "STATUS",
			condition: job.FilterCondition{
				Equals: []any{"active"},
			},
		},
		{
			name:  "equals with multiple values (IN clause)",
			field: "STATUS",
			condition: job.FilterCondition{
				Equals: []any{"active", "pending", "completed"},
			},
		},
		{
			name:  "greater than",
			field: "AGE",
			condition: job.FilterCondition{
				GreaterThan: []any{18},
			},
		},
		{
			name:  "greater or equal",
			field: "AGE",
			condition: job.FilterCondition{
				GreaterOrEqual: []any{21},
			},
		},
		{
			name:  "less than",
			field: "AGE",
			condition: job.FilterCondition{
				LessThan: []any{65},
			},
		},
		{
			name:  "less or equal",
			field: "AGE",
			condition: job.FilterCondition{
				LessOrEqual: []any{100},
			},
		},
		{
			name:  "between two values",
			field: "AGE",
			condition: job.FilterCondition{
				Between: []any{18, 65},
			},
		},
		{
			name:  "in clause",
			field: "CATEGORY",
			condition: job.FilterCondition{
				In: []any{"A", "B", "C"},
			},
		},
		{
			name:  "not in clause",
			field: "STATUS",
			condition: job.FilterCondition{
				NotIn: []any{"deleted", "archived"},
			},
		},
		{
			name:  "between for date field with string dates",
			field: "CREATED_AT",
			condition: job.FilterCondition{
				Between: []any{"2023-01-01", "2023-12-31"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseQuery := ora.Select("*").From("TEST_TABLE")
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

func TestBuildAdvancedFilters(t *testing.T) {
	ora := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Colon)
	ds := &ExternalDataSource{}

	schema := []TableSchema{
		{
			TableName: "USERS",
			Columns: []ColumnInformation{
				{Name: "ID"},
				{Name: "NAME"},
				{Name: "AGE"},
				{Name: "STATUS"},
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
			table: "USERS",
			filter: map[string]job.FilterCondition{
				"STATUS": {
					Equals: []any{"active"},
				},
			},
			wantErr: false,
		},
		{
			name:  "multiple field filters",
			table: "USERS",
			filter: map[string]job.FilterCondition{
				"STATUS": {
					Equals: []any{"active"},
				},
				"AGE": {
					GreaterThan: []any{18},
				},
			},
			wantErr: false,
		},
		{
			name:    "empty filter",
			table:   "USERS",
			filter:  map[string]job.FilterCondition{},
			wantErr: false,
		},
		{
			name:  "filter with invalid field (should be ignored)",
			table: "USERS",
			filter: map[string]job.FilterCondition{
				"INVALID_FIELD": {
					Equals: []any{"value"},
				},
			},
			wantErr: false,
		},
		{
			name:  "filter with invalid between (wrong count)",
			table: "USERS",
			filter: map[string]job.FilterCondition{
				"AGE": {
					Between: []any{18}, // Should have 2 values
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseQuery := ora.Select("*").From(tt.table)
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

func TestBuildDynamicFilters_WithSchemaPrefix(t *testing.T) {
	ora := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Colon)

	schema := []TableSchema{
		{
			TableName: "USERS",
			Columns: []ColumnInformation{
				{Name: "ID"},
				{Name: "NAME"},
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
			table: "MYSCHEMA.USERS",
			filter: map[string][]any{
				"NAME": {"John"},
			},
		},
		{
			name:  "table without schema prefix",
			table: "USERS",
			filter: map[string][]any{
				"NAME": {"Jane"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseQuery := ora.Select("*").From(tt.table)
			resultQuery := buildDynamicFilters(baseQuery, schema, tt.table, tt.filter)

			sql, _, err := resultQuery.ToSql()
			require.NoError(t, err)
			assert.NotEmpty(t, sql)
		})
	}
}

func TestApplyAdvancedFilter_DateFieldHandling(t *testing.T) {
	ora := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Colon)
	ds := &ExternalDataSource{}

	tests := []struct {
		name      string
		field     string
		condition job.FilterCondition
	}{
		{
			name:  "date field between with date strings",
			field: "CREATED_AT",
			condition: job.FilterCondition{
				Between: []any{"2023-01-01", "2023-12-31"},
			},
		},
		{
			name:  "date field between adjusts end date",
			field: "UPDATED_AT",
			condition: job.FilterCondition{
				Between: []any{"2023-01-01", "2023-01-31"},
			},
		},
		{
			name:  "non-date field between",
			field: "AGE",
			condition: job.FilterCondition{
				Between: []any{18, 65},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseQuery := ora.Select("*").From("TEST_TABLE")
			resultQuery := ds.applyAdvancedFilter(baseQuery, tt.field, tt.condition)

			sql, args, err := resultQuery.ToSql()
			require.NoError(t, err)

			assert.Contains(t, sql, tt.field)
			assert.NotEmpty(t, args)
		})
	}
}

func TestApplyAdvancedFilter_AllOperators(t *testing.T) {
	ora := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Colon)
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

	baseQuery := ora.Select("*").From("TEST_TABLE")
	resultQuery := ds.applyAdvancedFilter(baseQuery, "TEST_FIELD", condition)

	sql, args, err := resultQuery.ToSql()
	require.NoError(t, err)

	assert.Contains(t, sql, "TEST_FIELD")
	assert.GreaterOrEqual(t, len(args), 5)
}

func TestBuildAdvancedFilters_WithSchemaPrefix(t *testing.T) {
	ora := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Colon)
	ds := &ExternalDataSource{}

	schema := []TableSchema{
		{
			TableName: "ORDERS",
			Columns: []ColumnInformation{
				{Name: "ID"},
				{Name: "AMOUNT"},
				{Name: "STATUS"},
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
			name:  "table with schema prefix MYSCHEMA.ORDERS",
			table: "MYSCHEMA.ORDERS",
			filter: map[string]job.FilterCondition{
				"STATUS": {
					Equals: []any{"completed"},
				},
			},
			wantErr: false,
		},
		{
			name:  "table with schema prefix SALES.ORDERS",
			table: "SALES.ORDERS",
			filter: map[string]job.FilterCondition{
				"AMOUNT": {
					GreaterThan: []any{100},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseQuery := ora.Select("*").From(tt.table)
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

func TestTableSchema_Structure(t *testing.T) {
	schema := TableSchema{
		TableName: "TEST_TABLE",
		Columns: []ColumnInformation{
			{
				Name:         "ID",
				DataType:     "NUMBER",
				IsNullable:   false,
				IsPrimaryKey: true,
			},
			{
				Name:         "NAME",
				DataType:     "VARCHAR2",
				IsNullable:   true,
				IsPrimaryKey: false,
			},
		},
	}

	assert.Equal(t, "TEST_TABLE", schema.TableName)
	assert.Len(t, schema.Columns, 2)
	assert.Equal(t, "ID", schema.Columns[0].Name)
	assert.True(t, schema.Columns[0].IsPrimaryKey)
	assert.Equal(t, "NAME", schema.Columns[1].Name)
	assert.False(t, schema.Columns[1].IsPrimaryKey)
}

func TestConnection_InitialState(t *testing.T) {
	mockLog := testLogger()

	tests := []struct {
		name string
		conn *Connection
	}{
		{
			name: "with connection string",
			conn: &Connection{
				ConnectionString:   "oracle://localhost/test",
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
				ConnectionString:   "oracle://localhost/mydb",
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

func TestConnect_ErrorHandling(t *testing.T) {
	mockLog := testLogger()

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

func TestGetDB_WithNilConnection(t *testing.T) {
	mockLog := testLogger()

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

func TestGetDB_MultipleCallsWithError(t *testing.T) {
	mockLog := testLogger()

	conn := &Connection{
		ConnectionString:   "invalid",
		Logger:             mockLog,
		ConnectionDB:       nil,
		MaxOpenConnections: 10,
		MaxIdleConnections: 5,
	}

	// First call should attempt to connect and fail
	_, err1 := conn.GetDB()
	assert.Error(t, err1)

	// Second call should also attempt to connect (since ConnectionDB is still nil)
	_, err2 := conn.GetDB()
	assert.Error(t, err2)
}

// mockLogger implements a basic logger for testing.
type mockLogger struct{}

func (m *mockLogger) Info(args ...any)                                      {}
func (m *mockLogger) Infof(format string, args ...any)                      {}
func (m *mockLogger) Warn(args ...any)                                      {}
func (m *mockLogger) Warnf(format string, args ...any)                      {}
func (m *mockLogger) Error(args ...any)                                     {}
func (m *mockLogger) Errorf(format string, args ...any)                     {}
func (m *mockLogger) Debug(args ...any)                                     {}
func (m *mockLogger) Debugf(format string, args ...any)                     {}
func (m *mockLogger) Fatal(args ...any)                                     {}
func (m *mockLogger) Fatalf(format string, args ...any)                     {}
func (m *mockLogger) Panic(args ...any)                                     {}
func (m *mockLogger) Panicf(format string, args ...any)                     {}
func (m *mockLogger) WithFields(fields ...any) log.Logger                   { return m }
func (m *mockLogger) WithField(key string, value any) log.Logger            { return m }
func (m *mockLogger) WithError(err error) log.Logger                        { return m }
func (m *mockLogger) GetLevel() string                                      { return "info" }
func (m *mockLogger) SetLevel(level string) error                           { return nil }
func (m *mockLogger) IsLevelEnabled(level string) bool                      { return true }
func (m *mockLogger) GetLogger() any                                        { return m }
func (m *mockLogger) GetOutput() any                                        { return nil }
func (m *mockLogger) SetOutput(output any) error                            { return nil }
func (m *mockLogger) GetFormatter() any                                     { return nil }
func (m *mockLogger) SetFormatter(formatter any) error                      { return nil }
func (m *mockLogger) GetHooks() any                                         { return nil }
func (m *mockLogger) AddHook(hook any) error                                { return nil }
func (m *mockLogger) Clone() any                                            { return m }
func (m *mockLogger) GetContext() any                                       { return nil }
func (m *mockLogger) SetContext(ctx any) error                              { return nil }
func (m *mockLogger) GetCallerInfo() bool                                   { return false }
func (m *mockLogger) SetCallerInfo(enabled bool)                            {}
func (m *mockLogger) GetReportCaller() bool                                 { return false }
func (m *mockLogger) SetReportCaller(enabled bool)                          {}
func (m *mockLogger) GetExitFunc() any                                      { return nil }
func (m *mockLogger) SetExitFunc(exitFunc any) error                        { return nil }
func (m *mockLogger) GetBufferPool() any                                    { return nil }
func (m *mockLogger) SetBufferPool(pool any) error                          { return nil }
func (m *mockLogger) Printf(format string, args ...any)                     {}
func (m *mockLogger) Print(args ...any)                                     {}
func (m *mockLogger) Println(args ...any)                                   {}
func (m *mockLogger) Trace(args ...any)                                     {}
func (m *mockLogger) Tracef(format string, args ...any)                     {}
func (m *mockLogger) Traceln(args ...any)                                   {}
func (m *mockLogger) Infoln(args ...any)                                    {}
func (m *mockLogger) Warnln(args ...any)                                    {}
func (m *mockLogger) Warningln(args ...any)                                 {}
func (m *mockLogger) Errorln(args ...any)                                   {}
func (m *mockLogger) Fatalln(args ...any)                                   {}
func (m *mockLogger) Panicln(args ...any)                                   {}
func (m *mockLogger) Debugln(args ...any)                                   {}
func (m *mockLogger) Warning(args ...any)                                   {}
func (m *mockLogger) Warningf(format string, args ...any)                   {}
func (m *mockLogger) Log(level string, args ...any)                         {}
func (m *mockLogger) Logf(level string, format string, args ...any)         {}
func (m *mockLogger) Logln(level string, args ...any)                       {}
func (m *mockLogger) Sync() error                                           { return nil }
func (m *mockLogger) WithDefaultMessageTemplate(template string) log.Logger { return m }
