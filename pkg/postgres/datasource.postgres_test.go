package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	libLog "github.com/LerianStudio/lib-commons/v2/commons/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
)

// testContext creates a context with proper tracking values for testing
func testContext() context.Context {
	logger := &libLog.GoLogger{Level: libLog.DebugLevel}
	values := &libCommons.CustomContextKeyValue{
		HeaderID: "test-request-id",
		Logger:   logger,
		Tracer:   otel.Tracer("test"),
	}
	return context.WithValue(context.Background(), libCommons.CustomContextKey, values)
}

// testLogger creates a logger for testing
func testLogger() *libLog.GoLogger {
	return &libLog.GoLogger{Level: libLog.ErrorLevel}
}

// setupMockDB creates a mock database connection for testing
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

func TestExternalDataSource_CloseConnection(t *testing.T) {
	t.Run("closes connection successfully", func(t *testing.T) {
		db, mock := setupMockDB(t)

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

		rows := sqlmock.NewRows([]string{"id", "name"}).
			AddRow(1, "Alice").
			AddRow(2, "Bob")

		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "integer", IsNullable: false, IsPrimaryKey: true},
					{Name: "name", DataType: "varchar", IsNullable: true, IsPrimaryKey: false},
				},
			},
		}

		result, err := ds.Query(ctx, schema, "users", []string{"id", "name"}, nil)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query with filters", func(t *testing.T) {
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

		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "integer"},
					{Name: "name", DataType: "varchar"},
				},
			},
		}

		filters := map[string][]any{"id": {1}}

		result, err := ds.Query(ctx, schema, "users", []string{"id", "name"}, filters)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query with star selector returns all columns", func(t *testing.T) {
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

		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "integer"},
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
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "integer"},
				},
			},
		}

		result, err := ds.Query(ctx, schema, "nonexistent", []string{"id"}, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
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
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "integer"},
				},
			},
		}

		result, err := ds.Query(ctx, schema, "users", []string{"nonexistent_field"}, nil)

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

		mock.ExpectQuery("SELECT").WillReturnError(errors.New("database connection error"))

		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "integer"},
				},
			},
		}

		result, err := ds.Query(ctx, schema, "users", []string{"id"}, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "error executing query")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query with multiple filters on same field", func(t *testing.T) {
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

		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "integer"},
					{Name: "name", DataType: "varchar"},
				},
			},
		}

		filters := map[string][]any{"id": {1, 2, 3}}

		result, err := ds.Query(ctx, schema, "users", []string{"id", "name"}, filters)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query with schema prefix in table name", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "integer"},
				},
			},
		}

		result, err := ds.Query(ctx, schema, "public.users", []string{"id"}, nil)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query with no valid fields specified", func(t *testing.T) {
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
					{Name: "id", DataType: "integer"},
				},
			},
		}

		result, err := ds.Query(ctx, schema, "users", []string{}, nil)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "no valid fields")
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
		errorMsg    string
	}{
		{
			name: "valid table and fields",
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
			fields:      []string{"id", "name"},
			expectError: false,
		},
		{
			name: "invalid table",
			schema: []TableSchema{
				{TableName: "users"},
			},
			table:       "nonexistent",
			fields:      []string{"id"},
			expectError: true,
			errorMsg:    "does not exist",
		},
		{
			name: "invalid field",
			schema: []TableSchema{
				{
					TableName: "users",
					Columns:   []ColumnInformation{{Name: "id"}},
				},
			},
			table:       "users",
			fields:      []string{"nonexistent"},
			expectError: true,
			errorMsg:    "invalid fields",
		},
		{
			name: "star selector returns all columns",
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
			table:       "users",
			fields:      []string{"*"},
			expectError: false,
		},
		{
			name: "mixed valid and invalid fields",
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
			errorMsg:    "invalid fields",
		},
		{
			name: "empty fields list",
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
			errorMsg:    "no valid fields",
		},
		{
			name: "table with schema prefix",
			schema: []TableSchema{
				{
					TableName: "orders",
					Columns: []ColumnInformation{
						{Name: "id"},
						{Name: "amount"},
					},
				},
			},
			table:       "public.orders",
			fields:      []string{"id", "amount"},
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

			validFields, err := ds.ValidateTableAndFields(ctx, tt.table, tt.fields, tt.schema)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, validFields)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, validFields)
			}
		})
	}
}

func TestExternalDataSource_GetDatabaseSchema(t *testing.T) {
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

		// Mock table query
		tableRows := sqlmock.NewRows([]string{"table_name"}).
			AddRow("users").
			AddRow("orders")
		mock.ExpectQuery("SELECT table_name").WillReturnRows(tableRows)

		// Mock primary keys query
		pkRows := sqlmock.NewRows([]string{"table_name", "column_name"}).
			AddRow("users", "id").
			AddRow("orders", "id")
		mock.ExpectQuery("SELECT tc.table_name, kc.column_name").WillReturnRows(pkRows)

		// Mock columns query for users table
		usersColRows := sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable"}).
			AddRow("id", "integer", false).
			AddRow("name", "varchar", true)
		mock.ExpectQuery("SELECT column_name, data_type").WillReturnRows(usersColRows)

		// Mock columns query for orders table
		ordersColRows := sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable"}).
			AddRow("id", "integer", false).
			AddRow("amount", "numeric", false)
		mock.ExpectQuery("SELECT column_name, data_type").WillReturnRows(ordersColRows)

		result, err := ds.GetDatabaseSchema(ctx, []string{"public"})

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("schema retrieval with empty schemas uses default", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		// Mock table query
		tableRows := sqlmock.NewRows([]string{"table_name"}).AddRow("users")
		mock.ExpectQuery("SELECT table_name").WillReturnRows(tableRows)

		// Mock primary keys query
		pkRows := sqlmock.NewRows([]string{"table_name", "column_name"}).
			AddRow("users", "id")
		mock.ExpectQuery("SELECT tc.table_name, kc.column_name").WillReturnRows(pkRows)

		// Mock columns query
		colRows := sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable"}).
			AddRow("id", "integer", false)
		mock.ExpectQuery("SELECT column_name, data_type").WillReturnRows(colRows)

		result, err := ds.GetDatabaseSchema(ctx, []string{})

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("schema retrieval with table query error", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		mock.ExpectQuery("SELECT table_name").WillReturnError(errors.New("database error"))

		result, err := ds.GetDatabaseSchema(ctx, []string{"public"})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "error querying tables")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("schema retrieval with primary key query error", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		// Mock table query
		tableRows := sqlmock.NewRows([]string{"table_name"}).AddRow("users")
		mock.ExpectQuery("SELECT table_name").WillReturnRows(tableRows)

		// Mock primary keys query with error
		mock.ExpectQuery("SELECT tc.table_name, kc.column_name").WillReturnError(errors.New("pk query error"))

		result, err := ds.GetDatabaseSchema(ctx, []string{"public"})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "error querying primary keys")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("schema retrieval with column query error", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		// Mock table query
		tableRows := sqlmock.NewRows([]string{"table_name"}).AddRow("users")
		mock.ExpectQuery("SELECT table_name").WillReturnRows(tableRows)

		// Mock primary keys query
		pkRows := sqlmock.NewRows([]string{"table_name", "column_name"}).
			AddRow("users", "id")
		mock.ExpectQuery("SELECT tc.table_name, kc.column_name").WillReturnRows(pkRows)

		// Mock columns query with error
		mock.ExpectQuery("SELECT column_name, data_type").WillReturnError(errors.New("column query error"))

		result, err := ds.GetDatabaseSchema(ctx, []string{"public"})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "error querying columns")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("schema retrieval with no tables returns empty slice", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		// Mock table query returning no rows
		tableRows := sqlmock.NewRows([]string{"table_name"})
		mock.ExpectQuery("SELECT table_name").WillReturnRows(tableRows)

		// Mock primary keys query
		pkRows := sqlmock.NewRows([]string{"table_name", "column_name"})
		mock.ExpectQuery("SELECT tc.table_name, kc.column_name").WillReturnRows(pkRows)

		result, err := ds.GetDatabaseSchema(ctx, []string{"public"})

		require.NoError(t, err)
		assert.Len(t, result, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("schema retrieval with multiple schemas", func(t *testing.T) {
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
			AddRow("users").
			AddRow("accounts")
		mock.ExpectQuery("SELECT table_name").WillReturnRows(tableRows)

		// Mock primary keys query
		pkRows := sqlmock.NewRows([]string{"table_name", "column_name"}).
			AddRow("users", "id").
			AddRow("accounts", "account_id")
		mock.ExpectQuery("SELECT tc.table_name, kc.column_name").WillReturnRows(pkRows)

		// Mock columns query for users
		usersColRows := sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable"}).
			AddRow("id", "uuid", false)
		mock.ExpectQuery("SELECT column_name, data_type").WillReturnRows(usersColRows)

		// Mock columns query for accounts
		accountsColRows := sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable"}).
			AddRow("account_id", "uuid", false)
		mock.ExpectQuery("SELECT column_name, data_type").WillReturnRows(accountsColRows)

		result, err := ds.GetDatabaseSchema(ctx, []string{"public", "private"})

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestExternalDataSource_QueryWithAdvancedFilters(t *testing.T) {
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
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "integer"},
					{Name: "status", DataType: "varchar"},
				},
			},
		}

		filters := map[string]job.FilterCondition{
			"status": {Equals: []any{"active"}},
		}

		result, err := ds.QueryWithAdvancedFilters(ctx, schema, "users", []string{"id", "status"}, filters)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query with greater than filter", func(t *testing.T) {
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
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "integer"},
					{Name: "age", DataType: "integer"},
				},
			},
		}

		filters := map[string]job.FilterCondition{
			"age": {GreaterThan: []any{20}},
		}

		result, err := ds.QueryWithAdvancedFilters(ctx, schema, "users", []string{"id", "age"}, filters)

		require.NoError(t, err)
		assert.Len(t, result, 2)
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

		rows := sqlmock.NewRows([]string{"id", "price"}).
			AddRow(1, 50.00)
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "products",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "integer"},
					{Name: "price", DataType: "numeric"},
				},
			},
		}

		filters := map[string]job.FilterCondition{
			"price": {Between: []any{10, 100}},
		}

		result, err := ds.QueryWithAdvancedFilters(ctx, schema, "products", []string{"id", "price"}, filters)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query with in filter", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		rows := sqlmock.NewRows([]string{"id", "category"}).
			AddRow(1, "electronics").
			AddRow(2, "clothing")
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "products",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "integer"},
					{Name: "category", DataType: "varchar"},
				},
			},
		}

		filters := map[string]job.FilterCondition{
			"category": {In: []any{"electronics", "clothing", "books"}},
		}

		result, err := ds.QueryWithAdvancedFilters(ctx, schema, "products", []string{"id", "category"}, filters)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query with not in filter", func(t *testing.T) {
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
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "integer"},
					{Name: "status", DataType: "varchar"},
				},
			},
		}

		filters := map[string]job.FilterCondition{
			"status": {NotIn: []any{"deleted", "archived"}},
		}

		result, err := ds.QueryWithAdvancedFilters(ctx, schema, "users", []string{"id", "status"}, filters)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query with combined filters", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		rows := sqlmock.NewRows([]string{"id", "age", "status"}).
			AddRow(1, 25, "active")
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "integer"},
					{Name: "age", DataType: "integer"},
					{Name: "status", DataType: "varchar"},
				},
			},
		}

		filters := map[string]job.FilterCondition{
			"age":    {GreaterOrEqual: []any{18}, LessOrEqual: []any{65}},
			"status": {Equals: []any{"active"}},
		}

		result, err := ds.QueryWithAdvancedFilters(ctx, schema, "users", []string{"id", "age", "status"}, filters)

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
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "integer"},
					{Name: "age", DataType: "integer"},
				},
			},
		}

		filters := map[string]job.FilterCondition{
			"age": {Between: []any{18}}, // Invalid: should have 2 values
		}

		result, err := ds.QueryWithAdvancedFilters(ctx, schema, "users", []string{"id", "age"}, filters)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "between operator")
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
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "integer"},
				},
			},
		}

		filters := map[string]job.FilterCondition{}

		result, err := ds.QueryWithAdvancedFilters(ctx, schema, "nonexistent", []string{"id"}, filters)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "does not exist")
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
					{Name: "id", DataType: "integer"},
				},
			},
		}

		filters := map[string]job.FilterCondition{}

		result, err := ds.QueryWithAdvancedFilters(ctx, schema, "users", []string{"id"}, filters)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "error executing query")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query with date field between filter adjusts end date", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		rows := sqlmock.NewRows([]string{"id", "created_at"}).
			AddRow(1, "2023-06-15")
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "events",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "integer"},
					{Name: "created_at", DataType: "timestamp"},
				},
			},
		}

		filters := map[string]job.FilterCondition{
			"created_at": {Between: []any{"2023-01-01", "2023-12-31"}},
		}

		result, err := ds.QueryWithAdvancedFilters(ctx, schema, "events", []string{"id", "created_at"}, filters)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query with empty filter skipped", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "integer"},
					{Name: "name", DataType: "varchar"},
				},
			},
		}

		filters := map[string]job.FilterCondition{
			"name": {}, // Empty condition should be skipped
		}

		result, err := ds.QueryWithAdvancedFilters(ctx, schema, "users", []string{"id"}, filters)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query with invalid column filter skipped", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		schema := []TableSchema{
			{
				TableName: "users",
				Columns: []ColumnInformation{
					{Name: "id", DataType: "integer"},
				},
			},
		}

		filters := map[string]job.FilterCondition{
			"nonexistent_column": {Equals: []any{"value"}}, // Invalid column should be skipped
		}

		result, err := ds.QueryWithAdvancedFilters(ctx, schema, "users", []string{"id"}, filters)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestScanRows(t *testing.T) {
	t.Run("scans rows with various types", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		rows := sqlmock.NewRows([]string{"id", "name", "active", "balance"}).
			AddRow(1, "Alice", true, 100.50).
			AddRow(2, "Bob", false, 200.75)
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		resultRows, err := db.Query("SELECT * FROM users")
		require.NoError(t, err)
		defer resultRows.Close()

		logger := testLogger()
		result, err := scanRows(resultRows, logger)

		require.NoError(t, err)
		assert.Len(t, result, 2)

		// Verify first row
		assert.Equal(t, int64(1), result[0]["id"])
		assert.Equal(t, "Alice", result[0]["name"])
		assert.Equal(t, true, result[0]["active"])
		assert.Equal(t, 100.50, result[0]["balance"])
	})

	t.Run("scans rows with nil values", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		rows := sqlmock.NewRows([]string{"id", "name"}).
			AddRow(1, nil)
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		resultRows, err := db.Query("SELECT * FROM users")
		require.NoError(t, err)
		defer resultRows.Close()

		logger := testLogger()
		result, err := scanRows(resultRows, logger)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Nil(t, result[0]["name"])
	})

	t.Run("scans rows with JSONB data", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		jsonData := []byte(`{"key":"value"}`)
		rows := sqlmock.NewRows([]string{"id", "metadata"}).
			AddRow(1, jsonData)
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		resultRows, err := db.Query("SELECT * FROM users")
		require.NoError(t, err)
		defer resultRows.Close()

		logger := testLogger()
		result, err := scanRows(resultRows, logger)

		require.NoError(t, err)
		assert.Len(t, result, 1)

		// JSONB should be parsed into map
		metadata, ok := result[0]["metadata"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "value", metadata["key"])
	})

	t.Run("handles empty result set", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		rows := sqlmock.NewRows([]string{"id", "name"})
		mock.ExpectQuery("SELECT").WillReturnRows(rows)

		resultRows, err := db.Query("SELECT * FROM users")
		require.NoError(t, err)
		defer resultRows.Close()

		logger := testLogger()
		result, err := scanRows(resultRows, logger)

		require.NoError(t, err)
		assert.Len(t, result, 0)
	})
}

func TestQueryTables(t *testing.T) {
	ctx := testContext()

	t.Run("returns tables successfully", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		tableRows := sqlmock.NewRows([]string{"table_name"}).
			AddRow("users").
			AddRow("orders").
			AddRow("products")
		mock.ExpectQuery("SELECT table_name").WillReturnRows(tableRows)

		tables, err := ds.queryTables(ctx, []string{"public"})

		require.NoError(t, err)
		assert.Len(t, tables, 3)
		assert.Contains(t, tables, "users")
		assert.Contains(t, tables, "orders")
		assert.Contains(t, tables, "products")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("uses default schema when empty", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		tableRows := sqlmock.NewRows([]string{"table_name"}).AddRow("users")
		mock.ExpectQuery("SELECT table_name").WillReturnRows(tableRows)

		tables, err := ds.queryTables(ctx, []string{})

		require.NoError(t, err)
		assert.Len(t, tables, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("handles query error", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		mock.ExpectQuery("SELECT table_name").WillReturnError(errors.New("query failed"))

		tables, err := ds.queryTables(ctx, []string{"public"})

		assert.Error(t, err)
		assert.Nil(t, tables)
		assert.Contains(t, err.Error(), "error querying tables")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("handles rows.Err after iteration", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		// Create rows with RowError to simulate iteration error
		tableRows := sqlmock.NewRows([]string{"table_name"}).
			AddRow("users").
			RowError(0, errors.New("iteration error"))
		mock.ExpectQuery("SELECT table_name").WillReturnRows(tableRows)

		tables, err := ds.queryTables(ctx, []string{"public"})

		assert.Error(t, err)
		assert.Nil(t, tables)
		assert.Contains(t, err.Error(), "rows error")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestQueryPrimaryKeys(t *testing.T) {
	ctx := testContext()

	t.Run("returns primary keys successfully", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		pkRows := sqlmock.NewRows([]string{"table_name", "column_name"}).
			AddRow("users", "id").
			AddRow("orders", "order_id").
			AddRow("orders", "user_id") // Composite primary key
		mock.ExpectQuery("SELECT tc.table_name, kc.column_name").WillReturnRows(pkRows)

		pks, err := ds.queryPrimaryKeys(ctx, []string{"public"})

		require.NoError(t, err)
		assert.Len(t, pks, 2)
		assert.True(t, pks["users"]["id"])
		assert.True(t, pks["orders"]["order_id"])
		assert.True(t, pks["orders"]["user_id"])
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("handles query error", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		mock.ExpectQuery("SELECT tc.table_name, kc.column_name").WillReturnError(errors.New("query failed"))

		pks, err := ds.queryPrimaryKeys(ctx, []string{"public"})

		assert.Error(t, err)
		assert.Nil(t, pks)
		assert.Contains(t, err.Error(), "error querying primary keys")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("uses default schema when empty", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		pkRows := sqlmock.NewRows([]string{"table_name", "column_name"}).
			AddRow("users", "id")
		mock.ExpectQuery("SELECT tc.table_name, kc.column_name").WillReturnRows(pkRows)

		pks, err := ds.queryPrimaryKeys(ctx, []string{})

		require.NoError(t, err)
		assert.Len(t, pks, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestBuildTableSchema(t *testing.T) {
	ctx := testContext()

	t.Run("builds table schema successfully", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		colRows := sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable"}).
			AddRow("id", "integer", false).
			AddRow("name", "varchar", true).
			AddRow("email", "varchar", false)
		mock.ExpectQuery("SELECT column_name, data_type").WillReturnRows(colRows)

		primaryKeys := map[string]map[string]bool{
			"users": {"id": true},
		}

		schema, err := ds.buildTableSchema(ctx, "users", primaryKeys, testLogger(), []string{"public"})

		require.NoError(t, err)
		assert.Equal(t, "users", schema.TableName)
		assert.Len(t, schema.Columns, 3)

		// Verify columns
		assert.Equal(t, "id", schema.Columns[0].Name)
		assert.True(t, schema.Columns[0].IsPrimaryKey)
		assert.Equal(t, "name", schema.Columns[1].Name)
		assert.False(t, schema.Columns[1].IsPrimaryKey)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("handles query error", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		mock.ExpectQuery("SELECT column_name, data_type").WillReturnError(errors.New("query failed"))

		primaryKeys := map[string]map[string]bool{}

		schema, err := ds.buildTableSchema(ctx, "users", primaryKeys, testLogger(), []string{"public"})

		assert.Error(t, err)
		assert.Equal(t, TableSchema{}, schema)
		assert.Contains(t, err.Error(), "error querying columns")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("uses default schema when empty", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		conn := &Connection{
			ConnectionDB: db,
			Connected:    true,
			Logger:       testLogger(),
		}

		ds := &ExternalDataSource{connection: conn}

		colRows := sqlmock.NewRows([]string{"column_name", "data_type", "is_nullable"}).
			AddRow("id", "integer", false)
		mock.ExpectQuery("SELECT column_name, data_type").WillReturnRows(colRows)

		primaryKeys := map[string]map[string]bool{}

		schema, err := ds.buildTableSchema(ctx, "users", primaryKeys, testLogger(), []string{})

		require.NoError(t, err)
		assert.Equal(t, "users", schema.TableName)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDefaultSchema(t *testing.T) {
	assert.Equal(t, "public", DefaultSchema)
}

func TestRepositoryInterface(t *testing.T) {
	// Ensure ExternalDataSource implements Repository interface
	var _ Repository = (*ExternalDataSource)(nil)
}
