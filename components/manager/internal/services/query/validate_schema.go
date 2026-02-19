package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/LerianStudio/fetcher/pkg"
	cacheRepo "github.com/LerianStudio/fetcher/pkg/ports/cache"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/datasource"
	"github.com/LerianStudio/fetcher/pkg/model"
	datasourceModel "github.com/LerianStudio/fetcher/pkg/model/datasource"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/pkg/schemautil"

	"github.com/LerianStudio/lib-commons/v2/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

const (
	// pluginCRMConfigName is the config name that requires special table name transformation.
	pluginCRMConfigName = "plugin_crm"
)

// pluginCRMTableMapping maps user-friendly table names to actual database collection names.
// The actual collection name in the database is: <mapped_name>_<organization_id>
var pluginCRMTableMapping = map[string]string{
	"holders": "holders",
	"aliases": "aliases",
}

// ValidateSchema is the query service for validating schema references.
type ValidateSchema struct {
	connRepo    connRepo.Repository
	cryptor     crypto.Cryptor
	schemaCache cacheRepo.SchemaCacheRepository
}

// NewValidateSchema creates a new ValidateSchema service without rate limiting.
func NewValidateSchema(
	connectionRepo connRepo.Repository,
	cryptor crypto.Cryptor,
	schemaCache cacheRepo.SchemaCacheRepository,
) *ValidateSchema {
	return &ValidateSchema{
		connRepo:    connectionRepo,
		cryptor:     cryptor,
		schemaCache: schemaCache,
	}
}

// Execute validates schema references against configured datasources.
func (s *ValidateSchema) Execute(
	ctx context.Context,
	organizationID uuid.UUID,
	request model.SchemaValidationRequest,
) (*model.SchemaValidationResponse, error) {
	logger, tracer, reqID, _ := commons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.validate_schema")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.organization_id", organizationID.String()),
		attribute.Int("app.request.datasource_count", len(request.MappedFields)),
	)

	if err := libOpentelemetry.SetSpanAttributesFromStruct(&span, "app.request.payload", request.ToMapWithMask()); err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to convert request to JSON string", err)
	}

	// Create validation spec from request
	spec := model.NewSchemaValidationSpec(request)

	// Validate request payload structure and limits
	if errValidation := spec.Validate(); errValidation != nil {
		libOpentelemetry.HandleSpanBusinessErrorEvent(&span, "Invalid request payload", errValidation)
		logger.Warnf("schema validation request invalid org=%s: %v", organizationID, errValidation)

		return nil, errValidation
	}

	// Fetch connections for the requested config names from the repository
	configNames := spec.GetConfigNames()
	span.SetAttributes(attribute.StringSlice("app.request.config_names", configNames))

	// Get connections from repository by config names
	connections, err := s.connRepo.FindByConfigNames(ctx, organizationID, configNames)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "Failed to find connections", err)
		logger.Errorf("failed to find connections org=%s: %v", organizationID, err)

		return nil, pkg.ValidateInternalError(err, "schema")
	}

	if len(connections) == 0 {
		libOpentelemetry.HandleSpanBusinessErrorEvent(&span, "No connections found for the provided datasources", nil)

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
			logger.Warnf("datasource not found config_name=%s org=%s", configName, organizationID)

			continue
		}

		tables := spec.GetTablesByConfigName(configName)

		// tableNameReverseMap maps transformed names back to original names for error reporting
		var tableNameReverseMap map[string]string

		// Transform table names for plugin_crm to include organization ID suffix
		if configName == pluginCRMConfigName {
			tables, tableNameReverseMap = transformPluginCRMTables(tables, organizationID)
			logger.Debugf("transformed plugin_crm tables org=%s tables=%v", organizationID, tables)
		}

		// Returns schemas only if they exist
		schemas := datasourceModel.GetUniqueSchemas(tables)

		// For PostgreSQL, ensure the default "public" schema is included
		// when there are unqualified table names (tables without a dot)
		if conn.Type == model.TypePostgreSQL {
			schemas = ensureDefaultSchemaForPostgreSQL(tables, schemas)
		}

		// For SQL Server, ensure the default "dbo" schema is included
		// when there are unqualified table names (tables without a dot)
		if conn.Type == model.TypeSQLServer {
			schemas = ensureDefaultSchemaForSQLServer(tables, schemas)
		}

		// Get or fetch schema for the connection
		schema, err := s.getOrFetchSchema(ctx, conn, schemas)
		if err != nil {
			validationErrors = append(validationErrors, model.NewDataSourceDownError(configName))
			logger.Warnf("failed to get schema config_name=%s org=%s: %v", configName, organizationID, err)

			continue
		}

		// Validate against schema using transformed table names
		schemaErrors := validateTablesAgainstSchema(configName, tables, schema, tableNameReverseMap, conn.Type)
		validationErrors = append(validationErrors, schemaErrors...)
	}

	// Prepare response based on validation results
	var response *model.SchemaValidationResponse
	if len(validationErrors) == 0 {
		response = model.NewSuccessResponse()

		logger.Infof("schema validation successful org=%s datasources=%d", organizationID, len(configNames))
	} else {
		response = model.NewFailureResponse(validationErrors)
		logger.Warnf("schema validation failed org=%s errors=%d", organizationID, len(validationErrors))
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
		logger.Warnf("cache error for config_name=%s: %v", conn.ConfigName, err)
	}

	if cachedSchema != nil {
		span.SetAttributes(attribute.Bool("app.schema.cache_hit", true))
		logger.Debugf("schema cache hit config_name=%s", conn.ConfigName)

		return cachedSchema, nil
	}

	span.SetAttributes(attribute.Bool("app.schema.cache_hit", false))
	logger.Debugf("schema cache miss config_name=%s", conn.ConfigName)

	// Get datasource instance because schema is not in cache
	ds, err := datasource.NewDataSourceFromConnection(ctx, conn, s.cryptor, logger)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to create datasource", err)
		return nil, fmt.Errorf("failed to create datasource: %w", err)
	}
	defer ds.Close(ctx)

	// Get schema info from datasource
	schema, err := ds.GetSchemaInfo(ctx, schemas)
	if err != nil {
		libOpentelemetry.HandleSpanError(&span, "failed to get schema info", err)
		return nil, fmt.Errorf("failed to get schema info: %w", err)
	}

	// Cache the fetched schema for future requests
	if err := s.schemaCache.Set(ctx, conn.ConfigName, schema, cacheRepo.DefaultSchemaCacheTTL); err != nil {
		logger.Warnf("failed to cache schema config_name=%s: %v", conn.ConfigName, err)
	}

	span.SetAttributes(attribute.Int("app.schema.tables_count", len(schema.Tables)))
	logger.Debugf("schema fetched and cached config_name=%s tables=%d", conn.ConfigName, len(schema.Tables))

	return schema, nil
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
	var errors []model.SchemaValidationError

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
			errors = append(errors, model.SchemaValidationError{
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
				errors = append(errors, model.SchemaValidationError{
					Type:         model.ErrTypeFieldNotFound,
					DataSourceID: configName,
					Table:        displayName,
					Field:        fieldName,
				})
			}
		}
	}

	return errors
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
func ensureDefaultSchemaForPostgreSQL(tables map[string][]string, schemas []string) []string {
	return ensureDefaultSchema(tables, schemas, schemautil.DefaultSchemaPostgreSQL)
}

// ensureDefaultSchemaForSQLServer adds the default "dbo" schema to the schemas list
// if any table name is unqualified (has no schema prefix with a dot).
// This ensures tables in the dbo schema are discoverable when mixed with schema-qualified tables.
func ensureDefaultSchemaForSQLServer(tables map[string][]string, schemas []string) []string {
	return ensureDefaultSchema(tables, schemas, schemautil.DefaultSchemaSQLServer)
}

// transformPluginCRMTables transforms table names for plugin_crm datasource.
// It maps user-friendly names (e.g., "holders") to actual database collection names
// with organization ID suffix (e.g., "holders_<org_id>").
// Returns the transformed tables and a reverse map (transformed -> original) for error reporting.
func transformPluginCRMTables(tables map[string][]string, organizationID uuid.UUID) (map[string][]string, map[string]string) {
	transformed := make(map[string][]string, len(tables))
	reverseMap := make(map[string]string, len(tables))

	for tableName, fields := range tables {
		var actualName string
		// Check if we have a mapping for this table name
		if mappedName, exists := pluginCRMTableMapping[tableName]; exists {
			// Transform to actual collection name: <mapped_name>_<org_id>
			actualName = mappedName + "_" + organizationID.String()
		} else {
			// No mapping found, use table name with org_id suffix as fallback
			actualName = tableName + "_" + organizationID.String()
		}

		transformed[actualName] = fields
		reverseMap[actualName] = tableName
	}

	return transformed, reverseMap
}
