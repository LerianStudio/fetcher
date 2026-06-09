package query

import (
	"context"
	"errors"

	observability "github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/engine"
	plugincrm "github.com/LerianStudio/fetcher/pkg/enginecompat/plugincrm"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/schemacompat"
	"github.com/LerianStudio/fetcher/pkg/enginecompat/tablenorm"
	"github.com/LerianStudio/fetcher/pkg/model"
	connRepo "github.com/LerianStudio/fetcher/pkg/ports/connection"
	"github.com/LerianStudio/fetcher/pkg/resolver"

	libLog "github.com/LerianStudio/lib-observability/log"
	libOpentelemetry "github.com/LerianStudio/lib-observability/tracing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ValidateSchema is the query service for validating schema references.
//
// Schema DISCOVERY for each resolved connection is delegated to the Engine
// (cache-first via the host-wired SchemaCache port + connector). Connection
// RESOLUTION (internal datasources via the resolver, external via the repository),
// per-table/field VALIDATION, DB-type NORMALIZATION, request LIMIT enforcement
// (spec.Validate), and plugin_crm policy remain HOST concerns: the Engine
// validator (canonical ValidationReport) is intentionally NOT used here so the
// Manager's public response shape and internal-datasource support stay
// byte-identical. The Engine owns the part with real infrastructure coupling —
// schema discovery and caching.
type ValidateSchema struct {
	connRepo connRepo.Repository
	engine   *engine.Engine              // schema discovery authority (cache + connector)
	resolver resolver.ConnectionResolver // nil-safe: if nil, uses connRepo only
}

// NewValidateSchema creates a new ValidateSchema service.
func NewValidateSchema(
	connectionRepo connRepo.Repository,
	eng *engine.Engine,
	connResolver resolver.ConnectionResolver,
) *ValidateSchema {
	return &ValidateSchema{
		connRepo: connectionRepo,
		engine:   eng,
		resolver: connResolver,
	}
}

// Execute validates schema references against configured datasources.
func (s *ValidateSchema) Execute(
	ctx context.Context,
	request model.SchemaValidationRequest,
) (*model.SchemaValidationResponse, error) {
	logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.validate_schema")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.Int("app.request.datasource_count", len(request.MappedFields)),
	)

	if err := libOpentelemetry.SetSpanAttributesFromValue(span, "app.request.payload", request.ToMapWithMask(), nil); err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to convert request to JSON string", err)
	}

	// Create validation spec from request.
	spec := model.NewSchemaValidationSpec(request)

	// Validate request payload structure and limits. This is the SOLE limit gate:
	// an over-limit request (datasources/tables/fields) surfaces here as the
	// ErrSchemaValidationLimit business error (HTTP 422) BEFORE any discovery,
	// preserving the legacy contract without relying on the Engine's report.
	if errValidation := spec.Validate(); errValidation != nil {
		libOpentelemetry.HandleSpanBusinessErrorEvent(span, "Invalid request payload", errValidation)
		logger.Log(ctx, libLog.LevelWarn, "schema validation request invalid",
			libLog.Err(errValidation),
		)

		return nil, errValidation
	}

	// Fetch connections for the requested config names.
	configNames := spec.GetConfigNames()
	span.SetAttributes(attribute.StringSlice("app.request.config_names", configNames))

	connections, err := s.resolveConnections(ctx, span, configNames)
	if err != nil {
		return nil, err
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

	// Map connections by config name for easy lookup during validation.
	connMap := make(map[string]*model.Connection, len(connections))
	for _, conn := range connections {
		connMap[conn.ConfigName] = conn
	}

	validationErrors, err := s.validateConfigs(ctx, span, configNames, connMap, spec)
	if err != nil {
		return nil, err
	}

	// Prepare response based on validation results.
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

// resolveConnections resolves the requested config names through the resolver
// (internal + external datasources) or, when no resolver is wired, the repository
// directly. A resolution error is mapped to the schema internal error.
func (s *ValidateSchema) resolveConnections(
	ctx context.Context,
	span trace.Span,
	configNames []string,
) ([]*model.Connection, error) {
	logger := observability.NewLoggerFromContext(ctx)

	if s.resolver != nil {
		connections, err := s.resolver.ResolveConnections(ctx, configNames)
		if err != nil {
			libOpentelemetry.HandleSpanError(span, "Failed to resolve connections", err)
			logger.Log(ctx, libLog.LevelError, "failed to resolve connections", libLog.Err(err))

			return nil, pkg.ValidateInternalError(err, "schema")
		}

		return connections, nil
	}

	connections, err := s.connRepo.FindByConfigNames(ctx, configNames)
	if err != nil {
		libOpentelemetry.HandleSpanError(span, "Failed to find connections", err)
		logger.Log(ctx, libLog.LevelError, "failed to find connections", libLog.Err(err))

		return nil, pkg.ValidateInternalError(err, "schema")
	}

	return connections, nil
}

// validateConfigs validates each requested datasource against its discovered
// schema, collecting per-datasource validation errors. A typed host-safety
// rejection (e.g. FET-0414) discovered during schema discovery is surfaced as a
// top-level error rather than buried as a per-datasource warning.
func (s *ValidateSchema) validateConfigs(
	ctx context.Context,
	span trace.Span,
	configNames []string,
	connMap map[string]*model.Connection,
	spec *model.SchemaValidationSpec,
) ([]model.SchemaValidationError, error) {
	logger := observability.NewLoggerFromContext(ctx)

	var validationErrors []model.SchemaValidationError

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
		schemas := schemaScopeForConfig(conn, tables)

		// ValidateSchema stays CACHE-FIRST (forceRefresh=false): a cache hit
		// short-circuits live discovery and a miss writes through, unchanged across
		// the embedded-Engine migration. Only GET .../schema is always-fresh.
		schema, err := discoverSchemaViaEngine(ctx, s.engine, conn, schemas, false)
		if err != nil {
			// Preserve typed validation errors (e.g. FET-0414 host safety) as
			// top-level errors: burying them as per-datasource warnings would yield
			// 200 + DataSourceDownError, dropping the audit signal and breaking the
			// FET-0414 -> HTTP 400 contract.
			var ve pkg.ValidationError
			if errors.As(err, &ve) {
				libOpentelemetry.HandleSpanError(span, "schema validation rejected by host safety guard", err)
				logger.Log(ctx, libLog.LevelWarn, "schema validation rejected by host safety guard",
					libLog.String("config_name", configName),
					libLog.Err(err),
				)

				return nil, err
			}

			validationErrors = append(validationErrors, model.NewDataSourceDownError(configName))
			logger.Log(ctx, libLog.LevelWarn, "failed to get schema",
				libLog.String("config_name", configName),
				libLog.Err(err),
			)

			continue
		}

		// tableNameReverseMap maps transformed names back to original names for
		// error reporting (used by the plugin_crm compatibility mapping).
		var tableNameReverseMap map[string]string

		// Transform table names for plugin_crm using auto-discovery against the real
		// schema, via the explicit, product-scoped host compatibility adapter. The
		// adapter is a NO-OP for any non-CRM source, so CRM policy never executes for
		// a generic datasource. Must happen AFTER discovery so the real collection
		// names are known.
		if plugincrm.IsPluginCRM(configName) && schema != nil {
			tables, tableNameReverseMap = plugincrm.MapTablesForCRMCompatibility(configName, tables, snapshotForCRM(configName, schema))
			logger.Log(ctx, libLog.LevelDebug, "transformed plugin_crm tables via schema auto-discovery",
				libLog.Any("tables", tables),
			)
		}

		schemaErrors := validateTablesAgainstSchema(configName, tables, schema, tableNameReverseMap, conn.Type)
		validationErrors = append(validationErrors, schemaErrors...)
	}

	return validationErrors, nil
}

// schemaScopeForConfig computes the schema-name list discovery should fetch for
// a connection, injecting the default schema for unqualified table names per
// database type. It delegates to the SINGLE type-aware helper
// tablenorm.SchemaScopeForTables (also used by the Worker extraction path) so the
// unqualified-table + default-schema rule has one implementation across both seams.
func schemaScopeForConfig(conn *model.Connection, tables map[string][]string) []string {
	return tablenorm.SchemaScopeForTables(conn.Type, tables)
}

// snapshotForCRM projects the discovered host schema into the Engine snapshot
// shape the plugin_crm compatibility adapter consumes. It is a thin, host-side
// projection over the single forward builder with NO filtering and NO
// normalization — the CRM adapter performs collection auto-discovery against the
// literal collection names.
func snapshotForCRM(configName string, schema *model.DataSourceSchema) engine.SchemaSnapshot {
	return schemacompat.BuildSnapshot(configName, "", schema, schemacompat.SnapshotOptions{})
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
		// Determine the display name for error messages.
		displayName := tableName
		if reverseMap != nil {
			if original, exists := reverseMap[tableName]; exists {
				displayName = original
			}
		}

		// Normalize table name for lookup based on database type.
		lookupName := normalizeTableNameForValidation(tableName, dbType)

		// Check if table exists in schema.
		if !schema.HasTable(lookupName) {
			validationErrors = append(validationErrors, model.SchemaValidationError{
				Type:         model.ErrTypeTableNotFound,
				DataSourceID: configName,
				Table:        displayName,
			})

			continue
		}

		// Check if each field exists in the table schema.
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

// normalizeTableNameForValidation normalizes a table name for schema lookup based on
// the database type, delegating to the SINGLE canonicalizer tablenorm.NormalizeTable
// (also used by the Manager discovery snapshot and the Worker extraction path). This
// strips default-schema prefixes (public./dbo.) for PostgreSQL/SQLServer and folds
// Oracle identifiers to UPPERCASE — matching the UPPERCASE Manager snapshot, the
// physical Oracle catalog, and the extracted result keys. One source of truth keeps
// the validation lookup and the snapshot it queries from ever diverging.
func normalizeTableNameForValidation(tableName string, dbType model.DBType) string {
	return tablenorm.NormalizeTable(dbType, tableName)
}

// normalizeFieldNameForValidation normalizes a field name for schema lookup based on
// the database type, delegating to the SINGLE canonicalizer tablenorm.NormalizeField.
// Oracle folds to UPPERCASE (matching the physical ALL_TAB_COLUMNS catalog, the
// UPPERCASE snapshot, and the extracted result column keys); PostgreSQL, SQL Server,
// and MySQL are left in their original case (case-sensitive lookup, no fold).
func normalizeFieldNameForValidation(fieldName string, dbType model.DBType) string {
	return tablenorm.NormalizeField(dbType, fieldName)
}
