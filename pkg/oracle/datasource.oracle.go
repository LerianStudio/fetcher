package oracle

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model/datasource"
	"github.com/LerianStudio/fetcher/pkg/model/job"

	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

// Repository defines an interface for querying data from a specified table and fields.
//
//go:generate mockgen --destination=datasource.oracle.mock.go --package=oracle . Repository
type Repository interface {
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

// ExternalDataSource provides an interface for interacting with an Oracle database connection.
type ExternalDataSource struct {
	connection *Connection
}

// NewDataSourceRepository creates a new ExternalDataSource instance using the provided oracle.Connection, initializing the database connection.
// Returns nil and error if connection fails.
func NewDataSourceRepository(oc *Connection) (*ExternalDataSource, error) {
	c := &ExternalDataSource{
		connection: oc,
	}

	_, err := c.connection.GetDB()
	if err != nil {
		oc.Logger.Errorf("Failed to establish Oracle connection: %v", err)
		return nil, fmt.Errorf("failed to establish Oracle connection: %w", err)
	}

	return c, nil
}

// CloseConnection closing the connection with Oracle.
func (ds *ExternalDataSource) CloseConnection() error {
	if ds.connection.ConnectionDB != nil {
		ds.connection.Logger.Info("Closing connection to Oracle...")

		err := ds.connection.ConnectionDB.Close()
		if err != nil {
			ds.connection.Logger.Errorf("Error closing Oracle connection: %v", err)
			return err
		}

		ds.connection.Connected = false
		ds.connection.ConnectionDB = nil
		ds.connection.Logger.Info("Oracle connection closed successfully.")
	}

	return nil
}

// Query executes a SELECT SQL query on the specified table with the given fields and filter criteria.
// It returns the query results as a slice of maps or an error in case of failure.
func (ds *ExternalDataSource) Query(ctx context.Context, schema []TableSchema, table string, fields []string, filter map[string][]any) ([]map[string]any, error) {
	logger, tracer, reqId, _ := libCommons.NewTrackingFromContext(ctx)

	_, span := tracer.Start(ctx, "oracle.data_source.query")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqId),
	)

	err := libOpentelemetry.SetSpanAttributesFromStruct(&span, "app.request.repository_filter", map[string]any{
		"table":  table,
		"fields": fields,
		"filter": filter,
	})
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert repository filter to JSON string", err)
	}

	logger.Infof("Querying %s table with fields %v", table, fields)

	// Validate requested table and fields
	queriedFields, err := ds.ValidateTableAndFields(ctx, table, fields, schema)
	if err != nil {
		return nil, err
	}

	// Oracle uses :1, :2, etc. for placeholders
	ora := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Colon)
	queryBuilder := ora.Select(queriedFields...).From(table)

	// Apply filters, but only if they correspond to valid columns
	queryBuilder = buildDynamicFilters(queryBuilder, schema, table, filter)

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("error generating SQL: %w", err)
	}

	logger.Infof("Executing SQL: %s with args: %v", query, args)

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

	_, span := tracer.Start(ctx, "oracle.data_source.get_database_schema")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqId),
	)

	logger.Info("Retrieving database schema information")

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

	logger.Infof("Retrieved schema for %d tables", len(schema))

	return schema, nil
}

// queryTables retrieves all table names from the database
func (ds *ExternalDataSource) queryTables(ctx context.Context, schemas []string) ([]string, error) {
	var (
		tableQuery string
		args       []any
	)

	if len(schemas) == 0 {
		tableQuery = `
            SELECT table_name
            FROM user_tables
            ORDER BY table_name
        `
		args = nil
	} else {
		placeholders := make([]string, 0, len(schemas))
		args = make([]any, 0, len(schemas))

		for i, s := range schemas {
			s = strings.ToUpper(strings.TrimSpace(s))
			if s == "" {
				continue
			}
			placeholders = append(placeholders, fmt.Sprintf(":%d", i+1))
			args = append(args, s)
		}

		if len(placeholders) == 0 {
			tableQuery = `
                SELECT table_name
                FROM user_tables
                ORDER BY table_name
            `
			args = nil
		} else {
			tableQuery = fmt.Sprintf(`
                SELECT table_name
                FROM all_tables
                WHERE owner IN (%s)
                ORDER BY owner, table_name
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
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("error scanning table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return tables, nil
}

// queryPrimaryKeys retrieves primary key information for all tables
func (ds *ExternalDataSource) queryPrimaryKeys(ctx context.Context, schemas []string) (map[string]map[string]bool, error) {
	var (
		pkQuery string
		args    []any
	)

	if len(schemas) == 0 {
		pkQuery = `
            SELECT table_name, column_name
            FROM user_cons_columns
            WHERE constraint_name IN (
                SELECT constraint_name
                FROM user_constraints
                WHERE constraint_type = 'P'
            )
        `
		args = nil
	} else {
		placeholders := make([]string, 0, len(schemas))
		args = make([]any, 0, len(schemas))

		for i, s := range schemas {
			s = strings.ToUpper(strings.TrimSpace(s))
			if s == "" {
				continue
			}
			placeholders = append(placeholders, fmt.Sprintf(":%d", i+1))
			args = append(args, s)
		}

		if len(placeholders) == 0 {
			pkQuery = `
                SELECT table_name, column_name
                FROM user_cons_columns
                WHERE constraint_name IN (
                    SELECT constraint_name
                    FROM user_constraints
                    WHERE constraint_type = 'P'
                )
            `
			args = nil
		} else {
			pkQuery = fmt.Sprintf(`
                SELECT acc.table_name, acc.column_name
                FROM all_cons_columns acc
                JOIN all_constraints ac
                  ON ac.owner = acc.owner
                 AND ac.constraint_name = acc.constraint_name
                WHERE ac.constraint_type = 'P'
                  AND ac.owner IN (%s)
            `, strings.Join(placeholders, ", "))
		}
	}

	pkRows, err := ds.connection.ConnectionDB.QueryContext(ctx, pkQuery, args...)
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
		var tableName, columnName string
		if err := pkRows.Scan(&tableName, &columnName); err != nil {
			return nil, fmt.Errorf("error scanning primary key info: %w", err)
		}

		if primaryKeys[tableName] == nil {
			primaryKeys[tableName] = make(map[string]bool)
		}
		primaryKeys[tableName][columnName] = true
	}

	if err := pkRows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return primaryKeys, nil
}

// buildSchema builds the complete schema information for all tables
func (ds *ExternalDataSource) buildSchema(ctx context.Context, tables []string, primaryKeys map[string]map[string]bool, logger log.Logger, schemas []string) ([]TableSchema, error) {
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
func (ds *ExternalDataSource) buildTableSchema(
	ctx context.Context,
	tableName string,
	primaryKeys map[string]map[string]bool,
	logger log.Logger,
	schemas []string,
) (TableSchema, error) {
	tableName = strings.ToUpper(tableName)

	var (
		columnQuery string
		args        []any
	)

	if len(schemas) == 0 {
		columnQuery = `
            SELECT column_name, data_type,
                   CASE WHEN nullable = 'Y' THEN 1 ELSE 0 END AS is_nullable
            FROM user_tab_columns
            WHERE table_name = :1
            ORDER BY column_id
        `
		args = []any{tableName}
	} else {
		placeholders := make([]string, 0, len(schemas))
		args = make([]any, 0, 1+len(schemas))
		args = append(args, tableName)

		for i, s := range schemas {
			s = strings.ToUpper(strings.TrimSpace(s))
			if s == "" {
				continue
			}
			placeholders = append(placeholders, fmt.Sprintf(":%d", i+2))
			args = append(args, s)
		}

		if len(placeholders) == 0 {
			columnQuery = `
                SELECT column_name, data_type,
                       CASE WHEN nullable = 'Y' THEN 1 ELSE 0 END AS is_nullable
                FROM user_tab_columns
                WHERE table_name = :1
                ORDER BY column_id
            `
			args = []any{tableName}
		} else {
			columnQuery = fmt.Sprintf(`
                SELECT column_name, data_type,
                       CASE WHEN nullable = 'Y' THEN 1 ELSE 0 END AS is_nullable
                FROM all_tab_columns
                WHERE table_name = :1
                  AND owner IN (%s)
                ORDER BY owner, column_id
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

// scanColumns scans column information from query results
func (ds *ExternalDataSource) scanColumns(colRows *sql.Rows, tableName string, primaryKeys map[string]map[string]bool, logger log.Logger) ([]ColumnInformation, error) {
	var columns []ColumnInformation

	for colRows.Next() {
		var (
			col           ColumnInformation
			isNullableInt int
		)

		if err := colRows.Scan(&col.Name, &col.DataType, &isNullableInt); err != nil {
			if closeErr := colRows.Close(); closeErr != nil {
				logger.Warnf("error closing rows after scan error: %v", closeErr)
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
		logger.Warnf("error closing column rows: %v", err)
	}

	return columns, nil
}

// scanRows processes the query rows and creates the resulting slice of maps.
func scanRows(rows *sql.Rows, logger log.Logger) ([]map[string]any, error) {
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
func createRowMap(columns []string, values []any, logger log.Logger) map[string]any {
	rowMap := make(map[string]any)

	for i, column := range columns {
		// Attempt to parse any value that could be JSON
		rowMap[column] = parseJSONField(values[i], logger)
	}

	return rowMap
}

// parseJSONField unmarshals any field that might be a JSON type
func parseJSONField(value any, logger log.Logger) any {
	if value == nil {
		return nil
	}

	// Check if the value is []uint8, which is how some drivers represent JSON fields
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
		logger.Warnf("Failed to unmarshal potential JSON data for value: %v", string(byteData))
	}

	return value
}

// ValidateTableAndFields checks if the specified table exists and validates that
// all requested fields exist in that table.
// It returns a list of valid fields and an error if the table doesn't exist or fields are invalid.
func (ds *ExternalDataSource) ValidateTableAndFields(ctx context.Context, tableName string, requestedFields []string, schema []TableSchema) ([]string, error) {
	logger, tracer, reqId, _ := libCommons.NewTrackingFromContext(ctx)

	_, span := tracer.Start(ctx, "oracle.data_source.validate_table_and_fields")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqId),
	)

	_, tableName = datasource.SplitSchemaTable(tableName)

	err := libOpentelemetry.SetSpanAttributesFromStruct(&span, "app.request.repository_filter", map[string]any{
		"table":  tableName,
		"fields": requestedFields,
		"schema": schema,
	})
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert repository filter to JSON string", err)
	}

	logger.Infof("Validating table '%s' and fields %v", tableName, requestedFields)

	// Check if table exists
	var tableFound bool

	var tableColumns []ColumnInformation

	for _, table := range schema {
		if strings.EqualFold(table.TableName, tableName) {
			tableFound = true
			tableColumns = table.Columns

			break
		}
	}

	if !tableFound {
		return nil, fmt.Errorf("table '%s' does not exist in the database", tableName)
	}

	// Create a map of valid column names for efficient lookup (case-insensitive for Oracle)
	validColumns := make(map[string]bool)
	for _, col := range tableColumns {
		validColumns[strings.ToUpper(col.Name)] = true
	}

	// Special case: if "*" is in the fields, return all columns
	if len(requestedFields) == 1 && requestedFields[0] == "*" {
		allFields := make([]string, len(tableColumns))
		for i, col := range tableColumns {
			allFields[i] = col.Name
		}

		return allFields, nil
	}

	// Validate each requested field (case-insensitive)
	var validFields []string

	var invalidFields []string

	for _, field := range requestedFields {
		if validColumns[strings.ToUpper(field)] {
			// Find the original case from schema
			for _, col := range tableColumns {
				if strings.EqualFold(col.Name, field) {
					validFields = append(validFields, col.Name)
					break
				}
			}
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

	logger.Infof("Successfully validated table '%s' and fields %v", tableName, validFields)

	return validFields, nil
}

// buildDynamicFilters applies filter criteria to the query builder based on valid columns.
func buildDynamicFilters(queryBuilder squirrel.SelectBuilder, schema []TableSchema, table string, filter map[string][]any) squirrel.SelectBuilder {
	// Find the table's column information
	var tableColumns []ColumnInformation

	_, table = datasource.SplitSchemaTable(table)

	for _, t := range schema {
		if strings.EqualFold(t.TableName, table) {
			tableColumns = t.Columns
			break
		}
	}

	// Create a map of valid column names for efficient lookup (case-insensitive)
	validColumns := make(map[string]bool)
	for _, col := range tableColumns {
		validColumns[strings.ToUpper(col.Name)] = true
	}

	for field, values := range filter {
		// Only apply filters for valid columns (case-insensitive)
		if validColumns[strings.ToUpper(field)] && len(values) > 0 {
			// Find original case
			var originalField string

			for _, col := range tableColumns {
				if strings.EqualFold(col.Name, field) {
					originalField = col.Name
					break
				}
			}

			queryBuilder = applyFilter(queryBuilder, originalField, values)
		}
	}

	return queryBuilder
}

// applyFilter adds a WHERE condition for a field with multiple possible values.
func applyFilter(queryBuilder squirrel.SelectBuilder, fieldName string, values []any) squirrel.SelectBuilder {
	if len(values) == 0 {
		return queryBuilder
	}

	// Oracle uses :1, :2, etc. for placeholders
	placeholder := squirrel.Placeholders(len(values))

	return queryBuilder.Where(fieldName+" IN ("+placeholder+")", values...)
}

// QueryWithAdvancedFilters executes a SELECT SQL query with advanced FilterCondition support
func (ds *ExternalDataSource) QueryWithAdvancedFilters(ctx context.Context, schema []TableSchema, table string, fields []string, filter map[string]job.FilterCondition) ([]map[string]any, error) {
	logger, tracer, reqId, _ := libCommons.NewTrackingFromContext(ctx)

	_, span := tracer.Start(ctx, "oracle.data_source.query_with_advanced_filters")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqId),
	)

	err := libOpentelemetry.SetSpanAttributesFromStruct(&span, "app.request.repository_filter", map[string]any{
		"table":  table,
		"fields": fields,
		"filter": filter,
	})
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert repository filter to JSON string", err)
	}

	logger.Infof("Querying %s table with advanced filters on fields %v", table, fields)

	// Validate requested table and fields
	queriedFields, err := ds.ValidateTableAndFields(ctx, table, fields, schema)
	if err != nil {
		return nil, err
	}

	// Oracle uses :1, :2, etc. for placeholders
	ora := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Colon)
	queryBuilder := ora.Select(queriedFields...).From(table)

	// Apply advanced filters
	queryBuilder, err = ds.buildAdvancedFilters(queryBuilder, schema, table, filter)
	if err != nil {
		return nil, fmt.Errorf("error building advanced filters: %w", err)
	}

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("error generating SQL: %w", err)
	}

	logger.Debugf("Executing advanced filter SQL: %s", query)
	logger.Debugf("SQL args: %v", args)
	logger.Debugf("Original filter conditions: %+v", filter)

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

	_, table = datasource.SplitSchemaTable(table)

	for _, t := range schema {
		if strings.EqualFold(t.TableName, table) {
			tableColumns = t.Columns
			break
		}
	}

	// Create a map of valid column names for efficient lookup (case-insensitive)
	validColumns := make(map[string]bool)
	for _, col := range tableColumns {
		validColumns[strings.ToUpper(col.Name)] = true
	}

	for field, condition := range filter {
		// Only apply filters for valid columns (case-insensitive)
		if !validColumns[strings.ToUpper(field)] {
			continue
		}

		if isFilterConditionEmpty(condition) {
			continue
		}

		// Validate the condition
		if err := validateFilterCondition(field, condition); err != nil {
			return queryBuilder, err
		}

		// Find original case
		var originalField string

		for _, col := range tableColumns {
			if strings.EqualFold(col.Name, field) {
				originalField = col.Name
				break
			}
		}

		// Apply each filter operator
		queryBuilder = ds.applyAdvancedFilter(queryBuilder, originalField, condition)
	}

	return queryBuilder, nil
}

// applyAdvancedFilter applies a single FilterCondition to the query builder
func (ds *ExternalDataSource) applyAdvancedFilter(queryBuilder squirrel.SelectBuilder, field string, condition job.FilterCondition) squirrel.SelectBuilder {
	// Handle equals (IN clause for multiple values, = for single value)
	if len(condition.Equals) > 0 {
		if len(condition.Equals) == 1 {
			queryBuilder = queryBuilder.Where(squirrel.Eq{field: condition.Equals[0]})
		} else {
			queryBuilder = queryBuilder.Where(squirrel.Eq{field: condition.Equals})
		}
	}

	// Handle greater than
	if len(condition.GreaterThan) > 0 {
		queryBuilder = queryBuilder.Where(squirrel.Gt{field: condition.GreaterThan[0]})
	}

	// Handle greater than or equal
	if len(condition.GreaterOrEqual) > 0 {
		queryBuilder = queryBuilder.Where(squirrel.GtOrEq{field: condition.GreaterOrEqual[0]})
	}

	// Handle less than
	if len(condition.LessThan) > 0 {
		queryBuilder = queryBuilder.Where(squirrel.Lt{field: condition.LessThan[0]})
	}

	// Handle less than or equal
	if len(condition.LessOrEqual) > 0 {
		queryBuilder = queryBuilder.Where(squirrel.LtOrEq{field: condition.LessOrEqual[0]})
	}

	// Handle between (using AND with >= and <=)
	if len(condition.Between) == 2 {
		startValue := condition.Between[0]
		endValue := condition.Between[1]

		// If it looks like a date field and we have date strings, adjust the end date to include the full day
		if isDateField(field) && isDateString(startValue) && isDateString(endValue) {
			if endStr, ok := endValue.(string); ok {
				if len(endStr) == 10 { // YYYY-MM-DD format
					endValue = endStr + "T23:59:59.999Z"
				}
			}
		}

		queryBuilder = queryBuilder.Where(squirrel.GtOrEq{field: startValue}).Where(squirrel.LtOrEq{field: endValue})
	}

	// Handle in
	if len(condition.In) > 0 {
		queryBuilder = queryBuilder.Where(squirrel.Eq{field: condition.In})
	}

	// Handle not in
	if len(condition.NotIn) > 0 {
		queryBuilder = queryBuilder.Where(squirrel.NotEq{field: condition.NotIn})
	}

	return queryBuilder
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
		len(condition.NotIn) == 0
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
