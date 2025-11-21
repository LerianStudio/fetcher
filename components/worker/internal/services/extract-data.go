package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/constant"
	"github.com/LerianStudio/fetcher/pkg/model"
	libCommons "github.com/LerianStudio/lib-commons/v2/commons"
	libCrypto "github.com/LerianStudio/lib-commons/v2/commons/crypto"
	"github.com/LerianStudio/lib-commons/v2/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v2/commons/opentelemetry"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ExtractExternalDataMessage contains the information needed to extract external data
type ExtractExternalDataMessage struct {
	// JobID is the unique identifier of the job extract.
	JobID uuid.UUID `json:"jobId"`

	// DataQueries maps database names to tables and their fields.
	// Format: map[databaseName]map[tableName][]fieldName.
	// Example: {"onboarding": {"organization": ["name"], "ledger": ["id"]}}.
	MappedFields map[string]map[string][]string `json:"mappedFields"`

	// Filters specify advanced filtering criteria using FilterCondition for complex queries.
	// Format: map[databaseName]map[tableName]map[fieldName]model.FilterCondition
	// Example: {"db": {"table": {"created_at": {"gte": ["2025-06-01"], "lte": ["2025-06-30"]}}}}
	Filters map[string]map[string]map[string]model.FilterCondition `json:"filters"`

	// Metadata contains additional metadata for the report.
	Metadata map[string]any `json:"metadata"`
}

// ExtractExternalData handles the extraction of data from external sources.
func (uc *UseCase) ExtractExternalData(ctx context.Context, body []byte) error {
	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.extract_external_data")
	defer span.End()

	message, err := uc.parseMessage(ctx, body, &span, logger)
	if err != nil {
		return err
	}

	//ATUALIZAR: Ajustar quando tiver o job entity
	//if skip := uc.shouldSkipProcessing(ctx, message.ReportID, logger); skip {
	//	return nil
	//}

	dataSourcesConnections := []pkg.DataSourceConfig{
		{
			ID:                "a059d33d-2e9c-4087-94e7-8f7ec0e42c2f",
			OrganizationID:    "06c4f684-19b0-449a-81f4-f9a4e503db83",
			ConfigName:        "midaz_onboarding",
			Type:              "postgresql",
			Host:              "localhost",
			Port:              "5702",
			DatabaseName:      "onboarding",
			Username:          "midaz",
			PasswordEncrypted: "lerian",
			SSL:               nil,
		},
		{
			ID:                "a059d33d-2e9c-4087-94e7-8f7ec0e42c2f",
			OrganizationID:    "06c4f684-19b0-449a-81f4-f9a4e503db83",
			ConfigName:        "midaz_transaction",
			Type:              "postgresql",
			Host:              "localhost",
			Port:              "5702",
			DatabaseName:      "transaction",
			Username:          "midaz",
			PasswordEncrypted: "lerian",
			SSL:               nil,
		},
		{
			ID:                "a059d33d-2e9c-4087-94e7-8f7ec0e42c2g",
			OrganizationID:    "06c4f684-19b0-449a-81f4-f9a4e503db83",
			ConfigName:        "plugin_crm",
			Type:              "mongodb",
			Host:              "localhost",
			Port:              "5706",
			DatabaseName:      "plugin-crm-db",
			Username:          "plugin-crm",
			PasswordEncrypted: "lerian",
			SSL:               nil,
		},
	}

	result := make(map[string]map[string][]map[string]any)

	if err := uc.queryExternalData(ctx, message, dataSourcesConnections, result); err != nil {
		return uc.handleErrorWithUpdate(ctx, message.JobID, &span, "Error querying external data", err, logger)
	}

	if err := uc.saveExternalDataToSeaweedFS(ctx, tracer, message, result, &span, logger); err != nil {
		return fmt.Errorf("saveExternalDataToSeaweedFS: %w", err)
	}

	return nil
}

// parseMessage parses the RabbitMQ message body into ExtractExternalDataMessage struct.
func (uc *UseCase) parseMessage(ctx context.Context, body []byte, span *trace.Span, logger log.Logger) (ExtractExternalDataMessage, error) {
	var message ExtractExternalDataMessage

	err := json.Unmarshal(body, &message)
	if err != nil {
		// ATUALIZAR: ADICIONAR UPDATE STATUS DO JOB
		//if errUpdate := uc.updateReportWithErrors(ctx, message, err.Error()); errUpdate != nil {
		//	libOtel.HandleSpanError(span, "Error to update report status with error.", errUpdate)
		//	logger.Errorf("Error update report status with error: %s", errUpdate.Error())
		//
		//	return message, errUpdate
		//}

		libOtel.HandleSpanError(span, "Error unmarshalling message.", err)
		logger.Errorf("Error unmarshalling message: %s", err.Error())

		return message, err
	}

	return message, nil
}

//// renderTemplate renders the template with data from external sources.
//func (uc *UseCase) renderTemplate(ctx context.Context, tracer trace.Tracer, templateBytes []byte, result map[string]map[string][]map[string]any, message ExtractExternalDataMessage, span *trace.Span, logger log.Logger) (string, error) {
//	ctx, spanRender := tracer.Start(ctx, "service.generate_report.render_template")
//	defer spanRender.End()
//
//	renderer := pongo.NewTemplateRenderer()
//
//	out, err := renderer.RenderFromBytes(ctx, templateBytes, result, logger)
//	if err != nil {
//		if errUpdate := uc.updateReportWithErrors(ctx, message.ReportID, err.Error()); errUpdate != nil {
//			libOtel.HandleSpanError(span, "Error to update report status with error.", errUpdate)
//			logger.Errorf("Error update report status with error: %s", errUpdate.Error())
//
//			return "", errUpdate
//		}
//
//		libOtel.HandleSpanError(&spanRender, "Error rendering template.", err)
//		logger.Errorf("Error rendering template: %s", err.Error())
//
//		return "", err
//	}
//
//	return out, nil
//}
//
//// markReportAsFinished updates report status to finished.
//func (uc *UseCase) markReportAsFinished(ctx context.Context, reportID uuid.UUID, span *trace.Span, logger log.Logger) error {
//	err := uc.ReportDataRepo.UpdateReportStatusById(ctx, constant.FinishedStatus, reportID, time.Now(), nil)
//	if err != nil {
//		if errUpdate := uc.updateReportWithErrors(ctx, reportID, err.Error()); errUpdate != nil {
//			libOtel.HandleSpanError(span, "Error to update report status with error.", errUpdate)
//			logger.Errorf("Error update report status with error: %s", errUpdate.Error())
//
//			return errUpdate
//		}
//
//		libOtel.HandleSpanError(span, "Error to update report status.", err)
//		logger.Errorf("Error saving report: %s", err.Error())
//
//		return err
//	}
//
//	return nil
//}

// handleErrorWithUpdate logs error and updates report status to error.
func (uc *UseCase) handleErrorWithUpdate(ctx context.Context, jobID uuid.UUID, span *trace.Span, errorMsg string, err error, logger log.Logger) error {
	if errUpdate := uc.updateJobWithErrors(ctx, jobID, err.Error()); errUpdate != nil {
		libOtel.HandleSpanError(span, "Error to update report status with error.", errUpdate)
		logger.Errorf("Error update report status with error: %s", errUpdate.Error())

		return errUpdate
	}

	libOtel.HandleSpanError(span, errorMsg, err)
	logger.Errorf("%s: %s", errorMsg, err.Error())

	return err
}

// updateJobWithErrors updates the status of a job to "Error" with the provided error message.
func (uc *UseCase) updateJobWithErrors(ctx context.Context, reportId uuid.UUID, errorMessage string) error {
	metadata := make(map[string]any)
	metadata["error"] = errorMessage

	//errUpdate := uc.ReportDataRepo.UpdateReportStatusById(ctx, constant.ErrorStatus,
	//	reportId, time.Now(), metadata)
	//if errUpdate != nil {
	//	return errUpdate
	//}

	return nil
}

// queryExternalData retrieves data from external data sources specified in the message and populates the result map.
func (uc *UseCase) queryExternalData(ctx context.Context, message ExtractExternalDataMessage, dataSourcesConnections []pkg.DataSourceConfig, result map[string]map[string][]map[string]any) error {
	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.extract_external_data.query_external_data")
	defer span.End()

	for databaseName, tables := range message.MappedFields {
		if err := uc.queryDatabase(ctx, databaseName, tables, dataSourcesConnections, message.Filters, result, logger, tracer); err != nil {
			return err
		}
	}

	return nil
}

// queryDatabase handles data retrieval for a specific database
func (uc *UseCase) queryDatabase(
	ctx context.Context,
	databaseName string,
	tables map[string][]string,
	dataSourcesConnections []pkg.DataSourceConfig,
	allFilters map[string]map[string]map[string]model.FilterCondition,
	result map[string]map[string][]map[string]any,
	logger log.Logger,
	tracer trace.Tracer,
) error {
	ctx, dbSpan := tracer.Start(ctx, "service.extract_external_data.query_external_data.database")
	defer dbSpan.End()

	logger.Infof("Querying database %s", databaseName)

	// Check circuit breaker state before attempting query
	//if !uc.CircuitBreakerManager.IsHealthy(databaseName) {
	//	cbState := uc.CircuitBreakerManager.GetState(databaseName)
	//	err := fmt.Errorf("datasource %s is unhealthy - circuit breaker state: %s", databaseName, cbState)
	//	libOtel.HandleSpanError(&dbSpan, "Circuit breaker blocking request", err)
	//	logger.Errorf("⚠️  Circuit breaker blocking request to datasource %s (state: %s)", databaseName, cbState)
	//
	//	return err
	//}

	var dataSource pkg.DataSourceConfig
	for _, data := range dataSourcesConnections {
		if data.ConfigName == databaseName {
			dataSource = data
		}
	}

	// Check if datasource is marked as unavailable from initialization
	//if dataSource.Status == libConstants.DataSourceStatusUnavailable {
	//	err := fmt.Errorf("datasource %s is unavailable (initialization failed)", databaseName)
	//	libOtel.HandleSpanError(&dbSpan, "Datasource unavailable", err)
	//	logger.Errorf("⚠️  Datasource %s is unavailable - last error: %v", databaseName, dataSource.LastError)
	//
	//	return err
	//}

	// Attempt to connect
	if err := pkg.ConnectToDataSource(&dataSource, logger); err != nil {
		libOtel.HandleSpanError(&dbSpan, "Error initializing database connection.", err)
		return err
	}

	// Prepare a result map for this database
	if _, databaseExists := result[databaseName]; !databaseExists {
		result[databaseName] = make(map[string][]map[string]any)
	}

	databaseFilters := allFilters[databaseName]

	switch dataSource.Type {
	case constant.PostgreSQLType:
		return uc.queryPostgresDatabase(ctx, &dataSource, databaseName, tables, databaseFilters, result, logger)
	case constant.MongoDBType:
		return uc.queryMongoDatabase(ctx, &dataSource, databaseName, tables, databaseFilters, result, logger)
	default:
		return fmt.Errorf("unsupported database type: %s for database: %s", dataSource.Type, databaseName)
	}
}

// queryPostgresDatabase handles querying PostgresSQL databases
func (uc *UseCase) queryPostgresDatabase(
	ctx context.Context,
	dataSource *pkg.DataSourceConfig,
	databaseName string,
	tables map[string][]string,
	databaseFilters map[string]map[string]model.FilterCondition,
	result map[string]map[string][]map[string]any,
	logger log.Logger,
) error {
	// Execute schema query with circuit breaker protection
	//schemaResult, err := uc.CircuitBreakerManager.Execute(databaseName, func() (any, error) {
	//	return dataSource.PostgresRepository.GetDatabaseSchema(ctx)
	//})
	schemaResult, err := dataSource.PostgresRepository.GetDatabaseSchema(ctx)
	if err != nil {
		logger.Errorf("Error getting database schema for %s (circuit breaker): %s", databaseName, err.Error())
		return err
	}

	//schema := schemaResult.([]postgres.TableSchema)

	for table, fields := range tables {
		tableFilters := getTableFilters(databaseFilters, table)

		// Execute query with circuit breaker protection
		var (
			tableResult []map[string]any
			queryResult any
			errQuery    error
		)

		//queryResult, err := uc.CircuitBreakerManager.Execute(databaseName, func() (any, error) {
		//	if len(tableFilters) > 0 {
		//		return dataSource.PostgresRepository.QueryWithAdvancedFilters(ctx, schema, table, fields, tableFilters)
		//	}
		//
		//	return dataSource.PostgresRepository.Query(ctx, schema, table, fields, nil)
		//})

		if len(tableFilters) > 0 {
			queryResult, errQuery = dataSource.PostgresRepository.QueryWithAdvancedFilters(ctx, schemaResult, table, fields, tableFilters)
		} else {
			queryResult, errQuery = dataSource.PostgresRepository.Query(ctx, schemaResult, table, fields, nil)
		}

		if errQuery != nil {
			logger.Errorf("Error querying table %s in %s: %s", table, databaseName, errQuery.Error())
			return errQuery
		}

		tableResult = queryResult.([]map[string]any)

		//if len(tableFilters) > 0 {
		//	logger.Infof("Successfully queried table %s with advanced filters (circuit breaker: %s)",
		//		table, uc.CircuitBreakerManager.GetState(databaseName))
		//} else {
		//	logger.Infof("Successfully queried table %s (circuit breaker: %s)",
		//		table, uc.CircuitBreakerManager.GetState(databaseName))
		//}

		result[databaseName][table] = tableResult
	}

	return nil
}

// getTableFilters extracts filters for a specific table/collection
func getTableFilters(databaseFilters map[string]map[string]model.FilterCondition, tableName string) map[string]model.FilterCondition {
	if databaseFilters == nil {
		return nil
	}

	return databaseFilters[tableName]
}

// saveExternalDataToSeaweedFS converts the result map to JSON and saves it to SeaweedFS storage.
func (uc *UseCase) saveExternalDataToSeaweedFS(
	ctx context.Context,
	tracer trace.Tracer,
	message ExtractExternalDataMessage,
	result map[string]map[string][]map[string]any,
	span *trace.Span,
	logger log.Logger,
) error {
	ctx, spanSave := tracer.Start(ctx, "service.extract_external_data.save_external_data")
	defer spanSave.End()

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		libOtel.HandleSpanError(span, "Error marshalling result to JSON", err)
		logger.Errorf("Error marshalling result to JSON: %s", err.Error())
		return fmt.Errorf("marshalling result to JSON: %w", err)
	}

	objectName := fmt.Sprintf("%s.json", message.JobID.String())
	contentType := "application/json"

	if err := uc.ExternalDataSeaweedFS.Put(ctx, objectName, contentType, jsonData); err != nil {
		libOtel.HandleSpanError(span, "Error saving external data to SeaweedFS", err)
		logger.Errorf("Error saving external data to SeaweedFS: %s", err.Error())
		return fmt.Errorf("saving external data to SeaweedFS: %w", err)
	}

	logger.Infof("Successfully saved external data to SeaweedFS: %s", objectName)

	return nil
}

// queryMongoDatabase handles querying MongoDB databases
func (uc *UseCase) queryMongoDatabase(
	ctx context.Context,
	dataSource *pkg.DataSourceConfig,
	databaseName string,
	collections map[string][]string,
	databaseFilters map[string]map[string]model.FilterCondition,
	result map[string]map[string][]map[string]any,
	logger log.Logger,
) error {
	_, tracer, reqId, _ := libCommons.NewTrackingFromContext(ctx)

	_, span := tracer.Start(ctx, "service.extract_external_data.query_mongo_database")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqId),
		attribute.String("app.request.database_name", databaseName),
	)

	for collection, fields := range collections {
		collectionFilters := getTableFilters(databaseFilters, collection)

		if err := uc.processMongoCollection(ctx, dataSource, databaseName, collection, fields, collectionFilters, collections, result, logger); err != nil {
			return err
		}
	}

	return nil
}

// processMongoCollection processes a single MongoDB collection
func (uc *UseCase) processMongoCollection(
	ctx context.Context,
	dataSource *pkg.DataSourceConfig,
	databaseName, collection string,
	fields []string,
	collectionFilters map[string]model.FilterCondition,
	allCollections map[string][]string,
	result map[string]map[string][]map[string]any,
	logger log.Logger,
) error {
	// Handle plugin_crm special case
	if databaseName == "plugin_crm" && collection != "organization" {
		return uc.processPluginCRMCollection(ctx, dataSource, collection, fields, collectionFilters, allCollections, result, logger)
	}

	// Handle regular collections
	return uc.processRegularMongoCollection(ctx, dataSource, collection, fields, collectionFilters, result, logger)
}

// processPluginCRMCollection handles plugin_crm specific collection processing
func (uc *UseCase) processPluginCRMCollection(
	ctx context.Context,
	dataSource *pkg.DataSourceConfig,
	collection string,
	fields []string,
	collectionFilters map[string]model.FilterCondition,
	allCollections map[string][]string,
	result map[string]map[string][]map[string]any,
	logger log.Logger,
) error {
	// Get organization field to create collection name
	orgFields, exists := allCollections["organization"]
	if !exists || len(orgFields) == 0 {
		logger.Errorf("Organization field not found for plugin_crm collection %s", collection)
		return nil
	}

	newCollection := collection + "_" + orgFields[0]

	// Query the collection
	collectionResult, err := uc.queryMongoCollectionWithFilters(ctx, dataSource, newCollection, fields, collectionFilters, logger, "plugin_crm")
	if err != nil {
		return err
	}

	result["plugin_crm"][collection] = collectionResult

	// Decrypt data for plugin_crm
	decryptedResult, err := uc.decryptPluginCRMData(logger, result["plugin_crm"][collection], fields)
	if err != nil {
		logger.Errorf("Error decrypting data for collection %s: %s", collection, err.Error())
		//return pkg.ValidateBusinessError(constant.ErrDecryptionData, "", err)
		return fmt.Errorf("error decrypting data for collection %s: %w", collection, err)
	}

	result["plugin_crm"][collection] = decryptedResult

	return nil
}

// processRegularMongoCollection handles regular MongoDB collection processing
func (uc *UseCase) processRegularMongoCollection(
	ctx context.Context,
	dataSource *pkg.DataSourceConfig,
	collection string,
	fields []string,
	collectionFilters map[string]model.FilterCondition,
	result map[string]map[string][]map[string]any,
	logger log.Logger,
) error {
	// Determine database name from context (assuming it's available in the result map)
	var databaseName string
	for dbName := range result {
		databaseName = dbName
		break
	}

	collectionResult, err := uc.queryMongoCollectionWithFilters(ctx, dataSource, collection, fields, collectionFilters, logger, databaseName)
	if err != nil {
		return err
	}

	result[databaseName][collection] = collectionResult

	return nil
}

// queryMongoCollectionWithFilters queries a MongoDB collection with or without filters
func (uc *UseCase) queryMongoCollectionWithFilters(
	ctx context.Context,
	dataSource *pkg.DataSourceConfig,
	collection string,
	fields []string,
	collectionFilters map[string]model.FilterCondition,
	logger log.Logger,
	databaseName string,
) ([]map[string]any, error) {
	// Execute query with circuit breaker protection
	//queryResult, err := uc.CircuitBreakerManager.Execute(databaseName, func() (any, error) {
	//	if len(collectionFilters) > 0 {
	//		// Check if this is plugin_crm and needs filter transformation
	//		if strings.Contains(collection, "_") && !strings.Contains(collection, "organization") {
	//			transformedFilter, err := uc.transformPluginCRMAdvancedFilters(collectionFilters, logger)
	//			if err != nil {
	//				return nil, fmt.Errorf("error transforming advanced filters for collection %s: %w", collection, err)
	//			}
	//
	//			collectionFilters = transformedFilter
	//		}
	//
	//		return dataSource.MongoDBRepository.QueryWithAdvancedFilters(ctx, collection, fields, collectionFilters)
	//	}
	//
	//	// No filters, use legacy method
	//	return dataSource.MongoDBRepository.Query(ctx, collection, fields, nil)
	//})

	var (
		queryResult    []map[string]any
		errQueryResult error
	)
	if len(collectionFilters) > 0 {
		// Check if this is plugin_crm and needs filter transformation
		if strings.Contains(collection, "_") && !strings.Contains(collection, "organization") {
			transformedFilter, err := uc.transformPluginCRMAdvancedFilters(collectionFilters, logger)
			if err != nil {
				return nil, fmt.Errorf("error transforming advanced filters for collection %s: %w", collection, err)
			}

			collectionFilters = transformedFilter
		}

		queryResult, errQueryResult = dataSource.MongoDBRepository.QueryWithAdvancedFilters(ctx, collection, fields, collectionFilters)
	}

	queryResult, errQueryResult = dataSource.MongoDBRepository.Query(ctx, collection, fields, nil)
	if errQueryResult != nil {
		logger.Errorf("Error querying collection %s in %s (circuit breaker): %s", collection, databaseName, errQueryResult.Error())
		return nil, errQueryResult
	}

	//if len(collectionFilters) > 0 {
	//	logger.Infof("Successfully queried collection %s with advanced filters (circuit breaker: %s)",
	//		collection, uc.CircuitBreakerManager.GetState(databaseName))
	//} else {
	//	logger.Infof("Successfully queried collection %s (circuit breaker: %s)",
	//		collection, uc.CircuitBreakerManager.GetState(databaseName))
	//}

	return queryResult, nil
}

// transformPluginCRMAdvancedFilters transforms advanced FilterCondition filters for plugin_crm to use search fields
func (uc *UseCase) transformPluginCRMAdvancedFilters(filter map[string]model.FilterCondition, logger log.Logger) (map[string]model.FilterCondition, error) {
	if filter == nil {
		return nil, nil
	}

	hashSecretKey := os.Getenv("CRYPTO_HASH_SECRET_KEY_PLUGIN_CRM")
	if hashSecretKey == "" {
		return nil, fmt.Errorf("CRYPTO_HASH_SECRET_KEY_PLUGIN_CRM environment variable not set")
	}

	crypto := &libCrypto.Crypto{
		HashSecretKey: hashSecretKey,
		Logger:        logger,
	}

	transformedFilter := make(map[string]model.FilterCondition)

	// Define field mappings: encrypted field -> search field
	fieldMappings := map[string]string{
		"document":                "search.document",
		"name":                    "search.name",
		"banking_details.account": "search.banking_details_account",
		"banking_details.iban":    "search.banking_details_iban",
		"contact.primary_email":   "search.contact_primary_email",
		"contact.secondary_email": "search.contact_secondary_email",
		"contact.mobile_phone":    "search.contact_mobile_phone",
		"contact.other_phone":     "search.contact_other_phone",
	}

	for fieldName, condition := range filter {
		if searchField, exists := fieldMappings[fieldName]; exists {
			// Transform the condition by hashing string values
			transformedCondition := model.FilterCondition{}

			// Transform Equals values
			if len(condition.Equals) > 0 {
				transformedCondition.Equals = uc.hashFilterValues(condition.Equals, crypto)
			}

			// Transform GreaterThan values
			if len(condition.GreaterThan) > 0 {
				transformedCondition.GreaterThan = uc.hashFilterValues(condition.GreaterThan, crypto)
			}

			// Transform GreaterOrEqual values
			if len(condition.GreaterOrEqual) > 0 {
				transformedCondition.GreaterOrEqual = uc.hashFilterValues(condition.GreaterOrEqual, crypto)
			}

			// Transform LessThan values
			if len(condition.LessThan) > 0 {
				transformedCondition.LessThan = uc.hashFilterValues(condition.LessThan, crypto)
			}

			// Transform LessOrEqual values
			if len(condition.LessOrEqual) > 0 {
				transformedCondition.LessOrEqual = uc.hashFilterValues(condition.LessOrEqual, crypto)
			}

			// Transform Between values
			if len(condition.Between) > 0 {
				transformedCondition.Between = uc.hashFilterValues(condition.Between, crypto)
			}

			// Transform In values
			if len(condition.In) > 0 {
				transformedCondition.In = uc.hashFilterValues(condition.In, crypto)
			}

			// Transform NotIn values
			if len(condition.NotIn) > 0 {
				transformedCondition.NotIn = uc.hashFilterValues(condition.NotIn, crypto)
			}

			transformedFilter[searchField] = transformedCondition

			logger.Infof("Transformed advanced filter: %s -> %s", fieldName, searchField)
		} else {
			// Keep non-mapped fields as-is
			transformedFilter[fieldName] = condition
		}
	}

	return transformedFilter, nil
}

// hashFilterValues hashes string values in a filter condition array
func (uc *UseCase) hashFilterValues(values []any, crypto *libCrypto.Crypto) []any {
	hashedValues := make([]any, len(values))

	for i, value := range values {
		if strValue, ok := value.(string); ok && strValue != "" {
			hash := crypto.GenerateHash(&strValue)
			hashedValues[i] = hash
		} else {
			hashedValues[i] = value // Keep non-string values as-is
		}
	}

	return hashedValues
}

// // getContentType returns the MIME type for a given file extension.
// // If the extension is not recognized, it returns "text/plain".
//
//	func getContentType(ext string) string {
//		if contentType, ok := mimeTypes[ext]; ok {
//			return contentType
//		}
//
//		return "text/plain"
//	}
//
// decryptPluginCRMData decrypts sensitive fields for plugin_crm database
func (uc *UseCase) decryptPluginCRMData(logger log.Logger, collectionResult []map[string]any, fields []string) ([]map[string]any, error) {
	// Check if we need to decrypt any fields
	needsDecryption := false

	for _, field := range fields {
		// Check for top-level encrypted fields
		if isEncryptedField(field) {
			needsDecryption = true
			break
		}
		// Check for nested fields that might need decryption
		if strings.Contains(field, ".") {
			needsDecryption = true
			break
		}
	}

	if !needsDecryption {
		return collectionResult, nil
	}

	// Initialize crypto instance
	hashSecretKey := "fe8af9629a42c16b13d933365ed37366d1cb6e19812154804e794fe5e30a2d9f"
	encryptSecretKey := "5044a8d5a3a3110871c99473af43f83f0150e293a09cdc29107235d028ed91e0"
	// TODO: Adicionar depois no .env as chaves de crypto
	//if encryptSecretKey == "" {
	//	return nil, fmt.Errorf("CRYPTO_ENCRYPT_SECRET_KEY_PLUGIN_CRM environment variable not set")
	//}
	//
	//if hashSecretKey == "" {
	//	return nil, fmt.Errorf("CRYPTO_HASH_SECRET_KEY_PLUGIN_CRM environment variable not set")
	//}

	crypto := &libCrypto.Crypto{
		HashSecretKey:    hashSecretKey,
		EncryptSecretKey: encryptSecretKey,
		Logger:           logger,
	}

	err := crypto.InitializeCipher()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cipher: %w", err)
	}

	// Process each record in the collection
	for i, record := range collectionResult {
		decryptedRecord, err := uc.decryptRecord(record, crypto)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt record %d: %w", i, err)
		}

		collectionResult[i] = decryptedRecord
	}

	return collectionResult, nil
}

// isEncryptedField checks if a field is known to be encrypted in plugin_crm
func isEncryptedField(field string) bool {
	encryptedFields := map[string]bool{
		"document": true,
		"name":     true,
	}

	return encryptedFields[field]
}

// decryptRecord decrypts a single record's encrypted fields
func (uc *UseCase) decryptRecord(record map[string]any, crypto *libCrypto.Crypto) (map[string]any, error) {
	// Create a copy of the record to avoid modifying the original
	decryptedRecord := make(map[string]any)
	for k, v := range record {
		decryptedRecord[k] = v
	}

	// Decrypt top-level fields
	if err := uc.decryptTopLevelFields(decryptedRecord, crypto); err != nil {
		return nil, err
	}

	// Decrypt nested fields
	if err := uc.decryptNestedFields(decryptedRecord, crypto); err != nil {
		return nil, err
	}

	return decryptedRecord, nil
}

// decryptTopLevelFields decrypts top-level encrypted fields
func (uc *UseCase) decryptTopLevelFields(record map[string]any, crypto *libCrypto.Crypto) error {
	for fieldName, fieldValue := range record {
		if isEncryptedField(fieldName) && fieldValue != nil {
			if err := uc.decryptFieldValue(record, fieldName, fieldValue, crypto); err != nil {
				return fmt.Errorf("failed to decrypt field %s: %w", fieldName, err)
			}
		}
	}

	return nil
}

// decryptNestedFields decrypts nested encrypted fields in the record
func (uc *UseCase) decryptNestedFields(record map[string]any, crypto *libCrypto.Crypto) error {
	if err := uc.decryptContactFields(record, crypto); err != nil {
		return err
	}

	if err := uc.decryptBankingDetailsFields(record, crypto); err != nil {
		return err
	}

	if err := uc.decryptLegalPersonFields(record, crypto); err != nil {
		return err
	}

	if err := uc.decryptNaturalPersonFields(record, crypto); err != nil {
		return err
	}

	return nil
}

// decryptContactFields decrypts fields within the contact object
func (uc *UseCase) decryptContactFields(record map[string]any, crypto *libCrypto.Crypto) error {
	contact, ok := record["contact"].(map[string]any)
	if !ok {
		return nil
	}

	contactFields := []string{"primary_email", "secondary_email", "mobile_phone", "other_phone"}
	for _, fieldName := range contactFields {
		if fieldValue, exists := contact[fieldName]; exists && fieldValue != nil {
			if err := uc.decryptFieldValue(contact, fieldName, fieldValue, crypto); err != nil {
				return fmt.Errorf("failed to decrypt contact.%s: %w", fieldName, err)
			}
		}
	}

	record["contact"] = contact

	return nil
}

// decryptBankingDetailsFields decrypts fields within the banking_details object
func (uc *UseCase) decryptBankingDetailsFields(record map[string]any, crypto *libCrypto.Crypto) error {
	bankingDetails, ok := record["banking_details"].(map[string]any)
	if !ok {
		return nil
	}

	bankingFields := []string{"account", "iban"}
	for _, fieldName := range bankingFields {
		if fieldValue, exists := bankingDetails[fieldName]; exists && fieldValue != nil {
			if err := uc.decryptFieldValue(bankingDetails, fieldName, fieldValue, crypto); err != nil {
				return fmt.Errorf("failed to decrypt banking_details.%s: %w", fieldName, err)
			}
		}
	}

	record["banking_details"] = bankingDetails

	return nil
}

// decryptLegalPersonFields decrypts fields within the legal_person object
func (uc *UseCase) decryptLegalPersonFields(record map[string]any, crypto *libCrypto.Crypto) error {
	legalPerson, ok := record["legal_person"].(map[string]any)
	if !ok {
		return nil
	}

	representative, ok := legalPerson["representative"].(map[string]any)
	if !ok {
		return nil
	}

	representativeFields := []string{"name", "document", "email"}
	for _, fieldName := range representativeFields {
		if fieldValue, exists := representative[fieldName]; exists && fieldValue != nil {
			if err := uc.decryptFieldValue(representative, fieldName, fieldValue, crypto); err != nil {
				return fmt.Errorf("failed to decrypt legal_person.representative.%s: %w", fieldName, err)
			}
		}
	}

	legalPerson["representative"] = representative
	record["legal_person"] = legalPerson

	return nil
}

// decryptNaturalPersonFields decrypts fields within the natural_person object
func (uc *UseCase) decryptNaturalPersonFields(record map[string]any, crypto *libCrypto.Crypto) error {
	naturalPerson, ok := record["natural_person"].(map[string]any)
	if !ok {
		return nil
	}

	naturalPersonFields := []string{"mother_name", "father_name"}
	for _, fieldName := range naturalPersonFields {
		if fieldValue, exists := naturalPerson[fieldName]; exists && fieldValue != nil {
			if err := uc.decryptFieldValue(naturalPerson, fieldName, fieldValue, crypto); err != nil {
				return fmt.Errorf("failed to decrypt natural_person.%s: %w", fieldName, err)
			}
		}
	}

	record["natural_person"] = naturalPerson

	return nil
}

// decryptFieldValue decrypts a single field value if it's a non-empty string
func (uc *UseCase) decryptFieldValue(container map[string]any, fieldName string, fieldValue any, crypto *libCrypto.Crypto) error {
	strValue, ok := fieldValue.(string)
	if !ok || strValue == "" {
		return nil
	}

	decryptedValue, err := crypto.Decrypt(&strValue)
	if err != nil {
		return err
	}

	container[fieldName] = *decryptedValue

	return nil
}

//// shouldSkipProcessing checks if report should be skipped due to idempotency.
//func (uc *UseCase) shouldSkipProcessing(ctx context.Context, reportID uuid.UUID, logger log.Logger) bool {
//	reportStatus, err := uc.checkReportStatus(ctx, reportID, logger)
//	if err == nil {
//		if reportStatus == constant.FinishedStatus {
//			logger.Infof("Report %s is already finished, skipping reprocessing", reportID)
//			return true
//		}
//
//		if reportStatus == constant.ErrorStatus {
//			logger.Warnf("Report %s is in error state, skipping reprocessing", reportID)
//			return true
//		}
//	}
//
//	return false
//}
//
//// checkReportStatus checks the current status of a report to implement idempotency.
//func (uc *UseCase) checkReportStatus(ctx context.Context, reportID uuid.UUID, logger log.Logger) (string, error) {
//	zeroUUID := uuid.UUID{}
//	report, err := uc.ReportDataRepo.FindByID(ctx, reportID, zeroUUID)
//	if err != nil {
//		logger.Debugf("Could not check report status for %s (may be first attempt): %v", reportID, err)
//		return "", err
//	}
//
//	logger.Debugf("Report %s current status: %s", reportID, report.Status)
//
//	return report.Status, nil
//}
