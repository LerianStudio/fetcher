package query

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/datasource"
	"github.com/LerianStudio/fetcher/pkg/model"
	datasourceModel "github.com/LerianStudio/fetcher/pkg/model/datasource"
	cacheRepo "github.com/LerianStudio/fetcher/pkg/ports/cache"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/pkg/resolver"
	"github.com/LerianStudio/fetcher/pkg/schemautil"

	"github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"

	"go.opentelemetry.io/otel/attribute"
)

const (
	// pluginCRMConfigName is the config name that requires special table name transformation.
	pluginCRMConfigName = "plugin_crm"
)

var errDataSourceFactoryNotConfigured = errors.New("datasource factory is not configured")

// ValidateSchema is the query service for validating schema references.
type ValidateSchema struct {
	connRepo    connRepo.Repository
	cryptor     crypto.Cryptor
	schemaCache cacheRepo.SchemaCacheRepository
	dsFactory   datasource.DataSourceFactory
	resolver    resolver.ConnectionResolver // nil-safe: if nil, uses connRepo only
}

// NewValidateSchema creates a new ValidateSchema service without rate limiting.
func NewValidateSchema(
	connectionRepo connRepo.Repository,
	cryptor crypto.Cryptor,
	schemaCache cacheRepo.SchemaCacheRepository,
	factory datasource.DataSourceFactory,
	connResolver resolver.ConnectionResolver,
) *ValidateSchema {
	return &ValidateSchema{
		connRepo:    connectionRepo,
		cryptor:     cryptor,
		schemaCache: schemaCache,
		dsFactory:   factory,
		resolver:    connResolver,
	}
}

// Execute validates schema references against configured datasources.
func (s *ValidateSchema) Execute(
	ctx context.Context,
	request model.SchemaValidationRequest,
) (*model.SchemaValidationResponse, error) {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.validate_schema")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.Int("app.request.datasource_count", len(request.MappedFields)),
	)

	if err := libOpentelemetry.SetSpanAttributesFromValue(span, "app.request.payload", request.ToMapWithMask(), nil); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert request to JSON string", err)
	}

	// Create validation spec from request
	spec := model.NewSchemaValidationSpec(request)

	// Validate request payload structure and limits
	if errValidation := spec.Validate(); errValidation != nil {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Invalid request payload", errValidation)
		logger.Log(ctx, libLog.LevelWarn, "schema validation request invalid",
			libLog.Err(errValidation),
		)

		return nil, errValidation
	}

	// Fetch connections for the requested config names from the repository
	configNames := spec.GetConfigNames()
	span.SetAttributes(attribute.StringSlice("app.request.config_names", configNames))

	// Get connections: use resolver (handles internal + external) or fallback to repo
	var (
		connections []*model.Connection
		err         error
	)

	if s.resolver != nil {
		connections, err = s.resolver.ResolveConnections(ctx, configNames)
		if err != nil {
			libOpentelemetry.HandleSpanError(span, "Failed to resolve connections", err)
			logger.Log(ctx, libLog.LevelError, "failed to resolve connections",
				libLog.Err(err),
			)

			return nil, pkg.ValidateInternalError(err, "schema")
		}
	} else {
		connections, err = s.connRepo.FindByConfigNames(ctx, configNames)
		if err != nil {
			libOpentelemetry.HandleSpanError(span, "Failed to find connections", err)
			logger.Log(ctx, libLog.LevelError, "failed to find connections",
				libLog.Err(err),
			)

			return nil, pkg.ValidateInternalError(err, "schema")
		}
	}

	if len(connections) == 0 {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "No connections found for the provided datasources", nil)

		return nil, pkg.ValidationError{
			EntityType: "schema",
			Code:       constant.ErrSchemaValidationNotFound.Error(),
			Title:      "No Connections Found",
			Message:    "No connections configured for the requested datasources",
		}
	}

	span.SetAttributes(attribute.Int("app.connections.found_count", len(connections)))

	// Map connections by config name for easy lookup during validation
	connMap := make(map[string]*model.Connection, len(connections))
	for _, conn := range connections {
		connMap[conn.ConfigName] = conn
	}

	var validationErrors []model.SchemaValidationError

	// Validate each datasource in the request against its schema
	for _, configName := range configNames {
		conn, found := connMap[configName]
		if !found {
			validationErrors = append(validationErrors, model.NewDataSourceNotFoundError(configName))
			logger.Log(ctx, libLog.LevelWarn, "datasource not found",
				libLog.String("config_name", configName),
			)

			continue
		}

		tables := spec.GetTablesByConfigName(configName)

		// tableNameReverseMap maps transformed names back to original names for error reporting
		var tableNameReverseMap map[string]string

		schemas := resolveSchemas(tables, conn)

		// Get or fetch schema for the connection
		schema, err := s.getOrFetchSchema(ctx, conn, schemas)
		if err != nil {
			if errors.Is(err, errDataSourceFactoryNotConfigured) {
				libOpentelemetry.HandleSpanError(span, "schema validation datasource factory misconfiguration", err)
				logger.Log(ctx, libLog.LevelError, "schema validation datasource factory misconfiguration",
					libLog.String("config_name", configName),
					libLog.Err(err),
				)

				return nil, pkg.ValidateInternalError(err, "schema")
			}

			validationErrors = append(validationErrors, model.NewDataSourceDownError(configName))
			logger.Log(ctx, libLog.LevelWarn, "failed to get schema",
				libLog.String("config_name", configName),
				libLog.Err(err),
			)

			continue
		}

		// Transform table names for plugin_crm using auto-discovery against the real schema.
		// Must happen AFTER getOrFetchSchema so we know the actual collection names.
		if configName == pluginCRMConfigName && schema != nil {
			tables, tableNameReverseMap = transformPluginCRMTablesFromSchema(tables, schema)
			logger.Log(ctx, libLog.LevelDebug, "transformed plugin_crm tables via schema auto-discovery",
				libLog.Any("tables", tables),
			)
		}

		// Validate against schema using transformed table names
		schemaErrors := validateTablesAgainstSchema(configName, tables, schema, tableNameReverseMap, conn.Type)
		validationErrors = append(validationErrors, schemaErrors...)
	}

	// Prepare response based on validation results
	var response *model.SchemaValidationResponse
	if len(validationErrors) == 0 {
		response = model.NewSuccessResponse()

		logger.Log(ctx, libLog.LevelInfo, "schema validation successful",
			libLog.Int("datasource_count", len(configNames)),
		)
	} else {
		response = model.NewFailureResponse(validationErrors)
		logger.Log(ctx, libLog.LevelWarn, "schema validation failed",
			libLog.Int("error_count", len(validationErrors)),
		)
	}

	span.SetAttributes(
		attribute.String("app.response.status", response.Status),
		attribute.Int("app.response.error_count", len(validationErrors)),
	)

	return response, nil
}

// getOrFetchSchema retrieves schema from cache or fetches from datasource.
func (s *ValidateSchema) getOrFetchSchema(
	ctx context.Context,
	conn *model.Connection,
	schemas []string,
) (*model.DataSourceSchema, error) {
	logger, tracer, _, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.validate_schema.get_or_fetch_schema")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.schema.config_name", conn.ConfigName),
		attribute.String("app.schema.database_type", string(conn.Type)),
	)

	// Check cache first
	cachedSchema, err := s.schemaCache.Get(ctx, conn.ConfigName)
	if err != nil {
		logger.Log(ctx, libLog.LevelWarn, "schema cache error",
			libLog.String("config_name", conn.ConfigName),
			libLog.Err(err),
		)
	}

	if cachedSchema != nil {
		span.SetAttributes(attribute.Bool("app.schema.cache_hit", true))
		logger.Log(ctx, libLog.LevelDebug, "schema cache hit", libLog.String("config_name", conn.ConfigName))

		return cachedSchema, nil
	}

	span.SetAttributes(attribute.Bool("app.schema.cache_hit", false))
	logger.Log(ctx, libLog.LevelDebug, "schema cache miss", libLog.String("config_name", conn.ConfigName))

	if s.dsFactory == nil {
		libOpentelemetry.HandleSpanError(span, "datasource factory not configured", errDataSourceFactoryNotConfigured)
		logger.Log(ctx, libLog.LevelError, "datasource factory not configured",
			libLog.String("config_name", conn.ConfigName),
		)

		return nil, errDataSourceFactoryNotConfigured
	}

	ds, err := s.dsFactory(ctx, conn, s.cryptor)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "failed to create datasource", err)
		return nil, fmt.Errorf("failed to create datasource: %w", err)
	}

	if isNilDataSource(ds) {
		factoryErr := fmt.Errorf("datasource factory returned nil datasource for config %s", conn.ConfigName)
		libOpentelemetry.HandleSpanError(span, "datasource factory returned nil", factoryErr)

		return nil, factoryErr
	}
	defer ds.Close(ctx)

	// Get schema info from datasource
	schema, err := ds.GetSchemaInfo(ctx, schemas)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "failed to get schema info", err)
		return nil, fmt.Errorf("failed to get schema info: %w", err)
	}

	if schema == nil {
		schema = model.NewDataSourceSchema(conn.ConfigName)
	}

	if schema.Tables == nil {
		schema.Tables = map[string]*model.TableSchema{}
	}

	// Cache the fetched schema for future requests
	if err := s.schemaCache.Set(ctx, conn.ConfigName, schema, cacheRepo.DefaultSchemaCacheTTL); err != nil {
		logger.Log(ctx, libLog.LevelWarn, "schema fetched but failed to cache",
			libLog.String("config_name", conn.ConfigName),
			libLog.Err(err),
		)
	} else {
		logger.Log(ctx, libLog.LevelDebug, "schema fetched and cached",
			libLog.String("config_name", conn.ConfigName),
			libLog.Int("table_count", len(schema.Tables)),
		)
	}

	span.SetAttributes(attribute.Int("app.schema.tables_count", len(schema.Tables)))

	return schema, nil
}

func isNilDataSource(ds datasourceModel.DataSource) bool {
	if ds == nil {
		return true
	}

	rv := reflect.ValueOf(ds)

	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

// validateTablesAgainstSchema validates tables against a DataSourceSchema.
// If reverseMap is provided, it uses original table names in error messages.
// The dbType parameter is used to normalize table names for databases with default schemas.
func validateTablesAgainstSchema(
	configName string,
	tables map[string][]string,
	schema *model.DataSourceSchema,
	reverseMap map[string]string,
	dbType model.DBType,
) []model.SchemaValidationError {
	var validationErrors []model.SchemaValidationError

	for tableName, fields := range tables {
		// Determine the display name for error messages
		displayName := tableName
		if reverseMap != nil {
			if original, exists := reverseMap[tableName]; exists {
				displayName = original
			}
		}

		// Normalize table name for lookup based on database type
		lookupName := normalizeTableNameForValidation(tableName, dbType)

		// Check if table exists in schema
		if !schema.HasTable(lookupName) {
			validationErrors = append(validationErrors, model.SchemaValidationError{
				Type:         model.ErrTypeTableNotFound,
				DataSourceID: configName,
				Table:        displayName,
			})

			continue
		}

		// Check if each field exists in the table schema
		for _, fieldName := range fields {
			lookupFieldName := normalizeFieldNameForValidation(fieldName, dbType)
			if !schema.HasField(lookupName, lookupFieldName) {
				validationErrors = append(validationErrors, model.SchemaValidationError{
					Type:         model.ErrTypeFieldNotFound,
					DataSourceID: configName,
					Table:        displayName,
					Field:        fieldName,
				})
			}
		}
	}

	return validationErrors
}

// normalizeTableNameForValidation normalizes a table name for schema lookup
// based on the database type. This handles cases where users request
// "dbo.users" but schema stores "users", or "SYSTEM.table" vs "table".
func normalizeTableNameForValidation(tableName string, dbType model.DBType) string {
	switch dbType {
	case model.TypeSQLServer:
		return schemautil.NormalizeTableNameForLookup(tableName, schemautil.DefaultSchemaSQLServer)
	case model.TypeOracle:
		// Oracle stores table names in lowercase after normalization
		// and uses the current user as default schema
		return strings.ToLower(tableName)
	case model.TypePostgreSQL:
		return schemautil.NormalizeTableNameForLookup(tableName, schemautil.DefaultSchemaPostgreSQL)
	default:
		return tableName
	}
}

// normalizeFieldNameForValidation normalizes a field name for schema lookup
// based on the database type. Oracle stores column names in UPPERCASE in its
// data dictionary (ALL_TAB_COLUMNS), but the Oracle datasource's GetSchemaInfo
// normalizes them to lowercase for case-insensitive matching.
func normalizeFieldNameForValidation(fieldName string, dbType model.DBType) string {
	switch dbType {
	case model.TypeOracle:
		// Oracle's GetSchemaInfo normalizes column names to lowercase.
		// User input like "ID" must be converted to "id" for lookup.
		return strings.ToLower(fieldName)
	default:
		// PostgreSQL, SQL Server, MySQL are case-insensitive for unquoted identifiers
		// but we store them in their original case, so no normalization needed.
		return fieldName
	}
}

// ensureDefaultSchema adds the default schema to the schemas list
// if any table name is unqualified (has no schema prefix with a dot).
// This ensures tables in the default schema are discoverable when mixed with schema-qualified tables.
func ensureDefaultSchema(tables map[string][]string, schemas []string, defaultSchema string) []string {
	// Check if any table has no dot (unqualified name)
	hasUnqualifiedTable := false

	for tableName := range tables {
		if !strings.Contains(tableName, ".") {
			hasUnqualifiedTable = true
			break
		}
	}

	// If there are unqualified tables, ensure default schema is included
	if hasUnqualifiedTable {
		defaultIncluded := false

		for _, s := range schemas {
			if s == defaultSchema {
				defaultIncluded = true
				break
			}
		}

		if !defaultIncluded {
			schemas = append(schemas, defaultSchema)
		}
	}

	return schemas
}

// ensureDefaultSchemaForPostgreSQL adds the default "public" schema to the schemas list
// if any table name is unqualified (has no schema prefix with a dot).
// This ensures tables in the public schema are discoverable when mixed with schema-qualified tables.
// resolveSchemas determines which database schemas to query based on connection type and config.
// Priority: metadata.schemas (explicit) > default per database type (public/dbo).
func resolveSchemas(tables map[string][]string, conn *model.Connection) []string {
	schemas := datasourceModel.GetUniqueSchemas(tables)

	switch conn.Type {
	case model.TypePostgreSQL:
		if conn.Metadata != nil {
			if s, ok := (*conn.Metadata)["schemas"].(string); ok && s != "" {
				for _, part := range strings.Split(s, ",") {
					trimmed := strings.TrimSpace(part)
					if trimmed != "" {
						schemas = append(schemas, trimmed)
					}
				}
			}
		}

		// Always ensure "public" is included when there are unqualified table
		// names (no dot), even if other schemas were already added from metadata.
		schemas = ensureDefaultSchemaForPostgreSQL(tables, schemas)
	case model.TypeSQLServer:
		schemas = ensureDefaultSchemaForSQLServer(tables, schemas)
	}

	return schemas
}

func ensureDefaultSchemaForPostgreSQL(tables map[string][]string, schemas []string) []string {
	return ensureDefaultSchema(tables, schemas, schemautil.DefaultSchemaPostgreSQL)
}

// ensureDefaultSchemaForSQLServer adds the default "dbo" schema to the schemas list
// if any table name is unqualified (has no schema prefix with a dot).
// This ensures tables in the dbo schema are discoverable when mixed with schema-qualified tables.
func ensureDefaultSchemaForSQLServer(tables map[string][]string, schemas []string) []string {
	return ensureDefaultSchema(tables, schemas, schemautil.DefaultSchemaSQLServer)
}

// transformPluginCRMTablesFromSchema transforms table names for plugin_crm datasource
// using auto-discovery against the real schema. For each logical name (e.g., "holders"),
// it finds the first real collection matching the prefix "holders_" in the schema.
// If the name already matches a real collection (e.g., "holders_06c4f684-..."), it passes through.
// Returns the transformed tables and a reverse map (transformed -> original) for error reporting.
func transformPluginCRMTablesFromSchema(tables map[string][]string, schema *model.DataSourceSchema) (map[string][]string, map[string]string) {
	transformed := make(map[string][]string, len(tables))
	reverseMap := make(map[string]string, len(tables))

	// Build a list of real collection names from the schema
	realCollections := make([]string, 0, len(schema.Tables))
	for tableName := range schema.Tables {
		realCollections = append(realCollections, tableName)
	}

	for tableName, fields := range tables {
		// Check if the table name already exists in the schema (full name passed)
		if schema.HasTable(tableName) {
			transformed[tableName] = fields
			reverseMap[tableName] = tableName

			continue
		}

		// Auto-discover: find the first collection matching prefix "tableName_"
		prefix := tableName + "_"
		found := false

		for _, realName := range realCollections {
			if strings.HasPrefix(realName, prefix) {
				transformed[realName] = fields
				reverseMap[realName] = tableName
				found = true

				break
			}
		}

		if !found {
			// No match — pass through as-is (will fail validation with TABLE_NOT_FOUND)
			transformed[tableName] = fields
			reverseMap[tableName] = tableName
		}
	}

	return transformed, reverseMap
}
