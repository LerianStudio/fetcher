package query

import (
	"context"
	"fmt"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/crypto"
	"github.com/LerianStudio/fetcher/pkg/datasource"
	"github.com/LerianStudio/fetcher/pkg/model"
	datasourceModel "github.com/LerianStudio/fetcher/pkg/model/datasource"
	connRepo "github.com/LerianStudio/fetcher/pkg/mongodb/connection"
	cacheRepo "github.com/LerianStudio/fetcher/pkg/repository/cache"

	"github.com/LerianStudio/lib-commons/v2/commons"
	libOpentelemetry "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

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
		libOpentelemetry.HandleSpanError(&span, "Invalid request payload", errValidation)
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
		libOpentelemetry.HandleSpanError(&span, "No connections found for the provided datasources", nil)

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

		// Returns schemas only if they exist
		schemas := datasourceModel.GetUniqueSchemas(tables)

		// Get or fetch schema for the connection
		schema, err := s.getOrFetchSchema(ctx, conn, schemas)
		if err != nil {
			validationErrors = append(validationErrors, model.NewDataSourceDownError(configName))
			logger.Warnf("failed to get schema config_name=%s org=%s: %v", configName, organizationID, err)

			continue
		}

		// Validate against schema
		schemaErrors := spec.ValidateAgainstSchema(configName, schema)
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
