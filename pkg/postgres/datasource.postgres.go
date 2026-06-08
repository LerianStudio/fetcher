package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/fetcher/pkg/schemautil"
	"github.com/lib/pq"

	libLog "github.com/LerianStudio/lib-observability/log"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

// DefaultSchema is the default PostgreSQL schema name
const DefaultSchema = "public"

// Datasource defines an interface for querying data from a PostgreSQL database.
//
// It provides methods for executing queries with simple or advanced filters,
// retrieving database schema information, and managing the database connection.
//
// Implementations must be safe for concurrent use by multiple goroutines.
//
// Example usage:
//
//	ds, err := NewDataSourceRepository(connection)
//	if err != nil {
//	    return err
//	}
//	defer ds.CloseConnection()
//
//	schema, err := ds.GetDatabaseSchema(ctx, []string{"public"})
//	if err != nil {
//	    return err
//	}
//
//	results, err := ds.Query(ctx, schema, "users", []string{"id", "name"}, nil)
//
//go:generate mockgen --destination=datasource.postgres.mock.go --package=postgres . Datasource
type Datasource interface {
	// Query executes a SELECT query with simple equality filters.
	Query(ctx context.Context, schema []TableSchema, table string, fields []string, filter map[string][]any) ([]map[string]any, error)
	// QueryWithAdvancedFilters executes a SELECT query with complex filter conditions.
	QueryWithAdvancedFilters(ctx context.Context, schema []TableSchema, table string, fields []string, filter map[string]job.FilterCondition) ([]map[string]any, error)
	// GetDatabaseSchema retrieves table and column metadata from the database.
	GetDatabaseSchema(ctx context.Context, schemas []string) ([]TableSchema, error)
	// CloseConnection terminates the database connection and releases resources.
	CloseConnection() error
}

// TableSchema represents the structure of a database table.
//
// It contains the table name and a list of column metadata that describes
// the table's schema. The TableName may be schema-qualified for non-public
// schemas (e.g., "accounting.invoices") or simple for public schema tables
// (e.g., "transactions").
//
// Example:
//
//	schema := TableSchema{
//	    TableName: "users",
//	    Columns: []ColumnInformation{
//	        {Name: "id", DataType: "uuid", IsPrimaryKey: true},
//	        {Name: "name", DataType: "character varying", IsNullable: true},
//	    },
//	}
type TableSchema struct {
	// TableName is the name of the table, optionally schema-qualified.
	TableName string `json:"table_name"`
	// Columns contains metadata for each column in the table.
	Columns []ColumnInformation `json:"columns"`
}

// ColumnInformation contains the metadata of a database column.
//
// It provides essential column attributes needed for query building,
// field validation, and schema introspection.
//
// Example:
//
//	col := ColumnInformation{
//	    Name:         "created_at",
//	    DataType:     "timestamp with time zone",
//	    IsNullable:   false,
//	    IsPrimaryKey: false,
//	}
type ColumnInformation struct {
	// Name is the column name as defined in the database.
	Name string `json:"name"`
	// DataType is the PostgreSQL data type (e.g., "uuid", "integer", "jsonb").
	DataType string `json:"data_type"`
	// IsNullable indicates whether the column accepts NULL values.
	IsNullable bool `json:"is_nullable"`
	// IsPrimaryKey indicates whether the column is part of the primary key.
	IsPrimaryKey bool `json:"is_primary_key"`
}

// ExternalDataSource provides methods for interacting with a PostgreSQL database.
//
// It implements the [Datasource] interface and manages the underlying database
// connection. Use [NewDataSourceRepository] to create a new instance.
//
// ExternalDataSource is safe for concurrent use by multiple goroutines.
type ExternalDataSource struct {
	connection *Connection
}

// NewDataSourceRepository creates a new ExternalDataSource instance.
//
// It initializes the database connection using the provided [Connection] configuration.
// The connection is validated during creation to ensure it is functional.
//
// Parameters:
//   - pc: A configured [Connection] containing database credentials and settings.
//
// Returns:
//   - A pointer to the initialized [ExternalDataSource].
//   - An error if the connection cannot be established.
//
// Example:
//
//	conn := &postgres.Connection{
//	    Host:     "localhost",
//	    Port:     5432,
//	    User:     "admin",
//	    Password: "secret",
//	    Database: "mydb",
//	}
//	ds, err := postgres.NewDataSourceRepository(conn)
//	if err != nil {
//	    log.Log(context.Background(), libLog.LevelError, fmt.Sprintf("failed to connect: %v", err))
//	}
//	defer ds.CloseConnection()
func NewDataSourceRepository(pc *Connection) (*ExternalDataSource, error) {
	c := &ExternalDataSource{
		connection: pc,
	}

	_, err := c.connection.GetDB()
	if err != nil {
		pc.Logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Failed to establish PostgreSQL connection: %v", err))
		return nil, fmt.Errorf("failed to establish PostgreSQL connection: %w", err)
	}

	return c, nil
}

// CloseConnection terminates the PostgreSQL database connection.
//
// It releases all resources associated with the connection and marks it as
// disconnected. After calling CloseConnection, the ExternalDataSource instance
// should not be used for further queries.
//
// Returns an error if closing the connection fails; otherwise, returns nil.
//
// Example:
//
//	ds, err := postgres.NewDataSourceRepository(conn)
//	if err != nil {
//	    return err
//	}
//	defer func() {
//	    if err := ds.CloseConnection(); err != nil {
//	        log.Printf("failed to close connection: %v", err)
//	    }
//	}()
func (ds *ExternalDataSource) CloseConnection() error {
	if ds.connection.ConnectionDB != nil {
		ds.connection.Logger.Log(context.Background(), libLog.LevelInfo, "Closing connection to PostgreSQL...")

		err := ds.connection.ConnectionDB.Close()
		if err != nil {
			ds.connection.Logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Error closing PostgreSQL connection: %v", err))
			return err
		}

		ds.connection.Connected = false
		ds.connection.ConnectionDB = nil
		ds.connection.Logger.Log(context.Background(), libLog.LevelInfo, "PostgreSQL connection closed successfully.")
	}

	return nil
}

// Query executes a SELECT SQL query on the specified table.
//
// It validates the table and fields against the provided schema, builds a query
// with the specified filter criteria, and returns the results as a slice of maps
// where each map represents a row with column names as keys.
//
// Parameters:
//   - ctx: Context for cancellation, tracing, and logging.
//   - schema: Database schema metadata for validation.
//   - table: Name of the table to query (can be schema-qualified, e.g., "accounting.invoices").
//   - fields: Column names to select, or ["*"] for all columns.
//   - filter: Simple equality filters as field-name to values map.
//
// Returns:
//   - A slice of maps, each representing a row with column-to-value mappings.
//   - An error if validation fails, query times out, or execution fails.
//
// Example:
//
//	// Query with specific fields and filter
//	:= map[string][]any{
//	    "status": {"active"},
//	    "role":   {"admin", "user"},
//	}
//	results, err := ds.Query(ctx, schema, "users", []string{"id", "name", "email"}, filter)
//	if err != nil {
//	    return err
//	}
//	for _, row := range results {
//	    fmt.Printf("User: %s\n", row["name"])
//	}
func (ds *ExternalDataSource) Query(ctx context.Context, schema []TableSchema, table string, fields []string, filter map[string][]any) ([]map[string]any, error) {
	logger, tracer, reqId, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "postgres.data_source.query")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqId),
	)

	span.SetAttributes(
		attribute.String("app.query.table", table),
		attribute.Int("app.query.fields_count", len(fields)),
		attribute.Int("app.query.filter_count", len(filter)),
	)

	logger.Log(ctx, libLog.LevelInfo, "querying table",
		libLog.String("table", table),
		libLog.Int("fields_count", len(fields)),
	)

	// Validate requested table and fields
	queriedFields, err := ds.ValidateTableAndFields(ctx, table, fields, schema)
	if err != nil {
		return nil, err
	}

	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	queryBuilder := psql.Select(queriedFields...).From(table)

	// Apply filters, but only if they correspond to valid columns
	queryBuilder = buildDynamicFilters(queryBuilder, schema, table, filter)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("error generating SQL: %w", err)
	}

	logger.Log(ctx, libLog.LevelDebug, "executing SQL query",
		libLog.String("table", table),
		libLog.Int("args_count", len(args)),
	)

	// Create timeout context for query execution
	queryCtx, cancel := context.WithTimeout(ctx, constant.QueryTimeoutMedium)
	defer cancel()

	rows, err := ds.connection.ConnectionDB.QueryContext(queryCtx, query, args...)
	if err != nil {
		if queryCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("query execution timeout after %v: %w", constant.QueryTimeoutMedium, err)
		}

		return nil, fmt.Errorf("error executing query: %w", err)
	}
	defer rows.Close()

	return scanRows(ctx, rows, logger)
}

// GetDatabaseSchema retrieves all tables and their column details from the database.
//
// It queries the information_schema to discover tables within the specified
// PostgreSQL schemas and returns complete metadata including column types,
// nullability, and primary key information.
//
// Parameters:
//   - ctx: Context for cancellation, tracing, and logging.
//   - schemas: List of PostgreSQL schema names to query (e.g., ["public", "accounting"]).
//     If empty, defaults to ["public"].
//
// Returns:
//   - A slice of [TableSchema] objects containing table and column metadata.
//   - An error if the schema discovery times out or fails.
//
// Example:
//
//	// Discover schema for multiple PostgreSQL schemas
//	:= []string{"public", "accounting"}
//	tableSchemas, err := ds.GetDatabaseSchema(ctx, schemas)
//	if err != nil {
//	    return err
//	}
//	for _, ts := range tableSchemas {
//	    fmt.Printf("Table: %s, Columns: %d\n", ts.TableName, len(ts.Columns))
//	}
func (ds *ExternalDataSource) GetDatabaseSchema(ctx context.Context, schemas []string) ([]TableSchema, error) {
	logger, tracer, reqId, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "postgres.data_source.get_database_schema")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqId),
	)

	logger.Log(ctx, libLog.LevelInfo, "Retrieving database schema information")

	schemaCtx, cancel := context.WithTimeout(ctx, constant.SchemaDiscoveryTimeout)
	defer cancel()

	tables, err := ds.queryTables(schemaCtx, schemas)
	if err != nil {
		return nil, err
	}

	primaryKeys, err := ds.queryPrimaryKeys(schemaCtx, schemas)
	if err != nil {
		return nil, err
	}

	schema, err := ds.buildSchema(schemaCtx, tables, primaryKeys, logger)
	if err != nil {
		return nil, err
	}

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Retrieved schema for %d tables", len(schema)))

	return schema, nil
}

// queryTables retrieves all table names from the specified database schemas.
//
// It queries the information_schema.tables to discover all BASE TABLEs within
// the given schemas. For non-public schemas, table names are returned as
// schema-qualified (e.g., "accounting.invoices"). For the public schema,
// simple names are returned (e.g., "transactions").
//
// Parameters:
//   - ctx: Context for cancellation and timeout.
//   - schemas: List of PostgreSQL schema names to query.
//
// Returns:
//   - A slice of table names, optionally schema-qualified.
//   - An error if the query fails.
//
// Example:
//
//	tables, err := ds.queryTables(ctx, []string{"public", "accounting"})
//	// tables = ["users", "transactions", "accounting.invoices", "accounting.payments"]
func (ds *ExternalDataSource) queryTables(ctx context.Context, schemas []string) ([]string, error) {
	if len(schemas) == 0 {
		schemas = []string{DefaultSchema}
	}

	tableQuery := `
		SELECT table_schema, table_name
		FROM information_schema.tables
		WHERE table_schema = ANY($1)
		  AND table_type = 'BASE TABLE'
		ORDER BY table_schema, table_name
	`

	rows, err := ds.connection.ConnectionDB.QueryContext(ctx, tableQuery, pq.Array(schemas))
	if err != nil {
		return nil, fmt.Errorf("error querying tables: %w", err)
	}
	defer rows.Close()

	var tables []string

	for rows.Next() {
		var schemaName, tableName string
		if err := rows.Scan(&schemaName, &tableName); err != nil {
			return nil, fmt.Errorf("error scanning table name: %w", err)
		}

		// Use schema-qualified name for non-public schemas
		qualifiedName := tableName
		if schemaName != DefaultSchema {
			qualifiedName = schemaName + "." + tableName
		}

		tables = append(tables, qualifiedName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return tables, nil
}

// queryPrimaryKeys retrieves primary key information for all tables in the given schemas.
//
// It queries the information_schema.table_constraints and key_column_usage tables
// to identify which columns are part of primary keys. The result is a nested map
// where the outer key is the table name and the inner map contains column names
// that are part of the primary key.
//
// Parameters:
//   - ctx: Context for cancellation and timeout.
//   - schemas: List of PostgreSQL schema names to query.
//
// Returns:
//   - A map of table names to column-name sets (map[tableName]map[columnName]bool).
//   - An error if the query fails or times out.
//
// Example:
//
//	primaryKeys, err := ds.queryPrimaryKeys(ctx, []string{"public"})
//	// primaryKeys["users"]["id"] = true
//	// primaryKeys["order_items"]["order_id"] = true
//	// primaryKeys["order_items"]["product_id"] = true
func (ds *ExternalDataSource) queryPrimaryKeys(ctx context.Context, schemas []string) (map[string]map[string]bool, error) {
	if len(schemas) == 0 {
		schemas = []string{DefaultSchema}
	}

	pkQuery := `
		SELECT tc.table_schema, tc.table_name, kc.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kc
			ON kc.table_name = tc.table_name
			AND kc.table_schema = tc.table_schema
			AND kc.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'PRIMARY KEY'
		  AND tc.table_schema = ANY($1)
	`

	pkRows, err := ds.connection.ConnectionDB.QueryContext(ctx, pkQuery, pq.Array(schemas))
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf(
				"schema discovery timeout after %v while querying primary keys: %w",
				constant.SchemaDiscoveryTimeout, err,
			)
		}

		return nil, fmt.Errorf("error querying primary keys: %w", err)
	}
	defer pkRows.Close()

	primaryKeys := make(map[string]map[string]bool)

	for pkRows.Next() {
		var schemaName, tableName, columnName string
		if err := pkRows.Scan(&schemaName, &tableName, &columnName); err != nil {
			return nil, fmt.Errorf("error scanning primary key info: %w", err)
		}

		// Use schema-qualified name for non-public schemas
		qualifiedName := tableName
		if schemaName != DefaultSchema {
			qualifiedName = schemaName + "." + tableName
		}

		if primaryKeys[qualifiedName] == nil {
			primaryKeys[qualifiedName] = make(map[string]bool)
		}

		primaryKeys[qualifiedName][columnName] = true
	}

	if err := pkRows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return primaryKeys, nil
}

// buildSchema builds the complete schema information for all discovered tables.
//
// It iterates through the provided table names and creates a [TableSchema]
// for each table by querying column information from the database.
//
// Parameters:
//   - ctx: Context for cancellation and timeout.
//   - tables: List of table names to build schema for.
//   - primaryKeys: Map of table names to their primary key columns.
//   - logger: Logger for diagnostic output.
//
// Returns:
//   - A slice of [TableSchema] containing complete schema information.
//   - An error if any table schema cannot be built.
//
// Example:
//
//	tables := []string{"users", "accounting.invoices"}
//	primaryKeys := map[string]map[string]bool{"users": {"id": true}}
//	schema, err := ds.buildSchema(ctx, tables, primaryKeys, logger)
func (ds *ExternalDataSource) buildSchema(ctx context.Context, tables []string, primaryKeys map[string]map[string]bool, logger libLog.Logger) ([]TableSchema, error) {
	schema := make([]TableSchema, 0, len(tables))

	for _, tableName := range tables {
		tableSchema, err := ds.buildTableSchema(ctx, tableName, primaryKeys, logger)
		if err != nil {
			return nil, err
		}

		schema = append(schema, tableSchema)
	}

	return schema, nil
}

// buildTableSchema builds schema information for a single table.
//
// It queries the information_schema.columns table to retrieve column metadata
// for the specified table and merges in primary key information.
//
// Parameters:
//   - ctx: Context for cancellation and timeout.
//   - tableName: Table name (can be schema-qualified, e.g., "accounting.invoices").
//   - primaryKeys: Map of table names to their primary key columns.
//   - logger: Logger for diagnostic output.
//
// Returns:
//   - A [TableSchema] containing the table name and column information.
//   - An error if the column query fails.
//
// Example:
//
//	primaryKeys := map[string]map[string]bool{"users": {"id": true}}
//	schema, err := ds.buildTableSchema(ctx, "users", primaryKeys, logger)
//	// schema.TableName = "users"
//	// schema.Columns[0] = {Name: "id", DataType: "uuid", IsPrimaryKey: true}
func (ds *ExternalDataSource) buildTableSchema(ctx context.Context, tableName string, primaryKeys map[string]map[string]bool, logger libLog.Logger) (TableSchema, error) {
	// Parse the qualified table name to get schema and table parts
	schemaName, simpleTableName, err := parseQualifiedTableName(tableName)
	if err != nil {
		return TableSchema{}, fmt.Errorf("error parsing table name %q: %w", tableName, err)
	}

	columnQuery := `
		SELECT column_name, data_type,
		       CASE WHEN is_nullable = 'YES' THEN true ELSE false END as is_nullable
		FROM information_schema.columns
		WHERE table_schema = $1
		AND table_name = $2
		ORDER BY ordinal_position
	`

	colRows, err := ds.connection.ConnectionDB.QueryContext(ctx, columnQuery, schemaName, simpleTableName)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return TableSchema{}, fmt.Errorf("schema discovery timeout after %v while querying columns for table %s: %w", constant.SchemaDiscoveryTimeout, tableName, err)
		}

		return TableSchema{}, fmt.Errorf("error querying columns for table %s: %w", tableName, err)
	}
	defer colRows.Close()

	columns, err := ds.scanColumns(ctx, colRows, tableName, primaryKeys, logger)
	if err != nil {
		return TableSchema{}, err
	}

	return TableSchema{
		TableName: tableName,
		Columns:   columns,
	}, nil
}

// parseQualifiedTableName splits a qualified table name into schema and table parts.
//
// For schema-qualified names, it returns the schema and table components.
// For simple names without a schema prefix, it returns [DefaultSchema] ("public")
// as the schema.
//
// Parameters:
//   - qualifiedName: The table name, optionally prefixed with schema.
//
// Returns:
//   - schema: The PostgreSQL schema name.
//   - table: The table name without schema prefix.
//
// Example:
//
//	schema, table, err := parseQualifiedTableName("accounting.invoices")
//	// schema = "accounting", table = "invoices", err = nil
//
//	schema, table, err := parseQualifiedTableName("transactions")
//	// schema = "public", table = "transactions", err = nil
func parseQualifiedTableName(qualifiedName string) (schema, table string, err error) {
	return schemautil.ParseQualifiedTableName(qualifiedName, DefaultSchema)
}

// normalizeTableNameForLookup normalizes a table name for schema lookup.
//
// The schema object stores tables using different naming conventions:
//   - Non-public schemas: "accounting.invoices" (schema-qualified)
//   - Public schema: "transactions" (simple name, no prefix)
//
// This function converts input table names to match the storage format:
//   - "accounting.invoices" → "accounting.invoices" (non-public, keep as-is)
//   - "public.transactions" → "transactions" (public schema, strip prefix)
//   - "transactions" → "transactions" (no prefix, keep as-is)
//
// Example:
//
//	normalized := normalizeTableNameForLookup("public.users")
//	// normalized = "users"
//
//	normalized := normalizeTableNameForLookup("accounting.invoices")
//	// normalized = "accounting.invoices"
func normalizeTableNameForLookup(tableName string) string {
	return schemautil.NormalizeTableNameForLookup(tableName, DefaultSchema)
}

// scanColumns scans column information from SQL query results.
//
// It iterates through the result set, extracts column metadata, and marks
// columns that are part of the table's primary key.
//
// Parameters:
//   - colRows: SQL result set containing column_name, data_type, and is_nullable.
//   - tableName: Name of the table being processed (for primary key lookup).
//   - primaryKeys: Map of table names to their primary key columns.
//   - logger: Logger for diagnostic output.
//
// Returns:
//   - A slice of [ColumnInformation] with complete column metadata.
//   - An error if row scanning fails.
//
// Example:
//
//	rows, _ := db.QueryContext(ctx, columnQuery, schemaName, tableName)
//	defer rows.Close()
//	columns, err := ds.scanColumns(rows, "users", primaryKeys, logger)
func (ds *ExternalDataSource) scanColumns(ctx context.Context, colRows *sql.Rows, tableName string, primaryKeys map[string]map[string]bool, logger libLog.Logger) ([]ColumnInformation, error) {
	var columns []ColumnInformation

	for colRows.Next() {
		var col ColumnInformation
		if err := colRows.Scan(&col.Name, &col.DataType, &col.IsNullable); err != nil {
			if closeErr := colRows.Close(); closeErr != nil {
				logger.Log(ctx, libLog.LevelWarn, fmt.Sprintf("error closing rows after scan error: %v", closeErr))
			}

			return nil, fmt.Errorf("error scanning column info: %w", err)
		}

		if pkCols, exists := primaryKeys[tableName]; exists {
			col.IsPrimaryKey = pkCols[col.Name]
		}

		columns = append(columns, col)
	}

	if err := colRows.Close(); err != nil {
		logger.Log(ctx, libLog.LevelWarn, fmt.Sprintf("error closing column rows: %v", err))
	}

	return columns, nil
}

// scanRows processes SQL query result rows into a slice of maps.
//
// Each row is converted to a map where column names are keys and the
// corresponding row values are the map values. JSONB and JSON fields
// are automatically parsed using [parseJSONBField].
//
// Parameters:
//   - rows: SQL result set to process.
//   - logger: Logger for diagnostic output.
//
// Returns:
//   - A slice of maps, each representing a row with column-to-value mappings.
//   - An error if row scanning fails.
//
// Example:
//
//	rows, err := db.QueryContext(ctx, "SELECT id, name FROM users")
//	if err != nil {
//	    return err
//	}
//	defer rows.Close()
//	results, err := scanRows(rows, logger)
//	// results[0] = map[string]any{"id": "uuid-...", "name": "John"}
func scanRows(ctx context.Context, rows *sql.Rows, logger libLog.Logger) ([]map[string]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("error getting column names: %w", err)
	}

	values := make([]any, len(columns))
	pointers := make([]any, len(columns))

	for i := range values {
		pointers[i] = &values[i]
	}

	var result []map[string]any

	for rows.Next() {
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}

		rowMap := createRowMap(ctx, columns, values, logger)
		result = append(result, rowMap)
	}

	return result, nil
}

// createRowMap maps column names to their respective scanned values.
//
// It creates a single row map from parallel slices of column names and values.
// Each value is processed through [parseJSONBField] to handle JSONB/JSON types.
//
// Parameters:
//   - columns: Slice of column names.
//   - values: Slice of values corresponding to each column.
//   - logger: Logger for diagnostic output.
//
// Returns a map with column names as keys and processed values.
//
// Example:
//
//	columns := []string{"id", "name", "metadata"}
//	values := []any{"123", "John", []byte(`{"role": "admin"}`)}
//	rowMap := createRowMap(columns, values, logger)
//	// rowMap = {"id": "123", "name": "John", "metadata": map[string]any{"role": "admin"}}
func createRowMap(ctx context.Context, columns []string, values []any, logger libLog.Logger) map[string]any {
	rowMap := make(map[string]any)

	for i, column := range columns {
		// Attempt to parse any value that could be JSONB
		rowMap[column] = parseJSONBField(ctx, values[i], logger)
	}

	return rowMap
}

// parseJSONBField attempts to unmarshal a value that might be a JSONB or JSON type.
//
// PostgreSQL returns JSONB and JSON fields as []uint8. This function attempts
// to deserialize such values into native Go types (map, slice, or string).
// If deserialization fails, the original value is returned unchanged.
//
// Parameters:
//   - value: The value to process (may be []uint8 for JSONB/JSON, or any other type).
//   - logger: Logger for warning messages on parse failures.
//
// Returns the parsed value (map, slice, or string) or the original value.
//
// Example:
//
//	// JSONB object
//	result := parseJSONBField([]byte(`{"key": "value"}`), logger)
//	// result = map[string]any{"key": "value"}
//
//	// JSONB array
//	result := parseJSONBField([]byte(`[1, 2, 3]`), logger)
//	// result = []any{1, 2, 3}
func parseJSONBField(ctx context.Context, value any, logger libLog.Logger) any {
	if value == nil {
		return nil
	}

	// Check if the value is []uint8, which is how the PostgreSQL driver
	// represents JSONB and JSON fields
	if byteData, ok := value.([]uint8); ok {
		// Try to deserialize as a generic map[string]any
		var jsonMap map[string]any
		if err := json.Unmarshal(byteData, &jsonMap); err == nil {
			return jsonMap
		}

		// If object parsing fails, try as array
		var jsonArray []any
		if err := json.Unmarshal(byteData, &jsonArray); err == nil {
			return jsonArray
		}

		// Try as string in case it's a JSON string format
		var jsonString string
		if err := json.Unmarshal(byteData, &jsonString); err == nil {
			return jsonString
		}

		// If all attempts fail, log a warning and return the original value
		logger.Log(ctx, libLog.LevelWarn, fmt.Sprintf("Failed to unmarshal potential JSONB data for value: %v", string(byteData)))
	}

	return value
}

// ValidateTableAndFields checks if the specified table exists and validates field names.
//
// It verifies that the table exists in the schema and all requested fields
// correspond to actual columns. This prevents SQL errors from invalid table
// or column references.
//
// Parameters:
//   - ctx: Context for tracing and cancellation.
//   - tableName: Name of the table to validate (supports schema-qualified names).
//   - requestedFields: List of column names to validate, or ["*"] for all columns.
//   - schema: Database schema metadata containing table and column information.
//
// Returns:
//   - A slice of validated field names (expands "*" to all column names).
//   - An error if the table doesn't exist or any field is invalid.
//
// Example:
//
//	// Validate specific fields
//	fields, err := ds.ValidateTableAndFields(ctx, "users", []string{"id", "name"}, schema)
//	// fields = ["id", "name"]
//
//	// Request all fields
//	fields, err := ds.ValidateTableAndFields(ctx, "users", []string{"*"}, schema)
//	// fields = ["id", "name", "email", "created_at", ...]
func (ds *ExternalDataSource) ValidateTableAndFields(ctx context.Context, tableName string, requestedFields []string, schema []TableSchema) ([]string, error) {
	logger, tracer, reqId, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "postgres.data_source.validate_table_and_fields")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqId),
	)

	// Normalize table name for lookup: schema-qualified names should match
	// the schema object which stores tables as "accounting.invoices" for non-public
	// schemas and "transactions" for public schema tables.
	lookupName := normalizeTableNameForLookup(tableName)

	span.SetAttributes(
		attribute.String("app.validate.table", tableName),
		attribute.Int("app.validate.fields_count", len(requestedFields)),
		attribute.Int("app.validate.schema_tables", len(schema)),
	)

	logger.Log(ctx, libLog.LevelInfo, "validating table and fields",
		libLog.String("table", tableName),
		libLog.Int("fields_count", len(requestedFields)),
	)

	// Check if table exists
	var tableFound bool

	var tableColumns []ColumnInformation

	for _, table := range schema {
		if table.TableName == lookupName {
			tableFound = true
			tableColumns = table.Columns

			break
		}
	}

	if !tableFound {
		return nil, fmt.Errorf("table '%s' does not exist in the database", tableName)
	}

	// Create a map of valid column names for efficient lookup
	validColumns := make(map[string]bool)
	for _, col := range tableColumns {
		validColumns[col.Name] = true
	}

	// Special case: if "*" is in the fields, return all columns
	if len(requestedFields) == 1 && requestedFields[0] == "*" {
		allFields := make([]string, len(tableColumns))
		for i, col := range tableColumns {
			allFields[i] = col.Name
		}

		return allFields, nil
	}

	// Validate each requested field
	var validFields []string

	var invalidFields []string

	for _, field := range requestedFields {
		if validColumns[field] {
			validFields = append(validFields, field)
		} else {
			invalidFields = append(invalidFields, field)
		}
	}

	if len(invalidFields) > 0 {
		return nil, fmt.Errorf("invalid fields for table '%s': %v", tableName, invalidFields)
	}

	if len(validFields) == 0 {
		return nil, fmt.Errorf("no valid fields specified for table '%s'", tableName)
	}

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Successfully validated table '%s' and fields %v", tableName, validFields))

	return validFields, nil
}

// buildDynamicFilters applies simple equality filters to a SQL query builder.
//
// It iterates through the filter map and adds WHERE clauses for columns that
// exist in the table schema. Invalid column names are silently ignored to
// prevent SQL injection or errors from typos.
//
// Parameters:
//   - queryBuilder: Squirrel SELECT builder to add filters to.
//   - schema: Database schema for column validation.
//   - table: Table name for column lookup.
//   - filter: Map of field names to values (equality conditions).
//
// Returns the modified query builder with filter conditions applied.
//
// Example:
//
//	filter := map[string][]any{
//	    "status": {"active"},
//	    "role":   {"admin", "user"},
//	}
//	queryBuilder = buildDynamicFilters(queryBuilder, schema, "users", filter)
//	// Generates: WHERE status = $1 AND role IN ($2, $3)
func buildDynamicFilters(queryBuilder squirrel.SelectBuilder, schema []TableSchema, table string, filter map[string][]any) squirrel.SelectBuilder {
	// Find the table's column information
	var tableColumns []ColumnInformation

	// Normalize table name for schema lookup
	lookupName := normalizeTableNameForLookup(table)

	for _, t := range schema {
		if t.TableName == lookupName {
			tableColumns = t.Columns
			break
		}
	}

	// Create a map of valid column names for efficient lookup
	validColumns := make(map[string]bool)
	for _, col := range tableColumns {
		validColumns[col.Name] = true
	}

	for field, values := range filter {
		// Only apply filters for valid columns
		if validColumns[field] && len(values) > 0 {
			queryBuilder = applyFilter(queryBuilder, field, values)
		}
	}

	return queryBuilder
}

// applyFilter adds a WHERE condition for a field with one or more values.
//
// For a single value, it generates an equality condition (field = value).
// For multiple values, it generates an IN condition (field IN (values...)).
//
// Parameters:
//   - queryBuilder: Squirrel SELECT builder to add the condition to.
//   - fieldName: Column name to filter on.
//   - values: One or more values to match against.
//
// Returns the modified query builder.
//
// Example:
//
//	// Single value - generates: status = $1
//	qb = applyFilter(qb, "status", []any{"active"})
//
//	// Multiple values - generates: role IN ($1, $2)
//	qb = applyFilter(qb, "role", []any{"admin", "user"})
func applyFilter(queryBuilder squirrel.SelectBuilder, fieldName string, values []any) squirrel.SelectBuilder {
	if len(values) == 0 {
		return queryBuilder
	}

	// Use squirrel.Eq which respects PlaceholderFormat
	// For single value, use direct equality; for multiple values, use IN
	if len(values) == 1 {
		return queryBuilder.Where(squirrel.Eq{fieldName: values[0]})
	}

	return queryBuilder.Where(squirrel.Eq{fieldName: values})
}

// QueryWithAdvancedFilters executes a SELECT query with complex filter conditions.
//
// Unlike [Query] which only supports simple equality filters, this method
// supports advanced operators including: equals, not equals, greater than,
// less than, between, in, not in, and like patterns.
//
// Parameters:
//   - ctx: Context for cancellation, tracing, and logging.
//   - schema: Database schema metadata for validation.
//   - table: Name of the table to query.
//   - fields: Column names to select, or ["*"] for all columns.
//   - filter: Map of field names to [job.FilterCondition] with advanced operators.
//
// Returns:
//   - A slice of maps representing query results.
//   - An error if validation fails, filters are invalid, or query fails.
//
// Example:
//
//	filter := map[string]job.FilterCondition{
//	    "created_at": {Between: []any{"2024-01-01", "2024-12-31"}},
//	    "status":     {In: []any{"active", "pending"}},
//	    "amount":     {GreaterThan: []any{100.00}},
//	}
//	results, err := ds.QueryWithAdvancedFilters(ctx, schema, "orders", []string{"*"}, filter)
func (ds *ExternalDataSource) QueryWithAdvancedFilters(ctx context.Context, schema []TableSchema, table string, fields []string, filter map[string]job.FilterCondition) ([]map[string]any, error) {
	logger, tracer, reqId, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "postgres.data_source.query_with_advanced_filters")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqId),
	)

	span.SetAttributes(
		attribute.String("app.query.table", table),
		attribute.Int("app.query.fields_count", len(fields)),
		attribute.Int("app.query.filter_count", len(filter)),
	)

	logger.Log(ctx, libLog.LevelInfo, "querying table with advanced filters",
		libLog.String("table", table),
		libLog.Int("fields_count", len(fields)),
	)

	// Validate requested table and fields
	queriedFields, err := ds.ValidateTableAndFields(ctx, table, fields, schema)
	if err != nil {
		return nil, err
	}

	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	queryBuilder := psql.Select(queriedFields...).From(table)

	// Apply advanced filters
	queryBuilder, err = ds.buildAdvancedFilters(queryBuilder, schema, table, filter)
	if err != nil {
		return nil, fmt.Errorf("error building advanced filters: %w", err)
	}

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("error generating SQL: %w", err)
	}

	logger.Log(ctx, libLog.LevelDebug, "executing advanced filter SQL query",
		libLog.String("table", table),
		libLog.Int("args_count", len(args)),
	)

	// Create timeout context for query execution (slower timeout for advanced filters)
	queryCtx, cancel := context.WithTimeout(ctx, constant.QueryTimeoutSlow)
	defer cancel()

	rows, err := ds.connection.ConnectionDB.QueryContext(queryCtx, query, args...)
	if err != nil {
		if queryCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("advanced filter query timeout after %v: %w", constant.QueryTimeoutSlow, err)
		}

		return nil, fmt.Errorf("error executing query: %w", err)
	}
	defer rows.Close()

	return scanRows(ctx, rows, logger)
}

// buildAdvancedFilters applies [job.FilterCondition] criteria to the query builder.
//
// It validates each filter condition, ensures columns exist in the schema,
// and applies the appropriate SQL operators for each filter type.
//
// Parameters:
//   - queryBuilder: Squirrel SELECT builder to add conditions to.
//   - schema: Database schema for column validation.
//   - table: Table name for column lookup.
//   - filter: Map of field names to filter conditions.
//
// Returns:
//   - The modified query builder.
//   - An error if any filter condition is invalid.
//
// Example:
//
//	filter := map[string]job.FilterCondition{
//	    "status": {Equals: []any{"active"}},
//	    "amount": {GreaterThan: []any{100}},
//	}
//	qb, err := ds.buildAdvancedFilters(qb, schema, "orders", filter)
func (ds *ExternalDataSource) buildAdvancedFilters(queryBuilder squirrel.SelectBuilder, schema []TableSchema, table string, filter map[string]job.FilterCondition) (squirrel.SelectBuilder, error) {
	var tableColumns []ColumnInformation

	// Normalize table name for schema lookup
	lookupName := normalizeTableNameForLookup(table)

	for _, t := range schema {
		if t.TableName == lookupName {
			tableColumns = t.Columns
			break
		}
	}

	// Create a map of valid column names for efficient lookup
	validColumns := make(map[string]bool)
	for _, col := range tableColumns {
		validColumns[col.Name] = true
	}

	for field, condition := range filter {
		// Only apply filters for valid columns
		if !validColumns[field] {
			continue
		}

		if isFilterConditionEmpty(condition) {
			continue
		}

		// Validate the condition
		if err := validateFilterCondition(field, condition); err != nil {
			return queryBuilder, err
		}

		// Apply each filter operator
		queryBuilder = ds.applyAdvancedFilter(queryBuilder, field, condition)
	}

	return queryBuilder, nil
}

// applyAdvancedFilter applies all operators from a [job.FilterCondition] to the query.
//
// It delegates to specialized functions for each operator type: equals, comparison
// operators (gt, gte, lt, lte), between, in/not-in, not-equals, and like patterns.
// All applicable operators are combined with AND logic.
//
// Parameters:
//   - queryBuilder: Squirrel SELECT builder to add conditions to.
//   - field: Column name to apply filters on.
//   - condition: Filter condition containing one or more operators.
//
// Returns the modified query builder.
//
// Example:
//
//	condition := job.FilterCondition{
//	    GreaterThan: []any{100},
//	    LessThan:    []any{1000},
//	}
//	qb = ds.applyAdvancedFilter(qb, "amount", condition)
//	// Generates: amount > $1 AND amount < $2
func (ds *ExternalDataSource) applyAdvancedFilter(queryBuilder squirrel.SelectBuilder, field string, condition job.FilterCondition) squirrel.SelectBuilder {
	queryBuilder = applyEqualsFilter(queryBuilder, field, condition)
	queryBuilder = applyComparisonFilters(queryBuilder, field, condition)
	queryBuilder = applyBetweenFilter(queryBuilder, field, condition)
	queryBuilder = applyInFilters(queryBuilder, field, condition)
	queryBuilder = applyNotEqualsFilter(queryBuilder, field, condition)
	queryBuilder = applyLikeFilter(queryBuilder, field, condition)

	return queryBuilder
}

// applyEqualsFilter applies equals (=) or IN operator to the query builder.
//
// For a single value, it generates an equality condition (field = value).
// For multiple values, it generates an IN condition (field IN (values...)).
//
// Example:
//
//	condition := job.FilterCondition{Equals: []any{"active"}}
//	qb = applyEqualsFilter(qb, "status", condition)
//	// Generates: status = $1
func applyEqualsFilter(queryBuilder squirrel.SelectBuilder, field string, condition job.FilterCondition) squirrel.SelectBuilder {
	if len(condition.Equals) == 0 {
		return queryBuilder
	}

	if len(condition.Equals) == 1 {
		return queryBuilder.Where(squirrel.Eq{field: condition.Equals[0]})
	}

	return queryBuilder.Where(squirrel.Eq{field: condition.Equals})
}

// applyComparisonFilters applies numeric comparison operators to the query builder.
//
// Supports: GreaterThan (>), GreaterOrEqual (>=), LessThan (<), LessOrEqual (<=).
// Each operator uses only the first value in its slice.
//
// Example:
//
//	condition := job.FilterCondition{
//	    GreaterThan:    []any{100},
//	    LessOrEqual:    []any{1000},
//	}
//	qb = applyComparisonFilters(qb, "amount", condition)
//	// Generates: amount > $1 AND amount <= $2
func applyComparisonFilters(queryBuilder squirrel.SelectBuilder, field string, condition job.FilterCondition) squirrel.SelectBuilder {
	if len(condition.GreaterThan) > 0 {
		queryBuilder = queryBuilder.Where(squirrel.Gt{field: condition.GreaterThan[0]})
	}

	if len(condition.GreaterOrEqual) > 0 {
		queryBuilder = queryBuilder.Where(squirrel.GtOrEq{field: condition.GreaterOrEqual[0]})
	}

	if len(condition.LessThan) > 0 {
		queryBuilder = queryBuilder.Where(squirrel.Lt{field: condition.LessThan[0]})
	}

	if len(condition.LessOrEqual) > 0 {
		queryBuilder = queryBuilder.Where(squirrel.LtOrEq{field: condition.LessOrEqual[0]})
	}

	return queryBuilder
}

// applyBetweenFilter applies a BETWEEN-style range filter to the query builder.
//
// Requires exactly 2 values in the Between slice: [start, end].
// For date fields, the end date is automatically adjusted to include the full day.
// Generates: field >= start AND field <= end.
//
// Example:
//
//	condition := job.FilterCondition{Between: []any{"2024-01-01", "2024-12-31"}}
//	qb = applyBetweenFilter(qb, "created_at", condition)
//	// Generates: created_at >= $1 AND created_at <= $2
func applyBetweenFilter(queryBuilder squirrel.SelectBuilder, field string, condition job.FilterCondition) squirrel.SelectBuilder {
	if len(condition.Between) != 2 {
		return queryBuilder
	}

	startValue := condition.Between[0]
	endValue := adjustEndDateIfNeeded(field, condition.Between[0], condition.Between[1])

	return queryBuilder.Where(squirrel.GtOrEq{field: startValue}).Where(squirrel.LtOrEq{field: endValue})
}

// adjustEndDateIfNeeded adjusts an end date to include the full day.
//
// When filtering date fields with a BETWEEN range, if the end date is in
// YYYY-MM-DD format, it is adjusted to "YYYY-MM-DDT23:59:59.999Z" to
// include all records from that day.
//
// Parameters:
//   - field: Column name to check for date patterns.
//   - startValue: Range start value (used for date detection).
//   - endValue: Range end value to potentially adjust.
//
// Returns the adjusted end value if applicable, otherwise the original value.
//
// Example:
//
//	endValue := adjustEndDateIfNeeded("created_at", "2024-01-01", "2024-01-31")
//	// endValue = "2024-01-31T23:59:59.999Z"
func adjustEndDateIfNeeded(field string, startValue, endValue any) any {
	if !isDateField(field) || !isDateString(startValue) || !isDateString(endValue) {
		return endValue
	}

	endStr, ok := endValue.(string)
	if !ok || len(endStr) != 10 { // Not YYYY-MM-DD format
		return endValue
	}

	return endStr + "T23:59:59.999Z"
}

// applyInFilters applies IN and NOT IN operators to the query builder.
//
// The In operator generates: field IN (values...).
// The NotIn operator generates: field NOT IN (values...).
//
// Example:
//
//	condition := job.FilterCondition{
//	    In:    []any{"active", "pending"},
//	    NotIn: []any{"archived"},
//	}
//	qb = applyInFilters(qb, "status", condition)
//	// Generates: status IN ($1, $2) AND status NOT IN ($3)
func applyInFilters(queryBuilder squirrel.SelectBuilder, field string, condition job.FilterCondition) squirrel.SelectBuilder {
	if len(condition.In) > 0 {
		queryBuilder = queryBuilder.Where(squirrel.Eq{field: condition.In})
	}

	if len(condition.NotIn) > 0 {
		queryBuilder = queryBuilder.Where(squirrel.NotEq{field: condition.NotIn})
	}

	return queryBuilder
}

// applyNotEqualsFilter applies not equals (<>) operator to the query builder.
//
// For a single value, generates: field <> value.
// For multiple values, generates multiple AND conditions: field <> v1 AND field <> v2.
//
// Example:
//
//	condition := job.FilterCondition{NotEquals: []any{"deleted", "archived"}}
//	qb = applyNotEqualsFilter(qb, "status", condition)
//	// Generates: status <> $1 AND status <> $2
func applyNotEqualsFilter(queryBuilder squirrel.SelectBuilder, field string, condition job.FilterCondition) squirrel.SelectBuilder {
	if len(condition.NotEquals) == 0 {
		return queryBuilder
	}

	if len(condition.NotEquals) == 1 {
		return queryBuilder.Where(squirrel.NotEq{field: condition.NotEquals[0]})
	}

	// Multiple values treated as AND NOT conditions
	for _, val := range condition.NotEquals {
		queryBuilder = queryBuilder.Where(squirrel.NotEq{field: val})
	}

	return queryBuilder
}

// applyLikeFilter applies SQL LIKE pattern matching to the query builder.
//
// It uses the first value in the Like slice as the pattern string.
// Supports standard SQL wildcards: % (any characters) and _ (single character).
//
// Example:
//
//	condition := job.FilterCondition{Like: []any{"%john%"}}
//	qb = applyLikeFilter(qb, "name", condition)
//	// Generates: name LIKE $1 (with value "%john%")
func applyLikeFilter(queryBuilder squirrel.SelectBuilder, field string, condition job.FilterCondition) squirrel.SelectBuilder {
	if len(condition.Like) == 0 {
		return queryBuilder
	}

	pattern, ok := condition.Like[0].(string)
	if !ok {
		return queryBuilder
	}

	return queryBuilder.Where(squirrel.Like{field: pattern})
}

// isFilterConditionEmpty checks if a [job.FilterCondition] has no active filters.
//
// Returns true if all filter operator slices (Equals, GreaterThan, Between, etc.)
// are empty, indicating the condition should be skipped.
//
// Example:
//
//	empty := isFilterConditionEmpty(job.FilterCondition{})
//	// empty = true
//
//	notEmpty := isFilterConditionEmpty(job.FilterCondition{Equals: []any{"active"}})
//	// notEmpty = false
func isFilterConditionEmpty(condition job.FilterCondition) bool {
	return len(condition.Equals) == 0 &&
		len(condition.GreaterThan) == 0 &&
		len(condition.GreaterOrEqual) == 0 &&
		len(condition.LessThan) == 0 &&
		len(condition.LessOrEqual) == 0 &&
		len(condition.Between) == 0 &&
		len(condition.In) == 0 &&
		len(condition.NotIn) == 0 &&
		len(condition.NotEquals) == 0 &&
		len(condition.Like) == 0
}

// validateFilterCondition validates that a [job.FilterCondition] has proper values.
//
// It enforces the following rules:
//   - Between operator must have exactly 2 values.
//   - Single-value operators (gt, gte, lt, lte) must have exactly 1 value.
//   - UUID fields must contain valid UUID values.
//
// Returns an error describing the validation failure, or nil if valid.
//
// Example:
//
//	// Valid condition
//	err := validateFilterCondition("amount", job.FilterCondition{GreaterThan: []any{100}})
//	// err = nil
//
//	// Invalid condition
//	err := validateFilterCondition("date", job.FilterCondition{Between: []any{"2024-01-01"}})
//	// err = "between operator for field 'date' must have exactly 2 values, got 1"
func validateFilterCondition(fieldName string, condition job.FilterCondition) error {
	// Validate between operator has exactly 2 values
	if len(condition.Between) > 0 && len(condition.Between) != 2 {
		return fmt.Errorf("between operator for field '%s' must have exactly 2 values, got %d", fieldName, len(condition.Between))
	}

	// Validate single-value operators have exactly 1 value
	singleValueOps := map[string][]any{
		"gt":  condition.GreaterThan,
		"gte": condition.GreaterOrEqual,
		"lt":  condition.LessThan,
		"lte": condition.LessOrEqual,
	}

	for opName, values := range singleValueOps {
		if len(values) > 0 && len(values) != 1 {
			return fmt.Errorf("%s operator for field '%s' must have exactly 1 value, got %d", opName, fieldName, len(values))
		}
	}

	// Validate field name patterns for common UUID fields
	if isLikelyUUIDField(fieldName) {
		if err := validateUUIDFieldValues(fieldName, condition); err != nil {
			return err
		}
	}

	return nil
}

// isLikelyUUIDField checks if a field name suggests it contains UUID values.
//
// It matches common UUID field naming patterns such as "id", "_id", "uuid",
// "user_id", "organization_id", etc. This is used to enable UUID validation.
//
// Example:
//
//	isLikelyUUIDField("user_id")       // true
//	isLikelyUUIDField("organization")  // false
//	isLikelyUUIDField("template_uuid") // true
func isLikelyUUIDField(fieldName string) bool {
	uuidPatterns := []string{"id", "_id", "uuid", "template_id", "organization_id", "user_id", "account_id"}
	fieldLower := strings.ToLower(fieldName)

	for _, pattern := range uuidPatterns {
		if strings.Contains(fieldLower, pattern) {
			return true
		}
	}

	return false
}

// validateUUIDFieldValues validates that filter values for UUID fields are valid UUIDs.
//
// It checks all filter operator values (Equals, In, Between, etc.) and returns
// an error if any string value is not a valid UUID format.
//
// Example:
//
//	// Valid UUID
//	err := validateUUIDFieldValues("user_id", job.FilterCondition{
//	    Equals: []any{"550e8400-e29b-41d4-a716-446655440000"},
//	})
//	// err = nil
//
//	// Invalid UUID
//	err := validateUUIDFieldValues("user_id", job.FilterCondition{
//	    Equals: []any{"invalid"},
//	})
//	// err = "field 'user_id' appears to be a UUID field but received non-UUID value..."
func validateUUIDFieldValues(fieldName string, condition job.FilterCondition) error {
	allValues := [][]any{
		condition.Equals,
		condition.GreaterThan,
		condition.GreaterOrEqual,
		condition.LessThan,
		condition.LessOrEqual,
		condition.Between,
		condition.In,
		condition.NotIn,
	}

	for _, values := range allValues {
		for _, value := range values {
			if str, ok := value.(string); ok {
				if !isValidUUIDFormat(str) {
					return fmt.Errorf("field '%s' appears to be a UUID field but received non-UUID value '%s'. UUID fields require valid UUID format (e.g., '550e8400-e29b-41d4-a716-446655440000') or use a date field for date filtering", fieldName, str)
				}
			}
		}
	}

	return nil
}

// isValidUUIDFormat checks if a string is a valid UUID format.
//
// It uses the google/uuid package to validate the format.
//
// Example:
//
//	isValidUUIDFormat("550e8400-e29b-41d4-a716-446655440000") // true
//	isValidUUIDFormat("invalid-uuid")                         // false
//	isValidUUIDFormat("123")                                  // false
func isValidUUIDFormat(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// isDateField checks if a field name suggests it contains date or timestamp values.
//
// It matches common date/time column naming patterns such as "created_at",
// "updated_at", "_date", "_time", etc. This is used to enable date-specific
// filter adjustments like end-of-day rounding.
//
// Example:
//
//	isDateField("created_at")   // true
//	isDateField("birth_date")   // true
//	isDateField("amount")       // false
//	isDateField("scheduled_at") // true
func isDateField(fieldName string) bool {
	datePatterns := []string{"created_at", "updated_at", "deleted_at", "completed_at", "date", "time", "_at", "_date", "_time"}
	fieldLower := strings.ToLower(fieldName)

	for _, pattern := range datePatterns {
		if strings.Contains(fieldLower, pattern) {
			return true
		}
	}

	return false
}

// isDateString checks if a value looks like a date string.
//
// It detects common date formats by checking for:
//   - Minimum length of 10 characters (YYYY-MM-DD).
//   - Contains hyphens as date separators.
//   - Either exactly 10 chars (date only) or contains "T" (ISO 8601 datetime).
//
// Example:
//
//	isDateString("2024-01-15")           // true
//	isDateString("2024-01-15T10:30:00Z") // true
//	isDateString("January 15, 2024")     // false
//	isDateString(123)                     // false
func isDateString(value any) bool {
	if str, ok := value.(string); ok {
		// Check for common date formats: YYYY-MM-DD, YYYY-MM-DDTHH:MM:SS, etc.
		return len(str) >= 10 && strings.Contains(str, "-") && (len(str) == 10 || strings.Contains(str, "T"))
	}

	return false
}
