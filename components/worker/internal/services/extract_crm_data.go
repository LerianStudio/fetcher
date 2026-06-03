// Package services provides business logic for data extraction operations.
package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/LerianStudio/lib-observability"

	"github.com/LerianStudio/fetcher/pkg"
	"github.com/LerianStudio/fetcher/pkg/model"
	modelDatasource "github.com/LerianStudio/fetcher/pkg/model/datasource"
	modelJob "github.com/LerianStudio/fetcher/pkg/model/job"
	portDS "github.com/LerianStudio/fetcher/pkg/ports/datasource"
	libCrypto "github.com/LerianStudio/lib-commons/v5/commons/crypto"
	libLog "github.com/LerianStudio/lib-observability/log"
	libOtel "github.com/LerianStudio/lib-observability/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// Static, host-free errors recorded on the EXPORTED span for plugin_crm datasource
// create/connect failures. The raw driver errors embed the DSN/host:port (via the
// MongoDB driver's ConnectionError / ServerSelectionError), and sanitizeSpanMessage
// only strips Bearer/Basic, so the raw error would leak host:port onto a span. These
// static messages mirror the generic engine adapter's redaction; the verbatim error
// is preserved only in the returned (wrapped) error and the local log.
var (
	errCRMDataSourceCreate  = errors.New("failed to create plugin_crm data source")
	errCRMDataSourceConnect = errors.New("failed to connect to plugin_crm data source")
)

var (
	processPluginCRMCollectionFn = func(
		uc *UseCase,
		ctx context.Context,
		dataSource portDS.CRMQueryable,
		collection string,
		fields []string,
		collectionFilters map[string]modelJob.FilterCondition,
		matchingCollections []string,
		result map[string]map[string][]map[string]any,
		logger libLog.Logger,
	) error {
		return uc.processPluginCRMCollection(ctx, dataSource, collection, fields, collectionFilters, matchingCollections, result, logger)
	}
	queryPluginCRMCollectionWithFiltersFn = func(
		uc *UseCase,
		ctx context.Context,
		dataSource portDS.CRMQueryable,
		collection string,
		fields []string,
		collectionFilters map[string]modelJob.FilterCondition,
		logger libLog.Logger,
	) ([]map[string]any, error) {
		return uc.queryPluginCRMCollectionWithFilters(ctx, dataSource, collection, fields, collectionFilters, logger)
	}
	decryptPluginCRMDataFn = func(uc *UseCase, logger libLog.Logger, collectionResult []map[string]any, fields []string) ([]map[string]any, error) {
		return uc.decryptPluginCRMData(logger, collectionResult, fields)
	}
)

// queryPluginCRMDatabase is the self-contained plugin_crm extraction path. It is
// the ONLY caller of the CRM compatibility chain after the strangler completion:
// the generic extraction path was removed, so plugin_crm no longer rides through a
// shared generic database query. It resolves the CRM connection, opens the datasource through
// the injected factory, asserts the CRM capability, and dispatches to QueryPluginCRM
// — preserving the legacy connection lifecycle (Connect, deferred Close) and CRM
// behavior (fan-out / merge / filter-hash / decrypt) byte-identically.
//
// It handles ONLY plugin_crm config names within the supplied message; the caller
// (extractInto) has already partitioned the CRM portion via splitCRMCompatibility,
// so every datasource here is a CRM datasource.
func (uc *UseCase) queryPluginCRMDatabase(
	ctx context.Context,
	message ExtractExternalDataMessage,
	connections []*model.Connection,
	result map[string]map[string][]map[string]any,
) error {
	logger, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.extract_external_data.query_plugin_crm_database")
	defer span.End()

	span.SetAttributes(attribute.String("app.request.request_id", reqID))

	for databaseName, collections := range message.MappedFields {
		foundConnection := findConnectionByConfigName(connections, databaseName)
		if foundConnection == nil {
			err := pkg.ValidationError{Code: "FET-0054", Title: "Connection Not Found", Message: fmt.Sprintf("connection not found for database: %s", databaseName)}
			libOtel.HandleSpanBusinessErrorEvent(span, "Connection not found", err)

			return fmt.Errorf("failed to query database %s: %w", databaseName, err)
		}

		dataSource, err := uc.CreateDataSource(ctx, foundConnection)
		if err != nil {
			// The factory's raw error may embed the DSN/host:port via the driver's
			// connection/topology errors. Record a STATIC, host-free message on the
			// EXPORTED span (mirroring the generic engine adapter, adapter.go); keep the
			// verbatim %w-wrapped error only in the returned error and the local log.
			libOtel.HandleSpanError(span, "Error creating data source", errCRMDataSourceCreate)
			logger.Log(ctx, libLog.LevelError, "error creating plugin_crm data source", libLog.Err(err))

			return fmt.Errorf("failed to query database %s: failed to create data source: %w", databaseName, err)
		}

		if err := dataSource.Connect(ctx, logger); err != nil {
			// Same redaction as above: the driver connect error embeds host:port.
			libOtel.HandleSpanError(span, "Error connecting to data source", errCRMDataSourceConnect)
			logger.Log(ctx, libLog.LevelError, "error connecting to plugin_crm data source", libLog.Err(err))

			return fmt.Errorf("failed to query database %s: failed to connect: %w", databaseName, err)
		}

		if err := uc.dispatchPluginCRMQuery(ctx, dataSource, databaseName, collections, message.Filters, result, logger); err != nil {
			return fmt.Errorf("failed to query database %s: %w", databaseName, err)
		}
	}

	return nil
}

// dispatchPluginCRMQuery asserts the CRM capability on the datasource, ensures the
// connection is closed after the query, and runs QueryPluginCRM. It is split from
// queryPluginCRMDatabase so the deferred Close is scoped per-datasource (closing
// each connection before the next, matching the legacy per-database lifecycle).
func (uc *UseCase) dispatchPluginCRMQuery(
	ctx context.Context,
	dataSource modelDatasource.DataSource,
	databaseName string,
	collections map[string][]string,
	allFilters map[string]map[string]map[string]modelJob.FilterCondition,
	result map[string]map[string][]map[string]any,
	logger libLog.Logger,
) error {
	defer func() {
		if closeErr := dataSource.Close(ctx); closeErr != nil {
			logger.Log(ctx, libLog.LevelWarn, "error closing connection",
				libLog.String("database_name", databaseName),
				libLog.Err(closeErr),
			)
		}
	}()

	crmDS, ok := dataSource.(portDS.CRMQueryable)
	if !ok {
		return pkg.ValidationError{Code: "FET-0055", Title: "Unsupported Operation", Message: "data source for plugin_crm does not support CRM queries"}
	}

	if result[databaseName] == nil {
		result[databaseName] = make(map[string][]map[string]any)
	}

	databaseFilters := allFilters[databaseName]

	return uc.QueryPluginCRM(ctx, crmDS, databaseName, collections, databaseFilters, result, logger)
}

// findConnectionByConfigName returns the connection whose ConfigName matches the
// given name, or nil when none matches.
func findConnectionByConfigName(connections []*model.Connection, configName string) *model.Connection {
	for _, conn := range connections {
		if conn != nil && conn.ConfigName == configName {
			return conn
		}
	}

	return nil
}

// QueryPluginCRM handles querying MongoDB plugin_crm database with special processing.
// It lists all available collections once and dispatches matching collections to each
// logical collection processor, avoiding repeated ListCollectionNames calls.
func (uc *UseCase) QueryPluginCRM(
	ctx context.Context,
	dataSource portDS.CRMQueryable,
	databaseName string,
	collections map[string][]string,
	databaseFilters map[string]map[string]modelJob.FilterCondition,
	result map[string]map[string][]map[string]any,
	logger libLog.Logger,
) error {
	_, tracer, reqID, _ := observability.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "service.extract_external_data.query_plugin_crm")
	defer span.End()

	span.SetAttributes(
		attribute.String("app.request.request_id", reqID),
		attribute.String("app.request.database_name", databaseName),
	)

	if len(collections) == 0 {
		return nil
	}

	// List all collections once for the entire plugin_crm database
	allCollectionNames, err := dataSource.ListCollectionNames(ctx)
	if err != nil {
		libOtel.HandleSpanError(span, "Error listing plugin_crm collections", err)
		return fmt.Errorf("failed to list collections in plugin_crm database: %w", err)
	}

	logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("plugin_crm database has %d collection(s): %v", len(allCollectionNames), allCollectionNames))

	for collection, fields := range collections {
		collectionFilters := getTableFilters(databaseFilters, collection)

		// Filter collections matching the prefix "collection_"
		prefix := collection + "_"

		var matchingCollections []string

		for _, c := range allCollectionNames {
			if strings.HasPrefix(c, prefix) {
				matchingCollections = append(matchingCollections, c)
			}
		}

		if len(matchingCollections) == 0 {
			err := pkg.ValidationError{Code: "FET-0059", Title: "Collection Not Found", Message: fmt.Sprintf("no collections found matching prefix %q in plugin_crm database", prefix)}
			libOtel.HandleSpanError(span, "No matching collections found", err)

			return err
		}

		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Prefix %q matched %d collection(s): %v", prefix, len(matchingCollections), matchingCollections))

		if err := processPluginCRMCollectionFn(uc, ctx, dataSource, collection, fields, collectionFilters, matchingCollections, result, logger); err != nil {
			libOtel.HandleSpanError(span, "Error processing plugin_crm collection", err)
			return fmt.Errorf("failed to process plugin_crm collection %s: %w", collection, err)
		}
	}

	return nil
}

// processPluginCRMCollection queries each matching physical collection and merges
// the results into a single dataset keyed by the original logical name.
// matchingCollections contains the pre-filtered list of real collection names
// (e.g., ["holders_06c4f684-...", "holders_abc123-..."]).
func (uc *UseCase) processPluginCRMCollection(
	ctx context.Context,
	dataSource portDS.CRMQueryable,
	collection string,
	fields []string,
	collectionFilters map[string]modelJob.FilterCondition,
	matchingCollections []string,
	result map[string]map[string][]map[string]any,
	logger libLog.Logger,
) error {
	// Query each matching collection and merge results
	var allResults []map[string]any

	for _, realCollection := range matchingCollections {
		collectionResult, err := queryPluginCRMCollectionWithFiltersFn(uc, ctx, dataSource, realCollection, fields, collectionFilters, logger)
		if err != nil {
			logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Error querying collection %s: %s", realCollection, err.Error()))
			return fmt.Errorf("failed to query plugin_crm collection %s: %w", realCollection, err)
		}

		logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Collection %s returned %d documents", realCollection, len(collectionResult)))

		allResults = append(allResults, collectionResult...)
	}

	if result["plugin_crm"] == nil {
		result["plugin_crm"] = make(map[string][]map[string]any)
	}

	result["plugin_crm"][collection] = allResults

	decryptedResult, err := decryptPluginCRMDataFn(uc, logger, result["plugin_crm"][collection], fields)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Error decrypting data for collection %s: %s", collection, err.Error()))
		return fmt.Errorf("error decrypting data for collection %s: %w", collection, err)
	}

	result["plugin_crm"][collection] = decryptedResult

	return nil
}

// queryPluginCRMCollectionWithFilters queries a MongoDB collection with plugin_crm specific filter transformation.
func (uc *UseCase) queryPluginCRMCollectionWithFilters(
	ctx context.Context,
	dataSource portDS.CRMQueryable,
	collection string,
	fields []string,
	collectionFilters map[string]modelJob.FilterCondition,
	logger libLog.Logger,
) ([]map[string]any, error) {
	var (
		queryResult    []map[string]any
		errQueryResult error
	)

	if len(collectionFilters) > 0 {
		transformedFilter, err := uc.transformPluginCRMAdvancedFilters(ctx, collectionFilters, logger)
		if err != nil {
			return nil, fmt.Errorf("error transforming advanced filters for collection %s: %w", collection, err)
		}

		queryResult, errQueryResult = dataSource.QueryCollectionWithAdvancedFilters(ctx, collection, fields, transformedFilter)
	} else {
		queryResult, errQueryResult = dataSource.QueryCollection(ctx, collection, fields, nil)
	}

	if errQueryResult != nil {
		logger.Log(ctx, libLog.LevelError, fmt.Sprintf("Error querying collection %s: %s", collection, errQueryResult.Error()))
		return nil, fmt.Errorf("failed to query collection %s: %w", collection, errQueryResult)
	}

	return queryResult, nil
}

// transformPluginCRMAdvancedFilters transforms advanced FilterCondition filters for plugin_crm to use search fields.
func (uc *UseCase) transformPluginCRMAdvancedFilters(ctx context.Context, filter map[string]modelJob.FilterCondition, logger libLog.Logger) (map[string]modelJob.FilterCondition, error) {
	if filter == nil {
		return nil, nil
	}

	if uc.crmHashSecretKey == "" {
		return nil, pkg.FailedPreconditionError{Code: "FET-0057", Title: "CRM Crypto Not Configured", Message: "CRM hash secret key not configured"}
	}

	crypto := &libCrypto.Crypto{
		HashSecretKey: uc.crmHashSecretKey,
		Logger:        logger,
	}

	transformedFilter := make(map[string]modelJob.FilterCondition)

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
			transformedCondition := modelJob.FilterCondition{}

			if len(condition.Equals) > 0 {
				transformedCondition.Equals = uc.hashFilterValues(condition.Equals, crypto)
			}

			if len(condition.GreaterThan) > 0 {
				transformedCondition.GreaterThan = uc.hashFilterValues(condition.GreaterThan, crypto)
			}

			if len(condition.GreaterOrEqual) > 0 {
				transformedCondition.GreaterOrEqual = uc.hashFilterValues(condition.GreaterOrEqual, crypto)
			}

			if len(condition.LessThan) > 0 {
				transformedCondition.LessThan = uc.hashFilterValues(condition.LessThan, crypto)
			}

			if len(condition.LessOrEqual) > 0 {
				transformedCondition.LessOrEqual = uc.hashFilterValues(condition.LessOrEqual, crypto)
			}

			if len(condition.Between) > 0 {
				transformedCondition.Between = uc.hashFilterValues(condition.Between, crypto)
			}

			// Transform In values
			if len(condition.In) > 0 {
				transformedCondition.In = uc.hashFilterValues(condition.In, crypto)
			}

			if len(condition.NotIn) > 0 {
				transformedCondition.NotIn = uc.hashFilterValues(condition.NotIn, crypto)
			}

			transformedFilter[searchField] = transformedCondition

			logger.Log(ctx, libLog.LevelInfo, fmt.Sprintf("Transformed advanced filter: %s -> %s", fieldName, searchField))
		} else {
			transformedFilter[fieldName] = condition
		}
	}

	return transformedFilter, nil
}

// hashFilterValues hashes string values in a filter condition array.
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

// decryptPluginCRMData decrypts sensitive fields for plugin_crm database.
// Note: ctx is intentionally omitted as the crypto layer receives its logger at construction time.
// Add ctx if trace propagation is needed in the future.
func (uc *UseCase) decryptPluginCRMData(logger libLog.Logger, collectionResult []map[string]any, fields []string) ([]map[string]any, error) {
	needsDecryption := false

	for _, field := range fields {
		if isEncryptedField(field) {
			needsDecryption = true
			break
		}

		if strings.Contains(field, ".") {
			needsDecryption = true
			break
		}
	}

	if !needsDecryption {
		return collectionResult, nil
	}

	// Initialize crypto instance
	if uc.crmEncryptSecretKey == "" {
		return nil, pkg.FailedPreconditionError{Code: "FET-0058", Title: "CRM Crypto Not Configured", Message: "CRM encrypt secret key not configured"}
	}

	if uc.crmHashSecretKey == "" {
		return nil, pkg.FailedPreconditionError{Code: "FET-0057", Title: "CRM Crypto Not Configured", Message: "CRM hash secret key not configured"}
	}

	crypto := &libCrypto.Crypto{
		HashSecretKey:    uc.crmHashSecretKey,
		EncryptSecretKey: uc.crmEncryptSecretKey,
		Logger:           logger,
	}

	err := crypto.InitializeCipher()
	if err != nil {
		return nil, pkg.FailedPreconditionError{Code: "FET-0064", Title: "Cipher Initialization Failed", Message: fmt.Sprintf("failed to initialize cipher: %s", err.Error()), Err: err}
	}

	// Process each record in the collection
	for i, record := range collectionResult {
		decryptedRecord, err := uc.decryptRecord(record, crypto)
		if err != nil {
			return nil, pkg.FailedPreconditionError{Code: "FET-0065", Title: "Decryption Failed", Message: fmt.Sprintf("failed to decrypt record %d: %s", i, err.Error()), Err: err}
		}

		collectionResult[i] = decryptedRecord
	}

	return collectionResult, nil
}

// isEncryptedField checks if a field is known to be encrypted in plugin_crm.
func isEncryptedField(field string) bool {
	encryptedFields := map[string]bool{
		"document": true,
		"name":     true,
	}

	return encryptedFields[field]
}

// decryptRecord decrypts a single record's encrypted fields.
func (uc *UseCase) decryptRecord(record map[string]any, crypto *libCrypto.Crypto) (map[string]any, error) {
	decryptedRecord := make(map[string]any)
	for k, v := range record {
		decryptedRecord[k] = v
	}

	if err := uc.decryptTopLevelFields(decryptedRecord, crypto); err != nil {
		return nil, fmt.Errorf("failed to decrypt top-level fields: %w", err)
	}

	if err := uc.decryptNestedFields(decryptedRecord, crypto); err != nil {
		return nil, fmt.Errorf("failed to decrypt nested fields: %w", err)
	}

	return decryptedRecord, nil
}

// decryptTopLevelFields decrypts top-level encrypted fields.
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

// decryptNestedFields decrypts nested encrypted fields in the record.
func (uc *UseCase) decryptNestedFields(record map[string]any, crypto *libCrypto.Crypto) error {
	if err := uc.decryptContactFields(record, crypto); err != nil {
		return fmt.Errorf("failed to decrypt contact fields: %w", err)
	}

	if err := uc.decryptBankingDetailsFields(record, crypto); err != nil {
		return fmt.Errorf("failed to decrypt banking details fields: %w", err)
	}

	if err := uc.decryptLegalPersonFields(record, crypto); err != nil {
		return fmt.Errorf("failed to decrypt legal person fields: %w", err)
	}

	if err := uc.decryptNaturalPersonFields(record, crypto); err != nil {
		return fmt.Errorf("failed to decrypt natural person fields: %w", err)
	}

	return nil
}

// decryptContactFields decrypts fields within the contact object.
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

// decryptBankingDetailsFields decrypts fields within the banking_details object.
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

// decryptLegalPersonFields decrypts fields within the legal_person object.
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

// decryptNaturalPersonFields decrypts fields within the natural_person object.
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

// decryptFieldValue decrypts a single field value if it's a non-empty string.
func (uc *UseCase) decryptFieldValue(container map[string]any, fieldName string, fieldValue any, crypto *libCrypto.Crypto) error {
	strValue, ok := fieldValue.(string)
	if !ok || strValue == "" {
		return nil
	}

	decryptedValue, err := crypto.Decrypt(&strValue)
	if err != nil {
		return fmt.Errorf("failed to decrypt value: %w", err)
	}

	container[fieldName] = *decryptedValue

	return nil
}
