package sqlserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model/job"
	"github.com/LerianStudio/fetcher/pkg/schemautil"

	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

// DefaultSchema is the default SQL Server schema name
const DefaultSchema = "dbo"

// Datasource defines an interface for querying data from a specified table and fields.
//
//go:generate mockgen --destination=datasource.sqlserver.mock.go --package=sqlserver . Datasource
type Datasource interface {
	Query(ctx context.Context, schema []TableSchema, table string, fields []string, filter map[string][]any) ([]map[string]any, error)
	QueryWithAdvancedFilters(ctx context.Context, schema []TableSchema, table string, fields []string, filter map[string]job.FilterCondition) ([]map[string]any, error)
	GetDatabaseSchema(ctx context.Context, schemas []string) ([]TableSchema, error)
	CloseConnection() error
}

// TableSchema represents the structure of a database table
type TableSchema struct {
	TableName string              `json:"table_name"`
	Columns   []ColumnInformation `json:"columns"`
}

// ColumnInformation contains the details of a database column
type ColumnInformation struct {
	Name         string `json:"name"`
	DataType     string `json:"data_type"`
	IsNullable   bool   `json:"is_nullable"`
	IsPrimaryKey bool   `json:"is_primary_key"`
}

// ExternalDataSource provides an interface for interacting with a SQL Server database connection.
type ExternalDataSource struct {
	connection *Connection
}

// NewDataSourceRepository creates a new ExternalDataSource instance using the provided sqlserver.Connection, initializing the database connection.
// Returns nil and error if connection fails.
func NewDataSourceRepository(pc *Connection) (*ExternalDataSource, error) {
	c := &ExternalDataSource{
		connection: pc,
	}

	_, err := c.connection.GetDB(context.Background())
	if err != nil {
		pc.Logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Failed to establish SQL Server connection: %v", err))
		return nil, fmt.Errorf("failed to establish SQL Server connection: %w", err)
	}

	return c, nil
}

// CloseConnection closing the connection with SQL Server.
func (ds *ExternalDataSource) CloseConnection() error {
	if ds.connection.ConnectionDB != nil {
		ds.connection.Logger.Log(context.Background(), libLog.LevelInfo, "Closing connection to SQL Server...")

		err := ds.connection.ConnectionDB.Close()
		if err != nil {
			ds.connection.Logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Error closing SQL Server connection: %v", err))
			return err
		}

		ds.connection.Connected = false
		ds.connection.ConnectionDB = nil
		ds.connection.Logger.Log(context.Background(), libLog.LevelInfo, "SQL Server connection closed successfully.")
	}

	return nil
}

// Query executes a SELECT SQL query on the specified table with the given fields and filter criteria.
// It returns the query results as a slice of maps or an error in case of failure.
func (ds *ExternalDataSource) Query(ctx context.Context, schema []TableSchema, table string, fields []string, filter map[string][]any) ([]map[string]any, error) {
	logger, tracer, reqId, _ := libCommons.NewTrackingFromContext(ctx)

	_, span := tracer.Start(ctx, "sqlserver.data_source.query")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqId),
	)

	err := libOpentelemetry.SetSpanAttributesFromValue(span, "app.request.repository_filter", map[string]any{
		"table":  table,
		"fields": fields,
		"filter": filter,
	}, nil)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert repository filter to JSON string", err)
	}

	logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Querying %s table with fields %v", table, fields))

	// Validate requested table and fields
	queriedFields, err := ds.ValidateTableAndFields(ctx, table, fields, schema)
	if err != nil {
		return nil, err
	}

	// SQL Server uses @p1, @p2, etc. for placeholders (not ?)
	sqlServer := squirrel.StatementBuilder.PlaceholderFormat(sqlServerPlaceholderFormat)
	queryBuilder := sqlServer.Select(queriedFields...).From(table)

	// Apply filters, but only if they correspond to valid columns
	queryBuilder = buildDynamicFilters(queryBuilder, schema, table, filter)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("error generating SQL: %w", err)
	}

	logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Executing SQL: %s with args: %v", query, args))

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

	return scanRows(rows, logger)
}

// GetDatabaseSchema retrieves all tables and their column details from the database
// It returns a slice of TableSchema objects or an error if the operation fails
func (ds *ExternalDataSource) GetDatabaseSchema(ctx context.Context, schemas []string) ([]TableSchema, error) {
	logger, tracer, reqId, _ := libCommons.NewTrackingFromContext(ctx)

	_, span := tracer.Start(ctx, "sqlserver.data_source.get_database_schema")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqId),
	)

	logger.Log(context.Background(), libLog.LevelInfo, "Retrieving database schema information")

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

	schema, err := ds.buildSchema(schemaCtx, tables, primaryKeys, logger, schemas)
	if err != nil {
		return nil, err
	}

	logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Retrieved schema for %d tables", len(schema)))

	return schema, nil
}

// queryTables retrieves all table names from the database
// Returns schema-qualified names (e.g., "accounting.invoices") for non-dbo schemas,
// and simple names (e.g., "transactions") for tables in the dbo schema.
func (ds *ExternalDataSource) queryTables(ctx context.Context, schemas []string) ([]string, error) {
	base := `
        SELECT table_schema, table_name
        FROM information_schema.tables
        WHERE table_type = 'BASE TABLE'
    `

	var (
		tableQuery string
		args       []any
	)

	if len(schemas) == 0 {
		tableQuery = base + `
          AND table_schema = @p1
          ORDER BY table_name
        `
		args = []any{DefaultSchema}
	} else {
		cleaned := make([]string, 0, len(schemas))
		for _, s := range schemas {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}

			cleaned = append(cleaned, s)
		}

		if len(cleaned) == 0 {
			tableQuery = base + `
              AND table_schema = @p1
              ORDER BY table_name
            `
			args = []any{DefaultSchema}
		} else {
			placeholders := make([]string, len(cleaned))

			args = make([]any, len(cleaned))
			for i, s := range cleaned {
				placeholders[i] = fmt.Sprintf("@p%d", i+1)
				args[i] = s
			}

			tableQuery = fmt.Sprintf(base+`
              AND table_schema IN (%s)
              ORDER BY table_schema, table_name
            `, strings.Join(placeholders, ", "))
		}
	}

	rows, err := ds.connection.ConnectionDB.QueryContext(ctx, tableQuery, args...)
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

		// Use schema-qualified name for non-dbo schemas
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

// queryPrimaryKeys retrieves primary key information for all tables
// Returns a map with schema-qualified table names (e.g., "accounting.invoices") for non-dbo schemas.
func (ds *ExternalDataSource) queryPrimaryKeys(ctx context.Context, schemas []string) (map[string]map[string]bool, error) {
	base := `
        SELECT tc.table_schema, tc.table_name, kc.column_name
        FROM information_schema.table_constraints tc
        JOIN information_schema.key_column_usage kc
          ON kc.table_name = tc.table_name
         AND kc.table_schema = tc.table_schema
         AND kc.constraint_name = tc.constraint_name
        WHERE tc.constraint_type = 'PRIMARY KEY'
    `

	var (
		pkQuery string
		args    []any
	)

	if len(schemas) == 0 {
		pkQuery = base + ` AND tc.table_schema = @p1`
		args = []any{DefaultSchema}
	} else {
		cleaned := make([]string, 0, len(schemas))
		for _, s := range schemas {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}

			cleaned = append(cleaned, s)
		}

		if len(cleaned) == 0 {
			pkQuery = base + ` AND tc.table_schema = @p1`
			args = []any{DefaultSchema}
		} else {
			placeholders := make([]string, len(cleaned))

			args = make([]any, len(cleaned))
			for i, s := range cleaned {
				placeholders[i] = fmt.Sprintf("@p%d", i+1)
				args[i] = s
			}

			pkQuery = fmt.Sprintf(base+` AND tc.table_schema IN (%s)`, strings.Join(placeholders, ", "))
		}
	}

	pkRows, err := ds.connection.ConnectionDB.QueryContext(ctx, pkQuery, args...)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("schema discovery timeout after %v while querying primary keys: %w", constant.SchemaDiscoveryTimeout, err)
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

		// Use schema-qualified name for non-dbo schemas
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

// buildSchema builds the complete schema information for all tables
func (ds *ExternalDataSource) buildSchema(ctx context.Context, tables []string, primaryKeys map[string]map[string]bool, logger libLog.Logger, schemas []string) ([]TableSchema, error) {
	schema := make([]TableSchema, 0, len(tables))

	for _, tableName := range tables {
		tableSchema, err := ds.buildTableSchema(ctx, tableName, primaryKeys, logger, schemas)
		if err != nil {
			return nil, err
		}

		schema = append(schema, tableSchema)
	}

	return schema, nil
}

// buildTableSchema builds schema information for a single table
// tableName can be schema-qualified (e.g., "accounting.invoices") or simple (e.g., "transactions")
func (ds *ExternalDataSource) buildTableSchema(
	ctx context.Context,
	tableName string,
	primaryKeys map[string]map[string]bool,
	logger libLog.Logger,
	schemas []string,
) (TableSchema, error) {
	// Parse the qualified table name to get schema and simple table name
	schemaName, simpleTableName, err := parseQualifiedTableName(tableName)
	if err != nil {
		return TableSchema{}, fmt.Errorf("error parsing table name %q: %w", tableName, err)
	}

	base := `
        SELECT column_name, data_type,
               CASE WHEN is_nullable = 'YES' THEN 1 ELSE 0 END AS is_nullable
        FROM information_schema.columns
        WHERE table_name = @p1
    `

	var (
		columnQuery string
		args        []any
	)

	// If we have a specific schema from the qualified name, use it directly
	if strings.Contains(tableName, ".") {
		columnQuery = base + `
          AND table_schema = @p2
          ORDER BY ordinal_position
        `
		args = []any{simpleTableName, schemaName}
	} else if len(schemas) == 0 {
		columnQuery = base + `
          AND table_schema = @p2
          ORDER BY ordinal_position
        `
		args = []any{simpleTableName, DefaultSchema}
	} else {
		cleaned := make([]string, 0, len(schemas))
		for _, s := range schemas {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}

			cleaned = append(cleaned, s)
		}

		if len(cleaned) == 0 {
			columnQuery = base + `
              AND table_schema = @p2
              ORDER BY ordinal_position
            `
			args = []any{simpleTableName, DefaultSchema}
		} else {
			placeholders := make([]string, len(cleaned))
			args = make([]any, 0, 1+len(cleaned))
			args = append(args, simpleTableName)

			for i, s := range cleaned {
				placeholders[i] = fmt.Sprintf("@p%d", i+2)

				args = append(args, s)
			}

			columnQuery = fmt.Sprintf(base+`
              AND table_schema IN (%s)
              ORDER BY ordinal_position
            `, strings.Join(placeholders, ", "))
		}
	}

	colRows, err := ds.connection.ConnectionDB.QueryContext(ctx, columnQuery, args...)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return TableSchema{}, fmt.Errorf(
				"schema discovery timeout after %v while querying columns for table %s: %w",
				constant.SchemaDiscoveryTimeout, tableName, err,
			)
		}

		return TableSchema{}, fmt.Errorf("error querying columns for table %s: %w", tableName, err)
	}
	defer colRows.Close()

	columns, err := ds.scanColumns(colRows, tableName, primaryKeys, logger)
	if err != nil {
		return TableSchema{}, err
	}

	return TableSchema{
		TableName: tableName,
		Columns:   columns,
	}, nil
}

// parseQualifiedTableName splits a qualified table name into schema and table parts.
// For "accounting.invoices" returns ("accounting", "invoices", nil).
// For "transactions" returns ("dbo", "transactions", nil).
// For "mydb.accounting.invoices" returns ("accounting", "invoices", nil) (3-part SQL Server name).
func parseQualifiedTableName(qualifiedName string) (schema, table string, err error) {
	return schemautil.ParseQualifiedTableName(qualifiedName, DefaultSchema)
}

// normalizeTableNameForLookup normalizes a table name for schema lookup.
// The schema object stores tables as:
// - "accounting.invoices" for non-dbo schemas
// - "transactions" for dbo schema tables
// This function handles input like:
// - "accounting.invoices" -> "accounting.invoices" (non-dbo, keep as-is)
// - "dbo.transactions" -> "transactions" (dbo schema, strip prefix)
// - "transactions" -> "transactions" (no prefix, keep as-is)
func normalizeTableNameForLookup(tableName string) string {
	return schemautil.NormalizeTableNameForLookup(tableName, DefaultSchema)
}

// scanColumns scans column information from query results
func (ds *ExternalDataSource) scanColumns(colRows *sql.Rows, tableName string, primaryKeys map[string]map[string]bool, logger libLog.Logger) ([]ColumnInformation, error) {
	var columns []ColumnInformation

	for colRows.Next() {
		var (
			col           ColumnInformation
			isNullableInt int
		)

		if err := colRows.Scan(&col.Name, &col.DataType, &isNullableInt); err != nil {
			if closeErr := colRows.Close(); closeErr != nil {
				logger.Log(context.Background(), libLog.LevelWarn, fmt.Sprintf("error closing rows after scan error: %v", closeErr))
			}

			return nil, fmt.Errorf("error scanning column info: %w", err)
		}

		col.IsNullable = isNullableInt == 1

		if pkCols, exists := primaryKeys[tableName]; exists {
			col.IsPrimaryKey = pkCols[col.Name]
		}

		columns = append(columns, col)
	}

	if err := colRows.Close(); err != nil {
		logger.Log(context.Background(), libLog.LevelWarn, fmt.Sprintf("error closing column rows: %v", err))
	}

	return columns, nil
}

// scanRows processes the query rows and creates the resulting slice of maps.
func scanRows(rows *sql.Rows, logger libLog.Logger) ([]map[string]any, error) {
	columns, _ := rows.Columns()
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

		rowMap := createRowMap(columns, values, logger)
		result = append(result, rowMap)
	}

	return result, nil
}

// createRowMap maps column names to their respective values.
func createRowMap(columns []string, values []any, logger libLog.Logger) map[string]any {
	rowMap := make(map[string]any)

	for i, column := range columns {
		// Attempt to parse any value that could be JSON
		rowMap[column] = parseJSONField(values[i], logger)
	}

	return rowMap
}

// parseJSONField unmarshals any field that might be a JSON type
func parseJSONField(value any, logger libLog.Logger) any {
	if value == nil {
		return nil
	}

	// Check if the value is []uint8, which is how the SQL Server driver
	// represents JSON and JSON fields
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
		logger.Log(context.Background(), libLog.LevelWarn, fmt.Sprintf("Failed to unmarshal potential JSON data for value: %v", string(byteData)))
	}

	return value
}

// ValidateTableAndFields checks if the specified table exists and validates that
// all requested fields exist in that table.
// It returns a list of valid fields and an error if the table doesn't exist or fields are invalid.
func (ds *ExternalDataSource) ValidateTableAndFields(ctx context.Context, tableName string, requestedFields []string, schema []TableSchema) ([]string, error) {
	logger, tracer, reqId, _ := libCommons.NewTrackingFromContext(ctx)

	_, span := tracer.Start(ctx, "sqlserver.data_source.validate_table_and_fields")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqId),
	)

	// Normalize table name for lookup: schema-qualified names should match
	// the schema object which stores tables as "accounting.invoices" for non-dbo
	// schemas and "transactions" for dbo schema tables.
	lookupName := normalizeTableNameForLookup(tableName)

	err := libOpentelemetry.SetSpanAttributesFromValue(span, "app.request.repository_filter", map[string]any{
		"table":  tableName,
		"fields": requestedFields,
		"schema": schema,
	}, nil)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert repository filter to JSON string", err)
	}

	logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Validating table '%s' and fields %v", tableName, requestedFields))

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

	logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Successfully validated table '%s' and fields %v", tableName, validFields))

	return validFields, nil
}

// buildDynamicFilters applies filter criteria to the query builder based on valid columns.
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

// applyFilter adds a WHERE condition for a field with multiple possible values.
// Uses squirrel.Eq which respects the PlaceholderFormat configured in the StatementBuilder
func applyFilter(queryBuilder squirrel.SelectBuilder, fieldName string, values []any) squirrel.SelectBuilder {
	if len(values) == 0 {
		return queryBuilder
	}

	// Use squirrel.Eq for IN clause which respects PlaceholderFormat
	// For single value, use direct equality; for multiple values, use IN
	if len(values) == 1 {
		return queryBuilder.Where(squirrel.Eq{fieldName: values[0]})
	}

	return queryBuilder.Where(squirrel.Eq{fieldName: values})
}

// QueryWithAdvancedFilters executes a SELECT SQL query with advanced FilterCondition support
func (ds *ExternalDataSource) QueryWithAdvancedFilters(ctx context.Context, schema []TableSchema, table string, fields []string, filter map[string]job.FilterCondition) ([]map[string]any, error) {
	logger, tracer, reqId, _ := libCommons.NewTrackingFromContext(ctx)

	_, span := tracer.Start(ctx, "sqlserver.data_source.query_with_advanced_filters")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqId),
	)

	err := libOpentelemetry.SetSpanAttributesFromValue(span, "app.request.repository_filter", map[string]any{
		"table":  table,
		"fields": fields,
		"filter": filter,
	}, nil)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert repository filter to JSON string", err)
	}

	logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Querying %s table with advanced filters on fields %v", table, fields))

	// Validate requested table and fields
	queriedFields, err := ds.ValidateTableAndFields(ctx, table, fields, schema)
	if err != nil {
		return nil, err
	}

	// SQL Server uses @p1, @p2, etc. for placeholders (not ?)
	sqlServer := squirrel.StatementBuilder.PlaceholderFormat(sqlServerPlaceholderFormat)
	queryBuilder := sqlServer.Select(queriedFields...).From(table)

	// Apply advanced filters
	queryBuilder, err = ds.buildAdvancedFilters(queryBuilder, schema, table, filter)
	if err != nil {
		return nil, fmt.Errorf("error building advanced filters: %w", err)
	}

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("error generating SQL: %w", err)
	}

	logger.Log(context.Background(), libLog.LevelDebug, fmt.Sprintf("Executing advanced filter SQL: %s", query))
	logger.Log(context.Background(), libLog.LevelDebug, fmt.Sprintf("SQL args: %v", args))
	logger.Log(context.Background(), libLog.LevelDebug, fmt.Sprintf("Original filter conditions: %+v", filter))

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

	return scanRows(rows, logger)
}

// buildAdvancedFilters applies FilterCondition criteria to the query builder
func (ds *ExternalDataSource) buildAdvancedFilters(queryBuilder squirrel.SelectBuilder, schema []TableSchema, table string, filter map[string]job.FilterCondition) (squirrel.SelectBuilder, error) {
	var tableColumns []ColumnInformation

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

// applyAdvancedFilter applies a single FilterCondition to the query builder
func (ds *ExternalDataSource) applyAdvancedFilter(queryBuilder squirrel.SelectBuilder, field string, condition job.FilterCondition) squirrel.SelectBuilder {
	queryBuilder = applyEqualsFilter(queryBuilder, field, condition)
	queryBuilder = applyComparisonFilters(queryBuilder, field, condition)
	queryBuilder = applyBetweenFilter(queryBuilder, field, condition)
	queryBuilder = applyInFilters(queryBuilder, field, condition)
	queryBuilder = applyNotEqualsFilter(queryBuilder, field, condition)
	queryBuilder = applyLikeFilter(queryBuilder, field, condition)

	return queryBuilder
}

// applyEqualsFilter applies equals operator to the query builder
func applyEqualsFilter(queryBuilder squirrel.SelectBuilder, field string, condition job.FilterCondition) squirrel.SelectBuilder {
	if len(condition.Equals) == 0 {
		return queryBuilder
	}

	if len(condition.Equals) == 1 {
		return queryBuilder.Where(squirrel.Eq{field: condition.Equals[0]})
	}

	return queryBuilder.Where(squirrel.Eq{field: condition.Equals})
}

// applyComparisonFilters applies comparison operators (gt, gte, lt, lte) to the query builder
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

// applyBetweenFilter applies between operator to the query builder
func applyBetweenFilter(queryBuilder squirrel.SelectBuilder, field string, condition job.FilterCondition) squirrel.SelectBuilder {
	if len(condition.Between) != 2 {
		return queryBuilder
	}

	startValue := condition.Between[0]
	endValue := adjustEndDateIfNeeded(field, condition.Between[0], condition.Between[1])

	return queryBuilder.Where(squirrel.GtOrEq{field: startValue}).Where(squirrel.LtOrEq{field: endValue})
}

// adjustEndDateIfNeeded adjusts end date to end of day for date fields
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

// applyInFilters applies in and not in operators to the query builder
func applyInFilters(queryBuilder squirrel.SelectBuilder, field string, condition job.FilterCondition) squirrel.SelectBuilder {
	if len(condition.In) > 0 {
		queryBuilder = queryBuilder.Where(squirrel.Eq{field: condition.In})
	}

	if len(condition.NotIn) > 0 {
		queryBuilder = queryBuilder.Where(squirrel.NotEq{field: condition.NotIn})
	}

	return queryBuilder
}

// applyNotEqualsFilter applies not equals operator to the query builder
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

// applyLikeFilter applies like pattern matching to the query builder
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

// isFilterConditionEmpty checks if a FilterCondition has no active filters
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

// validateFilterCondition validates that a FilterCondition has proper values for each operator
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

// isLikelyUUIDField checks if a field name suggests it contains UUID values
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

// validateUUIDFieldValues validates that values for UUID fields are valid UUIDs
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

// isValidUUIDFormat checks if a string is a valid UUID format
func isValidUUIDFormat(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

// isDateField checks if a field name suggests it contains date/timestamp values
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

// isDateString checks if a value looks like a date string
func isDateString(value any) bool {
	if str, ok := value.(string); ok {
		// Check for common date formats: YYYY-MM-DD, YYYY-MM-DDTHH:MM:SS, etc.
		return len(str) >= 10 && strings.Contains(str, "-") && (len(str) == 10 || strings.Contains(str, "T"))
	}

	return false
}

// sqlServerPlaceholder is a custom placeholder format for SQL Server
// SQL Server requires @p1, @p2, @p3, etc. for parameterized queries
type sqlServerPlaceholder struct{}

// ReplacePlaceholders replaces ? with @p1, @p2, @p3, etc. as required by the mssql driver
func (p sqlServerPlaceholder) ReplacePlaceholders(sqlStr string) (string, error) {
	placeholderCount := 1
	result := make([]byte, 0, len(sqlStr)+20)

	for i := 0; i < len(sqlStr); i++ {
		if sqlStr[i] == '?' {
			result = append(result, '@', 'p')
			result = append(result, fmt.Sprintf("%d", placeholderCount)...)
			placeholderCount++
		} else {
			result = append(result, sqlStr[i])
		}
	}

	return string(result), nil
}

// sqlServerPlaceholderFormat is the PlaceholderFormat instance for SQL Server
var sqlServerPlaceholderFormat = sqlServerPlaceholder{}
