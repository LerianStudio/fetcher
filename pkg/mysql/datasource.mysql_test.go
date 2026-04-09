package mysql

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/fetcher/pkg/testutil"
	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

func testContext() context.Context {
	logger := &libLog.GoLogger{Level: libLog.LevelDebug}
	values := &libCommons.CustomContextKeyValue{
		HeaderID: "test-request-id",
		Logger:   logger,
		Tracer:   otel.Tracer("test"),
	}
	return context.WithValue(context.Background(), libCommons.CustomContextKey, values)
}

func testLogger() *testutil.MockLogger {
	return &testutil.MockLogger{}
}

func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return db, mock
}

// =============================================================================
// NewDataSourceRepository Tests
// =============================================================================

func TestNewDataSourceRepository_MySQL(t *testing.T) {
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

	t.Run("failure with nil connection", func(t *testing.T) {
		conn := &Connection{
			ConnectionDB:     nil,
			Connected:        false,
			Logger:           testLogger(),
			ConnectionString: "invalid://connection",
		}

		ds, err := NewDataSourceRepository(conn)

		assert.Error(t, err)
		assert.Nil(t, ds)
	})
}

// =============================================================================
// CloseConnection Tests
// =============================================================================

func TestExternalDataSource_MySQL_CloseConnection(t *testing.T) {
	t.Run("closes connection successfully", func(t *testing.T) {
		db, mock := setupMockDB(t)

		// Expect the Close call
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

	t.Run("no error when connection is nil", func(t *testing.T) {
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

// =============================================================================
// Query Tests
// =============================================================================

func TestExternalDataSource_MySQL_Query(t *testing.T) {
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

		rows := sqlmock.NewRows([]string{"id", "name"}).
			AddRow(1, "Alice").
			AddRow(2, "Bob")

		mock.ExpectQuery("SELECT id, name FROM users").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "int"},
					{Name: "name", DataType: "varchar"},
				},
			},
		}

		result, err := ds.Query(ctx, schema, "users", []string{"id", "name"}, nil)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, int64(1), result[0]["id"])
		assert.Equal(t, "Alice", result[0]["name"])
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful query with wildcard fields", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		rows := sqlmock.NewRows([]string{"id", "name", "email"}).
			AddRow(1, "Alice", "alice@example.com")

		mock.ExpectQuery("SELECT id, name, email FROM users").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "int"},
					{Name: "name", DataType: "varchar"},
					{Name: "email", DataType: "varchar"},
				},
			},
		}

		result, err := ds.Query(ctx, schema, "users", []string{"*"}, nil)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful query with filters", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		rows := sqlmock.NewRows([]string{"id", "name"}).
			AddRow(1, "Alice")

		mock.ExpectQuery("SELECT id, name FROM users WHERE name IN").
			WithArgs("Alice").
			WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "int"},
					{Name: "name", DataType: "varchar"},
				},
			},
		}

		filter := map[string][]any{
			"name": {"Alice"},
		}

		result, err := ds.Query(ctx, schema, "users", []string{"id", "name"}, filter)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query with invalid table", func(t *testing.T) {
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
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "int"},
				},
			},
		}

		result, err := ds.Query(ctx, schema, "nonexistent", []string{"id"}, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("query with invalid fields", func(t *testing.T) {
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
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "int"},
				},
			},
		}

		result, err := ds.Query(ctx, schema, "users", []string{"invalid_field"}, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid fields")
	})

	t.Run("query with database error", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		mock.ExpectQuery("SELECT").WillReturnError(errors.New("database error"))

		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "int"},
				},
			},
		}

		result, err := ds.Query(ctx, schema, "users", []string{"id"}, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "error executing query")
	})

	t.Run("query returns empty result", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		rows := sqlmock.NewRows([]string{"id", "name"})

		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "int"},
					{Name: "name", DataType: "varchar"},
				},
			},
		}

		result, err := ds.Query(ctx, schema, "users", []string{"id", "name"}, nil)

		require.NoError(t, err)
		assert.Empty(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// =============================================================================
// QueryWithAdvancedFilters Tests
// =============================================================================

func TestExternalDataSource_MySQL_QueryWithAdvancedFilters(t *testing.T) {
	ctx := testContext()

	t.Run("successful query with equals filter", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		rows := sqlmock.NewRows([]string{"id", "status"}).
			AddRow(1, "active")

		mock.ExpectQuery("SELECT id, status FROM orders WHERE status = ").
			WithArgs("active").
			WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "orders",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "int"},
					{Name: "status", DataType: "varchar"},
				},
			},
		}

		filter := map[string]job.FilterCondition{
			"status": {
				Equals: []any{"active"},
			},
		}

		result, err := ds.QueryWithAdvancedFilters(ctx, schema, "orders", []string{"id", "status"}, filter)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful query with range filters", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		rows := sqlmock.NewRows([]string{"id", "amount"}).
			AddRow(1, 150.00).
			AddRow(2, 200.00)

		mock.ExpectQuery("SELECT id, amount FROM orders").
			WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "orders",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "int"},
					{Name: "amount", DataType: "decimal"},
				},
			},
		}

		filter := map[string]job.FilterCondition{
			"amount": {
				GreaterThan: []any{100},
			},
		}

		result, err := ds.QueryWithAdvancedFilters(ctx, schema, "orders", []string{"id", "amount"}, filter)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful query with between filter", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		rows := sqlmock.NewRows([]string{"id", "age"}).
			AddRow(1, 25).
			AddRow(2, 30)

		mock.ExpectQuery("SELECT id, age FROM users").
			WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "int"},
					{Name: "age", DataType: "int"},
				},
			},
		}

		filter := map[string]job.FilterCondition{
			"age": {
				Between: []any{18, 35},
			},
		}

		result, err := ds.QueryWithAdvancedFilters(ctx, schema, "users", []string{"id", "age"}, filter)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful query with in filter", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		rows := sqlmock.NewRows([]string{"id", "category"}).
			AddRow(1, "A").
			AddRow(2, "B")

		mock.ExpectQuery("SELECT id, category FROM products").
			WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "products",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "int"},
					{Name: "category", DataType: "varchar"},
				},
			},
		}

		filter := map[string]job.FilterCondition{
			"category": {
				In: []any{"A", "B", "C"},
			},
		}

		result, err := ds.QueryWithAdvancedFilters(ctx, schema, "products", []string{"id", "category"}, filter)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query with invalid between filter", func(t *testing.T) {
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
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "int"},
					{Name: "age", DataType: "int"},
				},
			},
		}

		filter := map[string]job.FilterCondition{
			"age": {
				Between: []any{18}, // Only one value, should have two
			},
		}

		result, err := ds.QueryWithAdvancedFilters(ctx, schema, "users", []string{"id", "age"}, filter)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "between operator")
	})
}

// =============================================================================
// ValidateTableAndFields Tests
// =============================================================================

func TestExternalDataSource_MySQL_ValidateTableAndFields(t *testing.T) {
	ctx := testContext()

	tests := []struct {
		name          string
		schema        []TableSchema
		table         string
		fields        []string
		expectError   bool
		expectedCount int
	}{
		{
			name: "valid table and fields",
			schema: []TableSchema{
				{
					TableName: "users",
					Columns: []ColumnInformation{
						{Name: "id"},
						{Name: "name"},
						{Name: "email"},
					},
				},
			},
			table:         "users",
			fields:        []string{"id", "name"},
			expectError:   false,
			expectedCount: 2,
		},
		{
			name: "invalid table",
			schema: []TableSchema{
				{TableName: "users"},
			},
			table:       "nonexistent",
			fields:      []string{"id"},
			expectError: true,
		},
		{
			name: "invalid field",
			schema: []TableSchema{
				{
					TableName: "users",
					Columns: []ColumnInformation{
						{Name: "id"},
						{Name: "name"},
					},
				},
			},
			table:       "users",
			fields:      []string{"id", "invalid_field"},
			expectError: true,
		},
		{
			name: "wildcard field returns all columns",
			schema: []TableSchema{
				{
					TableName: "users",
					Columns: []ColumnInformation{
						{Name: "id"},
						{Name: "name"},
						{Name: "email"},
					},
				},
			},
			table:         "users",
			fields:        []string{"*"},
			expectError:   false,
			expectedCount: 3,
		},
		{
			name: "no valid fields specified",
			schema: []TableSchema{
				{
					TableName: "users",
					Columns: []ColumnInformation{
						{Name: "id"},
					},
				},
			},
			table:       "users",
			fields:      []string{"nonexistent1", "nonexistent2"},
			expectError: true,
		},
		{
			name: "empty fields",
			schema: []TableSchema{
				{
					TableName: "users",
					Columns: []ColumnInformation{
						{Name: "id"},
					},
				},
			},
			table:       "users",
			fields:      []string{},
			expectError: true,
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

			validFields, err := ds.ValidateTableAndFields(ctx, tt.table, tt.fields, tt.schema)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, validFields)
			} else {
				assert.NoError(t, err)
				assert.Len(t, validFields, tt.expectedCount)
			}
		})
	}
}

// =============================================================================
// GetDatabaseSchema Tests
// =============================================================================

func TestExternalDataSource_MySQL_GetDatabaseSchema(t *testing.T) {
	ctx := testContext()

	t.Run("successful schema retrieval", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		// Mock tables query
		tableRows := sqlmock.NewRows([]string{"table_name"}).
			AddRow("users").
			AddRow("orders")
		mock.ExpectQuery("SELECT table_name FROM information_schema.tables").
			WillReturnRows(tableRows)

		// Mock primary keys query
		pkRows := sqlmock.NewRows([]string{"table_name", "column_name"}).
			AddRow("users", "id").
			AddRow("orders", "id")
		mock.ExpectQuery("SELECT tc.table_name, kc.column_name").
			WillReturnRows(pkRows)

		// Mock column queries for each table
		usersColumnRows := sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable"}).
			AddRow("id", "int", false).
			AddRow("name", "varchar", true)
		mock.ExpectQuery("SELECT column_name, data_type").
			WithArgs("users").
			WillReturnRows(usersColumnRows)

		ordersColumnRows := sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable"}).
			AddRow("id", "int", false).
			AddRow("user_id", "int", false).
			AddRow("amount", "decimal", true)
		mock.ExpectQuery("SELECT column_name, data_type").
			WithArgs("orders").
			WillReturnRows(ordersColumnRows)

		schema, err := ds.GetDatabaseSchema(ctx)

		require.NoError(t, err)
		assert.Len(t, schema, 2)

		// Verify users table
		assert.Equal(t, "users", schema[0].TableName)
		assert.Len(t, schema[0].Columns, 2)
		assert.True(t, schema[0].Columns[0].IsPrimaryKey)

		// Verify orders table
		assert.Equal(t, "orders", schema[1].TableName)
		assert.Len(t, schema[1].Columns, 3)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error querying tables", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		mock.ExpectQuery("SELECT table_name FROM information_schema.tables").
			WillReturnError(errors.New("database error"))

		schema, err := ds.GetDatabaseSchema(ctx)

		assert.Error(t, err)
		assert.Nil(t, schema)
		assert.Contains(t, err.Error(), "error querying tables")
	})

	t.Run("error querying primary keys", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		tableRows := sqlmock.NewRows([]string{"table_name"}).
			AddRow("users")
		mock.ExpectQuery("SELECT table_name FROM information_schema.tables").
			WillReturnRows(tableRows)

		mock.ExpectQuery("SELECT tc.table_name, kc.column_name").
			WillReturnError(errors.New("pk query error"))

		schema, err := ds.GetDatabaseSchema(ctx)

		assert.Error(t, err)
		assert.Nil(t, schema)
		assert.Contains(t, err.Error(), "error querying primary keys")
	})

	t.Run("error querying columns", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		tableRows := sqlmock.NewRows([]string{"table_name"}).
			AddRow("users")
		mock.ExpectQuery("SELECT table_name FROM information_schema.tables").
			WillReturnRows(tableRows)

		pkRows := sqlmock.NewRows([]string{"table_name", "column_name"}).
			AddRow("users", "id")
		mock.ExpectQuery("SELECT tc.table_name, kc.column_name").
			WillReturnRows(pkRows)

		mock.ExpectQuery("SELECT column_name, data_type").
			WithArgs("users").
			WillReturnError(errors.New("column query error"))

		schema, err := ds.GetDatabaseSchema(ctx)

		assert.Error(t, err)
		assert.Nil(t, schema)
		assert.Contains(t, err.Error(), "error querying columns")
	})

	t.Run("empty database with no tables", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		tableRows := sqlmock.NewRows([]string{"table_name"})
		mock.ExpectQuery("SELECT table_name FROM information_schema.tables").
			WillReturnRows(tableRows)

		pkRows := sqlmock.NewRows([]string{"table_name", "column_name"})
		mock.ExpectQuery("SELECT tc.table_name, kc.column_name").
			WillReturnRows(pkRows)

		schema, err := ds.GetDatabaseSchema(ctx)

		require.NoError(t, err)
		assert.Empty(t, schema)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// =============================================================================
// ValidateFieldsInSchemaMySQL Tests
// =============================================================================

func TestValidateFieldsInSchemaMySQL(t *testing.T) {
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
			got := ValidateFieldsInSchemaMySQL(tt.expectedFields, tt.schema, &count)

			if count != tt.wantCount {
				t.Errorf("count = %v, want %v", count, tt.wantCount)
			}

			if len(got) != len(tt.wantMissing) {
				t.Errorf("missing fields = %v, want %v", got, tt.wantMissing)
				return
			}

			for i := range got {
				if got[i] != tt.wantMissing[i] {
					t.Errorf("missing fields = %v, want %v", got, tt.wantMissing)
					return
				}
			}
		})
	}
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestIsFilterConditionEmpty_MySQL(t *testing.T) {
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

func TestValidateFilterCondition_MySQL(t *testing.T) {
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
			name:      "valid equals with multiple values",
			fieldName: "status",
			condition: job.FilterCondition{
				Equals: []any{"active", "pending", "completed"},
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

func TestIsLikelyUUIDField_MySQL(t *testing.T) {
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
			name:      "plain id field",
			fieldName: "id",
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

func TestIsValidUUIDFormat_MySQL(t *testing.T) {
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
			if got != tt.want {
				t.Errorf("isValidUUIDFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsDateField_MySQL(t *testing.T) {
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
			name:      "non-date field name",
			fieldName: "name",
			want:      false,
		},
		{
			name:      "non-date field id",
			fieldName: "id",
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

func TestIsDateString_MySQL(t *testing.T) {
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
			name:  "not a string value - int",
			value: 12345,
			want:  false,
		},
		{
			name:  "not a string value - nil",
			value: nil,
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

// =============================================================================
// Filter Builder Tests
// =============================================================================

func TestBuildDynamicFilters_MySQL(t *testing.T) {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question)

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
				if !containsMySQL(sql, field) {
					t.Errorf("buildDynamicFilters() SQL = %q, want to contain field %q", sql, field)
				}
			}
		})
	}
}

func TestApplyAdvancedFilter_MySQL(t *testing.T) {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question)
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
			if !containsMySQL(sql, tt.field) {
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

func TestBuildAdvancedFilters_MySQL(t *testing.T) {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question)
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

func TestApplyFilter_MySQL(t *testing.T) {
	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question)

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
			wantClause: "status IN",
		},
		{
			name:       "multiple values",
			fieldName:  "id",
			values:     []any{1, 2, 3},
			wantClause: "id IN",
		},
		{
			name:       "empty values returns unchanged",
			fieldName:  "name",
			values:     []any{},
			wantClause: "",
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
				if tt.wantClause != "" && !containsMySQL(sql, tt.wantClause) {
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

// =============================================================================
// JSON Parsing Tests
// =============================================================================

func TestParseJSONField_MySQL(t *testing.T) {
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
			mockLog := testLogger()
			got := parseJSONField(tt.value, mockLog)

			// For map comparisons, we need to do a deep comparison
			if gotMap, ok := got.(map[string]any); ok {
				wantMap := tt.want.(map[string]any)
				if !mapsEqualMySQL(gotMap, wantMap) {
					t.Errorf("parseJSONField() = %v, want %v", got, tt.want)
				}
			} else if gotSlice, ok := got.([]any); ok {
				wantSlice := tt.want.([]any)
				if !slicesEqualMySQL(gotSlice, wantSlice) {
					t.Errorf("parseJSONField() = %v, want %v", got, tt.want)
				}
			} else if gotBytes, ok := got.([]uint8); ok {
				wantBytes := tt.want.([]uint8)
				if !bytesEqualMySQL(gotBytes, wantBytes) {
					t.Errorf("parseJSONField() = %v, want %v", got, tt.want)
				}
			} else {
				if got != tt.want {
					t.Errorf("parseJSONField() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestCreateRowMap_MySQL(t *testing.T) {
	mockLog := testLogger()

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createRowMap(tt.columns, tt.values, mockLog)

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
						if !mapsEqualMySQL(gotMap, wantMap) {
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

// =============================================================================
// Connection Tests
// =============================================================================

func TestConnect_ErrorHandling_MySQL(t *testing.T) {
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

func TestGetDB_WithNilConnection_MySQL(t *testing.T) {
	mockLog := testLogger()

	conn := &Connection{
		ConnectionString:   "invalid://connection",
		Logger:             mockLog,
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

func TestConnection_InitialState_MySQL(t *testing.T) {
	mockLog := testLogger()

	tests := []struct {
		name string
		conn *Connection
	}{
		{
			name: "with connection string",
			conn: &Connection{
				ConnectionString:   "mysql://localhost/test",
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
				ConnectionString:   "mysql://localhost/mydb",
				DBName:             "mydb",
				Logger:             mockLog,
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

// =============================================================================
// Schema Structure Tests
// =============================================================================

func TestTableSchema_Structure_MySQL(t *testing.T) {
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

// =============================================================================
// Helper Functions
// =============================================================================

func containsMySQL(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func mapsEqualMySQL(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || !valuesEqualMySQL(v, bv) {
			return false
		}
	}
	return true
}

func slicesEqualMySQL(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !valuesEqualMySQL(a[i], b[i]) {
			return false
		}
	}
	return true
}

func bytesEqualMySQL(a, b []uint8) bool {
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

func valuesEqualMySQL(a, b any) bool {
	if aMap, ok := a.(map[string]any); ok {
		if bMap, ok := b.(map[string]any); ok {
			return mapsEqualMySQL(aMap, bMap)
		}
		return false
	}
	return a == b
}
