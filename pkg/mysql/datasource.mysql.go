package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model/job"

	libLog "github.com/LerianStudio/lib-observability/log"
	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

// Datasource defines an interface for querying data from a specified table and fields.
//
//go:generate mockgen --destination=datasource.mysql.mock.go --package=mysql . Datasource
type Datasource interface {
	Query(ctx context.Context, schema []TableSchema, table string, fields []string, filter map[string][]any) ([]map[string]any, error)
	QueryWithAdvancedFilters(ctx context.Context, schema []TableSchema, table string, fields []string, filter map[string]job.FilterCondition) ([]map[string]any, error)
	GetDatabaseSchema(ctx context.Context) ([]TableSchema, error)
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

// ExternalDataSource provides an interface for interacting with a MySQL database connection.
type ExternalDataSource struct {
	connection *Connection
}

// NewDataSourceRepository creates a new ExternalDataSource instance using the provided mysql.Connection, initializing the database connection.
// Returns nil and error if connection fails.
func NewDataSourceRepository(pc *Connection) (*ExternalDataSource, error) {
	c := &ExternalDataSource{
		connection: pc,
	}

	_, err := c.connection.GetDB()
	if err != nil {
		pc.Logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Failed to establish MySQL connection: %v", err))
		return nil, fmt.Errorf("failed to establish MySQL connection: %w", err)
	}

	return c, nil
}

// CloseConnection closing the connection with MySQL.
func (ds *ExternalDataSource) CloseConnection() error {
	if ds.connection.ConnectionDB != nil {
		ds.connection.Logger.Log(context.Background(), libLog.LevelInfo, "Closing connection to MySQL...")

		err := ds.connection.ConnectionDB.Close()
		if err != nil {
			ds.connection.Logger.Log(context.Background(), libLog.LevelError, fmt.Sprintf("Error closing MySQL connection: %v", err))
			return err
		}

		ds.connection.Connected = false
		ds.connection.ConnectionDB = nil
		ds.connection.Logger.Log(context.Background(), libLog.LevelInfo, "MySQL connection closed successfully.")
	}

	return nil
}

// Query executes a SELECT SQL query on the specified table with the given fields and filter criteria.
// It returns the query results as a slice of maps or an error in case of failure.
func (ds *ExternalDataSource) Query(ctx context.Context, schema []TableSchema, table string, fields []string, filter map[string][]any) ([]map[string]any, error) {
	logger, tracer, reqId, _ := observability.NewTrackingFromContext(ctx)

	_, span := tracer.Start(ctx, "mysql.data_source.query")
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

	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question)
	queryBuilder := psql.Select(queriedFields...).From(table)

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
func (ds *ExternalDataSource) GetDatabaseSchema(ctx context.Context) ([]TableSchema, error) {
	logger, tracer, reqId, _ := observability.NewTrackingFromContext(ctx)

	_, span := tracer.Start(ctx, "mysql.data_source.get_database_schema")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqId),
	)

	logger.Log(context.Background(), libLog.LevelInfo, "Retrieving database schema information")

	schemaCtx, cancel := context.WithTimeout(ctx, constant.SchemaDiscoveryTimeout)
	defer cancel()

	tables, err := ds.queryTables(schemaCtx)
	if err != nil {
		return nil, err
	}

	primaryKeys, err := ds.queryPrimaryKeys(schemaCtx)
	if err != nil {
		return nil, err
	}

	schema, err := ds.buildSchema(schemaCtx, tables, primaryKeys, logger)
	if err != nil {
		return nil, err
	}

	logger.Log(context.Background(), libLog.LevelInfo, fmt.Sprintf("Retrieved schema for %d tables", len(schema)))

	return schema, nil
}

// queryTables retrieves all table names from the database
func (ds *ExternalDataSource) queryTables(ctx context.Context) ([]string, error) {
	tableQuery := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = DATABASE()
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := ds.connection.ConnectionDB.QueryContext(ctx, tableQuery)
	if err != nil {
		return nil, fmt.Errorf("error querying tables: %w", err)
	}
	defer rows.Close()

	var tables []string

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("error scanning table name: %w", err)
		}

		tables = append(tables, tableName)
	}

	return tables, nil
}

// queryPrimaryKeys retrieves primary key information for all tables
func (ds *ExternalDataSource) queryPrimaryKeys(ctx context.Context) (map[string]map[string]bool, error) {
	pkQuery := `
		SELECT tc.table_name, kc.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kc 
			ON kc.table_name = tc.table_name 
			AND kc.table_schema = tc.table_schema
			AND kc.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'PRIMARY KEY'
		AND tc.table_schema = DATABASE()
	`

	pkRows, err := ds.connection.ConnectionDB.QueryContext(ctx, pkQuery)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("schema discovery timeout after %v while querying primary keys: %w", constant.SchemaDiscoveryTimeout, err)
		}

		return nil, fmt.Errorf("error querying primary keys: %w", err)
	}
	defer pkRows.Close()

	primaryKeys := make(map[string]map[string]bool)

	for pkRows.Next() {
		var tableName, columnName string
		if err := pkRows.Scan(&tableName, &columnName); err != nil {
			return nil, fmt.Errorf("error scanning primary key info: %w", err)
		}

		if _, exists := primaryKeys[tableName]; !exists {
			primaryKeys[tableName] = make(map[string]bool)
		}

		primaryKeys[tableName][columnName] = true
	}

	return primaryKeys, nil
}

// buildSchema builds the complete schema information for all tables
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

// buildTableSchema builds schema information for a single table
func (ds *ExternalDataSource) buildTableSchema(ctx context.Context, tableName string, primaryKeys map[string]map[string]bool, logger libLog.Logger) (TableSchema, error) {
	columnQuery := `
		SELECT column_name, data_type, 
		       CASE WHEN is_nullable = 'YES' THEN true ELSE false END as is_nullable
		FROM information_schema.columns
		WHERE table_schema = DATABASE()
		AND table_name = ?
		ORDER BY ordinal_position
	`

	colRows, err := ds.connection.ConnectionDB.QueryContext(ctx, columnQuery, tableName)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return TableSchema{}, fmt.Errorf("schema discovery timeout after %v while querying columns for table %s: %w", constant.SchemaDiscoveryTimeout, tableName, err)
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

// scanColumns scans column information from query results
func (ds *ExternalDataSource) scanColumns(colRows *sql.Rows, tableName string, primaryKeys map[string]map[string]bool, logger libLog.Logger) ([]ColumnInformation, error) {
	var columns []ColumnInformation

	for colRows.Next() {
		var col ColumnInformation
		if err := colRows.Scan(&col.Name, &col.DataType, &col.IsNullable); err != nil {
			if closeErr := colRows.Close(); closeErr != nil {
				logger.Log(context.Background(), libLog.LevelWarn, fmt.Sprintf("error closing rows after scan error: %v", closeErr))
			}

			return nil, fmt.Errorf("error scanning column info: %w", err)
		}

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

	// Check if the value is []uint8, which is how the MySQL driver
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
	logger, tracer, reqId, _ := observability.NewTrackingFromContext(ctx)

	_, span := tracer.Start(ctx, "mysql.data_source.validate_table_and_fields")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqId),
	)

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
		if table.TableName == tableName {
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

	for _, t := range schema {
		if t.TableName == table {
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
func applyFilter(queryBuilder squirrel.SelectBuilder, fieldName string, values []any) squirrel.SelectBuilder {
	if len(values) == 0 {
		return queryBuilder
	}

	// No need for conversion since values is already []any
	placeholder := squirrel.Placeholders(len(values))

	return queryBuilder.Where(fieldName+" IN ("+placeholder+")", values...)
}

// QueryWithAdvancedFilters executes a SELECT SQL query with advanced FilterCondition support
func (ds *ExternalDataSource) QueryWithAdvancedFilters(ctx context.Context, schema []TableSchema, table string, fields []string, filter map[string]job.FilterCondition) ([]map[string]any, error) {
	logger, tracer, reqId, _ := observability.NewTrackingFromContext(ctx)

	_, span := tracer.Start(ctx, "mysql.data_source.query_with_advanced_filters")
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

	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question)
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

	logger.Log(context.Background(), libLog.LevelDebug, fmt.Sprintf("Executing advanced filter SQL: %s with args: %v", query, args))

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

	for _, t := range schema {
		if t.TableName == table {
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
